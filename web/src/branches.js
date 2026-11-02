import { clone } from './flow.js';

// The persisted false-chain is the execution order; the canvas presents its alternatives.
export function scenarioBranches(flow) {
  const request=flow.nodes.find(node=>node.type==='request');
  if(!request)return null;
  const nodes=new Map(flow.nodes.map(node=>[node.id,node])), seen=new Set([request.id]), branches=[];
  const next=(id,branch='')=>nodes.get(flow.edges.find(edge=>edge.from===id && (edge.branch || '')===branch)?.to);
  let current=next(request.id);
  while(current?.type==='condition' && !seen.has(current.id)){
    seen.add(current.id);
    const response=next(current.id,'true');
    if(response?.type!=='response')return null;
    seen.add(response.id);branches.push({condition:current,response,priority:branches.length+1});
    current=next(current.id,'false');
  }
  if(current?.type!=='response')return null;
  seen.add(current.id);
  if(seen.size!==flow.nodes.length || flow.edges.length!==1+branches.length*2)return null;
  return {request,branches,fallback:current};
}

export function arrangeBranches(flow) {
  const model=scenarioBranches(flow);if(!model)return;
  model.request.x=80;model.request.y=100+model.branches.length*100;
  model.branches.forEach(({condition,response},index)=>{
    condition.x=450;condition.y=80+index*220;
    response.x=820;response.y=condition.y;
  });
  // Shared response nodes are displayed once; all matching branches still point to them.
  model.fallback.x=820;model.fallback.y=80+model.branches.length*220;
  flow.layout='scenarios';
}

export function branchEdges(flow) {
  const model=scenarioBranches(flow);if(!model)return flow.edges;
  return [
    ...model.branches.flatMap(({condition,response,priority})=>[
      {from:model.request.id,to:condition.id,branch:'scenario-'+condition.id,label:'#'+priority,scenario:true},
      {from:condition.id,to:response.id,branch:'true',label:'Match'}
    ]),
    {from:model.request.id,to:model.fallback.id,branch:'default',label:'Default',scenario:true}
  ];
}

export function branchRows(flow) {
  const model=scenarioBranches(flow);if(!model)return null;
  const row=(condition,response,priority)=>({key:condition?.id || 'default',order:priority || model.branches.length+1,priority:priority || null,responseID:response.id,pathIDs:[model.request.id,...(condition?[condition.id]:[]),response.id],conditions:condition?[{id:condition.id,matched:true}]:[],isDefault:!condition,label:condition?.label || 'Default response',status:response.config.status,body:response.config.body || '',headerIDs:[],delayIDs:[]});
  return [...model.branches.map(({condition,response,priority})=>row(condition,response,priority)),row(null,model.fallback,null)];
}

export function setPriority(flow, conditionID, priority) {
  const model=scenarioBranches(flow);
  if(!model)throw Error('Priority is available for independent condition scenarios.');
  if(!Number.isInteger(priority)||priority<1||priority>model.branches.length)throw Error('Choose a valid condition priority.');
  const order=model.branches.map(item=>item.condition.id),index=order.indexOf(conditionID);
  if(index<0)throw Error('Default response has no editable priority.');
  order.splice(index,1);order.splice(priority-1,0,conditionID);
  const original=clone(flow.edges);
  flow.edges=[{from:model.request.id,to:order[0] || model.fallback.id,branch:''},...order.flatMap((id,i)=>[
    original.find(edge=>edge.from===id && edge.branch==='true'),
    {from:id,to:order[i+1] || model.fallback.id,branch:'false'}
  ])];
  arrangeBranches(flow);
}
