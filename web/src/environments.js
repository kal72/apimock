export const environmentEditor={
 environments:[],activeEnvironmentID:'',environmentForm:{name:''},variableForm:{key:'',kind:'string',value:''},
 get activeEnvironment(){return this.environments.find(item=>item.id===this.activeEnvironmentID)||null;},
 get environmentVariables(){return this.activeEnvironment?.variables||[];},
 get isEditingVariable(){return Boolean(this.variableForm.key && this.environmentVariables.some(v=>v.key===this.variableForm.key));},
 resetVariableForm(){this.variableForm={key:'',kind:'string',value:''};},
 async loadEnvironments(){
  this.environments=await this.request('/api/environments');
  const storageKey=`mocklab-environment:${this.currentUser?.id||'user'}`;
  let saved='';try{saved=localStorage.getItem(storageKey)||'';}catch{}
  this.activeEnvironmentID=this.environments.some(item=>item.id===saved)?saved:(this.environments[0]?.id||'');
  this.$watch('activeEnvironmentID',value=>{try{localStorage.setItem(storageKey,value||'');}catch{}});
 },
 openEnvironments(){this.environmentForm={name:''};this.resetVariableForm();this.dialogError='';this.modal='environments';},
 async createEnvironment(){
  const name=this.environmentForm.name.trim();
  if(!name)return this.dialogError='Environment name is required.';
  this.busy=true;try{
   const saved=await this.request('/api/environments','POST',{name});
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
   await this.loadEnvironments();
   if(this.activeEnvironmentID===item.id){
    this.activeEnvironmentID=this.environments[0]?.id||'';
   }
   this.resetVariableForm();
   this.dialogError='';
   this.notify('Environment deleted.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 async saveEnvironmentVariable(){
  const form=this.variableForm;
  const key=form.key.trim();
  if(!key)return this.dialogError='Variable key is required.';
  let value=form.value;
  if(form.kind==='number')value=Number(value);
  if(form.kind==='boolean')value=form.value===true||form.value==='true';
  if(form.kind==='number'&&!Number.isFinite(value))return this.dialogError='Enter a valid number.';
  this.busy=true;try{
   await this.request(`/api/environments/${this.activeEnvironmentID}/variables/${encodeURIComponent(key)}`,'PUT',{kind:form.kind,value});
   await this.loadEnvironments();
   this.resetVariableForm();
   this.dialogError='';
   this.notify('Variable saved.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 editEnvironmentVariable(variable){
  this.variableForm={key:variable.key,kind:variable.kind,value:variable.kind==='secret'?'':(variable.value ?? '')};
  this.dialogError='';
 },
 async deleteEnvironmentVariable(variable){
  if(!confirm(`Delete variable "${variable.key}"?`))return;
  this.busy=true;try{
   await this.request(`/api/environments/${this.activeEnvironmentID}/variables/${encodeURIComponent(variable.key)}`,'DELETE');
   await this.loadEnvironments();
   if(this.variableForm.key===variable.key)this.resetVariableForm();
   this.dialogError='';
   this.notify('Variable deleted.');
  }catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
};
