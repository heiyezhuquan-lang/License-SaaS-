<template>
  <div class="dashboard-page">
    <section class="dash-hero">
      <div>
        <span class="hero-pill">License SaaS · 运营总览</span>
        <h1>授权业务控制中心</h1>
        <p>集中查看软件、用户、卡密、设备、代理和今日激活情况，快速发现库存、到期和销售状态。</p>
      </div>
      <div class="hero-score">
        <small>今日激活</small>
        <b>{{ n('today_activations') }}</b>
        <span>今日发卡 {{ n('today_cards') }} 张</span>
      </div>
    </section>

    <n-grid :cols="4" :x-gap="16" :y-gap="16" responsive="screen">
      <n-gi v-for="m in metrics" :key="m.key">
        <n-card class="metric pro-metric">
          <div class="metric-top">
            <span :class="['metric-icon', m.tone]">{{ m.icon }}</span>
            <small>{{ m.label }}</small>
          </div>
          <b>{{ m.money ? money(n(m.key)) : n(m.key) }}</b>
          <p>{{ m.sub }}</p>
        </n-card>
      </n-gi>
    </n-grid>

    <n-grid :cols="3" :x-gap="16" :y-gap="16" responsive="screen" style="margin-top:16px">
      <n-gi :span="2">
        <n-card class="panel-card" title="卡密运营状态">
          <div class="status-bars">
            <div v-for="item in cardStatus" :key="item.key" class="bar-row">
              <div class="bar-label"><span>{{ item.label }}</span><b>{{ item.value }}</b></div>
              <div class="bar-track"><i :class="item.tone" :style="{width:item.percent+'%'}"></i></div>
            </div>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="panel-card" title="账号与设备风险">
          <div class="risk-list">
            <div><span>7天内到期账号</span><b>{{ n('expiring_users') }}</b></div>
            <div><span>已过期账号</span><b>{{ n('expired_users') }}</b></div>
            <div><span>活跃设备</span><b>{{ n('active_devices') }}</b></div>
            <div><span>启用代理</span><b>{{ n('active_agents') }}/{{ n('agents') }}</b></div>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-grid :cols="3" :x-gap="16" :y-gap="16" responsive="screen" style="margin-top:16px">
      <n-gi :span="2">
        <n-card class="panel-card" title="软件概览">
          <n-data-table :columns="appCols" :data="stats.app_overview || []" :pagination="false" size="small" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="panel-card" title="快捷入口">
          <div class="quick-grid">
            <button @click="emit('go','apps')">软件配置<span>版本/公告/密钥</span></button>
            <button @click="emit('go','cards')">生成卡密<span>发卡与导出</span></button>
            <button @click="emit('go','agents')">代理管理<span>余额与权限</span></button>
            <button @click="emit('go','client-api')">客户端接口<span>签名与接入</span></button>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-grid :cols="3" :x-gap="16" :y-gap="16" responsive="screen" style="margin-top:16px">
      <n-gi :span="2">
        <n-card class="panel-card" title="最近卡密">
          <n-data-table :columns="cardCols" :data="stats.recent_cards || []" :pagination="false" size="small" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="panel-card" title="运营提示">
          <div class="tips">
            <p><b>账号模式</b><span>注册必须填写卡密，到期后可用卡密续费。</span></p>
            <p><b>卡密模式</b><span>支持单独卡密登录，首次登录绑定机器码。</span></p>
            <p><b>代理风控</b><span>禁用代理后，该代理未使用卡密无法继续激活。</span></p>
          </div>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>
<script setup>
import { computed, h, onMounted, ref } from 'vue'
import { NTag } from 'naive-ui'
import { get } from '../../api/client'
import { zhStatus, statusTagType } from '../../utils/status'

const emit = defineEmits(['go'])
const stats = ref({})
const n = key => Number(stats.value?.[key] || 0)
const money = v => '￥' + Number(v || 0).toFixed(2)
const pct = (v, total) => total ? Math.max(4, Math.round(v * 100 / total)) : 0

const metrics = computed(() => [
  {key:'apps', label:'软件数', icon:'软', tone:'blue', sub:`启用 ${n('active_apps')} 个软件`},
  {key:'users', label:'用户数', icon:'用', tone:'green', sub:`正常 ${n('active_users')} · 到期 ${n('expired_users')}`},
  {key:'cards', label:'卡密总数', icon:'卡', tone:'purple', sub:`未使用 ${n('unused_cards')} · 已使用 ${n('used_cards')}`},
  {key:'revenue', label:'卡密金额', icon:'￥', tone:'orange', money:true, sub:`代理发卡成本 ${money(n('agent_cost'))}`},
  {key:'devices', label:'设备数', icon:'设', tone:'cyan', sub:`活跃设备 ${n('active_devices')}`},
  {key:'agents', label:'代理数', icon:'代', tone:'blue', sub:`启用代理 ${n('active_agents')}`},
  {key:'cloud_vars', label:'云变量', icon:'云', tone:'green', sub:'客户端运行配置'},
  {key:'today_cards', label:'今日发卡', icon:'今', tone:'orange', sub:`今日激活 ${n('today_activations')}`},
])

const cardStatus = computed(() => {
  const total = n('cards')
  return [
    {key:'unused', label:'未使用卡密', value:n('unused_cards'), percent:pct(n('unused_cards'), total), tone:'ok'},
    {key:'used', label:'已使用卡密', value:n('used_cards'), percent:pct(n('used_cards'), total), tone:'info'},
    {key:'disabled', label:'已禁用卡密', value:n('disabled_cards'), percent:pct(n('disabled_cards'), total), tone:'danger'},
  ]
})

const appCols = [
  {title:'软件', key:'name'},
  {title:'App Key', key:'app_key'},
  {title:'状态', key:'status', render:r=>h(NTag,{type:statusTagType(r.status),round:true},()=>zhStatus(r.status))},
  {title:'用户', key:'users'},
  {title:'卡密', key:'cards'},
]
const cardCols = [
  {title:'卡密', key:'card_key'},
  {title:'软件', key:'app_name'},
  {title:'状态', key:'status', render:r=>h(NTag,{type:statusTagType(r.status),round:true},()=>zhStatus(r.status))},
  {title:'使用者', key:'used_by', render:r=>r.used_by || '-'},
  {title:'时间', key:'created_at'},
]

onMounted(async()=>{ stats.value=(await get('/api/admin/stats')).data || {} })
</script>
