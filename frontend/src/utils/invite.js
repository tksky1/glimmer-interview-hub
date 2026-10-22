import http from '@/utils/http'
import { ElMessage } from 'element-plus'
import { directionLabel } from '@/apis'

// 待核销的邀请码（由 index.html 前置脚本从 URL 捕获并暂存至此）
const PENDING_KEY = 'glimmer_pending_invite'

// 读取暂存的邀请码
export function pendingInviteCode() {
  return sessionStorage.getItem(PENDING_KEY) || ''
}

export function clearPendingInvite() {
  sessionStorage.removeItem(PENDING_KEY)
}

// 核销暂存的邀请码（登录成功后调用）；成功返回方向，无待核销码返回 null
export async function redeemPendingInvite() {
  const code = pendingInviteCode()
  if (!code) return null
  try {
    const res = await http.post('/invite/redeem', { code })
    clearPendingInvite()
    ElMessage.success(`方向绑定成功：${directionLabel(res.data?.direction)}`)
    return res.data?.direction || ''
  } catch (e) {
    return null
  }
}
