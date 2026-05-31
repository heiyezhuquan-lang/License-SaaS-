<template>
  <n-card>
    <template #header>
      <div class="card-head">
        <div>
          <b>云变量 / 云配置</b>
          <div class="muted">按软件隔离的客户端文本配置，客户端通过签名接口读取</div>
        </div>
        <n-button type="primary" @click="openCreate">新增变量</n-button>
      </div>
    </template>
    <n-space class="toolbar" wrap>
      <n-select v-model:value="filter.appId" :options="appOptions" clearable placeholder="软件" style="width:180px" @update:value="load" />
      <n-select v-model:value="filter.status" :options="statusOptions" clearable placeholder="状态" style="width:140px" @update:value="load" />
      <n-input v-model:value="filter.keyword" placeholder="搜索变量名/值/备注" clearable style="width:240px" @keyup.enter="load" />
      <n-button type="primary" @click="load">筛选</n-button>
    </n-space>
    <n-data-table :columns="cols" :data="rows" :pagination="{ pageSize: 10 }" />
  </n-card>

  <n-modal v-model:show="show" preset="card" :title="editing ? '编辑云变量' : '新增云变量'" class="modal">
    <n-form label-placement="top">
      <n-form-item label="软件"><n-select v-model:value="form.appId" :options="appOptions" :disabled="!!editing" /></n-form-item>
      <n-form-item label="变量名"><n-input v-model:value="form.varKey" placeholder="例如 notice_text / server_url / feature_text" /></n-form-item>
      <n-form-item label="变量值"><n-input v-model:value="form.varValue" type="textarea" :autosize="{ minRows: 3 }" placeholder="文本内容" /></n-form-item>
      <n-form-item label="状态"><n-select v-model:value="form.status" :options="statusOptions" /></n-form-item>
      <n-form-item label="备注"><n-input v-model:value="form.remark" /></n-form-item>
      <n-alert type="info" style="margin-bottom:12px">云变量统一按文本返回；客户端读取地址：/api/client/cloud-vars?app_key=软件AppKey，必须带客户端签名 Header。</n-alert>
      <n-space justify="end"><n-button @click="show=false">取消</n-button><n-button type="primary" @click="save">保存</n-button></n-space>
    </n-form>
  </n-modal>
</template>
<script setup>
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NButton, NTag, useDialog, useMessage } from 'naive-ui'
import { api, get, post, put } from '../../api/client'

const msg = useMessage()
const dialog = useDialog()
const rows = ref([])
const apps = ref([])
const show = ref(false)
const editing = ref(null)
const filter = reactive({ appId: null, status: null, keyword: '' })
const form = reactive({ appId: null, varKey: '', varValue: '', status: 'active', remark: '' })
const statusOptions = [{ label: '启用', value: 'active' }, { label: '停用', value: 'disabled' }]
const appOptions = computed(() => apps.value.map(x => ({ label: x.name, value: x.id })))
const cols = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '软件', key: 'app_name' },
  { title: '变量名', key: 'var_key' },
  { title: '变量值', key: 'var_value', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', render: r => h(NTag, { type: r.status === 'active' ? 'success' : 'error' }, () => r.status === 'active' ? '启用' : '停用') },
  { title: '备注', key: 'remark', ellipsis: { tooltip: true } },
  { title: '更新时间', key: 'updated_at' },
  { title: '操作', key: 'actions', render: r => h('div', { class: 'actions' }, [h(NButton, { size: 'small', onClick: () => openEdit(r) }, () => '编辑'), h(NButton, { size: 'small', type: 'error', tertiary: true, onClick: () => remove(r) }, () => '删除')]) }
]
function query(){ const p = new URLSearchParams(); if (filter.appId) p.set('app_id', filter.appId); if (filter.status) p.set('status', filter.status); if (filter.keyword) p.set('keyword', filter.keyword); return p.toString() }
async function load(){ rows.value = (await get('/api/admin/cloud-vars?' + query())).data }
async function loadApps(){ apps.value = (await get('/api/admin/apps')).data; if (!form.appId && apps.value[0]) form.appId = apps.value[0].id }
function resetForm(){ Object.assign(form, { appId: apps.value[0]?.id || null, varKey: '', varValue: '', status: 'active', remark: '' }) }
function openCreate(){ editing.value = null; resetForm(); show.value = true }
function openEdit(r){ editing.value = r; Object.assign(form, { appId: r.app_id, varKey: r.var_key, varValue: r.var_value, status: r.status || 'active', remark: r.remark || '' }); show.value = true }
function payload(){ return { appId: form.appId, varKey: form.varKey, varValue: form.varValue, valueType: 'text', status: form.status, remark: form.remark } }
async function save(){ try { if (editing.value) await put(`/api/admin/cloud-vars/${editing.value.id}`, payload()); else await post('/api/admin/cloud-vars', payload()); msg.success('已保存'); show.value = false; await load() } catch(e){ msg.error(e.response?.data?.message || '保存失败') } }
function remove(r){ dialog.warning({ title: '确认删除？', content: `删除变量 ${r.var_key} 后客户端将不再收到该配置。`, positiveText: '删除', negativeText: '取消', onPositiveClick: async () => { await api.delete(`/api/admin/cloud-vars/${r.id}`); msg.success('已删除'); await load() } }) }
onMounted(async () => { await loadApps(); await load() })
</script>
<style scoped>
.card-head{display:flex;align-items:center;justify-content:space-between;gap:12px}.muted{font-size:12px;color:#64748b;margin-top:4px}.toolbar{margin-bottom:16px}.modal{max-width:720px}:deep(.actions){display:flex;gap:8px}
</style>
