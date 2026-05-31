<template>
  <div>
    <n-card>
      <template #header><div class="card-head"><b>我的卡密</b><n-button @click="load">查询</n-button></div></template>
      <n-space class="toolbar" wrap>
        <n-select v-model:value="filter.cardTypeId" :options="typeOptions" clearable placeholder="卡密类型" style="width:220px" />
        <n-select v-model:value="filter.timeField" :options="timeFieldOptions" placeholder="时间类型" style="width:140px" />
        <n-date-picker v-model:formatted-value="filter.startDate" value-format="yyyy-MM-dd" type="date" clearable placeholder="开始日期" style="width:150px" />
        <n-date-picker v-model:formatted-value="filter.endDate" value-format="yyyy-MM-dd" type="date" clearable placeholder="结束日期" style="width:150px" />
        <n-checkbox v-model:checked="filter.includeDisabled">包含禁用卡密</n-checkbox>
        <n-button type="primary" @click="load">筛选</n-button>
      </n-space>
      <n-grid :cols="4" :x-gap="16" style="margin:8px 0 16px">
        <n-gi><n-card class="metric"><small>销量张数</small><b>{{ summary.cards || 0 }}</b></n-card></n-gi>
        <n-gi><n-card class="metric"><small>扣款金额</small><b>￥{{ money(summary.amount) }}</b></n-card></n-gi>
        <n-gi><n-card class="metric"><small>未使用</small><b>{{ summary.unused_cards || 0 }}</b></n-card></n-gi>
        <n-gi><n-card class="metric"><small>已使用</small><b>{{ summary.used_cards || 0 }}</b></n-card></n-gi>
      </n-grid>
      <n-data-table :columns="cols" :data="rows" :pagination="{pageSize:10}" />
    </n-card>
  </div>
</template>
<script setup>
import { computed, h, reactive, ref, onMounted } from 'vue'
import { NTag } from 'naive-ui'
import { get } from '../../api/client'
import { statusTagType, zhStatus } from '../../utils/status'
const rows=ref([]), scopes=ref({cardTypes:[]}), summary=ref({})
const filter=reactive({cardTypeId:null,timeField:'created_at',startDate:null,endDate:null,includeDisabled:false})
const typeOptions=computed(()=>scopes.value.cardTypes.map(t=>({label:`${t.app_name}-${t.name}`,value:t.id})))
const timeFieldOptions=[{label:'生成时间',value:'created_at'},{label:'激活时间',value:'used_at'}]
function money(v){ return Number(v||0).toFixed(2) }
function durationLabel(hours){ const h=Number(hours||0); if(!h) return '-'; if(h%24===0) return `${h} 小时（${h/24} 天）`; return `${h} 小时` }
function qs(){ const p=new URLSearchParams(); if(filter.cardTypeId)p.set('card_type_id',filter.cardTypeId); if(filter.timeField)p.set('time_field',filter.timeField); if(filter.startDate)p.set('start_date',filter.startDate); if(filter.endDate)p.set('end_date',filter.endDate); if(filter.includeDisabled)p.set('include_disabled','1'); return p.toString() }
const cols=[{title:'卡密',key:'card_key'},{title:'软件',key:'app_name'},{title:'卡类',key:'card_type_name'},{title:'状态',key:'status',render:r=>h(NTag,{type:statusTagType(r.status)},()=>zhStatus(r.status))},{title:'时长',key:'expire_hours',render:r=>durationLabel(r.expire_hours || Number(r.expire_days||0)*24)},{title:'解绑规则',key:'unbind_rule',render:r=>`${r.unbind_used||0}/${r.max_unbinds||0}（免费${r.free_unbinds||0}，扣${r.unbind_deduct_hours||0}h）`},{title:'单张成本',key:'cost',render:r=>'￥'+money(r.cost)},{title:'使用者',key:'used_by'},{title:'生成时间',key:'created_at'},{title:'激活时间',key:'used_at'}]
async function load(){ const r=await get('/api/agent/sales?'+qs()); rows.value=r.data.recent || []; summary.value=r.data.summary || {} }
async function init(){ scopes.value=(await get('/api/agent/scopes')).data; await load() }
onMounted(init)
</script>
