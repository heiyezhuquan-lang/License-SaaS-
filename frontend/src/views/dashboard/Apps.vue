<template>
  <n-space vertical size="large">
    <n-card>
      <template #header>
        <div class="card-head">
          <div>
            <b>软件管理 / 软件配置</b>
            <div class="muted">每个软件独立管理公告、整数版本、强更、下载地址、心跳策略和客户端签名密钥</div>
          </div>
          <n-button type="primary" @click="openCreate">新增软件</n-button>
        </div>
      </template>
      <n-data-table :columns="cols" :data="rows" :pagination="{ pageSize: 10 }" />
    </n-card>

    <n-modal v-model:show="show" preset="card" :title="editing ? '编辑软件配置' : '新增软件'" class="modal">
      <n-form label-placement="left" label-width="120">
        <n-grid :cols="2" :x-gap="16">
          <n-form-item-gi label="软件名称"><n-input v-model:value="form.name" /></n-form-item-gi>
          <n-form-item-gi label="App Key"><n-input v-model:value="form.appKey" :disabled="!!editing" /></n-form-item-gi>
          <n-form-item-gi label="状态"><n-select v-model:value="form.status" :options="statusOptions" /></n-form-item-gi>
          <n-form-item-gi label="当前版本号"><n-input-number v-model:value="form.version" :min="1" :precision="0" placeholder="例如 100" style="width:100%" /></n-form-item-gi>
          <n-form-item-gi label="最低版本号"><n-input-number v-model:value="form.minVersion" :min="0" :precision="0" placeholder="低于此整数版本提示强更" style="width:100%" /></n-form-item-gi>
          <n-form-item-gi label="强制更新"><n-switch v-model:value="form.forceUpdateBool" /></n-form-item-gi>
          <n-form-item-gi label="心跳间隔秒"><n-input-number v-model:value="form.heartbeatInterval" :min="10" /></n-form-item-gi>
          <n-form-item-gi label="心跳超时秒"><n-input-number v-model:value="form.heartbeatTimeout" :min="10" /></n-form-item-gi>
        </n-grid>
        <n-form-item label="主下载地址"><n-input v-model:value="form.downloadURL" placeholder="https://..." /></n-form-item>
        <n-form-item label="备用下载地址"><n-input v-model:value="form.backupDownloadURL" placeholder="https://..." /></n-form-item>
        <n-form-item label="强更提示"><n-input v-model:value="form.forceUpdateMessage" type="textarea" placeholder="请下载最新版本后继续使用" /></n-form-item>
        <n-form-item label="公告"><n-input v-model:value="form.announcement" type="textarea" :autosize="{ minRows: 3 }" /></n-form-item>
        <n-alert v-if="editing" type="warning" style="margin-bottom: 12px">重置客户端密钥后，旧客户端签名会立刻失效，需要同步更新客户端配置。</n-alert>
        <n-space justify="space-between">
          <n-button v-if="editing" tertiary type="warning" @click="resetSecret">重置客户端密钥</n-button>
          <n-space>
            <n-button @click="show=false">取消</n-button>
            <n-button type="primary" @click="save">保存</n-button>
          </n-space>
        </n-space>
      </n-form>
    </n-modal>
  </n-space>
</template>
<script setup>
import { h, onMounted, reactive, ref } from 'vue'
import { NButton, NTag, useDialog, useMessage } from 'naive-ui'
import { get, post, put } from '../../api/client'

const msg = useMessage()
const dialog = useDialog()
const rows = ref([])
const show = ref(false)
const editing = ref(null)
const statusOptions = [{ label: '启用', value: 'active' }, { label: '停用', value: 'disabled' }]
const form = reactive({ name: '', appKey: '', status: 'active', version: 1, minVersion: 0, forceUpdateBool: false, forceUpdateMessage: '', downloadURL: '', backupDownloadURL: '', announcement: '', heartbeatInterval: 60, heartbeatTimeout: 180 })
const cols = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '名称', key: 'name' },
  { title: 'App Key', key: 'app_key' },
  { title: '状态', key: 'status', render: r => h(NTag, { type: r.status === 'active' ? 'success' : 'error' }, () => r.status === 'active' ? '启用' : '停用') },
  { title: '版本号', key: 'version', render: r => `${num(r.version, 1)} / 最低 ${num(r.min_version, 0)}` },
  { title: '强更', key: 'force_update', render: r => h(NTag, { type: Number(r.force_update) ? 'warning' : 'default' }, () => Number(r.force_update) ? '开启' : '关闭') },
  { title: '心跳', key: 'heartbeat_interval', render: r => `${r.heartbeat_interval || 60}s / 超时 ${r.heartbeat_timeout || 180}s` },
  { title: '密钥', key: 'client_secret', width: 230, render: r => h('div', { class: 'secret-cell' }, [h('code', { class: 'secret-code', title: r.client_secret || '' }, mask(r.client_secret)), h(NButton, { size: 'tiny', tertiary: true, type: 'primary', disabled: !r.client_secret, onClick: () => copySecret(r.client_secret) }, () => '复制')]) },
  { title: '操作', key: 'actions', render: r => h(NButton, { size: 'small', onClick: () => openEdit(r) }, () => '配置') }
]
function num(v, fallback){ const n = Number(v); return Number.isFinite(n) ? Math.trunc(n) : fallback }
function mask(v){ return v ? `${String(v).slice(0,8)}...${String(v).slice(-6)}` : '-' }
function resetForm(){ Object.assign(form, { name: '', appKey: '', status: 'active', version: 1, minVersion: 0, forceUpdateBool: false, forceUpdateMessage: '', downloadURL: '', backupDownloadURL: '', announcement: '', heartbeatInterval: 60, heartbeatTimeout: 180 }) }
function openCreate(){ editing.value = null; resetForm(); show.value = true }
function openEdit(r){ editing.value = r; Object.assign(form, { name: r.name, appKey: r.app_key, status: r.status, version: num(r.version, 1), minVersion: num(r.min_version, 0), forceUpdateBool: Number(r.force_update) === 1, forceUpdateMessage: r.force_update_message || '', downloadURL: r.download_url || '', backupDownloadURL: r.backup_download_url || '', announcement: r.announcement || '', heartbeatInterval: Number(r.heartbeat_interval || 60), heartbeatTimeout: Number(r.heartbeat_timeout || 180) }); show.value = true }
async function load(){ rows.value = (await get('/api/admin/apps')).data }
function payload(){ return { name: form.name, appKey: form.appKey, status: form.status, version: num(form.version, 1), minVersion: num(form.minVersion, 0), forceUpdate: form.forceUpdateBool ? 1 : 0, forceUpdateMessage: form.forceUpdateMessage, downloadURL: form.downloadURL, backupDownloadURL: form.backupDownloadURL, announcement: form.announcement, heartbeatInterval: form.heartbeatInterval, heartbeatTimeout: form.heartbeatTimeout } }
async function save(){ if (editing.value) await put(`/api/admin/apps/${editing.value.id}`, payload()); else await post('/api/admin/apps', payload()); msg.success('已保存'); show.value = false; await load() }
async function copySecret(secret){
  if (!secret) return
  try {
    if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(secret)
    else {
      const ta = document.createElement('textarea')
      ta.value = secret
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      document.body.appendChild(ta)
      ta.focus()
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    msg.success('客户端密钥已复制')
  } catch (e) {
    msg.error('复制失败，请手动选中复制')
  }
}
function resetSecret(){ dialog.warning({ title: '确认重置密钥？', content: '重置后旧客户端签名立即失效。', positiveText: '确认重置', negativeText: '取消', onPositiveClick: async () => { const res = await put(`/api/admin/apps/${editing.value.id}/secret/reset`, {}); await copySecret(res.client_secret); msg.success('新密钥已生成并复制'); await load() } }) }
onMounted(load)
</script>
<style scoped>
.card-head{display:flex;align-items:center;justify-content:space-between;gap:12px}.muted{font-size:12px;color:#64748b;margin-top:4px}.modal{max-width:860px}.secret-cell{display:flex;align-items:center;gap:8px}.secret-code{display:inline-block;max-width:132px;padding:2px 6px;border-radius:6px;background:#f1f5f9;color:#334155;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
</style>
