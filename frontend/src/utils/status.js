export const statusText = {
  active: '正常',
  disabled: '禁用',
  banned: '封禁',
  unused: '未使用',
  used: '已使用',
  new: '未使用',
  offline: '离线',
  online: '在线',
  pending: '待处理',
  success: '成功',
  failed: '失败',
  error: '异常'
}

export function zhStatus(status){
  return statusText[status] || status || '-'
}

export function statusTagType(status){
  if (status === 'active' || status === 'unused' || status === 'new' || status === 'online' || status === 'success') return 'success'
  if (status === 'used' || status === 'pending') return 'info'
  if (status === 'banned' || status === 'disabled' || status === 'failed' || status === 'error') return 'error'
  return 'default'
}
