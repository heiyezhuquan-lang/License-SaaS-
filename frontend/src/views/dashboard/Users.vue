<template>
  <n-card>
    <template #header><div class="card-head"><b>用户管理</b><n-button type="primary" @click="openCreate">新增用户</n-button></div></template>
    <n-space class="toolbar" wrap>
      <n-input v-model:value="filter.keyword" placeholder="搜索用户名/机器码" clearable style="width:220px" @keyup.enter="load" />
      <n-select v-model:value="filter.status" :options="statusOptions" clearable placeholder="状态" style="width:140px" />
      <n-select v-model:value="filter.appId" :options="apps" clearable placeholder="软件" style="width:180px" />
      <n-button type="primary" @click="load">筛选</n-button>
    </n-space>
    <n-data-table :columns="cols" :data="rows" :pagination="{pageSize:10}" />
  </n-card>
  <n-modal v-model:show="show" preset="card" :title="editing ? '编辑用户' : '新增用户'" class="modal">
    <n-form label-placement="top">
      <n-form-item label="软件"><n-select v-model:value="form.appId" :options="apps" :disabled="!!editing" /></n-form-item>
      <n-form-item v-if="!editing" label="用户名"><n-input v-model:value="form.username" /></n-form-item>
      <n-form-item v-if="!editing" label="密码"><n-input v-model:value="form.password" type="password" /></n-form-item>
      <n-form-item label="状态"><n-select v-model:value="form.status" :options="statusOptions" /></n-form-item>
      <n-form-item label="到期时间"><n-input v-model:value="form.expireAt" placeholder="例如 2026-12-31 23:59:59，留空为未设置" /></n-form-item>
      <n-form-item label="机器码"><n-input v-model:value="form.machineCode" placeholder="留空表示不绑定" /></n-form-item>
      <n-form-item label="最大设备"><n-input-number v-model:value="form.maxDevices" :min="1" /></n-form-item>
      <n-form-item label="免费解绑"><n-input-number v-model:value="form.freeUnbinds" :min="0" /></n-form-item>
      <n-form-item label="最多解绑"><n-input-number v-model:value="form.maxUnbinds" :min="0" /></n-form-item>
      <n-form-item label="已解绑"><n-input-number v-model:value="form.unbindUsed" :min="0" disabled /></n-form-item>
      <n-form-item label="解绑扣时"><n-input-number v-model:value="form.unbindDeductHours" :min="0" /></n-form-item>
      <n-button type="primary" @click="save">保存</n-button>
    </n-form>
  </n-modal>
  <n-modal v-model:show="passwordShow" preset="card" title="修改密码" class="modal">
    <n-form label-placement="top"><n-form-item label="新密码"><n-input v-model:value="newPassword" type="password" /></n-form-item><n-button type="primary" @click="savePassword">确认修改</n-button></n-form>
  </n-modal>
  <n-modal v-model:show="devicesShow" preset="card" title="用户设备" class="modal">
    <n-data-table :columns="deviceCols" :data="devices" :pagination="{pageSize:8}" />
  </n-modal>
</template>
<script setup>
import { h, onMounted, reactive, ref } from 'vue'
import { NButton, NPopconfirm, NSpace, NTag, useMessage } from 'naive-ui'
import { del, get, post, put } from '../../api/client'
import { statusTagType, zhStatus } from '../../utils/status'
const msg=useMessage(), rows=ref([]), apps=ref([]), show=ref(false), passwordShow=ref(false), devicesShow=ref(false), devices=ref([]), editing=ref(null), pwdUser=ref(null), newPassword=ref('')
const statusOptions=[{label:'正常',value:'active'},{label:'禁用',value:'disabled'},{label:'冻结',value:'frozen'}]
const filter=reactive({keyword:'',status:null,appId:null})
const form=reactive({appId:null,username:'',password:'123456',status:'active',expireAt:'',machineCode:'',maxDevices:1,freeUnbinds:0,maxUnbinds:0,unbindUsed:0,unbindDeductHours:0})
const cols=[{title:'ID',key:'id'},{title:'软件',key:'app_name'},{title:'用户名',key:'username'},{title:'状态',key:'status',render:r=>h(NTag,{type:statusTagType(r.status)},()=>zhStatus(r.status))},{title:'到期时间',key:'expire_at'},{title:'机器码',key:'machine_code'},{title:'设备数',key:'max_devices'},{title:'解绑',key:'unbind_rule',render:r=>`${r.unbind_used||0}/${r.max_unbinds||0}（免费${r.free_unbinds||0}，扣${r.unbind_deduct_hours||0}h）`},{title:'操作',key:'actions',render:r=>h(NSpace,null,()=>[
  h(NButton,{size:'small',onClick:()=>openEdit(r)},()=> '编辑'),
  h(NButton,{size:'small',onClick:()=>openPwd(r)},()=> '改密'),
  h(NButton,{size:'small',onClick:()=>viewDevices(r)},()=> '设备'),
  h(NButton,{size:'small',type:'warning',onClick:()=>unbind(r)},()=> '解绑'),
  r.status==='active'?h(NButton,{size:'small',type:'error',onClick:()=>quickStatus(r,'disabled')},()=> '禁用'):h(NButton,{size:'small',type:'primary',onClick:()=>quickStatus(r,'active')},()=> '启用'),
  h(NPopconfirm,{onPositiveClick:()=>deleteUser(r)},{trigger:()=>h(NButton,{size:'small',type:'error',secondary:true},()=> '删除'),default:()=>`确定删除用户「${r.username}」吗？该操作会同时删除绑定设备，不能恢复。`})
])}]
const deviceCols=[{title:'ID',key:'id'},{title:'机器码',key:'machine_code'},{title:'在线',key:'online_status',render:r=>h(NTag,{type:r.online_status==='online'?'success':'default',round:true},()=>r.online_text || (r.online_status==='online'?'在线':'离线'))},{title:'状态',key:'status',render:r=>h(NTag,{type:statusTagType(r.status)},()=>zhStatus(r.status))},{title:'版本',key:'client_version'},{title:'IP',key:'ip'},{title:'最后心跳',key:'last_seen'}]
function qs(){ const p=new URLSearchParams(); if(filter.keyword)p.set('keyword',filter.keyword); if(filter.status)p.set('status',filter.status); if(filter.appId)p.set('app_id',filter.appId); return p.toString() }
async function load(){ rows.value=(await get('/api/admin/users?'+qs())).data; apps.value=(await get('/api/admin/apps')).data.map(a=>({label:a.name,value:a.id})); if(!form.appId&&apps.value[0]) form.appId=apps.value[0].value }
function reset(){ Object.assign(form,{appId:apps.value[0]?.value||null,username:'',password:'123456',status:'active',expireAt:'',machineCode:'',maxDevices:1,freeUnbinds:0,maxUnbinds:0,unbindUsed:0,unbindDeductHours:0}) }
function openCreate(){ editing.value=null; reset(); show.value=true }
function openEdit(r){ editing.value=r; Object.assign(form,{appId:r.app_id,username:r.username,password:'',status:r.status||'active',expireAt:r.expire_at||'',machineCode:r.machine_code||'',maxDevices:r.max_devices||1,freeUnbinds:r.free_unbinds||0,maxUnbinds:r.max_unbinds||0,unbindUsed:r.unbind_used||0,unbindDeductHours:r.unbind_deduct_hours||0}); show.value=true }
async function save(){ if(editing.value){ await put(`/api/admin/users/${editing.value.id}`,{status:form.status,expireAt:form.expireAt,machineCode:form.machineCode,maxDevices:form.maxDevices,freeUnbinds:form.freeUnbinds,maxUnbinds:form.maxUnbinds,unbindDeductHours:form.unbindDeductHours}); msg.success('已保存') } else { await post('/api/admin/users',{appId:form.appId,username:form.username,password:form.password,maxDevices:form.maxDevices}); msg.success('已创建') } show.value=false; await load() }
function openPwd(r){ pwdUser.value=r; newPassword.value=''; passwordShow.value=true }
async function savePassword(){ await put(`/api/admin/users/${pwdUser.value.id}/password`,{password:newPassword.value}); msg.success('密码已修改'); passwordShow.value=false }
async function unbind(r){ await put(`/api/admin/users/${r.id}/unbind`,{}); msg.success('已解绑，旧设备记录保留'); await load() }
async function quickStatus(r,status){ await put(`/api/admin/users/${r.id}`,{status,expireAt:r.expire_at||'',machineCode:r.machine_code||'',maxDevices:r.max_devices||1,freeUnbinds:r.free_unbinds||0,maxUnbinds:r.max_unbinds||0,unbindDeductHours:r.unbind_deduct_hours||0}); msg.success('状态已更新'); await load() }
async function viewDevices(r){ devices.value=(await get(`/api/admin/users/${r.id}/devices`)).data; devicesShow.value=true }
async function deleteUser(r){ await del(`/api/admin/users/${r.id}`); msg.success('用户已删除'); await load() }
onMounted(load)
</script>