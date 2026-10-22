<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { confirmDialog } from '@/utils/dialog'
import { useUserStore } from '@/stores/user'
import logoUrl from '@/assets/logo.png'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const navItems = computed(() => {
  const items = [
    { path: '/home', label: '面试预约' },
    { path: '/my', label: '我的心仪时间' }
  ]
  if (userStore.isAdmin) {
    items.push({ path: '/admin', label: '管理后台' })
  }
  return items
})

async function onLogout() {
  try {
    await confirmDialog('确定要退出当前账号吗？', '退出登录', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }
  userStore.logout()
  ElMessage.success('已退出登录')
}
</script>

<template>
  <header class="app-header">
    <div class="header-left" @click="router.push('/home')">
      <img :src="logoUrl" alt="微光工作室" class="header-logo" />
      <span class="header-title">微光工作室</span>
      <span class="header-sub">面试预约平台</span>
    </div>

    <!-- 桌面端导航 -->
    <nav class="header-nav">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :class="{ active: route.path.startsWith(item.path) }"
      >
        {{ item.label }}
      </router-link>
    </nav>

    <div class="header-right">
      <span v-if="userStore.userInfo" class="header-user">
        <el-icon><User /></el-icon>
        {{ userStore.userInfo.username }}
        <el-tag v-if="userStore.isAdmin" size="small" type="success" effect="light" round>管理员</el-tag>
      </span>
      <el-button class="logout-desktop" type="primary" text @click="onLogout">
        <el-icon style="margin-right: 4px"><SwitchButton /></el-icon>退出
      </el-button>

      <!-- 手机端汉堡菜单 -->
      <el-popover placement="bottom-end" :width="180" trigger="click" popper-class="header-menu-pop">
        <template #reference>
          <el-button class="menu-mobile" type="primary" text>
            <el-icon :size="22"><Menu /></el-icon>
          </el-button>
        </template>
        <div class="mobile-menu">
          <div v-if="userStore.userInfo" class="mobile-user">
            <el-icon><User /></el-icon>
            <span>{{ userStore.userInfo.username }}</span>
            <el-tag v-if="userStore.isAdmin" size="small" type="success" effect="light" round>管理员</el-tag>
          </div>
          <router-link
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            class="mobile-menu-item"
            :class="{ active: route.path.startsWith(item.path) }"
          >
            {{ item.label }}
          </router-link>
          <div class="mobile-menu-item danger" @click="onLogout">
            <el-icon><SwitchButton /></el-icon> 退出登录
          </div>
        </div>
      </el-popover>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: 0 2px 12px rgba(11, 99, 246, 0.08);
  z-index: 100;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
}

.header-logo {
  /* 方形素材 + 方形画框：等比缩放，不拉伸不变形 */
  width: 44px;
  height: 44px;
  object-fit: contain;
  flex: 0 0 auto;
  max-width: none;
}

.header-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--glimmer-text);
}

.header-sub {
  font-size: 13px;
  color: var(--glimmer-text-3);
  margin-left: 2px;
  padding-left: 10px;
  border-left: 1px solid var(--glimmer-border);
}

.header-nav {
  display: flex;
  gap: 6px;
}

.nav-item {
  padding: 8px 16px;
  border-radius: 10px;
  font-size: 14px;
  color: var(--glimmer-text-2);
  transition: all 0.2s ease;
}

.nav-item:hover {
  color: var(--glimmer-green);
  background: var(--glimmer-green-bg);
}

.nav-item.active {
  color: #fff;
  background: linear-gradient(135deg, var(--glimmer-green), var(--glimmer-green-2));
  font-weight: 500;
  box-shadow: 0 4px 10px rgba(11, 99, 246, 0.28);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-user {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: var(--glimmer-text-2);
}

.menu-mobile {
  display: none;
}

@media (max-width: 900px) {
  .app-header {
    padding: 0 14px;
  }
  .header-sub {
    display: none;
  }
}

/* ===== 手机端：导航折叠为汉堡菜单 ===== */
@media (max-width: 768px) {
  .header-nav,
  .header-user,
  .logout-desktop {
    display: none;
  }

  .menu-mobile {
    display: inline-flex;
  }

  .header-title {
    font-size: 15px;
  }

  .header-logo {
    width: 36px;
    height: 36px;
  }
}
</style>

<style>
.header-menu-pop .mobile-menu {
  display: flex;
  flex-direction: column;
}

.header-menu-pop .mobile-user {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  font-size: 14px;
  color: var(--glimmer-text-2);
  border-bottom: 1px solid var(--glimmer-border);
  margin-bottom: 6px;
}

.header-menu-pop .mobile-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-radius: 8px;
  font-size: 15px;
  color: var(--glimmer-text);
  cursor: pointer;
}

.header-menu-pop .mobile-menu-item.active {
  color: var(--glimmer-green);
  background: var(--glimmer-green-bg);
  font-weight: 600;
}

.header-menu-pop .mobile-menu-item.danger {
  color: var(--el-color-danger);
  border-top: 1px solid var(--glimmer-border);
  border-radius: 0 0 8px 8px;
  margin-top: 6px;
}
</style>
