import { scenarioBranches } from './branches.js';
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

// Implication is conservative: only proven relationships produce unreachable warnings.
const sourceKey = c => c.condition_type === 'body' ? `body:${c.body_mode}:${c.body_mode === 'raw' ? c.raw_format : ''}` : c.condition_type;
const terms = c => c.match === 'any' ? c.rules.map(rule => [rule]) : [c.rules];
const sameField = (c,a,b) => c.condition_type === 'header' ? a.field.toLowerCase() === b.field.toLowerCase() : a.field === b.field;
function nonemptyLiteral(c,value) {
  if (!String(value).trim()) return false;
  // A JSON field may also be a string containing the same text as an empty JSON value.
  if (sourceKey(c) === 'body:raw:json') {
    try { const parsed=JSON.parse(value); if (parsed===null || Array.isArray(parsed)&&!parsed.length || typeof parsed==='object'&&!Object.keys(parsed).length) return false; } catch {}
  }
  return true;
}
function impliesRule(c,a,b) {
  if (!sameField(c,a,b)) return false;
  const nonempty=a.required || a.operator==='equals'&&nonemptyLiteral(c,a.value || '');
  if (b.required && !nonempty) return false;
  if (a.operator===b.operator && (['exists','empty'].includes(a.operator) || (a.value || '')===(b.value || ''))) return true;
  if (b.operator==='exists' && a.operator!=='empty') return true;
  if (a.operator==='equals') {
    if (b.operator==='contains') return (a.value || '').includes(b.value || '');
    if (b.operator==='not_equals') return a.value!==b.value;
  }
  if (a.operator==='contains' && b.operator==='contains') return (a.value || '').includes(b.value || '');
  return false;
}
function implies(later,earlier) {
  if (sourceKey(later)!==sourceKey(earlier)) return false;
  return terms(later).every(term => terms(earlier).some(target => target.every(rule => term.some(candidate => impliesRule(later,candidate,rule)))));
}
function incompatible(c,a,b) {
  if (!sameField(c,a,b)) return false;
  if (a.operator==='equals' && b.operator==='equals') return a.value!==b.value;
  if (a.operator==='equals' && b.operator==='not_equals' || a.operator==='not_equals' && b.operator==='equals') return a.value===b.value;
  if (a.operator==='equals' && b.operator==='contains') return !(a.value || '').includes(b.value || '');
  if (b.operator==='equals' && a.operator==='contains') return !(b.value || '').includes(a.value || '');
  if (a.operator==='empty') return b.required || b.operator==='equals'&&nonemptyLiteral(c,b.value || '');
  if (b.operator==='empty') return a.required || a.operator==='equals'&&nonemptyLiteral(c,a.value || '');
  return false;
}
function mayOverlap(a,b) {
  if (a.condition_type==='body' && b.condition_type==='body' && sourceKey(a)!==sourceKey(b)) return false;
  if (sourceKey(a)!==sourceKey(b)) return true;
  return terms(a).some(x => terms(b).some(y => !x.some(left => y.some(right => incompatible(a,left,right)))));
}
export function flowConditionConflicts(flow) {
  const model=scenarioBranches(flow);
  // Nested legacy graphs do not share a single first-match priority list.
  if (!model) return [];
  const conflicts=[];
  for (let j=1;j<model.branches.length;j++) {
    const later=model.branches[j], b=normalizeCondition(later.condition.config);
    if (!later.condition.config.condition_type) continue;
    if (conditionDiagnostics(b).errors.length) continue;
    for (let i=0;i<j;i++) {
      const earlier=model.branches[i], a=normalizeCondition(earlier.condition.config);
      if (!earlier.condition.config.condition_type) continue;
      if (conditionDiagnostics(a).errors.length) continue;
      const covered=implies(b,a), duplicate=covered&&implies(a,b);
      if (!covered && !mayOverlap(a,b)) continue;
      const prefix=`#${j+1} ${later.condition.label}`;
      conflicts.push({kind:duplicate?'duplicate':covered?'shadowed':'overlap',conditionID:later.condition.id,earlierID:earlier.condition.id,
        message:duplicate?`${prefix} is equivalent to #${i+1} ${earlier.condition.label}. This duplicate cannot win at its current priority.`:
        covered?`${prefix} is covered by #${i+1} ${earlier.condition.label} and cannot win at its current priority. Move the more specific condition earlier or refine its rules.`:
        `${prefix} may overlap with #${i+1} ${earlier.condition.label}. If both match, #${i+1} wins. Verify the intended result in Playground.`});
    }
  }
  return conflicts;
}
export function flowConditionWarnings(flow) {
  const warnings=flow.nodes.filter(node=>node.type==='condition').flatMap(node=>conditionDiagnostics(node.config).warnings.map(message=>`${node.label}: ${message}`));
  warnings.push(...flowConditionConflicts(flow).map(conflict=>conflict.message));
  const limit=30;
  return warnings.length>limit?[...warnings.slice(0,limit),`${warnings.length-limit} more warnings. Resolve earlier conflicts and review again.`]:warnings;
}
