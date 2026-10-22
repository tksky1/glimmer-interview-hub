import { defineStore } from 'pinia'
import { ref } from 'vue'
import http from '@/utils/http'

// 管理用户登录态与身份信息
export const useUserStore = defineStore('user', () => {
  // token 由旧系统登录后写入 localStorage，此处保持响应式镜像
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref(null)
  const isAdmin = ref(false)
  const direction = ref('') // 首个方向（兼容旧逻辑）
  const directions = ref([]) // 已获邀请的全部方向
  const studentid = ref('')
  const name = ref('')
  const qq = ref('')
  const phone = ref('')
  const loaded = ref(false)

  // 外部（其他标签页 / 401 拦截）修改 localStorage 后同步
  function syncToken() {
    token.value = localStorage.getItem('token') || ''
  }

  // 拉取当前用户信息
  async function loadMe() {
    if (!token.value) {
      userInfo.value = null
      isAdmin.value = false
      direction.value = ''
      directions.value = []
      studentid.value = ''
      name.value = ''
      qq.value = ''
      phone.value = ''
      loaded.value = true
      return null
    }
    try {
      const res = await http.get('/me')
      userInfo.value = res.data
      isAdmin.value = !!res.data?.isAdmin
      const list = Array.isArray(res.data?.directions) ? [...res.data.directions] : []
      if (!list.length && res.data?.direction) list.push(res.data.direction)
      directions.value = list
      direction.value = res.data?.direction || list[0] || ''
      studentid.value = res.data?.studentid || ''
      name.value = res.data?.name || ''
      qq.value = res.data?.qq || ''
      phone.value = res.data?.phone || ''
    } catch (e) {
      userInfo.value = null
      isAdmin.value = false
      direction.value = ''
      directions.value = []
      studentid.value = ''
      name.value = ''
      qq.value = ''
      phone.value = ''
    }
    loaded.value = true
    return userInfo.value
  }

  // 绑定方向成功后刷新身份
  async function refreshDirection() {
    await loadMe()
  }

  function logout() {
    localStorage.removeItem('token')
    token.value = ''
    userInfo.value = null
    isAdmin.value = false
    direction.value = ''
    directions.value = []
    studentid.value = ''
    name.value = ''
    qq.value = ''
    phone.value = ''
  }

  return {
    token,
    userInfo,
    isAdmin,
    direction,
    directions,
    studentid,
    name,
    qq,
    phone,
    loaded,
    syncToken,
    loadMe,
    refreshDirection,
    logout
  }
})
