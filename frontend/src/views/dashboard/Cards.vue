<template>
  <n-card>
    <template #header><div class="card-head"><b>卡密管理</b><n-space><n-button type="primary" @click="show=true">生成卡密</n-button></n-space></div></template>
    <n-space class="toolbar" wrap>
      <n-input v-model:value="filter.keyword" placeholder="搜索卡密/用户/代理" clearable style="width:220px" @keyup.enter="load" />
      <n-select v-model:value="filter.status" :options="statusOptions" clearable placeholder="状态" style="width:140px" />
      <n-select v-model:value="filter.appId" :options="apps" clearable placeholder="软件" style="width:180px" />
      <n-select v-model:value="filter.agentId" :options="agents" clearable placeholder="代理" style="width:180px" />
      <n-button type="primary" @click="load">筛选</n-button>
    </n-space>
    <n-data-table :columns="cols" :data="rows" :pagination="{pageSize:10}" />
  </n-card>
  <n-modal v-model:show="show" preset="card" title="批量生成卡密" class="modal"><n-form><n-form-item label="软件"><n-select v-model:value="form.appId" :options="apps" /></n-form-item><n-form-item label="卡类"><n-select v-model:value="form.cardTypeId" :options="types" placeholder="必须选择卡类套餐" /></n-form-item><n-form-item label="数量"><n-input-number v-model:value="form.count" :min="1" :max="200" /></n-form-item><n-button type="primary" @click="save">生成</n-button></n-form><n-alert v-if="generated.length" type="success" style="margin-top:14px"><pre>{{ generated.join('\n') }}</pre></n-alert></n-modal>
</template>
<script setup>
import { h, onMounted, reactive, ref } from 'vue'
import { NButton, NSpace, NTag, useMessage } from 'naive-ui'
import { get, post, put } from '../../api/client'
import { statusTagType, zhStatus } from '../../utils/status'
const msg=useMessage(), rows=ref([]), apps=ref([]), types=ref([]), agents=ref([]), show=ref(false), generated=ref([])
const form=reactive({appId:null,cardTypeId:null,count:5})
const filter=reactive({keyword:'',status:null,appId:null,agentId:null})
const statusOptions=[{label:'未使用',value:'unused'},{label:'已使用',value:'used'},{label:'已禁用',value:'disabled'}]
function durationLabel(hours){ const h=Number(hours||0); if(!h) return '-'; if(h%24===0) return `${h} 小时（${h/24} 天）`; return `${h} 小时` }
const cols=[{title:'ID',key:'id'},{title:'软件',key:'app_name'},{title:'卡密',key:'card_key'},{title:'状态',key:'status',render:r=>h(NTag,{type:statusTagType(r.status)},()=>zhStatus(r.status))},{title:'代理',key:'agent_name'},{title:'时长',key:'expire_hours',render:r=>durationLabel(r.expire_hours || Number(r.expire_days||0)*24)},{title:'解绑规则',key:'unbind_rule',render:r=>`${r.unbind_used||0}/${r.max_unbinds||0}（免费${r.free_unbinds||0}，扣${r.unbind_deduct_hours||0}h）`},{title:'使用者',key:'used_by'},{title:'使用时间',key:'used_at'},{title:'操作',key:'actions',render:r=>h(NSpace,null,()=>[r.status==='disabled'?h(NButton,{size:'small',onClick:()=>enable(r)},()=> '解禁'):h(NButton,{size:'small',type:'error',onClick:()=>disable(r)},()=> '禁用')])}]
function qs(){ const p=new URLSearchParams(); if(filter.keyword)p.set('keyword',filter.keyword); if(filter.status)p.set('status',filter.status); if(filter.appId)p.set('app_id',filter.appId); if(filter.agentId)p.set('agent_id',filter.agentId); return p.toString() }
async function load(){ rows.value=(await get('/api/admin/cards?'+qs())).data; apps.value=(await get('/api/admin/apps')).data.map(a=>({label:a.name,value:a.id})); types.value=(await get('/api/admin/card-types')).data.map(t=>({label:`${t.app_name}-${t.name} · ${durationLabel(t.hours || Number(t.days||0)*24)}`,value:t.id})); agents.value=(await get('/api/admin/agents')).data.map(a=>({label:a.username,value:a.id})); if(!form.appId&&apps.value[0]) form.appId=apps.value[0].value }
async function save(){
  if(!form.appId){ msg.error('请选择软件'); return }
  if(!form.cardTypeId){ msg.error('请选择卡类套餐'); return }
  try{ const r=await post('/api/admin/cards/generate',form); generated.value=r.data; msg.success('已生成'); await load() }
  catch(e){ msg.error(e.response?.data?.message||'生成失败') }
}
async function disable(r){ await put(`/api/admin/cards/${r.id}/disable`,{}); msg.success('已禁用'); load() }
async function enable(r){ await put(`/api/admin/cards/${r.id}/enable`,{}); msg.success('已解禁'); load() }
onMounted(load)
</script>