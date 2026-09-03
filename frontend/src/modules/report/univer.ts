import {
  BooleanNumber,
  HorizontalAlign,
  LocaleType,
  type ICellData,
  type IStyleData,
  type IWorkbookData,
  type IWorksheetData,
} from '@univerjs/presets'

import { hasReportCellConfig, makeReportCellId, normalizeReportSheet } from './schema'
import type {
  ReportCellBinding,
  ReportCellStyle,
  ReportSheetCell,
  ReportSheetConfig,
} from './types'

export const REPORT_UNIVER_WORKBOOK_ID = 'sweet-report-workbook'
export const REPORT_UNIVER_WORKSHEET_ID = 'report-sheet'

const REPORT_CUSTOM_KEY = 'sweetReport'
const REPORT_UNIVER_SCHEMA_VERSION = 1
const DEFAULT_COLUMN_WIDTH = 120
const DEFAULT_ROW_HEIGHT = 34

interface ReportUniverCellCustom {
  binding?: ReportCellBinding
}

interface ReportUniverWorksheetCustom {
  activeCell?: string
  scale?: number
}

type ReportUniverRowType = 'normal' | 'detail' | 'groupSummary' | 'summary'

interface ReportUniverRowCustom {
  rowType: ReportUniverRowType
}

export const getReportUniverFillSourceIndex = (
  sourceIndices: number[],
  targetOffset: number,
  reverse = false,
) => {
  if (sourceIndices.length === 0) return undefined
  const sourceOffset = targetOffset % sourceIndices.length
  return reverse
    ? sourceIndices[sourceIndices.length - sourceOffset - 1]
    : sourceIndices[sourceOffset]
}

export const copyReportUniverCellMetadata = (
  source: ICellData | null | undefined,
  target: ICellData,
): ICellData => {
  const reportCustom = source?.custom?.[REPORT_CUSTOM_KEY]
  if (!reportCustom) return target
  const next: ICellData = {
    ...target,
    custom: {
      ...(target.custom || {}),
      [REPORT_CUSTOM_KEY]: structuredClone(reportCustom),
    },
  }
  const binding = next.custom?.[REPORT_CUSTOM_KEY]?.binding as
    | { type?: string; formula?: string }
    | undefined
  if (binding?.type === 'formula' && next.f) binding.formula = next.f
  return next
}

const cloneBinding = (binding?: ReportCellBinding) =>
  binding ? ({ ...binding } satisfies ReportCellBinding) : undefined

const reportStyleToUniver = (style?: ReportCellStyle): IStyleData | undefined => {
  if (!style || Object.keys(style).length === 0) return undefined
  const align =
    style.align === 'center'
      ? HorizontalAlign.CENTER
      : style.align === 'right'
        ? HorizontalAlign.RIGHT
        : style.align === 'left'
          ? HorizontalAlign.LEFT
          : undefined
  return {
    ...((style.univer || {}) as IStyleData),
    ...(style.bold !== undefined && { bl: style.bold ? BooleanNumber.TRUE : BooleanNumber.FALSE }),
    ...(style.italic !== undefined && {
      it: style.italic ? BooleanNumber.TRUE : BooleanNumber.FALSE,
    }),
    ...(align !== undefined && { ht: align }),
    ...(style.background && { bg: { rgb: style.background } }),
    ...(style.color && { cl: { rgb: style.color } }),
  }
}

const univerStyleToReport = (style?: IStyleData | null): ReportCellStyle | undefined => {
  if (!style) return undefined
  const align =
    style.ht === HorizontalAlign.CENTER
      ? 'center'
      : style.ht === HorizontalAlign.RIGHT
        ? 'right'
        : style.ht === HorizontalAlign.LEFT
          ? 'left'
          : undefined
  const result: ReportCellStyle = {
    ...(style.bl !== undefined && { bold: style.bl === BooleanNumber.TRUE }),
    ...(style.it !== undefined && { italic: style.it === BooleanNumber.TRUE }),
    ...(align && { align }),
    ...(style.bg?.rgb && { background: style.bg.rgb }),
    ...(style.cl?.rgb && { color: style.cl.rgb }),
    ...(Object.keys(style).some((key) => !['bl', 'it', 'ht', 'bg', 'cl'].includes(key)) && {
      univer: structuredClone(style) as Record<string, unknown>,
    }),
  }
  return Object.keys(result).length > 0 ? result : undefined
}

const resolveCellStyle = (cell: ICellData, snapshot: IWorkbookData) => {
  if (!cell.s) return undefined
  if (typeof cell.s !== 'string') return cell.s
  return snapshot.styles[cell.s] || undefined
}

const reportCellToUniver = (cell: ReportSheetCell): ICellData => {
  const formula = cell.binding?.type === 'formula' ? cell.binding.formula || cell.value : ''
  const style = reportStyleToUniver(cell.style)
  const custom: ReportUniverCellCustom = {
    ...(cell.binding && { binding: { ...cell.binding } }),
  }
  return {
    ...(formula ? { f: formula.startsWith('=') ? formula : `=${formula}` } : { v: cell.value }),
    ...(style && { s: style }),
    ...(Object.keys(custom).length > 0 && { custom: { [REPORT_CUSTOM_KEY]: custom } }),
  }
}

export const reportSheetToUniverSnapshot = (
  source: ReportSheetConfig,
  workbookName = '报表设计器',
): IWorkbookData => {
  const sheet = normalizeReportSheet(source)
  const cellData: IWorksheetData['cellData'] = {}
  const mergeData: IWorksheetData['mergeData'] = []
  sheet.cells.forEach((cell) => {
    cellData[cell.row - 1] ||= {}
    cellData[cell.row - 1]![cell.col - 1] = reportCellToUniver(cell)
    if ((cell.rowspan || 1) > 1 || (cell.colspan || 1) > 1) {
      mergeData.push({
        startRow: cell.row - 1,
        endRow: Math.min(sheet.rows - 1, cell.row - 1 + (cell.rowspan || 1) - 1),
        startColumn: cell.col - 1,
        endColumn: Math.min(sheet.cols - 1, cell.col - 1 + (cell.colspan || 1) - 1),
      })
    }
  })

  const rowData: IWorksheetData['rowData'] = {}
  Object.entries(sheet.row_heights || {}).forEach(([row, height]) => {
    rowData[Number(row) - 1] = { h: height }
  })
  for (let row = 1; row <= sheet.rows; row += 1) {
    const rowType: ReportUniverRowType = sheet.group_summary_rows?.includes(row)
      ? 'groupSummary'
      : sheet.summary_rows?.includes(row)
        ? 'summary'
        : sheet.detail_rows?.includes(row)
          ? 'detail'
          : 'normal'
    const rowIndex = row - 1
    rowData[rowIndex] = {
      ...(rowData[rowIndex] || {}),
      custom: {
        ...(rowData[rowIndex]?.custom || {}),
        [REPORT_CUSTOM_KEY]: { rowType } satisfies ReportUniverRowCustom,
      },
    }
  }
  const columnData: IWorksheetData['columnData'] = {}
  Object.entries(sheet.column_widths || {}).forEach(([column, width]) => {
    columnData[Number(column) - 1] = { w: width }
  })

  const worksheetCustom: ReportUniverWorksheetCustom = {
    ...(sheet.active_cell && { activeCell: sheet.active_cell }),
    ...(sheet.scale !== undefined && { scale: sheet.scale }),
  }
  return {
    id: REPORT_UNIVER_WORKBOOK_ID,
    name: workbookName,
    appVersion: '0.25.1',
    locale: LocaleType.ZH_CN,
    styles: {},
    sheetOrder: [REPORT_UNIVER_WORKSHEET_ID],
    sheets: {
      [REPORT_UNIVER_WORKSHEET_ID]: {
        id: REPORT_UNIVER_WORKSHEET_ID,
        name: '报表',
        rowCount: sheet.rows,
        columnCount: sheet.cols,
        zoomRatio: sheet.scale || 1,
        defaultColumnWidth: DEFAULT_COLUMN_WIDTH,
        defaultRowHeight: DEFAULT_ROW_HEIGHT,
        mergeData,
        cellData,
        rowData,
        columnData,
        rowHeader: { width: 46 },
        columnHeader: { height: 28 },
        showGridlines: BooleanNumber.TRUE,
        custom: { [REPORT_CUSTOM_KEY]: worksheetCustom },
      },
    },
    custom: {
      [REPORT_CUSTOM_KEY]: { version: REPORT_UNIVER_SCHEMA_VERSION },
    },
  }
}

const reportCellFromUniver = (
  row: number,
  col: number,
  cell: ICellData,
  snapshot: IWorkbookData,
): ReportSheetCell | undefined => {
  const custom = cell.custom?.[REPORT_CUSTOM_KEY] as ReportUniverCellCustom | undefined
  const formula = typeof cell.f === 'string' ? cell.f : ''
  const binding = cloneBinding(
    custom?.binding || (formula ? { type: 'formula', formula } : undefined),
  )
  if (binding?.type === 'formula') binding.formula = formula || binding.formula || ''
  const result: ReportSheetCell = {
    id: makeReportCellId(row, col),
    row,
    col,
    value: formula || (cell.v === null || cell.v === undefined ? '' : String(cell.v)),
    ...(binding && { binding }),
    ...(univerStyleToReport(resolveCellStyle(cell, snapshot)) && {
      style: univerStyleToReport(resolveCellStyle(cell, snapshot)),
    }),
  }
  return hasReportCellConfig(result) ? result : undefined
}

export const univerSnapshotToReportSheet = (snapshot: IWorkbookData): ReportSheetConfig => {
  const sheetId = snapshot.sheetOrder[0] || REPORT_UNIVER_WORKSHEET_ID
  const worksheet = snapshot.sheets[sheetId] || {}
  const rows = Math.max(Number(worksheet.rowCount || 24), 8)
  const cols = Math.max(Number(worksheet.columnCount || 12), 6)
  const worksheetCustom = worksheet.custom?.[REPORT_CUSTOM_KEY] as
    | ReportUniverWorksheetCustom
    | undefined
  const cells = new Map<string, ReportSheetCell>()

  Object.entries(worksheet.cellData || {}).forEach(([rowIndex, columns]) => {
    Object.entries(columns || {}).forEach(([columnIndex, cell]) => {
      if (!cell) return
      const row = Number(rowIndex) + 1
      const col = Number(columnIndex) + 1
      const converted = reportCellFromUniver(row, col, cell, snapshot)
      if (converted) cells.set(converted.id, converted)
    })
  })
  ;(worksheet.mergeData || []).forEach((merge) => {
    const row = merge.startRow + 1
    const col = merge.startColumn + 1
    const id = makeReportCellId(row, col)
    const cell = cells.get(id) || { id, row, col, value: '' }
    const rowspan = merge.endRow - merge.startRow + 1
    const colspan = merge.endColumn - merge.startColumn + 1
    if (rowspan > 1) cell.rowspan = rowspan
    if (colspan > 1) cell.colspan = colspan
    cells.set(id, cell)
  })

  const rowHeights: Record<string, number> = {}
  const detailRows: number[] = []
  const summaryRows: number[] = []
  const groupSummaryRows: number[] = []
  Object.entries(worksheet.rowData || {}).forEach(([rowIndex, row]) => {
    const reportRow = Number(rowIndex) + 1
    if (row?.h) rowHeights[String(reportRow)] = row.h
    const rowCustom = row?.custom?.[REPORT_CUSTOM_KEY] as ReportUniverRowCustom | undefined
    if (rowCustom?.rowType === 'detail') detailRows.push(reportRow)
    if (rowCustom?.rowType === 'summary') summaryRows.push(reportRow)
    if (rowCustom?.rowType === 'groupSummary') {
      groupSummaryRows.push(reportRow)
      summaryRows.push(reportRow)
    }
  })
  const columnWidths: Record<string, number> = {}
  Object.entries(worksheet.columnData || {}).forEach(([columnIndex, column]) => {
    if (column?.w) columnWidths[String(Number(columnIndex) + 1)] = column.w
  })

  return normalizeReportSheet({
    rows,
    cols,
    scale:
      typeof worksheet.zoomRatio === 'number' ? worksheet.zoomRatio : worksheetCustom?.scale,
    active_cell: worksheetCustom?.activeCell,
    detail_rows: detailRows,
    summary_rows: summaryRows,
    group_summary_rows: groupSummaryRows,
    row_heights: rowHeights,
    column_widths: columnWidths,
    cells: [...cells.values()],
  })
}
