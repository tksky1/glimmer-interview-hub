<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useUserStore } from '@/stores/user'
import { LEGACY_LOGIN_URL } from '@/config'
import logoUrl from '@/assets/logo.png'

const userStore = useUserStore()
let timer = null

// 兜底轮询：部分浏览器/场景 storage 事件不可靠时，自动检测登录完成
function checkToken() {
  if (localStorage.getItem('token') && localStorage.getItem('token') !== userStore.token) {
    userStore.syncToken()
    userStore.loadMe()
  }
}

function goLogin() {
  window.location.href = LEGACY_LOGIN_URL
}

onMounted(() => {
  timer = setInterval(checkToken, 800)
})
onUnmounted(() => {
  clearInterval(timer)
})
</script>

<template>
  <div class="login-gate">
    <div class="gate-card">
      <div class="gate-left">
        <div class="gate-logo">
          <img :src="logoUrl" alt="微光" />
          <div class="gate-logo-text">
            <strong>微光</strong>
            <p>心有微光，不惧黑暗</p>
          </div>
        </div>
        <ul class="gate-points">
          <li>
            <span class="dot dot-1"></span>
            <div>
              <strong>面试日程预约</strong>
              <p>在日历上选择心仪的面试时段</p>
            </div>
          </li>
          <li>
            <span class="dot dot-2"></span>
            <div>
              <strong>心仪时间智能分配</strong>
              <p>按偏好自动分配，尽可能满足所有人</p>
            </div>
          </li>
          <li>
            <span class="dot dot-3"></span>
            <div>
              <strong>结果实时可查</strong>
              <p>确认后随时查看面试时间与地点</p>
            </div>
          </li>
        </ul>
      </div>
      <div class="gate-right">
        <h2>微光工作室招新面试预约</h2>
        <p class="gate-desc">请使用你的微光账号登录。本平台与微光招新交题系统账号互通，无需重复注册。</p>
        <div class="gate-actions">
          <button class="gate-btn" @click="goLogin">
            使用微光账号登录
          </button>
          <p class="gate-tip">点击后将跳转至微光统一登录页，登录成功后自动返回本平台</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-gate {
  min-height: 100vh;
  width: 100%;
  background: #4c64ec url(https://img.bosszhipin.com/static/file/2022/zlqc2m9fao1667185843533.png) bottom / 100% auto no-repeat;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
}

.gate-card {
  width: 880px;
  max-width: 100%;
  min-height: 440px;
  background: #fff;
  border-radius: 28px;
  box-shadow: 0 6px 13px 0 rgba(0, 0, 0, 0.1);
  display: flex;
  overflow: hidden;
  animation: card-in 0.5s cubic-bezier(0.22, 1, 0.36, 1);
}

@keyframes card-in {
  from {
    opacity: 0;
    transform: translateY(24px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

.gate-left {
  width: 320px;
  flex-shrink: 0;
  background: linear-gradient(214deg, #dffbff, #faf6f3);
  border-radius: 28px 0 0 28px;
  padding: 40px 28px;
}

.gate-logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.gate-logo img {
  height: 52px;
  width: auto;
  flex: 0 0 auto;
  object-fit: contain;
}

.gate-logo-text strong {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: rgb(86, 152, 196);
  line-height: 22px;
}

.gate-logo-text p {
  margin: 2px 0 0;
  font-size: 13px;
  color: rgb(86, 152, 196);
}

.gate-points {
  list-style: none;
  margin: 44px 0 0;
  padding: 0;
}

.gate-points li {
  display: flex;
  gap: 12px;
  margin-bottom: 26px;
}

.gate-points .dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-top: 6px;
  flex-shrink: 0;
}

.dot-1 {
  background: #4c64ec;
  box-shadow: 0 0 0 4px rgba(11, 99, 246, 0.15);
}
.dot-2 {
  background: var(--glimmer-blue);
  box-shadow: 0 0 0 4px rgba(11, 99, 246, 0.15);
}
.dot-3 {
  background: #e6a23c;
  box-shadow: 0 0 0 4px rgba(230, 162, 60, 0.15);
}

.gate-points strong {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--glimmer-text);
}

.gate-points p {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--glimmer-text-3);
}

.gate-right {
  flex: 1;
  padding: 56px 48px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.gate-right h2 {
  font-size: 24px;
  font-weight: 600;
  color: var(--glimmer-text);
  margin: 0 0 14px;
}

.gate-desc {
  font-size: 14px;
  color: var(--glimmer-text-2);
  line-height: 1.7;
  margin: 0 0 36px;
}

.gate-btn {
  display: inline-block;
  padding: 0 44px;
  height: 48px;
  line-height: 46px;
  font-size: 17px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(135deg, var(--glimmer-blue), var(--glimmer-blue-2));
  border: none;
  border-radius: 24px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 8px 20px rgba(11, 99, 246, 0.3);
}

.gate-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 26px rgba(11, 99, 246, 0.38);
}

.gate-btn:active {
  transform: translateY(0);
}

.gate-tip {
  margin: 14px 0 0;
  font-size: 12px;
  color: var(--glimmer-text-3);
}

@media (max-width: 760px) {
  .gate-card {
    flex-direction: column;
  }
  .gate-left {
    width: 100%;
    border-radius: 28px 28px 0 0;
  }
  .gate-right {
    padding: 36px 28px;
  }
}
</style>
