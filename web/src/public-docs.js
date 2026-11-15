import { branchRows } from './branches.js';
import { conditionKindLabel, conditionSummary, normalizeCondition, pathParameters } from './conditions.js';
import { defaultExample } from './flow.js';
import { formatValue as formatPreviewValue } from './preview.js';
import { projectScenarios } from './scenarios.js';

export function createPublicDocs(initialData) {
  let payload = initialData;
  if (!payload && typeof document !== 'undefined') {
    try {
      const el = document.getElementById('public-docs-data');
      if (el?.textContent) payload = JSON.parse(el.textContent);
    } catch {
      payload = null;
    }
  }
  const workspace = payload?.workspace || { id: '', name: 'Workspace', slug: '', public_docs: true };
  const endpoints = Array.isArray(payload?.endpoints) ? payload.endpoints : [];
  const publicOrigin = payload?.public_base_url || '';

  const state = {
    workspace,
    endpoints,
    publicOrigin,
    endpoint: null,
    search: '',
    methodFilter: 'ALL',
    notice: '',
    noticeTimer: null,
    playTab: 'params',
    playParams: {},
    playQuery: [],
    playHeaders: [],
    playBodyMode: 'json',
    playBodyJSON: '{}',
    playBodyForm: '{}',
    playBodyText: '',
    testing: false,
    playError: '',
    testResult: null,

    get origin() {
      return this.publicOrigin || (typeof location !== 'undefined' ? location.origin : 'http://localhost:8080');
    },
    get baseURL() {
      return `${this.origin}/public/${this.workspace?.slug || ''}`;
    },
    get shareURL() {
      return `${this.origin}/share/${this.workspace?.slug || ''}`;
    },
    get filteredEndpoints() {
      const q = this.search.trim().toLowerCase();
      return this.endpoints.filter(item => {
        if (this.methodFilter !== 'ALL' && item.method !== this.methodFilter) return false;
        if (!q) return true;
        return `${item.method} ${item.path} ${item.name}`.toLowerCase().includes(q);
      });
    },
    get requestNode() {
      return this.endpoint?.flow?.nodes?.find(node => node.type === 'request') || null;
    },
    get requestExample() {
      return this.requestNode?.config?.example || defaultExample();
    },
    get routeParams() {
      return pathParameters(this.endpoint?.path || '');
    },
    get scenarios() {
      if (!this.endpoint?.flow) return [];
      const rawRows = branchRows(this.endpoint.flow) || projectScenarios(this.endpoint.flow).rows;
      let priorityCounter = 0;
      return rawRows.map(row => {
        const responseNode = this.endpoint.flow.nodes.find(n => n.id === row.responseID);
        const conditionDetails = (row.conditions || [])
          .map(item => {
            const node = this.endpoint.flow.nodes.find(n => n.id === item.id);
            if (!node) return null;
            const cfg = normalizeCondition(node.config);
            return {
              id: node.id,
              name: node.label,
              matched: item.matched,
              config: cfg,
              meta: `${conditionKindLabel(cfg)}${cfg.condition_type === 'body' ? ' · ' + (cfg.body_mode === 'form' ? 'Form' : cfg.raw_format === 'text' ? 'Raw Text' : 'Raw JSON') : ''} · ${cfg.match === 'any' ? 'ANY' : 'ALL'}`,
              summary: conditionSummary(cfg),
            };
          })
          .filter(Boolean);
        const priority = row.isDefault ? null : row.priority || ++priorityCounter;
        return {
          ...row,
          priority,
          responseMode: responseNode?.config?.response_mode === 'dynamic' ? 'dynamic' : 'static',
          delay: responseNode?.config?.delay_ms || 0,
          conditionDetails,
        };
      });
    },
    get defaultScenarioRow() {
      return this.scenarios.find(row => row.isDefault) || this.scenarios.at(-1) || null;
    },
    get documentedQueryParams() {
      if (!this.endpoint) return [];
      const map = new Map();
      const exQuery = this.requestExample.query || {};
      for (const [k, v] of Object.entries(exQuery)) {
        map.set(k, Array.isArray(v) ? v.join(', ') : String(v ?? ''));
      }
      for (const node of this.endpoint.flow?.nodes || []) {
        if (node.type === 'condition') {
          const cfg = normalizeCondition(node.config);
          if (cfg.condition_type === 'query') {
            for (const rule of cfg.rules || []) {
              if (rule.field && !map.has(rule.field)) {
                map.set(rule.field, rule.value || `(${rule.operator})`);
              }
            }
          }
        }
      }
      return [...map.entries()].map(([key, value]) => ({ key, value }));
    },
    get documentedHeaders() {
      if (!this.endpoint) return [];
      const map = new Map();
      const exHeaders = this.requestExample.headers || {};
      for (const [k, v] of Object.entries(exHeaders)) {
        map.set(k, String(v ?? ''));
      }
      for (const node of this.endpoint.flow?.nodes || []) {
        if (node.type === 'condition') {
          const cfg = normalizeCondition(node.config);
          if (cfg.condition_type === 'header') {
            for (const rule of cfg.rules || []) {
              if (rule.field && !map.has(rule.field)) {
                map.set(rule.field, rule.value || `(${rule.operator})`);
              }
            }
          }
        }
      }
      return [...map.entries()].map(([key, value]) => ({ key, value }));
    },
    get hasBodyExample() {
      if (!this.endpoint) return false;
      const ex = this.requestExample;
      if (ex.body_mode === 'form' && ex.form && Object.keys(ex.form).length > 0) return true;
      if (ex.body_mode === 'text' && ex.raw_body) return true;
      if (ex.body && Object.keys(ex.body).length > 0) return true;
      return ['POST', 'PUT', 'PATCH'].includes(this.endpoint.method);
    },
    get formattedExampleBody() {
      const ex = this.requestExample;
      if (ex.body_mode === 'form') return JSON.stringify(ex.form || {}, null, 2);
      if (ex.body_mode === 'text') return ex.raw_body || '';
      if (ex.raw_body) return formatPreviewValue(ex.raw_body);
      return JSON.stringify(ex.body || {}, null, 2);
    },
    get resolvedPlaygroundURL() {
      if (!this.endpoint) return this.baseURL;
      const resolvedPath = (this.endpoint.path || '/').replace(/:([A-Za-z0-9_]+)/g, (_, name) => {
        const val = (this.playParams[name] ?? '').trim();
        return val ? encodeURIComponent(val) : `:${name}`;
      });
      const qs = this.playQuery
        .filter(item => item.key.trim())
        .map(item => `${encodeURIComponent(item.key.trim())}=${encodeURIComponent(item.value ?? '')}`)
        .join('&');
      return `${this.baseURL}${resolvedPath}${qs ? '?' + qs : ''}`;
    },
    get curlSnippet() {
      if (!this.endpoint) return '';
      const lines = [`curl -i -X ${this.endpoint.method} "${this.resolvedPlaygroundURL}"`];
      for (const h of this.playHeaders) {
        if (h.key.trim()) {
          lines.push(`  -H "${h.key.trim()}: ${(h.value ?? '').replace(/"/g, '\\"')}"`);
        }
      }
      if (!['GET', 'HEAD'].includes(this.endpoint.method)) {
        const payloadText = this.playBodyMode === 'text' ? this.playBodyText : this.playBodyMode === 'form' ? this.playBodyForm : this.playBodyJSON;
        if (payloadText && payloadText.trim() && payloadText.trim() !== '{}') {
          lines.push(`  -d '${payloadText.replace(/'/g, "'\\''")}'`);
        }
      }
      return lines.join(' \\\n');
    },
    get fetchSnippet() {
      if (!this.endpoint) return '';
      const headers = {};
      for (const h of this.playHeaders) {
        if (h.key.trim()) headers[h.key.trim()] = h.value ?? '';
      }
      const options = { method: this.endpoint.method };
      if (Object.keys(headers).length) options.headers = headers;
      if (!['GET', 'HEAD'].includes(this.endpoint.method)) {
        const payloadText = this.playBodyMode === 'text' ? this.playBodyText : this.playBodyMode === 'form' ? this.playBodyForm : this.playBodyJSON;
        if (payloadText && payloadText.trim() && payloadText.trim() !== '{}') {
          options.body = payloadText;
        }
      }
      return `const response = await fetch(${JSON.stringify(this.resolvedPlaygroundURL)}, ${JSON.stringify(options, null, 2)});\nconst data = await response.json();`;
    },
    get lastMatchedResponseID() {
      return this.testResult?.response_id || '';
    },
    get matchedScenarioLabel() {
      if (!this.testResult?.response_id) return '';
      const matched = this.scenarios.find(row => row.responseID === this.testResult.response_id);
      if (!matched) return '';
      return matched.isDefault ? 'Matched: Default response' : `Matched: #${matched.priority} ${matched.label}`;
    },

    notify(message) {
      this.notice = message;
      if (this.noticeTimer) clearTimeout(this.noticeTimer);
      if (message) {
        this.noticeTimer = setTimeout(() => {
          if (this.notice === message) this.notice = '';
        }, 2400);
        this.noticeTimer?.unref?.();
      }
    },
    init() {
      if (!this.endpoint && this.endpoints.length) {
        this.selectEndpoint(this.endpoints[0]);
      }
    },
    formatValue(value) {
      return formatPreviewValue(value);
    },
    selectEndpoint(ep) {
      if (!ep) return;
      this.endpoint = ep;
      this.testResult = null;
      this.playError = '';
      this.resetPlayground();
    },
    resetPlayground() {
      if (!this.endpoint) return;
      const ex = this.requestExample;
      const params = {};
      for (const name of this.routeParams) {
        params[name] = ex.params?.[name] || '123';
      }
      this.playParams = params;
      this.playQuery = Object.entries(ex.query || {}).flatMap(([key, values]) =>
        (Array.isArray(values) ? values : [String(values ?? '')]).map(value => ({ key, value: String(value ?? '') }))
      );
      this.playHeaders = Object.entries(ex.headers || {}).map(([key, value]) => ({ key, value: String(value ?? '') }));
      this.playBodyMode = ex.body_mode || 'json';
      this.playBodyJSON = ex.raw_body && this.playBodyMode === 'json' ? ex.raw_body : JSON.stringify(ex.body || {}, null, 2);
      this.playBodyForm = JSON.stringify(ex.form || {}, null, 2);
      this.playBodyText = ex.raw_body || '';
      this.playError = '';
    },
    loadScenarioIntoPlayground(row) {
      if (!row) return;
      this.resetPlayground();
      let targetTab = 'params';
      for (const cond of row.conditionDetails || []) {
        if (!cond.matched) continue;
        const cfg = cond.config;
        for (const rule of cfg.rules || []) {
          if (!rule.field) continue;
          const sampleValue = rule.operator === 'empty' ? '' : rule.value || 'sample-value';
          if (cfg.condition_type === 'path') {
            this.playParams[rule.field] = sampleValue || '123';
            targetTab = 'params';
          } else if (cfg.condition_type === 'query') {
            const existing = this.playQuery.find(q => q.key === rule.field);
            if (existing) existing.value = sampleValue;
            else this.playQuery.push({ key: rule.field, value: sampleValue });
            targetTab = 'params';
          } else if (cfg.condition_type === 'header') {
            const existing = this.playHeaders.find(h => h.key.toLowerCase() === rule.field.toLowerCase());
            if (existing) existing.value = sampleValue;
            else this.playHeaders.push({ key: rule.field, value: sampleValue });
            targetTab = 'headers';
          } else if (cfg.condition_type === 'body') {
            targetTab = 'body';
            if (cfg.body_mode === 'form') {
              this.playBodyMode = 'form';
              try {
                const formObj = JSON.parse(this.playBodyForm || '{}');
                formObj[rule.field] = [sampleValue];
                this.playBodyForm = JSON.stringify(formObj, null, 2);
              } catch {
                this.playBodyForm = JSON.stringify({ [rule.field]: [sampleValue] }, null, 2);
              }
            } else if (cfg.raw_format === 'text') {
              this.playBodyMode = 'text';
              this.playBodyText = sampleValue;
            } else {
              this.playBodyMode = 'json';
              try {
                const bodyObj = JSON.parse(this.playBodyJSON || '{}');
                setNestedValue(bodyObj, rule.field, sampleValue);
                this.playBodyJSON = JSON.stringify(bodyObj, null, 2);
              } catch {
                this.playBodyJSON = JSON.stringify({ [rule.field]: sampleValue }, null, 2);
              }
            }
          }
        }
      }
      this.playTab = targetTab;
      this.notify(`Loaded "${row.label}" inputs into Playground.`);
    },
    buildPlaygroundPayload() {
      if (!this.endpoint) throw Error('Select an endpoint first.');
      const headers = {};
      for (const row of this.playHeaders) {
        const key = row.key.trim();
        if (key) headers[key] = row.value ?? '';
      }
      const query = {};
      for (const row of this.playQuery) {
        const key = row.key.trim();
        if (!key) continue;
        query[key] = query[key] || [];
        query[key].push(row.value ?? '');
      }
      const params = {};
      for (const param of this.routeParams) {
        params[param] = String(this.playParams[param] ?? '');
      }
      let body = {};
      let rawBody = '';
      let form = {};
      if (this.playBodyMode === 'json') {
        const text = (this.playBodyJSON || '').trim();
        if (text) {
          let parsed;
          try {
            parsed = JSON.parse(text);
          } catch {
            throw Error('Request JSON body must be valid JSON.');
          }
          if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) body = parsed;
          else rawBody = text;
        }
      } else if (this.playBodyMode === 'form') {
        const text = (this.playBodyForm || '').trim();
        if (text) {
          try {
            form = JSON.parse(text);
          } catch {
            throw Error('Form body must be valid JSON mapping fields to string arrays.');
          }
        }
      } else if (this.playBodyMode === 'text') {
        rawBody = this.playBodyText || '';
      }
      return {
        endpoint_id: this.endpoint.id,
        headers,
        params,
        query,
        body_mode: this.playBodyMode,
        body,
        raw_body: rawBody,
        form,
      };
    },
    async sendRequest() {
      if (!this.endpoint || this.testing) return;
      this.playError = '';
      this.testing = true;
      try {
        const payload = this.buildPlaygroundPayload();
        const response = await fetch(`/share/${encodeURIComponent(this.workspace.slug)}/playground`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        const result = await response.json();
        if (!response.ok) throw Error(result.error || 'Playground request failed.');
        this.testResult = result;
      } catch (error) {
        this.playError = error.message;
      } finally {
        this.testing = false;
      }
    },
    async copyBaseURL() {
      await this.copyText(this.baseURL, 'Mock base URL copied.');
    },
    async copyShareURL() {
      await this.copyText(this.shareURL, 'Public API documentation link copied.');
    },
    async copyEndpointURL() {
      if (!this.endpoint) return;
      await this.copyText(this.baseURL + this.endpoint.path, 'Endpoint mock URL copied.');
    },
    async copyText(text, message) {
      try {
        await navigator.clipboard.writeText(text);
        this.notify(message || 'Copied to clipboard.');
      } catch {
        this.notify('Copied: ' + text);
      }
    },
  };

  if (state.endpoints.length) {
    state.selectEndpoint(state.endpoints[0]);
  }
  return state;
}

function setNestedValue(target, path, value) {
  if (!path || path === '$') return;
  const parts = path.split('.').filter(Boolean);
  let cursor = target;
  for (let i = 0; i < parts.length - 1; i++) {
    const key = parts[i];
    if (!cursor[key] || typeof cursor[key] !== 'object') cursor[key] = {};
    cursor = cursor[key];
  }
  cursor[parts.at(-1)] = value;
}
