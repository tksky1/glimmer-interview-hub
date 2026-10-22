import http from '@/utils/http'

// ===== 成员端 =====
export const meApi = () => http.get('/me')

export const sessionsApi = () => http.get('/sessions')

// 日历视图：区间内的场次与时段（含首选人数、我的心仪时间）
// direction 为空时返回本人已获邀请的全部方向
export const calendarApi = (start, end, direction = '') =>
  http.get('/sessions/calendar', { params: direction ? { start, end, direction } : { start, end } })

// 保存真实姓名、QQ 与手机号（管理端分配结果表需要）
export const saveProfileApi = (name, qq, phone) => http.put('/profile', { name, qq, phone })

export const sessionDetailApi = id => http.get(`/sessions/${id}`)

export const savePrefsApi = (sessionId, slotIds) =>
  http.put('/preferences', { sessionId, slotIds })

export const myScheduleApi = () => http.get('/my/schedule')

// 邀请码绑定方向
export const redeemInviteApi = code => http.post('/invite/redeem', { code })

// ===== 管理员端 =====
export const createSessionApi = data => http.post('/admin/sessions', data)

export const updateSessionApi = (id, data) => http.put(`/admin/sessions/${id}`, data)

export const deleteSessionApi = id => http.delete(`/admin/sessions/${id}`)

export const publishSessionApi = id => http.post(`/admin/sessions/${id}/publish`)

export const generateAssignmentApi = id => http.post(`/admin/sessions/${id}/assign`)

export const getAssignmentApi = id => http.get(`/admin/sessions/${id}/assignment`)

export const reassignApi = (id, slotId) => http.put(`/admin/assignments/${id}`, { slotId })

// 把未分配的成员手动安排到空闲时段
export const manualAssignApi = (sessionId, username, slotId) =>
  http.post(`/admin/sessions/${sessionId}/manual-assign`, { username, slotId })

export const swapApi = (id, otherId) => http.post(`/admin/assignments/${id}/swap`, { otherId })

export const confirmAssignmentApi = id => http.post(`/admin/sessions/${id}/confirm`)

export const adminsApi = () => http.get('/admin/admins')

export const addAdminApi = username => http.post('/admin/admins', { username })

export const removeAdminApi = username => http.delete(`/admin/admins/${username}`)

// 邀请码管理
export const invitesApi = () => http.get('/admin/invites')

export const generateInvitesApi = (direction, note, count) =>
  http.post('/admin/invites', { direction, note, count })

export const deleteInviteApi = id => http.delete(`/admin/invites/${id}`)

// 方向常量（与后端一致）
export const DIRECTIONS = [
  { value: 'c', label: 'C / 计系', color: '#00B0FF' },
  { value: 'java', label: 'Java 后端', color: '#1E9600' },
  { value: 'web', label: 'Web 前端', color: '#936BB3' },
  { value: 'ml', label: 'ML 算法', color: '#E6A23C' }
]

export function directionLabel(d) {
  return DIRECTIONS.find(x => x.value === d)?.label || d || '未设置'
}

export function directionColor(d) {
  return DIRECTIONS.find(x => x.value === d)?.color || '#909399'
}
