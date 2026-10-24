<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'

const props = defineProps({
  // 日期数组（YYYY-MM-DD），至少 3 天，按天分列渲染
  days: { type: Array, required: true },
  // 时段数组，需带 date 字段；支持 myRank（已保存）/ pickedRank（本地未保存）/ myConfirmed / occupant
  slots: { type: Array, default: () => [] },
  // 拖拽框选模式（管理员建场次）
  selectable: { type: Boolean, default: false },
  // 是否允许点击时段（成员预约）
  clickable: { type: Boolean, default: true },
  // 管理员当前已框选的范围（持久高亮显示）：{ date, startTime, endTime } 或它们的数组
  selection: { type: [Object, Array], default: null }
})

const emit = defineEmits(['update:days', 'range-selected', 'slot-click', 'selection-remove'])

const PX_PER_MIN = 1.45
// 左侧时间轴宽度 / 表头高度：时间轴刻度要跳过表头，才能和右侧网格对齐
const AXIS_WIDTH = 48
const HEAD_H = 44
const WEEK_CN = ['日', '一', '二', '三', '四', '五', '六']
const scrollRef = ref(null)

// ---------- 窄屏适配：单日翻阅 ----------
const MOBILE_QUERY = '(max-width: 768px)'
const isMobile = ref(false)
let mq = null
function onMQChange(e) {
  isMobile.value = e.matches
}

// 单列宽度：决定时段行是否只显示开始时间（窄列才省略结束时间）
const rootRef = ref(null)
const colWidth = ref(999)
let ro = null

onMounted(() => {
  mq = window.matchMedia(MOBILE_QUERY)
  isMobile.value = mq.matches
  mq.addEventListener('change', onMQChange)

  ro = new ResizeObserver(entries => {
    const w = entries[0]?.contentRect.width || 0
    colWidth.value = w ? Math.max(0, (w - AXIS_WIDTH) / Math.max(1, visibleDays.value.length)) : 999
  })
  if (rootRef.value) ro.observe(rootRef.value)
})
onUnmounted(() => {
  mq?.removeEventListener('change', onMQChange)
  ro?.disconnect()
})
// 窄屏只渲染中间一天（前后翻页仍可浏览全部日期）
const visibleDays = computed(() => {
  if (!isMobile.value) return props.days
  if (!props.days.length) return []
  const mid = Math.floor(props.days.length / 2)
  return [props.days[mid] || props.days[0]]
})

// ---------- 时间工具 ----------
function toMin(hm) {
  const [h, m] = hm.split(':').map(Number)
  return h * 60 + m
}
function fmt(min) {
  const h = Math.floor(min / 60)
  const m = min % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}
function snap(min) {
  return Math.round(min / 5) * 5
}
function parseDate(v) {
  // 兼容 Date 实例与 YYYY-MM-DD 字符串，避免二次解析得到 Invalid Date
  if (v instanceof Date) return new Date(v)
  return new Date(`${v}T00:00:00`)
}
function isValidDate(d) {
  return d instanceof Date && !Number.isNaN(d.getTime())
}
function fmtDate(d) {
  if (!isValidDate(d)) return ''
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
// 安全格式化：无效日期返回空串，不产生 NaN月NaN日
function fmtMD(d) {
  if (!isValidDate(d)) return ''
  return `${d.getMonth() + 1}月${d.getDate()}日`
}

// ---------- 日期导航 ----------
function shiftDay(delta) {
  const base = parseDate(props.days[0])
  if (!isValidDate(base)) return
  const first = new Date(base)
  first.setDate(first.getDate() + delta)
  const next = []
  for (let i = 0; i < props.days.length; i++) {
    // 以 first 为基准逐日偏移，日/月/年/闰年跨月由 Date 自动进位
    const d = new Date(first)
    d.setDate(first.getDate() + i)
    next.push(fmtDate(d))
  }
  emit('update:days', next)
}
function goToday() {
  const today = new Date()
  const start = fmtDate(today)
  if (!start) return
  const days = [start]
  for (let i = 1; i < Math.max(3, props.days.length); i++) {
    const d = new Date(today)
    d.setDate(d.getDate() + i)
    days.push(fmtDate(d))
  }
  emit('update:days', days)
}
const rangeLabel = computed(() => {
  if (!props.days.length) return ''
  if (isMobile.value) {
    const d = parseDate(visibleDays.value[0])
    if (!isValidDate(d)) return ''
    return `${fmtMD(d)} 周${WEEK_CN[d.getDay()]}`
  }
  const a = parseDate(props.days[0])
  const b = parseDate(props.days[props.days.length - 1])
  if (!isValidDate(a) || !isValidDate(b)) return ''
  return `${fmtMD(a)} - ${fmtMD(b)}`
})

// 窄屏日期圆点指示（当前显示 3 天窗口中的第几天）
const dayDots = computed(() => {
  if (!isMobile.value) return []
  const cur = visibleDays.value[0]
  return props.days.map((d, i) => ({ date: d, active: d === cur, index: i }))
})

// ---------- 可见时间范围（按内容自适应） ----------
const dayStart = computed(() => {
  if (!props.slots.length) return 8 * 60
  const minSlot = Math.min(...props.slots.map(s => toMin(s.startTime)))
  return Math.max(0, Math.floor((minSlot - 30) / 60) * 60)
})
const dayEnd = computed(() => {
  if (!props.slots.length) return 22 * 60
  const maxSlot = Math.max(...props.slots.map(s => toMin(s.endTime)))
  return Math.min(24 * 60, Math.ceil((maxSlot + 30) / 60) * 60)
})
const totalMin = computed(() => dayEnd.value - dayStart.value)
const gridHeight = computed(() => totalMin.value * PX_PER_MIN)

const ticks = computed(() => {
  const list = []
  for (let m = dayStart.value; m <= dayEnd.value; m += 30) {
    list.push({ min: m, major: m % 60 === 0, label: m % 60 === 0 ? fmt(m) : '' })
  }
  return list
})

// 底部那条网格线会贴着容器边缘多出 1px（触发滚动条），不画它
const gridLines = computed(() => ticks.value.filter(t => t.min < dayEnd.value))

// ---------- 按天分组 ----------
const slotsByDay = computed(() => {
  const map = new Map()
  for (const d of props.days) map.set(d, [])
  for (const s of props.slots) {
    if (!map.has(s.date)) map.set(s.date, [])
    map.get(s.date).push(s)
  }
  // 每天内按开始时间排序
  for (const list of map.values()) {
    list.sort((a, b) => toMin(a.startTime) - toMin(b.startTime))
  }
  return map
})

function dayLabel(d) {
  const date = parseDate(d)
  const today = fmtDate(new Date())
  const isToday = d === today
  if (!isValidDate(date)) return { text: '', week: '', isToday: false }
  return {
    text: fmtMD(date),
    week: `周${WEEK_CN[date.getDay()]}`,
    isToday
  }
}

// ---------- 拖拽框选 ----------
const dragging = ref(false)
const dragDay = ref('')
const dragStart = ref(0)
const dragCur = ref(0)

function posToMin(dayEl, clientY) {
  const rect = dayEl.getBoundingClientRect()
  const y = clientY - rect.top
  return snap(Math.min(dayEnd.value, Math.max(dayStart.value, dayStart.value + y / PX_PER_MIN)))
}

// 拖拽经过容器边缘时自动滚动，保证能选到可视区外的时间
function autoScroll(clientY) {
  const el = scrollRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const edge = 48
  if (clientY < rect.top + edge) el.scrollTop -= 14
  else if (clientY > rect.bottom - edge) el.scrollTop += 14
}

function onMouseDown(e, day, dayEl) {
  if (!props.selectable || e.button !== 0) return
  dragging.value = true
  dragDay.value = day
  dragStart.value = posToMin(dayEl, e.clientY)
  dragCur.value = dragStart.value
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function onMouseMove(e) {
  if (!dragging.value) return
  const dayEl = document.querySelector(`[data-day="${dragDay.value}"]`)
  if (dayEl) dragCur.value = posToMin(dayEl, e.clientY)
  autoScroll(e.clientY)
}

function onMouseUp() {
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
  if (!dragging.value) return
  dragging.value = false
  const lo = Math.min(dragStart.value, dragCur.value)
  const hi = Math.max(dragStart.value, dragCur.value)
  if (hi - lo >= 5) {
    emit('range-selected', { date: dragDay.value, startTime: fmt(lo), endTime: fmt(hi) })
  }
}

const dragStyle = computed(() => {
  const lo = Math.min(dragStart.value, dragCur.value)
  const hi = Math.max(dragStart.value, dragCur.value)
  return {
    top: (lo - dayStart.value) * PX_PER_MIN + 'px',
    height: Math.max(5, (hi - lo) * PX_PER_MIN) + 'px'
  }
})

// ---------- 时段块 ----------
function slotHeight(slot) {
  return (toMin(slot.endTime) - toMin(slot.startTime)) * PX_PER_MIN
}

// 同一场次连续的时段合并成一整块，内部用行分隔（避免一堆小方块堆叠）
function buildSlotGroups(list) {
  const groups = []
  for (const s of list) {
    const last = groups[groups.length - 1]
    // 容量不同的时段不合并成一块：否则「可多人」的虚线框会把整段都框住，看不出来到底是哪些时段
    if (last && last.sessionId === s.sessionId && last.endTime === s.startTime &&
      capOf(last.slots[last.slots.length - 1]) === capOf(s)) {
      last.slots.push(s)
      last.endTime = s.endTime
      if (capOf(s) > 1) last.multi = true
    } else {
      groups.push({
        key: `${s.sessionId || 's'}-${s.date || ''}-${s.startTime}-${s.endTime}`,
        sessionId: s.sessionId,
        startTime: s.startTime,
        endTime: s.endTime,
        multi: capOf(s) > 1, // 含「可多人同时面试」的时段
        slots: [s]
      })
    }
  }
  return groups
}

const slotGroupsByDay = computed(() => {
  const map = new Map()
  for (const [day, list] of slotsByDay.value) {
    map.set(day, buildSlotGroups(list))
  }
  return map
})

function groupStyle(g) {
  const top = (toMin(g.startTime) - dayStart.value) * PX_PER_MIN
  const height = (toMin(g.endTime) - toMin(g.startTime)) * PX_PER_MIN
  return { top: top + 'px', height: height + 'px' }
}

function rowStyle(slot) {
  return { height: Math.max(14, slotHeight(slot) - 1) + 'px' }
}

// 行高不足（如 30 分钟一场）时的紧凑排版
function isCompact(slot) {
  return slotHeight(slot) < 40
}

// 窄行里也尽量显示结束时间，只有列真的很窄（手机）才只剩开始时间
function showEndTime(slot) {
  return !isCompact(slot) || colWidth.value >= 130
}

function rankOf(slot) {
  return slot.pickedRank || slot.myRank || 0
}

// 时段容量（可同时面试人数）与已占用人数
function capOf(slot) {
  const c = Number(slot.capacity)
  return c > 0 ? c : 1
}
function occOf(slot) {
  const n = Number(slot.occupied)
  return n > 0 ? n : 0
}
// 是否满员（只用于徽章配色与提示，不阻止选择：名额超出交给分配算法协调）
function isFull(slot) {
  if (slot.myConfirmed || rankOf(slot) > 0) return false
  return capOf(slot) > 1 && occOf(slot) >= capOf(slot)
}

// 容量徽章状态：空 / 有人占 / 满员
function capState(slot) {
  if (occOf(slot) >= capOf(slot)) return 'full'
  if (occOf(slot) > 0) return 'partial'
  return 'free'
}

function slotClass(slot) {
  if (slot.myConfirmed) return 'slot-confirmed'
  const r = rankOf(slot)
  if (r === 1) return 'slot-rank1'
  if (r === 2) return 'slot-rank2'
  if (r === 3) return 'slot-rank3'
  if (slot.occupant) return 'slot-taken'
  return 'slot-available'
}

function onSlotClick(slot) {
  if (!props.clickable || props.selectable) return
  emit('slot-click', slot)
}

// 管理员框选：支持单个（对象）或多个（数组）日程
const selectionList = computed(() => {
  const v = props.selection
  if (!v) return []
  return Array.isArray(v) ? v.filter(x => x && x.date) : [v]
})
function selectionsOf(day) {
  return selectionList.value.filter(s => s.date === day)
}
function selectionStyle(sel) {
  const top = (toMin(sel.startTime) - dayStart.value) * PX_PER_MIN
  const height = (toMin(sel.endTime) - toMin(sel.startTime)) * PX_PER_MIN
  return { top: top + 'px', height: Math.max(5, height) + 'px' }
}
</script>

<template>
  <div class="week-calendar" ref="rootRef">
    <div class="cal-header">
      <div class="cal-date">
        <el-button-group>
          <el-button size="small" @click="shiftDay(-1)">
            <el-icon><ArrowLeft /></el-icon>
          </el-button>
          <el-button size="small" @click="shiftDay(1)">
            <el-icon><ArrowRight /></el-icon>
          </el-button>
        </el-button-group>
        <el-button size="small" text @click="goToday">今天</el-button>
        <span class="cal-date-label">{{ rangeLabel }}</span>
        <span v-if="isMobile" class="day-dots">
          <i v-for="d in dayDots" :key="d.date" :class="{ active: d.active }"></i>
        </span>
      </div>
      <div class="cal-legend">
        <span v-if="selectable" class="legend-item"><i class="dot drag"></i>拖拽框选时间段</span>
        <span class="legend-item"><i class="dot avail"></i>可预约</span>
        <span class="legend-item"><i class="dot rank"></i>我的心仪时间</span>
        <span class="legend-item"><i class="dot confirm"></i>已确认</span>
        <span class="legend-item"><i class="dot multi"></i>虚线框 = 可多人同时面试</span>
      </div>
    </div>

    <div class="cal-scroll" ref="scrollRef">
      <div class="cal-inner" :style="{ height: gridHeight + HEAD_H + 'px' }">
        <!-- 共享时间轴 -->
        <div class="cal-axis">
          <div class="axis-corner"></div>
          <div v-for="t in ticks" :key="t.min" class="axis-tick" :style="{ top: HEAD_H + (t.min - dayStart) * PX_PER_MIN + 'px' }">
            <span v-if="t.major" class="axis-label">{{ t.label }}</span>
          </div>
        </div>

        <!-- 每天一列（窄屏仅渲染中间一天） -->
        <div v-for="day in visibleDays" :key="day" class="cal-day">
          <div class="day-head" :class="{ today: dayLabel(day).isToday }">
            <span class="day-text">{{ dayLabel(day).text }}</span>
            <span class="day-week">{{ dayLabel(day).week }}</span>
          </div>
          <div
            class="day-grid"
            :class="{ selectable }"
            :data-day="day"
            :style="{ height: gridHeight + 'px' }"
            @mousedown="onMouseDown($event, day, $event.currentTarget)"
          >
            <div v-for="t in gridLines" :key="'l' + t.min" class="grid-line" :class="{ major: t.major }" :style="{ top: (t.min - dayStart) * PX_PER_MIN + 'px' }"></div>

            <!-- 管理员已框选范围（持久显示，可跨多个日程，悬浮可点 × 删除） -->
            <div
              v-for="(sel, si) in selectionsOf(day)"
              :key="'s' + si"
              class="sel-preview"
              :class="{ removable: selectable }"
              :style="selectionStyle(sel)"
            >
              <span>{{ sel.startTime }} - {{ sel.endTime }}</span>
              <button
                v-if="selectable"
                type="button"
                class="sel-del"
                title="删除这个日程"
                @mousedown.stop.prevent
                @click.stop="emit('selection-remove', { date: sel.date, startTime: sel.startTime, endTime: sel.endTime })"
              >×</button>
            </div>

            <!-- 时段：同一场次连续时段合并为一整块，内部按行细分 -->
            <div
              v-for="g in slotGroupsByDay.get(day) || []"
              :key="g.key"
              class="slot-group"
              :class="{ 'multi-cap': g.multi }"
              :style="groupStyle(g)"
            >
              <div
                v-for="slot in g.slots"
                :key="slot.id"
                class="slot-row"
                :class="[slotClass(slot), { clickable: clickable && !selectable, compact: isCompact(slot) }]"
                :style="rowStyle(slot)"
                :title="`${slot.startTime} - ${slot.endTime}${slot.location ? ' · ' + slot.location : ''}`"
                @click.stop="onSlotClick(slot)"
              >
                <span class="slot-time">
                  {{ slot.startTime }}<span v-if="showEndTime(slot)" class="slot-end">–{{ slot.endTime }}</span>
                </span>
                <!-- 占用者/地点不在日历里展示（鼠标悬浮 title 里仍可查看），节省行宽 -->
                <span v-if="slot.occupant" class="slot-occupant">{{ slot.occupant }}</span>
                <span class="slot-badges">
                  <span v-if="!isCompact(slot) && rankOf(slot)" class="badge rank-badge">心仪{{ ['一', '二', '三'][rankOf(slot) - 1] }}</span>
                  <span v-if="!isCompact(slot) && slot.myConfirmed" class="badge confirm-badge">已确认</span>
                  <!-- 热度徽章在窄行里也保留，方便一眼看出竞争程度 -->
                  <span v-if="slot.firstChoiceCount > 0" class="badge hot-badge">{{ slot.firstChoiceCount }} 人首选</span>
                  <!-- 可多人同时面试的时段：始终显示 已占用/容量，满员变红（仍可的心仪，算法协调） -->
                  <span
                    v-if="capOf(slot) > 1"
                    class="badge cap-badge"
                    :class="capState(slot)"
                    :title="isFull(slot) ? '名额已满：仍可选，分配时会按心仪顺序协调到其他时段' : '已占用 / 可同时面试人数'"
                  >
                    {{ capOf(slot) }} 人 · {{ occOf(slot) }}/{{ capOf(slot) }}
                  </span>
                </span>
              </div>
            </div>

            <!-- 拖拽预览 -->
            <div v-if="dragging && dragDay === day" class="drag-preview" :style="dragStyle">
              <span>{{ fmt(Math.min(dragStart, dragCur)) }} - {{ fmt(Math.max(dragStart, dragCur)) }}</span>
            </div>

            <div v-if="!(slotsByDay.get(day) || []).length" class="day-empty">
              {{ selectable ? '拖拽框选当天时间段' : '当天暂无日程' }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.week-calendar {
  background: var(--glimmer-card);
  border-radius: 16px;
  box-shadow: var(--glimmer-shadow);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.cal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--glimmer-border);
  flex-wrap: wrap;
  gap: 8px;
}

.cal-date {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cal-date-label {
  font-size: 16px;
  font-weight: 600;
  color: var(--glimmer-text);
}

.cal-legend {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--glimmer-text-2);
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
  display: inline-block;
}
.dot.drag {
  background: rgba(64, 158, 255, 0.4);
  border: 1px dashed #409eff;
}
.dot.avail {
  background: #fff;
  border: 1px solid #b3d8ff;
}
.dot.rank {
  background: var(--glimmer-green-bg);
  border: 1px solid var(--glimmer-green);
}
.dot.confirm {
  background: linear-gradient(135deg, #1e9600, #4eb52e);
}

.dot.multi {
  background: #fbfdff;
  border: 1px dashed #7fb0ff;
}

.cal-scroll {
  overflow: auto;
  /* 多留些高度，避免日历被压得太憋屈 */
  max-height: 68vh;
}

.cal-inner {
  display: flex;
  /* 5 列时仍能保证每列有可点击的时段宽度 */
  min-width: 900px;
}

.cal-axis {
  width: 48px;
  flex-shrink: 0;
  position: relative;
  border-right: 1px solid var(--glimmer-border);
  background: #fafbfd;
  position: sticky;
  left: 0;
  z-index: 3;
}

.axis-corner {
  height: 44px;
  border-bottom: 1px solid var(--glimmer-border);
}

.axis-tick {
  position: absolute;
  right: 6px;
  transform: translateY(-50%);
}

/* 最后一个刻度贴着网格底部，标签改向上对齐，避免撑出 9px 触发滚动条 */
.axis-tick:last-child {
  transform: translateY(-100%);
}

.axis-label {
  font-size: 11px;
  color: var(--glimmer-text-3);
}

.cal-day {
  flex: 1;
  min-width: 0;
  border-right: 1px solid var(--glimmer-border);
}

.cal-day:last-child {
  border-right: none;
}

.day-head {
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--glimmer-text);
  border-bottom: 1px solid var(--glimmer-border);
  background: #fff;
  position: sticky;
  top: 0;
  z-index: 2;
}

.day-head.today {
  color: var(--glimmer-blue);
}

.day-week {
  font-size: 12px;
  font-weight: 400;
  color: var(--glimmer-text-3);
}

.day-grid {
  position: relative;
  user-select: none;
}

.day-grid.selectable {
  cursor: crosshair;
}

.grid-line {
  position: absolute;
  left: 0;
  right: 0;
  height: 1px;
  background: #f0f2f5;
  pointer-events: none;
}
.grid-line.major {
  background: #e6eaf2;
}

/* ===== 时段：同一场次连续时段合并为一整块 ===== */
.slot-group {
  position: absolute;
  left: 3px;
  right: 3px;
  border: 1px solid #cfe0fb;
  border-radius: 10px;
  background: #fff;
  overflow: hidden;
  z-index: 1;
  /* 有日程的块比空白网格"突"出一点点：轻描边 + 极淡投影 */
  box-shadow: 0 1px 2px rgba(11, 99, 246, 0.07);
  transition: box-shadow 0.18s ease, transform 0.18s ease, border-color 0.18s ease;
}

.slot-group:hover {
  z-index: 5;
  border-color: #a9cbff;
  box-shadow: 0 6px 18px rgba(11, 99, 246, 0.16);
  transform: translateY(-1px);
}

/* 含「可多人同时面试」时段的块：虚线描边 + 更明显的淡蓝底，一眼可辨 */
.slot-group.multi-cap {
  border: 1px dashed #7fb0ff;
  background: #f5faff;
}

.slot-group.multi-cap:hover {
  border-color: #4b8ef7;
}

.slot-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  font-size: 12px;
  color: var(--glimmer-text-2);
  border-bottom: 1px solid #eef3fb;
  box-sizing: border-box;
  overflow: hidden;
  white-space: nowrap;
  transition: background-color 0.15s ease;
}

.slot-row:last-child {
  border-bottom: none;
}

.slot-row.clickable {
  cursor: pointer;
  transition: background-color 0.16s ease, box-shadow 0.16s ease, filter 0.16s ease;
}

/* 悬浮提示：可点击时段轻微提亮 + 描边，时间文字变色（用 outline 避免覆盖心仪色条） */
.slot-row.clickable:hover {
  filter: brightness(0.97);
  outline: 1px solid rgba(64, 158, 255, 0.55);
  outline-offset: -1px;
}

.slot-row.clickable:hover .slot-time {
  color: #0b63f6;
}

.day-grid.selectable {
  transition: background-color 0.16s ease;
}

.day-grid.selectable:hover {
  background: rgba(64, 158, 255, 0.05);
}

.slot-row.compact {
  gap: 4px;
  padding: 0 7px;
}

.slot-row.compact .slot-time {
  font-size: 11.5px;
}

/* 30 分钟等窄行：开始时间为主，结束时间弱化，保证一眼能看出区间 */
.slot-end {
  font-weight: 400;
  font-size: 0.93em;
  opacity: 0.62;
}

.slot-row.compact .hot-badge {
  font-size: 9px;
  padding: 0 4px;
  line-height: 13px;
}

.slot-time {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 0;
}

.slot-occupant {
  font-size: 11px;
  opacity: 0.9;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 0 1 auto;
  min-width: 0;
}

.slot-row.compact .slot-occupant {
  font-size: 10px;
}

/* 状态配色（作用在行上，整块保持统一描边） */
.slot-row.slot-available {
  background: #fbfdff;
}

.slot-row.slot-rank1 {
  background: #d8e9ff;
  color: #0a3f9e;
  box-shadow: inset 3px 0 0 var(--glimmer-blue);
}

.slot-row.slot-rank2 {
  background: #e6f0ff;
  color: #0a3f9e;
  box-shadow: inset 3px 0 0 #3b82f6;
}

.slot-row.slot-rank3 {
  background: #f1f7ff;
  color: #0a3f9e;
  box-shadow: inset 3px 0 0 #60a5fa;
}

.slot-row.slot-confirmed {
  background: linear-gradient(135deg, #1e9600, #4eb52e);
  color: #fff;
}

.slot-row.slot-taken {
  background: #fdf3ec;
  color: #a0620a;
}

.slot-badges {
  margin-left: auto;
  display: flex;
  gap: 3px;
  flex-shrink: 0;
}


.badge {
  font-size: 10px;
  padding: 0 5px;
  border-radius: 8px;
  line-height: 15px;
  white-space: nowrap;
}

.rank-badge {
  background: var(--glimmer-green);
  color: #fff;
}

.confirm-badge {
  background: rgba(255, 255, 255, 0.28);
  color: #fff;
}

.hot-badge {
  background: #fef0e6;
  color: #e65c00;
  border: 1px solid #f8d8b8;
}

/* 容量徽章：N 人 · 已占用/容量；空(蓝) / 有人占(橙) / 满员(红) */
.cap-badge {
  background: #e8f1ff;
  color: #0b63f6;
  border: 1px solid #b9d4ff;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.cap-badge.partial {
  background: #fff4e8;
  color: #c26a00;
  border-color: #f6d4ac;
}

.cap-badge.full {
  background: #fdecec;
  color: #d94a4a;
  border-color: #f5c2c2;
}

.sel-preview {
  position: absolute;
  left: 3px;
  right: 3px;
  background: rgba(64, 158, 255, 0.1);
  border: 1px solid #409eff;
  border-radius: 8px;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  font-size: 13px;
  font-weight: 600;
  color: #2575fc;
  z-index: 6;
  pointer-events: none;
  transition: background-color 0.16s ease, box-shadow 0.16s ease;
}

.sel-preview > span {
  margin: 3px 4px 0 0;
  padding: 0 7px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.94);
  box-shadow: 0 1px 3px rgba(11, 99, 246, 0.18);
  font-size: 12px;
  line-height: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 可删除态：允许鼠标进入，悬浮时高亮并浮出删除按钮 */
.sel-preview.removable {
  pointer-events: auto;
}

.sel-preview.removable:hover {
  background: rgba(64, 158, 255, 0.24);
  box-shadow: 0 3px 12px rgba(37, 117, 252, 0.22);
}

.sel-del {
  position: absolute;
  top: -8px;
  right: -6px;
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: #f56c6c;
  color: #fff;
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
  opacity: 0;
  transform: scale(0.7);
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.sel-preview:hover .sel-del {
  opacity: 1;
  transform: scale(1);
}

.sel-del:hover {
  background: #d94a4a;
}

.drag-preview {
  position: absolute;
  left: 3px;
  right: 3px;
  background: rgba(64, 158, 255, 0.22);
  border: 1px dashed #409eff;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  color: #2575fc;
  z-index: 10;
  pointer-events: none;
}

.day-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 30%;
  color: var(--glimmer-text-3);
  font-size: 13px;
  pointer-events: none;
}

@media (max-width: 1200px) {
  .cal-scroll {
    max-height: 60vh;
  }
}

/* ===== 手机端 ===== */
@media (max-width: 768px) {
  .cal-header {
    padding: 10px 12px;
  }

  .cal-date-label {
    font-size: 15px;
  }

  .day-dots {
    display: inline-flex;
    gap: 5px;
    margin-left: 4px;
  }

  .day-dots i {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #d5dbe6;
  }

  .day-dots i.active {
    background: var(--glimmer-green);
  }

  .cal-legend {
    gap: 10px;
  }

  .cal-scroll {
    max-height: 46vh;
  }

  .cal-inner {
    min-width: 0;
  }

  .cal-day {
    border-right: none;
  }

  .slot-row {
    padding: 0 8px;
  }

  .slot-row .slot-time {
    font-size: 12px;
  }

  .slot-row.compact .slot-time {
    font-size: 11.5px;
  }

  .day-empty {
    padding-top: 24%;
  }
}
</style>
