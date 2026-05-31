<template>
  <n-config-provider>
    <n-message-provider>
      <n-dialog-provider>
        <div v-if="checking" class="boot-check">正在校验登录状态...</div>
        <Login v-else-if="!token" @login="onLogin" />
        <AgentConsole v-else-if="role==='agent'" @logout="logout" />
        <Console v-else @logout="logout" />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import Login from './views/Login.vue'
import Console from './views/Console.vue'
import AgentConsole from './views/AgentConsole.vue'
import { get, clearAuthCache } from './api/client'

const token = ref('')
const role = ref('admin')
const checking = ref(true)

function readStoredAuth(){
  const storedRole = localStorage.getItem('login_role') || (localStorage.getItem('agent_token') ? 'agent' : 'admin')
  const storedToken = storedRole === 'agent' ? localStorage.getItem('agent_token') : localStorage.getItem('admin_token')
  return { storedRole, storedToken }
}
async function validateStoredAuth(){
  const { storedRole, storedToken } = readStoredAuth()
  role.value = storedRole
  if(!storedToken){
    token.value = ''
    checking.value = false
    return
  }
  try{
    if(storedRole === 'agent') await get('/api/agent/me')
    else await get('/api/admin/stats')
    token.value = storedToken
  }catch(e){
    clearAuthCache()
    token.value = ''
  }finally{
    checking.value = false
  }
}
function onLogin(payload){ token.value=payload.token; role.value=payload.role }
function logout(){ clearAuthCache(); token.value='' }
function onAuthExpired(){ token.value=''; checking.value=false }
onMounted(()=>{
  window.addEventListener('auth-expired', onAuthExpired)
  validateStoredAuth()
})
onUnmounted(()=>window.removeEventListener('auth-expired', onAuthExpired))
</script>
<style>
.boot-check{min-height:100vh;display:grid;place-items:center;color:#64748b;background:#f6f8fc;font-size:15px}
</style>
