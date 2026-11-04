import test from 'node:test';
import assert from 'node:assert/strict';
import { createEditor } from './editor.js';
import { defaultFlow } from './flow.js';
import { responseForm, applyResponseTemplate } from './scenarios.js';
import { responseConfig } from './responses.js';
function editor(){const e=createEditor();e.endpoint={id:'e',name:'Users',method:'GET',path:'/users',flow:defaultFlow()};return e;}
test('legacy forms default to static and dynamic settings survive scoped edits',()=>{
 const e=editor();assert.equal(responseForm(e.endpoint.flow.nodes[1]).response_mode,'static');
 e.editNodeScenario(e.endpoint.flow.nodes[1]);const f=e.scenarioForm.response;f.response_mode='dynamic';f.response_seed='demo';f.body='{"id":"{{ generate.uuid }}"}';e.saveScenario();
 assert.equal(e.modal,null);assert.equal(e.endpoint.flow.nodes[1].config.response_mode,'dynamic');assert.equal(e.endpoint.flow.nodes[1].config.response_seed,'demo');
 e.editScenario(e.fallbackRow);assert.equal(e.scenarioForm.response.response_mode,'dynamic');
 applyResponseTemplate(e.endpoint.flow.nodes[1],'success');assert.equal(e.endpoint.flow.nodes[1].config.response_mode,'static');
});
test('variable insertion encodes defaults safely and preserves special object keys',()=>{
 const e=editor();e.editNodeScenario(e.endpoint.flow.nodes[1]);const f=e.scenarioForm.response;f.response_mode='dynamic';f.body='{}';
 Object.assign(e.responseVariable,{key:'__proto__',field:'name',hasDefault:true,defaultJSON:'"Guest"'});e.insertResponseVariable('response');
 const data=JSON.parse(f.body);assert.equal(data.__proto__,'{{ request.body.name | default: "Guest" }}');assert.equal({}.polluted,undefined);
 Object.assign(e.responseVariable,{key:'X-Request-ID',source:'generate.uuid',target:'headers'});e.insertResponseVariable('response');assert.equal(f.headers.find(h=>h.key==='X-Request-ID').value,'{{ generate.uuid }}');
});
test('preview submits only response configuration and leaves the endpoint untouched',async()=>{
 const e=editor();const before=JSON.stringify(e.endpoint);e.editNodeScenario(e.endpoint.flow.nodes[1]);e.scenarioForm.response.response_mode='dynamic';
 e.request=async(url,method,payload)=>{assert.equal(url,'/api/test');assert.equal(method,'POST');assert.equal(payload.flow.nodes[1].config.response_mode,'dynamic');assert.equal(payload.flow.nodes[1].config.previewResult,undefined);return {body:'{"ok":true}',headers:{}};};
 await e.previewResponse('response');assert.equal(JSON.stringify(e.endpoint),before);assert.equal(e.scenarioForm.response.previewResult.body,'{"ok":true}');
 e.request=async()=>{throw Error('missing variable request.body.name');};await e.previewResponse('response');assert.match(e.scenarioForm.response.previewError,/missing variable/);assert.equal(e.scenarioForm.response.previewBusy,false);
 assert.throws(()=>responseConfig({response_mode:'script',headers:[]}),/Static JSON/);
});
