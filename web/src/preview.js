import { clone } from './flow.js';
import { normalizeCondition } from './conditions.js';

export function formatValue(value) {
  if (typeof value !== 'string') return JSON.stringify(value, null, 2) ?? '';
  try { return JSON.stringify(JSON.parse(value), null, 2); } catch { return value; }
}

export function pathPreview(endpoint, row, baseURL) {
  const nodes = new Map(endpoint.flow.nodes.map(node => [node.id, node]));
  const request = endpoint.flow.nodes.find(node => node.type === 'request');
  const response = nodes.get(row.responseID);
  const headers = {};
  let delay = 0;
  for (const id of row.pathIDs) {
    const node = nodes.get(id);
    if (['headers', 'response'].includes(node.type)) {
      for (const [key, value] of Object.entries(node.config.headers || {})) {
        const previous = Object.keys(headers).find(name => name.toLowerCase() === key.toLowerCase());
        if (previous) delete headers[previous];
        Object.defineProperty(headers, key, { value, enumerable: true, configurable: true });
      }
    }
    if (['delay', 'response'].includes(node.type)) delay += node.config.delay_ms || 0;
  }
  return {
    name: endpoint.name, method: endpoint.method, path: endpoint.path, url: baseURL + endpoint.path,
    example: clone(request.config.example || {}),
    conditions: row.conditions.map(item => ({ matched: item.matched, ...normalizeCondition(nodes.get(item.id).config) })),
    status: response.config.status, delay, headers, body: response.config.body || '', isDefault: row.isDefault
  };
}
