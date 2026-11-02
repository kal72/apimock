import { clone } from './flow.js';

export const conditionKinds = [
  { key: 'header', name: 'Header', icon: '≡', description: 'Check one or more request headers.' },
  { key: 'query', name: 'Query Params', icon: '?', description: 'Check one or more query parameters.' },
  { key: 'body', name: 'Body', icon: '{}', description: 'Check Form, Raw JSON, or Raw Text fields.' },
  { key: 'path', name: 'Path Params', icon: '/', description: 'Check parameters available in the route.' }
];

export function pathParameters(path) { return path.split('/').filter(part => part.startsWith(':') && part.length > 1).map(part => part.slice(1)); }

export function typedCondition(kind, path = '/', example = {}) {
  const fields = { header: 'Authorization', query: Object.keys(example.query || {})[0] || 'page', body: Object.keys(example.body || {})[0] || 'id', path: pathParameters(path)[0] };
  if (!conditionKinds.some(item => item.key === kind)) throw Error('Unknown condition type.');
  if (!fields[kind]) throw Error('Add a path parameter such as :id to the route first.');
  return { condition_type: kind, match: 'all', ...(kind === 'body' ? { body_mode: 'raw', raw_format: 'json' } : {}), rules: [{ field: fields[kind], operator: 'exists', value: '', required: true }] };
}

export function normalizeCondition(config) {
  if (config.condition_type && Array.isArray(config.rules)) return clone(config);
  const [prefix, ...parts] = (config.field || 'params.id').split('.');
  const kind = { headers: 'header', query: 'query', body: 'body', params: 'path' }[prefix] || 'path';
  return { condition_type: kind, match: 'all', ...(kind === 'body' ? { body_mode: 'raw', raw_format: 'json' } : {}), rules: [{ field: parts.join('.') || 'id', operator: config.operator || 'exists', value: config.value || '', required: false }] };
}

export function conditionDiagnostics(config, path) {
  const c = normalizeCondition(config), errors = [], warnings = [];
  if (!conditionKinds.some(item => item.key === c.condition_type)) errors.push('Invalid condition type.');
  if (!['all', 'any'].includes(c.match)) errors.push('Choose AND or OR.');
  if (!c.rules.length || c.rules.length > 50) errors.push('A node requires 1–50 fields.');
  if (c.condition_type === 'body' && (!['form', 'raw'].includes(c.body_mode) || c.body_mode === 'raw' && !['json', 'text'].includes(c.raw_format))) errors.push('Choose Form or Raw body mode with JSON/Text format.');
  const seen = new Set();
  for (const rule of c.rules) {
    if (typeof rule.field !== 'string' || !rule.field.trim() || rule.field.length > 256) { errors.push('Field name is required (maximum 256 characters).'); continue; }
    if (c.condition_type === 'header' && !/^[!#$%&'*+.^_`|~0-9a-z-]+$/i.test(rule.field)) errors.push('Invalid header name.');
    if (c.condition_type === 'path' && path !== undefined && !pathParameters(path).includes(rule.field)) errors.push(`Path parameter ${rule.field} is not available in the route.`);
    if (c.condition_type === 'body' && c.body_mode === 'raw' && c.raw_format === 'text' && rule.field !== '$raw') errors.push('Raw Text uses the $raw field.');
    if (c.condition_type === 'body' && c.body_mode === 'raw' && c.raw_format === 'json' && rule.field !== '$' && rule.field.split('.').some(part=>!part)) errors.push('Invalid JSON field path.');
    if (!['exists', 'equals', 'not_equals', 'contains', 'empty'].includes(rule.operator)) errors.push('Invalid condition operator.');
    if (rule.required && (rule.operator === 'empty' || rule.operator === 'equals' && !(rule.value || '').trim())) errors.push(`${rule.field}: Required cannot be combined with an empty-value check.`);
    const field = c.condition_type === 'header' ? rule.field.toLowerCase() : rule.field;
    const key = JSON.stringify([field, rule.operator, ['exists','empty'].includes(rule.operator) ? '' : rule.value || '', !!rule.required]);
    if (seen.has(key)) warnings.push(`Duplicate rule for ${rule.field}.`);
    seen.add(key);
  }
  if (c.match === 'all') c.rules.forEach((a, index) => c.rules.slice(index + 1).forEach(b => {
    const same = c.condition_type === 'header' ? a.field?.toLowerCase() === b.field?.toLowerCase() : a.field === b.field;
    if (!same) return;
    if (a.operator === 'equals' && b.operator === 'equals' && a.value !== b.value || a.operator === 'equals' && b.operator === 'not_equals' && a.value === b.value || b.operator === 'equals' && a.operator === 'not_equals' && a.value === b.value || a.required && b.operator === 'empty' || b.required && a.operator === 'empty' || a.operator === 'equals' && b.operator === 'contains' && !(a.value || '').includes(b.value || '') || b.operator === 'equals' && a.operator === 'contains' && !(b.value || '').includes(a.value || '')) errors.push(`${a.field}: AND conflict: these rules cannot be satisfied together.`);
  }));
  return { errors: [...new Set(errors)], warnings: [...new Set(warnings)] };
}

export function conditionSummary(config) {
  const c = normalizeCondition(config);
  return c.rules.map(rule => `${rule.field}${rule.required ? ' [required]' : ''} ${rule.operator}${['exists', 'empty'].includes(rule.operator) ? '' : ' ' + (rule.value || '')}`).join(c.match === 'any' ? ' OR ' : ' AND ');
}

export function conditionKindLabel(config) {
  const c = normalizeCondition(config);
  return conditionKinds.find(kind => kind.key === c.condition_type)?.name || 'Condition';
}

export function flowConditionWarnings(flow) {
  const seen = new Map(), warnings = [];
  for (const node of flow.nodes.filter(item => item.type === 'condition')) {
    const c = normalizeCondition(node.config);
    warnings.push(...conditionDiagnostics(c).warnings.map(message => `${node.label}: ${message}`));
    const key = JSON.stringify([c.condition_type, c.match, c.condition_type==='body' ? c.body_mode : '', c.condition_type==='body' && c.body_mode==='raw' ? c.raw_format : '', c.rules.map(rule => [c.condition_type === 'header' ? rule.field.toLowerCase() : rule.field, rule.operator, ['exists','empty'].includes(rule.operator) ? '' : rule.value || '', !!rule.required]).sort()]);
    if (seen.has(key)) warnings.push(`${node.label} is equivalent to ${seen.get(key)}. Review priority; the first matching rule is used.`);
    else seen.set(key, node.label);
  }
  return warnings;
}
