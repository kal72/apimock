export const workspaceEditor = {
 workspaceMenuOpen:false, members:[], memberForm:{username:'',role:'viewer'},
 async openMembers(){
  this.modal='members';this.dialogError='';this.members=[];this.memberForm={username:'',role:'viewer'};
  try{this.members=await this.request(`/api/workspaces/${this.workspace.id}/members`);}catch(error){this.dialogError=error.message;}
 },
 async saveMember(username=this.memberForm.username,role=this.memberForm.role){
  this.busy=true;this.dialogError='';
  try{await this.request(`/api/workspaces/${this.workspace.id}/members/${encodeURIComponent(username.trim())}`,'PUT',{role});await this.refreshMembership();this.memberForm={username:'',role:'viewer'};}catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 async removeMember(member){
  if(!window.confirm(`Remove ${member.name} from this workspace?`))return;
  this.busy=true;this.dialogError='';
  try{await this.request(`/api/workspaces/${this.workspace.id}/members/${encodeURIComponent(member.username)}`,'DELETE');await this.refreshMembership();}catch(error){this.dialogError=error.message;}finally{this.busy=false;}
 },
 async refreshMembership(){
  this.workspaces=await this.request('/api/workspaces');this.workspace=this.workspaces.find(w=>w.id===this.workspaceID)||null;
  if(!this.workspace || !this.canManageWorkspace){this.modal=null;this.dirty=false;await this.selectWorkspace(this.workspace?.id || this.workspaces[0]?.id || '');return;}
  this.members=await this.request(`/api/workspaces/${this.workspace.id}/members`);
 }
};
