import { makeReportCellId } from './schema'
import type {
  ReportCellBindingType,
  ReportDataset,
  ReportField,
  ReportPreviewRes,
  ReportSheetCell,
  ReportSheetConfig,
} from './types'

export const reportColumnName = (col: number) => {
  let name = ''
  let value = col
  while (value > 0) {
    const index = (value - 1) % 26
    name = String.fromCharCode(65 + index) + name
    value = Math.floor((value - index) / 26)
  }
  return name
}

export const reportFieldIcon = (field: ReportField) => {
  if (field.role === 'metric') return 'pin'
  if (field.role === 'time') return 'event'
  if (field.role === 'dimension') return 'category'
  return 'text_fields'
}

export const defaultReportBindingType = (field: ReportField): ReportCellBindingType => {
  if (field.role === 'metric') return 'sum'
  if (field.role === 'dimension' || field.role === 'time') return 'group'
  return 'field'
}

export const reportBindingText = (
  type: ReportCellBindingType,
  dataset: ReportDataset,
  field: ReportField,
) => {
  const prefix = dataset.name || dataset.id
  if (type === 'group') return `${prefix}.G(${field.name})`
  if (type === 'sum') return `SUM(${prefix}.${field.code})`
  if (type === 'count') return `COUNT(${prefix}.${field.code})`
  if (type === 'formula') return `=${field.code}`
  return `${prefix}.S(${field.name})`
}

export const reportCellStyle = (cell: ReportSheetCell) => {
  const style = cell.style || {}
  return {
    gridColumn: cell.colspan && cell.colspan > 1 ? `span ${cell.colspan}` : undefined,
    gridRow: cell.rowspan && cell.rowspan > 1 ? `span ${cell.rowspan}` : undefined,
    fontWeight: style.bold ? 800 : 500,
    fontStyle: style.italic ? 'italic' : 'normal',
    textAlign: style.align || 'left',
    background: style.background || '#fff',
    color: style.color || '#172033',
  }
}

export const collectReportUsedFields = (
  datasets: ReportDataset[],
  sheet: ReportSheetConfig,
) => {
  const used = new Map<string, ReportField>()
  const primary = datasets.find((item) => item.primary) || datasets[0]
  sheet.cells.forEach((cell) => {
    const binding = cell.binding
    if (!binding?.field || !binding.dataset_id) return
    const dataset = datasets.find((item) => item.id === binding.dataset_id)
    if (!dataset || dataset.type !== 'table') return
    if (primary && dataset.id !== primary.id) return
    const field = dataset.fields.find((item) => item.code === binding.field)
    if (field) used.set(field.code, field)
  })
  if (used.size === 0 && primary) {
    primary.fields.slice(0, 6).forEach((field) => used.set(field.code, field))
  }
  return [...used.values()]
}

export const reportSampleCellValue = (
  field: ReportField,
  rowIndex: number,
  fieldIndex: number,
) => {
  if (field.role === 'metric') return rowIndex * 1000 + fieldIndex * 120
  if (field.role === 'time') return `2026-07-${String(rowIndex + fieldIndex).padStart(2, '0')}`
  if (field.code.includes('status')) return ['待发车', '已创建', '已发车'][rowIndex - 1] || '已创建'
  return `${field.name}${rowIndex}`
}

export const buildReportLocalPreview = (fields: ReportField[]): ReportPreviewRes => ({
  columns: fields,
  rows: [1, 2, 3].map((id) => {
    const row: Record<string, unknown> = { id }
    fields.forEach((field, index) => {
      row[field.code] = reportSampleCellValue(field, id, index)
    })
    return row
  }),
  total: 3,
})

export const reportCellId = makeReportCellId
