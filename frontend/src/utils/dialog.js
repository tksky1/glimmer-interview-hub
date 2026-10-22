import { ElMessageBox } from 'element-plus'

// 同一时刻只允许一个确认类弹窗存在：
// 连续点击（移动端双击/误触）时不会叠加出多个弹窗，后续调用直接忽略。
let locked = false

function acquire() {
  if (locked) return false
  locked = true
  return true
}

function release() {
  locked = false
}

const BASE = {
  customClass: 'glimmer-message-box',
  closeOnClickModal: false,
  distinguishCancelAndClose: true,
  confirmButtonText: '确定',
  cancelButtonText: '取消'
}

export async function confirmDialog(message, title, options = {}) {
  if (!acquire()) return Promise.reject(new Error('dialog-busy'))
  try {
    return await ElMessageBox.confirm(message, title, { ...BASE, type: 'warning', ...options })
  } finally {
    release()
  }
}

export async function alertDialog(message, title, options = {}) {
  if (!acquire()) return Promise.reject(new Error('dialog-busy'))
  try {
    return await ElMessageBox.alert(message, title, {
      ...BASE,
      type: 'info',
      customClass: 'glimmer-message-box',
      confirmButtonText: '知道了',
      ...options
    })
  } finally {
    release()
  }
}
