import test from 'node:test';
import assert from 'node:assert/strict';
import { defaultFlow, graphIssues, insertBlock, newID, setConnection } from './flow.js';
import { addConditionScenario, defaultScenario, projectScenarios, scenarioForm, removeScenario } from './scenarios.js';
import { createEditor } from './editor.js';

const endpoint = () => ({ id: 'e', name: 'Products', method: 'GET', path: '/products/:id', flow: defaultFlow() });
const editor = () => { const state = createEditor(); state.endpoint = endpoint(); return state; };

test('an unconditional endpoint has one default request-to-response row', () => {
  const { rows, truncated } = projectScenarios(defaultFlow());
  assert.equal(rows.length, 1); assert.equal(truncated, false);
  assert.equal(rows[0].isDefault, true); assert.deepEqual(rows[0].pathIDs, ['request', 'response']);
});

test('new condition scenarios run before the preserved default response', () => {
  const flow = defaultFlow();
  const originalID = defaultScenario(flow).responseID;
  const first = addConditionScenario(flow, 'product');
  assert.equal(first.status, 200);
  assert.equal(defaultScenario(flow).responseID, originalID);
  assert.equal(defaultScenario(flow).status, 404);
  assert.equal(flow.nodes.find(node => node.id === originalID).label, 'Not found');
  const second = addConditionScenario(flow, 'custom');
  const rows = projectScenarios(flow).rows;
  assert.equal(rows.length, 3);
  assert.equal(rows[0].responseID, first.responseID);
  assert.equal(rows[1].responseID, second.responseID);
  assert.deepEqual(rows[1].conditions.map(item => item.matched), [false, true]);
  assert.equal(rows[2].isDefault, true); assert.equal(rows[2].status, 404);
  assert.deepEqual(graphIssues(flow), []);
  for (const a of flow.nodes) for (const b of flow.nodes) {
    if (a.id !== b.id) assert.ok(Math.abs(a.x - b.x) >= 280 || Math.abs(a.y - b.y) >= 158, 'inserted nodes overlap');
  }
});

test('the popup projects existing header/delay graph steps without flattening', () => {
  const e = endpoint();
  const header = insertBlock(e.flow, { type: 'headers', name: 'Shared header', config: { headers: { 'X-Shared': 'yes' } } }, 'request');
  const delay = insertBlock(e.flow, { type: 'delay', name: 'Shared delay', config: { delay_ms: 100 } }, header.id);
  const row = addConditionScenario(e.flow, 'custom');
  const form = scenarioForm(e, e.flow, row);
  assert.equal(form.legacyHeaders[0].id, header.id);
  assert.equal(form.legacyDelays[0].id, delay.id);
  assert.ok(form.connections.some(edge => edge.from === 'request' && edge.to === header.id));
});

test('the scenario dialog stages template changes without mutating the endpoint', () => {
  const state = editor(); const before = JSON.stringify(state.endpoint);
  state.openAddScenario(); state.chooseScenarioTemplate('auth');
  assert.equal(state.modal, 'scenario'); assert.equal(state.scenarioForm.response.status, 200);
  assert.equal(state.scenarioForm.fallback.status, 401);
  assert.equal(JSON.stringify(state.endpoint), before);
  state.modal = null; assert.equal(JSON.stringify(state.endpoint), before);
});

test('one popup atomically applies request, condition, response, headers, delay, and default', () => {
  const state = editor(); state.openAddScenario(); state.chooseScenarioTemplate('product');
  state.scenarioForm.route.name = 'Find product';
  state.scenarioForm.params = '{"id":"42"}';
  state.scenarioForm.headers = [{ key: 'X-Example', value: 'demo' }];
  state.scenarioForm.conditions[0].label = 'Product 42';
  state.scenarioForm.conditions[0].config.rules[0].value = '42';
  state.scenarioForm.response.status = 201; state.scenarioForm.response.delay_ms = 25;
  state.scenarioForm.response.body = '{"found":true}';
  state.scenarioForm.response.headers = [{ key: 'Content-Type', value: 'application/json' }, { key: 'X-Match', value: 'yes' }];
  state.saveScenario();
  assert.equal(state.modal, null); assert.equal(state.endpoint.name, 'Find product');
  assert.equal(state.scenarioRows.length, 2); assert.equal(state.scenarioRows[0].label, 'Product 42');
  assert.equal(state.scenarioRows[0].status, 201); assert.equal(state.fallbackRow.status, 404);
  assert.equal(state.endpoint.flow.nodes.find(node => node.id === state.scenarioRows[0].responseID).config.delay_ms, 25);
  assert.equal(JSON.parse(state.testInput).params.id, '42'); assert.deepEqual(state.flowIssues, []);
});

test('invalid edits keep the whole scenario unmodified and show the relevant tab', () => {
  const state = editor(); const before = JSON.stringify(state.endpoint);
  state.editScenario(state.fallbackRow); state.scenarioForm.response.body = '{';
  state.scenarioForm.route.name = 'Must not leak'; state.saveScenario();
  assert.equal(state.modal, 'scenario'); assert.equal(state.scenarioTab, 'response');
  assert.equal(JSON.stringify(state.endpoint), before); assert.match(state.dialogError, /valid JSON/);
});

test('default editing retains existing condition paths', () => {
  const state = editor(); addConditionScenario(state.endpoint.flow, 'product');
  state.editScenario(state.fallbackRow); state.scenarioForm.response.status = 401; state.saveScenario();
  assert.equal(state.conditions.length, 1); assert.equal(state.fallbackRow.status, 401);
  assert.equal(state.scenarioRows[0].status, 200); assert.deepEqual(state.flowIssues, []);
});

test('node edits open the same complete scenario popup as table actions', () => {
  const state = editor(); addConditionScenario(state.endpoint.flow, 'custom');
  state.editNodeScenario(state.conditions[0]);
  assert.equal(state.modal, 'scenario'); assert.ok(state.scenarioForm.response);
  assert.ok(state.scenarioForm.conditions.some(item => item.matched));
  state.editNodeScenario(state.requestNode); assert.equal(state.scenarioForm.isDefault, true);
});

test('removing a condition retains default and subsequent rules', () => {
  const flow = defaultFlow(); const first = addConditionScenario(flow, 'product');
  const second = addConditionScenario(flow, 'custom'); const fallbackID = defaultScenario(flow).responseID;
  removeScenario(flow, first.conditions.find(item => item.matched).id);
  const rows = projectScenarios(flow).rows;
  assert.equal(rows.length, 2); assert.equal(rows[0].responseID, second.responseID);
  assert.equal(rows[1].responseID, fallbackID); assert.deepEqual(graphIssues(flow), []);
});

test('path enumeration is bounded while default traversal stays available', () => {
  const flow = defaultFlow();
  let source = 'request';
  for (let index = 0; index < 8; index++) {
    const node = { id: newID(), type: 'condition', label: `Shared ${index}`, x: 0, y: 0, config: { field: 'params.id', operator: 'exists' } };
    flow.nodes.push(node);
    if (source === 'request') setConnection(flow, source, '', node.id);
    else { setConnection(flow, source, 'true', node.id); setConnection(flow, source, 'false', node.id); }
    source = node.id;
  }
  setConnection(flow, source, 'true', 'response'); setConnection(flow, source, 'false', 'response');
  const projection = projectScenarios(flow, 8);
  assert.equal(projection.rows.length, 8); assert.equal(projection.truncated, true);
  assert.equal(defaultScenario(flow).responseID, 'response');
});
