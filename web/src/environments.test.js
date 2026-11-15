import test from 'node:test';
import assert from 'node:assert/strict';
import { environmentEditor } from './environments.js';

function createMockEditor(initial = {}) {
  const calls = [];
  const state = {
    ...environmentEditor,
    get activeEnvironment() {
      return this.environments.find(item => item.id === this.activeEnvironmentID) || (this.environments.length ? this.environments[0] : null);
    },
    get environmentVariables() {
      return this.activeEnvironment?.variables || [];
    },
    get isEditingVariable() {
      return Boolean(this.variableForm.key && this.environmentVariables.some(v => v.key === this.variableForm.key));
    },
    environments: [],
    activeEnvironmentID: '',
    environmentForm: { name: '' },
    variableForm: { key: '', kind: 'string', value: '' },
    dialogError: '',
    busy: false,
    notice: '',
    currentUser: { id: 'u1' },
    request: async (url, method = 'GET', data) => {
      calls.push({ url, method, data });
      if (method === 'GET' && url === '/api/environments') {
        return state.environments;
      }
      if (method === 'POST' && url === '/api/environments') {
        const env = { id: 'env_' + (state.environments.length + 1), name: data.name, variables: [] };
        state.environments.push(env);
        return env;
      }
      if (method === 'PUT' && url.includes('/variables/')) {
        const parts = url.split('/');
        const envID = parts[3];
        const key = decodeURIComponent(parts[5]);
        const env = state.environments.find(e => e.id === envID);
        if (env) {
          const existing = env.variables.find(v => v.key === key);
          if (existing) {
            existing.kind = data.kind;
            existing.value = data.value;
          } else {
            env.variables.push({ key, kind: data.kind, value: data.value });
          }
        }
        return { saved: true };
      }
      if (method === 'DELETE' && url.includes('/variables/')) {
        const parts = url.split('/');
        const envID = parts[3];
        const key = decodeURIComponent(parts[5]);
        const env = state.environments.find(e => e.id === envID);
        if (env) {
          env.variables = env.variables.filter(v => v.key !== key);
        }
        return { deleted: true };
      }
      return {};
    },
    notify: (msg) => { state.notice = msg; },
    ...initial
  };
  return { state, calls };
}

test('openEnvironments auto-selects first environment if none active', () => {
  const { state } = createMockEditor({
    environments: [{ id: 'env_1', name: 'Dev', variables: [] }]
  });
  state.openEnvironments();
  assert.equal(state.modal, 'environments');
  assert.equal(state.activeEnvironmentID, 'env_1');
  assert.equal(state.activeEnvironment?.name, 'Dev');
});

test('saveEnvironmentVariable creates a string variable and resets form', async () => {
  const { state } = createMockEditor({
    environments: [{ id: 'env_1', name: 'Dev', variables: [] }],
    activeEnvironmentID: 'env_1'
  });
  state.variableForm = { key: 'api_url', kind: 'string', value: 'https://api.example.com' };
  await state.saveEnvironmentVariable();

  assert.equal(state.dialogError, '');
  assert.equal(state.variableForm.key, '');
  assert.equal(state.activeEnvironmentID, 'env_1');
  assert.equal(state.environmentVariables.length, 1);
  assert.equal(state.environmentVariables[0].key, 'api_url');
  assert.equal(state.environmentVariables[0].value, 'https://api.example.com');
});

test('saveEnvironmentVariable converts boolean and number types properly', async () => {
  const { state } = createMockEditor({
    environments: [{ id: 'env_1', name: 'Dev', variables: [] }],
    activeEnvironmentID: 'env_1'
  });

  // boolean
  state.variableForm = { key: 'debug_mode', kind: 'boolean', value: 'true' };
  await state.saveEnvironmentVariable();
  assert.equal(state.environmentVariables.find(v => v.key === 'debug_mode')?.value, true);

  // number
  state.variableForm = { key: 'max_retries', kind: 'number', value: '5' };
  await state.saveEnvironmentVariable();
  assert.equal(state.environmentVariables.find(v => v.key === 'max_retries')?.value, 5);

  // invalid number
  state.variableForm = { key: 'bad_number', kind: 'number', value: 'not-a-number' };
  await state.saveEnvironmentVariable();
  assert.match(state.dialogError, /valid number/);
});

test('editEnvironmentVariable populates form and isEditingVariable is true', () => {
  const { state } = createMockEditor({
    environments: [
      { id: 'env_1', name: 'Dev', variables: [{ key: 'timeout', kind: 'number', value: 30 }] }
    ],
    activeEnvironmentID: 'env_1'
  });

  state.editEnvironmentVariable(state.environmentVariables[0]);
  assert.equal(state.isEditingVariable, true);
  assert.equal(state.variableForm.key, 'timeout');
  assert.equal(state.variableForm.kind, 'number');
  assert.equal(state.variableForm.value, 30);
});

test('copyVariableExpression and copyVariableValue invoke copyText properly', async () => {
  const copied = [];
  const { state } = createMockEditor({
    environments: [
      { id: 'env_1', name: 'Dev', variables: [{ key: 'base_url', kind: 'string', value: 'https://api.test' }] }
    ],
    activeEnvironmentID: 'env_1'
  });
  state.copyText = async (val, msg) => { copied.push({ val, msg }); };

  await state.copyVariableExpression('base_url');
  assert.equal(copied.length, 1);
  assert.equal(copied[0].val, '{{ environment.base_url }}');

  await state.copyVariableValue(state.environmentVariables[0]);
  assert.equal(copied.length, 2);
  assert.equal(copied[1].val, 'https://api.test');
});
