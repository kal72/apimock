export const clone = value => JSON.parse(JSON.stringify(value));

export function newID() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
  const bytes = new Uint8Array(16);
  globalThis.crypto.getRandomValues(bytes);
  return Array.from(bytes, value => value.toString(16).padStart(2, '0')).join('');
}

export const defaultExample = () => ({ params: {}, query: {}, headers: {}, body: {} });
export function defaultFlow() {
  return {
    nodes: [
      { id: 'request', type: 'request', label: 'Incoming request', x: 100, y: 130, config: { example: defaultExample() } },
      { id: 'response', type: 'response', label: 'Success response', x: 500, y: 130, config: { status: 200, headers: { 'Content-Type': 'application/json' }, body: '{\n  "message": "Hello from Mocklab"\n}' } }
    ],
    edges: [{ from: 'request', to: 'response', branch: '' }]
  };
}

export function outgoingBranches(node) {
  return node.type === 'response' ? [] : node.type === 'condition' ? ['true', 'false'] : [''];
}

export function reaches(flow, from, target, visited = new Set()) {
  if (from === target) return true;
  if (visited.has(from)) return false;
  visited.add(from);
  return flow.edges.filter(edge => edge.from === from).some(edge => reaches(flow, edge.to, target, visited));
}

export function setConnection(flow, from, branch, to) {
  const source = flow.nodes.find(node => node.id === from);
  const target = flow.nodes.find(node => node.id === to);
  if (!source || !outgoingBranches(source).includes(branch)) throw Error('Invalid connection source or branch.');
  if (to && (!target || target.type === 'request' || reaches(flow, to, from))) throw Error('Connections cannot form cycles or point to Incoming request.');
  flow.edges = flow.edges.filter(edge => !(edge.from === from && (edge.branch || '') === branch));
  if (to) flow.edges.push({ from, to, branch });
}

export function insertBlock(flow, template, from, branch = '') {
  const source = flow.nodes.find(node => node.id === from);
  if (!source || !outgoingBranches(source).includes(branch)) throw Error('Choose the step and branch where this setting should be added.');
  const previousEdge = flow.edges.find(edge => edge.from === from && (edge.branch || '') === branch);
  const target = flow.nodes.find(node => node.id === previousEdge?.to);
  let terminal = target;
  const visited = new Set();
  while (template.type === 'response' && terminal && ['headers', 'delay'].includes(terminal.type) && !visited.has(terminal.id)) {
    visited.add(terminal.id);
    terminal = flow.nodes.find(node => node.id === flow.edges.find(edge => edge.from === terminal.id)?.to);
  }
  if (template.type === 'response' && terminal?.type === 'response') {
    terminal.label = template.name;
    terminal.config = clone(template.config);
    return terminal;
  }
  if (template.type === 'response' && target) throw Error('Select the connection immediately before a response to replace its template.');
  if (template.type !== 'response' && !target) throw Error('Connect this step to a response first.');
  const node = { id: newID(), type: template.type, label: template.name, x: Math.min(1400, source.x + 340), y: source.y, config: clone(template.config) };
  flow.nodes.push(node);
  setConnection(flow, from, branch, node.id);
  if (node.type !== 'response') {
    setConnection(flow, node.id, node.type === 'condition' ? 'true' : '', target.id);
    if (node.type === 'condition') {
      const fallback = { id: newID(), type: 'response', label: 'Condition not matched', x: Math.min(1480, node.x + 340), y: Math.min(800, node.y + 230), config: { status: 404, headers: { 'Content-Type': 'application/json' }, body: '{\n  "error": "Condition not matched"\n}' } };
      flow.nodes.push(fallback);
      setConnection(flow, node.id, 'false', fallback.id);
    }
    // Move the original downstream path right to keep the inserted block visible.
    const moved = new Set();
    const shift = id => {
      if (moved.has(id) || id === node.id || id === from) return;
      moved.add(id);
      const downstream = flow.nodes.find(item => item.id === id);
      if (downstream) downstream.x = Math.min(1480, Math.max(downstream.x, node.x + 340));
      flow.edges.filter(edge => edge.from === id).forEach(edge => shift(edge.to));
    };
    shift(target.id);
  }
  return node;
}

export function preferredResponse(flow) {
  return flow.nodes.find(node => node.type === 'response' && node.config.status >= 200 && node.config.status < 300)
    || flow.nodes.find(node => node.type === 'response');
}

export function directResponseFlow(flow, responseID) {
  const request = clone(flow.nodes.find(node => node.type === 'request') || defaultFlow().nodes[0]);
  const fresh = () => ({ ...defaultFlow().nodes[1], id: newID() });
  const existing = flow.nodes.find(node => node.type === 'response' && node.id === responseID) || preferredResponse(flow);
  const response = clone(responseID === '' ? fresh() : existing || fresh());
  if (existing) { response.x = existing.x; response.y = existing.y; }
  const pathTarget = existing?.id || response.id;
  const visited = new Set();
  const findPath = id => {
    if (id === pathTarget) return [response];
    if (visited.has(id)) return null;
    visited.add(id);
    const node = flow.nodes.find(item => item.id === id);
    if (!node) return null;
    for (const edge of flow.edges.filter(item => item.from === id)) {
      const rest = findPath(edge.to);
      if (rest) return [clone(node), ...rest];
    }
    return null;
  };
  // Preserve the selected path's header/delay settings while removing branching.
  const path = findPath(request.id) || [request, response];
  const nodes = path.filter(node => node.type !== 'condition');
  const edges = nodes.slice(0, -1).map((node, index) => ({ from: node.id, to: nodes[index + 1].id, branch: '' }));
  return { nodes, edges };
}

export function removeConfiguration(flow, id) {
  const node = flow.nodes.find(item => item.id === id);
  if (!node || node.type === 'request') throw Error('Incoming request cannot be deleted.');
  if (node.type === 'response') throw Error('A response is required. Edit the response or use a direct response.');
  const branch = node.type === 'condition' ? 'true' : '';
  const target = flow.edges.find(edge => edge.from === id && (edge.branch || '') === branch)?.to;
  if (!target) throw Error('Connect the next path before deleting this setting.');
  flow.edges = flow.edges.filter(edge => edge.from !== id).map(edge => edge.to === id ? { ...edge, to: target } : edge);
  flow.nodes = flow.nodes.filter(item => item.id !== id);
  // Only the discarded branch's exclusively reachable nodes are pruned.
  const request = flow.nodes.find(item => item.type === 'request');
  const reachable = new Set();
  const visit = current => {
    if (reachable.has(current)) return;
    reachable.add(current);
    flow.edges.filter(edge => edge.from === current).forEach(edge => visit(edge.to));
  };
  if (request) visit(request.id);
  flow.nodes = flow.nodes.filter(item => reachable.has(item.id));
  flow.edges = flow.edges.filter(edge => reachable.has(edge.from) && reachable.has(edge.to));
}

export function graphIssues(flow) {
  const issues = [];
  const requests = flow.nodes.filter(node => node.type === 'request');
  if (requests.length !== 1) return ['The flow must have exactly one Incoming request.'];
  const seen = new Set();
  const active = new Set();
  const walk = id => {
    if (active.has(id)) { issues.push('Connections form a cycle.'); return; }
    if (seen.has(id)) return;
    seen.add(id); active.add(id);
    const node = flow.nodes.find(item => item.id === id);
    if (!node) { issues.push('Connection destination not found.'); active.delete(id); return; }
    const edges = flow.edges.filter(edge => edge.from === id);
    const branches = outgoingBranches(node);
    if (edges.length !== branches.length || branches.some(branch => edges.filter(edge => (edge.branch || '') === branch).length !== 1)) issues.push(`${node.label}: connections are incomplete.`);
    edges.forEach(edge => walk(edge.to));
    active.delete(id);
  };
  walk(requests[0].id);
  if (flow.nodes.some(node => !seen.has(node.id))) issues.push('Some settings are not connected to Incoming request.');
  return [...new Set(issues)];
}

export function nodeRows(flow) {
  const rows = [];
  const visited = new Set();
  const walk = id => {
    if (visited.has(id)) return;
    const node = flow.nodes.find(item => item.id === id);
    if (!node) return;
    visited.add(id); rows.push(node);
    outgoingBranches(node).forEach(branch => {
      const edge = flow.edges.find(item => item.from === id && (item.branch || '') === branch);
      if (edge) walk(edge.to);
    });
  };
  const request = flow.nodes.find(node => node.type === 'request');
  if (request) walk(request.id);
  return [...rows, ...flow.nodes.filter(node => !visited.has(node.id))];
}

export function connectionPoints(flow, edge) {
  const from = flow.nodes.find(node => node.id === edge.from);
  const to = flow.nodes.find(node => node.id === edge.to);
  return { x1: (from?.x || 0) + 280, y1: (from?.y || 0) + (from?.type === 'condition' ? edge.branch === 'false' ? 126 : 94 : 110), x2: to?.x || 0, y2: (to?.y || 0) + 110 };
}

export function parseObject(text, label) {
  const value = JSON.parse(text);
  if (!value || Array.isArray(value) || typeof value !== 'object') throw Error(`${label} must be a JSON object.`);
  return value;
}

export function headersFromRows(rows) {
  const result = {};
  const keys = new Set();
  for (const row of rows) {
    if (!row.key.trim() && !row.value) continue;
    const key = row.key.trim();
    if (!/^[!#$%&'*+.^_`|~0-9a-z-]+$/i.test(key) || /[\r\n]/.test(row.value)) throw Error('Invalid header name or value.');
    if (keys.has(key.toLowerCase())) throw Error(`Duplicate header ${key}.`);
    keys.add(key.toLowerCase());
    Object.defineProperty(result, key, { value: row.value, enumerable: true, configurable: true });
  }
  return result;
}
