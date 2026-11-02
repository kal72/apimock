import test from 'node:test';
import assert from 'node:assert/strict';
import { conditionDiagnostics, normalizeCondition, typedCondition, flowConditionWarnings } from './conditions.js';
import { createEditor } from './editor.js';
import { defaultFlow, graphIssues } from './flow.js';

function editor(mode='node',path='/users/:id') { const state=createEditor();state.mode=mode;state.endpoint={id:'e',name:'Users',method:'POST',path,flow:defaultFlow()};state.insertAfter='request';return state; }

test('typed conditions start with one nonempty required field and AND',()=>{
  for(const kind of ['header','query','body','path']){const c=typedCondition(kind,'/users/:userId');assert.equal(c.condition_type,kind);assert.equal(c.match,'all');assert.equal(c.rules.length,1);assert.equal(c.rules[0].required,true);assert.deepEqual(conditionDiagnostics(c,'/users/:userId').errors,[]);}
  assert.equal(typedCondition('path','/users/:userId').rules[0].field,'userId');
  assert.throws(()=>typedCondition('path','/users'),/path parameter/);
});

test('legacy conditions upgrade in the editor without changing empty-value semantics',()=>{
  const c=normalizeCondition({field:'headers.Authorization',operator:'exists'});
  assert.equal(c.condition_type,'header');assert.equal(c.rules[0].field,'Authorization');assert.equal(c.rules[0].required,false);
});

test('AND rejects contradictions but OR and valid additional fields are accepted',()=>{
  const c=typedCondition('header');c.rules=[{field:'X-State',operator:'equals',value:'active',required:false},{field:'x-state',operator:'equals',value:'inactive',required:false}];
  assert.ok(conditionDiagnostics(c).errors.some(error=>error.includes('AND conflict')));
  c.match='any';assert.deepEqual(conditionDiagnostics(c).errors,[]);
  c.match='all';c.rules[1].field='X-Other';assert.deepEqual(conditionDiagnostics(c).errors,[]);
  c.rules[1]={...c.rules[0]};assert.ok(conditionDiagnostics(c).warnings.length);assert.deepEqual(conditionDiagnostics(c).errors,[]);
});

test('required cannot be combined with an empty comparison',()=>{
  const c=typedCondition('query');c.rules[0].operator='empty';assert.ok(conditionDiagnostics(c).errors.length);
  c.rules[0].required=false;assert.deepEqual(conditionDiagnostics(c).errors,[]);
});

test('Node palette selection creates and connects immediately without opening another form',()=>{
  const state=editor();state.openAddScenario();assert.equal(state.modal,'palette');
  state.quickAddCondition('header');assert.equal(state.modal,null);assert.equal(state.conditions.length,1);assert.equal(state.conditions[0].config.condition_type,'header');assert.equal(state.dirty,true);assert.deepEqual(graphIssues(state.endpoint.flow),[]);
  state.quickAddCondition('header');assert.ok(flowConditionWarnings(state.endpoint.flow).some(warning=>warning.includes('equivalent')));
});

test('a selected matching output inserts an additional typed check before success',()=>{
  const state=editor();state.quickAddCondition('header');const header=state.conditions[0];const previous=state.targetFor(header.id,'true');
  state.connecting={from:header.id,branch:'true'};state.quickAddCondition('query');
  const query=state.conditions.find(node=>node.config.condition_type==='query');
  assert.equal(state.targetFor(header.id,'true'),query.id);assert.equal(state.targetFor(query.id,'true'),previous);
  assert.deepEqual(state.flowIssues,[]);assert.equal(state.responses.length,2);
});

test('path quick-add is disabled and does not mutate a route without parameters',()=>{
  const state=editor('node','/users');const before=JSON.stringify(state.endpoint);state.openAddScenario();state.quickAddCondition('path');
  assert.equal(state.canAddCondition('path'),false);assert.equal(JSON.stringify(state.endpoint),before);assert.equal(state.modal,'palette');
});

test('multiple body fields, Form mode and OR persist from one popup',()=>{
  const state=editor('table');state.openAddScenario();state.chooseScenarioTemplate('body');const c=state.scenarioForm.conditions.at(-1);
  c.config.body_mode='form';c.config.match='any';c.config.rules=[{field:'email',operator:'exists',value:'',required:true},{field:'role',operator:'equals',value:'admin',required:false}];
  state.scenarioForm.bodyMode='form';state.scenarioForm.form='{"email":["demo@example.com"]}';state.saveScenario();
  assert.equal(state.modal,null);assert.equal(state.conditions[0].config.rules.length,2);assert.equal(state.conditions[0].config.body_mode,'form');assert.equal(JSON.parse(state.testInput).form.email[0],'demo@example.com');
});

test('Raw Text checks use $raw while raw JSON array examples persist',()=>{
  const state=editor('table');state.openAddScenario();state.chooseScenarioTemplate('body');const c=state.scenarioForm.conditions.at(-1);
  c.config.raw_format='text';state.changeBodyMode(c);assert.equal(c.config.rules[0].field,'$raw');state.addConditionField(c);assert.equal(c.config.rules[1].field,'$raw');
  state.scenarioForm.bodyMode='text';state.scenarioForm.rawBody='hello';state.saveScenario();assert.equal(JSON.parse(state.testInput).raw_body,'hello');
  state.editScenario(state.fallbackRow);state.scenarioForm.bodyMode='json';state.scenarioForm.body='[{"id":0}]';state.saveScenario();assert.equal(JSON.parse(state.testInput).raw_body,'[{"id":0}]');
});
