<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { redeemInviteApi, DIRECTIONS, directionLabel } from '@/apis'
import { pendingInviteCode, clearPendingInvite } from '@/utils/invite'

const router = useRouter()
const userStore = useUserStore()

const code = ref('')
const submitting = ref(false)
const fromLink = ref(false)

const boundDirections = computed(() => userStore.directions)

onMounted(() => {
  const pending = pendingInviteCode()
  if (pending) {
    code.value = pending
    fromLink.value = true
  }
})

async function onSubmit() {
  const c = code.value.trim()
  if (!c) {
    ElMessage.warning('请输入邀请码')
    return
  }
  submitting.value = true
  try {
    const res = await redeemInviteApi(c)
    clearPendingInvite()
    ElMessage.success(`方向绑定成功：${directionLabel(res.data?.direction)}`)
    await userStore.refreshDirection()
    router.replace('/home')
  } catch (e) {
    // 错误已由拦截器弹窗
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="page-container bind-page">
    <div class="bind-card glimmer-card">
      <div class="bind-head">
        <h2 class="bind-title">{{ boundDirections.length ? '绑定更多方向' : '绑定你的方向' }}</h2>
        <p class="bind-desc">
          不同方向的面试日程互不冲突，预约前请先绑定方向。请输入管理员提供的邀请码（或直接点击管理员发的专属链接）。
          参加过多个方向面试的同学，可分别核销对应方向的邀请码，之后在预约页顶部切换方向即可。
        </p>
      </div>

      <div v-if="boundDirections.length" class="bind-bound">
        <span class="bind-directions-label">已绑定：</span>
        <el-tag
          v-for="d in boundDirections"
          :key="d"
          size="large"
          round
          type="primary"
          effect="light"
        >
          {{ directionLabel(d) }}
        </el-tag>
      </div>

      <el-alert
        v-if="fromLink"
        class="bind-alert"
        type="info"
        :closable="false"
        show-icon
        title="检测到你是通过专属邀请链接进入的，邀请码已自动填入，点击下方按钮即可完成绑定。"
      />

      <div class="bind-form">
        <el-input
          v-model="code"
          size="large"
          placeholder="请输入邀请码，如 K7QM2X9P"
          maxlength="32"
          :disabled="submitting"
          @keyup.enter="onSubmit"
        />
        <el-button type="primary" size="large" :loading="submitting" @click="onSubmit">
          绑定方向
        </el-button>
      </div>

      <div class="bind-directions">
        <span class="bind-directions-label">当前开放方向：</span>
        <el-tag
          v-for="d in DIRECTIONS"
          :key="d.value"
          size="large"
          round
          :style="{ borderColor: d.color, color: d.color, background: '#fff' }"
        >
          {{ d.label }}
        </el-tag>
      </div>

      <p class="bind-tip">
        没有邀请码？请联系对应方向的管理员获取。一个邀请码只能由一位同学核销；绑定多个方向时，各方向的日程互不影响。
      </p>
    </div>
  </div>
</template>

<style scoped>
.bind-page {
  display: flex;
  justify-content: center;
  padding-top: 60px;
}

.bind-card {
  width: 640px;
  max-width: 100%;
  padding: 36px 40px;
}

.bind-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--glimmer-text);
  margin: 0 0 10px;
}

.bind-desc {
  font-size: 14px;
  color: var(--glimmer-text-2);
  line-height: 1.7;
  margin: 0;
}

.bind-alert {
  margin-top: 16px;
}

.bind-bound {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 18px;
  padding: 12px 14px;
  background: var(--glimmer-blue-bg);
  border-radius: 12px;
}

.bind-form {
  display: flex;
  gap: 12px;
  margin-top: 20px;
}

.bind-form .el-input {
  flex: 1;
}

.bind-directions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 24px;
}

.bind-directions-label {
  font-size: 13px;
  color: var(--glimmer-text-3);
}

.bind-tip {
  font-size: 12px;
  color: var(--glimmer-text-3);
  margin: 16px 0 0;
  line-height: 1.7;
}

/* ===== 手机端 ===== */
@media (max-width: 768px) {
  .bind-page {
    padding-top: 24px;
  }

  .bind-card {
    padding: 24px 20px;
  }

  .bind-form {
    flex-direction: column;
  }

  .bind-form .el-button {
    width: 100%;
    min-height: 46px;
  }
}
</style>
