<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { confirmDialog } from '@/utils/dialog'
import { downloadCSV } from '@/utils/export'
import WeekCalendar from '@/components/WeekCalendar.vue'
import {
  getAssignmentApi,
  generateAssignmentApi,
  confirmAssignmentApi,
  reassignApi,
  swapApi,
  manualAssignApi,
  directionLabel,
  directionColor
} from '@/apis'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.id)

const view = ref(null)
const generating = ref(false)
const reassignDialog = ref(false)
const swapDialog = ref(false)
const manualDialog = ref(false)
const activeRow = ref(null)
const targetSlot = ref(null)
const swapTarget = ref(null)
const manualUser = ref(null)
const manualSlot = ref(null)
const manualSaving = ref(false)

const stats = computed(() => view.value?.stats || {})
const rows = computed(() => view.value?.assignments || [])

// 场次的日程区间（多日程时显示起止与日程数）
const sessionDateText = computed(() => {
  const parts = view.value?.partitions || []
  if (parts.length <= 1) return view.value?.session?.date || ''
  return `${parts[0].date} - ${parts[parts.length - 1].date}（${parts.length} 个日程）`
})
const freeSlots = computed(() => view.value?.freeSlots || [])
const unsubmitted = computed(() => view.value?.unsubmitted || [])
const unassignedUsers = computed(() => view.value?.unassignedUsers || [])
const confirmed = computed(() => view.value?.version?.confirmed === 1)

// 对话框宽度随视口自适应
const dialogWidth = computed(() => (window.innerWidth < 768 ? '92%' : '440px'))

// 窄一些的笔记本上先收起低优先级列（学号/QQ/手机号/来源），避免表格被裁切
const vw = ref(window.innerWidth)
const compactCols = computed(() => vw.value < 1360)
function onResize() {
  vw.value = window.innerWidth
}

function fmtDate(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
// 日历展示场次的全部日程日期（多日程场次逐天展示）
const days = computed(() => {
  if (!view.value) return []
  const parts = view.value.partitions || []
  if (parts.length > 1) return parts.map(p => p.date)
  const base = new Date(view.value.session.date + 'T00:00:00')
  return [-1, 0, 1].map(i => {
    const d = new Date(base)
    d.setDate(d.getDate() + i)
    return fmtDate(d)
  })
})

// 分配总览日历：按时段聚合（同一时段可容多人，必须只渲染一块，
// 否则每人一条会在同一天同一时间产生多个绝对定位块互相重叠遮挡）
const calSlots = computed(() => {
  if (!view.value) return []
  const bySlot = new Map()
  for (const a of rows.value) {
    let it = bySlot.get(a.slot.id)
    if (!it) {
      it = {
        id: `a-${a.slot.id}`,
        sessionId: view.value.session.id,
        date: a.slot.date || view.value.session.date,
        startTime: a.slot.startTime,
        endTime: a.slot.endTime,
        location: a.slot.location,
        names: [],
        capacity: a.slot.capacity,
        occupied: 0
      }
      bySlot.set(a.slot.id, it)
    }
    it.names.push(a.name || a.username)
    it.occupied++
  }
  const items = [...bySlot.values()].map(it => ({
    ...it,
    occupant: it.occupied > 1 ? `${it.names.join('、')} 共 ${it.occupied} 人` : it.names[0]
  }))
  // 空闲时段：剔除已有分配的时段（容量 >1 且部分占用时，同一时段会同时出现在两边，产生重叠块）
  const free = freeSlots.value
    .filter(s => !bySlot.has(s.id))
    .map(s => ({
      id: `f-${s.id}`,
      sessionId: view.value.session.id,
      date: s.date || view.value.session.date,
      startTime: s.startTime,
      endTime: s.endTime,
      location: s.location,
      occupant: '',
      capacity: s.capacity,
      occupied: 0
    }))
  return items.concat(free)
})

async function load() {
  try {
    const res = await getAssignmentApi(sessionId)
    view.value = res.data
  } catch (e) {
    view.value = null
  }
}

async function onGenerate() {
  const hasData = rows.value.length > 0
  if (hasData) {
    try {
      await confirmDialog(
        '重新生成将覆盖当前未确认的分配版本（历史版本仍会保留），确定继续？',
        '重新生成分配',
        { confirmButtonText: '重新生成', cancelButtonText: '取消', type: 'warning' }
      )
    } catch {
      return
    }
  }
  generating.value = true
  try {
    await generateAssignmentApi(sessionId)
    ElMessage.success('已按心仪时间偏好生成新的分配版本')
    await load()
  } finally {
    generating.value = false
  }
}

// 导出最终分配结果（CSV，Excel 可直接打开）
function exportResult() {
  downloadCSV(
    `${view.value.session.title}-分配结果.csv`,
    ['姓名', '用户名', '学号', 'QQ', '手机号', '日期', '面试时段', '地点', '心仪序', '来源'],
    rows.value.map(a => [
      a.name,
      a.username,
      a.studentid,
      a.qq,
      a.phone,
      a.slot.date || view.value.session.date,
      `${a.slot.startTime}-${a.slot.endTime}`,
      a.slot.location,
      a.matchedRank ? `第${['一', '二', '三'][a.matchedRank - 1]}心仪` : '不在心仪内',
      a.source === 'manual' ? '人工' : '自动'
    ])
  )
  ElMessage.success('分配结果已导出')
}

async function onConfirm() {
  try {
    await confirmDialog(
      '确认后面试者即可看到自己的面试时间与地点，且心仪时间将被锁定。确定确认分配？',
      '确认分配',
      { confirmButtonText: '确认分配', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  await confirmAssignmentApi(sessionId)
  ElMessage.success('分配已确认，成员现在可以看到结果了')
  await load()
}

function openReassign(row) {
  activeRow.value = row
  targetSlot.value = null
  reassignDialog.value = true
}

async function doReassign() {
  if (!targetSlot.value) {
    ElMessage.warning('请选择目标时段')
    return
  }
  await reassignApi(activeRow.value.id, targetSlot.value)
  ElMessage.success(`已将 ${activeRow.value.username} 改派成功`)
  reassignDialog.value = false
  await load()
}

function openSwap(row) {
  activeRow.value = row
  swapTarget.value = null
  swapDialog.value = true
}

function openManual(u) {
  manualUser.value = u
  manualSlot.value = null
  manualDialog.value = true
}

async function doManual() {
  if (!manualSlot.value) {
    ElMessage.warning('请选择空闲时段')
    return
  }
  manualSaving.value = true
  try {
    await manualAssignApi(sessionId, manualUser.value.username, manualSlot.value)
    ElMessage.success(`已将 ${manualUser.value.username} 安排进所选时段`)
    manualDialog.value = false
    await load()
  } finally {
    manualSaving.value = false
  }
}

async function doSwap() {
  if (!swapTarget.value) {
    ElMessage.warning('请选择对调成员')
    return
  }
  await swapApi(activeRow.value.id, swapTarget.value)
  ElMessage.success('已对调两人的面试时段')
  swapDialog.value = false
  await load()
}

onMounted(() => {
  window.addEventListener('resize', onResize)
  load()
})
onUnmounted(() => window.removeEventListener('resize', onResize))
</script>

<template>
  <div class="page-container" v-if="view">
    <div class="page-head">
      <div>
        <h2 class="page-title"><el-icon><DataBoard /></el-icon> 分配管理</h2>
        <p class="page-subtitle">
          <el-tag size="small" round :style="{ borderColor: directionColor(view.session.direction), color: directionColor(view.session.direction), background: '#fff' }">
            {{ directionLabel(view.session.direction) }}
          </el-tag>
          {{ view.session.title }} · {{ sessionDateText }} · {{ view.session.startTime }}-{{ view.session.endTime }} · {{ view.session.location }}
          <template v-if="view.session.contact">
            · 联系人 {{ view.session.contact }}<template v-if="view.session.contactQQ">（QQ {{ view.session.contactQQ }}）</template>
          </template>
        </p>
      </div>
      <div class="head-actions">
        <el-button size="large" :loading="generating" @click="onGenerate">
          <el-icon style="margin-right: 4px"><MagicStick /></el-icon>
          {{ rows.length ? '重新生成分配' : '生成分配' }}
        </el-button>
        <el-button
          v-if="rows.length"
          type="primary"
          size="large"
          :disabled="confirmed"
          @click="onConfirm"
        >
          <el-icon style="margin-right: 4px"><CircleCheck /></el-icon>
          {{ confirmed ? '已确认' : '确认分配' }}
        </el-button>
        <el-button v-if="rows.length" size="large" @click="exportResult">
          <el-icon style="margin-right: 4px"><Download /></el-icon> 导出分配结果
        </el-button>
        <el-button size="large" @click="router.push('/admin')">返回</el-button>
      </div>
    </div>

    <el-alert
      v-if="confirmed"
      class="state-alert"
      type="success"
      :closable="false"
      show-icon
      title="分配已确认：面试者已可在系统中查看自己的面试时间与地点；你仍可随时改派或对调。"
    />

    <div class="stats-row">
      <div class="stat-card glimmer-card">
        <div class="stat-value">{{ stats.totalParticipants || 0 }}</div>
        <div class="stat-label">参与人数</div>
      </div>
      <div class="stat-card stat-first glimmer-card">
        <div class="stat-value">{{ stats.first || 0 }}</div>
        <div class="stat-label">第一心仪达成</div>
      </div>
      <div class="stat-card stat-second glimmer-card">
        <div class="stat-value">{{ stats.second || 0 }}</div>
        <div class="stat-label">第二心仪达成</div>
      </div>
      <div class="stat-card stat-third glimmer-card">
        <div class="stat-value">{{ stats.third || 0 }}</div>
        <div class="stat-label">第三心仪达成</div>
      </div>
      <div class="stat-card stat-none glimmer-card">
        <div class="stat-value">{{ stats.unassigned || 0 }}</div>
        <div class="stat-label">未分配</div>
      </div>
      <div class="stat-card glimmer-card">
        <div class="stat-value">{{ stats.slotCount || 0 }}</div>
        <div class="stat-label">时段总数</div>
      </div>
    </div>

    <div v-if="unassignedUsers.length" class="unsubmitted glimmer-card">
      <div class="unsubmitted-title">
        <el-icon><Warning /></el-icon> 提交了心仪时间但未能分配的成员（{{ unassignedUsers.length }}）
      </div>
      <div class="unsubmitted-tags">
        <span v-for="u in unassignedUsers" :key="u.username" class="unassign-item">
          <el-tag type="danger" effect="plain" round>
            {{ u.name ? `${u.name}（${u.username}）` : u.username }}
          </el-tag>
          <el-button size="small" text type="primary" :disabled="!freeSlots.length" @click="openManual(u)">
            安排时段
          </el-button>
        </span>
      </div>
      <p v-if="!freeSlots.length" class="unassign-hint">
        当前没有空闲时段：可改派其他人腾出时段，或联系成员调整心仪时间后重新生成。
      </p>
    </div>

    <div v-if="unsubmitted.length" class="unsubmitted glimmer-card">
      <div class="unsubmitted-title">
        <el-icon><Warning /></el-icon> 尚未提交心仪时间的成员（{{ unsubmitted.length }}）
      </div>
      <div class="unsubmitted-tags">
        <el-tag v-for="u in unsubmitted" :key="u.username" type="warning" effect="plain" round>
          {{ u.name ? `${u.name}（${u.username}）` : u.username }}
        </el-tag>
      </div>
    </div>

    <div class="content-grid">
      <div class="table-card glimmer-card">
        <div class="card-title">逐人分配</div>
        <div class="table-scroll">
        <el-table :data="rows" style="width: 100%" empty-text="点击「生成分配」开始">
          <el-table-column prop="username" label="用户名" min-width="110" show-overflow-tooltip />
          <el-table-column label="姓名" min-width="90" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="cell-strong">{{ row.name || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!compactCols" prop="studentid" label="学号" min-width="110" class-name="hide-sm">
            <template #default="{ row }">{{ row.studentid || '—' }}</template>
          </el-table-column>
          <el-table-column v-if="!compactCols" label="QQ" min-width="110" class-name="hide-sm">
            <template #default="{ row }">
              <span class="mono">{{ row.qq || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column v-if="!compactCols" label="手机号" min-width="120" class-name="hide-sm">
            <template #default="{ row }">
              <span class="mono">{{ row.phone || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="日期" width="110" align="center" class-name="hide-sm">
            <template #default="{ row }">
              <span class="mono">{{ row.slot.date || view.session.date }}</span>
            </template>
          </el-table-column>
          <el-table-column label="面试时段" min-width="150">
            <template #default="{ row }">
              <span class="mono">{{ row.slot.startTime }} - {{ row.slot.endTime }}</span>
              <span v-if="(row.slot.capacity || 1) > 1" class="slot-cap-tip">
                {{ row.slotOccupied }}/{{ row.slot.capacity }} 人
              </span>
            </template>
          </el-table-column>
          <el-table-column label="心仪序" width="96" align="center" class-name="hide-sm">
            <template #default="{ row }">
              <el-tag
                v-if="row.matchedRank"
                :type="row.matchedRank === 1 ? 'success' : row.matchedRank === 2 ? 'primary' : 'info'"
                size="small"
                effect="light"
                round
              >
                第{{ ['一', '二', '三'][row.matchedRank - 1] }}心仪
              </el-tag>
              <span v-else title="时段不在本人提交的心仪时间内">—</span>
            </template>
          </el-table-column>
          <el-table-column prop="slot.location" label="地点" min-width="110" show-overflow-tooltip class-name="hide-sm" />
          <el-table-column v-if="!compactCols" label="来源" width="80" align="center" class-name="hide-sm">
            <template #default="{ row }">
              <el-tag :type="row.source === 'manual' ? 'warning' : 'info'" size="small" effect="plain">
                {{ row.source === 'manual' ? '人工' : '自动' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" align="center">
            <template #default="{ row }">
              <el-button size="small" text type="primary" @click="openReassign(row)">改派</el-button>
              <el-button size="small" text type="primary" @click="openSwap(row)">对调</el-button>
            </template>
          </el-table-column>
        </el-table>
        </div>
      </div>

      <div class="calendar-card">
        <div class="card-title">分配总览</div>
        <WeekCalendar :days="days" :slots="calSlots" :clickable="false" :selectable="false" />
      </div>
    </div>

    <!-- 改派对话框 -->
    <el-dialog v-model="reassignDialog" title="改派面试时段" :width="dialogWidth">
      <p class="dialog-desc">
        将 <b>{{ activeRow?.username }}</b>（当前 {{ activeRow?.slot.startTime }}-{{ activeRow?.slot.endTime }}）改派至：
      </p>
      <el-select v-model="targetSlot" placeholder="选择空闲时段" style="width: 100%">
        <el-option
          v-for="s in freeSlots"
          :key="s.id"
          :label="`${s.startTime} - ${s.endTime}（${s.location}）`"
          :value="s.id"
        />
      </el-select>
      <template #footer>
        <el-button @click="reassignDialog = false">取消</el-button>
        <el-button type="primary" @click="doReassign">确定改派</el-button>
      </template>
    </el-dialog>

    <!-- 对调对话框 -->
    <el-dialog v-model="swapDialog" title="对调面试时段" :width="dialogWidth">
      <p class="dialog-desc">
        选择要与 <b>{{ activeRow?.username }}</b> 对调的成员：
      </p>
      <el-select v-model="swapTarget" placeholder="选择成员" style="width: 100%">
        <el-option
          v-for="r in rows.filter(r => r.id !== activeRow?.id)"
          :key="r.id"
          :label="`${r.username}（${r.slot.startTime}-${r.slot.endTime}）`"
          :value="r.id"
        />
      </el-select>
      <template #footer>
        <el-button @click="swapDialog = false">取消</el-button>
        <el-button type="primary" @click="doSwap">确定对调</el-button>
      </template>
    </el-dialog>

    <!-- 未分配成员手动安排对话框 -->
    <el-dialog v-model="manualDialog" :title="`为 ${manualUser?.username} 安排时段`" :width="dialogWidth">
      <p class="dialog-desc">
        从空闲时段中选一个。若所选时段在其提交的心仪时间内，会自动记录对应的心仪序。
      </p>
      <el-select v-model="manualSlot" placeholder="选择空闲时段" style="width: 100%">
        <el-option
          v-for="s in freeSlots"
          :key="s.id"
          :label="`${s.startTime} - ${s.endTime}（${s.location}）`"
          :value="s.id"
        />
      </el-select>
      <template #footer>
        <el-button @click="manualDialog = false">取消</el-button>
        <el-button type="primary" :loading="manualSaving" @click="doManual">确定安排</el-button>
      </template>
    </el-dialog>
  </div>

  <div v-else class="page-container">
    <el-empty description="尚未生成分配">
      <el-button type="primary" size="large" :loading="generating" @click="onGenerate">
        <el-icon style="margin-right: 4px"><MagicStick /></el-icon> 立即生成分配
      </el-button>
      <el-button size="large" style="margin-left: 12px" @click="router.push('/admin')">返回列表</el-button>
    </el-empty>
  </div>
</template>

<style scoped>
/* 分配表列较多，容器比其他页面更宽 */
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

.head-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.state-alert {
  margin-bottom: 16px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 14px;
  margin-bottom: 20px;
}

.stat-card {
  padding: 16px;
  text-align: center;
}

.stat-value {
  font-size: 30px;
  font-weight: 700;
  color: var(--glimmer-text);
  font-variant-numeric: tabular-nums;
}

.stat-label {
  font-size: 12px;
  color: var(--glimmer-text-3);
  margin-top: 4px;
}

.stat-first .stat-value {
  color: var(--glimmer-blue);
}
.stat-second .stat-value {
  color: var(--glimmer-blue-2);
}
.stat-third .stat-value {
  color: #7aabff;
}
.stat-none .stat-value {
  color: #e6a23c;
}

.unsubmitted {
  padding: 16px 20px;
  margin-bottom: 20px;
}

.unsubmitted-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: #b88230;
}

.unsubmitted-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.unassign-item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.unassign-hint {
  margin: 10px 0 0;
  font-size: 12px;
  color: var(--glimmer-text-3);
}

/* 逐人分配表与分配总览都按整行展示：
   表格列多、日历自带 900px 最小宽度，并排放会把两边都挤变形 */
.content-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}

.table-card,
.calendar-card {
  min-width: 0;
}

.calendar-card {
  padding-bottom: 4px;
}

.calendar-card :deep(.week-calendar) {
  margin: 12px 16px 16px;
  border: 1px solid var(--glimmer-border);
  box-shadow: none;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--glimmer-text);
  padding: 16px 20px 0;
}

.table-card {
  padding-bottom: 12px;
}

.mono {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.slot-cap-tip {
  display: inline-block;
  margin-left: 6px;
  padding: 0 5px;
  border-radius: 6px;
  background: #eef2f7;
  color: #5a6b85;
  font-size: 11px;
  font-weight: 500;
}

.cell-strong {
  font-weight: 600;
  color: var(--glimmer-text);
}

.table-scroll {
  width: 100%;
  overflow-x: auto;
}

.dialog-desc {
  font-size: 14px;
  color: var(--glimmer-text-2);
  line-height: 1.7;
  margin: 0 0 14px;
}

@media (max-width: 1100px) {
  .stats-row {
    grid-template-columns: repeat(3, 1fr);
  }
}

/* ===== 手机端 ===== */
@media (max-width: 768px) {
  :deep(.hide-sm) {
    display: none !important;
  }

  .page-container {
    padding: 14px 12px;
  }

  .head-actions {
    width: 100%;
  }

  .head-actions .el-button {
    flex: 1;
    min-height: 42px;
  }

  .stats-row {
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
  }

  .stat-card {
    padding: 12px 8px;
  }

  .stat-value {
    font-size: 24px;
  }

  .stat-label {
    font-size: 11px;
  }

  .table-card :deep(.el-table__cell) {
    padding: 6px 0;
  }

  .table-card :deep(.el-table .cell) {
    padding: 0 8px;
  }

  .unsubmitted {
    padding: 14px 16px;
  }
}

/* ===== 手机端 ===== */
</style>
