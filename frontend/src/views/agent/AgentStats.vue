<template>
  <div>
    <n-card>
      <template #header><div class="card-head"><b>销售统计</b><n-button @click="load">查询</n-button></div></template>
      <n-space class="toolbar" wrap>
        <n-select v-model:value="filter.cardTypeId" :options="typeOptions" clearable placeholder="卡密类型" style="width:220px" />
        <n-select v-model:value="filter.timeField" :options="timeFieldOptions" placeholder="时间类型" style="width:140px" />
        <n-date-picker v-model:formatted-value="filter.startDate" value-format="yyyy-MM-dd" type="date" clearable placeholder="开始日期" style="width:150px" />
        <n-date-picker v-model:formatted-value="filter.endDate" value-format="yyyy-MM-dd" type="date" clearable placeholder="结束日期" style="width:150px" />
        <n-checkbox v-model:checked="filter.includeDisabled">包含禁用卡密</n-checkbox>
        <n-button type="primary" @click="load">筛选</n-button>
      </n-space>
    </n-card>
    <n-grid :cols="4" :x-gap="16" style="margin-top:16px">
      <n-gi><n-card class="metric"><small>销量张数</small><b>{{ summary.cards || 0 }}</b></n-card></n-gi>
      <n-gi><n-card class="metric"><small>扣款金额</small><b>￥{{ money(summary.amount) }}</b></n-card></n-gi>
      <n-gi><n-card class="metric"><small>未使用</small><b>{{ summary.unused_cards || 0 }}</b></n-card></n-gi>
      <n-gi><n-card class="metric"><small>已使用</small><b>{{ summary.used_cards || 0 }}</b></n-card></n-gi>
    </n-grid>
    <n-grid :cols="2" :x-gap="16" style="margin-top:16px">
      <n-gi><n-card title="按卡密类型汇总"><n-data-table :columns="typeCols" :data="report.by_type || []" :pagination="{pageSize:8}" /></n-card></n-gi>
      <n-gi><n-card title="余额流水"><n-data-table :columns="logCols" :data="logs" :pagination="{pageSize:8}" /></n-card></n-gi>
    </n-grid>
    <n-card title="销售明细" style="margin-top:16px"><n-data-table :columns="detailCols" :data="report.recent || []" :pagination="{pageSize:10}" /></n-card>
  </div>
</template>
<script setup>
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NTag } from 'naive-ui'
import { get } from '../../api/client'
import { statusTagType, zhStatus } from '../../utils/status'
const report=ref({summary:{},recent:[],by_type:[]}), logs=ref([]), scopes=ref({cardTypes:[]})
const filter=reactive({cardTypeId:null,timeField:'created_at',startDate:null,endDate:null,includeDisabled:false})
const summary=computed(()=>report.value.summary || {})
const typeOptions=computed(()=>scopes.value.cardTypes.map(t=>({label:`${t.app_name}-${t.name}`,value:t.id})))
const timeFieldOptions=[{label:'生成时间',value:'created_at'},{label:'激活时间',value:'used_at'}]
function money(v){ return Number(v||0).toFixed(2) }
function qs(){ const p=new URLSearchParams(); if(filter.cardTypeId)p.set('card_type_id',filter.cardTypeId); if(filter.timeField)p.set('time_field',filter.timeField); if(filter.startDate)p.set('start_date',filter.startDate); if(filter.endDate)p.set('end_date',filter.endDate); if(filter.includeDisabled)p.set('include_disabled','1'); return p.toString() }
const typeCols=[{title:'卡密类型',key:'card_type_name'},{title:'销量',key:'count'},{title:'金额',key:'amount',render:r=>'￥'+money(r.amount)}]
const logCols=[{title:'类型',key:'type'},{title:'金额',key:'amount'},{title:'变动后',key:'after_balance'},{title:'备注',key:'remark'},{title:'时间',key:'created_at'}]
const detailCols=[{title:'软件',key:'app_name'},{title:'卡类',key:'card_type_name'},{title:'卡密',key:'card_key'},{title:'状态',key:'status',render:r=>h(NTag,{type:statusTagType(r.status)},()=>zhStatus(r.status))},{title:'金额',key:'cost',render:r=>'￥'+money(r.cost)},{title:'生成时间',key:'created_at'},{title:'激活时间',key:'used_at'},{title:'使用者',key:'used_by'}]
async function load(){ report.value=(await get('/api/agent/sales?'+qs())).data; logs.value=(await get('/api/agent/balance-logs')).data }
async function init(){ scopes.value=(await get('/api/agent/scopes')).data; await load() }
onMounted(init)
</script>