import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

const routes = [
  { path: '/', redirect: '/home' },
  {
    path: '/home',
    name: 'home',
    component: () => import('@/views/member/Home.vue'),
    meta: { title: '面试预约' }
  },
  {
    path: '/my',
    name: 'my',
    component: () => import('@/views/member/MySchedule.vue'),
    meta: { title: '我的心仪时间' }
  },
  {
    path: '/bind',
    name: 'bind',
    component: () => import('@/views/member/BindDirection.vue'),
    meta: { title: '绑定方向', public: true }
  },
  {
    path: '/admin',
    name: 'admin-dashboard',
    component: () => import('@/views/admin/Dashboard.vue'),
    meta: { title: '管理后台', admin: true }
  },
  {
    path: '/admin/session/new',
    name: 'admin-session-new',
    component: () => import('@/views/admin/SessionEdit.vue'),
    meta: { title: '新建面试场次', admin: true }
  },
  {
    path: '/admin/session/:id/edit',
    name: 'admin-session-edit',
    component: () => import('@/views/admin/SessionEdit.vue'),
    meta: { title: '编辑面试场次', admin: true }
  },
  {
    path: '/admin/session/:id/bookings',
    name: 'admin-session-bookings',
    component: () => import('@/views/admin/Bookings.vue'),
    meta: { title: '预约管理', admin: true }
  },
  {
    path: '/admin/session/:id/append',
    name: 'admin-session-append',
    component: () => import('@/views/admin/SessionAppend.vue'),
    meta: { title: '追加日程', admin: true }
  },
  {
    path: '/admin/session/:id/assign',
    name: 'admin-assignment',
    component: () => import('@/views/admin/Assignment.vue'),
    meta: { title: '分配管理', admin: true }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

// 登录态校验：无 token 时由 App.vue 展示登录引导；管理员路由二次校验；
// 成员进入预约相关页面前必须已绑定方向
router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()
  const token = localStorage.getItem('token')
  if (!token) {
    // 未登录：仅允许停留在引导态（App 负责渲染 LoginGate）
    return next()
  }
  if (!userStore.loaded) {
    await userStore.loadMe()
  }
  if (to.meta.admin) {
    if (!userStore.isAdmin) {
      ElMessage.error('无管理员权限')
      return next('/home')
    }
    return next()
  }
  if (!to.meta.public && !userStore.isAdmin && !userStore.directions.length) {
    return next('/bind')
  }
  next()
})

export default router
