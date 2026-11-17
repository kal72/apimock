import test from 'node:test';
import assert from 'node:assert/strict';
import { defaultFlow, insertBlock } from './flow.js';
import { projectScenarios } from './scenarios.js';
import { formatValue, httpStatusText, pathPreview } from './preview.js';

test('preview formats JSON while preserving raw text and JSON-like strings safely', () => {
  assert.equal(formatValue('{"id":1}'), '{\n  "id": 1\n}');
  assert.equal(formatValue('plain text'), 'plain text');
  assert.equal(formatValue('[false,0]'), '[\n  false,\n  0\n]');
});

test('httpStatusText provides readable HTTP reason phrases', () => {
  assert.equal(httpStatusText(200), 'OK');
  assert.equal(httpStatusText(201), 'Created');
  assert.equal(httpStatusText(404), 'Not Found');
  assert.equal(httpStatusText(500), 'Internal Server Error');
  assert.equal(httpStatusText(999), '');
});

test('path preview is read-only and includes effective headers and accumulated delay', () => {
  const flow=defaultFlow();
  insertBlock(flow,{type:'headers',name:'Headers',config:{headers:{'content-type':'text/plain','X-Test':'demo'}}},'request');
  flow.nodes.find(node=>node.type==='response').config.delay_ms=20;
  const before=structuredClone(flow),endpoint={name:'Products',method:'GET',path:'/products/:id',flow};
  const result=pathPreview(endpoint,projectScenarios(flow).rows[0],'http://localhost/public/demo');
  assert.equal(result.headers['Content-Type'],'application/json');assert.equal(result.headers['X-Test'],'demo');
  assert.equal(result.delay,20);assert.equal(result.url,'http://localhost/public/demo/products/:id');
  assert.equal(result.status,200);assert.equal(result.statusText,'OK');
  result.example.body={changed:true};assert.deepEqual(flow,before);
});
