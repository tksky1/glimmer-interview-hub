<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { confirmDialog, alertDialog } from '@/utils/dialog'
import {
  sessionsApi,
  deleteSessionApi,
  publishSessionApi,
  adminsApi,
  addAdminApi,
  removeAdminApi,
  invitesApi,
  generateInvitesApi,
  deleteInviteApi,
  DIRECTIONS,
  directionLabel,
  directionColor
} from '@/apis'

const router = useRouter()
const sessions = ref([])
const admins = ref([])
const invites = ref([])
const newAdmin = ref('')
const dirFilter = ref('')

// 邀请码生成表单
const invDirection = ref('java')
const invNote = ref('')
const invCount = ref(1)
const generating = ref(false)

const statusMap = {
  draft: { text: '草稿', type: 'info' },
  published: { text: '预约中', type: 'primary' },
  assigned: { text: '已分配待确认', type: 'warning' },
  confirmed: { text: '已确认', type: 'success' }
}

const filteredSessions = computed(() =>
  dirFilter.value ? sessions.value.filter(s => s.direction === dirFilter.value) : sessions.value
)

async function loadSessions() {
  const res = await sessionsApi()
  sessions.value = res.data?.sessions || []
}

async function loadAdmins() {
  const res = await adminsApi()
  admins.value = res.data?.admins || []
}

async function loadInvites() {
  const res = await invitesApi()
  invites.value = res.data?.invites || []
}

async function onPublish(row) {
  const ok = await confirmDialog(`发布后成员即可预约「${row.title}」，确定发布？`, '发布场次', {
    confirmButtonText: '发布',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => true).catch(() => false)
  if (!ok) return
  await publishSessionApi(row.id)
  ElMessage.success('已发布，成员现在可以预约了')
  loadSessions()
}

async function onDelete(row) {
  const msg = row.status === 'draft'
    ? `删除草稿「${row.title}」？此操作不可恢复`
    : `删除「${row.title}」？该场次的 ${row.participants || 0} 人心仪时间与全部分配结果会一并删除，且不可恢复`
  const ok = await confirmDialog(msg, '删除场次', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => true).catch(() => false)
  if (!ok) return
  await deleteSessionApi(row.id)
  ElMessage.success('已删除')
  loadSessions()
}

async function onAddAdmin() {
  const username = newAdmin.value.trim()
  if (!username) {
    ElMessage.warning('请输入用户名')
    return
  }
  await addAdminApi(username)
  ElMessage.success(`已添加管理员 ${username}`)
  newAdmin.value = ''
  loadAdmins()
}

async function onRemoveAdmin(row) {
  const ok = await confirmDialog(`移除管理员 ${row.username}？`, '移除管理员', {
    confirmButtonText: '移除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => true).catch(() => false)
  if (!ok) return
  await removeAdminApi(row.username)
  ElMessage.success('已移除')
  loadAdmins()
}

async function onGenerateInvites() {
  generating.value = true
  try {
    const res = await generateInvitesApi(invDirection.value, invNote.value.trim(), invCount.value)
    const codes = (res.data?.invites || []).map(i => i.code).join('、')
    ElMessage.success(`已生成 ${res.data?.invites?.length || 0} 个邀请码：${codes}`)
    invNote.value = ''
    loadInvites()
  } finally {
    generating.value = false
  }
}

async function onDeleteInvite(row) {
  const ok = await confirmDialog(`删除邀请码 ${row.code}？`, '删除邀请码', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => true).catch(() => false)
  if (!ok) return
  await deleteInviteApi(row.id)
  ElMessage.success('已删除')
  loadInvites()
}

// 复制邀请码专属链接
async function copyInviteLink(row) {
  const link = `${window.location.origin}/interview/?invite=${row.code}`
  try {
    await navigator.clipboard.writeText(link)
    ElMessage.success('专属链接已复制，发给该方向的同学即可')
  } catch {
    alertDialog(link, '请手动复制专属链接', { confirmButtonText: '知道了' }).catch(() => {})
  }
}

onMounted(() => {
  loadSessions()
  loadAdmins()
  loadInvites()
})
</script>

<template>
  <div class="page-container">
    <div class="page-head">
      <div>
        <h2 class="page-title"><el-icon><Setting /></el-icon> 面试场次管理</h2>
        <p class="page-subtitle">按方向创建面试场次并发布，成员预约后即可进行智能分配</p>
      </div>
      <el-button type="primary" size="large" @click="router.push('/admin/session/new')">
        <el-icon style="margin-right: 6px"><Plus /></el-icon> 新建面试场次
      </el-button>
    </div>

    <div class="toolbar glimmer-card">
      <span class="toolbar-label">方向筛选</span>
      <el-radio-group v-model="dirFilter">
        <el-radio-button value="">全部</el-radio-button>
        <el-radio-button v-for="d in DIRECTIONS" :key="d.value" :value="d.value">{{ d.label }}</el-radio-button>
      </el-radio-group>
    </div>

    <div class="glimmer-card table-card">
      <div class="table-scroll">
      <el-table :data="filteredSessions" style="width: 100%" empty-text="还没有面试场次，点击右上角创建">
        <el-table-column prop="title" label="场次" min-width="190">
          <template #default="{ row }">
            <div class="cell-title-line">
              <span class="cell-title">{{ row.title }}</span>
              <el-tag :type="statusMap[row.status]?.type" size="small" effect="light" round>
                {{ statusMap[row.status]?.text || row.status }}
              </el-tag>
            </div>
            <span class="cell-tags-sm">
              <el-tag size="small" round :style="{ borderColor: directionColor(row.direction), color: directionColor(row.direction), background: '#fff' }">
                {{ directionLabel(row.direction) }}
              </el-tag>
            </span>
          </template>
        </el-table-column>
        <el-table-column label="方向" width="110" align="center" class-name="hide-sm">
          <template #default="{ row }">
            <el-tag size="small" round :style="{ borderColor: directionColor(row.direction), color: directionColor(row.direction), background: '#fff' }">
              {{ directionLabel(row.direction) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="日期" width="150">
          <template #default="{ row }">
            <span v-if="(row.partitions || []).length <= 1">{{ row.date }}</span>
            <span v-else>
              {{ row.partitions[0].date }} - {{ row.partitions[row.partitions.length - 1].date }}
              <span class="part-tip">{{ row.partitions.length }} 个日程</span>
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="slotMinutes" label="每场时长" width="90" class-name="hide-sm">
          <template #default="{ row }">{{ row.slotMinutes }} 分钟</template>
        </el-table-column>
        <el-table-column prop="location" label="地点" min-width="110" show-overflow-tooltip class-name="hide-sm" />
        <el-table-column label="联系人" min-width="130" show-overflow-tooltip class-name="hide-sm">
          <template #default="{ row }">
            {{ row.contact }}<span v-if="row.contactQQ">（{{ row.contactQQ }}）</span>
          </template>
        </el-table-column>
        <el-table-column prop="slotCount" label="时段数" width="80" align="center" class-name="hide-sm" />
        <el-table-column prop="participants" label="已预约人数" width="100" align="center" class-name="hide-sm" />
        <el-table-column label="操作" width="360" align="center" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'draft'">
              <el-button size="small" text type="primary" @click="router.push(`/admin/session/${row.id}/edit`)">编辑</el-button>
              <el-button size="small" text type="primary" @click="onPublish(row)">发布</el-button>
              <el-button size="small" text type="danger" @click="onDelete(row)">删除</el-button>
            </template>
            <template v-else>
              <el-button size="small" text type="primary" @click="router.push(`/admin/session/${row.id}/bookings`)">预约管理</el-button>
              <el-button size="small" text type="primary" @click="router.push(`/admin/session/${row.id}/assign`)">分配管理</el-button>
              <!-- 未确认才能改日程：已有人选择的时段会被保护 -->
              <el-button
                v-if="row.status !== 'confirmed'"
                size="small"
                text
                type="primary"
                @click="router.push(`/admin/session/${row.id}/edit`)"
              >编辑日程</el-button>
              <el-button size="small" text type="primary" @click="router.push(`/admin/session/${row.id}/append`)">追加日程</el-button>
              <el-button size="small" text type="danger" @click="onDelete(row)">删除</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </div>

    <div class="glimmer-card invite-card">
      <div class="invite-head">
        <div>
          <div class="invite-title"><el-icon><Link /></el-icon> 方向邀请码</div>
          <p class="invite-desc">
            同学通过邀请码或专属链接登录后即可绑定方向，只能看到并预约本方向的日程。不同方向的日程互不冲突。
            邀请码不限制使用次数，同一个码可以发给多位同学，右侧会统计已使用人数。
          </p>
        </div>
        <div class="invite-generate">
          <el-select v-model="invDirection" style="width: 130px">
            <el-option v-for="d in DIRECTIONS" :key="d.value" :label="d.label" :value="d.value" />
          </el-select>
          <el-input v-model="invNote" placeholder="备注（可选），如：Java 方向新生群" style="width: 200px" />
          <el-input-number v-model="invCount" :min="1" :max="20" style="width: 110px" />
          <el-button type="primary" :loading="generating" @click="onGenerateInvites">生成</el-button>
        </div>
      </div>
      <div class="table-scroll">
      <el-table :data="invites" style="width: 100%" max-height="320" empty-text="还没有邀请码">
        <el-table-column prop="code" label="邀请码" width="130">
          <template #default="{ row }">
            <span class="invite-code">{{ row.code }}</span>
          </template>
        </el-table-column>
        <el-table-column label="方向" width="110" align="center">
          <template #default="{ row }">
            <el-tag size="small" round :style="{ borderColor: directionColor(row.direction), color: directionColor(row.direction), background: '#fff' }">
              {{ directionLabel(row.direction) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="备注" min-width="140" show-overflow-tooltip class-name="hide-sm" />
        <el-table-column label="使用情况" width="110" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.usedCount > 0" type="success" size="small" effect="plain">
              已用 {{ row.usedCount }} 人
            </el-tag>
            <el-tag v-else type="info" size="small" effect="plain">未使用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最近使用" width="120" align="center" class-name="hide-sm">
          <template #default="{ row }">{{ row.usedBy || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150" align="center">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="copyInviteLink(row)">复制链接</el-button>
            <el-button size="small" text type="danger" @click="onDeleteInvite(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      </div>
    </div>

    <div class="glimmer-card admin-card">
      <div class="admin-head">
        <div>
          <div class="admin-title"><el-icon><UserFilled /></el-icon> 管理员名单</div>
          <p class="admin-desc">管理员可以发布场次、生成分配并确认结果。初始管理员由部署配置指定。</p>
        </div>
        <div class="admin-add">
          <el-input v-model="newAdmin" placeholder="输入旧系统用户名" style="width: 200px" @keyup.enter="onAddAdmin" />
          <el-button type="primary" @click="onAddAdmin">添加</el-button>
        </div>
      </div>
      <div class="admin-tags">
        <el-tag
          v-for="a in admins"
          :key="a.username"
          size="large"
          effect="light"
          round
          closable
          class="admin-tag"
          @close="onRemoveAdmin(a)"
        >
          {{ a.username }}
        </el-tag>
      </div>
    </div>
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

.cell-title-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 18px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.toolbar-label {
  font-size: 13px;
  color: var(--glimmer-text-3);
}

.table-card {
  padding: 8px 16px;
  margin-bottom: 20px;
}

.cell-title {
  font-weight: 600;
  color: var(--glimmer-text);
}

.part-tip {
  display: block;
  font-size: 11.5px;
  color: var(--glimmer-text-3);
}

/* 手机端在场次列内联展示方向/状态（对应独立列已隐藏） */
.cell-tags-sm {
  display: none;
  gap: 6px;
  margin-top: 4px;
}

.table-scroll {
  width: 100%;
  overflow-x: auto;
}

.invite-card {
  padding: 18px 20px;
  margin-bottom: 20px;
}

.invite-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.invite-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--glimmer-text);
  display: flex;
  align-items: center;
  gap: 6px;
}

.invite-desc {
  font-size: 13px;
  color: var(--glimmer-text-3);
  margin: 6px 0 0;
  max-width: 560px;
  line-height: 1.6;
}

.invite-generate {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.invite-code {
  font-family: 'Roboto Mono', monospace;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 1px;
  color: var(--glimmer-text);
}

.admin-card {
  padding: 20px 24px;
}

.admin-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.admin-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--glimmer-text);
  display: flex;
  align-items: center;
  gap: 6px;
}

.admin-desc {
  font-size: 13px;
  color: var(--glimmer-text-3);
  margin: 6px 0 0;
}

.admin-add {
  display: flex;
  gap: 10px;
}

.admin-tags {
  margin-top: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.admin-tag {
  font-size: 14px;
  padding: 0 14px;
  height: 34px;
  line-height: 34px;
}

/* 手机端隐藏次要表格列（Element Plus 会将 class-name 应用到表头与单元格） */
@media (max-width: 768px) {
  :deep(.hide-sm) {
    display: none !important;
  }

  .cell-tags-sm {
    display: inline-flex;
  }

  .page-head {
    gap: 10px;
  }

  .page-head :deep(.el-button--large) {
    padding: 10px 16px;
    min-height: 42px;
  }

  .toolbar {
    padding: 10px 14px;
    gap: 10px;
  }

  .toolbar :deep(.el-radio-button__inner) {
    padding: 8px 10px;
  }

  .table-card {
    padding: 4px 8px;
  }

  .table-card :deep(.el-table__cell) {
    padding: 6px 0;
  }

  .table-card :deep(.el-table .cell) {
    padding: 0 8px;
  }

  .invite-card,
  .admin-card {
    padding: 16px;
  }

  .invite-generate {
    width: 100%;
  }

  .invite-generate > * {
    flex: 1;
  }

  .admin-add {
    width: 100%;
  }

  .admin-add .el-input {
    flex: 1;
  }

  .admin-tag {
    height: 32px;
    line-height: 32px;
  }
}
</style>
