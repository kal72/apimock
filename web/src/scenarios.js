import { conditionKinds, typedCondition, normalizeCondition } from './conditions.js';
import { clone, defaultExample, graphIssues, newID, setConnection } from './flow.js';

export const responseTemplates = [
  { key: 'success', name: 'Success · 200', status: 200, body: '{\n  "success": true\n}' },
  { key: 'bad-request', name: 'Bad request · 400', status: 400, body: '{\n  "error": "Required field is missing or empty"\n}' },
  { key: 'not-found', name: 'Not found · 404', status: 404, body: '{\n  "error": "Not found"\n}' },
  { key: 'unauthorized', name: 'Unauthorized · 401', status: 401, body: '{\n  "error": "Unauthorized"\n}' },
  { key: 'error', name: 'Server error · 500', status: 500, body: '{\n  "error": "Internal server error"\n}' }
];

const legacyTemplates = [
  { key: 'custom', name: 'Custom condition', description: 'Define your own request fields and response.', field: 'params.id', operator: 'equals', value: '123', response: 'success', fallback: 'keep' },
  { key: 'product', name: 'Product found', description: 'ID matches → 200; default → 404.', field: 'params.id', operator: 'equals', value: '123', response: 'success', fallback: 'not-found' },
  { key: 'auth', name: 'Authorization header', description: 'Token matches → 200; default → 401.', field: 'headers.authorization', operator: 'equals', value: 'Bearer demo', response: 'success', fallback: 'unauthorized' },
  { key: 'query', name: 'Simulate an error', description: 'Query scenario=error → 500; default → 200.', field: 'query.scenario', operator: 'equals', value: 'error', response: 'error', fallback: 'success' }
];

export const conditionTemplates = conditionKinds.map(kind => ({ ...kind, response: 'success', fallback: 'keep' }));

export function projectScenarios(flow, limit = 128) {
  const nodes = new Map(flow.nodes.map(node => [node.id, node]));
  const rows = [];
  let truncated = false;
  const walk = (id, path, conditions, active) => {
    if (rows.length >= limit) { truncated = true; return; }
    if (active.has(id)) return;
    const node = nodes.get(id);
    if (!node) return;
    const nextPath = [...path, id];
    if (node.type === 'response') {
      const isDefault = conditions.every(item => !item.matched);
      rows.push({
        key: conditions.map(item => `${item.id}:${item.matched}`).join('|') + ':' + nextPath.join('>'),
        responseID: id, pathIDs: nextPath, conditions, isDefault,
        label: isDefault ? 'Default response' : conditions.filter(item => item.matched).map(item => nodes.get(item.id).label).join(' → '),
        status: node.config.status, body: node.config.body || '',
        headerIDs: nextPath.filter(item => nodes.get(item).type === 'headers'),
        delayIDs: nextPath.filter(item => nodes.get(item).type === 'delay')
      });
      return;
    }
    const nextActive = new Set(active).add(id);
    const branches = node.type === 'condition' ? ['true', 'false'] : [''];
    for (const branch of branches) {
      const edge = flow.edges.find(item => item.from === id && (item.branch || '') === branch);
      if (edge) walk(edge.to, nextPath, node.type === 'condition' ? [...conditions, { id, matched: branch === 'true' }] : conditions, nextActive);
    }
  };
  const request = flow.nodes.find(node => node.type === 'request');
  if (request) walk(request.id, [], [], new Set());
  return { rows: rows.map((row, index) => ({ ...row, order: index + 1 })), truncated };
}

export function defaultScenario(flow) {
  const pathIDs = [], conditions = [], visited = new Set();
  let node = flow.nodes.find(item => item.type === 'request');
  while (node && !visited.has(node.id)) {
    visited.add(node.id); pathIDs.push(node.id);
    if (node.type === 'response') return { key: conditions.map(item => `${item.id}:${item.matched}`).join('|') + ':' + pathIDs.join('>'), responseID: node.id, pathIDs, conditions, isDefault: true, label: 'Default response', status: node.config.status, body: node.config.body || '', headerIDs: pathIDs.filter(id => flow.nodes.find(item => item.id === id).type === 'headers'), delayIDs: pathIDs.filter(id => flow.nodes.find(item => item.id === id).type === 'delay') };
    const branch = node.type === 'condition' ? 'false' : '';
    if (node.type === 'condition') conditions.push({ id: node.id, matched: false });
    const edge = flow.edges.find(item => item.from === node.id && (item.branch || '') === branch);
    node = flow.nodes.find(item => item.id === edge?.to);
  }
  return null;
}

export function addConditionScenario(flow, templateKey = 'header', options = {}) {
  const template = conditionTemplates.find(item => item.key === templateKey) || legacyTemplates.find(item => item.key === templateKey);
  const fallback = defaultScenario(flow);
  if (!template || !fallback) throw Error('Connect the default flow before adding conditions.');
  const fallbackResponse = flow.nodes.find(node => node.id === fallback.responseID);
  const predecessorID = options.source?.from || fallback.pathIDs.at(-2);
  const predecessor = flow.nodes.find(node => node.id === predecessorID);
  if(!predecessor)throw Error('Connection source not found.');
  const branch = options.source ? options.source.branch || '' : predecessor.type === 'condition' ? 'false' : '';
  const edge = flow.edges.find(item=>item.from===predecessorID && (item.branch || '')===branch);
  const target = flow.nodes.find(node=>node.id===edge?.to);
  if(!target)throw Error('Connect the selected output first.');
  const separate = target.id === fallbackResponse.id;
  const condition = { id: newID(), type: 'condition', label: `${template.field ? 'Condition' : template.name} ${flow.nodes.filter(node => node.type === 'condition').length + 1}`, x: target.x, y: target.y, config: template.field ? { field: template.field, operator: template.operator, value: template.value } : typedCondition(templateKey, options.path || '/:id', options.example || {}) };
  let response=target;
  flow.nodes.push(condition);
  if(separate){
    response={ id:newID(),type:'response',label:'Response '+condition.label,x:condition.x+320,y:condition.y,config:{status:200,headers:{'Content-Type':'application/json'},body:'{"success":true}'}};
    flow.nodes.push(response);fallbackResponse.x=condition.x;fallbackResponse.y=condition.y+220;
    applyResponseTemplate(response,template.response);
  }else{
    const shifted=new Set();const shift=id=>{if(shifted.has(id))return;shifted.add(id);const node=flow.nodes.find(item=>item.id===id);if(node)node.x+=320;flow.edges.filter(item=>item.from===id).forEach(item=>shift(item.to));};shift(target.id);
  }
  setConnection(flow,predecessor.id,branch,condition.id);
  setConnection(flow,condition.id,'true',response.id);
  setConnection(flow,condition.id,'false',fallbackResponse.id);
  if(template.fallback !== 'keep')applyResponseTemplate(fallbackResponse,template.fallback);
  return projectScenarios(flow).rows.find(row=>row.conditions.some(item=>item.id===condition.id && item.matched)) || {conditionID:condition.id};
}

export function applyResponseTemplate(node, key) {
  if (key === 'keep') return;
  const template = responseTemplates.find(item => item.key === key);
  if (!template) throw Error('Response template not found.');
  node.label = template.name.split(' · ')[0];
  node.config = { ...node.config, status: template.status, body: template.body, headers: { 'Content-Type': 'application/json' }, delay_ms: 0 };
}

export function responseForm(node) {
  return { id: node.id, label: node.label, status: node.config.status, body: node.config.body || '', delay_ms: node.config.delay_ms || 0, headers: Object.entries(node.config.headers || {}).map(([key, value]) => ({ key, value })), template: 'keep' };
}

export function scenarioForm(endpoint, flow, row) {
  const request = flow.nodes.find(node => node.type === 'request');
  const example = request.config.example || defaultExample();
  const fallback = defaultScenario(flow);
  const fallbackResponse = flow.nodes.find(node => node.id === fallback?.responseID);
  return {
    route: { name: endpoint.name, method: endpoint.method, path: endpoint.path }, requestID: request.id,
    params: JSON.stringify(example.params || {}, null, 2), query: JSON.stringify(example.query || {}, null, 2), body: example.body_mode === 'json' && example.raw_body ? example.raw_body : JSON.stringify(example.body || {}, null, 2),
    bodyMode: example.body_mode || 'json', rawBody: example.raw_body || '', form: JSON.stringify(example.form || {},null,2),
    headers: Object.entries(example.headers || {}).map(([key, value]) => ({ key, value })),
    conditions: row.conditions.map(item => ({ ...item, label: flow.nodes.find(node => node.id === item.id).label, config: normalizeCondition(flow.nodes.find(node => node.id === item.id).config) })),
    response: responseForm(flow.nodes.find(node => node.id === row.responseID)),
    fallback: fallbackResponse && fallbackResponse.id !== row.responseID ? responseForm(fallbackResponse) : null,
    legacyHeaders: row.headerIDs.map(id => ({ id, label: flow.nodes.find(node => node.id === id).label, headers: Object.entries(flow.nodes.find(node => node.id === id).config.headers || {}).map(([key, value]) => ({ key, value })) })),
    legacyDelays: row.delayIDs.map(id => ({ id, label: flow.nodes.find(node => node.id === id).label, delay_ms: flow.nodes.find(node => node.id === id).config.delay_ms })),
    connections: row.pathIDs.slice(0, -1).map((id, index) => clone(flow.edges.find(edge => edge.from === id && edge.to === row.pathIDs[index + 1] && (flow.nodes.find(node => node.id === id).type !== 'condition' || edge.branch === String(row.conditions.find(item => item.id === id).matched))))),
    isDefault: row.isDefault, label: row.label
  };
}

export function removeScenario(flow, conditionID) {
  const fallback = flow.edges.find(edge => edge.from === conditionID && edge.branch === 'false');
  if (!fallback) throw Error('This condition has no connected default response.');
  flow.edges = flow.edges.filter(edge => edge.from !== conditionID).map(edge => edge.to === conditionID ? { ...edge, to: fallback.to } : edge);
  flow.nodes = flow.nodes.filter(node => node.id !== conditionID);
  const reachable = new Set();
  const visit = id => { if (reachable.has(id)) return; reachable.add(id); flow.edges.filter(edge => edge.from === id).forEach(edge => visit(edge.to)); };
  visit(flow.nodes.find(node => node.type === 'request').id);
  flow.nodes = flow.nodes.filter(node => reachable.has(node.id));
  flow.edges = flow.edges.filter(edge => reachable.has(edge.from) && reachable.has(edge.to));
  const issues = graphIssues(flow);
  if (issues.length) throw Error(issues.join(' '));
}
