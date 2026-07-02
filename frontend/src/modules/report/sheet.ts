import { makeReportCellId } from './schema'
import type {
  ReportCellBindingType,
  ReportDataset,
  ReportField,
  ReportPreviewRes,
  ReportSheetCell,
  ReportSheetConfig,
} from './types'

export type ReportSheetBounds = {
  minRow: number
  maxRow: number
  minCol: number
  maxCol: number
}

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

export const reportRuntimeColumnAlias = (datasetId: string | undefined, fieldCode: string | undefined) => {
  const raw = `${datasetId || ''}__${fieldCode || ''}`.replace(/^_+|_+$/g, '') || fieldCode || ''
  const normalized = raw.replace(/[^A-Za-z0-9_]/g, '_')
  if (!normalized) return ''
  return /^[0-9]/.test(normalized) ? `c_${normalized}` : normalized
}

export const reportRuntimeCellValue = (
  row: Record<string, unknown> | undefined,
  cell: ReportSheetCell,
  datasets: ReportDataset[],
) => {
  const binding = cell.binding
  if (!binding?.field || !row) return cell.value || ''
  const dataset = datasets.find((item) => item.id === binding.dataset_id)
  const candidates = [
    reportRuntimeColumnAlias(binding.dataset_id, binding.field),
    reportRuntimeColumnAlias(dataset?.id, binding.field),
    reportRuntimeColumnAlias(dataset?.source_code, binding.field),
    binding.field,
  ].filter(Boolean)
  for (const key of candidates) {
    if (Object.prototype.hasOwnProperty.call(row, key)) {
      const value = row[key]
      if (value === null || value === undefined) return ''
      if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
        return String(value)
      }
      return JSON.stringify(value)
    }
  }
  return ''
}

export const reportSheetCellAt = (
  sheet: ReportSheetConfig,
  row: number,
  col: number,
): ReportSheetCell => {
  return sheet.cells.find((cell) => cell.row === row && cell.col === col) || {
    id: makeReportCellId(row, col),
    row,
    col,
    value: '',
  }
}

export const reportSheetCellSpan = (cell: ReportSheetCell, bounds?: Partial<ReportSheetBounds>) => {
  const rawRowspan = Math.max(cell.rowspan || 1, 1)
  const rawColspan = Math.max(cell.colspan || 1, 1)
  return {
    rowspan: Math.max(Math.min(rawRowspan, (bounds?.maxRow || cell.row + rawRowspan - 1) - cell.row + 1), 1),
    colspan: Math.max(Math.min(rawColspan, (bounds?.maxCol || cell.col + rawColspan - 1) - cell.col + 1), 1),
  }
}

export const reportSheetIsCoveredCell = (
  sheet: ReportSheetConfig,
  row: number,
  col: number,
) => {
  return sheet.cells.some((cell) => {
    if (cell.row === row && cell.col === col) return false
    const { rowspan, colspan } = reportSheetCellSpan(cell)
    return row >= cell.row && row < cell.row + rowspan && col >= cell.col && col < cell.col + colspan
  })
}

export const reportSheetUsedBounds = (sheet: ReportSheetConfig): ReportSheetBounds => {
  const used = sheet.cells.filter((cell) => {
    return Boolean(
      cell.value ||
      cell.binding?.field ||
      (cell.colspan && cell.colspan > 1) ||
      (cell.rowspan && cell.rowspan > 1),
    )
  })
  if (!used.length) {
    return { minRow: 1, maxRow: 1, minCol: 1, maxCol: 1 }
  }
  const minRow = Math.min(...used.map((cell) => cell.row))
  const minCol = Math.min(...used.map((cell) => cell.col))
  const maxRow = Math.max(1, ...used.map((cell) => cell.row + (cell.rowspan || 1) - 1))
  const maxCol = Math.max(1, ...used.map((cell) => cell.col + (cell.colspan || 1) - 1))
  return {
    minRow: Math.max(minRow, 1),
    maxRow: Math.min(Math.max(maxRow, 1), sheet.rows || maxRow),
    minCol: Math.max(minCol, 1),
    maxCol: Math.min(Math.max(maxCol, 1), sheet.cols || maxCol),
  }
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
    const field = dataset.fields.find((item) => item.code === binding.field)
    if (field) used.set(`${dataset.id}:${field.code}`, field)
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
