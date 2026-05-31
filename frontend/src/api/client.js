import axios from 'axios'

export const API_BASE = import.meta.env.VITE_API_BASE || ''
export const api = axios.create({ baseURL: API_BASE })

export function clearAuthCache(){
  localStorage.removeItem('admin_token')
  localStorage.removeItem('agent_token')
  localStorage.removeItem('login_role')
}

api.interceptors.request.use((config) => {
  const role = localStorage.getItem('login_role')
  const token = role === 'agent'
    ? localStorage.getItem('agent_token')
    : role === 'admin'
      ? localStorage.getItem('admin_token')
      : (localStorage.getItem('admin_token') || localStorage.getItem('agent_token'))
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (resp) => resp,
  (error) => {
    const status = error?.response?.status
    const url = error?.config?.url || ''
    if (status === 401 && !url.includes('/api/admin/login') && !url.includes('/api/agent/login')) {
      clearAuthCache()
      window.dispatchEvent(new Event('auth-expired'))
    }
    return Promise.reject(error)
  }
)

export async function get(url){ return (await api.get(url)).data }
export async function post(url, data){ return (await api.post(url, data)).data }
export async function put(url, data){ return (await api.put(url, data)).data }
export async function del(url, data){ return (await api.delete(url, { data })).data }
