import test from 'node:test';
import assert from 'node:assert/strict';
import { createEditor } from './editor.js';
import { defaultFlow, graphIssues } from './flow.js';

function editor() {
  const state = createEditor();
  state.workspace = { id: 'workspace', slug: 'demo' };
  state.endpoint = { id: 'endpoint', name: 'Product', method: 'GET', path: '/products/:id', flow: defaultFlow() };
  state.insertAfter = 'request';
  return state;
}

test('creating an endpoint posts and selects a preconnected flow', async () => {
  const state = editor();
  state.endpointForm = { name: 'Products', method: 'GET', path: '/products' };
  let posted;
  state.request = async (_url, _method, payload) => { posted = payload; return { ...payload, id: 'created' }; };
  await state.createEndpoint();
  assert.deepEqual(graphIssues(posted.flow), []);
  assert.equal(state.endpoint.flow.edges[0].to, 'response');
  assert.equal(state.dirty, false);
});

test('configuration dialogs do not mutate the flow until applied', () => {
  const state = editor();
  const before = JSON.stringify(state.endpoint);
  state.addConfiguration('condition');
  assert.equal(JSON.stringify(state.endpoint), before);
  state.modal = null;
  assert.equal(JSON.stringify(state.endpoint), before);
  state.addConfiguration('condition');
  state.nodeForm.config.value = '42';
  state.saveNode();
  assert.equal(state.conditions.length, 1);
  assert.equal(state.responses.length, 2);
  assert.deepEqual(state.flowIssues, []);
  assert.equal(state.dirty, true);
});

test('request editing persists examples, endpoint fields, and playground data', () => {
  const state = editor();
  state.editNode(state.requestNode);
  state.requestRoute.name = 'Find product';
  state.headerRows = [{ key: 'X-Api-Key', value: 'demo' }];
  state.paramsText = '{"id":"123"}';
  state.queryText = '{"page":["2"]}';
  state.requestBodyText = '{"user":{"active":true}}';
  state.saveNode();
  assert.equal(state.endpoint.name, 'Find product');
  assert.deepEqual(state.requestNode.config.example.headers, { 'X-Api-Key': 'demo' });
  assert.equal(JSON.parse(state.testInput).params.id, '123');
  const saved = structuredClone(state.endpoint);
  state.dirty = false;
  state.selectEndpoint(saved);
  assert.deepEqual(JSON.parse(state.testInput).query, { page: ['2'] });
});

test('switching connection branches preserves the other branch', () => {
  const state = editor();
  state.addConfiguration('condition'); state.saveNode();
  const condition = state.conditions[0];
  state.editNodeConnections(condition);
  state.connectionForm.branch = 'false';
  state.connectionBranchChanged();
  state.saveConnection();
  assert.ok(state.targetFor(condition.id, 'true'));
  assert.ok(state.targetFor(condition.id, 'false'));
  assert.deepEqual(state.flowIssues, []);
});

test('invalid request examples keep the dialog open and preserve the endpoint', () => {
  const state = editor();
  const before = JSON.stringify(state.endpoint);
  state.editNode(state.requestNode);
  state.queryText = '{"page":"2"}';
  state.saveNode();
  assert.equal(state.modal, 'node');
  assert.match(state.dialogError, /arrays of strings/);
  assert.equal(JSON.stringify(state.endpoint), before);
});

test('the default editor opens configuration and exposes unconditional response setup', () => {
  const state=editor();assert.equal(state.mode,'table');assert.equal(state.responseMode,'direct');
  state.endpoint.flow.nodes=state.endpoint.flow.nodes.filter(node=>node.type!=='response');state.endpoint.flow.edges=[];
  state.openDirectResponse();
  assert.equal(state.modal,'node');assert.equal(state.nodeForm.type,'response');assert.equal(state.nodeForm.config.status,200);
  assert.equal(state.endpoint.flow.nodes.length,1);
  state.saveNode();
  assert.deepEqual(state.flowIssues,[]);assert.equal(state.conditions.length,0);assert.equal(state.responses.length,1);
});

test('conditional to direct conversion is staged and undo restores all branches', () => {
  const state=editor();state.addConfiguration('condition');state.saveNode();
  const before=JSON.stringify(state.endpoint);
  state.openDirectResponse();assert.equal(state.modal,'direct');assert.equal(JSON.stringify(state.endpoint),before);
  state.prepareDirectResponse();assert.equal(JSON.stringify(state.endpoint),before);
  state.nodeForm.config.status=201;state.nodeForm.config.body='{"created":true}';state.saveNode();
  assert.equal(state.responseMode,'direct');assert.equal(state.conditions.length,0);assert.deepEqual(state.flowIssues,[]);
  state.undoChange();assert.equal(JSON.stringify(state.endpoint),before);assert.equal(state.conditions.length,1);
});

test('saving does not discard edits made while a request is in flight', async () => {
  const state=editor();state.dirty=true;state.endpoints=[structuredClone(state.endpoint)];
  let finish;state.request=(_url,_method,payload)=>new Promise(resolve=>{finish=()=>resolve(payload);});
  const saving=state.saveEndpoint();
  state.endpoint.name='Edited while saving';finish();await saving;
  assert.equal(state.endpoint.name,'Edited while saving');assert.equal(state.dirty,true);
  assert.equal(state.endpoints[0].name,'Product');
});

test('an in-flight save does not replace a newly selected endpoint', async () => {
  const state=editor();state.dirty=true;let finish;
  state.request=(_url,_method,payload)=>new Promise(resolve=>{finish=()=>resolve(payload);});
  const saving=state.saveEndpoint();state.endpoint={...state.endpoint,id:'another-endpoint',name:'Other'};finish();await saving;
  assert.equal(state.endpoint.id,'another-endpoint');
});

test('zoom clamps to bounds, preserves the viewport center, and never edits the flow', () => {
  const state=editor(), before=structuredClone(state.endpoint.flow);
  const viewport={scrollLeft:300,scrollTop:100,clientWidth:600,clientHeight:400,scrollTo(value){this.scrollLeft=value.left;this.scrollTop=value.top;}};
  state.$root={querySelector:()=>viewport};state.$nextTick=callback=>callback();
  state.setZoom(4);assert.equal(state.zoom,2);assert.equal(viewport.scrollLeft,900);assert.equal(viewport.scrollTop,400);
  state.setZoom(-1);assert.equal(state.zoom,0.5);
  state.setZoom(NaN);assert.equal(state.zoom,0.5);
  state.setZoom(1);assert.equal(state.zoom,1);assert.equal(viewport.scrollLeft,300);assert.equal(viewport.scrollTop,200);
  assert.deepEqual(state.endpoint.flow,before);assert.equal(state.dirty,false);
});

test('node dragging converts screen movement to canvas coordinates at each zoom', () => {
  for(const zoom of [0.5,1,2]){
    const state=editor(),node=state.endpoint.flow.nodes[0];state.zoom=zoom;
    const original={x:node.x,y:node.y};
    state.startDrag({button:0,clientX:100,clientY:100,preventDefault(){}},node);
    state.moveNode({clientX:150,clientY:125});
    assert.equal(node.x,original.x+50/zoom);assert.equal(node.y,original.y+25/zoom);
  }
});
