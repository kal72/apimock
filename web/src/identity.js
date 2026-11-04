export function createLogin() {
  return {
    setup:false, loading:true, busy:false, error:'', name:'', username:'', password:'', confirmation:'',
    async init(){try{const response=await fetch('/api/auth/status');if(!response.ok)throw Error('Unable to load sign-in settings.');this.setup=(await response.json()).setup_required;}catch(error){this.error=error.message;}finally{this.loading=false;}},
    async submit(){
      this.error='';if(this.setup&&this.password!==this.confirmation){this.error='Passwords do not match.';return;}
      this.busy=true;
      try{
        const response=await fetch(this.setup?'/api/auth/setup':'/api/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(this.setup?{name:this.name,username:this.username,password:this.password}:{username:this.username,password:this.password})});
        const result=await response.json();if(!response.ok)throw Error(result.error || 'Unable to sign in.');
        this.password='';this.confirmation='';
        if(this.setup){this.setup=false;this.error='Administrator created. Sign in to continue.';}else window.location.assign('/app');
      }catch(error){this.error=error.message;}finally{this.busy=false;}
    }
  };
}

export const identityEditor={
  currentUser:null,csrfToken:'',users:[],userForm:null,passwordForm:{current:'',next:'',confirmation:''},
  async loadIdentity(){const data=await this.request('/api/auth/me');this.currentUser=data.user;this.csrfToken=data.csrf_token;},
  async logout(){if(!this.mayDiscard())return;try{await this.request('/api/auth/logout','POST',{});window.location.assign('/login');}catch(error){this.notify(error.message);}},
  async openUsers(){try{this.users=await this.request('/api/users');this.modal='users';this.dialogError='';}catch(error){this.notify(error.message);}},
  editUser(user){this.userForm=user?{id:user.id,username:user.username,name:user.name,role:user.role,active:user.active,password:''}:{id:'',username:'',name:'',role:'user',active:true,password:''};this.modal='user';this.dialogError='';},
  async saveUser(){this.busy=true;try{const {id,...input}=this.userForm;await this.request('/api/users'+(id?'/'+id:''),id?'PUT':'POST',input);this.userForm.password='';await this.openUsers();}catch(error){this.dialogError=error.message;}finally{this.busy=false;}},
  async deleteUser(user){if(!window.confirm('Delete user "'+user.username+'"? Their sessions will be revoked. Workspaces are preserved.'))return;try{await this.request('/api/users/'+user.id,'DELETE');await this.openUsers();}catch(error){this.dialogError=error.message;}},
  openPassword(){this.passwordForm={current:'',next:'',confirmation:''};this.dialogError='';this.modal='password';},
  async changePassword(){if(this.passwordForm.next!==this.passwordForm.confirmation){this.dialogError='Passwords do not match.';return;}this.busy=true;try{await this.request('/api/auth/password','POST',{current_password:this.passwordForm.current,new_password:this.passwordForm.next});this.passwordForm={current:'',next:'',confirmation:''};window.location.assign('/login');}catch(error){this.dialogError=error.message;}finally{this.busy=false;}}
};
