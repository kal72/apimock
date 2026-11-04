import test from 'node:test';
import assert from 'node:assert/strict';
import { flowConditionConflicts, flowConditionWarnings, typedCondition } from './conditions.js';
import { defaultFlow, clone } from './flow.js';
import { setPriority } from './branches.js';
import { createEditor } from './editor.js';
function config(rules,match='all',type='query'){return {...typedCondition(type),match,rules};}
const rule=(field,operator='exists',value='',required=false)=>({field,operator,value,required});
function flow(...configs){
 const f=defaultFlow();f.edges=[{from:'request',to:configs.length?'c0':'response'}];
 configs.forEach((c,i)=>{f.nodes.push({id:'c'+i,type:'condition',label:'Condition '+i,config:c},{id:'r'+i,type:'response',label:'Response '+i,config:{status:200,body:'{}'}});f.edges.push({from:'c'+i,to:'r'+i,branch:'true'},{from:'c'+i,to:i+1<configs.length?'c'+(i+1):'response',branch:'false'});});return f;
}
test('duplicates normalize header casing and field order without mutating the flow',()=>{
 const f=flow(config([rule('X-Role','equals','admin'),rule('Authorization')],'all','header'),config([rule('authorization'),rule('x-role','equals','admin')],'all','header'));
 const before=clone(f);assert.equal(flowConditionConflicts(f)[0].kind,'duplicate');assert.deepEqual(f,before);
});
test('execution edges determine shadowing and a priority change fixes broad-before-specific',()=>{
 const f=flow(config([rule('status')]),config([rule('status','equals','active')]));f.nodes.reverse();
 assert.equal(flowConditionConflicts(f)[0].kind,'shadowed');setPriority(f,'c1',1);
 assert.equal(flowConditionConflicts(f)[0].kind,'overlap');assert.ok(flowConditionWarnings(f)[0].includes('#1'));
});
test('AND and OR implication supports extra fields without overclaiming coverage',()=>{
 assert.equal(flowConditionConflicts(flow(config([rule('status','equals','active')]),config([rule('status','equals','active'),rule('role','equals','admin')])))[0].kind,'shadowed');
 assert.equal(flowConditionConflicts(flow(config([rule('status','equals','active')]),config([rule('status','equals','active'),rule('role','equals','admin')],'any')))[0].kind,'overlap');
 assert.equal(flowConditionConflicts(flow(config([rule('status','equals','active'),rule('status','equals','pending')],'any'),config([rule('status','equals','active')])))[0].kind,'shadowed');
});
test('disjoint values and body formats do not warn; different sources may overlap',()=>{
 assert.deepEqual(flowConditionConflicts(flow(config([rule('status','equals','active')]),config([rule('status','equals','inactive')]))),[]);
 const a=config([rule('id')],'all','body'),b={...a,body_mode:'form'};
 assert.deepEqual(flowConditionConflicts(flow(a,b)),[]);
 assert.equal(flowConditionConflicts(flow(config([rule('id')]),a))[0].kind,'overlap');
});
test('Required respects empty JSON values and does not infer nonempty from contains',()=>{
 const required=config([rule('id','exists','',true)],'all','body');
 for(const value of ['null','[]','{}','']){const c=config([rule('id','equals',value)],'all','body');assert.notEqual(flowConditionConflicts(flow(required,c))[0]?.kind,'shadowed');}
 for(const value of ['0','false','hello']){const c=config([rule('id','equals',value)],'all','body');assert.equal(flowConditionConflicts(flow(required,c))[0].kind,'shadowed');}
 assert.notEqual(flowConditionConflicts(flow(required,config([rule('id','contains','[]')],'all','body')))[0]?.kind,'shadowed');
});
test('nested legacy graphs do not receive independent priority claims',()=>{
 const f=flow(config([rule('id')]),config([rule('id')]));f.edges.find(e=>e.from==='c0'&&e.branch==='true').to='c1';
 assert.deepEqual(flowConditionConflicts(f),[]);
});
test('popup previews unsaved conflicts while preserving the active draft',()=>{
 const editor=createEditor();editor.endpoint={name:'Users',method:'GET',path:'/users',flow:flow(config([rule('status')]),config([rule('status','equals','active')]))};
 editor.editNodeScenario(editor.endpoint.flow.nodes.find(n=>n.id==='c1'));
 assert.ok(editor.scenarioConditionWarnings.some(w=>w.includes('cannot win')));
 editor.scenarioForm.conditions[0].config.rules=[rule('other','equals','active')];
 assert.ok(editor.scenarioConditionWarnings.some(w=>w.includes('may overlap')));
 assert.equal(editor.endpoint.flow.nodes.find(n=>n.id==='c1').config.rules[0].field,'status');
});
