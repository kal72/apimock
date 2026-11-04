import test from 'node:test';
import assert from 'node:assert/strict';
import { createEditor } from './editor.js';

test('workspace permissions react to selection and membership refresh', async()=>{
 const editor=createEditor();assert.equal(editor.canEditWorkspace,false);
 for(const [role,edit,manage] of [['viewer',false,false],['editor',true,false],['owner',true,true]]){
  editor.workspace={id:'workspace',role};assert.equal(editor.canEditWorkspace,edit);assert.equal(editor.canManageWorkspace,manage);
 }
 editor.workspaceID='workspace';editor.modal='members';editor.dirty=true;
 editor.request=async()=>[{id:'workspace',role:'viewer'}];
 editor.selectWorkspace=async id=>{assert.equal(id,'workspace');};
 await editor.refreshMembership();assert.equal(editor.modal,null);assert.equal(editor.canEditWorkspace,false);assert.equal(editor.dirty,false);
});
