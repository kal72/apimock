export const environmentEditor={
 environments:[],activeEnvironmentID:'',environmentForm:{name:''},variableForm:{key:'',kind:'string',value:''},
 get activeEnvironment(){return this.environments.find(item=>item.id===this.activeEnvironmentID)||null;},
 get environmentVariables(){return this.activeEnvironment?.variables||[];},
 async loadEnvironments(){
  this.environments=await this.request('/api/environments');
  const storageKey=`mocklab-environment:${this.currentUser?.id||'user'}`;
  let saved='';try{saved=localStorage.getItem(storageKey)||'';}catch{}
  this.activeEnvironmentID=this.environments.some(item=>item.id===saved)?saved:(this.environments[0]?.id||'');
  this.$watch('activeEnvironmentID',value=>{try{localStorage.setItem(storageKey,value||'');}catch{}});
 },
 openEnvironments(){this.environmentForm={name:''};this.variableForm={key:'',kind:'string',value:''};this.modal='environments';},
 async saveEnvironment(){
  const item=this.activeEnvironment;this.busy=true;try{
   const saved=await this.request('/api/environments'+(item?`/${item.id}`:''),item?'PUT':'POST',{name:item?item.name:this.environmentForm.name});
   await this.loadEnvironments();this.activeEnvironmentID=saved.id;this.notify('Environment saved.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 async deleteEnvironment(){if(!this.activeEnvironment||!confirm(`Delete environment "${this.activeEnvironment.name}" and all its variables?`))return;try{await this.request(`/api/environments/${this.activeEnvironment.id}`,'DELETE');await this.loadEnvironments();this.notify('Environment deleted.');}catch(error){this.dialogError=error.message;}},
 async saveEnvironmentVariable(){
  const form=this.variableForm;if(!form.key.trim())return this.dialogError='Variable key is required.';
  let value=form.value;if(form.kind==='number')value=Number(value);if(form.kind==='boolean')value=form.value===true||form.value==='true';
  if(form.kind==='number'&&!Number.isFinite(value))return this.dialogError='Enter a valid number.';
  try{await this.request(`/api/environments/${this.activeEnvironmentID}/variables/${encodeURIComponent(form.key.trim())}`,'PUT',{kind:form.kind,value});await this.loadEnvironments();this.variableForm={key:'',kind:'string',value:''};this.notify('Variable saved.');}catch(error){this.dialogError=error.message;}
 },
 editEnvironmentVariable(variable){this.variableForm={key:variable.key,kind:variable.kind,value:variable.kind==='secret'?'':(variable.value ?? '')};},
 async deleteEnvironmentVariable(variable){if(!confirm(`Delete variable "${variable.key}"?`))return;try{await this.request(`/api/environments/${this.activeEnvironmentID}/variables/${encodeURIComponent(variable.key)}`,'DELETE');await this.loadEnvironments();this.notify('Variable deleted.');}catch(error){this.dialogError=error.message;}},
};
