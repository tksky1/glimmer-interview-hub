import axios from 'axios'
import { API_BASE } from '@/config'
import { alertDialog } from '@/utils/dialog'

// 创建 axios 实例（与旧前端一致的 token 头机制）
const http = axios.create({
  baseURL: API_BASE,
  timeout: 15000
})

// 请求拦截：注入旧系统下发的 JWT
http.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers['token'] = token
    }
    return config
  },
  e => Promise.reject(e)
)

// 错误弹窗：避免提示出现在页面边缘被忽略
function showError(title, msg) {
  alertDialog(msg, title, { type: 'error', confirmButtonText: '知道了', closeOnClickModal: false }).catch(() => {})
}

// 响应拦截：统一取出 body；错误一律弹窗提示
http.interceptors.response.use(
  res => res.data,
  err => {
    const status = err.response?.status
    const msg = err.response?.data?.msg || '网络请求失败，请稍后重试'
    if (status === 401) {
      localStorage.removeItem('token')
      window.dispatchEvent(new Event('glimmer-token-expired'))
      showError('登录状态失效', '登录状态已失效，请重新登录')
    } else if (status === 403) {
      showError('没有权限', msg)
    } else if (status !== undefined) {
      showError('出错了', msg)
    } else {
      showError('网络异常', '网络异常，请检查网络连接')
    }
    return Promise.reject(err)
  }
)

export default http
