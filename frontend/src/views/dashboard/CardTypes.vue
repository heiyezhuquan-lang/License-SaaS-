<template>
  <n-card>
    <template #header>
      <div class="card-head"><b>卡类套餐</b><n-button type="primary" @click="openCreate">新增卡类</n-button></div>
    </template>
    <n-data-table :columns="cols" :data="rows" :pagination="{pageSize:10}" />
  </n-card>
  <n-modal v-model:show="show" preset="card" title="新增卡类" class="modal">
    <n-form label-placement="left" label-width="100">
      <n-form-item label="软件"><n-select v-model:value="form.appId" :options="apps" /></n-form-item>
      <n-form-item label="名称"><n-input v-model:value="form.name" placeholder="例如：2小时体验卡 / 720小时月卡" /></n-form-item>
      <n-form-item label="授权小时"><n-input-number v-model:value="form.hours" :min="1" placeholder="按小时设置授权时长" /></n-form-item>
      <n-form-item label="最大设备"><n-input-number v-model:value="form.maxDevices" :min="1" /></n-form-item>
      <n-form-item label="免费解绑"><n-input-number v-model:value="form.freeUnbinds" :min="0" /></n-form-item>
      <n-form-item label="最多解绑"><n-input-number v-model:value="form.maxUnbinds" :min="0" placeholder="0 表示不限制" /></n-form-item>
      <n-form-item label="解绑扣时"><n-input-number v-model:value="form.unbindDeductHours" :min="0" placeholder="超过免费次数后每次扣除小时数" /></n-form-item>
      <n-form-item label="价格"><n-input-number v-model:value="form.price" :min="0" /></n-form-item>
      <n-button type="primary" @click="save">保存</n-button>
    </n-form>
  </n-modal>
</template>
<script setup>
import { h, onMounted, reactive, ref } from 'vue'
import { NButton, NPopconfirm, NTag, useMessage } from 'naive-ui'
import { del, get, post } from '../../api/client'
const msg=useMessage(), rows=ref([]), apps=ref([]), show=ref(false)
const form=reactive({appId:null,name:'2小时体验卡',hours:2,maxDevices:1,freeUnbinds:1,maxUnbinds:3,unbindDeductHours:24,price:1})
function durationLabel(hours){ const h=Number(hours||0); if(!h) return '-'; if(h%24===0) return `${h} 小时（${h/24} 天）`; return `${h} 小时` }
const cols=[
  {title:'ID',key:'id'},
  {title:'软件',key:'app_name'},
  {title:'名称',key:'name'},
  {title:'授权时长',key:'hours',render:r=>durationLabel(r.hours || Number(r.days||0)*24)},
  {title:'设备',key:'max_devices'},
  {title:'解绑规则',key:'unbind_rule',render:r=>`免费 ${r.free_unbinds||0} 次 / 最多 ${r.max_unbinds||0} 次 / 扣 ${r.unbind_deduct_hours||0} 小时`},
  {title:'价格',key:'price'},
  {title:'状态',key:'status',render:r=>h(NTag,{type:r.status==='active'?'success':'error'},()=>r.status==='active'?'启用':'停用')},
  {title:'操作',key:'actions',render:r=>h(NPopconfirm,{onPositiveClick:()=>remove(r)}, {
    trigger:()=>h(NButton,{size:'small',type:'error',secondary:true},()=> '删除'),
    default:()=>`确定删除套餐「${r.name}」吗？已有卡密使用的套餐会被系统拒绝删除。`
  })}
]
function openCreate(){ show.value=true }
async function load(){ rows.value=(await get('/api/admin/card-types')).data; apps.value=(await get('/api/admin/apps')).data.map(a=>({label:a.name,value:a.id})); if(!form.appId&&apps.value[0]) form.appId=apps.value[0].value }
async function save(){ await post('/api/admin/card-types',form); msg.success('已创建'); show.value=false; await load() }
async function remove(r){ try{ await del(`/api/admin/card-types/${r.id}`); msg.success('已删除'); await load() }catch(e){ msg.error(e.response?.data?.message || '删除失败') } }
onMounted(load)
</script>
<style scoped>.card-head{display:flex;align-items:center;justify-content:space-between}.modal{max-width:620px}</style>
