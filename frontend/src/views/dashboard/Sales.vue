<template>
  <div>
    <n-card>
      <template #header><div class="card-head"><b>销量查询</b><n-button @click="load">查询</n-button></div></template>
      <n-space class="toolbar" wrap>
        <n-select v-model:value="filter.appId" :options="apps" clearable placeholder="软件" style="width:180px" />
        <n-select v-model:value="filter.agentId" :options="agentOptions" clearable placeholder="代理" style="width:180px" />
        <n-select v-model:value="filter.cardTypeId" :options="typeOptions" clearable placeholder="卡密类型" style="width:220px" />
        <n-select v-model:value="filter.timeField" :options="timeFieldOptions" placeholder="时间类型" style="width:140px" />
        <n-date-picker v-model:formatted-value="filter.startDate" value-format="yyyy-MM-dd" type="date" clearable placeholder="开始日期" style="width:150px" />
        <n-date-picker v-model:formatted-value="filter.endDate" value-format="yyyy-MM-dd" type="date" clearable placeholder="结束日期" style="width:150px" />
        <n-checkbox v-model:checked="filter.includeDisabled">包含禁用卡密</n-checkbox>
        <n-button type="primary" @click="load">筛选</n-button>
      </n-space>
    </n-card>
    <n-grid :cols="5" :x-gap="16" style="margin-top:16px">
      <n-gi><n-card class="metric"><small>销量张数</small><b>{{ summary.cards || 0 }}</b></n-card></n-gi>
      <n-gi><n-card class="metric"><small>销售金额</small><b>￥{{ money(summary.amount) }}</b></n-card></n-gi>
      <n-gi><n-card class="metric"><small>未使用</small><b>{{ summary.unused_cards || 0 }}</b></n-card></n-gi>
      <n-gi><n-card class="metric"><small>已使用</small><b>{{ summary.used_cards || 0 }}</b></n-card></n-gi>
      <n-gi><n-card class="metric"><small>已禁用</small><b>{{ summary.disabled_cards || 0 }}</b></n-card></n-gi>
    </n-grid>
    <n-grid :cols="2" :x-gap="16" style="margin-top:16px">
      <n-gi><n-card title="按代理汇总"><n-data-table :columns="agentCols" :data="report.by_agent || []" :pagination="{pageSize:8}" /></n-card></n-gi>
      <n-gi><n-card title="按卡密类型汇总"><n-data-table :columns="typeCols" :data="report.by_type || []" :pagination="{pageSize:8}" /></n-card></n-gi>
    </n-grid>
    <n-card title="明细" style="margin-top:16px"><n-data-table :columns="detailCols" :data="report.recent || []" :pagination="{pageSize:10}" /></n-card>
  </div>
</template>
<script setup>
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NTag } from 'naive-ui'
import { get } from '../../api/client'
import { statusTagType, zhStatus } from '../../utils/status'
const apps=ref([]), agents=ref([]), types=ref([]), report=ref({summary:{},recent:[],by_agent:[],by_type:[]})
const filter=reactive({appId:null,agentId:null,cardTypeId:null,timeField:'created_at',startDate:null,endDate:null,includeDisabled:false})
const summary=computed(()=>report.value.summary || {})
const agentOptions=computed(()=>agents.value.map(a=>({label:a.username,value:a.id})))
const typeOptions=computed(()=>types.value.filter(t=>!filter.appId||t.app_id===filter.appId).map(t=>({label:`${t.app_name}-${t.name}`,value:t.id})))
const timeFieldOptions=[{label:'生成时间',value:'created_at'},{label:'激活时间',value:'used_at'}]
function money(v){ return Number(v||0).toFixed(2) }
function qs(){ const p=new URLSearchParams(); if(filter.appId)p.set('app_id',filter.appId); if(filter.agentId)p.set('agent_id',filter.agentId); if(filter.cardTypeId)p.set('card_type_id',filter.cardTypeId); if(filter.timeField)p.set('time_field',filter.timeField); if(filter.startDate)p.set('start_date',filter.startDate); if(filter.endDate)p.set('end_date',filter.endDate); if(filter.includeDisabled)p.set('include_disabled','1'); return p.toString() }
const agentCols=[{title:'代理',key:'agent_name'},{title:'销量',key:'count'},{title:'金额',key:'amount',render:r=>'￥'+money(r.amount)}]
const typeCols=[{title:'卡密类型',key:'card_type_name'},{title:'销量',key:'count'},{title:'金额',key:'amount',render:r=>'￥'+money(r.amount)}]
const detailCols=[{title:'ID',key:'id'},{title:'软件',key:'app_name'},{title:'卡类',key:'card_type_name'},{title:'卡密',key:'card_key'},{title:'代理',key:'agent_name'},{title:'状态',key:'status',render:r=>h(NTag,{type:statusTagType(r.status)},()=>zhStatus(r.status))},{title:'金额',key:'cost',render:r=>'￥'+money(r.cost)},{title:'生成时间',key:'created_at'},{title:'激活时间',key:'used_at'},{title:'使用者',key:'used_by'}]
async function load(){ report.value=(await get('/api/admin/sales?'+qs())).data }
async function init(){ apps.value=(await get('/api/admin/apps')).data.map(a=>({label:a.name,value:a.id})); agents.value=(await get('/api/admin/agents')).data; types.value=(await get('/api/admin/card-types')).data; await load() }
onMounted(init)
</script>
