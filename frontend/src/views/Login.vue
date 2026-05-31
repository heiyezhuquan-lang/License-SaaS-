<template>
  <div class="login-page">
    <div class="login-bg login-bg-one"></div>
    <div class="login-bg login-bg-two"></div>
    <main class="login-layout">
      <section class="login-brand-panel">
        <div class="login-logo-row">
          <span class="login-logo">LS</span>
          <div>
            <b>License SaaS</b>
            <small>授权商业控制台</small>
          </div>
        </div>
        <h1>软件授权业务<br />统一管理平台</h1>
        <p>集成软件配置、卡类套餐、卡密激活、代理发卡、设备绑定与客户端安全验证，适合本地 SQLite 单机部署。</p>
        <div class="login-feature-grid">
          <div><b>账号模式</b><span>注册必须卡密激活，到期后可续费</span></div>
          <div><b>卡密模式</b><span>单独卡密登录，首次绑定机器码</span></div>
          <div><b>代理体系</b><span>手动加扣余额，按软件授权发卡</span></div>
          <div><b>客户端安全</b><span>HMAC 签名 + Nonce 防重放</span></div>
        </div>
      </section>

      <n-card v-if="checking" class="login-card" :bordered="false">
        <div class="brand">SYSTEM CHECK</div>
        <h2>正在检查系统状态</h2>
        <p class="login-subtitle">请稍候，正在确认是否需要初始化管理员账户。</p>
        <n-spin size="large" />
      </n-card>

      <n-card v-else-if="needsSetup" class="login-card setup-card" :bordered="false">
        <div class="brand">FIRST SETUP</div>
        <h2>首次启动安装向导</h2>
        <p class="login-subtitle">当前数据库还没有管理员账户，请先设置后台管理员账号和密码。</p>
        <n-form class="login-form" @submit.prevent="submitSetup">
          <n-form-item label="管理员账号">
            <n-input v-model:value="setupForm.username" size="large" placeholder="请输入管理员账号" />
          </n-form-item>
          <n-form-item label="管理员密码">
            <n-input v-model:value="setupForm.password" size="large" type="password" show-password-on="click" placeholder="至少 6 位密码" />
          </n-form-item>
          <n-form-item label="确认密码">
            <n-input v-model:value="setupForm.confirm" size="large" type="password" show-password-on="click" placeholder="再次输入密码" />
          </n-form-item>
          <n-button type="primary" block size="large" class="login-submit" :loading="loading" @click="submitSetup">创建管理员并进入后台</n-button>
        </n-form>
        <div class="login-safe-note setup-note">
          <span>安全初始化</span>
          <em>系统不再内置默认管理员密码，请妥善保存你设置的账户信息。</em>
        </div>
      </n-card>

      <n-card v-else class="login-card" :bordered="false">
        <div class="brand">{{ mode==='admin' ? 'ADMIN CONSOLE' : 'AGENT PORTAL' }}</div>
        <h2>{{ mode==='admin' ? '登录授权控制台' : '登录代理工作台' }}</h2>
        <p class="login-subtitle">{{ mode==='admin' ? '管理软件、用户、卡密、代理与客户端验证接口' : '代理发卡、余额扣款、销售记录独立后台' }}</p>
        <n-tabs v-model:value="mode" type="segment" animated class="login-tabs">
          <n-tab name="admin">管理员</n-tab>
          <n-tab name="agent">代理商</n-tab>
        </n-tabs>
        <n-form class="login-form" @submit.prevent="submit">
          <n-form-item :label="mode==='admin' ? '管理员账号' : '代理账号'">
            <n-input v-model:value="form.username" size="large" :placeholder="mode==='admin' ? '请输入管理员账号' : '请输入代理账号'" />
          </n-form-item>
          <n-form-item label="密码">
            <n-input v-model:value="form.password" size="large" type="password" show-password-on="click" placeholder="请输入密码" />
          </n-form-item>
          <n-button type="primary" block size="large" class="login-submit" :loading="loading" @click="submit">
            {{ mode==='admin' ? '登录后台' : '登录代理端' }}
          </n-button>
        </n-form>
        <div class="login-safe-note">
          <span>安全提示</span>
          <em>请使用安装向导中设置的管理员账户登录，勿在客户端或公开位置暴露后台密码。</em>
        </div>
      </n-card>
    </main>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { get, post } from '../api/client'
const emit = defineEmits(['login'])
const msg = useMessage()
const loading = ref(false)
const checking = ref(true)
const needsSetup = ref(false)
const mode = ref('admin')
const form = reactive({ username: '', password: '' })
const setupForm = reactive({ username: '', password: '', confirm: '' })
watch(mode, ()=>{ form.username=''; form.password='' })
onMounted(loadSetupStatus)
async function loadSetupStatus(){
  checking.value = true
  try{
    const r = await get('/api/setup/status')
    needsSetup.value = !r.installed
  }catch(e){
    msg.error(e.response?.data?.message || '安装状态检查失败')
  }finally{
    checking.value = false
  }
}
function saveLogin(token, role){
  if(role==='admin') localStorage.setItem('admin_token', token); else localStorage.setItem('agent_token', token)
  localStorage.setItem('login_role', role)
  emit('login', { token, role })
}
async function submitSetup(){
  if(!setupForm.username.trim()) return msg.error('请输入管理员账号')
  if(setupForm.password.length < 6) return msg.error('管理员密码至少 6 位')
  if(setupForm.password !== setupForm.confirm) return msg.error('两次输入的密码不一致')
  loading.value=true
  try{
    const r = await post('/api/setup/admin', { username: setupForm.username.trim(), password: setupForm.password })
    msg.success('管理员创建成功')
    needsSetup.value = false
    saveLogin(r.token, 'admin')
  }catch(e){
    msg.error(e.response?.data?.message || '初始化失败')
  }finally{ loading.value=false }
}
async function submit(){
  loading.value=true
  try{
    const path = mode.value==='admin' ? '/api/admin/login' : '/api/agent/login'
    const r=await post(path, form)
    saveLogin(r.token, mode.value)
    msg.success('登录成功')
  }
  catch(e){ msg.error(e.response?.data?.message || '登录失败') }
  finally{ loading.value=false }
}
</script>
