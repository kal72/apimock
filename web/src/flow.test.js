import test from 'node:test';
import assert from 'node:assert/strict';
import { defaultFlow, graphIssues, insertBlock, setConnection, connectionPoints, newID, headersFromRows, directResponseFlow, removeConfiguration } from './flow.js';

test('new flows connect Incoming request directly to a response', () => {
  const flow = defaultFlow();
  assert.deepEqual(graphIssues(flow), []);
  assert.deepEqual(flow.edges, [{ from: 'request', to: 'response', branch: '' }]);
  const points = connectionPoints(flow, flow.edges[0]);
  assert.ok(points.x2 > points.x1);
  assert.equal(points.y1, points.y2);
});

test('headers and delays are inserted into the existing path', () => {
  const flow = defaultFlow();
  const headers = insertBlock(flow, { type: 'headers', name: 'JSON headers', config: { headers: { 'X-Test': 'yes' } } }, 'request');
  const delay = insertBlock(flow, { type: 'delay', name: 'Delay', config: { delay_ms: 200 } }, headers.id);
  assert.deepEqual(graphIssues(flow), []);
  assert.equal(flow.edges.find(edge => edge.from === 'request').to, headers.id);
  assert.equal(flow.edges.find(edge => edge.from === headers.id).to, delay.id);
  assert.equal(flow.edges.find(edge => edge.from === delay.id).to, 'response');
});

test('conditions connect the matching path and a separate fallback response', () => {
  const flow = defaultFlow();
  const condition = insertBlock(flow, { type: 'condition', name: 'ID matches', config: { field: 'params.id', operator: 'equals', value: '123' } }, 'request');
  const yes = flow.edges.find(edge => edge.from === condition.id && edge.branch === 'true');
  const no = flow.edges.find(edge => edge.from === condition.id && edge.branch === 'false');
  assert.equal(yes.to, 'response');
  assert.notEqual(no.to, yes.to);
  assert.equal(flow.nodes.find(node => node.id === no.to).config.status, 404);
  assert.deepEqual(graphIssues(flow), []);
  assert.notEqual(connectionPoints(flow, yes).y1, connectionPoints(flow, no).y1);
});

test('connection updates replace a branch atomically and reject cycles', () => {
  const flow = defaultFlow();
  const headers = insertBlock(flow, { type: 'headers', name: 'Headers', config: {} }, 'request');
  const before = JSON.stringify(flow);
  assert.throws(() => setConnection(flow, headers.id, '', 'request'), /cycles/);
  assert.equal(JSON.stringify(flow), before);
  const delay = insertBlock(flow, { type: 'delay', name: 'Delay', config: { delay_ms: 100 } }, headers.id);
  assert.throws(() => setConnection(flow, delay.id, '', headers.id), /cycles/);
  assert.throws(() => setConnection(flow, 'response', '', headers.id), /branch/);
  setConnection(flow, headers.id, '', 'response');
  assert.equal(flow.edges.filter(edge => edge.from === headers.id).length, 1);
  assert.ok(graphIssues(flow).some(issue => issue.includes('not connected')));
});

test('response templates update a terminal response without orphaning it', () => {
  const flow = defaultFlow();
  const response = insertBlock(flow, { type: 'response', name: 'Unauthorized', config: { status: 401, body: '{"error":"unauthorized"}' } }, 'request');
  assert.equal(response.id, 'response');
  assert.equal(flow.nodes.length, 2);
  assert.deepEqual(graphIssues(flow), []);
});

test('node IDs work on internal HTTP origins without randomUUID', () => {
  const original = globalThis.crypto;
  Object.defineProperty(globalThis, 'crypto', { configurable: true, value: { getRandomValues: original.getRandomValues.bind(original) } });
  try { assert.match(newID(), /^[a-f0-9]{32}$/); }
  finally { Object.defineProperty(globalThis, 'crypto', { configurable: true, value: original }); }
});

test('header rows reject duplicates and injection instead of silently overwriting', () => {
  assert.deepEqual(headersFromRows([{ key: 'Content-Type', value: 'application/json' }, { key: '', value: '' }]), { 'Content-Type': 'application/json' });
  assert.throws(() => headersFromRows([{ key: 'X-Test', value: 'a' }, { key: 'x-test', value: 'b' }]), /Duplicate/);
  assert.throws(() => headersFromRows([{ key: 'X-Test', value: 'a\r\nInjected: yes' }]), /valid/);
});

test('direct response recovery creates a success response without conditions', () => {
  const flow=defaultFlow();flow.nodes=flow.nodes.filter(node=>node.type!=='response');flow.edges=[];
  const repaired=directResponseFlow(flow,'');
  assert.equal(repaired.nodes.length,2);
  assert.equal(repaired.nodes[1].config.status,200);
  assert.deepEqual(graphIssues(repaired),[]);
  assert.equal(repaired.edges[0].from,'request');
  assert.equal(repaired.edges[0].to,repaired.nodes[1].id);
  assert.equal(flow.nodes.length,1);
});

test('converting to a direct response preserves request examples and selected path settings', () => {
  const flow=defaultFlow();flow.nodes[0].config.example.headers={'X-Api-Key':'sample'};
  const header=insertBlock(flow,{type:'headers',name:'Shared headers',config:{headers:{'X-Shared':'yes'}}},'request');
  const condition=insertBlock(flow,{type:'condition',name:'Check ID',config:{field:'params.id',operator:'equals',value:'1'}},header.id);
  const converted=directResponseFlow(flow,'response');
  assert.equal(converted.nodes.some(node=>node.type==='condition'),false);
  assert.equal(converted.nodes.filter(node=>node.type==='response').length,1);
  assert.equal(converted.nodes.find(node=>node.type==='headers').config.headers['X-Shared'],'yes');
  assert.equal(converted.nodes[0].config.example.headers['X-Api-Key'],'sample');
  assert.equal(converted.nodes.find(node=>node.type==='headers').x,header.x);
  assert.deepEqual(graphIssues(converted),[]);
  assert.equal(flow.nodes.some(node=>node.id===condition.id),true);
});

test('explicit automatic success replaces an error-only flow with HTTP 200', () => {
  const flow=defaultFlow();flow.nodes[1].config.status=404;
  insertBlock(flow,{type:'headers',name:'Shared headers',config:{headers:{'X-Shared':'yes'}}},'request');
  const converted=directResponseFlow(flow,'');
  const response=converted.nodes.find(node=>node.type==='response');
  assert.equal(response.config.status,200);
  assert.notEqual(response.id,'response');
  assert.equal(converted.nodes.find(node=>node.type==='headers').config.headers['X-Shared'],'yes');
  assert.deepEqual(graphIssues(converted),[]);
});

test('response templates work through a header and delay chain without conditions', () => {
  const flow=defaultFlow();const header=insertBlock(flow,{type:'headers',name:'Headers',config:{}},'request');
  insertBlock(flow,{type:'delay',name:'Delay',config:{delay_ms:10}},header.id);
  const response=insertBlock(flow,{type:'response',name:'Created',config:{status:201,body:'{"created":true}'}},'request');
  assert.equal(response.config.status,201);
  assert.equal(flow.nodes.filter(node=>node.type==='response').length,1);
  assert.deepEqual(graphIssues(flow),[]);
});

test('removing optional steps reconnects request to response', () => {
  const flow=defaultFlow();const header=insertBlock(flow,{type:'headers',name:'Headers',config:{}},'request');
  removeConfiguration(flow,header.id);
  assert.deepEqual(graphIssues(flow),[]);
  assert.deepEqual(flow.edges,[{from:'request',to:'response',branch:''}]);
  const condition=insertBlock(flow,{type:'condition',name:'Condition',config:{field:'params.id',operator:'exists'}},'request');
  removeConfiguration(flow,condition.id);
  assert.equal(flow.nodes.length,2);
  assert.deepEqual(graphIssues(flow),[]);
  assert.throws(()=>removeConfiguration(flow,'response'),/A response is required/);
});
