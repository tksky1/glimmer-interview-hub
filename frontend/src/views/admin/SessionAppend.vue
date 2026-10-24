<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import WeekCalendar from '@/components/WeekCalendar.vue'
import { sessionDetailApi, appendSessionApi, directionLabel, directionColor } from '@/apis'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.id)

const loading = ref(false)
const submitting = ref(false)
const session = ref(null)
const existingPartitions = ref([])
const existingSlots = ref([])
// 本次新增的日程
const newParts = ref([])
const slotCaps = ref({})
const days = ref([])

const statusMap = {
  draft: { text: '草稿', type: 'info' },
  published: { text: '预约中', type: 'primary' },
  assigned: { text: '已分配待确认', type: 'warning' },
  confirmed: { text: '已确认', type: 'success' }
}
const isDraft = computed(() => session.value?.status === 'draft')
const isConfirmed = computed(() => session.value?.status === 'confirmed')
const slotMinutes = computed(() => Number(session.value?.slotMinutes) || 20)
const defaultCap = computed(() => Number(session.value?.slotCapacity) || 1)

function today() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function fmtDate(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function daysFrom(start, n = 5) {
  const first = new Date(start + 'T00:00:00')
  if (Number.isNaN(first.getTime())) return Array.from({ length: n }, (_, i) => {
    const d = new Date()
    d.setDate(d.getDate() + i)
    return fmtDate(d)
  })
  return Array.from({ length: n }, (_, i) => {
    const d = new Date(first)
    d.setDate(d.getDate() + i)
    return fmtDate(d)
  })
}
function fmtDayShort(v) {
  const dt = new Date(`${v}T00:00:00`)
  if (Number.isNaN(dt.getTime())) return v || ''
  return `${dt.getMonth() + 1}月${dt.getDate()}日 周${'日一二三四五六'[dt.getDay()]}`
}

function toMin(hm) {
  const [h, m] = String(hm || '0:0').split(':').map(Number)
  return h * 60 + m
}
function fmtMin(m) {
  return `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`
}

function newPartition(date = today()) {
  return { date, startTime: '09:00', endTime: '12:00', location: '', capacity: defaultCap.value }
}

// 新增日程切分出的时段（容量优先级：逐时段覆盖 > 日程人数 > 场次默认）
const newSlotRows = computed(() => {
  const step = slotMinutes.value
  if (!step) return []
  const rows = []
  for (const p of newParts.value) {
    if (!p.date || !p.startTime || !p.endTime) continue
    const start = toMin(p.startTime)
    const n = Math.max(0, Math.floor((toMin(p.endTime) - start) / step))
    const cap = Number(p.capacity) || defaultCap.value
    for (let i = 0; i < n; i++) {
      const s = start + i * step
      const key = `${p.date} ${fmtMin(s)}`
      const over = Number(slotCaps.value[key]) || 0
      rows.push({
        key,
        date: p.date,
        startTime: fmtMin(s),
        endTime: fmtMin(s + step),
        defaultCap: cap,
        cap: over > 0 ? over : cap,
        overridden: over > 0
      })
    }
  }
  return rows.sort((a, b) =>
    a.date === b.date ? a.startTime.localeCompare(b.startTime) : a.date.localeCompare(b.date)
  )
})

const newSlotCount = computed(() => newSlotRows.value.length)
const newTotalCapacity = computed(() => newSlotRows.value.reduce((sum, r) => sum + r.cap, 0))

// 日历上显示的已有时段（只读）
const existingCalSlots = computed(() => existingSlots.value.map(sl => ({
  id: sl.id,
  sessionId: sl.sessionId,
  date: sl.date || session.value?.date,
  startTime: sl.startTime,
  endTime: sl.endTime,
  location: sl.location,
  capacity: sl.capacity,
  firstChoiceCount: sl.firstChoiceCount || 0,
  occupied: sl.occupied || 0
})))

// 新增日程的蓝色框选预览（已有时段由日历上的时段块展示，不能被重切）
const selection = computed(() => newParts.value.map(p => ({
  date: p.date,
  startTime: p.startTime,
  endTime: p.endTime
})))

function sortNewParts() {
  newParts.value.sort((a, b) =>
    a.date === b.date
      ? toMin(a.startTime || '00:00') - toMin(b.startTime || '00:00')
      : (a.date || '').localeCompare(b.date || '')
  )
}

function addPartition() {
  const list = newParts.value
  const last = list[list.length - 1]
  let date = days.value[0] || today()
  if (last?.date) {
    const next = new Date(last.date + 'T00:00:00')
    next.setDate(next.getDate() + 1)
    date = fmtDate(next)
  }
  list.push(newPartition(date))
  sortNewParts()
}

function removePartition(i) {
  const [removed] = newParts.value.splice(i, 1)
  // 清掉该日程上被单独指定过的时段容量
  for (const k of Object.keys(slotCaps.value)) {
    if (removed && k.startsWith(removed.date + ' ')) delete slotCaps.value[k]
  }
}

// 与已有时段冲突（已有日程不能被重切，只能另选空闲时间）
function clashWithExisting(date, start, end) {
  return existingSlots.value.some(sl => {
    const d = sl.date || session.value?.date
    return d === date && toMin(sl.startTime) < end && toMin(sl.endTime) > start
  })
}

function onSelectionRemove({ date, startTime, endTime }) {
  const i = newParts.value.findIndex(
    p => p.date === date && p.startTime === startTime && p.endTime === endTime
  )
  if (i >= 0) removePartition(i)
}

// 拖拽框选新增日程：与已有时段重叠直接拒绝；新增日程之间以新框选为准裁剪
function onRangeSelected({ date, startTime, endTime }) {
  const s = toMin(startTime)
  const e = toMin(endTime)
  if (e - s < 5) return
  if (clashWithExisting(date, s, e)) {
    ElMessage.warning(`${date} ${startTime}-${endTime} 与已有时段重叠，请避开已有时间段框选`)
    return
  }
  const dup = newParts.value.some(
    p => p.date === date && toMin(p.startTime) === s && toMin(p.endTime) === e
  )
  if (dup) return
  const next = []
  for (const p of newParts.value) {
    if (p.date !== date) {
      next.push(p)
      continue
    }
    const ps = toMin(p.startTime || '00:00')
    const pe = toMin(p.endTime || '00:00')
    if (ps >= e || pe <= s) {
      next.push(p)
      continue
    }
    if (ps < s) next.push({ ...p, endTime: fmtMin(s) })
    if (pe > e) next.push({ ...p, startTime: fmtMin(e) })
  }
  next.push({ date, startTime, endTime, location: '', capacity: defaultCap.value })
  newParts.value = next
  sortNewParts()
  if (!days.value.includes(date)) days.value = daysFrom(date)
}

function setSlotCap(key, v) {
  const n = Number(v)
  if (n > 0) slotCaps.value[key] = n
  else delete slotCaps.value[key]
}

function hasOverlap() {
  const byDay = {}
  for (const p of newParts.value) {
    const s = toMin(p.startTime || '00:00')
    const e = toMin(p.endTime || '00:00')
    ;(byDay[p.date] = byDay[p.date] || []).push([s, e])
  }
  for (const list of Object.values(byDay)) {
    list.sort((a, b) => a[0] - b[0])
    for (let i = 1; i < list.length; i++) {
      if (list[i][0] < list[i - 1][1]) return true
    }
  }
  return false
}

function onDaysChange(v) {
  days.value = v
}

function validate() {
  if (isDraft.value) {
    ElMessage.warning('草稿场次请直接编辑场次')
    return false
  }
  if (!newParts.value.length) {
    ElMessage.warning('请在日历上框选至少一个新增日程')
    return false
  }
  for (const p of newParts.value) {
    if (!p.date) {
      ElMessage.warning('请为每个新增日程选择日期')
      return false
    }
    const s = toMin(p.startTime || '00:00')
    const e = toMin(p.endTime || '00:00')
    if (e <= s) {
      ElMessage.warning(`${p.date} 的结束时间必须晚于开始时间`)
      return false
    }
    if (clashWithExisting(p.date, s, e)) {
      ElMessage.warning(`${p.date} 的新增日程与已有时段重叠`)
      return false
    }
  }
  if (hasOverlap()) {
    ElMessage.warning('新增日程之间不能重叠')
    return false
  }
  if (newSlotCount.value < 1) {
    ElMessage.warning('新增时间段不足以容纳一场面试')
    return false
  }
  return true
}

async function onSubmit() {
  if (!validate()) return
  submitting.value = true
  try {
    const ps = newParts.value
      .filter(p => p.date && p.startTime && p.endTime)
      .map(p => ({
        date: p.date,
        startTime: p.startTime,
        endTime: p.endTime,
        location: (p.location || '').trim(),
        capacity: Number(p.capacity) || defaultCap.value
      }))
      .sort((a, b) => (a.date === b.date ? a.startTime.localeCompare(b.startTime) : a.date.localeCompare(b.date)))
    const res = await appendSessionApi(
      sessionId,
      ps,
      newSlotRows.value.filter(r => r.overridden).map(r => ({ date: r.date, startTime: r.startTime, capacity: r.cap }))
    )
    ElMessage.success(`已追加 ${res.data?.added || 0} 个面试时段`)
    router.push('/admin')
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await sessionDetailApi(sessionId)
    session.value = res.data.session
    existingPartitions.value = res.data.partitions || []
    existingSlots.value = res.data.slots || []
    const first = existingPartitions.value[0]?.date || session.value?.date || today()
    days.value = daysFrom(first)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div v-loading="loading" class="page-container">
    <template v-if="session">
      <div class="page-head">
        <div>
          <h2 class="page-title"><el-icon><Calendar /></el-icon> 追加日程</h2>
          <p class="page-subtitle">
            <el-tag size="small" round :style="{ borderColor: directionColor(session.direction), color: directionColor(session.direction), background: '#fff' }">
              {{ directionLabel(session.direction) }}
            </el-tag>
            {{ session.title }} · {{ session.location }} · 每场 {{ session.slotMinutes }} 分钟
            <el-tag :type="statusMap[session.status]?.type" size="small" effect="light" round class="status-tag">
              {{ statusMap[session.status]?.text || session.status }}
            </el-tag>
          </p>
        </div>
        <el-button @click="router.push('/admin')">
          <el-icon style="margin-right: 4px"><ArrowLeft /></el-icon> 返回列表
        </el-button>
      </div>

      <el-alert
        class="append-tip"
        type="info"
        :closable="false"
        show-icon
        title="追加只新增时段，不改动已有时段：已有的心仪时间、分配结果都会保留，新时段对成员显示为空闲。"
      />

      <el-alert
        v-if="isDraft"
        class="append-tip"
        type="warning"
        :closable="false"
        show-icon
        title="该场次仍是草稿：请直接使用「编辑」重新设置全部日程，草稿不需要追加。"
      />

      <el-alert
        v-if="isConfirmed"
        class="append-tip"
        type="warning"
        :closable="false"
        show-icon
        title="该场次分配已确认，日程不可再修改（含追加）。如需调整请删除场次后重建。"
      />

      <div class="edit-layout">
        <div class="edit-main">
          <WeekCalendar
            :days="days"
            :slots="existingCalSlots"
            :selection="selection"
            selectable
            :clickable="false"
            @update:days="onDaysChange"
            @range-selected="onRangeSelected"
            @selection-remove="onSelectionRemove"
          />
          <el-alert class="drag-tip" type="info" :closable="false" show-icon>
            <template #title>
              日历上的日程块是已有时间段（不能改动）；按住鼠标在空白时间上拖拽即可新增日程，可跨多天、同一天可多段。
              蓝色区块为本次新增，鼠标移上去点右上角 × 可删除。与已有时段重叠的时间无法框选。
            </template>
          </el-alert>
        </div>

        <div class="edit-side glimmer-card">
          <div class="side-section">
            <div class="side-title">已有日程</div>
            <div class="existing-list">
              <span v-for="(p, i) in existingPartitions" :key="i" class="existing-chip">
                {{ fmtDayShort(p.date) }} {{ p.startTime }}-{{ p.endTime }}
                <span class="existing-count">{{ p.slotCount }} 段</span>
              </span>
              <span v-if="!existingPartitions.length" class="existing-empty">暂无日程</span>
            </div>
            <div class="form-tip">共 {{ existingPartitions.length }} 个日程、{{ existingSlots.length }} 个时段</div>
          </div>

          <el-form label-position="top">
            <el-form-item label="新增日程（在左侧日历上拖拽框选，也可手动填写）" required>
              <div class="partition-list">
                <div v-if="!newParts.length" class="partition-empty">
                  还没有新增日程，在左侧日历空白时间上拖拽框选即可
                </div>
                <div v-for="(p, i) in newParts" :key="i" class="partition-row">
                  <div class="part-line">
                    <el-date-picker
                      v-model="p.date"
                      type="date"
                      value-format="YYYY-MM-DD"
                      placeholder="选择日期"
                      class="part-date"
                      @change="v => { if (v && !days.includes(v)) days = daysFrom(v); sortNewParts() }"
                    />
                    <el-time-select
                      v-model="p.startTime"
                      start="00:00"
                      end="23:55"
                      step="00:05"
                      placeholder="开始"
                      class="part-time"
                      @change="sortNewParts"
                    />
                    <span class="time-sep">至</span>
                    <el-time-select
                      v-model="p.endTime"
                      start="00:05"
                      end="24:00"
                      step="00:05"
                      placeholder="结束"
                      class="part-time"
                      @change="sortNewParts"
                    />
                    <el-button text type="danger" title="删除该新增日程" @click="removePartition(i)">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                  <div class="part-line">
                    <el-input v-model="p.location" placeholder="地点（留空沿用场次地点）" class="part-loc" />
                    <el-input-number
                      v-model="p.capacity"
                      class="part-cap"
                      :controls="false"
                      :min="1"
                      :max="20"
                      size="small"
                      :title="`该日程每个时段可同时面试的人数（默认 ${defaultCap} 人）`"
                    />
                    <span class="part-count">
                      {{ Math.max(0, Math.floor((toMin(p.endTime) - toMin(p.startTime)) / slotMinutes)) }} 段
                    </span>
                  </div>
                </div>
              </div>
              <el-button text type="primary" @click="addPartition">
                <el-icon style="margin-right: 4px"><Plus /></el-icon> 添加日程
              </el-button>
            </el-form-item>

            <el-form-item v-if="newSlotRows.length" label="新增时段的各时段人数（可选）">
              <div class="slot-cap-list">
                <div v-for="(r, i) in newSlotRows" :key="r.key" class="slot-cap-row">
                  <span v-if="i === 0 || newSlotRows[i - 1].date !== r.date" class="sc-date">{{ fmtDayShort(r.date) }}</span>
                  <span class="sc-time">{{ r.startTime }} - {{ r.endTime }}</span>
                  <el-input-number
                    class="sc-input"
                    :model-value="slotCaps[r.key] || null"
                    :controls="false"
                    :min="0"
                    :max="20"
                    size="small"
                    :placeholder="String(r.defaultCap)"
                    @update:model-value="v => setSlotCap(r.key, v)"
                  />
                  <span class="sc-hint" :class="{ off: r.overridden }">{{ r.overridden ? '已单独指定' : '默认' }}</span>
                </div>
              </div>
              <div class="form-tip">清空输入框即恢复默认（日程人数 / 场次默认）。</div>
            </el-form-item>

            <div class="preview-box">
              <div class="preview-count">
                将新增 <b>{{ newSlotCount }}</b> 个面试时段 · 共可同时面试 <b>{{ newTotalCapacity }}</b> 人
              </div>
              <div class="form-tip" style="margin-top: 6px">
                场次总时段数将从 {{ existingSlots.length }} 增加到 {{ existingSlots.length + newSlotCount }}
              </div>
            </div>

            <div class="form-actions">
              <el-button size="large" @click="router.push('/admin')">取消</el-button>
              <el-button
                type="primary"
                size="large"
                :loading="submitting"
                :disabled="isDraft || isConfirmed || !newParts.length"
                @click="onSubmit"
              >
                确认追加
              </el-button>
            </div>
          </el-form>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.page-container {
  max-width: 1640px;
  min-height: 300px;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.status-tag {
  margin-left: 6px;
}

.append-tip {
  margin-bottom: 12px;
}

.drag-tip {
  margin-top: 16px;
}

.edit-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 460px;
  gap: 20px;
  align-items: start;
}

.edit-main {
  min-width: 0;
}

@media (min-width: 1100px) and (max-width: 1520px) {
  .edit-main :deep(.cal-inner) {
    min-width: 0;
  }
}

.edit-side {
  padding: 24px;
  position: sticky;
  top: 84px;
}

.side-section {
  margin-bottom: 18px;
}

.side-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--glimmer-text);
  margin-bottom: 8px;
}

.existing-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.existing-chip {
  font-size: 12px;
  padding: 3px 8px;
  border-radius: 8px;
  background: #f2f6fd;
  border: 1px solid #dbe6f7;
  color: #3d5878;
  font-variant-numeric: tabular-nums;
}

.existing-count {
  color: var(--glimmer-text-3);
  margin-left: 2px;
}

.existing-empty {
  font-size: 12px;
  color: var(--glimmer-text-3);
}

.partition-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.partition-empty {
  padding: 14px 12px;
  border: 1px dashed var(--glimmer-border);
  border-radius: 10px;
  background: #fafbfd;
  font-size: 13px;
  color: var(--glimmer-text-3);
  text-align: center;
}

.partition-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border: 1px solid var(--glimmer-border);
  border-radius: 10px;
  background: #fafbfd;
}

.part-line {
  display: flex;
  align-items: center;
  gap: 4px;
}

.partition-row .part-date {
  flex: 1 1 138px;
  min-width: 132px;
}

.partition-row .part-time {
  flex: 0 0 100px;
  width: 100px;
}

.partition-row .part-loc {
  flex: 1;
}

.partition-row .time-sep {
  font-size: 12px;
  color: var(--glimmer-text-3);
}

.partition-row .part-cap {
  flex: 0 0 auto;
  width: 52px;
}

.partition-row .part-cap :deep(.el-input__wrapper) {
  padding: 0 6px;
}

.partition-row .part-cap :deep(.el-input__inner) {
  text-align: center;
}

.part-count {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--glimmer-text-3);
  white-space: nowrap;
}

.slot-cap-list {
  width: 100%;
  max-height: 220px;
  overflow-y: auto;
  border: 1px solid var(--glimmer-border);
  border-radius: 10px;
  background: #fafbfd;
  padding: 4px 0;
}

.slot-cap-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 12px;
  font-size: 12.5px;
}

.slot-cap-row:hover {
  background: #f2f6fd;
}

.sc-date {
  flex: 0 0 96px;
  font-weight: 600;
  color: var(--glimmer-text-2);
}

.sc-time {
  flex: 0 0 104px;
  font-variant-numeric: tabular-nums;
  color: var(--glimmer-text-2);
}

.sc-input {
  flex: 0 0 52px;
}

.sc-input :deep(.el-input__wrapper) {
  padding: 0 6px;
}

.sc-input :deep(.el-input__inner) {
  text-align: center;
}

.sc-hint {
  font-size: 11px;
  color: var(--glimmer-text-3);
}

.sc-hint.off {
  color: var(--glimmer-blue);
}

.form-tip {
  font-size: 12px;
  color: var(--glimmer-text-3);
  line-height: 1.6;
  margin-top: 2px;
}

.preview-box {
  background: var(--glimmer-green-bg);
  border: 1px dashed var(--glimmer-green-2);
  border-radius: 12px;
  padding: 14px 16px;
  margin-bottom: 8px;
}

.preview-count {
  font-size: 14px;
  color: #145a00;
}

.preview-count b {
  font-size: 20px;
  margin: 0 2px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}

@media (max-width: 1099px) {
  .edit-layout {
    grid-template-columns: minmax(0, 1fr);
  }
  .edit-side {
    position: static;
  }
  .edit-main :deep(.cal-scroll) {
    max-height: 46vh;
  }
}

@media (max-width: 768px) {
  .page-container {
    padding: 14px 12px;
  }

  .edit-side {
    padding: 18px 16px;
  }

  .form-actions {
    flex-direction: column;
  }

  .form-actions .el-button {
    width: 100%;
    min-height: 44px;
  }

  .drag-tip {
    margin-top: 12px;
  }
}
</style>
