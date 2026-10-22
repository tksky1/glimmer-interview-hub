<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import WeekCalendar from '@/components/WeekCalendar.vue'
import { sessionDetailApi, createSessionApi, updateSessionApi, publishSessionApi, DIRECTIONS } from '@/apis'

const route = useRoute()
const router = useRouter()
const editId = route.params.id ? Number(route.params.id) : null
const isEdit = computed(() => !!editId)

function today() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function fmtDate(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function daysFrom(start, n = 5) {
  const first = new Date(start + 'T00:00:00')
  return Array.from({ length: n }, (_, i) => {
    const d = new Date(first)
    d.setDate(d.getDate() + i)
    return fmtDate(d)
  })
}

// 一个场次可由多个「日程」组成：不同日期 / 同一天多段，成员可在所有日程里挑 3 个心仪时段
function newPartition(date = today()) {
  return { date, startTime: '09:00', endTime: '12:00', location: '' }
}

const form = ref({
  title: '',
  direction: 'java',
  slotMinutes: 20,
  slotCapacity: 1,
  location: '',
  contact: '',
  contactQQ: '',
  partitions: [newPartition()]
})
const days = ref(daysFrom(today()))
const draftSlots = ref([])
const submitting = ref(false)
// 逐时段容量覆盖：key = `日期 开始时间`，只保存被单独指定过的值
const slotCaps = ref({})
// 场次状态：仅草稿可编辑/发布，已发布后改为只读提示
const status = ref('draft')
const publishNow = ref(true)
const canEdit = computed(() => !isEdit.value || status.value === 'draft')

const durationOptions = [5, 10, 15, 20, 25, 30, 45, 60]

function toMin(hm) {
  const [h, m] = hm.split(':').map(Number)
  return h * 60 + m
}

// 各日程的时段数
const perPartition = computed(() =>
  form.value.partitions.map(p => {
    const diff = toMin(p.endTime || '00:00') - toMin(p.startTime || '00:00')
    return diff > 0 && form.value.slotMinutes ? Math.floor(diff / form.value.slotMinutes) : 0
  })
)

const slotCount = computed(() => perPartition.value.reduce((a, b) => a + b, 0))

// 全部时段（按日期 + 开始时间排序）；容量优先级：逐时段覆盖 > 场次默认
const slotRows = computed(() => {
  const step = Number(form.value.slotMinutes) || 0
  if (!step) return []
  const def = Number(form.value.slotCapacity) || 1
  const rows = []
  for (const p of form.value.partitions) {
    if (!p.date || !p.startTime || !p.endTime) continue
    const start = toMin(p.startTime)
    const n = Math.max(0, Math.floor((toMin(p.endTime) - start) / step))
    for (let i = 0; i < n; i++) {
      const s = start + i * step
      const key = `${p.date} ${fmtMin(s)}`
      const over = Number(slotCaps.value[key]) || 0
      rows.push({
        key,
        date: p.date,
        startTime: fmtMin(s),
        endTime: fmtMin(s + step),
        defaultCap: def,
        cap: over > 0 ? over : def,
        overridden: over > 0
      })
    }
  }
  return rows.sort((a, b) =>
    a.date === b.date ? a.startTime.localeCompare(b.startTime) : a.date.localeCompare(b.date)
  )
})

// 各时段人数之和 = 本场次最多可同时面试的人数
const totalCapacity = computed(() => slotRows.value.reduce((sum, r) => sum + r.cap, 0))

// 某个日程当前的人数：该天所有时段覆盖值一致时返回它，否则 null（混合）
function partitionCapacity(date) {
  const caps = slotRows.value.filter(r => r.date === date && r.overridden).map(r => r.cap)
  if (!caps.length) return null
  return caps.every(c => c === caps[0]) ? caps[0] : null
}

// 日程级输入的占位提示：未设置 -> 默认；各时段不一致 -> 混合
function partitionCapPlaceholder(date) {
  const caps = slotRows.value.filter(r => r.date === date && r.overridden).map(r => r.cap)
  if (!caps.length) return '默认'
  return caps.every(c => c === caps[0]) ? String(caps[0]) : '混合'
}

// 批量设置某个日程所有时段的人数（清空/等于默认则恢复默认）
function setPartitionCapacity(date, v) {
  const n = Number(v) || 0
  const def = Number(form.value.slotCapacity) || 1
  for (const r of slotRows.value) {
    if (r.date !== date) continue
    if (n > 0 && n !== def) slotCaps.value[r.key] = n
    else delete slotCaps.value[r.key]
  }
}

function setSlotCap(key, v) {
  const n = Number(v)
  if (n > 0) slotCaps.value[key] = n
  else delete slotCaps.value[key]
}

function fmtDayShort(v) {
  const dt = new Date(`${v}T00:00:00`)
  if (Number.isNaN(dt.getTime())) return v || ''
  return `${dt.getMonth() + 1}月${dt.getDate()}日 周${'日一二三四五六'[dt.getDay()]}`
}

// 预览也按日期 + 开始时间全局排序，最多列出前 6 个
const previewSlots = computed(() => {
  const step = Number(form.value.slotMinutes) || 0
  if (!step) return []
  const all = []
  for (const p of form.value.partitions) {
    if (!p.date || !p.startTime || !p.endTime) continue
    const start = toMin(p.startTime)
    const n = Math.max(0, Math.floor((toMin(p.endTime) - start) / step))
    for (let i = 0; i < n; i++) all.push({ date: p.date, min: start + i * step })
  }
  all.sort((a, b) => (a.date === b.date ? a.min - b.min : a.date.localeCompare(b.date)))
  const shown = all.slice(0, 6).map(x => `${x.date.slice(5)} ${fmtMin(x.min)}`)
  if (all.length > shown.length) shown.push(`… 共 ${all.length} 个`)
  return shown
})

// 日历上持久显示的已框选范围（多个日程）
const selection = computed(() => form.value.partitions.map(p => ({
  date: p.date,
  startTime: p.startTime,
  endTime: p.endTime
})))

// 日程始终按「日期 -> 开始时间」排序，右侧列表与时段预览都跟着有序
function sortPartitions() {
  form.value.partitions.sort((a, b) =>
    a.date === b.date
      ? toMin(a.startTime || '00:00') - toMin(b.startTime || '00:00')
      : (a.date || '').localeCompare(b.date || '')
  )
}

function addPartition() {
  const list = form.value.partitions
  const last = list[list.length - 1]
  if (!last) {
    list.push(newPartition(days.value[0] || today()))
  } else {
    const next = new Date(last.date + 'T00:00:00')
    next.setDate(next.getDate() + 1)
    list.push(newPartition(fmtDate(next)))
  }
  sortPartitions()
}

function removePartition(i) {
  form.value.partitions.splice(i, 1)
}

// 在日历上点 × 删除某个已框选日程
function onSelectionRemove({ date, startTime, endTime }) {
  const i = form.value.partitions.findIndex(
    p => p.date === date && p.startTime === startTime && p.endTime === endTime
  )
  if (i >= 0) form.value.partitions.splice(i, 1)
}

function fmtMin(m) {
  return `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`
}

// 在日历上拖拽：同一天可以框选多段；与新框选冲突的旧日程会被自动裁剪/取消
function onRangeSelected({ date, startTime, endTime }) {
  const s = toMin(startTime)
  const e = toMin(endTime)
  if (e - s < 5) return
  // 完全相同的区间不重复添加（也不误删）
  const dup = form.value.partitions.some(
    p => p.date === date && toMin(p.startTime || '00:00') === s && toMin(p.endTime || '00:00') === e
  )
  if (dup) return

  const next = []
  for (const p of form.value.partitions) {
    if (p.date !== date) {
      next.push(p)
      continue
    }
    const ps = toMin(p.startTime || '00:00')
    const pe = toMin(p.endTime || '00:00')
    if (ps >= e || pe <= s) {
      next.push(p) // 不相交，原样保留
      continue
    }
    // 有重叠：以新框选为准，保留旧日程未被覆盖的前/后片段
    if (ps < s) next.push({ ...p, endTime: fmtMin(s) })
    if (pe > e) next.push({ ...p, startTime: fmtMin(e) })
  }
  next.push({ date, startTime, endTime, location: '' })
  form.value.partitions = next
  sortPartitions()
  if (!days.value.includes(date)) {
    days.value = daysFrom(date)
  }
}

// 同一天多段时检查重叠
function hasOverlap() {
  const byDay = {}
  for (const p of form.value.partitions) {
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
  if (!form.value.title.trim()) {
    ElMessage.warning('请填写场次标题')
    return false
  }
  if (!form.value.location.trim()) {
    ElMessage.warning('请填写面试地点')
    return false
  }
  if (!form.value.contact.trim()) {
    ElMessage.warning('请填写联系人')
    return false
  }
  if (!form.value.contactQQ.trim()) {
    ElMessage.warning('请填写联系人 QQ')
    return false
  }
  if (!form.value.partitions.length) {
    ElMessage.warning('请在日历上框选至少一个日程')
    return false
  }
  for (const p of form.value.partitions) {
    if (!p.date) {
      ElMessage.warning('请为每个日程选择日期')
      return false
    }
    if (toMin(p.endTime || '00:00') <= toMin(p.startTime || '00:00')) {
      ElMessage.warning(`${p.date} 的结束时间必须晚于开始时间`)
      return false
    }
  }
  if (hasOverlap()) {
    ElMessage.warning('同一天的日程时间段不能重叠')
    return false
  }
  if (slotCount.value < 1) {
    ElMessage.warning('所选时间段不足以容纳一场面试')
    return false
  }
  return true
}

async function onSubmit() {
  if (!validate()) return
  submitting.value = true
  try {
    const ps = form.value.partitions
      .filter(p => p.date && p.startTime && p.endTime)
      .map(p => ({
        date: p.date,
        startTime: p.startTime,
        endTime: p.endTime,
        location: (p.location || '').trim() || form.value.location.trim()
      }))
      .sort((a, b) => (a.date === b.date ? a.startTime.localeCompare(b.startTime) : a.date.localeCompare(b.date)))
    const payload = {
      title: form.value.title.trim(),
      direction: form.value.direction,
      date: ps[0].date,
      startTime: ps[0].startTime,
      endTime: ps[0].endTime,
      slotMinutes: form.value.slotMinutes,
      slotCapacity: Number(form.value.slotCapacity) || 1,
      location: form.value.location.trim(),
      contact: form.value.contact.trim(),
      contactQQ: form.value.contactQQ.trim(),
      partitions: ps,
      // 只提交被单独指定过的时段容量（清空的走默认）
      slotCapacities: slotRows.value
        .filter(r => r.overridden)
        .map(r => ({ date: r.date, startTime: r.startTime, capacity: r.cap }))
    }
    let sid = editId
    if (isEdit.value) {
      await updateSessionApi(editId, payload)
    } else {
      const res = await createSessionApi(payload)
      sid = res.data?.session?.id
    }
    if (publishNow.value && canEdit.value && sid) {
      await publishSessionApi(sid)
      ElMessage.success(isEdit.value ? '修改已保存并发布，成员现在可以预约' : '场次已创建并发布，成员现在可以预约')
    } else {
      ElMessage.success(isEdit.value ? '修改已保存（仍为草稿）' : '场次已保存为草稿，可在列表中发布')
    }
    router.push('/admin')
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  if (!isEdit.value) return
  const res = await sessionDetailApi(editId)
  const s = res.data.session
  status.value = s.status || 'draft'
  const parts = (res.data.partitions || []).length
    ? res.data.partitions
    : [{ date: s.date, startTime: s.startTime, endTime: s.endTime, location: s.location }]
  form.value = {
    title: s.title,
    direction: s.direction || 'java',
    slotMinutes: s.slotMinutes,
    slotCapacity: s.slotCapacity || 1,
    location: s.location,
    contact: s.contact || '',
    contactQQ: s.contactQQ || '',
    partitions: parts.map(p => ({
      date: p.date,
      startTime: p.startTime,
      endTime: p.endTime,
      location: p.location || ''
    }))
  }
  days.value = daysFrom(parts[0].date)
  // 逐时段容量覆盖：直接用接口返回的显式覆盖表，避免从时段容量反推出错
  slotCaps.value = {}
  for (const [k, v] of Object.entries(res.data.slotCapacities || {})) {
    if (Number(v) > 0) slotCaps.value[k] = Number(v)
  }
  draftSlots.value = (res.data.slots || []).map(sl => ({
    id: sl.id,
    date: sl.date || s.date,
    startTime: sl.startTime,
    endTime: sl.endTime,
    location: sl.location,
    firstChoiceCount: sl.firstChoiceCount || 0,
    myRank: sl.myRank || 0,
    myConfirmed: sl.myConfirmed || false
  }))
})
</script>

<template>
  <div class="page-container">
    <div class="page-head">
      <div>
        <h2 class="page-title">
          <el-icon><Calendar /></el-icon> {{ isEdit ? '编辑面试场次' : '新建面试场次' }}
        </h2>
        <p class="page-subtitle">
          在日历上拖拽框选面试时间段（可跨多天、每天可多段），设置方向、每场时长与地点，系统自动切分为面试时段。
          成员可在所有日程里挑 3 个心仪时间，最终只确定 1 个面试时段。
        </p>
      </div>
      <el-button @click="router.push('/admin')">
        <el-icon style="margin-right: 4px"><ArrowLeft /></el-icon> 返回列表
      </el-button>
    </div>

    <div class="edit-layout">
      <div class="edit-main">
        <WeekCalendar
          :days="days"
          :slots="draftSlots"
          :selection="selection"
          selectable
          :clickable="false"
          @update:days="onDaysChange"
          @range-selected="onRangeSelected"
          @selection-remove="onSelectionRemove"
        />
        <el-alert class="drag-tip" type="info" :closable="false" show-icon>
          <template #title>
            按住鼠标拖拽即可框选时间段（5 分钟精度，贴近上下边缘会自动滚动）：同一天可以框选多段，换一天就是新增日程。
            蓝色区块是已框选的日程，鼠标移上去点右上角 × 可删除；与新框选冲突的旧片段会自动裁掉。
          </template>
        </el-alert>
      </div>

      <div class="edit-side glimmer-card">
        <el-form label-position="top" :model="form">
          <el-form-item label="场次标题" required>
            <el-input v-model="form.title" placeholder="例如：2026 招新一面（Web 方向）" maxlength="50" />
          </el-form-item>

          <el-form-item label="面试方向" required>
            <el-select v-model="form.direction" style="width: 100%">
              <el-option v-for="d in DIRECTIONS" :key="d.value" :label="d.label" :value="d.value" />
            </el-select>
          </el-form-item>

          <el-form-item label="面试日程（可跨多天，同一天可多段）" required>
            <div class="partition-list">
              <div v-if="!form.partitions.length" class="partition-empty">
                还没有日程，在左侧日历上拖拽框选即可添加
              </div>
              <div v-for="(p, i) in form.partitions" :key="i" class="partition-row">
                <div class="part-line">
                  <el-date-picker
                    v-model="p.date"
                    type="date"
                    value-format="YYYY-MM-DD"
                    placeholder="选择日期"
                    class="part-date"
                    @change="v => { if (v && !days.includes(v)) days = daysFrom(v); sortPartitions() }"
                  />
                  <el-time-select
                    v-model="p.startTime"
                    start="00:00"
                    end="23:55"
                    step="00:05"
                    placeholder="开始"
                    class="part-time"
                    @change="sortPartitions"
                  />
                  <span class="time-sep">至</span>
                  <el-time-select
                    v-model="p.endTime"
                    start="00:05"
                    end="24:00"
                    step="00:05"
                    placeholder="结束"
                    class="part-time"
                    @change="sortPartitions"
                  />
                  <el-button text type="danger" title="删除该日程" @click="removePartition(i)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
                <div class="part-line">
                  <el-input v-model="p.location" placeholder="地点（留空沿用场次地点）" class="part-loc" />
                  <el-input-number
                    class="part-cap"
                    :model-value="partitionCapacity(p.date)"
                    :controls="false"
                    :min="0"
                    :max="20"
                    size="small"
                    :placeholder="partitionCapPlaceholder(p.date)"
                    title="一次设置该日程所有时段的人数，留空用场次默认；下面还可逐个时段微调"
                    @update:model-value="v => setPartitionCapacity(p.date, v)"
                  />
                  <span class="part-count">{{ perPartition[i] }} 段</span>
                </div>
              </div>
            </div>
            <el-button text type="primary" @click="addPartition">
              <el-icon style="margin-right: 4px"><Plus /></el-icon> 添加日程
            </el-button>
          </el-form-item>

          <el-form-item label="每场面试时长" required>
            <el-radio-group v-model="form.slotMinutes">
              <el-radio-button v-for="m in durationOptions" :key="m" :value="m">{{ m }} 分钟</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="新时段默认人数" required>
            <div class="capacity-row">
              <el-input-number
                v-model="form.slotCapacity"
                :controls="false"
                :min="1"
                :max="20"
                size="default"
                class="capacity-input"
              />
              <span class="capacity-unit">人</span>
            </div>
            <div class="form-tip">
              新生成的时段默认允许多人同时面试；下面的「各时段人数」可以逐个时段单独指定，日程上的设置是中间层默认值。
            </div>
          </el-form-item>

          <!-- 逐时段容量：优先级最高，覆盖日程与场次设置 -->
          <el-form-item v-if="slotRows.length" label="各时段人数（逐个指定）">
            <div class="slot-cap-list">
              <div v-for="(r, i) in slotRows" :key="r.key" class="slot-cap-row">
                <span v-if="i === 0 || slotRows[i - 1].date !== r.date" class="sc-date">{{ fmtDayShort(r.date) }}</span>
                <span class="sc-time">{{ r.startTime }} - {{ r.endTime }}</span>
                <el-input-number
                  class="sc-input"
                  :model-value="slotCaps[r.key] || null"
                  :controls="false"
                  :min="0"
                  :max="20"
                  size="small"
                  :placeholder="String(r.defaultCap)"
                  title="该时段可同时面试的人数，清空则用默认值"
                  @update:model-value="v => setSlotCap(r.key, v)"
                />
                <span class="sc-hint" :class="{ off: r.overridden }">{{ r.overridden ? '已单独指定' : '默认' }}</span>
              </div>
            </div>
            <div class="form-tip">清空输入框即恢复默认；共 {{ slotRows.length }} 个时段、可同时面试 {{ totalCapacity }} 人。</div>
          </el-form-item>

          <el-form-item label="面试地点" required>
            <el-input v-model="form.location" placeholder="例如：教学楼三-301 / 线上腾讯会议" maxlength="50" />
          </el-form-item>

          <div class="contact-row">
            <el-form-item label="联系人" required>
              <el-input v-model="form.contact" placeholder="联系人姓名" maxlength="20" />
            </el-form-item>
            <el-form-item label="联系人 QQ" required>
              <el-input v-model="form.contactQQ" placeholder="QQ 号" maxlength="15" />
            </el-form-item>
          </div>

          <div class="preview-box">
            <div class="preview-count">
              将自动创建 <b>{{ slotCount }}</b> 个面试时段 · 共可同时面试 <b>{{ totalCapacity }}</b> 人
              <span v-if="slotRows.length && slotRows.some(r => r.overridden)" class="preview-note">（含单独指定的时段人数）</span>
            </div>
            <div class="preview-slots">
              <el-tag v-for="p in previewSlots" :key="p" size="small" effect="plain" class="preview-tag">{{ p }}</el-tag>
            </div>
          </div>

          <el-alert
            v-if="!canEdit"
            class="readonly-tip"
            type="warning"
            :closable="false"
            show-icon
            title="该场次已发布，日程不可再修改。如需调整请联系负责人先删除重建。"
          />

          <div v-else class="publish-row">
            <el-checkbox v-model="publishNow">保存后立即发布</el-checkbox>
            <span class="publish-hint">发布后成员即可在日历上看到并预约这些时段</span>
          </div>

          <div class="form-actions">
            <el-button size="large" @click="router.push('/admin')">取消</el-button>
            <el-button v-if="canEdit" type="primary" size="large" :loading="submitting" @click="onSubmit">
              {{ (publishNow ? (isEdit ? '保存并发布' : '创建并发布') : (isEdit ? '保存草稿' : '保存为草稿')) }}
            </el-button>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 编辑页需要放置日历 + 设置面板，容器比其他页面更宽 */
.page-container {
  max-width: 1640px;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
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

/* 并排时（1100-1520）让日历按列宽自适应收缩，避免内层横向滚动条 */
@media (min-width: 1100px) and (max-width: 1520px) {
  .edit-main :deep(.cal-inner) {
    min-width: 0;
  }
}

.drag-tip {
  margin-top: 16px;
}

.edit-side {
  padding: 24px;
  position: sticky;
  top: 84px;
}

.time-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.time-row > .el-form-item {
  flex: 1;
}

.time-sep {
  color: var(--glimmer-text-3);
  padding-top: 26px;
}

.partition-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
}

.capacity-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.capacity-input {
  width: 88px;
}

.capacity-unit {
  font-size: 13px;
  color: var(--glimmer-text-3);
}

.form-tip {
  font-size: 12px;
  color: var(--glimmer-text-3);
  line-height: 1.6;
  margin-top: 2px;
}

.slot-cap-list {
  width: 100%;
  max-height: 240px;
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

.preview-note {
  font-size: 12px;
  color: var(--glimmer-text-3);
  font-weight: 400;
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

.publish-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 10px;
  padding: 10px 14px;
  margin-bottom: 16px;
  border: 1px solid var(--glimmer-border);
  border-radius: 10px;
  background: #f7faff;
}

.publish-hint {
  font-size: 12px;
  color: var(--glimmer-text-3);
}

.readonly-tip {
  margin-bottom: 16px;
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

/* 100px 才能完整显示 09:00（Element Plus 选择器右侧还有箭头图标） */
.partition-row .part-time {
  flex: 0 0 100px;
  width: 100px;
}

.partition-row .part-loc {
  flex: 1;
}

.partition-row .time-sep {
  padding-top: 0;
  font-size: 12px;
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

.contact-row {
  display: flex;
  gap: 12px;
}

.contact-row > .el-form-item {
  flex: 1;
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

.preview-slots {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.preview-tag {
  font-variant-numeric: tabular-nums;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}

/* 窄屏（平板/小笔记本）改为上下堆叠，并把日历压矮一点，方便够到表单 */
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

/* ===== 手机端 ===== */
@media (max-width: 768px) {
  .page-container {
    padding: 14px 12px;
  }

  .edit-side {
    padding: 18px 16px;
  }

  .time-row,
  .contact-row {
    flex-direction: column;
    gap: 0;
  }

  .time-sep {
    display: none;
  }

  .time-row .el-form-item,
  .contact-row .el-form-item {
    width: 100%;
  }

  .form-actions {
    flex-direction: column;
  }

  .form-actions .el-button {
    width: 100%;
    min-height: 44px;
  }

  .edit-side :deep(.el-radio-button__inner) {
    padding: 8px 9px;
  }

  .drag-tip {
    margin-top: 12px;
  }
}
</style>
