<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import WeekCalendar from '@/components/WeekCalendar.vue'
import { useUserStore } from '@/stores/user'
import { confirmDialog } from '@/utils/dialog'
import { calendarApi, savePrefsApi, saveProfileApi, directionLabel, DIRECTIONS } from '@/apis'

const router = useRouter()
const userStore = useUserStore()

const days = ref([])
const sessions = ref([])
const activeId = ref(null)
const picks = ref({}) // { [sessionId]: [slotId...] } 本地编辑中的选择
const saving = ref(false)

// ---------- 方向切换：只展示自己收到过邀请的方向 ----------
const DIR_KEY = 'glimmer-active-direction'
const activeDirection = ref(localStorage.getItem(DIR_KEY) || '')

const availableDirections = computed(() =>
  DIRECTIONS.filter(d => userStore.directions.includes(d.value))
)

// 当前方向可见的场次（多方向同学切换后只看本方向日程）
const visibleSessions = computed(() =>
  activeDirection.value
    ? sessions.value.filter(s => s.session.direction === activeDirection.value)
    : sessions.value
)

// 方向列表变化（绑定新方向）时纠正当前选择
watch(
  () => userStore.directions,
  list => {
    if (!list.includes(activeDirection.value)) {
      activeDirection.value = list[0] || ''
    }
  },
  { immediate: true }
)

watch(activeDirection, d => {
  localStorage.setItem(DIR_KEY, d || '')
  const first = visibleSessions.value[0]
  if (!first || !visibleSessions.value.some(s => s.session.id === activeId.value)) {
    activeId.value = first ? first.session.id : null
  }
})

// ---------- 个人资料（真实姓名 / QQ，用于分配结果表） ----------
const profileDialog = ref(false)
const profileSaving = ref(false)
const profileForm = ref({ name: '', qq: '', phone: '' })
const profileMissing = computed(() => !userStore.name || !userStore.qq || !userStore.phone)

function openProfile() {
  profileForm.value = { name: userStore.name || '', qq: userStore.qq || '', phone: userStore.phone || '' }
  profileDialog.value = true
}

async function onSaveProfile() {
  const name = profileForm.value.name.trim()
  const qq = profileForm.value.qq.trim()
  const phone = profileForm.value.phone.trim()
  if (!name) {
    ElMessage.warning('请填写真实姓名')
    return
  }
  if (!/^\d{5,15}$/.test(qq)) {
    ElMessage.warning('请填写正确的 QQ 号（纯数字）')
    return
  }
  if (!/^1[3-9]\d{9}$/.test(phone)) {
    ElMessage.warning('请填写正确的 11 位手机号')
    return
  }
  profileSaving.value = true
  try {
    await saveProfileApi(name, qq, phone)
    await userStore.loadMe()
    profileDialog.value = false
    ElMessage.success('资料已保存')
  } finally {
    profileSaving.value = false
  }
}

function fmtDate(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
// 展示用：2026-10-24 -> 10月24日 周六
function fmtDay(v) {
  const dt = new Date(`${v}T00:00:00`)
  if (Number.isNaN(dt.getTime())) return v || ''
  return `${dt.getMonth() + 1}月${dt.getDate()}日 周${'日一二三四五六'[dt.getDay()]}`
}
// 场次的日程区间：单日 -> 10月22日 周四；多日 -> 10月22日 - 10月24日（3 个日程）
function sessionDateText(s) {
  const parts = s?.partitions || []
  if (!parts.length) return fmtDay(s?.session?.date)
  if (parts.length === 1) return fmtDay(parts[0].date)
  return `${fmtDay(parts[0].date)} - ${fmtDay(parts[parts.length - 1].date)}（${parts.length} 个日程）`
}
// 预约窗口：近 5 天（一个方向的场次分布在不同日期，成员从中挑一个）
// 时间范围：各日程一致 -> 每天 09:00-10:00；否则标注各日程不同
function sessionTimeText(s) {
  const parts = s?.partitions || []
  if (!parts.length) return `${s?.session?.startTime || ''}-${s?.session?.endTime || ''}`
  const same = parts.every(p => p.startTime === parts[0].startTime && p.endTime === parts[0].endTime)
  if (same) return `每天 ${parts[0].startTime}-${parts[0].endTime}`
  return `${parts[0].startTime}-${parts[parts.length - 1].endTime}（各日程不同）`
}

function initDays() {
  const today = new Date()
  days.value = [0, 1, 2, 3, 4].map(i => {
    const d = new Date(today)
    d.setDate(d.getDate() + i)
    return fmtDate(d)
  })
}

const activeSession = computed(() =>
  visibleSessions.value.find(s => s.session.id === activeId.value) || null
)
const activeLocked = computed(() => activeSession.value?.session.status === 'confirmed')

// 当前场次每个时段可同时面试的人数（各日程一致时显示具体数字，否则给区间）
const activeCapacityText = computed(() => {
  const slots = activeSession.value?.slots || []
  if (!slots.length) return ''
  const caps = slots.map(s => (Number(s.capacity) > 0 ? Number(s.capacity) : 1))
  const min = Math.min(...caps)
  const max = Math.max(...caps)
  if (min === max) return min === 1 ? '每时段 1 人' : `每时段 ${min} 人`
  return `每时段 ${min}~${max} 人（虚线框标识）`
})

const myConfirmedSlot = computed(() => {
  if (!activeSession.value) return null
  return activeSession.value.slots.find(s => s.myConfirmed) || null
})

// 同一方向只能预约 1 个场次：本方向已预约的其它场次（保存新场次时会被替换）
const bookedSessions = computed(() =>
  visibleSessions.value.filter(s => (s.slots || []).some(x => x.myRank > 0))
)
const willReplace = computed(() => {
  if (!activeSession.value) return null
  return bookedSessions.value.find(s => s.session.id !== activeId.value) || null
})

// 扁平化所有场次时段供日历渲染（本地未保存的选择即时高亮）
const calSlots = computed(() => {
  const out = []
  for (const s of visibleSessions.value) {
    const local = picks.value[s.session.id]
    // 本地列表存在时以它为准：否则取消选择后 pickedRank=0，会回退到服务端 myRank，
    // 导致已取消的时段仍显示「心仪一」并高亮（旧版问题）
    const hasLocal = Array.isArray(local)
    for (const sl of s.slots) {
      out.push({
        id: sl.id,
        date: sl.date || s.session.date, // 场次可跨多天，时段自带日期
        sessionId: s.session.id,
        startTime: sl.startTime,
        endTime: sl.endTime,
        location: sl.location,
        firstChoiceCount: sl.firstChoiceCount,
        capacity: sl.capacity,
        occupied: sl.occupied,
        myRank: hasLocal ? 0 : sl.myRank,
        pickedRank: hasLocal ? local.indexOf(sl.id) + 1 : 0,
        myConfirmed: sl.myConfirmed
      })
    }
  }
  return out
})

const activePicks = computed(() => {
  if (!activeSession.value) return []
  const list = picks.value[activeId.value] || []
  return list
    .map(id => activeSession.value.slots.find(s => s.id === id))
    .filter(Boolean)
})

const dirty = computed(() => {
  if (!activeSession.value) return false
  const local = (picks.value[activeId.value] || []).join(',')
  const saved = (activeSession.value.slots || [])
    .filter(s => s.myRank > 0)
    .sort((a, b) => a.myRank - b.myRank)
    .map(s => s.id)
    .join(',')
  return local !== saved
})

async function load() {
  // 先记录刷新前服务端已保存的顺序，用于识别哪些场次还有未保存的本地改动
  const prevSaved = {}
  for (const s of sessions.value) {
    prevSaved[s.session.id] = (s.slots || [])
      .filter(x => x.myRank > 0)
      .sort((a, b) => a.myRank - b.myRank)
      .map(x => x.id)
      .join(',')
  }
  const res = await calendarApi(days.value[0], days.value[days.value.length - 1])
  sessions.value = res.data?.sessions || []
  // 先保留窗口外场次的本地编辑，翻到别的日期再翻回来时选择不会丢
  const next = { ...picks.value }
  for (const s of sessions.value) {
    const key = s.session.id
    const ranked = (s.slots || [])
      .filter(x => x.myRank > 0)
      .sort((a, b) => a.myRank - b.myRank)
      .map(x => x.id)
    const local = picks.value[key]
    const localStr = Array.isArray(local) ? local.join(',') : null
    // 有未保存改动时保留本地选择：翻日期 / 重新拉取不应把正在编辑的选择冲掉
    next[key] = localStr !== null && localStr !== prevSaved[key] ? [...local] : ranked
  }
  picks.value = next
  if (!visibleSessions.value.some(s => s.session.id === activeId.value)) {
    activeId.value = visibleSessions.value[0]?.session.id || null
  }
}

function onSlotClick(slot) {
  const sid = slot.sessionId
  const target = sessions.value.find(s => s.session.id === sid)
  if (!target) return
  activeId.value = sid
  if (target.session.status === 'confirmed') {
    ElMessage.warning('分配已确认，心仪时间已锁定')
    return
  }
  // 容量只是「同时面试人数」的参考：超出名额照样可以选，交给分配算法去协调
  const list = [...(picks.value[sid] || [])]
  const idx = list.indexOf(slot.id)
  if (idx >= 0) {
    list.splice(idx, 1)
  } else if (list.length < 3) {
    list.push(slot.id)
  } else {
    ElMessage.warning('最多选择 3 个心仪时间，请先移除一个')
    return
  }
  picks.value[sid] = list
}

function move(index, delta) {
  const list = [...(picks.value[activeId.value] || [])]
  const target = index + delta
  if (target < 0 || target >= list.length) return
  ;[list[index], list[target]] = [list[target], list[index]]
  picks.value[activeId.value] = list
}

function remove(index) {
  const list = [...(picks.value[activeId.value] || [])]
  list.splice(index, 1)
  picks.value[activeId.value] = list
}

function onDaysChange(v) {
  days.value = v
}

// 手机端底部保存条：有改动或已选时段时显示（未选满则提示还差几个）
const showMobileSave = computed(
  () => !activeLocked.value && (dirty.value || activePicks.value.length > 0)
)

async function save() {
  if (activeLocked.value) return
  const list = picks.value[activeId.value] || []
  if (list.length !== 3) {
    ElMessage.warning(`每场必须选择 3 个心仪时间，当前已选 ${list.length} 个，还差 ${3 - list.length} 个`)
    return
  }
  if (!dirty.value) {
    ElMessage.info('心仪时间没有改动')
    return
  }
  // 同一方向只能预约一个场次：换场次前显式确认
  if (willReplace.value) {
    const other = willReplace.value.session
    try {
      await confirmDialog(
        `你在「${directionLabel(activeSession.value.session.direction)}」已预约「${other.title}（${fmtDay(other.date)}）」。` +
        `保存后将放弃该场次的心仪时间，改为预约「${activeSession.value.session.title}（${fmtDay(activeSession.value.session.date)}）」。`,
        '更换预约场次',
        { confirmButtonText: '确定更换', cancelButtonText: '继续用原来场次', type: 'warning' }
      )
    } catch {
      return
    }
  }
  saving.value = true
  try {
    await savePrefsApi(activeId.value, list)
    ElMessage.success('心仪时间已保存')
    await load()
  } finally {
    saving.value = false
  }
}

watch(days, load)
onMounted(() => {
  initDays()
  load()
})
</script>

<template>
  <div class="page-container">
    <!-- 顶部：方向切换 + 资料补全 -->
    <div class="home-topbar glimmer-card">
      <div class="dir-switch" v-if="availableDirections.length">
        <span class="dir-label">面试方向</span>
        <el-radio-group v-model="activeDirection" size="large" class="dir-group">
          <el-radio-button v-for="d in availableDirections" :key="d.value" :value="d.value">
            {{ d.label }}
          </el-radio-button>
        </el-radio-group>
      </div>
      <div class="topbar-right">
        <span v-if="userStore.studentid" class="topbar-meta">学号 {{ userStore.studentid }}</span>
        <el-button v-if="profileMissing" type="primary" plain @click="openProfile">
          <el-icon style="margin-right: 4px"><EditPen /></el-icon>完善姓名 / QQ / 手机
        </el-button>
        <el-button v-else text @click="openProfile">
          {{ userStore.name }}<span v-if="userStore.phone"> · {{ userStore.phone }}</span>
        </el-button>
        <el-button text type="primary" @click="router.push('/bind')">＋绑定其他方向</el-button>
      </div>
    </div>

    <div class="member-layout" v-if="visibleSessions.length">
      <div class="calendar-area">
        <WeekCalendar
          :days="days"
          :slots="calSlots"
          clickable
          @update:days="onDaysChange"
          @slot-click="onSlotClick"
        />
        <div class="tips">
          <el-alert type="info" :closable="false" show-icon>
            <template #title>
              点击日历中任意可用时段，按偏好顺序选满 3 个心仪时间：同一个场次的多个日程都可以选，同一天也可以选多个时段。
              再次点击可取消；选满后在右侧「我的心仪时间」用 ↑↓ 调整顺序并保存，保存后的顺序就是分配时的优先顺序。
              <b>虚线框 + 「N 人 · 0/N」徽章表示该时段计划同时面试 N 人</b>：超过 N 人也照样可以选，分配时会按心仪顺序协调，未必都能进这个时段。
            </template>
          </el-alert>
        </div>
      </div>

    <!-- 手机端底部保存条：选满 3 个且有改动时出现 -->
    <div
      v-if="showMobileSave"
      class="mobile-save-bar"
    >
      <span class="msb-count">
        心仪时间 {{ activePicks.length }}/3
        <em v-if="activePicks.length < 3">还差 {{ 3 - activePicks.length }} 个</em>
      </span>
      <el-button type="primary" size="large" :loading="saving" @click="save">
        {{ activePicks.length === 3 ? '保存心仪时间' : '选满 3 个后可保存' }}
      </el-button>
    </div>

    <div class="side-panel">
        <div v-if="myConfirmedSlot" class="confirmed-banner">
          <div class="confirmed-title">
            <el-icon><CircleCheck /></el-icon> 面试已确认
          </div>
          <div class="confirmed-time">{{ myConfirmedSlot.startTime }} - {{ myConfirmedSlot.endTime }}</div>
          <div class="confirmed-loc">{{ myConfirmedSlot.location }}</div>
          <div class="confirmed-date">{{ fmtDay(myConfirmedSlot.date || activeSession.session.date) }}</div>
        </div>

        <div class="panel-card glimmer-card" v-if="activeSession">
          <div class="panel-title-row">
            <span class="panel-title">{{ activeSession.session.title }}</span>
            <el-tag size="small" effect="plain" round>{{ directionLabel(activeSession.session.direction) }}</el-tag>
          </div>

          <div class="session-meta">
            <span><el-icon><Calendar /></el-icon> {{ sessionDateText(activeSession) }}</span>
            <span><el-icon><Clock /></el-icon> {{ sessionTimeText(activeSession) }}</span>
            <span><el-icon><Location /></el-icon> {{ activeSession.session.location }}</span>
            <span><el-icon><Timer /></el-icon> 每场 {{ activeSession.session.slotMinutes }} 分钟</span>
            <span v-if="activeCapacityText"><el-icon><UserFilled /></el-icon> {{ activeCapacityText }}</span>
            <span><el-icon><User /></el-icon> 已参与 {{ activeSession.participants }} 人</span>
            <span v-if="activeSession.session.contact">
              <el-icon><Iphone /></el-icon> {{ activeSession.session.contact }}
              <template v-if="activeSession.session.contactQQ">（QQ {{ activeSession.session.contactQQ }}）</template>
            </span>
          </div>

          <div v-if="willReplace" class="replace-tip">
            本方向已预约「{{ willReplace.session.title }}（{{ sessionDateText(willReplace) }}）」，
            保存本场次会自动替换它
          </div>

          <div class="panel-title">
            我的心仪时间
            <el-tag v-if="activeLocked" type="info" size="small">已锁定</el-tag>
          </div>
          <div v-if="!activePicks.length" class="panel-empty">
            还没有选择心仪时间<br />点击左侧日历时段开始选择
          </div>
          <div v-for="(item, index) in activePicks" :key="item.id" class="pick-item">
            <span class="pick-rank" :class="`rank-${index + 1}`">{{ ['①', '②', '③'][index] }}</span>
            <div class="pick-info">
              <div class="pick-time">
                <span class="pick-date">{{ fmtDay(item.date) }}</span>
                <span class="pick-range">{{ item.startTime }} - {{ item.endTime }}</span>
                <span v-if="(item.capacity || 1) > 1" class="pick-cap">可面 {{ item.capacity }} 人</span>
              </div>
              <div class="pick-loc">{{ item.location }}</div>
            </div>
            <div class="pick-actions">
              <div class="move-group">
                <el-button text :disabled="index === 0 || activeLocked" title="上移" @click="move(index, -1)">
                  <el-icon><ArrowUp /></el-icon>
                </el-button>
                <el-button text :disabled="index === activePicks.length - 1 || activeLocked" title="下移" @click="move(index, 1)">
                  <el-icon><ArrowDown /></el-icon>
                </el-button>
              </div>
              <el-button class="remove-btn" text type="danger" :disabled="activeLocked" title="移除此心仪时间" @click="remove(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div v-if="!activeLocked" class="pick-hint" :class="{ ok: activePicks.length === 3 }">
            <template v-if="activePicks.length < 3">
              还需选择 {{ 3 - activePicks.length }} 个时段，每场必须选满 3 个才能保存
            </template>
            <template v-else-if="dirty">已选满 3 个，记得点击保存</template>
            <template v-else>已保存，点击左侧时段可继续调整</template>
          </div>
          <div class="panel-footer">
            <span class="pick-count">{{ activePicks.length }} / 3</span>
            <el-button type="primary" :disabled="!dirty || activeLocked" :loading="saving" @click="save">
              保存心仪时间
            </el-button>
          </div>
        </div>

        <div class="panel-card glimmer-card" v-if="visibleSessions.length > 1">
          <div class="panel-title">切换场次</div>
          <el-select v-model="activeId" placeholder="选择场次" style="width: 100%">
            <el-option
              v-for="s in visibleSessions"
              :key="s.session.id"
              :label="`${s.session.title}（${sessionDateText(s)}）`"
              :value="s.session.id"
            />
          </el-select>
        </div>

        <div class="panel-card glimmer-card">
          <div class="panel-title">时段热度</div>
          <p class="hot-desc">「N 人首选」= 已有 N 人把这个时段排在第一位。人数越多竞争越激烈，可以多看看其他时段。</p>
        </div>
      </div>
    </div>

    <el-empty
      v-else
      :description="`「${directionLabel(activeDirection)}」近 5 天暂无你可预约的面试日程，请关注后续通知`"
    />

    <!-- 资料弹窗：真实姓名 / QQ（分配结果表需要） -->
    <el-dialog
      v-model="profileDialog"
      title="完善个人信息"
      width="420px"
      class="profile-dialog"
      append-to-body
    >
      <p class="dialog-desc">
        管理员分配面试时段后，需要用真实姓名、QQ 号与手机号联系你，请如实填写（仅管理员可见）。
      </p>
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="真实姓名">
          <el-input v-model="profileForm.name" placeholder="请输入真实姓名" maxlength="20" />
        </el-form-item>
        <el-form-item label="QQ 号">
          <el-input v-model="profileForm.qq" placeholder="请输入 QQ 号（纯数字）" maxlength="15" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="profileForm.phone" placeholder="请输入 11 位手机号" maxlength="11" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="profileDialog = false">取消</el-button>
        <el-button type="primary" :loading="profileSaving" @click="onSaveProfile">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* 底部保存条只在手机端出现：桌面端必须隐藏，否则会以普通 div 插入布局把右侧面板挤乱 */
.mobile-save-bar {
  display: none;
}

/* 选满 3 个的显式提示 */
.pick-hint {
  margin-top: 12px;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--glimmer-blue-bg);
  color: var(--glimmer-blue);
  font-size: 12.5px;
  line-height: 1.5;
}

.pick-hint.ok {
  background: var(--glimmer-success-bg);
  color: var(--glimmer-success);
}

.msb-count em {
  display: block;
  font-style: normal;
  font-size: 12px;
  font-weight: 400;
  color: var(--glimmer-blue);
  margin-top: 2px;
}

.home-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  padding: 12px 16px;
  margin-bottom: 16px;
}

.dir-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.dir-label {
  font-size: 13px;
  color: var(--glimmer-text-3);
  white-space: nowrap;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.topbar-meta {
  font-size: 13px;
  color: var(--glimmer-text-3);
}

.dialog-desc {
  font-size: 13px;
  color: var(--glimmer-text-2);
  line-height: 1.7;
  margin: 0 0 14px;
}

.member-layout {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 20px;
  align-items: start;
}

.tips {
  margin-top: 16px;
}

.side-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: sticky;
  top: 84px;
}

.confirmed-banner {
  background: linear-gradient(135deg, #1e9600, #4eb52e);
  border-radius: 16px;
  padding: 20px;
  color: #fff;
  box-shadow: 0 8px 22px rgba(30, 150, 0, 0.3);
}

.confirmed-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  opacity: 0.95;
}

.confirmed-time {
  font-size: 26px;
  font-weight: 700;
  margin-top: 8px;
  font-variant-numeric: tabular-nums;
}

.confirmed-loc {
  font-size: 14px;
  margin-top: 4px;
  opacity: 0.92;
}

.confirmed-date {
  font-size: 13px;
  margin-top: 2px;
  opacity: 0.8;
}

.panel-card {
  padding: 18px;
}

.panel-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--glimmer-text);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.panel-title-row + .session-meta {
  margin-bottom: 14px;
}

.session-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: var(--glimmer-text-2);
  padding-bottom: 14px;
  border-bottom: 1px dashed var(--glimmer-border);
}

.session-meta span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.panel-card .panel-title:not(.panel-title-row) {
  margin-bottom: 14px;
  margin-top: 4px;
}

.panel-empty {
  text-align: center;
  color: var(--glimmer-text-3);
  font-size: 13px;
  line-height: 1.8;
  padding: 20px 0;
}

.pick-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--glimmer-border);
  border-radius: 12px;
  margin-bottom: 10px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.pick-item:hover {
  border-color: var(--glimmer-blue-2);
  box-shadow: 0 4px 10px rgba(11, 99, 246, 0.12);
}

.pick-rank {
  font-size: 20px;
  line-height: 1;
}

.rank-1 {
  color: var(--glimmer-blue);
}
.rank-2 {
  color: var(--glimmer-blue-2);
}
.rank-3 {
  color: #7aabff;
}

.pick-info {
  flex: 1;
  min-width: 0;
}

.pick-time {
  display: flex;
  align-items: center;
  gap: 6px;
  /* 放不下时让日期整体换行，绝不把 15:00 - 15:30 拆开 */
  flex-wrap: wrap;
  font-size: 13px;
  font-weight: 600;
  color: var(--glimmer-text);
  font-variant-numeric: tabular-nums;
}

/* 时间区间整体不换行：宁可将日期省略，也不把 15:00 - 15:30 拆成两行 */
.pick-range {
  flex: 0 0 auto;
  white-space: nowrap;
}

.pick-date {
  flex: 0 0 auto;
  padding: 0 5px;
  border-radius: 6px;
  background: var(--glimmer-blue-bg);
  color: var(--glimmer-blue);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
}

/* 同方向换场次的提示 */
.replace-tip {
  margin: 10px 0 0;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--glimmer-warning-bg, #fdf6ec);
  color: #b88230;
  font-size: 12.5px;
  line-height: 1.5;
}

.pick-cap {
  flex: 0 0 auto;
  padding: 0 5px;
  border-radius: 6px;
  background: #e8f1ff;
  border: 1px solid #b9d4ff;
  color: #0b63f6;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
}

.pick-loc {
  font-size: 12px;
  color: var(--glimmer-text-3);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 上移/下移拼成一个紧凑的竖排按钮组，删除按钮单独靠右 */
.pick-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.move-group {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--glimmer-border);
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.move-group .el-button {
  margin: 0 !important;
  width: 26px;
  height: 20px;
  padding: 0;
  border-radius: 0;
  color: var(--glimmer-text-2);
}

.move-group .el-button + .el-button {
  margin-left: 0 !important;
  border-top: 1px solid var(--glimmer-border);
}

.move-group .el-button:hover:not(.is-disabled) {
  background: var(--glimmer-blue-bg);
  color: var(--glimmer-blue);
}

.move-group .el-icon {
  font-size: 12px;
}

.pick-actions .remove-btn {
  margin-left: 2px !important;
  height: 26px;
  padding: 0 4px;
}

.panel-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px dashed var(--glimmer-border);
}

.pick-count {
  font-size: 13px;
  color: var(--glimmer-text-3);
}

.hot-desc {
  font-size: 13px;
  color: var(--glimmer-text-2);
  line-height: 1.7;
  margin: 0;
}

@media (max-width: 1100px) {
  .member-layout {
    grid-template-columns: 1fr;
  }
  .side-panel {
    position: static;
  }
}

/* ===== 手机端 ===== */
@media (max-width: 768px) {
  .page-container {
    padding: 14px 12px 84px;
  }

  .home-topbar {
    padding: 10px 12px;
    gap: 10px;
  }

  .dir-switch {
    width: 100%;
    overflow-x: auto;
    padding-bottom: 2px;
  }

  .dir-group {
    flex-wrap: nowrap;
  }

  .dir-group :deep(.el-radio-button__inner) {
    padding: 8px 12px;
  }

  .topbar-right {
    width: 100%;
    flex-wrap: wrap;
    justify-content: flex-start;
    gap: 6px 10px;
  }

  .topbar-meta {
    font-size: 12px;
  }

  .topbar-right :deep(.el-button) {
    padding: 6px 8px;
  }

  .session-meta {
    font-size: 12.5px;
  }

  .pick-item {
    padding: 12px;
  }

  .pick-actions {
    flex-direction: row;
    align-items: center;
  }

  .pick-actions .remove-btn {
    min-height: 36px;
    min-width: 36px;
  }

  .move-group .el-button {
    width: 32px;
    height: 24px;
  }

  .panel-footer .el-button {
    min-height: 42px;
  }

  .tips {
    margin-top: 12px;
  }

  .mobile-save-bar {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 90;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 16px calc(10px + env(safe-area-inset-bottom));
    background: rgba(255, 255, 255, 0.96);
    backdrop-filter: blur(8px);
    border-top: 1px solid var(--glimmer-border);
    box-shadow: 0 -4px 16px rgba(11, 99, 246, 0.1);
    animation: bar-in 0.25s ease;
  }

  @keyframes bar-in {
    from {
      transform: translateY(100%);
    }
    to {
      transform: none;
    }
  }

  .msb-count {
    font-size: 14px;
    font-weight: 600;
    color: var(--glimmer-text);
  }

  .mobile-save-bar .el-button {
    min-height: 42px;
  }
}
</style>
