import { defaultFlow, headersFromRows } from './flow.js';

export function responseConfig(form){
 if(!['static','dynamic'].includes(form.response_mode || 'static'))throw Error('Choose Static JSON or Dynamic JSON.');
 if(form.body)JSON.parse(form.body);
 return {status:form.status,body:form.body,headers:headersFromRows(form.headers),delay_ms:form.delay_ms,response_mode:form.response_mode || 'static',response_seed:form.response_seed || ''};
}
export const responseEditor={
 variablePicker:null,
 responseVariable:{source:'request.body',field:'name',target:'body',key:'value',hasDefault:false,defaultJSON:'"Guest"',arguments:'[1, 100]'},
 openResponseVariables(section){this.variablePicker=this.variablePicker===section?null:section;},
 insertResponseVariable(section){
  try{
   const v=this.responseVariable,form=this.scenarioForm[section];let source=v.source;
   if(source==='environment'){
    if(!/^[A-Za-z_][A-Za-z0-9_]{0,63}$/.test(v.field))throw Error('Select an environment variable.');source+='.'+v.field;
   } else if(['request.path','request.query','request.headers','request.body','request.form'].includes(source)){
    if(!/^[A-Za-z0-9_.$-]+$/.test(v.field))throw Error('Enter a field path using letters, numbers, dots, underscores or hyphens.');source+='.'+v.field;
   }
   if(['generate.integer','generate.choice'].includes(source))source+=': '+JSON.stringify(JSON.parse(v.arguments));
   if(v.hasDefault && v.source.startsWith('request.'))source+=' | default: '+JSON.stringify(JSON.parse(v.defaultJSON));
   const expression='{{ '+source+' }}';
   if(!v.key.trim())throw Error('Enter the response field or header name.');
   if(v.target==='headers'){
    const existing=form.headers.find(h=>h.key.toLowerCase()===v.key.trim().toLowerCase());
    if(existing)existing.value=expression;else form.headers.push({key:v.key.trim(),value:expression});
   }else{
    const body=JSON.parse(form.body || '{}');if(!body || Array.isArray(body)||typeof body!=='object')throw Error('Insert variable requires an object body. Edit array/scalar templates directly in the JSON editor.');
    if(Object.hasOwn(body,v.key) && !window.confirm(`Replace response field "${v.key}"?`))return;
    Object.defineProperty(body,v.key,{value:expression,enumerable:true,configurable:true,writable:true});form.body=JSON.stringify(body,null,2);
   }
   this.dialogError='';this.variablePicker=null;
  }catch(error){this.dialogError=error.message;}
 },
 async previewResponse(section){
  const form=this.scenarioForm[section];form.previewBusy=true;form.previewError='';form.previewResult=null;
  try{
   const flow=defaultFlow();flow.nodes.find(n=>n.type==='response').config=responseConfig(form);
   form.previewResult=await this.request('/api/test','POST',{...JSON.parse(this.testInput),environment_id:this.activeEnvironmentID,flow,method:this.endpoint.method,path:this.endpoint.path});
  }catch(error){form.previewError=error.message;}finally{form.previewBusy=false;}
 }
};
