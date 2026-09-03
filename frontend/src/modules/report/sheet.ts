import { translate as t } from '@/boot/i18n'
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
  if (type === 'avg') return `AVG(${prefix}.${field.code})`
  if (type === 'max') return `MAX(${prefix}.${field.code})`
  if (type === 'min') return `MIN(${prefix}.${field.code})`
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

export type ReportRuntimeRowGroup = {
  key: string
  row: Record<string, unknown>
  rows: Record<string, unknown>[]
}

export const reportRuntimeRowGroups = (
  rows: Record<string, unknown>[],
  sheet: ReportSheetConfig,
  datasets: ReportDataset[],
): ReportRuntimeRowGroup[] => {
  const configuredDetailRows = new Set(sheet.detail_rows || [])
  const summaryRows = new Set(sheet.summary_rows || [])
  const groupCells = sheet.cells.filter((cell) => {
    if (cell.binding?.type !== 'group' || !cell.binding.field) return false
    if (summaryRows.has(cell.row)) return false
    return configuredDetailRows.size === 0 || configuredDetailRows.has(cell.row)
  })
  if (!groupCells.length) {
    return rows.map((row, index) => ({ key: `row:${index}`, row, rows: [row] }))
  }

  const groups = new Map<string, ReportRuntimeRowGroup>()
  rows.forEach((row) => {
    const values = groupCells.map((cell) => reportRuntimeCellValue(row, cell, datasets))
    const key = JSON.stringify(values)
    const existing = groups.get(key)
    if (existing) {
      existing.rows.push(row)
      return
    }
    groups.set(key, { key, row, rows: [row] })
  })
  return [...groups.values()]
}

export const reportRuntimeAggregateValue = (
  rows: Record<string, unknown>[],
  cell: ReportSheetCell,
  datasets: ReportDataset[],
) => {
  const type = cell.binding?.type
  const values = rows
    .map((row) => reportRuntimeCellValue(row, cell, datasets))
    .filter((value) => value !== '')
  if (type === 'count') return String(values.length)
  if (!values.length) return ''

  const numericValues = values.map(Number).filter(Number.isFinite)
  if (type === 'sum') return String(numericValues.reduce((sum, value) => sum + value, 0))
  if (type === 'avg') {
    if (!numericValues.length) return ''
    return String(numericValues.reduce((sum, value) => sum + value, 0) / numericValues.length)
  }
  if (type === 'min' || type === 'max') {
    if (numericValues.length === values.length) {
      return String(type === 'min' ? Math.min(...numericValues) : Math.max(...numericValues))
    }
    const sorted = [...values].sort((left, right) =>
      left.localeCompare(right, undefined, { numeric: true }),
    )
    return (type === 'min' ? sorted[0] : sorted[sorted.length - 1]) || ''
  }
  return ''
}

export type ReportFormulaError = 'syntax' | 'division_by_zero' | 'circular_reference'

export type ReportFormulaCellReference = {
  row: number
  col: number
}

export type ReportFormulaResolver = {
  cell: (reference: ReportFormulaCellReference) => unknown
  range: (start: ReportFormulaCellReference, end: ReportFormulaCellReference) => unknown[]
}

export type ReportFormulaResult = {
  value: string
  error?: ReportFormulaError
}

type ReportFormulaTokenType =
  | 'number'
  | 'cell'
  | 'identifier'
  | 'operator'
  | 'leftParen'
  | 'rightParen'
  | 'comma'
  | 'colon'
  | 'eof'

type ReportFormulaToken = {
  type: ReportFormulaTokenType
  value: string
}

class ReportFormulaFailure extends Error {
  constructor(readonly code: ReportFormulaError) {
    super(code)
  }
}

const reportFormulaErrors = new Set<ReportFormulaError>([
  'syntax',
  'division_by_zero',
  'circular_reference',
])

export const reportEvaluateFormula = (
  formula: string,
  resolver: ReportFormulaResolver,
): ReportFormulaResult => {
  try {
    const parser = new ReportFormulaParser(reportFormulaTokens(formula), resolver)
    const rawValue = parser.parse()
    const value = Math.abs(rawValue) < 1e-12 ? 0 : Number(rawValue.toPrecision(12))
    return { value: String(value) }
  } catch (error) {
    const code =
      error instanceof ReportFormulaFailure
        ? error.code
        : error instanceof Error && reportFormulaErrors.has(error.message as ReportFormulaError)
          ? (error.message as ReportFormulaError)
          : 'syntax'
    return { value: '', error: code }
  }
}

export const reportValidateFormula = (formula: string) =>
  reportEvaluateFormula(formula, {
    cell: () => 1,
    range: () => [1],
  }).error

export const reportShiftFormulaReferences = (
  formula: string,
  rowOffset: number,
  colOffset: number,
) =>
  formula.replace(
    /(^|[^A-Za-z0-9_])(\$?)([A-Za-z]+)(\$?)([1-9]\d*)(?![A-Za-z0-9_])/g,
    (
      _match,
      prefix: string,
      absoluteCol: string,
      colName: string,
      absoluteRow: string,
      row: string,
    ) => {
      const reference = reportFormulaReference(`${colName}${row}`)
      const nextRow = absoluteRow ? reference.row : reference.row + rowOffset
      const nextCol = absoluteCol ? reference.col : reference.col + colOffset
      if (nextRow < 1 || nextCol < 1) return `${prefix}#REF!`
      return `${prefix}${absoluteCol}${reportColumnName(nextCol)}${absoluteRow}${nextRow}`
    },
  )

class ReportFormulaParser {
  private index = 0

  constructor(
    private readonly tokens: ReportFormulaToken[],
    private readonly resolver: ReportFormulaResolver,
  ) {}

  parse() {
    const value = this.parseExpression()
    if (this.current().type !== 'eof') this.fail('syntax')
    return value
  }

  private parseExpression(): number {
    let value = this.parseTerm()
    while (this.isOperator('+') || this.isOperator('-')) {
      const operator = this.consume().value
      const right = this.parseTerm()
      value = operator === '+' ? value + right : value - right
    }
    return value
  }

  private parseTerm(): number {
    let value = this.parseUnary()
    while (this.isOperator('*') || this.isOperator('/') || this.isOperator('%')) {
      const operator = this.consume().value
      const right = this.parseUnary()
      if ((operator === '/' || operator === '%') && right === 0) this.fail('division_by_zero')
      if (operator === '*') value *= right
      else if (operator === '/') value /= right
      else value %= right
    }
    return value
  }

  private parseUnary(): number {
    if (this.isOperator('+')) {
      this.consume()
      return this.parseUnary()
    }
    if (this.isOperator('-')) {
      this.consume()
      return -this.parseUnary()
    }
    return this.parsePrimary()
  }

  private parsePrimary(): number {
    const token = this.current()
    if (token.type === 'number') {
      this.consume()
      return this.toNumber(token.value)
    }
    if (token.type === 'cell') {
      this.consume()
      return this.toNumber(this.resolver.cell(reportFormulaReference(token.value)))
    }
    if (token.type === 'identifier') return this.parseFunction()
    if (token.type === 'leftParen') {
      this.consume()
      const value = this.parseExpression()
      this.expect('rightParen')
      return value
    }
    this.fail('syntax')
  }

  private parseFunction() {
    const name = this.consume().value.toUpperCase()
    this.expect('leftParen')
    const args: Array<number | unknown[]> = []
    if (this.current().type !== 'rightParen') {
      while (true) {
        args.push(this.parseFunctionArgument())
        if (this.current().type !== 'comma') break
        this.consume()
      }
    }
    this.expect('rightParen')

    const values = args.flatMap((value) => (Array.isArray(value) ? value : [value]))
    const numbers = values
      .filter((value) => !reportFormulaValueIsEmpty(value))
      .map((value) => Number(value))
      .filter(Number.isFinite)
    if (name === 'COUNT') return numbers.length
    if (name === 'SUM') return numbers.reduce((sum, value) => sum + value, 0)
    if (name === 'AVG') {
      if (!numbers.length) this.fail('division_by_zero')
      return numbers.reduce((sum, value) => sum + value, 0) / numbers.length
    }
    if (name === 'MIN') return numbers.length ? Math.min(...numbers) : 0
    if (name === 'MAX') return numbers.length ? Math.max(...numbers) : 0
    if (name === 'ABS') {
      const value = args[0]
      if (args.length !== 1 || typeof value !== 'number') this.fail('syntax')
      return Math.abs(value)
    }
    if (name === 'ROUND') {
      if (args.length < 1 || args.length > 2 || args.some((value) => typeof value !== 'number'))
        this.fail('syntax')
      const [value = 0, rawDigits = 0] = args as number[]
      const digits = Math.trunc(rawDigits)
      if (digits < -10 || digits > 10) this.fail('syntax')
      const factor = 10 ** digits
      return Math.round((value + Number.EPSILON) * factor) / factor
    }
    this.fail('syntax')
  }

  private parseFunctionArgument(): number | unknown[] {
    if (
      this.current().type === 'cell' &&
      this.tokens[this.index + 1]?.type === 'colon' &&
      this.tokens[this.index + 2]?.type === 'cell'
    ) {
      const start = reportFormulaReference(this.consume().value)
      this.consume()
      const end = reportFormulaReference(this.consume().value)
      return this.resolver.range(start, end)
    }
    return this.parseExpression()
  }

  private toNumber(value: unknown) {
    if (typeof value === 'boolean') return value ? 1 : 0
    if (reportFormulaValueIsEmpty(value)) return 0
    const number = Number(value)
    if (!Number.isFinite(number)) this.fail('syntax')
    return number
  }

  private current() {
    return this.tokens[this.index] || { type: 'eof' as const, value: '' }
  }

  private consume() {
    const token = this.current()
    this.index += 1
    return token
  }

  private expect(type: ReportFormulaTokenType) {
    if (this.current().type !== type) this.fail('syntax')
    return this.consume()
  }

  private isOperator(value: string) {
    const token = this.current()
    return token.type === 'operator' && token.value === value
  }

  private fail(code: ReportFormulaError): never {
    throw new ReportFormulaFailure(code)
  }
}

const reportFormulaTokens = (formula: string): ReportFormulaToken[] => {
  const source = formula.trim().replace(/^=/, '')
  if (!source) throw new ReportFormulaFailure('syntax')
  const tokens: ReportFormulaToken[] = []
  let index = 0
  while (index < source.length) {
    const remaining = source.slice(index)
    const whitespace = remaining.match(/^\s+/)?.[0]
    if (whitespace) {
      index += whitespace.length
      continue
    }
    const number = remaining.match(/^(?:\d+(?:\.\d*)?|\.\d+)/)?.[0]
    if (number) {
      tokens.push({ type: 'number', value: number })
      index += number.length
      continue
    }
    const cell = remaining.match(/^\$?[A-Za-z]+\$?[1-9]\d*/)?.[0]
    if (cell) {
      tokens.push({ type: 'cell', value: cell })
      index += cell.length
      continue
    }
    const identifier = remaining.match(/^[A-Za-z_][A-Za-z0-9_]*/)?.[0]
    if (identifier) {
      tokens.push({ type: 'identifier', value: identifier })
      index += identifier.length
      continue
    }
    const char = source[index]!
    const tokenType: Partial<Record<string, ReportFormulaTokenType>> = {
      '+': 'operator',
      '-': 'operator',
      '*': 'operator',
      '/': 'operator',
      '%': 'operator',
      '(': 'leftParen',
      ')': 'rightParen',
      ',': 'comma',
      ':': 'colon',
    }
    const type = tokenType[char]
    if (!type) throw new ReportFormulaFailure('syntax')
    tokens.push({ type, value: char })
    index += 1
  }
  tokens.push({ type: 'eof', value: '' })
  return tokens
}

const reportFormulaReference = (value: string): ReportFormulaCellReference => {
  const match = value.replace(/\$/g, '').match(/^([A-Za-z]+)([1-9]\d*)$/)
  if (!match) throw new ReportFormulaFailure('syntax')
  let col = 0
  for (const char of match[1]!.toUpperCase()) {
    col = col * 26 + char.charCodeAt(0) - 64
  }
  return { row: Number(match[2]), col }
}

const reportFormulaValueIsEmpty = (value: unknown) =>
  value === null || value === undefined || (typeof value === 'string' && value.trim() === '')

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

export const reportFillSheetCells = (
  sheet: ReportSheetConfig,
  sourceRow: number,
  sourceCol: number,
  targetRow: number,
  targetCol: number,
) => {
  const source = reportSheetCellAt(sheet, sourceRow, sourceCol)
  const sourceSpan = reportSheetCellSpan(source, { maxRow: sheet.rows, maxCol: sheet.cols })
  if (sourceSpan.rowspan > 1 || sourceSpan.colspan > 1) return cloneReportSheet(sheet)

  const vertical = Math.abs(targetRow - sourceRow) >= Math.abs(targetCol - sourceCol)
  const endRow = vertical ? targetRow : sourceRow
  const endCol = vertical ? sourceCol : targetCol
  const bounds: ReportSheetBounds = {
    minRow: Math.min(sourceRow, endRow),
    maxRow: Math.max(sourceRow, endRow),
    minCol: Math.min(sourceCol, endCol),
    maxCol: Math.max(sourceCol, endCol),
  }

  const intersectsMergedCell = sheet.cells.some((cell) => {
    const span = reportSheetCellSpan(cell, { maxRow: sheet.rows, maxCol: sheet.cols })
    if (span.rowspan === 1 && span.colspan === 1) return false
    const cellMaxRow = cell.row + span.rowspan - 1
    const cellMaxCol = cell.col + span.colspan - 1
    return (
      cell.row <= bounds.maxRow &&
      cellMaxRow >= bounds.minRow &&
      cell.col <= bounds.maxCol &&
      cellMaxCol >= bounds.minCol
    )
  })
  if (intersectsMergedCell) return cloneReportSheet(sheet)

  const clipboardCell = reportSheetClipboardMatrix(sheet, {
    minRow: sourceRow,
    maxRow: sourceRow,
    minCol: sourceCol,
    maxCol: sourceCol,
  })[0]?.[0] || { value: '' }
  const matrix = Array.from({ length: bounds.maxRow - bounds.minRow + 1 }, (_, rowIndex) =>
    Array.from({ length: bounds.maxCol - bounds.minCol + 1 }, (_, colIndex) => {
      const rowOffset = bounds.minRow + rowIndex - sourceRow
      const colOffset = bounds.minCol + colIndex - sourceCol
      const binding = clipboardCell.binding
        ? {
            ...clipboardCell.binding,
            ...(clipboardCell.binding.type === 'formula' && clipboardCell.binding.formula
              ? {
                  formula: reportShiftFormulaReferences(
                    clipboardCell.binding.formula,
                    rowOffset,
                    colOffset,
                  ),
                }
              : {}),
          }
        : undefined
      const value =
        binding?.type === 'formula' && clipboardCell.value.trim().startsWith('=')
          ? reportShiftFormulaReferences(clipboardCell.value, rowOffset, colOffset)
          : clipboardCell.value
      return {
        ...clipboardCell,
        value,
        ...(binding ? { binding } : {}),
        ...(clipboardCell.style ? { style: { ...clipboardCell.style } } : {}),
      }
    }),
  )
  return reportPasteSheetCells(sheet, bounds.minRow, bounds.minCol, matrix)
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
  next.group_summary_rows = shiftMarkersForInsert(next.group_summary_rows, insertAt)
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
  next.group_summary_rows = shiftMarkersForDelete(next.group_summary_rows, row)
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
  group_summary_rows: [...(sheet.group_summary_rows || [])],
  column_widths: { ...(sheet.column_widths || {}) },
  row_heights: { ...(sheet.row_heights || {}) },
})

const hasClipboardCellConfig = (cell: ReportSheetClipboardCell) =>
  Boolean(
    cell.value ||
      cell.binding?.field ||
      cell.binding?.formula ||
      (cell.binding?.type && cell.binding.type !== 'static') ||
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
  ;(sheet.group_summary_rows || []).forEach((row) => rows.add(row))
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
    background: style.background || undefined,
    color: style.color || '#172033',
  }
}

export const collectReportUsedFields = (datasets: ReportDataset[], sheet: ReportSheetConfig) => {
  const used = new Map<string, ReportField>()
  sheet.cells.forEach((cell) => {
    const binding = cell.binding
    if (!binding?.field || !binding.dataset_id) return
    const dataset = datasets.find((item) => item.id === binding.dataset_id)
    if (!dataset || dataset.type !== 'table') return
    const field = dataset.fields.find((item) => item.code === binding.field)
    if (field) used.set(`${dataset.id}:${field.code}`, field)
  })
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

  return {
    columns: boundFields.map((item) => item.field),
    datasets,
    rows: boundFields.length
      ? [1, 2, 3].map((id) => {
          const row: Record<string, unknown> = { id }
          boundFields.forEach(({ dataset, field }, index) => {
            const value = reportSampleCellValue(field, id, index)
            row[reportRuntimeColumnAlias(dataset.id, field.code)] = value
            row[reportRuntimeColumnAlias(dataset.source_code, field.code)] = value
            row[field.code] = value
          })
          return row
        })
      : [],
    total: boundFields.length ? 3 : 0,
  }
}

export const reportCellId = makeReportCellId
