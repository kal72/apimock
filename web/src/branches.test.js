import test from 'node:test';
import assert from 'node:assert/strict';
import { defaultFlow, graphIssues, insertBlock } from './flow.js';
import { addConditionScenario } from './scenarios.js';
import { arrangeBranches, branchRows, branchEdges, scenarioBranches, setPriority } from './branches.js';
import { createEditor } from './editor.js';

test('independent scenarios fan out from request and hide failed-condition edges',()=>{
  const flow=defaultFlow();addConditionScenario(flow,'header');addConditionScenario(flow,'query');arrangeBranches(flow);
  const model=scenarioBranches(flow),edges=branchEdges(flow),rows=branchRows(flow);
  assert.equal(model.branches.length,2);assert.equal(edges.filter(edge=>edge.from===model.request.id).length,3);
  assert.ok(edges.every(edge=>edge.branch!=='false'));assert.equal(rows.length,3);
  assert.equal(rows[0].conditions.length,1);assert.equal(rows[1].conditions.length,1);assert.deepEqual(rows[2].conditions,[]);
  assert.equal(rows[2].priority,null);assert.equal(flow.layout,'scenarios');assert.deepEqual(graphIssues(flow),[]);
});

test('priority reorder persists execution order, response connections, and fixed fallback',()=>{
  const flow=defaultFlow();addConditionScenario(flow,'header');addConditionScenario(flow,'query');
  const before=scenarioBranches(flow),id=before.branches[1].condition.id;
  const response=before.branches[1].response.id,fallback=before.fallback.id;
  setPriority(flow,id,1);
  const after=scenarioBranches(JSON.parse(JSON.stringify(flow)));
  assert.equal(after.branches[0].condition.id,id);assert.equal(after.branches[0].response.id,response);
  assert.equal(after.fallback.id,fallback);assert.deepEqual(graphIssues(flow),[]);
  assert.throws(()=>setPriority(flow,fallback,1),/Default response/);
  assert.throws(()=>setPriority(flow,id,0),/valid/);
});

test('complex legacy graphs stay unchanged and have no inferred priorities',()=>{
  const flow=defaultFlow();insertBlock(flow,{name:'Headers',type:'headers',config:{headers:{'X-Test':'demo'}}},'request');
  const before=structuredClone(flow);assert.equal(scenarioBranches(flow),null);arrangeBranches(flow);
  assert.deepEqual(flow,before);assert.equal(branchRows(flow),null);assert.deepEqual(branchEdges(flow),flow.edges);
});

test('changing priority is undoable and scoped node edits preserve arranged positions',()=>{
  const state=createEditor();state.endpoint={id:'e',name:'Demo',method:'GET',path:'/users',flow:defaultFlow()};
  state.quickAddCondition('header');state.quickAddCondition('query');
  const original=structuredClone(state.endpoint.flow),id=state.branchModel.branches[1].condition.id;
  state.changePriority(id,1);assert.equal(state.branchModel.branches[0].condition.id,id);
  state.undoChange();assert.deepEqual(state.endpoint.flow,original);
  const node=state.conditions[0];node.x=575;state.editNodeScenario(node);state.scenarioForm.conditions[0].label='Updated';state.saveScenario();
  assert.equal(state.endpoint.flow.nodes.find(item=>item.id===node.id).x,575);
});

test('Table creation exposes only the new independent condition and never the dispatch chain',()=>{
  const state=createEditor();state.endpoint={id:'e',name:'Demo',method:'GET',path:'/users',flow:defaultFlow()};
  state.quickAddCondition('header');state.mode='table';state.openAddScenario();
  assert.equal(state.scenarioForm.conditions.length,1);assert.equal(state.scenarioForm.conditions[0].matched,true);
  assert.ok(state.scenarioForm.connections.every(edge=>edge.branch==='true'));
  state.saveScenario();assert.equal(state.branchModel.branches.length,2);assert.equal(state.scenarioRows.length,3);
});
