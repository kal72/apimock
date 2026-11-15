export const environmentEditor={
 environments:[],activeEnvironmentID:'',environmentForm:{name:''},variableForm:{key:'',kind:'string',value:''},_envWatchInstalled:false,
 get activeEnvironment(){
  return this.environments.find(item=>item.id===this.activeEnvironmentID) || (this.environments.length ? this.environments[0] : null);
 },
 get environmentVariables(){return this.activeEnvironment?.variables||[];},
 get isEditingVariable(){return Boolean(this.variableForm.key && this.environmentVariables.some(v=>v.key===this.variableForm.key));},
 resetVariableForm(){this.variableForm={key:'',kind:'string',value:''};this.dialogError='';},
 async loadEnvironments(){
  this.environments=await this.request('/api/environments');
  const storageKey=`mocklab-environment:${this.currentUser?.id||'user'}`;
  let saved='';try{saved=localStorage.getItem(storageKey)||'';}catch{}
  const current=this.activeEnvironmentID;
  if(current && this.environments.some(item=>item.id===current)){
   this.activeEnvironmentID=current;
  }else{
   this.activeEnvironmentID=this.environments.some(item=>item.id===saved)?saved:(this.environments[0]?.id||'');
  }
  if(!this._envWatchInstalled && typeof this.$watch==='function'){
   this._envWatchInstalled=true;
   this.$watch('activeEnvironmentID',value=>{try{localStorage.setItem(storageKey,value||'');}catch{}});
  }
 },
 openEnvironments(asModal=true){
  this.environmentForm={name:''};
  this.resetVariableForm();
  this.dialogError='';
  if((!this.activeEnvironmentID || !this.environments.some(e=>e.id===this.activeEnvironmentID)) && this.environments.length){
   this.activeEnvironmentID=this.environments[0].id;
  }
  this.modal=asModal?'environments':null;
  if(!asModal)this.navPage='environments';
 },
 async createEnvironment(){
  const name=this.environmentForm.name.trim();
  if(!name)return this.dialogError='Environment name is required.';
  this.busy=true;try{
   const saved=await this.request('/api/environments','POST',{name});
   this.activeEnvironmentID=saved.id;
   await this.loadEnvironments();
   this.activeEnvironmentID=saved.id;
   this.environmentForm={name:''};
   this.dialogError='';
   this.notify('Environment created.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 async saveEnvironment(env){
  const item=env||this.activeEnvironment;
  if(!item)return this.createEnvironment();
  const name=(item.name||'').trim();
  if(!name)return this.dialogError='Environment name is required.';
  this.busy=true;try{
   const saved=await this.request(`/api/environments/${item.id}`,'PUT',{name});
   this.activeEnvironmentID=saved.id;
   await this.loadEnvironments();
   this.activeEnvironmentID=saved.id;
   this.dialogError='';
   this.notify('Environment saved.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 async deleteEnvironment(env){
  const item=env||this.activeEnvironment;
  if(!item||!confirm(`Delete environment "${item.name}" and all its variables?`))return;
  this.busy=true;try{
   await this.request(`/api/environments/${item.id}`,'DELETE');
   if(this.activeEnvironmentID===item.id){
    this.activeEnvironmentID='';
   }
   await this.loadEnvironments();
   this.resetVariableForm();
   this.dialogError='';
   this.notify('Environment deleted.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 async saveEnvironmentVariable(){
  const form=this.variableForm;
  const key=(form.key||'').trim();
  if(!key)return this.dialogError='Variable key is required.';
  const activeID=this.activeEnvironmentID || this.activeEnvironment?.id;
  if(!activeID)return this.dialogError='Please select or create an environment first.';
  let value=form.value;
  if(form.kind==='number'){
   value=Number(value);
   if(!Number.isFinite(value))return this.dialogError='Enter a valid number.';
  }
  if(form.kind==='boolean'){
   value=form.value===true||form.value==='true';
  }
  if(form.kind==='string'||form.kind==='secret'){
   value=String(value ?? '');
  }
  this.busy=true;
  this.dialogError='';
  try{
   await this.request(`/api/environments/${activeID}/variables/${encodeURIComponent(key)}`,'PUT',{kind:form.kind,value});
   this.activeEnvironmentID=activeID;
   await this.loadEnvironments();
   this.activeEnvironmentID=activeID;
   this.resetVariableForm();
   this.dialogError='';
   this.notify('Variable saved.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 editEnvironmentVariable(variable){
  this.variableForm={
   key:variable.key,
   kind:variable.kind,
   value:variable.kind==='secret'?'':(variable.kind==='boolean'?String(variable.value):(variable.value ?? ''))
  };
  this.dialogError='';
 },
 async copyVariableExpression(key){
  await this.copyText('{{ environment.' + key + ' }}', 'Copied {{ environment.' + key + ' }}');
 },
 async copyVariableValue(variable){
  const val = typeof variable.value === 'object' ? JSON.stringify(variable.value) : String(variable.value ?? '');
  await this.copyText(val, 'Value copied to clipboard.');
 },
 async deleteEnvironmentVariable(variable){
  const activeID=this.activeEnvironmentID || this.activeEnvironment?.id;
  if(!activeID)return;
  if(!confirm(`Delete variable "${variable.key}"?`))return;
  this.busy=true;try{
   await this.request(`/api/environments/${activeID}/variables/${encodeURIComponent(variable.key)}`,'DELETE');
   this.activeEnvironmentID=activeID;
   await this.loadEnvironments();
   this.activeEnvironmentID=activeID;
   if(this.variableForm.key===variable.key)this.resetVariableForm();
   this.dialogError='';
   this.notify('Variable deleted.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
};
