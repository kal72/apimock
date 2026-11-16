import test from 'node:test';
import assert from 'node:assert/strict';
import { createEditor } from './editor.js';
import { createLogin } from './identity.js';

test('management requests attach the in-memory CSRF token and never store session credentials',async()=>{
  const state=createEditor();state.csrfToken='csrf-demo';const original=globalThis.fetch;let sent;
  globalThis.fetch=async(url,options)=>{sent={url,options};return {ok:true,status:200,json:async()=>({saved:true})};};
  try{await state.request('/api/users','POST',{name:'Demo'});assert.equal(sent.options.headers['X-CSRF-Token'],'csrf-demo');assert.equal(sent.options.headers['Content-Type'],'application/json');}finally{globalThis.fetch=original;}
});

test('new users default to the user role and password dialogs start empty',()=>{
  const state=createEditor();state.editUser();assert.equal(state.userForm.role,'user');assert.equal(state.userForm.active,true);assert.equal(state.userForm.password,'');
  state.editUser({id:'u',username:'demo',name:'Demo',role:'admin',active:true});assert.equal(state.userForm.password,'');
  state.openPassword();assert.deepEqual(state.passwordForm,{current:'',next:'',confirmation:''});
});

test('setup confirmation mismatch does not send credentials',async()=>{
  const login=createLogin();login.setup=true;login.password='unique long password';login.confirmation='different';await login.submit();assert.equal(login.error,'Passwords do not match.');assert.equal(login.busy,false);
});

test('api keys management in identity editor',async()=>{
  const state=createEditor();
  let calls=[];
  state.request=async(url,method='GET',data)=>{
    calls.push({url,method,data});
    if(url==='/api/auth/api-keys'&&method==='GET'){
      return [{id:'k1',name:'Cursor Mac',prefix:'mlk_1234...'}];
    }
    if(url==='/api/auth/api-keys'&&method==='POST'){
      return {id:'k2',name:data.name,raw_key:'mlk_secret123'};
    }
    return {};
  };

  await state.openApiKeys();
  assert.equal(state.modal,'api-keys');
  assert.equal(state.apiKeys.length,1);

  state.apiKeyName='New Key';
  await state.generateApiKey();
  assert.equal(state.newGeneratedKey.name,'New Key');
  assert.equal(state.apiKeyName,'');
});
