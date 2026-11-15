import assert from 'node:assert/strict';
import test from 'node:test';
import { createEditor } from './editor.js';
import { defaultFlow } from './flow.js';
import { createPublicDocs } from './public-docs.js';
import { addConditionScenario } from './scenarios.js';

function samplePublicPayload() {
  const flow = defaultFlow();
  addConditionScenario(flow, 'header', { path: '/products/:id', example: { params: { id: '42' }, query: { page: ['1'] }, headers: { 'X-Client': 'web' }, body: {} } });
  return {
    workspace: { id: 'w1', name: 'Demo Store', slug: 'demo-store', public_docs: true },
    public_base_url: 'https://mock.example.com',
    endpoints: [
      { id: 'e1', workspace_id: 'w1', name: 'Get product', method: 'GET', path: '/products/:id', flow },
      { id: 'e2', workspace_id: 'w1', name: 'Create order', method: 'POST', path: '/orders', flow: defaultFlow() },
    ],
  };
}

test('publicDocs initializes endpoints, URLs, method filters, and code snippets', () => {
  const state = createPublicDocs(samplePublicPayload());
  assert.equal(state.baseURL, 'https://mock.example.com/public/demo-store');
  assert.equal(state.shareURL, 'https://mock.example.com/share/demo-store');
  assert.equal(state.endpoint.id, 'e1');
  assert.deepEqual(state.routeParams, ['id']);
  assert.equal(state.playParams.id, '123');
  assert.equal(state.scenarios.length, 2);
  assert.equal(state.resolvedPlaygroundURL, 'https://mock.example.com/public/demo-store/products/123');

  state.methodFilter = 'POST';
  assert.equal(state.filteredEndpoints.length, 1);
  assert.equal(state.filteredEndpoints[0].id, 'e2');

  state.methodFilter = 'ALL';
  state.search = 'product';
  assert.equal(state.filteredEndpoints.length, 1);
  assert.ok(state.curlSnippet.includes('curl -i -X GET "https://mock.example.com/public/demo-store/products/123"'));
  assert.ok(state.fetchSnippet.includes('await fetch("https://mock.example.com/public/demo-store/products/123"'));
});

test('loadScenarioIntoPlayground populates playground inputs from condition rules', () => {
  const state = createPublicDocs(samplePublicPayload());
  const conditionRow = state.scenarios.find(row => !row.isDefault);
  state.loadScenarioIntoPlayground(conditionRow);
  assert.equal(state.playTab, 'headers');
  assert.ok(state.playHeaders.some(h => h.key.toLowerCase() === 'authorization'));
});

test('sendRequest posts to public playground endpoint and updates matchedScenarioLabel', async () => {
  const payload = samplePublicPayload();
  const state = createPublicDocs(payload);
  const originalFetch = globalThis.fetch;
  let calledURL = '';
  let calledBody = null;
  globalThis.fetch = async (url, options) => {
    calledURL = url;
    calledBody = JSON.parse(options.body);
    return {
      ok: true,
      status: 200,
      async json() {
        return {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: '{"id":"42"}',
          response_id: state.scenarios[0].responseID,
          duration_ms: 4,
          conditions: [],
        };
      },
    };
  };
  try {
    state.playParams.id = '42';
    await state.sendRequest();
    assert.equal(calledURL, '/share/demo-store/playground');
    assert.equal(calledBody.endpoint_id, 'e1');
    assert.equal(calledBody.params.id, '42');
    assert.equal(state.testResult.status, 200);
    assert.match(state.matchedScenarioLabel, /Matched: #1/);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test('editor togglePublicDocs updates workspace public_docs state and publicDocsURL', async () => {
  const state = createEditor();
  state.publicOrigin = 'https://mock.example.com';
  state.user = { id: 'u1', role: 'admin' };
  state.workspace = { id: 'w1', name: 'Demo', slug: 'demo', role: 'owner', public_docs: false };
  state.workspaces = [state.workspace];
  assert.equal(state.publicDocsURL, 'https://mock.example.com/share/demo');

  let putData = null;
  state.request = async (url, method = 'GET', data) => {
    if (method === 'PUT' && url === '/api/workspaces/w1') {
      putData = data;
      return { ...data, role: 'owner' };
    }
    if (url === '/api/workspaces') {
      return [{ id: 'w1', name: 'Demo', slug: 'demo', role: 'owner', public_docs: Boolean(putData?.public_docs) }];
    }
    return {};
  };

  await state.togglePublicDocs();
  assert.equal(putData.public_docs, true);
  assert.equal(state.workspace.public_docs, true);
});
