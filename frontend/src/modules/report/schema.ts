import { translate as t } from 'src/i18n/runtime/instance'
import type {
  Report,
  ReportDataset,
  ReportField,
  ReportKind,
  ReportLayoutConfig,
  ReportParameterOperator,
  ReportParameterType,
  ReportSheetCell,
  ReportSheetConfig,
} from './types'

export const REPORT_SCHEMA_VERSION = 1

export const makeReportCellId = (row: number, col: number) => `${row}:${col}`

export const normalizeReportKind = (value: unknown): ReportKind => {
  if (value === 'summary') return value
  return 'detail'
}

export const guessReportFieldRole = (
  code: string,
  type: string,
): NonNullable<ReportField['role']> => {
  const lower = code.toLowerCase()
  const normalizedType = String(type || '').toLowerCase()
  if (
    lower.includes('date') ||
    lower.includes('time') ||
    ['6', '7', '8'].includes(normalizedType) ||
    ['date', 'datetime', 'timestamp', 'time'].some((item) => normalizedType.includes(item))
  )
    return 'time'
  if (lower === 'id' || lower.endsWith('_id')) {
    return 'dimension'
  }
  if (
    ['1', '2', '9', '11'].includes(normalizedType) ||
    ['number', 'numeric', 'decimal', 'float', 'double', 'int', 'bigint'].some((item) =>
      normalizedType.includes(item),
    )
  ) {
    return 'metric'
  }
  if (lower.includes('name') || lower.includes('status')) {
    return 'dimension'
  }
  return 'text'
}

export const reportParameterDefaultsForField = (
  field?: ReportField,
): { type: ReportParameterType; operator: ReportParameterOperator } => {
  if (!field) return { type: 'text', operator: 'like' }
  if (field.role === 'time') return { type: 'date_range', operator: 'between' }
  if (field.role === 'metric') return { type: 'number', operator: 'eq' }
  return { type: 'text', operator: 'like' }
}

export const createBlankReportSheet = (rows = 24, cols = 12): ReportSheetConfig => {
  return { rows, cols, scale: 0.85, detail_rows: [], summary_rows: [], cells: [] }
}

export const defaultReportSheet = (): ReportSheetConfig => createBlankReportSheet()

export const normalizeReportSheet = (sheet?: Partial<ReportSheetConfig>): ReportSheetConfig => {
  const rows = Math.max(Number(sheet?.rows || 24), 8)
  const cols = Math.max(Number(sheet?.cols || 12), 6)
  const blank = createBlankReportSheet(rows, cols)
  const incoming = new Map<string, ReportSheetCell>()
  ;(sheet?.cells || []).forEach((cell) => {
    const row = Number(cell.row)
    const col = Number(cell.col)
    if (!Number.isInteger(row) || !Number.isInteger(col)) return
    if (row < 1 || row > rows || col < 1 || col > cols) return
    const next = {
      ...cell,
      id: cell.id || makeReportCellId(row, col),
      row,
      col,
      value: cell.value || '',
    }
    if (hasReportCellConfig(next)) incoming.set(makeReportCellId(row, col), next)
  })
  blank.cells = [...incoming.values()].sort((a, b) => a.row - b.row || a.col - b.col)
  if (sheet?.active_cell) blank.active_cell = sheet.active_cell
  if (sheet?.scale) blank.scale = sheet.scale
  blank.detail_rows = normalizeSheetRows(sheet?.detail_rows, rows)
  blank.summary_rows = normalizeSheetRows(sheet?.summary_rows, rows)
  return blank
}

const normalizeSheetRows = (rows: number[] | undefined, maxRows: number) => {
  const unique = new Set<number>()
  ;(rows || []).forEach((row) => {
    const value = Number(row)
    if (Number.isInteger(value) && value >= 1 && value <= maxRows) unique.add(value)
  })
  return [...unique].sort((a, b) => a - b)
}

export const hasReportCellConfig = (cell: Partial<ReportSheetCell>) =>
  Boolean(
    cell.value ||
      cell.binding?.field ||
      cell.binding?.formula ||
      (cell.style && Object.keys(cell.style).length > 0) ||
      (cell.colspan && cell.colspan > 1) ||
      (cell.rowspan && cell.rowspan > 1),
  )

export const createReportLayout = (report?: Partial<Report>): ReportLayoutConfig => ({
  version: REPORT_SCHEMA_VERSION,
  view: 'sheet',
  title: report?.report_name || report?.name || t('ui.unnamedReport'),
  subtitle: report?.description || '',
  kind: normalizeReportKind(report?.report_kind),
  datasets: [],
  dataset_joins: [],
  parameters: [],
  sheet: defaultReportSheet(),
  runtime_display: 'paged',
  runtime_page_size: 20,
})

export const primaryTableDataset = (datasets: ReportDataset[] = []) =>
  datasets.find((item) => item.primary && item.type === 'table') ||
  datasets.find((item) => item.type === 'table')

export const ensureOnePrimaryDataset = (datasets: ReportDataset[]) => {
  let primaryFound = false
  return datasets.map((dataset) => {
    const primary = dataset.type === 'table' && (dataset.primary || !primaryFound)
    if (primary) primaryFound = true
    return { ...dataset, primary }
  })
}
