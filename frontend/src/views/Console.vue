<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="logo"><span>LS</span><div><b>License SaaS</b><small>授权商业控制台</small></div></div>
      <button v-for="m in menus" :key="m.key" :class="['nav', active===m.key && 'on']" @click="active=m.key">{{ m.label }}</button>
      <button class="nav logout" @click="$emit('logout')">退出登录</button>
    </aside>
    <main class="main">
      <header class="top">
        <div><h2>{{ current.label }}</h2><p>{{ current.desc }}</p></div>
        <a class="qq-group-popover" href="https://qm.qq.com/cgi-bin/qm/qr?k=0xLW5-LVTvKz25J5ywDcN_D2RB1XQJaT&jump_from=webapi&authKey=anNucgynP0bOqSJa/gUj4wVyymlxoliy7b0MkPmqQstabml7exC7uCYNMiBzN72E" target="_blank" rel="noopener noreferrer" title="点击加入QQ群465663266">
          <n-tag type="success" round>QQ群465663266</n-tag>
          <div class="qq-qrcode-card">
            <img src="/qq-group-465663266.png" alt="QQ群465663266二维码" />
            <b>QQ群465663266</b>
            <small>点击或扫码加入交流群</small>
          </div>
        </a>
      </header>
      <Dashboard v-if="active==='dashboard'" @go="active=$event" />
      <Apps v-if="active==='apps'" />
      <Users v-if="active==='users'" />
      <CardTypes v-if="active==='types'" />
      <Cards v-if="active==='cards'" />
      <Sales v-if="active==='sales'" />
      <Agents v-if="active==='agents'" />
      <Devices v-if="active==='devices'" />
      <CloudVars v-if="active==='cloud'" />
      <ClientApi v-if="active==='client'" />
    </main>
  </div>
</template>
<script setup>
import { computed, ref } from 'vue'
import Dashboard from './dashboard/Dashboard.vue'
import Apps from './dashboard/Apps.vue'
import Users from './dashboard/Users.vue'
import CardTypes from './dashboard/CardTypes.vue'
import Cards from './dashboard/Cards.vue'
import Sales from './dashboard/Sales.vue'
import Agents from './dashboard/Agents.vue'
import Devices from './dashboard/Devices.vue'
import CloudVars from './dashboard/CloudVars.vue'
import ClientApi from './dashboard/ClientApi.vue'
const active = ref('dashboard')
const menus = [
  {key:'dashboard', label:'控制中心', desc:'整体运营数据与状态'},
  {key:'apps', label:'软件管理', desc:'多软件、公告、版本、强更'},
  {key:'users', label:'用户管理', desc:'客户账号、到期与机器码'},
  {key:'types', label:'卡类套餐', desc:'月卡、季卡、年卡等套餐'},
  {key:'cards', label:'卡密管理', desc:'生成、查询、禁用卡密'},
  {key:'sales', label:'销量查询', desc:'按日期、代理、卡类统计销量'},
  {key:'agents', label:'代理管理', desc:'代理账号、余额、发卡权限'},
  {key:'devices', label:'设备管理', desc:'机器码、在线心跳、封禁设备'},
  {key:'cloud', label:'云变量', desc:'客户端动态配置、开关和参数'},
  {key:'client', label:'客户端接口', desc:'注册、登录、充值、心跳 API'},
]
const current = computed(()=>menus.find(x=>x.key===active.value) || menus[0])
</script>
