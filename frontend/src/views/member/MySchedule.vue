<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { myScheduleApi, directionLabel } from '@/apis'

const router = useRouter()
const loading = ref(false)
const data = ref({ preferences: [], assignments: [] })

// 按场次聚合
const groups = computed(() => {
  const map = new Map()
  const ensure = (session, partitions) => {
    if (!map.has(session.id)) {
      map.set(session.id, { session, partitions: partitions || [], prefs: [], assignment: null })
    }
    return map.get(session.id)
  }
  for (const p of data.value.preferences) {
    ensure(p.session, p.partitions).prefs.push(p)
  }
  for (const a of data.value.assignments) {
    const g = ensure(a.session, a.partitions)
    g.assignment = a
  }
  return [...map.values()].sort((x, y) => (x.session.date < y.session.date ? 1 : -1))
})

const statusText = {
  draft: '草稿',
  published: '预约中',
  assigned: '已分配待确认',
  confirmed: '已确认'
}

// 日期展示：2026-10-22 -> 10月22日 周X
function fmtDay(d) {
  const dt = new Date(`${d}T00:00:00`)
  if (Number.isNaN(dt.getTime())) return d || ''
  return `${dt.getMonth() + 1}月${dt.getDate()}日 周${'日一二三四五六'[dt.getDay()]}`
}
// 时间范围：各日程一致 -> 每天 09:00-10:00；否则标注各日程不同
function timeRangeText(g) {
  const parts = g.partitions || []
  if (!parts.length) return `${g.session.startTime}-${g.session.endTime}`
  const same = parts.every(p => p.startTime === parts[0].startTime && p.endTime === parts[0].endTime)
  if (same) return `每天 ${parts[0].startTime}-${parts[0].endTime}`
  return `${parts[0].startTime}-${parts[parts.length - 1].endTime}（各日程不同）`
}
// 每时段可同时面试人数：各日程一致才展示
function capacityText(g) {
  const parts = g.partitions || []
  if (!parts.length) return ''
  const caps = parts.map(p => (Number(p.capacity) > 0 ? Number(p.capacity) : 1))
  const same = caps.every(c => c === caps[0])
  if (!same) return '各日程每时段人数不同'
  if (caps[0] <= 1) return ''
  return `每时段 ${caps[0]} 人`
}

// 日程区间：单日 -> 10月22日 周四；多日 -> 10月22日 - 10月24日（3 个日程）
function dateRangeText(g) {
  const parts = g.partitions || []
  if (!parts.length) return fmtDay(g.session.date)
  if (parts.length === 1) return fmtDay(parts[0].date)
  return `${fmtDay(parts[0].date)} - ${fmtDay(parts[parts.length - 1].date)}（${parts.length} 个日程）`
}

async function load() {
  loading.value = true
  try {
    const res = await myScheduleApi()
    data.value = res.data || { preferences: [], assignments: [] }
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page-container" v-loading="loading">
    <h2 class="page-title">
      <el-icon><Notebook /></el-icon> 我的心仪时间
    </h2>
    <p class="page-subtitle">每个方向只能预约 1 个场次，这里按场次查看你的心仪时间选择与最终面试安排</p>

    <el-empty v-if="!groups.length" description="还没有预约记录，去首页选择心仪的面试时段吧">
      <el-button type="primary" @click="router.push('/home')">前往预约</el-button>
    </el-empty>

    <div v-else class="groups">
      <div v-for="g in groups" :key="g.session.id" class="group-card glimmer-card">
        <div class="group-head">
          <div>
            <div class="group-title">
              {{ g.session.title }}
              <el-tag size="small" effect="plain" round style="margin-left: 8px">
                {{ directionLabel(g.session.direction) }}
              </el-tag>
            </div>
            <div class="group-meta">
              {{ dateRangeText(g) }} · {{ timeRangeText(g) }} · {{ g.session.location }}
              <template v-if="capacityText(g)"> · {{ capacityText(g) }}</template>
              <template v-if="g.session.contact">
                · 联系人 {{ g.session.contact }}<template v-if="g.session.contactQQ">（QQ {{ g.session.contactQQ }}）</template>
              </template>
            </div>
          </div>
          <el-tag :type="g.session.status === 'confirmed' ? 'success' : 'primary'" effect="light">
            {{ statusText[g.session.status] || g.session.status }}
          </el-tag>
        </div>

        <div class="group-body">
          <div class="pref-list">
            <div class="pref-list-title">我的心仪时间</div>
            <div v-for="p in g.prefs" :key="p.rank" class="pref-row">
              <span class="pref-rank" :class="`rank-${p.rank}`">{{ ['①', '②', '③'][p.rank - 1] }}</span>
              <span class="pref-date">{{ fmtDay(p.slot.date || g.session.date) }}</span>
              <span class="pref-time">{{ p.slot.startTime }} - {{ p.slot.endTime }}</span>
              <span class="pref-loc">{{ p.slot.location }}</span>
            </div>
          </div>

          <div v-if="g.assignment" class="assign-result">
            <div class="assign-label">
              <el-icon><CircleCheck /></el-icon> 最终面试安排
            </div>
            <div class="assign-time">
              <span class="pref-date">{{ fmtDay(g.assignment.slot.date || g.session.date) }}</span>
              {{ g.assignment.slot.startTime }} - {{ g.assignment.slot.endTime }}
              <span v-if="(g.assignment.slot.capacity || 1) > 1" class="assign-cap">
                该时段可同时面试 {{ g.assignment.slot.capacity }} 人
              </span>
            </div>
            <div class="assign-loc">{{ g.assignment.slot.location }}</div>
          </div>
          <div v-else class="assign-pending">
            <el-icon><Clock /></el-icon>
            {{ g.session.status === 'confirmed' ? '本场次未分配到时段，请联系管理员' : '等待管理员分配与确认' }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.groups {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.group-card {
  padding: 20px 24px;
}

.group-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.group-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--glimmer-text);
  display: flex;
  align-items: center;
}

.group-meta {
  font-size: 13px;
  color: var(--glimmer-text-3);
  margin-top: 4px;
}

.group-body {
  margin-top: 16px;
  display: grid;
  grid-template-columns: 1fr 240px;
  gap: 20px;
  align-items: center;
}

.pref-list-title {
  font-size: 13px;
  color: var(--glimmer-text-3);
  margin-bottom: 8px;
}

.pref-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.pref-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: var(--glimmer-green-bg);
  border-radius: 10px;
  font-size: 13px;
}

.pref-rank {
  font-size: 16px;
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

.pref-time {
  font-weight: 600;
  color: var(--glimmer-text);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  flex: 0 0 auto;
}

.pref-date {
  flex: 0 0 auto;
  padding: 0 6px;
  border-radius: 6px;
  background: var(--glimmer-blue-bg);
  color: var(--glimmer-blue);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.pref-loc {
  flex: 1 1 auto;
  min-width: 0;
  color: var(--glimmer-text-3);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.assign-result {
  background: linear-gradient(135deg, #1e9600, #4eb52e);
  border-radius: 14px;
  padding: 16px;
  color: #fff;
}

.assign-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  opacity: 0.92;
}

.assign-time {
  font-size: 22px;
  font-weight: 700;
  margin-top: 6px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.assign-cap {
  display: inline-block;
  margin-left: 8px;
  padding: 0 6px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.22);
  font-size: 12px;
  font-weight: 500;
}

.assign-loc {
  font-size: 13px;
  margin-top: 2px;
  opacity: 0.9;
}

.assign-pending {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background: #fdf6ec;
  color: #b88230;
  border-radius: 14px;
  font-size: 13px;
}

@media (max-width: 900px) {
  .group-body {
    grid-template-columns: 1fr;
  }
}

/* ===== 手机端 ===== */
@media (max-width: 768px) {
  .page-container {
    padding: 14px 12px;
  }

  .group-card {
    padding: 16px;
  }

  .group-head {
    flex-direction: column;
    gap: 8px;
  }

  .group-title {
    font-size: 16px;
    flex-wrap: wrap;
  }
}
</style>
