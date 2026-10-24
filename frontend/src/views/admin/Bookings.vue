<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import WeekCalendar from '@/components/WeekCalendar.vue'
import { sessionBookingsApi, deleteBookingApi, directionLabel, directionColor } from '@/apis'
import { downloadCSV } from '@/utils/export'
import { confirmDialog } from '@/utils/dialog'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.id)

const detail = ref(null)
const loading = ref(false)
// 当前在日历上查看心仪时间的成员
const selectedUser = ref(null)
const deleting = ref('')

const statusMap = {
  draft: { text: '草稿', type: 'info' },
  published: { text: '预约中', type: 'primary' },
  assigned: { text: '已分配待确认', type: 'warning' },
  confirmed: { text: '已确认', type: 'success' }
}

async function load() {
  loading.value = true
  try {
    const res = await sessionBookingsApi(sessionId)
    detail.value = res.data
  } catch {
    detail.value = null
  } finally {
    loading.value = false
  }
}

function pickUser(row) {
  // 再点一次取消选中
  selectedUser.value = selectedUser.value?.username === row.username ? null : row
}

const pendingCount = computed(() => detail.value?.pending?.length || 0)
const fullCount = computed(() => (detail.value?.bookings || []).filter(b => b.full).length)

// 日历展示场次的全部日程日期
const calDays = computed(() => {
  const dates = [...new Set((detail.value?.slots || []).map(s => s.date || detail.value?.session.date))]
  return dates.filter(Boolean).sort()
})

// 选中成员的心仪时间映射到全部时段上（myRank 高亮）
const userSlots = computed(() => {
  if (!detail.value || !selectedUser.value) return []
  const rankOf = {}
  for (const p of selectedUser.value.prefs || []) rankOf[p.slot.id] = p.rank
  return (detail.value.slots || []).map(s => ({
    id: s.id,
    sessionId: s.sessionId,
    date: s.date,
    startTime: s.startTime,
    endTime: s.endTime,
    location: s.location,
    capacity: s.capacity,
    myRank: rankOf[s.id] || 0
  }))
})

function prefText(p) {
  return `${p.slot.date} ${p.slot.startTime}-${p.slot.endTime}`
}

function displayName(u) {
  return u.name ? `${u.name}（${u.username}）` : u.username
}

function sessionDateText(sess) {
  const ps = sess?.partitions || []
  if (ps.length <= 1) return sess?.date || ''
  return `${ps[0].date} - ${ps[ps.length - 1].date}（${ps.length} 个日程）`
}

// 导出该场次全部成员的心仪时间
function exportDetail() {
  const d = detail.value
  if (!d) return
  const submitted = new Set((d.bookings || []).map(b => b.username))
  const rows = (d.bookings || []).map(b => [
    b.name,
    b.username,
    b.studentid,
    b.qq,
    b.phone,
    b.full ? '已提交' : '未提交满 3 个',
    b.prefs[0] ? prefText(b.prefs[0]) : '',
    b.prefs[1] ? prefText(b.prefs[1]) : '',
    b.prefs[2] ? prefText(b.prefs[2]) : ''
  ])
  // 从未提交过任何心仪时间的成员也列出来
  for (const u of d.pending || []) {
    if (submitted.has(u.username)) continue
    rows.push([u.name, u.username, u.studentid, u.qq, u.phone, '未提交', '', '', ''])
  }
  downloadCSV(`${d.session.title}-心仪时间.csv`, ['姓名', '用户名', '学号', 'QQ', '手机号', '状态', '第一心仪', '第二心仪', '第三心仪'], rows)
  ElMessage.success('心仪时间明细已导出')
}

// 删除某成员的预约：心仪时间 + 本场次分配记录一并撤下
async function deleteBooking(row) {
  const confirmed = detail.value?.session?.status === 'confirmed'
  try {
    await confirmDialog(
      `将删除 ${displayName(row)} 在「${detail.value.session.title}」的全部心仪时间` +
      (confirmed ? '，并撤下其已确认的面试安排（成员端会立刻看不到结果）' : '，其在未确认版本中的分配也会一并撤下') +
      '。删除后需要成员重新提交，或由你手动安排。',
      '删除预约',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  deleting.value = row.username
  try {
    await deleteBookingApi(sessionId, row.username)
    ElMessage.success(`已删除 ${displayName(row)} 的预约`)
    if (selectedUser.value?.username === row.username) selectedUser.value = null
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.msg || '删除失败')
  } finally {
    deleting.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="page-container">
    <template v-if="detail">
      <div class="page-head">
        <div>
          <h2 class="page-title"><el-icon><Calendar /></el-icon> 预约管理</h2>
          <p class="page-subtitle">
            <el-tag size="small" round :style="{ borderColor: directionColor(detail.session.direction), color: directionColor(detail.session.direction), background: '#fff' }">
              {{ directionLabel(detail.session.direction) }}
            </el-tag>
            {{ detail.session.title }} · {{ sessionDateText(detail) }} · {{ detail.session.startTime }}-{{ detail.session.endTime }} · {{ detail.session.location }}
            <el-tag :type="statusMap[detail.session.status]?.type" size="small" effect="light" round class="status-tag">
              {{ statusMap[detail.session.status]?.text || detail.session.status }}
            </el-tag>
          </p>
        </div>
        <div class="head-actions">
          <el-button size="large" @click="exportDetail">
            <el-icon style="margin-right: 4px"><Download /></el-icon> 导出心仪时间
          </el-button>
          <el-button size="large" @click="router.push('/admin')">返回场次列表</el-button>
        </div>
      </div>

      <div class="stats-row">
        <div class="stat-card glimmer-card">
          <div class="stat-value">{{ fullCount }}</div>
          <div class="stat-label">已提交满 3 个</div>
        </div>
        <div class="stat-card glimmer-card">
          <div class="stat-value">{{ (detail.bookings || []).length }}</div>
          <div class="stat-label">已提交（含不满 3 个）</div>
        </div>
        <div class="stat-card stat-pending glimmer-card">
          <div class="stat-value">{{ pendingCount }}</div>
          <div class="stat-label">未提交满 3 个</div>
        </div>
        <div class="stat-card glimmer-card">
          <div class="stat-value">{{ (detail.slots || []).length }}</div>
          <div class="stat-label">时段总数</div>
        </div>
      </div>

      <div v-if="pendingCount" class="pending glimmer-card">
        <div class="pending-title">
          <el-icon><Warning /></el-icon> 尚未提交满 3 个心仪时间的成员（{{ pendingCount }}）
        </div>
        <div class="pending-tags">
          <el-tag v-for="u in detail.pending" :key="u.username" type="warning" effect="plain" round>
            {{ displayName(u) }}
          </el-tag>
        </div>
      </div>

      <div class="table-card glimmer-card">
        <div class="card-title">
          心仪时间明细
          <span class="title-tip">点击「查看日历」可在下方日历中查看该成员选择的心仪场次</span>
        </div>
        <div class="table-scroll">
          <el-table :data="detail.bookings" style="width: 100%" empty-text="还没有成员提交心仪时间">
            <el-table-column label="姓名" min-width="90">
              <template #default="{ row }">
                <span class="cell-strong">{{ row.name || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="username" label="用户名" min-width="110" show-overflow-tooltip />
            <el-table-column label="学号" min-width="110" class-name="hide-sm">
              <template #default="{ row }">{{ row.studentid || '—' }}</template>
            </el-table-column>
            <el-table-column label="QQ" min-width="100" class-name="hide-sm">
              <template #default="{ row }">{{ row.qq || '—' }}</template>
            </el-table-column>
            <el-table-column label="手机号" min-width="115" class-name="hide-sm">
              <template #default="{ row }">{{ row.phone || '—' }}</template>
            </el-table-column>
            <el-table-column label="状态" width="120" align="center">
              <template #default="{ row }">
                <el-tag :type="row.full ? 'success' : 'warning'" size="small" effect="light" round>
                  {{ row.full ? '已提交满' : '未满 3 个' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="第一心仪" min-width="150" class-name="hide-sm">
              <template #default="{ row }">
                <span v-if="row.prefs[0]" class="mono">{{ prefText(row.prefs[0]) }}</span>
                <span v-else>—</span>
              </template>
            </el-table-column>
            <el-table-column label="第二心仪" min-width="150" class-name="hide-sm">
              <template #default="{ row }">
                <span v-if="row.prefs[1]" class="mono">{{ prefText(row.prefs[1]) }}</span>
                <span v-else>—</span>
              </template>
            </el-table-column>
            <el-table-column label="第三心仪" min-width="150" class-name="hide-sm">
              <template #default="{ row }">
                <span v-if="row.prefs[2]" class="mono">{{ prefText(row.prefs[2]) }}</span>
                <span v-else>—</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="190" align="center" fixed="right">
              <template #default="{ row }">
                <el-button size="small" :type="selectedUser?.username === row.username ? 'primary' : ''" text @click="pickUser(row)">
                  {{ selectedUser?.username === row.username ? '收起日历' : '查看日历' }}
                </el-button>
                <el-button
                  size="small"
                  type="danger"
                  text
                  :loading="deleting === row.username"
                  @click="deleteBooking(row)"
                >
                  删除预约
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <div v-if="selectedUser" class="calendar-card glimmer-card">
        <div class="card-title">
          {{ displayName(selectedUser) }} 的心仪场次
          <el-tag size="small" :type="selectedUser.full ? 'success' : 'warning'" effect="light" round class="title-tag">
            {{ selectedUser.full ? '已提交满 3 个' : '未满 3 个' }}
          </el-tag>
        </div>
        <WeekCalendar :days="calDays" :slots="userSlots" :clickable="false" :selectable="false" />
      </div>
    </template>

    <el-empty v-else-if="!loading" description="场次不存在或已被删除">
      <el-button type="primary" size="large" @click="router.push('/admin')">返回场次列表</el-button>
    </el-empty>
  </div>
</template>

<style scoped>
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

.status-tag {
  margin-left: 6px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
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

.stat-pending .stat-value {
  color: #e6a23c;
}

.pending {
  padding: 16px 20px;
  margin-bottom: 20px;
}

.pending-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: #b88230;
}

.pending-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.table-card {
  padding: 8px 16px;
  margin-bottom: 20px;
  padding-bottom: 12px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--glimmer-text);
  padding: 16px 4px 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.title-tip {
  font-size: 12px;
  font-weight: 400;
  color: var(--glimmer-text-3);
}

.title-tag {
  margin-left: 4px;
}

.table-scroll {
  width: 100%;
  overflow-x: auto;
}

.cell-strong {
  font-weight: 600;
  color: var(--glimmer-text);
}

.mono {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.calendar-card {
  margin-bottom: 20px;
  padding-bottom: 4px;
}

.calendar-card :deep(.week-calendar) {
  margin: 0 16px 16px;
  border: 1px solid var(--glimmer-border);
  box-shadow: none;
}

@media (max-width: 768px) {
  :deep(.hide-sm) {
    display: none !important;
  }

  .head-actions {
    width: 100%;
  }

  .head-actions .el-button {
    flex: 1;
    min-height: 42px;
  }

  .stats-row {
    grid-template-columns: repeat(2, 1fr);
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
}
</style>
