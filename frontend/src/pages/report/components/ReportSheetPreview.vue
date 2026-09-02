<template>
  <div class="report-sheet-preview">
    <div v-if="loading" class="report-sheet-preview__state">
      <q-spinner color="primary" size="32px" />
      <span>{{ t('ui.loadingReportData') }}</span>
    </div>

    <div v-else-if="!renderCells.length" class="report-sheet-preview__state">
      <q-icon name="dataset_off" size="32px" />
      <span>{{ t('ui.noDataOrCellConfigurationForWhichPreviewIsAvailable') }}</span>
    </div>

    <div v-else class="report-sheet-preview__scroll">
      <div class="report-sheet-preview__grid" :style="gridStyle">
        <div
          v-for="cell in renderCells"
          :key="cell.key"
          class="report-sheet-preview__cell"
          :class="{
            'is-bound': cell.bound,
            'is-detail': cell.detail,
            'is-summary': cell.summary,
            'is-group-summary': cell.groupSummary,
            'is-empty': !cell.value,
          }"
          :style="cell.style"
        >
          {{ cell.value }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed } from 'vue'
import type {
  ReportDataset,
  ReportKind,
  ReportPreviewRes,
  ReportSheetCell,
  ReportSheetConfig,
} from 'src/api/services/report'
import {
  reportEvaluateFormula,
  reportRuntimeAggregateValue,
  reportRuntimeCellValue,
  reportRuntimeRowGroups,
  reportSheetCellAt,
  reportSheetCellSpan,
  reportSheetIsCoveredCell,
  reportSheetUsedBounds,
  type ReportFormulaCellReference,
  type ReportFormulaError,
  type ReportFormulaResult,
} from 'src/modules/report/sheet'

const { t } = useI18n({ useScope: 'global' })

const props = withDefaults(
  defineProps<{
    sheet: ReportSheetConfig
    datasets: ReportDataset[]
    previewData: ReportPreviewRes
    loading?: boolean
    reportKind?: ReportKind
  }>(),
  {
    loading: false,
    reportKind: 'detail',
  },
)

type RenderCell = {
  key: string
  value: string
  bound: boolean
  detail: boolean
  summary: boolean
  groupSummary: boolean
  style: Record<string, string | number | undefined>
}

type RenderPlanItem = {
  sourceRow: number
  renderRow: number
  dataRow: Record<string, unknown> | undefined
  dataRows: Record<string, unknown>[]
  suffix: string
}

const usedBounds = computed(() => reportSheetUsedBounds(props.sheet))

const sourceColCount = computed(() =>
  Math.max(1, usedBounds.value.maxCol - usedBounds.value.minCol + 1),
)

const detailRows = computed(() => {
  const configured = props.sheet.detail_rows?.filter(
    (row) => row >= 1 && row <= props.sheet.rows && rowHasTemplateContent(row),
  )
  if (configured?.length) return [...new Set(configured)].sort((a, b) => a - b)

  const rows = new Set<number>()
  props.sheet.cells.forEach((cell) => {
    if (
      (cell.binding?.field || cell.binding?.formula) &&
      cell.binding.type !== 'static' &&
      !props.sheet.summary_rows?.includes(cell.row)
    ) {
      rows.add(cell.row)
    }
  })
  return [...rows].sort((a, b) => a - b)
})

const groupSummaryRows = computed(
  () =>
    new Set(
      (props.sheet.group_summary_rows || []).filter(
        (row) => row >= 1 && row <= props.sheet.rows && rowHasTemplateContent(row),
      ),
    ),
)

const configuredRows = computed(() => {
  const rows = new Set<number>()
  ;(props.sheet.detail_rows || []).forEach((row) => {
    if (rowHasTemplateContent(row)) rows.add(row)
  })
  ;(props.sheet.summary_rows || []).forEach((row) => {
    if (rowHasTemplateContent(row)) rows.add(row)
  })
  props.sheet.cells.forEach((cell) => {
    if (
      cell.value ||
      cell.binding?.field ||
      cell.binding?.formula ||
      (cell.colspan && cell.colspan > 1) ||
      (cell.rowspan && cell.rowspan > 1)
    ) {
      rows.add(cell.row)
    }
  })
  return [...rows].sort((a, b) => a - b)
})

const detailRowGroups = computed(() => {
  const groups: number[][] = []
  detailRows.value.forEach((row) => {
    const lastGroup = groups[groups.length - 1]
    const lastRow = lastGroup?.[lastGroup.length - 1]
    if (lastGroup && lastRow !== undefined && row === lastRow + 1) {
      lastGroup.push(row)
      return
    }
    groups.push([row])
  })
  return groups
})

const runtimeRowGroups = computed(() =>
  reportRuntimeRowGroups(props.previewData.rows || [], props.sheet, props.datasets),
)

const renderPlan = computed(() => {
  const plan: RenderPlanItem[] = []
  const groupByStart = new Map(detailRowGroups.value.map((group) => [group[0], group]))
  const rowsInDetailGroup = new Set(detailRowGroups.value.flat())
  let renderRow = 1

  const sourceRows = configuredRows.value.length ? configuredRows.value : [usedBounds.value.minRow]
  for (let index = 0; index < sourceRows.length; index += 1) {
    const sourceRow = sourceRows[index]!
    const detailGroup = groupByStart.get(sourceRow)
    if (detailGroup && props.reportKind === 'detail') {
      const scopedSummaryRows: number[] = []
      let nextSourceIndex = index + detailGroup.length
      while (groupSummaryRows.value.has(sourceRows[nextSourceIndex] || 0)) {
        scopedSummaryRows.push(sourceRows[nextSourceIndex]!)
        nextSourceIndex += 1
      }
      const repeatedGroups = runtimeRowGroups.value.length
        ? runtimeRowGroups.value
        : [{ key: 'empty', row: undefined, rows: [] }]
      repeatedGroups.forEach((dataGroup, dataIndex) => {
        detailGroup.forEach((groupRow) => {
          plan.push({
            sourceRow: groupRow,
            renderRow,
            dataRow: dataGroup.row,
            dataRows: dataGroup.rows,
            suffix: `data-${dataIndex}-${dataGroup.key}-${groupRow}`,
          })
          renderRow += 1
        })
        scopedSummaryRows.forEach((summaryRow) => {
          plan.push({
            sourceRow: summaryRow,
            renderRow,
            dataRow: dataGroup.row,
            dataRows: dataGroup.rows,
            suffix: `group-summary-${dataIndex}-${dataGroup.key}-${summaryRow}`,
          })
          renderRow += 1
        })
      })
      index += Math.max(detailGroup.length + scopedSummaryRows.length - 1, 0)
      continue
    }
    if (props.reportKind === 'detail' && rowsInDetailGroup.has(sourceRow)) continue
    plan.push({
      sourceRow,
      renderRow,
      dataRow: undefined,
      dataRows: props.previewData.rows || [],
      suffix: `row-${sourceRow}`,
    })
    renderRow += 1
  }

  return trimTrailingBlankRenderRows(plan)
})

const gridStyle = computed(() => ({
  gridTemplateColumns: Array.from(
    { length: sourceColCount.value },
    (_, index) =>
      `${props.sheet.column_widths?.[String(usedBounds.value.minCol + index)] || 118}px`,
  ).join(' '),
  gridTemplateRows: Array.from({ length: Math.max(renderRowCount.value, 1) }, (_, index) => {
    const sourceRow = renderPlan.value.find((item) => item.renderRow === index + 1)?.sourceRow
    return `${props.sheet.row_heights?.[String(sourceRow || 0)] || 42}px`
  }).join(' '),
}))

const renderRowCount = computed(() =>
  Math.max(1, ...renderPlan.value.map((item) => item.renderRow)),
)

const renderCells = computed<RenderCell[]>(() => {
  const cells: RenderCell[] = []
  const maxRow = props.sheet.rows || usedBounds.value.maxRow
  const maxCol = usedBounds.value.maxCol

  renderPlan.value.forEach((item) => {
    for (let col = usedBounds.value.minCol; col <= usedBounds.value.maxCol; col += 1) {
      if (reportSheetIsCoveredCell(props.sheet, item.sourceRow, col)) continue
      const sourceCell = cellAt(item.sourceRow, col)
      const { rowspan, colspan } = reportSheetCellSpan(sourceCell, { maxRow, maxCol })
      cells.push({
        key: `${sourceCell.row}:${sourceCell.col}:${item.suffix}`,
        value: displayCellValue(sourceCell, item.dataRow, item.dataRows),
        bound: Boolean(sourceCell.binding?.field || sourceCell.binding?.formula),
        detail: detailRows.value.includes(item.sourceRow),
        summary: props.sheet.summary_rows?.includes(item.sourceRow) || false,
        groupSummary: groupSummaryRows.value.has(item.sourceRow),
        style: {
          ...cellStyle(sourceCell),
          gridColumn: `${col - usedBounds.value.minCol + 1} / span ${colspan}`,
          gridRow: `${item.renderRow} / span ${rowspan}`,
        },
      })
    }
  })

  return cells
})

function trimTrailingBlankRenderRows(plan: RenderPlanItem[]) {
  let end = plan.length
  while (end > 1) {
    const item = plan[end - 1]
    if (!item || rowHasContent(item.sourceRow, item.dataRow, item.dataRows)) break
    end -= 1
  }
  return plan.slice(0, end)
}

function rowHasContent(
  sourceRow: number,
  dataRow: Record<string, unknown> | undefined,
  dataRows: Record<string, unknown>[],
) {
  for (let col = usedBounds.value.minCol; col <= usedBounds.value.maxCol; col += 1) {
    if (reportSheetIsCoveredCell(props.sheet, sourceRow, col)) continue
    if (String(displayCellValue(cellAt(sourceRow, col), dataRow, dataRows) || '').trim() !== '')
      return true
  }
  return false
}

function rowHasTemplateContent(sourceRow: number) {
  for (let col = usedBounds.value.minCol; col <= usedBounds.value.maxCol; col += 1) {
    if (reportSheetIsCoveredCell(props.sheet, sourceRow, col)) continue
    const cell = cellAt(sourceRow, col)
    if (
      cell.value ||
      cell.binding?.field ||
      (cell.colspan && cell.colspan > 1) ||
      (cell.rowspan && cell.rowspan > 1)
    ) {
      return true
    }
  }
  return false
}

function cellAt(row: number, col: number): ReportSheetCell {
  return reportSheetCellAt(props.sheet, row, col)
}

function displayCellValue(
  cell: ReportSheetCell,
  dataRow: Record<string, unknown> | undefined,
  dataRows: Record<string, unknown>[],
  resolving = new Set<string>(),
) {
  if (cell.binding?.type === 'formula') {
    const result = evaluateFormulaCell(cell, dataRow, dataRows, resolving)
    return result.error ? formulaErrorText(result.error) : result.value
  }
  if (cell.binding?.field && cell.binding.type !== 'static') {
    if (['sum', 'count', 'avg', 'max', 'min'].includes(cell.binding.type)) {
      return reportRuntimeAggregateValue(dataRows, cell, props.datasets)
    }
    if (
      !dataRow &&
      (props.reportKind !== 'detail' || props.sheet.summary_rows?.includes(cell.row))
    ) {
      return aggregateCellValue(cell)
    }
    return reportRuntimeCellValue(dataRow, cell, props.datasets)
  }
  return cell.value || ''
}

function evaluateFormulaCell(
  cell: ReportSheetCell,
  dataRow: Record<string, unknown> | undefined,
  dataRows: Record<string, unknown>[],
  resolving: Set<string>,
): ReportFormulaResult {
  if (resolving.has(cell.id)) return { value: '', error: 'circular_reference' }
  const formula = cell.binding?.formula?.trim() || cell.value?.trim() || ''
  const nextResolving = new Set(resolving)
  nextResolving.add(cell.id)
  return reportEvaluateFormula(formula, {
    cell: (reference) => formulaReferenceValue(reference, dataRow, dataRows, nextResolving),
    range: (start, end) => formulaRangeValues(start, end, dataRow, dataRows, nextResolving),
  })
}

function formulaReferenceValue(
  reference: ReportFormulaCellReference,
  dataRow: Record<string, unknown> | undefined,
  dataRows: Record<string, unknown>[],
  resolving: Set<string>,
) {
  assertFormulaReference(reference)
  const cell = cellAt(reference.row, reference.col)
  if (cell.binding?.type === 'formula') {
    const result = evaluateFormulaCell(cell, dataRow, dataRows, resolving)
    if (result.error) throw new Error(result.error)
    return result.value
  }
  if (['sum', 'count', 'avg', 'max', 'min'].includes(cell.binding?.type || '')) {
    return reportRuntimeAggregateValue(dataRows, cell, props.datasets)
  }
  if (cell.binding?.field) {
    return reportRuntimeCellValue(dataRow || dataRows[0], cell, props.datasets)
  }
  return cell.value || ''
}

function formulaRangeValues(
  start: ReportFormulaCellReference,
  end: ReportFormulaCellReference,
  dataRow: Record<string, unknown> | undefined,
  dataRows: Record<string, unknown>[],
  resolving: Set<string>,
) {
  assertFormulaReference(start)
  assertFormulaReference(end)
  const values: unknown[] = []
  const minRow = Math.min(start.row, end.row)
  const maxRow = Math.max(start.row, end.row)
  const minCol = Math.min(start.col, end.col)
  const maxCol = Math.max(start.col, end.col)
  for (let row = minRow; row <= maxRow; row += 1) {
    for (let col = minCol; col <= maxCol; col += 1) {
      const cell = cellAt(row, col)
      if (
        cell.binding?.field &&
        !['sum', 'count', 'avg', 'max', 'min'].includes(cell.binding.type) &&
        detailRows.value.includes(row) &&
        dataRows.length
      ) {
        dataRows.forEach((scopeRow) => {
          values.push(reportRuntimeCellValue(scopeRow, cell, props.datasets))
        })
        continue
      }
      values.push(formulaReferenceValue({ row, col }, dataRow, dataRows, resolving))
    }
  }
  return values
}

function assertFormulaReference(reference: ReportFormulaCellReference) {
  if (
    reference.row < 1 ||
    reference.row > props.sheet.rows ||
    reference.col < 1 ||
    reference.col > props.sheet.cols
  ) {
    throw new Error('syntax')
  }
}

function formulaErrorText(error: ReportFormulaError) {
  if (error === 'division_by_zero') return t('ui.formulaDivisionByZero')
  if (error === 'circular_reference') return t('ui.formulaCircularReference')
  return t('ui.formulaSyntaxError')
}

function aggregateCellValue(cell: ReportSheetCell) {
  const dataRows = props.previewData.rows || []
  if (!dataRows.length) return cell.value || ''
  if (['sum', 'count', 'avg', 'max', 'min'].includes(cell.binding?.type || '')) {
    return reportRuntimeAggregateValue(dataRows, cell, props.datasets)
  }
  const values = dataRows
    .map((row) => reportRuntimeCellValue(row, cell, props.datasets))
    .filter((value) => value !== '')
  const uniqueValues = [...new Set(values)]
  if (!uniqueValues.length) return ''
  if (uniqueValues.length === 1) return uniqueValues[0] || ''
  return t('ui.andTotalValues', { value1: uniqueValues[0], value2: uniqueValues.length })
}

function cellStyle(cell: ReportSheetCell) {
  const style = cell.style || {}
  const align = style.align || 'left'
  return {
    textAlign: align,
    justifyContent: align === 'right' ? 'flex-end' : align === 'center' ? 'center' : 'flex-start',
    fontWeight: style.bold ? 800 : 500,
    fontStyle: style.italic ? 'italic' : 'normal',
    background: style.background || undefined,
    color: style.color || '#172033',
  }
}
</script>

<style scoped lang="scss">
.report-sheet-preview {
  min-height: 260px;
  border: 1px solid #dfe5f2;
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
}

.report-sheet-preview__state {
  min-height: 260px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  color: #71809a;
}

.report-sheet-preview__scroll {
  max-height: min(62vh, 720px);
  overflow: auto;
}

.report-sheet-preview__grid {
  min-width: 100%;
  display: grid;
}

.report-sheet-preview__cell {
  min-height: 0;
  padding: 8px 10px;
  border-right: 1px solid #dfe5f2;
  border-bottom: 1px solid #dfe5f2;
  color: #172033;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
  display: flex;
  align-items: center;
}

.report-sheet-preview__cell.is-bound {
  background: #fbfaff;
}

.report-sheet-preview__cell.is-detail {
  background: #fff;
}

.report-sheet-preview__cell.is-summary {
  background: #fff8e8;
  font-weight: 800;
}

.report-sheet-preview__cell.is-group-summary {
  background: #eefbf6;
}

.report-sheet-preview__cell.is-empty {
  color: transparent;
}
</style>
