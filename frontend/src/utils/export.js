// CSV 导出工具：带 UTF-8 BOM，Excel 直接打开中文不乱码
function csvCell(v) {
  const s = v == null ? '' : String(v)
  return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

// downloadCSV('文件名.csv', ['表头1', '表头2'], [['a', 'b'], ...])
export function downloadCSV(filename, header, rows) {
  const lines = [header, ...rows].map(r => r.map(csvCell).join(','))
  const blob = new Blob(['\ufeff' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename.endsWith('.csv') ? filename : filename + '.csv'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
