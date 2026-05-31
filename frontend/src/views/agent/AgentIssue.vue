<template><n-card title="生成卡密"><n-alert type="info" style="margin-bottom:16px">代理发卡会按卡类价格自动扣除余额，余额不足无法生成。</n-alert><n-form label-placement="top"><n-form-item label="软件"><n-select v-model:value="form.appId" :options="appOptions" @update:value="form.cardTypeId=null" /></n-form-item><n-form-item label="卡类套餐"><n-select v-model:value="form.cardTypeId" :options="typeOptions" /></n-form-item><n-form-item label="数量"><n-input-number v-model:value="form.count" :min="1" :max="200" style="width:100%" /></n-form-item><n-button type="primary" :loading="loading" @click="submit">生成并扣余额</n-button></n-form><n-card v-if="keys.length" title="生成结果" style="margin-top:16px"><pre>{{ keys.join('\n') }}</pre></n-card></n-card></template>
<script setup>
import { computed, reactive, ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { get, post } from '../../api/client'
const msg=useMessage(); const scopes=ref({apps:[],cardTypes:[]}); const keys=ref([]); const loading=ref(false)
const form=reactive({appId:null,cardTypeId:null,count:1})
const appOptions=computed(()=>scopes.value.apps.map(x=>({label:x.name,value:x.id})))
const typeOptions=computed(()=>scopes.value.cardTypes.filter(x=>!form.appId||x.app_id===form.appId).map(x=>({label:`${x.name} / ${x.days}天 / ￥${x.price}`,value:x.id})))
async function load(){ scopes.value=(await get('/api/agent/scopes')).data; if(scopes.value.apps[0]) form.appId=scopes.value.apps[0].id; if(scopes.value.cardTypes[0]) form.cardTypeId=scopes.value.cardTypes[0].id }
async function submit(){ loading.value=true; try{ const r=await post('/api/agent/cards/generate',{appId:form.appId,cardTypeId:form.cardTypeId,count:form.count}); keys.value=r.data; msg.success(`生成成功，扣款￥${r.cost}`) }catch(e){ msg.error(e.response?.data?.message||'生成失败') } finally{ loading.value=false } }
onMounted(load)
</script>
