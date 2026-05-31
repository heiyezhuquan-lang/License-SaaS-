<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="logo"><span>AG</span><div><b>代理工作台</b><small>发卡与余额中心</small></div></div>
      <button v-for="m in menus" :key="m.key" :class="['nav', active===m.key && 'on']" @click="active=m.key">{{ m.label }}</button>
      <button class="nav logout" @click="$emit('logout')">退出登录</button>
    </aside>
    <main class="main">
      <header class="top"><div><h2>{{ current.label }}</h2><p>{{ current.desc }}</p></div><n-tag type="info">代理端</n-tag></header>
      <AgentHome v-if="active==='home'" />
      <AgentIssue v-if="active==='issue'" />
      <AgentCards v-if="active==='cards'" />
      <AgentStats v-if="active==='stats'" />
    </main>
  </div>
</template>
<script setup>
import { computed, ref } from 'vue'
import AgentHome from './agent/AgentHome.vue'
import AgentIssue from './agent/AgentIssue.vue'
import AgentCards from './agent/AgentCards.vue'
import AgentStats from './agent/AgentStats.vue'
const active=ref('home')
const menus=[{key:'home',label:'代理首页',desc:'余额、权限和销售概览'},{key:'issue',label:'生成卡密',desc:'按授权范围发卡并自动扣余额'},{key:'cards',label:'我的卡密',desc:'查看自己生成的卡密'},{key:'stats',label:'销售统计',desc:'发卡数量、扣款汇总和余额流水'}]
const current=computed(()=>menus.find(x=>x.key===active.value)||menus[0])
</script>
