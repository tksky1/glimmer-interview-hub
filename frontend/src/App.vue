<script setup>
import { onMounted, onUnmounted, computed } from 'vue'
import AppHeader from '@/components/AppHeader.vue'
import LoginGate from '@/components/LoginGate.vue'
import { useUserStore } from '@/stores/user'
import { redeemPendingInvite } from '@/utils/invite'

const userStore = useUserStore()
const loggedIn = computed(() => !!userStore.token)

// 旧登录页在另一个标签页写入 token 时，storage 事件跨标签页通知
function onStorage(e) {
  if (e.key === 'token' || e.key === null) {
    userStore.syncToken()
    if (userStore.token) userStore.loadMe()
  }
}
// http 拦截器在 401 时派发
function onExpired() {
  userStore.syncToken()
}

onMounted(async () => {
  window.addEventListener('storage', onStorage)
  window.addEventListener('glimmer-token-expired', onExpired)

  await userStore.loadMe()
  // 已登录则自动核销邀请码绑定方向（邀请码由 index.html 前置脚本捕获）
  if (userStore.token) {
    await redeemPendingInvite()
    await userStore.refreshDirection()
  }
})

onUnmounted(() => {
  window.removeEventListener('storage', onStorage)
  window.removeEventListener('glimmer-token-expired', onExpired)
})
</script>

<template>
  <div class="app-root">
    <AppHeader v-if="loggedIn" />
    <div class="app-body" :class="{ 'with-header': loggedIn }">
      <LoginGate v-if="!loggedIn" />
      <router-view v-else v-slot="{ Component }">
        <transition name="page-fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>
  </div>
</template>

<style>
html,
body {
  margin: 0;
  padding: 0;
  height: 100%;
}

body {
  font-family: 'Roboto', 'PingFang SC', 'Noto Sans SC', 'Microsoft YaHei', sans-serif;
  background: var(--glimmer-bg);
  color: var(--glimmer-text);
  -webkit-font-smoothing: antialiased;
}

#app {
  height: 100%;
}

.app-root {
  min-height: 100%;
}

.app-body.with-header {
  padding-top: 60px;
}

.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.page-fade-enter-from,
.page-fade-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
