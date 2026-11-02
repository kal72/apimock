import { scenarioBranches, arrangeBranches, branchEdges, branchRows, setPriority } from './branches.js';
import { formatValue, pathPreview } from './preview.js';
import { conditionKinds, pathParameters, typedCondition, normalizeCondition, conditionDiagnostics, conditionSummary, conditionKindLabel, flowConditionWarnings } from './conditions.js';
import { conditionTemplates, responseTemplates, projectScenarios, defaultScenario, addConditionScenario, applyResponseTemplate, responseForm, scenarioForm, removeScenario } from './scenarios.js';
import { clone, defaultExample, defaultFlow, outgoingBranches, setConnection, insertBlock, graphIssues, nodeRows, connectionPoints, parseObject, headersFromRows, preferredResponse, directResponseFlow, removeConfiguration } from './flow.js';

export const createEditor = () => ({
  workspaces:[], endpoints:[], workspaceID:'', workspace:null, endpoint:null, publicOrigin:'',
  methods:['GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS'],
  preview:null, formatValue,
  conditionKinds, conditionTemplates, responseTemplates, scenarioForm:null, scenarioDraft:null, scenarioScope:null, scenarioTab:'response', scenarioTemplate:'header', scenarioIsNew:false,
  highlightedNodeID:null, mode:'table', search:'', modal:null, dialogError:'', notice:'', busy:false, dirty:false,
  workspaceForm:{}, endpointForm:{}, nodeForm:null, headerRows:[], connecting:null, drag:null, zoom:1, minZoom:0.5, maxZoom:2, zoomStep:0.25,
  nodeSection:'all', directResponseID:'', pendingSummary:'', undoSnapshot:null, requestRoute:{}, paramsText:'{}', queryText:'{}', requestBodyText:'{}',
  insertAfter:'', insertBranch:'', pendingFlow:null, connectionForm:{from:'',to:'',branch:''}, originalConnection:null,
  testOpen:true, testing:false, trace:[], testResult:null,
  testInput:JSON.stringify({params:{id:'123'},query:{},headers:{},body:{}},null,2),
  templates:[
    {key:'headers',type:'headers',name:'Default JSON headers',description:'Content-Type and Cache-Control.',config:{headers:{'Content-Type':'application/json','Cache-Control':'no-cache'}}},
    {key:'condition',type:'condition',name:'Condition',description:'Branch the flow using request values.',config:{field:'params.id',operator:'equals',value:'123'}},
    {key:'success',type:'response',name:'Success response',description:'HTTP 200 with a JSON body.',config:{status:200,headers:{'Content-Type':'application/json'},body:'{\n  "success": true\n}'}},
    {key:'not-found',type:'response',name:'Not found',description:'HTTP 404 for missing data.',config:{status:404,headers:{'Content-Type':'application/json'},body:'{\n  "error": "Not found"\n}'}},
    {key:'unauthorized',type:'response',name:'Unauthorized',description:'HTTP 401 for unauthorized access.',config:{status:401,headers:{'Content-Type':'application/json'},body:'{\n  "error": "Unauthorized"\n}'}},
    {key:'delay',type:'delay',name:'Response delay',description:'Simulate network latency.',config:{delay_ms:500}}
  ],
  get conditionWarnings(){return this.endpoint ? flowConditionWarnings(this.endpoint.flow) : [];},
  get scenarioConditionWarnings(){return (this.scenarioForm?.conditions || []).flatMap(condition => conditionDiagnostics(condition.config,this.scenarioForm.route.path).warnings);},
  get availablePathFields(){return pathParameters(this.scenarioForm?.route.path || this.endpoint?.path || '');},
  get branchModel(){return this.endpoint?scenarioBranches(this.endpoint.flow):null;},
  get canvasEdges(){return this.endpoint?branchEdges(this.endpoint.flow):[];},
  get scenarioProjection(){const rows=this.endpoint?branchRows(this.endpoint.flow):null;return rows?{rows,truncated:false}:this.endpoint?projectScenarios(this.endpoint.flow):{rows:[],truncated:false};},
  priorityFor(id){return this.branchModel?.branches.find(item=>item.condition.id===id)?.priority || null;},
  changePriority(id,value){if(this.priorityFor(id)===Number(value))return;try{const flow=clone(this.endpoint.flow);setPriority(flow,id,Number(value));this.rememberChange();this.endpoint.flow=flow;this.dirty=true;this.trace=[];this.testResult=null;this.connecting=null;}catch(error){this.notify(error.message);}},
  get scenarioRows(){return this.scenarioProjection.rows;},
  get fallbackRow(){return this.endpoint ? defaultScenario(this.endpoint.flow) : null;},
  get filteredEndpoints(){return this.endpoints.filter(e=>(e.path+' '+e.name+' '+e.method).toLowerCase().includes(this.search.toLowerCase()));},
  get canvasWidth(){return Math.max(1100,...(this.endpoint?.flow.nodes || []).map(node=>node.x+360));},
  get canvasHeight(){return Math.max(640,...(this.endpoint?.flow.nodes || []).map(node=>node.y+240));},
  get orderedNodes(){return this.endpoint ? nodeRows(this.endpoint.flow) : [];},
  get requestNode(){return this.endpoint?.flow.nodes.find(node => node.type === 'request');},
  get conditions(){return this.orderedNodes.filter(node => node.type === 'condition');},
  get responses(){return this.orderedNodes.filter(node => node.type === 'response');},
  get responseMode(){return this.conditions.length ? 'conditional' : 'direct';},
  get defaultResponse(){return this.endpoint ? preferredResponse(this.endpoint.flow) : null;},
  get readyToSave(){return !!this.endpoint && this.dirty && !this.busy && !this.flowIssues.length;},
  get headerNodes(){return this.orderedNodes.filter(node => node.type === 'headers');},
  get behaviors(){return this.orderedNodes.filter(node => node.type === 'delay');},
  get flowIssues(){return this.endpoint ? graphIssues(this.endpoint.flow) : [];},
  get insertionSources(){return this.orderedNodes.filter(node => node.type !== 'response');},
  get baseURL(){return `${this.publicOrigin || location.origin}/public/${this.workspace?.slug || ''}`;},
  get modalTitle(){return {view:'View request path',workspace:this.workspaceForm.id?'Edit workspace':'New workspace',endpoint:'New endpoint',templates:'Add configuration',node:this.nodeForm?.type === 'request' ? 'Edit incoming request' : 'Edit configuration',connection:'Edit connection',direct:'Use an unconditional response',palette:'Add condition node',scenario:this.scenarioScope ? 'Edit node · '+this.scenarioScope.label : this.scenarioIsNew ? 'Add condition' : 'Edit path · ' + (this.scenarioForm?.label || '')}[this.modal] || '';},
  async request(url,method='GET',data){
    const response=await fetch(url,{method,headers:{'Content-Type':'application/json'},body:data===undefined?undefined:JSON.stringify(data)});
    const result=await response.json();if(!response.ok)throw Error(result.error || 'Request failed');return result;
  },
  notify(message){this.notice=message;},
  async boot(){
    this.$watch('modal',()=>{this.dialogError='';});
    window.addEventListener('beforeunload',event=>{if(this.dirty){event.preventDefault();event.returnValue='';}});
    try{const settings=await this.request('/api/config');this.publicOrigin=settings.public_base_url;this.workspaces=await this.request('/api/workspaces');if(this.workspaces.length)await this.selectWorkspace(this.workspaces[0].id);}catch(error){this.notify(error.message);}
  },
  mayDiscard(){return !this.dirty || window.confirm('You have unsaved changes. Continue and discard them?');},
  async selectWorkspace(id){
    if(!this.mayDiscard()){this.workspaceID=this.workspace?.id || '';return;}
    try{const endpoints=id?await this.request(`/api/workspaces/${id}/endpoints`):[];this.workspaceID=id;this.workspace=this.workspaces.find(w=>w.id===id)||null;this.endpoints=endpoints;this.endpoint=null;this.dirty=false;this.trace=[];this.testResult=null;if(endpoints.length)this.selectEndpoint(endpoints[0]);}catch(error){this.workspaceID=this.workspace?.id || '';this.notify(error.message);}
  },
  selectEndpoint(endpoint){if(!this.mayDiscard())return;this.endpoint=clone(endpoint);if(this.endpoint.flow.layout!=='scenarios')arrangeBranches(this.endpoint.flow);this.dirty=false;this.trace=[];this.testResult=null;this.connecting=null;this.pendingFlow=null;this.pendingSummary='';this.undoSnapshot=null;this.highlightedNodeID=null;this.insertAfter=this.requestNode?.id || '';this.testInput=JSON.stringify(this.requestNode?.config.example || defaultExample(),null,2);},
  openWorkspace(workspace){this.workspaceForm=workspace?clone(workspace):{name:'',slug:''};this.modal='workspace';},
  async saveWorkspace(){this.busy=true;try{const w=this.workspaceForm;const saved=await this.request('/api/workspaces'+(w.id?`/${w.id}`:''),w.id?'PUT':'POST',w);this.workspaces=await this.request('/api/workspaces');this.modal=null;if(this.workspace?.id===saved.id){this.workspace=saved;}else{await this.selectWorkspace(saved.id);}this.notify('Workspace saved.');}catch(error){this.dialogError=error.message;}finally{this.busy=false;}},
  async removeWorkspace(){if(!this.workspace||!window.confirm(`Delete workspace "${this.workspace.name}" and all its endpoints?`))return;try{await this.request(`/api/workspaces/${this.workspace.id}`,'DELETE');this.workspaces=this.workspaces.filter(w=>w.id!==this.workspace.id);this.dirty=false;await this.selectWorkspace(this.workspaces[0]?.id || '');}catch(error){this.notify(error.message);}},
  newEndpoint(){if(!this.mayDiscard())return;this.endpointForm={name:'',method:'GET',path:'/'};this.modal='endpoint';},
  async createEndpoint(){this.busy=true;try{const saved=await this.request(`/api/workspaces/${this.workspace.id}/endpoints`,'POST',{...this.endpointForm,flow:defaultFlow()});this.endpoints.push(saved);this.dirty=false;this.selectEndpoint(saved);this.modal=null;this.mode='table';this.notify('Endpoint active: Incoming request → Success response 200. Conditions are optional.');}catch(error){this.dialogError=error.message;}finally{this.busy=false;}},
  async saveEndpoint(){
    if(!this.endpoint || this.busy)return;
    if(this.flowIssues.length){this.notify(this.flowIssues.join(' '));return;}
    const workspaceID=this.workspace.id;
    const payload=clone(this.endpoint);
    const snapshot=JSON.stringify(payload);
    this.busy=true;
    try{
      const saved=await this.request(`/api/workspaces/${workspaceID}/endpoints/${payload.id}`,'PUT',payload);
      if(this.workspace?.id === workspaceID)this.endpoints=this.endpoints.map(item => item.id === saved.id ? saved : item);
      if(this.endpoint?.id === payload.id && this.workspace?.id === workspaceID){
        if(JSON.stringify(this.endpoint) === snapshot){this.endpoint=clone(saved);this.dirty=false;this.undoSnapshot=null;this.notify('Saved. The mock API is using this configuration.');}
        else {this.dirty=true;this.notify('The version submitted when you clicked Save is active. Newer changes still need saving.');}
      }
    }catch(error){this.notify(error.message);}finally{this.busy=false;}
  },
  rememberChange(){this.undoSnapshot=clone(this.endpoint);},
  undoChange(){if(!this.undoSnapshot)return;const previous=clone(this.undoSnapshot);this.undoSnapshot=null;this.endpoint=previous;this.pendingFlow=null;this.dirty=JSON.stringify(previous)!==JSON.stringify(this.endpoints.find(item => item.id === previous.id));this.trace=[];this.testResult=null;this.testInput=JSON.stringify(this.requestNode.config.example || defaultExample(),null,2);this.notify('Last change undone.');},
  discardChanges(){const saved=this.endpoints.find(item => item.id === this.endpoint?.id);if(!saved || !this.mayDiscard())return;this.dirty=false;this.selectEndpoint(saved);this.notify('Restored the saved configuration.');},
  chooseResponseMode(mode){if(mode === 'direct')this.openDirectResponse();else if(!this.conditions.length)this.addConfiguration('condition');else this.notify('Edit conditions and response destinations in Response rules.');},
  openDirectResponse(){
    this.directResponseID=this.defaultResponse?.id || '';
    if(this.conditions.length){this.modal='direct';this.dialogError='';return;}
    this.prepareDirectResponse();
  },
  prepareDirectResponse(){
    const candidate=directResponseFlow(this.endpoint.flow,this.directResponseID);
    const response=candidate.nodes.find(node => node.type === 'response');
    this.editNode(response);
    this.pendingFlow=candidate;
    const removed=this.endpoint.flow.nodes.length-candidate.nodes.length;
    this.pendingSummary=removed>0 ? `After applying: one unconditional path. ${removed} unused settings/branches are removed from the draft. You can undo this change.` : 'All incoming requests receive this response. No conditions are needed.';
  },
  async removeEndpoint(){if(!window.confirm('Delete this endpoint?'))return;try{await this.request(`/api/workspaces/${this.workspace.id}/endpoints/${this.endpoint.id}`,'DELETE');this.endpoints=this.endpoints.filter(e=>e.id!==this.endpoint.id);this.endpoint=null;this.dirty=false;this.trace=[];this.testResult=null;if(this.endpoints.length)this.selectEndpoint(this.endpoints[0]);}catch(error){this.notify(error.message);}},
  openAddScenario(){
    this.scenarioScope=null;
    try{
      if(this.mode === 'node'){this.modal='palette';this.dialogError='';return;}
      this.scenarioIsNew=true;this.scenarioTemplate='header';this.scenarioTab='conditions';
      const candidate=clone(this.endpoint.flow);
      const row=addConditionScenario(candidate,'header',{path:this.endpoint.path,example:this.requestNode?.config.example});
      this.scenarioDraft=candidate;this.scenarioForm=scenarioForm(this.endpoint,candidate,this.branchRowFor(candidate,row));if(this.branchModel){arrangeBranches(candidate);this.scenarioForm.connections=this.scenarioForm.connections.filter(edge=>candidate.nodes.find(node=>node.id===edge.from)?.type==='condition');}this.modal='scenario';this.dialogError='';
    }catch(error){this.notify(error.message);}
  },
  quickAddCondition(kind){
    try{
      const candidate=clone(this.endpoint.flow);
      const row=addConditionScenario(candidate,kind,{path:this.endpoint.path,example:this.requestNode?.config.example,source:this.branchModel?null:this.connecting});
      const issues=graphIssues(candidate);if(issues.length)throw Error(issues.join(' '));
      if(this.branchModel)arrangeBranches(candidate);this.rememberChange();this.endpoint.flow=candidate;this.dirty=true;this.trace=[];this.testResult=null;this.modal=null;this.connecting=null;
      const added=candidate.nodes.find(node=>node.type==='condition' && !this.undoSnapshot.flow.nodes.some(old=>old.id===node.id));this.highlightedNodeID=added.id;
      this.$nextTick?.(()=>{this.$root?.querySelector('.canvas-scroll')?.scrollTo({left:Math.max(0,(added.x-80)*this.zoom),top:Math.max(0,(added.y-80)*this.zoom),behavior:'smooth'});});
      const warnings=flowConditionWarnings(candidate);this.notify(`${conditionKindLabel(added.config)} added and connected as a draft. Use Edit to add fields.${warnings.length ? ' '+warnings.at(-1) : ''}`);
    }catch(error){this.dialogError=error.message;}
  },
  canAddCondition(kind){return kind !== 'path' || pathParameters(this.endpoint?.path || '').length>0;},
  addConditionField(condition){const field=condition.config.condition_type==='path' ? this.availablePathFields[0] || '' : condition.config.condition_type==='body' && condition.config.raw_format==='text' && condition.config.body_mode==='raw' ? '$raw' : '';condition.config.rules.push({field,operator:'exists',value:'',required:true});},
  changeConditionKind(condition,kind){
    try{condition.config=typedCondition(kind,this.scenarioForm.route.path,{});this.dialogError='';}catch(error){this.dialogError=error.message;}
  },
  changeBodyMode(condition){
    if(condition.config.body_mode==='raw' && condition.config.raw_format==='text')condition.config.rules.forEach(rule=>rule.field='$raw');
    else condition.config.rules.forEach(rule=>{if(rule.field==='$raw')rule.field='id';});
  },
  chooseScenarioTemplate(key){
    try{
      const candidate=clone(this.endpoint.flow);
      const row=addConditionScenario(candidate,key,{path:this.endpoint.path,example:this.requestNode?.config.example});
      this.scenarioDraft=candidate;this.scenarioForm=scenarioForm(this.endpoint,candidate,this.branchRowFor(candidate,row));if(this.branchModel)this.scenarioForm.connections=this.scenarioForm.connections.filter(edge=>candidate.nodes.find(node=>node.id===edge.from)?.type==='condition');this.scenarioTemplate=key;this.dialogError='';
    }catch(error){this.dialogError=error.message;}
  },
  branchRowFor(flow,row){return branchRows(flow)?.find(item=>item.conditions.some(condition=>row.conditions?.some(old=>old.id===condition.id && old.matched))) || row;},
  viewScenario(row){this.preview=pathPreview(this.endpoint,row,this.baseURL);this.modal='view';this.dialogError='';},
  editScenario(row){
    if(!row)return;
    const candidate=clone(this.endpoint.flow);
    this.scenarioScope=null;this.scenarioIsNew=false;this.scenarioDraft=candidate;this.scenarioForm=scenarioForm(this.endpoint,candidate,row);this.scenarioTab='response';if(this.branchModel)this.scenarioForm.connections=this.scenarioForm.connections.filter(edge=>candidate.nodes.find(node=>node.id===edge.from)?.type==='condition');this.modal='scenario';this.dialogError='';
  },
  editNodeScenario(node){
    const candidate=clone(this.endpoint.flow);
    const row={conditions:node.type==='condition'?[{id:node.id,matched:true}]:[],responseID:node.type==='response'?node.id:this.responses[0]?.id,pathIDs:[node.id],headerIDs:node.type==='headers'?[node.id]:[],delayIDs:node.type==='delay'?[node.id]:[],isDefault:false,label:node.label};
    this.scenarioIsNew=false;this.scenarioScope={id:node.id,type:node.type,label:node.label};
    this.scenarioDraft=candidate;this.scenarioForm=scenarioForm(this.endpoint,candidate,row);
    this.scenarioForm.fallback=null;this.scenarioForm.connections=[];
    if(node.type!=='response')this.scenarioForm.response=null;
    this.scenarioTab={request:'request',condition:'conditions',response:'response',headers:'advanced',delay:'advanced'}[node.type];
    this.modal='scenario';this.dialogError='';
  },
  selectScenarioResponseTemplate(section,key){
    try{
      const form=this.scenarioForm[section];
      if(!form || key === 'keep')return;
      const node={id:form.id,label:form.label,config:{}};applyResponseTemplate(node,key);this.scenarioForm[section]={...responseForm(node),template:key};
    }catch(error){this.dialogError=error.message;}
  },
  saveScenario(){
    let validationSection='request';
    try{
      const form=this.scenarioForm, candidate=clone(this.scenarioDraft);
      if(!this.scenarioScope || this.scenarioScope.type==='request'){
      if(!form.route.name.trim() || !form.route.path.startsWith('/') || !this.methods.includes(form.route.method))throw Error('Endpoint name, method, and path must be valid.');
      const params=parseObject(form.params,'Path parameters'), query=parseObject(form.query,'Query parameters');
      let body={},rawBody='',formFields={};
      if(form.bodyMode==='json'){const parsed=JSON.parse(form.body);if(parsed && !Array.isArray(parsed) && typeof parsed==='object')body=parsed;else rawBody=form.body;}else if(form.bodyMode==='form'){formFields=parseObject(form.form,'Form body');if(Object.values(formFields).some(value=>!Array.isArray(value)||value.some(item=>typeof item!=='string')))throw Error('Form fields must be arrays of strings.');}else if(form.bodyMode==='text')rawBody=form.rawBody;else throw Error('Choose a body example format.');
      if(Object.values(params).some(value=>typeof value!=='string'))throw Error('Path parameters must be strings.');
      if(Object.values(query).some(value=>!Array.isArray(value)||value.some(item=>typeof item!=='string')))throw Error('Query parameters must be arrays of strings.');
      candidate.nodes.find(node=>node.id===form.requestID).config.example={params,query,body,headers:headersFromRows(form.headers),body_mode:form.bodyMode,raw_body:rawBody,form:formFields};
      }
      validationSection='conditions';
      for(const condition of form.conditions){
        const diagnostics=conditionDiagnostics(condition.config,form.route.path);if(!condition.label.trim() || diagnostics.errors.length)throw Error(diagnostics.errors.join(' ') || 'Condition name is required.');
        const node=candidate.nodes.find(item=>item.id===condition.id);node.label=condition.label;node.config=clone(condition.config);
      }
      for(const response of [form.response,form.fallback].filter(Boolean)){
        validationSection=response===form.response?'response':'fallback';
        if(!response.label.trim() || !Number.isInteger(response.status) || response.status<200 || response.status>599)throw Error('Response label and HTTP status (200–599) must be valid.');
        if(response.body){try{JSON.parse(response.body);}catch{throw Error('Response body must be valid JSON.');}}
        if(!Number.isInteger(response.delay_ms)||response.delay_ms<0||response.delay_ms>10000)throw Error('Response delay must be 0–10000 ms.');
        const node=candidate.nodes.find(item=>item.id===response.id);
        node.label=response.label;node.config={...node.config,status:response.status,body:response.body,headers:headersFromRows(response.headers),delay_ms:response.delay_ms};
      }
      validationSection='advanced';
      for(const block of form.legacyHeaders)candidate.nodes.find(node=>node.id===block.id).config.headers=headersFromRows(block.headers);
      for(const block of form.legacyDelays){if(!Number.isInteger(block.delay_ms)||block.delay_ms<0||block.delay_ms>10000)throw Error('Delay must be 0–10000 ms.');candidate.nodes.find(node=>node.id===block.id).config.delay_ms=block.delay_ms;}
      for(const edge of form.connections)setConnection(candidate,edge.from,edge.branch || '',edge.to);
      if(this.branchModel){const seen=new Set();const visit=id=>{if(seen.has(id))return;seen.add(id);candidate.edges.filter(edge=>edge.from===id).forEach(edge=>visit(edge.to));};visit(this.requestNode.id);candidate.nodes=candidate.nodes.filter(node=>node.type!=='response'||seen.has(node.id));}
      const issues=graphIssues(candidate);if(issues.length)throw Error(issues.join(' '));
      validationSection='conditions';for(const node of candidate.nodes.filter(item=>item.type==='condition' && item.config.condition_type)){const errors=conditionDiagnostics(node.config,form.route.path).errors;if(errors.length)throw Error(node.label+': '+errors.join(' '));}
      if(this.scenarioIsNew && this.branchModel)arrangeBranches(candidate);
      this.rememberChange();this.endpoint={...this.endpoint,...(!this.scenarioScope || this.scenarioScope.type==='request'?form.route:{}),flow:candidate};this.testInput=JSON.stringify(candidate.nodes.find(node=>node.type==='request').config.example,null,2);this.dirty=true;this.trace=[];this.testResult=null;this.modal=null;this.scenarioDraft=null;
    }catch(error){this.scenarioTab=validationSection;this.dialogError=error.message;}
  },
  removeScenarioCondition(id){
    try{
      if(!window.confirm('Delete this condition and its response? The default response and other conditions are preserved.'))return;
      const candidate=clone(this.endpoint.flow);removeScenario(candidate,id);this.rememberChange();this.endpoint.flow=candidate;this.dirty=true;this.trace=[];this.modal=null;
    }catch(error){this.dialogError=error.message;}
  },
  scenarioConditionDetails(row){return row.conditions.map(condition=>{const node=this.endpoint.flow.nodes.find(item=>item.id===condition.id);return {id:condition.id,name:node.label,matched:condition.matched,summary:conditionSummary(node.config)};});},
  scenarioPath(row){return row.pathIDs.map(id=>this.endpoint.flow.nodes.find(node=>node.id===id)?.label || id).join(' → ');},
  scenarioConditionSummary(row){return row.isDefault ? (row.conditions.length ? 'When no previous conditions match' : 'All incoming requests · no conditions') : row.conditions.map(item=>{const node=this.endpoint.flow.nodes.find(node=>node.id===item.id);return `${item.matched?'':'No match: '}${conditionSummary(node.config)}`;}).join(' · ');},
  scenarioHeaderCount(row){const nodes=[...row.headerIDs,row.responseID].map(id=>this.endpoint.flow.nodes.find(node=>node.id===id));return new Set(nodes.flatMap(node=>Object.keys(node.config.headers || {}).map(key=>key.toLowerCase()))).size;},
  openTemplates(){this.insertAfter=this.requestNode.id;this.insertBranch='';this.modal='templates';},
  addConfiguration(type){this.insertAfter=this.requestNode.id;this.insertBranch='';this.addNode(this.templates.find(item => item.type === type));},
  addNode(template){
    try{
      const candidate=clone(this.endpoint.flow);
      const node=insertBlock(candidate,template,this.insertAfter || this.requestNode.id,this.insertBranch);
      this.editNode(node);
      this.pendingFlow=candidate;
    }catch(error){if(this.modal)this.dialogError=error.message;else this.notify(error.message);}
  },
  editNode(node,section='all'){
    this.pendingFlow=null;this.pendingSummary='';this.nodeForm=clone(node);this.nodeSection=section;
    if(node.type === 'delay')this.nodeForm.config.delay_ms ??= 0;
    if(node.type === 'response')this.nodeForm.config.body ??= '';
    const example=node.config.example || defaultExample();
    this.headerRows=Object.entries(node.type === 'request' ? example.headers || {} : node.config.headers || {}).map(([key,value]) => ({key,value}));
    this.paramsText=JSON.stringify(example.params || {},null,2);this.queryText=JSON.stringify(example.query || {},null,2);this.requestBodyText=JSON.stringify(example.body || {},null,2);
    this.requestRoute={name:this.endpoint.name,method:this.endpoint.method,path:this.endpoint.path};
    this.modal='node';this.dialogError='';
  },
  saveNode(){
    try{
      const node=clone(this.nodeForm);
      if(['request','headers','response'].includes(node.type)){
        const headers=headersFromRows(this.headerRows);
        if(node.type === 'request'){
          const params=parseObject(this.paramsText,'Path parameters');
          const query=parseObject(this.queryText,'Query parameters');
          const body=parseObject(this.requestBodyText,'Request body');
          if(Object.values(params).some(value => typeof value !== 'string'))throw Error('Path parameters must have string values.');
          if(Object.values(query).some(value => !Array.isArray(value) || value.some(item => typeof item !== 'string')))throw Error('Query parameters must contain arrays of strings, for example {"page":["1"]}.');
          if(!this.requestRoute.name.trim() || !this.requestRoute.path.startsWith('/'))throw Error('Endpoint name and path are required.');
          node.config.example={params,query,headers,body};
        }else node.config.headers=headers;
      }
      if(node.type === 'response'){
        if(!Number.isInteger(node.config.status) || node.config.status < 200 || node.config.status > 599)throw Error('HTTP status must be 200–599.');
        if(node.config.body)JSON.parse(node.config.body);
      }
      if(node.type === 'condition' && !/^(params|query|headers|body)\..+/.test(node.config.field || ''))throw Error('Choose a request field such as params.id or headers.authorization.');
      if(node.type === 'delay' && (!Number.isInteger(node.config.delay_ms) || node.config.delay_ms < 0 || node.config.delay_ms > 10000))throw Error('Delay must be 0–10000 ms.');
      this.rememberChange();
      const flow=this.pendingFlow || this.endpoint.flow;
      flow.nodes=flow.nodes.map(item => item.id === node.id ? node : item);
      this.endpoint.flow=flow;
      if(node.type === 'request'){Object.assign(this.endpoint,this.requestRoute);this.testInput=JSON.stringify(node.config.example,null,2);}
      this.pendingFlow=null;this.dirty=true;this.trace=[];this.testResult=null;this.modal=null;
    }catch(error){this.dialogError=error.message;}
  },
  removeNode(){
    try{
      const type=this.nodeForm.type;
      const message=type === 'condition' ? 'Delete this condition? The matching path is preserved and the unused fallback is removed.' : 'Delete this setting? Requests will continue to the next step.';
      if(!window.confirm(message))return;
      const candidate=clone(this.endpoint.flow);
      removeConfiguration(candidate,this.nodeForm.id);
      this.rememberChange();this.endpoint.flow=candidate;this.dirty=true;this.trace=[];this.connecting=null;this.pendingFlow=null;this.modal=null;
    }catch(error){this.dialogError=error.message;}
  },
  headerEntries(node){return Object.entries(node.type === 'request' ? node.config.example?.headers || {} : node.config.headers || {});},
  headerSummary(node){return this.headerEntries(node).map(([key,value]) => `${key}: ${value}`).join(' · ') || 'No headers yet';},
  targetLabel(id,branch=''){const target=this.targetFor(id,branch);return this.endpoint.flow.nodes.find(node => node.id === target)?.label || 'Not connected';},
  bodyPreview(node){return node.config.body || '(empty body)';},
  setZoom(value){
    if(!Number.isFinite(value))return;
    const next=Math.max(this.minZoom,Math.min(this.maxZoom,value));
    if(next===this.zoom)return;
    const viewport=this.$root?.querySelector('.canvas-scroll');
    const center=viewport?{x:(viewport.scrollLeft+viewport.clientWidth/2)/this.zoom,y:(viewport.scrollTop+viewport.clientHeight/2)/this.zoom}:null;
    this.drag=null;this.zoom=next;
    if(center)this.$nextTick?.(()=>viewport.scrollTo({left:Math.max(0,center.x*next-viewport.clientWidth/2),top:Math.max(0,center.y*next-viewport.clientHeight/2)}));
  },
  startDrag(event,node){if(event.button!==0)return;event.preventDefault();this.drag={id:node.id,startX:event.clientX,startY:event.clientY,x:node.x,y:node.y,zoom:this.zoom};},
  moveNode(event){if(!this.drag||!this.endpoint)return;const node=this.endpoint.flow.nodes.find(n=>n.id===this.drag.id);if(!node)return;node.x=Math.max(20,this.drag.x+(event.clientX-this.drag.startX)/this.drag.zoom);node.y=Math.max(20,this.drag.y+(event.clientY-this.drag.startY)/this.drag.zoom);this.dirty=true;},
  startConnection(node,branch){this.connecting={from:node.id,branch};},
  connectTo(node){if(!this.connecting)return;try{const candidate=clone(this.endpoint.flow);setConnection(candidate,this.connecting.from,this.connecting.branch,node.id);this.rememberChange();this.endpoint.flow=candidate;this.dirty=true;this.trace=[];this.connecting=null;}catch(error){this.notify(error.message);}},
  targetFor(id,branch){return this.endpoint.flow.edges.find(edge => edge.from === id && (edge.branch || '') === branch)?.to || '';},
  branchesFor(id){const node=this.endpoint.flow.nodes.find(item => item.id === id);return node ? outgoingBranches(node) : [];},
  editConnection(edge){
    if(this.branchModel){this.editScenarioConnection(edge.branch==='default'?null:edge.scenario?edge.to:edge.from);return;}
    this.originalConnection={...edge};this.connectionForm={from:edge.from,to:edge.to,branch:edge.branch || ''};this.modal='connection';this.dialogError='';
  },
  editNodeConnections(node){
    if(this.branchModel){this.editScenarioConnection(node.type==='condition'?node.id:null);return;}
    const branch=outgoingBranches(node)[0];if(branch===undefined)return;
    const edge=this.endpoint.flow.edges.find(item => item.from === node.id && (item.branch || '') === branch);
    this.originalConnection=edge?{...edge}:null;this.connectionForm={from:node.id,to:edge?.to || '',branch};this.modal='connection';this.dialogError='';
  },
  editScenarioConnection(conditionID){const model=this.branchModel;const item=model.branches.find(item=>item.condition.id===conditionID);this.connectionForm={from:conditionID || model.request.id,to:item?.response.id || model.fallback.id,branch:item?'true':'default'};this.originalConnection=null;this.modal='connection';this.dialogError='';},
  connectionSourceChanged(){this.connectionForm.branch=this.branchesFor(this.connectionForm.from)[0] || '';this.connectionBranchChanged();},
  connectionBranchChanged(){const edge=this.endpoint.flow.edges.find(item => item.from === this.connectionForm.from && (item.branch || '') === this.connectionForm.branch);this.originalConnection=edge?{...edge}:null;this.connectionForm.to=edge?.to || '';},
  saveConnection(){
    try{
      const flow=clone(this.endpoint.flow);
      if(this.branchModel){
        const target=flow.nodes.find(node=>node.id===this.connectionForm.to);
        if(target?.type!=='response')throw Error('Choose a response node.');
        const model=scenarioBranches(flow),id=this.connectionForm.from;
        if(this.connectionForm.branch==='default'){
          const last=model.branches.at(-1)?.condition.id || model.request.id;
          flow.edges.find(edge=>edge.from===last && (edge.branch || '')===(model.branches.length?'false':'')).to=target.id;
        }else flow.edges.find(edge=>edge.from===id && edge.branch==='true').to=target.id;
        const reachable=new Set();const visit=id=>{if(reachable.has(id))return;reachable.add(id);flow.edges.filter(edge=>edge.from===id).forEach(edge=>visit(edge.to));};visit(model.request.id);
        flow.nodes=flow.nodes.filter(node=>reachable.has(node.id));flow.edges=flow.edges.filter(edge=>reachable.has(edge.from)&&reachable.has(edge.to));
        arrangeBranches(flow);this.rememberChange();this.endpoint.flow=flow;this.dirty=true;this.trace=[];this.modal=null;this.connecting=null;return;
      }
      if(this.originalConnection)flow.edges=flow.edges.filter(edge => !(edge.from === this.originalConnection.from && (edge.branch || '') === (this.originalConnection.branch || '')));
      setConnection(flow,this.connectionForm.from,this.connectionForm.branch,this.connectionForm.to);
      this.rememberChange();this.endpoint.flow=flow;this.dirty=true;this.trace=[];this.modal=null;this.connecting=null;
    }catch(error){this.dialogError=error.message;}
  },
  removeConnection(){
    if(!this.originalConnection)return;
    this.rememberChange();
    this.endpoint.flow.edges=this.endpoint.flow.edges.filter(edge => !(edge.from === this.originalConnection.from && (edge.branch || '') === (this.originalConnection.branch || '')));
    this.dirty=true;this.trace=[];this.modal=null;
  },
  edgePoints(edge){const p=connectionPoints(this.endpoint.flow,edge);if(this.branchModel){const from=this.endpoint.flow.nodes.find(node=>node.id===edge.from);p.y1=from.y+110;}return p;},
  edgePath(edge){const p=this.edgePoints(edge);const bend=Math.max(70,Math.abs(p.x2-p.x1)/2);return `M${p.x1},${p.y1} C${p.x1+bend},${p.y1} ${p.x2-bend},${p.y2} ${p.x2},${p.y2}`;},
  edgeMid(edge){const p=this.edgePoints(edge);return{x:(p.x1+p.x2)/2,y:(p.y1+p.y2)/2-10};},
  edgeTraced(edge){if(edge.scenario)return edge.branch==='default'?this.trace.includes(edge.to)&&!this.trace.some(id=>this.branchModel?.branches.some(item=>item.response.id===id)):this.trace.includes(edge.to)&&this.trace.includes(this.branchModel?.branches.find(item=>item.condition.id===edge.to)?.response.id);const index=this.trace.indexOf(edge.from);return index>=0&&this.trace[index+1]===edge.to;},
  conditionIcon(config){const c=normalizeCondition(config);return conditionKinds.find(kind=>kind.key===c.condition_type)?.icon || '◇';},
  conditionMeta(config){const c=normalizeCondition(config);return `${conditionKindLabel(c)}${c.condition_type==='body' ? ' · '+(c.body_mode==='form'?'Form':c.raw_format==='text'?'Raw Text':'Raw JSON') : ''} · ${c.match==='any'?'OR':'AND'} · ${c.rules.length} field`;},
  nodeIcon(type){return {request:'↳',condition:'◇',response:'↗',headers:'≡',delay:'◷'}[type] || '◇';},
  nodeSummary(node){switch(node.type){case 'request':return `${this.endpoint.method} ${this.endpoint.path}`;case 'response':return `${node.config.status} · JSON response`;case 'headers':return `${Object.keys(node.config.headers||{}).length} response headers`;case 'condition':return conditionSummary(node.config);case 'delay':return `${node.config.delay_ms} ms`;default:return '';}},
  async runTest(){this.testOpen=true;this.testing=true;this.trace=[];this.testResult=null;try{const input=JSON.parse(this.testInput);this.testResult=await this.request('/api/test','POST',{...input,flow:this.endpoint.flow,method:this.endpoint.method,path:this.endpoint.path});this.trace=this.testResult.trace;}catch(error){this.notify(error.message);}finally{this.testing=false;}},
  async copyBaseURL(){await this.copyText(this.baseURL,'Workspace base URL copied.');},
  async copyURL(){await this.copyText(this.baseURL+this.endpoint.path,'Endpoint URL copied. Replace :parameter with a value before use.');},
  async copyText(value,message){try{await navigator.clipboard.writeText(value);this.notify(message);}catch{this.notify('Copy this URL: '+value);}},
  commitModal(){if(this.modal==='node')this.saveNode();if(this.modal==='connection')this.saveConnection();if(this.modal==='direct')this.prepareDirectResponse();if(this.modal==='scenario')this.saveScenario();}
});
