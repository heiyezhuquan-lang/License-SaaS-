<template>
  <n-card>
    <template #header><div class="card-head"><b>代理管理</b><n-button type="primary" @click="openCreate">新增代理</n-button></div></template>
    <n-data-table :columns="cols" :data="rows" :pagination="{pageSize:10}" />
  </n-card>
  <n-modal v-model:show="show"><n-card class="modal" :title="editing?'编辑代理':'新增代理'">
    <n-form label-placement="top">
      <n-form-item label="代理账号"><n-input v-model:value="form.username" /></n-form-item>
      <n-form-item :label="editing?'密码（留空不修改）':'密码'"><n-input v-model:value="form.password" type="password" /></n-form-item>
      <n-form-item label="状态"><n-select v-model:value="form.status" :options="[{label:'启用',value:'active'},{label:'禁用',value:'disabled'}]" /></n-form-item>
      <n-form-item v-if="!editing" label="初始余额"><n-input-number v-model:value="form.balance" style="width:100%" /></n-form-item>
      <n-form-item label="可发软件"><n-select v-model:value="form.appIds" multiple :options="appOptions" /></n-form-item>
      <n-form-item label="可发卡类"><n-select v-model:value="form.cardTypeIds" multiple :options="typeOptions" /></n-form-item>
      <n-form-item label="备注"><n-input v-model:value="form.remark" type="textarea" /></n-form-item>
      <n-button type="primary" block @click="save">保存</n-button>
    </n-form>
  </n-card></n-modal>
  <n-modal v-model:show="balanceShow"><n-card class="modal" title="余额调整">
    <n-form label-placement="top">
      <n-form-item label="类型"><n-select v-model:value="balance.type" :options="[{label:'加余额',value:'add'},{label:'扣余额',value:'deduct'}]" /></n-form-item>
      <n-form-item label="金额"><n-input-number v-model:value="balance.amount" style="width:100%" /></n-form-item>
      <n-form-item label="备注"><n-input v-model:value="balance.remark" /></n-form-item>
      <n-button type="primary" block @click="saveBalance">确认调整</n-button>
    </n-form>
  </n-card></n-modal>
</template>
<script setup>
import { h, reactive, ref, onMounted } from 'vue'
import { NButton, NSpace, NTag, useMessage } from 'naive-ui'
import { get, post, put, del } from '../../api/client'
const msg=useMessage(); const rows=ref([]); const apps=ref([]); const types=ref([]); const show=ref(false); const balanceShow=ref(false); const editing=ref(null)
const form=reactive({username:'',password:'',status:'active',balance:0,remark:'',appIds:[],cardTypeIds:[]})
const balance=reactive({id:0,type:'add',amount:100,remark:''})
const appOptions=ref([]); const typeOptions=ref([])
const cols=[{title:'ID',key:'id'},{title:'账号',key:'username'},{title:'状态',key:'status',render:r=>h(NTag,{type:r.status==='active'?'success':'error'},()=>r.status==='active'?'启用':'禁用')},{title:'余额',key:'balance'},{title:'备注',key:'remark'},{title:'创建时间',key:'created_at'},{title:'操作',key:'actions',render:r=>h(NSpace,null,()=>[h(NButton,{size:'small',onClick:()=>openEdit(r)},()=> '编辑'),h(NButton,{size:'small',type:'primary',onClick:()=>openBalance(r)},()=> '余额'),r.status==='active'?h(NButton,{size:'small',type:'warning',onClick:()=>setStatus(r,'disabled')},()=> '禁用'):h(NButton,{size:'small',type:'success',onClick:()=>setStatus(r,'active')},()=> '启用'),h(NButton,{size:'small',type:'error',onClick:()=>removeAgent(r)},()=> '删除')])}]
function reset(){ Object.assign(form,{username:'',password:'',status:'active',balance:0,remark:'',appIds:apps.value.map(x=>x.id),cardTypeIds:types.value.map(x=>x.id)}) }
async function load(){ rows.value=(await get('/api/admin/agents')).data; apps.value=(await get('/api/admin/apps')).data; types.value=(await get('/api/admin/card-types')).data; appOptions.value=apps.value.map(x=>({label:x.name,value:x.id})); typeOptions.value=types.value.map(x=>({label:`${x.app_name}-${x.name} ￥${x.price}`,value:x.id})) }
function openCreate(){ editing.value=null; reset(); show.value=true }
function openEdit(r){ editing.value=r; Object.assign(form,{username:r.username,password:'',status:r.status,remark:r.remark||'',balance:r.balance||0,appIds:apps.value.map(x=>x.id),cardTypeIds:types.value.map(x=>x.id)}); show.value=true }
async function save(){ try{ if(editing.value) await put(`/api/admin/agents/${editing.value.id}`,form); else await post('/api/admin/agents',form); msg.success('保存成功'); show.value=false; load() }catch(e){ msg.error(e.response?.data?.message||'保存失败') } }
function openBalance(r){ Object.assign(balance,{id:r.id,type:'add',amount:100,remark:''}); balanceShow.value=true }
async function saveBalance(){ try{ await post(`/api/admin/agents/${balance.id}/balance`,balance); msg.success('调整成功'); balanceShow.value=false; load() }catch(e){ msg.error(e.response?.data?.message||'调整失败') } }
async function setStatus(r,status){ try{ await put(`/api/admin/agents/${r.id}/status`,{status}); msg.success(status==='active'?'代理已启用':'代理已禁用，该代理制作的未使用卡密将无法继续激活/充值'); load() }catch(e){ msg.error(e.response?.data?.message||'操作失败') } }
async function removeAgent(r){ if(!window.confirm(`确定删除代理「${r.username}」吗？\n\n此操作会把该代理生成的所有卡密全部删除，且不可恢复。`)) return; try{ await del(`/api/admin/agents/${r.id}`,{}); msg.success('代理已删除，该代理生成的卡密已全部删除'); load() }catch(e){ msg.error(e.response?.data?.message||'删除失败') } }
onMounted(load)
</script>
