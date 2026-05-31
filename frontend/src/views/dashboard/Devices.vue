<template>
  <n-card>
    <template #header>
      <div class="card-head">
        <div>
          <b>{{ title }}</b>
          <span class="online-summary">当前在线 {{ onlineCount }} 台</span>
          <div class="muted">默认只看在线设备；需要处理被禁用/封禁的机器时，切换“查看范围”。</div>
        </div>
        <n-button @click="load">刷新</n-button>
      </div>
    </template>
    <n-space class="toolbar" wrap>
      <n-input v-model:value="filter.keyword" placeholder="搜索用户/卡密/机器码/IP/版本" clearable style="width:240px" @keyup.enter="load" />
      <n-select v-model:value="filter.appId" :options="appOptions" clearable placeholder="软件" style="width:180px" />
      <n-select v-model:value="filter.view" :options="viewOptions" placeholder="查看范围" style="width:170px" @update:value="load" />
      <n-button type="primary" @click="load">筛选</n-button>
    </n-space>
    <n-alert v-if="filter.view==='disabled'" type="warning" class="hint-alert">
      这里显示后台禁用的设备。要把设备拉出来，点“恢复”即可改回正常；恢复后客户端重新登录/心跳才会重新在线。
    </n-alert>
    <n-alert v-else-if="filter.view==='all'" type="info" class="hint-alert">
      全部历史会包含离线、禁用、封禁设备；日常管理建议切回“仅在线设备”。
    </n-alert>
    <n-data-table :columns="cols" :data="rows" :pagination="{ pageSize: 10 }" />
  </n-card>
</template>
<script setup>
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NButton, NSpace, NTag, useMessage } from 'naive-ui'
import { get, put } from '../../api/client'
import { zhStatus, statusTagType } from '../../utils/status'
const msg=useMessage()
const rows=ref([])
const apps=ref([])
const onlineCount=ref(0)
const filter=reactive({keyword:'',appId:null,view:'online'})
const viewOptions=[
  {label:'仅在线设备',value:'online'},
  {label:'禁用设备',value:'disabled'},
  {label:'封禁设备',value:'banned'},
  {label:'全部历史设备',value:'all'},
]
const appOptions=computed(()=>apps.value.map(x=>({label:x.name,value:x.id})))
const title=computed(()=>({online:'在线设备',disabled:'禁用设备',banned:'封禁设备',all:'全部历史设备'}[filter.view] || '设备管理'))
const cols=[
  {title:'ID',key:'id'},
  {title:'软件',key:'app_name'},
  {title:'授权方式',key:'auth_mode',render:r=>h(NTag,{type:r.auth_mode==='card'?'warning':'info'},()=>r.auth_mode==='card'?'卡密授权':'账号授权')},
  {title:'用户/卡密',key:'username'},
  {title:'机器码',key:'machine_code'},
  {title:'在线',key:'online_status',render:r=>h(NTag,{type:r.online_status==='online'?'success':'default',round:true},()=>r.online_text||'离线')},
  {title:'设备状态',key:'status',render:r=>h(NTag,{type:statusTagType(r.status)},()=>zhStatus(r.status))},
  {title:'版本',key:'client_version'},
  {title:'IP',key:'ip'},
  {title:'最后心跳',key:'last_seen'},
  {title:'操作',key:'actions',render:r=>h(NSpace,{size:6},()=>[
    r.status!=='active' ? h(NButton,{size:'small',type:'primary',onClick:()=>update(r,'active')},()=> '恢复') : null,
    r.status!=='disabled' ? h(NButton,{size:'small',onClick:()=>update(r,'disabled')},()=> '禁用') : null,
    r.status!=='banned' ? h(NButton,{size:'small',type:'error',onClick:()=>update(r,'banned')},()=> '封禁') : null,
  ].filter(Boolean))}
]
function query(){
  const p=new URLSearchParams()
  if(filter.view==='online') p.set('online','1')
  if(filter.view==='disabled') p.set('status','disabled')
  if(filter.view==='banned') p.set('status','banned')
  if(filter.keyword) p.set('keyword',filter.keyword)
  if(filter.appId) p.set('app_id',filter.appId)
  return p.toString()
}
async function load(){
  const res=await get('/api/admin/devices?'+query())
  rows.value=res.data
  onlineCount.value=res.online_count||0
  apps.value=(await get('/api/admin/apps')).data.map(x=>({label:x.name,value:x.id,id:x.id,name:x.name}))
}
async function update(r,status){ await put(`/api/admin/devices/${r.id}/status`,{status}); msg.success(status==='active'?'已恢复':'已更新'); load() }
onMounted(load)
</script>
<style scoped>
.card-head{display:flex;align-items:center;justify-content:space-between;gap:12px}.online-summary{margin-left:12px;color:#16a34a;font-weight:700}.muted{font-size:12px;color:#64748b;margin-top:4px}.toolbar{margin-bottom:16px}.hint-alert{margin-bottom:14px}
</style>
