<template><div><div class="hero"><h1>代理首页</h1><p>只展示自己的余额、可发软件和可发套餐，不能管理全局软件。</p></div><n-grid :cols="3" :x-gap="16"><n-gi><n-card class="metric"><small>当前余额</small><b>￥{{ me.balance || 0 }}</b></n-card></n-gi><n-gi><n-card class="metric"><small>可发软件</small><b>{{ scopes.apps?.length || 0 }}</b></n-card></n-gi><n-gi><n-card class="metric"><small>可发套餐</small><b>{{ scopes.cardTypes?.length || 0 }}</b></n-card></n-gi></n-grid><n-card title="权限范围" style="margin-top:16px"><n-space><n-tag v-for="a in scopes.apps" :key="a.id" type="success">{{ a.name }}</n-tag><n-tag v-for="t in scopes.cardTypes" :key="t.id" type="info">{{ t.app_name }} / {{ t.name }} / ￥{{ t.price }}</n-tag></n-space></n-card></div></template>
<script setup>
import { ref,onMounted } from 'vue'
import { get } from '../../api/client'
const me=ref({}); const scopes=ref({apps:[],cardTypes:[]})
async function load(){ const m=(await get('/api/agent/me')).data; me.value=m[0]||{}; scopes.value=(await get('/api/agent/scopes')).data }
onMounted(load)
</script>
