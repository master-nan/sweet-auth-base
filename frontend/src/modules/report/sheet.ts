import { translate as t } from 'src/boot/i18n'
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

export type ReportSheetRange = {
  startRow: number
  startCol: number
  endRow: number
  endCol: number
}

export type ReportSheetClipboardCell = Pick<
  ReportSheetCell,
  'value' | 'binding' | 'style' | 'colspan' | 'rowspan'
>

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

export const reportRuntimeColumnAlias = (
  datasetId: string | undefined,
  fieldCode: string | undefined,
) => {
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

const reportBoundFields = (datasets: ReportDataset[], sheet: ReportSheetConfig) => {
  const fields: Array<{ dataset: ReportDataset; field: ReportField }> = []
  const seen = new Set<string>()
  sheet.cells.forEach((cell) => {
    const binding = cell.binding
    if (!binding?.dataset_id || !binding.field || binding.type === 'static') return
    const dataset = datasets.find((item) => item.id === binding.dataset_id)
    const field = dataset?.fields.find((item) => item.code === binding.field)
    if (!dataset || !field) return
    const key = `${dataset.id}:${field.code}`
    if (seen.has(key)) return
    seen.add(key)
    fields.push({ dataset, field })
  })
  return fields
}

export const reportSheetCellAt = (
  sheet: ReportSheetConfig,
  row: number,
  col: number,
): ReportSheetCell => {
  return (
    sheet.cells.find((cell) => cell.row === row && cell.col === col) || {
      id: makeReportCellId(row, col),
      row,
      col,
      value: '',
    }
  )
}

export const reportSheetCellSpan = (cell: ReportSheetCell, bounds?: Partial<ReportSheetBounds>) => {
  const rawRowspan = Math.max(cell.rowspan || 1, 1)
  const rawColspan = Math.max(cell.colspan || 1, 1)
  return {
    rowspan: Math.max(
      Math.min(rawRowspan, (bounds?.maxRow || cell.row + rawRowspan - 1) - cell.row + 1),
      1,
    ),
    colspan: Math.max(
      Math.min(rawColspan, (bounds?.maxCol || cell.col + rawColspan - 1) - cell.col + 1),
      1,
    ),
  }
}

export const reportSheetIsCoveredCell = (sheet: ReportSheetConfig, row: number, col: number) => {
  return sheet.cells.some((cell) => {
    if (cell.row === row && cell.col === col) return false
    const { rowspan, colspan } = reportSheetCellSpan(cell)
    return (
      row >= cell.row && row < cell.row + rowspan && col >= cell.col && col < cell.col + colspan
    )
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

export const reportNormalizeSheetRange = (range: ReportSheetRange): ReportSheetBounds => ({
  minRow: Math.min(range.startRow, range.endRow),
  maxRow: Math.max(range.startRow, range.endRow),
  minCol: Math.min(range.startCol, range.endCol),
  maxCol: Math.max(range.startCol, range.endCol),
})

export const reportSheetRangeContains = (
  range: ReportSheetRange | null | undefined,
  row: number,
  col: number,
) => {
  if (!range) return false
  const bounds = reportNormalizeSheetRange(range)
  return (
    row >= bounds.minRow && row <= bounds.maxRow && col >= bounds.minCol && col <= bounds.maxCol
  )
}

export const reportSheetClipboardMatrix = (
  sheet: ReportSheetConfig,
  bounds: ReportSheetBounds,
): ReportSheetClipboardCell[][] => {
  const cells = new Map(sheet.cells.map((cell) => [`${cell.row}:${cell.col}`, cell]))
  const matrix: ReportSheetClipboardCell[][] = []
  for (let row = bounds.minRow; row <= bounds.maxRow; row += 1) {
    const values: ReportSheetClipboardCell[] = []
    for (let col = bounds.minCol; col <= bounds.maxCol; col += 1) {
      const cell = cells.get(`${row}:${col}`)
      values.push({
        value: cell?.value || '',
        ...(cell?.binding ? { binding: { ...cell.binding } } : {}),
        ...(cell?.style ? { style: { ...cell.style } } : {}),
        ...(cell?.colspan && cell.colspan > 1 ? { colspan: cell.colspan } : {}),
        ...(cell?.rowspan && cell.rowspan > 1 ? { rowspan: cell.rowspan } : {}),
      })
    }
    matrix.push(values)
  }
  return matrix
}

export const reportPasteSheetCells = (
  sheet: ReportSheetConfig,
  startRow: number,
  startCol: number,
  matrix: ReportSheetClipboardCell[][],
) => {
  const next = cloneReportSheet(sheet)
  const cellMap = new Map(next.cells.map((cell) => [`${cell.row}:${cell.col}`, cell]))
  matrix.forEach((rowCells, rowOffset) => {
    rowCells.forEach((source, colOffset) => {
      const row = startRow + rowOffset
      const col = startCol + colOffset
      if (row < 1 || row > next.rows || col < 1 || col > next.cols) return
      const cell: ReportSheetCell = {
        id: makeReportCellId(row, col),
        row,
        col,
        value: source.value || '',
        ...(source.binding ? { binding: { ...source.binding } } : {}),
        ...(source.style ? { style: { ...source.style } } : {}),
        ...(source.colspan && source.colspan > 1
          ? { colspan: Math.min(source.colspan, next.cols - col + 1) }
          : {}),
        ...(source.rowspan && source.rowspan > 1
          ? { rowspan: Math.min(source.rowspan, next.rows - row + 1) }
          : {}),
      }
      const key = `${row}:${col}`
      if (hasClipboardCellConfig(cell)) cellMap.set(key, cell)
      else cellMap.delete(key)
    })
  })
  next.cells = [...cellMap.values()].sort((a, b) => a.row - b.row || a.col - b.col)
  return next
}

export const reportInsertSheetRow = (sheet: ReportSheetConfig, afterRow: number) => {
  const next = cloneReportSheet(sheet)
  const insertAt = Math.min(Math.max(afterRow + 1, 1), next.rows + 1)
  next.cells = next.cells.map((cell) => {
    const span = Math.max(cell.rowspan || 1, 1)
    if (cell.row >= insertAt) {
      return { ...cell, row: cell.row + 1, id: makeReportCellId(cell.row + 1, cell.col) }
    }
    if (cell.row < insertAt && cell.row + span > insertAt) {
      return { ...cell, rowspan: span + 1 }
    }
    return cell
  })
  next.rows += 1
  next.detail_rows = shiftMarkersForInsert(next.detail_rows, insertAt)
  next.summary_rows = shiftMarkersForInsert(next.summary_rows, insertAt)
  next.row_heights = shiftSizesForInsert(next.row_heights, insertAt)
  return next
}

export const reportDeleteSheetRow = (sheet: ReportSheetConfig, row: number) => {
  if (sheet.rows <= 8 || row < 1 || row > sheet.rows) return cloneReportSheet(sheet)
  const next = cloneReportSheet(sheet)
  next.cells = next.cells.flatMap((cell) => {
    const span = Math.max(cell.rowspan || 1, 1)
    if (cell.row > row) {
      return [{ ...cell, row: cell.row - 1, id: makeReportCellId(cell.row - 1, cell.col) }]
    }
    if (cell.row === row) {
      return span > 1 ? [{ ...cell, rowspan: span - 1 }] : []
    }
    if (cell.row < row && cell.row + span - 1 >= row) {
      return [{ ...cell, rowspan: Math.max(span - 1, 1) }]
    }
    return [cell]
  })
  next.rows -= 1
  next.detail_rows = shiftMarkersForDelete(next.detail_rows, row)
  next.summary_rows = shiftMarkersForDelete(next.summary_rows, row)
  next.row_heights = shiftSizesForDelete(next.row_heights, row)
  return next
}

export const reportInsertSheetColumn = (sheet: ReportSheetConfig, afterCol: number) => {
  const next = cloneReportSheet(sheet)
  const insertAt = Math.min(Math.max(afterCol + 1, 1), next.cols + 1)
  next.cells = next.cells.map((cell) => {
    const span = Math.max(cell.colspan || 1, 1)
    if (cell.col >= insertAt) {
      return { ...cell, col: cell.col + 1, id: makeReportCellId(cell.row, cell.col + 1) }
    }
    if (cell.col < insertAt && cell.col + span > insertAt) {
      return { ...cell, colspan: span + 1 }
    }
    return cell
  })
  next.cols += 1
  next.column_widths = shiftSizesForInsert(next.column_widths, insertAt)
  return next
}

export const reportDeleteSheetColumn = (sheet: ReportSheetConfig, col: number) => {
  if (sheet.cols <= 6 || col < 1 || col > sheet.cols) return cloneReportSheet(sheet)
  const next = cloneReportSheet(sheet)
  next.cells = next.cells.flatMap((cell) => {
    const span = Math.max(cell.colspan || 1, 1)
    if (cell.col > col) {
      return [{ ...cell, col: cell.col - 1, id: makeReportCellId(cell.row, cell.col - 1) }]
    }
    if (cell.col === col) {
      return span > 1 ? [{ ...cell, colspan: span - 1 }] : []
    }
    if (cell.col < col && cell.col + span - 1 >= col) {
      return [{ ...cell, colspan: Math.max(span - 1, 1) }]
    }
    return [cell]
  })
  next.cols -= 1
  next.column_widths = shiftSizesForDelete(next.column_widths, col)
  return next
}

const cloneReportSheet = (sheet: ReportSheetConfig): ReportSheetConfig => ({
  ...sheet,
  cells: sheet.cells.map((cell) => ({
    ...cell,
    ...(cell.binding ? { binding: { ...cell.binding } } : {}),
    ...(cell.style ? { style: { ...cell.style } } : {}),
  })),
  detail_rows: [...(sheet.detail_rows || [])],
  summary_rows: [...(sheet.summary_rows || [])],
  column_widths: { ...(sheet.column_widths || {}) },
  row_heights: { ...(sheet.row_heights || {}) },
})

const hasClipboardCellConfig = (cell: ReportSheetClipboardCell) =>
  Boolean(
    cell.value ||
      cell.binding?.field ||
      cell.binding?.formula ||
      (cell.style && Object.keys(cell.style).length) ||
      (cell.colspan && cell.colspan > 1) ||
      (cell.rowspan && cell.rowspan > 1),
  )

const shiftMarkersForInsert = (markers: number[] | undefined, insertAt: number) =>
  (markers || []).map((item) => (item >= insertAt ? item + 1 : item))

const shiftMarkersForDelete = (markers: number[] | undefined, deleted: number) =>
  (markers || [])
    .filter((item) => item !== deleted)
    .map((item) => (item > deleted ? item - 1 : item))

const shiftSizesForInsert = (sizes: Record<string, number> | undefined, insertAt: number) => {
  const shifted: Record<string, number> = {}
  Object.entries(sizes || {}).forEach(([rawIndex, size]) => {
    const index = Number(rawIndex)
    shifted[String(index >= insertAt ? index + 1 : index)] = size
  })
  return shifted
}

const shiftSizesForDelete = (sizes: Record<string, number> | undefined, deleted: number) => {
  const shifted: Record<string, number> = {}
  Object.entries(sizes || {}).forEach(([rawIndex, size]) => {
    const index = Number(rawIndex)
    if (index === deleted) return
    shifted[String(index > deleted ? index - 1 : index)] = size
  })
  return shifted
}

export const reportSheetMarkedRows = (sheet: ReportSheetConfig) => {
  const rows = new Set<number>()
  ;(sheet.detail_rows || []).forEach((row) => rows.add(row))
  ;(sheet.summary_rows || []).forEach((row) => rows.add(row))
  return [...rows].sort((a, b) => a - b)
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

export const collectReportUsedFields = (datasets: ReportDataset[], sheet: ReportSheetConfig) => {
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

export const reportSampleCellValue = (field: ReportField, rowIndex: number, fieldIndex: number) => {
  if (field.role === 'metric') return rowIndex * 1000 + fieldIndex * 120
  if (field.role === 'time') return `2026-07-${String(rowIndex + fieldIndex).padStart(2, '0')}`
  if (field.code.includes('status'))
    return (
      [t('ui.pendingDispatch'), t('ui.createdStatus'), t('ui.sended')][rowIndex - 1] ||
      t('ui.createdStatus')
    )
  return `${field.name}${rowIndex}`
}

export const buildReportLocalPreview = (
  datasets: ReportDataset[],
  sheet: ReportSheetConfig,
): ReportPreviewRes => {
  const boundFields = reportBoundFields(datasets, sheet)
  const fallbackDataset = datasets.find((item) => item.primary) || datasets[0]
  const fallbackFields =
    boundFields.length || !fallbackDataset
      ? boundFields
      : fallbackDataset.fields.slice(0, 6).map((field) => ({ dataset: fallbackDataset, field }))

  return {
    columns: fallbackFields.map((item) => item.field),
    datasets,
    rows: [1, 2, 3].map((id) => {
      const row: Record<string, unknown> = { id }
      fallbackFields.forEach(({ dataset, field }, index) => {
        const value = reportSampleCellValue(field, id, index)
        row[reportRuntimeColumnAlias(dataset.id, field.code)] = value
        row[reportRuntimeColumnAlias(dataset.source_code, field.code)] = value
        row[field.code] = value
      })
      return row
    }),
    total: 3,
  }
}

export const reportCellId = makeReportCellId
