import { describe, expect, it } from 'vitest'

import { createBlankReportSheet, hasReportCellConfig, makeReportCellId } from './schema'
import {
  reportDeleteSheetColumn,
  reportDeleteSheetRow,
  reportEvaluateFormula,
  reportFillSheetCells,
  reportInsertSheetColumn,
  reportInsertSheetRow,
  reportPasteSheetCells,
  reportRuntimeAggregateValue,
  reportRuntimeRowGroups,
  reportShiftFormulaReferences,
  reportSheetClipboardMatrix,
  reportValidateFormula,
} from './sheet'
import type { ReportDataset, ReportSheetCell } from './types'

describe('report sheet structure editing', () => {
  it('keeps an incomplete non-static binding while an empty cell is being configured', () => {
    expect(hasReportCellConfig({ binding: { type: 'formula', formula: '' } })).toBe(true)
    expect(hasReportCellConfig({ binding: { type: 'field' } })).toBe(true)
    expect(hasReportCellConfig({ binding: { type: 'static' } })).toBe(false)
  })

  it('keeps cells, merged spans, row markers and heights aligned when rows change', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', rowspan: 2 },
      { id: makeReportCellId(3, 4), row: 3, col: 4, value: '金额' },
    ]
    sheet.detail_rows = [2, 4]
    sheet.summary_rows = [3]
    sheet.group_summary_rows = [3]
    sheet.row_heights = { '2': 50, '3': 60 }

    const inserted = reportInsertSheetRow(sheet, 2)
    expect(inserted.rows).toBe(11)
    expect(inserted.cells).toEqual([
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', rowspan: 3 },
      { id: makeReportCellId(4, 4), row: 4, col: 4, value: '金额' },
    ])
    expect(inserted.detail_rows).toEqual([2, 5])
    expect(inserted.summary_rows).toEqual([4])
    expect(inserted.group_summary_rows).toEqual([4])
    expect(inserted.row_heights).toEqual({ '2': 50, '4': 60 })

    const deleted = reportDeleteSheetRow(inserted, 3)
    expect(deleted.rows).toBe(10)
    expect(deleted.cells).toEqual(sheet.cells)
    expect(deleted.detail_rows).toEqual([2, 4])
    expect(deleted.summary_rows).toEqual([3])
    expect(deleted.group_summary_rows).toEqual([3])
    expect(deleted.row_heights).toEqual({ '2': 50, '3': 60 })
  })

  it('keeps cells, merged spans and widths aligned when columns change', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', colspan: 2 },
      { id: makeReportCellId(4, 3), row: 4, col: 3, value: '金额' },
    ]
    sheet.column_widths = { '2': 150, '3': 180 }

    const inserted = reportInsertSheetColumn(sheet, 2)
    expect(inserted.cols).toBe(9)
    expect(inserted.cells).toEqual([
      { id: makeReportCellId(2, 2), row: 2, col: 2, value: '标题', colspan: 3 },
      { id: makeReportCellId(4, 4), row: 4, col: 4, value: '金额' },
    ])
    expect(inserted.column_widths).toEqual({ '2': 150, '4': 180 })

    const deleted = reportDeleteSheetColumn(inserted, 3)
    expect(deleted.cols).toBe(8)
    expect(deleted.cells).toEqual(sheet.cells)
    expect(deleted.column_widths).toEqual({ '2': 150, '3': 180 })
  })

  it('copies and pastes values, bindings and styles as a matrix', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      {
        id: makeReportCellId(2, 2),
        row: 2,
        col: 2,
        value: '[订单.金额]',
        binding: { type: 'sum', dataset_id: 'orders', field: 'amount' },
        style: { bold: true, color: '#6d5dfc' },
      },
      { id: makeReportCellId(2, 3), row: 2, col: 3, value: '元' },
    ]

    const matrix = reportSheetClipboardMatrix(sheet, {
      minRow: 2,
      maxRow: 2,
      minCol: 2,
      maxCol: 3,
    })
    const pasted = reportPasteSheetCells(sheet, 5, 4, matrix)

    expect(pasted.cells).toContainEqual({
      id: makeReportCellId(5, 4),
      row: 5,
      col: 4,
      value: '[订单.金额]',
      binding: { type: 'sum', dataset_id: 'orders', field: 'amount' },
      style: { bold: true, color: '#6d5dfc' },
    })
    expect(pasted.cells).toContainEqual({
      id: makeReportCellId(5, 5),
      row: 5,
      col: 5,
      value: '元',
    })
  })

  it('fills a cell binding and style along the dominant drag axis', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      {
        id: makeReportCellId(2, 2),
        row: 2,
        col: 2,
        value: '订单.S(金额)',
        binding: { type: 'field', dataset_id: 'orders', field: 'amount' },
        style: { bold: true, align: 'right' },
      },
    ]

    const filledDown = reportFillSheetCells(sheet, 2, 2, 5, 3)
    expect(
      filledDown.cells.filter((cell) => cell.col === 2 && cell.row >= 2 && cell.row <= 5),
    ).toHaveLength(4)
    expect(filledDown.cells).toContainEqual({
      id: makeReportCellId(5, 2),
      row: 5,
      col: 2,
      value: '订单.S(金额)',
      binding: { type: 'field', dataset_id: 'orders', field: 'amount' },
      style: { bold: true, align: 'right' },
    })

    const filledRight = reportFillSheetCells(sheet, 2, 2, 3, 5)
    expect(
      filledRight.cells.filter((cell) => cell.row === 2 && cell.col >= 2 && cell.col <= 5),
    ).toHaveLength(4)
  })

  it('adjusts relative formula references while preserving absolute references when filling', () => {
    const sheet = createBlankReportSheet(10, 8)
    sheet.cells = [
      {
        id: makeReportCellId(2, 2),
        row: 2,
        col: 2,
        value: '=A2*$A$1',
        binding: { type: 'formula', formula: '=A2*$A$1' },
      },
    ]

    const filled = reportFillSheetCells(sheet, 2, 2, 5, 2)
    expect(filled.cells).toContainEqual({
      id: makeReportCellId(5, 2),
      row: 5,
      col: 2,
      value: '=A5*$A$1',
      binding: { type: 'formula', formula: '=A5*$A$1' },
    })
    expect(reportShiftFormulaReferences('=SUM(A2:B2)+$C2+D$1', 2, 1)).toBe('=SUM(B4:C4)+$C4+E$1')
  })

  it('groups detail data by configured group cells and calculates aggregates per group', () => {
    const datasets: ReportDataset[] = [
      {
        id: 'orders',
        name: '订单',
        type: 'table',
        source_code: 'sales_order',
        primary: true,
        fields: [
          { name: '部门', code: 'department', type: 'varchar', role: 'dimension' },
          { name: '金额', code: 'amount', type: 'decimal', role: 'metric' },
        ],
      },
    ]
    const sheet = createBlankReportSheet(10, 8)
    const groupCell: ReportSheetCell = {
      id: makeReportCellId(2, 1),
      row: 2,
      col: 1,
      value: '订单.G(部门)',
      binding: { type: 'group', dataset_id: 'orders', field: 'department' },
    }
    const aggregateCell: ReportSheetCell = {
      id: makeReportCellId(2, 2),
      row: 2,
      col: 2,
      value: 'SUM(订单.amount)',
      binding: { type: 'sum', dataset_id: 'orders', field: 'amount' },
    }
    sheet.detail_rows = [2]
    sheet.cells = [groupCell, aggregateCell]
    const rows = [
      { orders__department: '研发', orders__amount: 10 },
      { orders__department: '研发', orders__amount: 20 },
      { orders__department: '销售', orders__amount: 5 },
    ]

    const groups = reportRuntimeRowGroups(rows, sheet, datasets)

    expect(groups).toHaveLength(2)
    expect(groups[0]?.rows).toHaveLength(2)
    expect(groups[1]?.rows).toHaveLength(1)
    expect(reportRuntimeAggregateValue(groups[0]!.rows, aggregateCell, datasets)).toBe('30')

    aggregateCell.binding = { ...aggregateCell.binding!, type: 'avg' }
    expect(reportRuntimeAggregateValue(groups[0]!.rows, aggregateCell, datasets)).toBe('15')
    aggregateCell.binding = { ...aggregateCell.binding!, type: 'max' }
    expect(reportRuntimeAggregateValue(groups[0]!.rows, aggregateCell, datasets)).toBe('20')
    aggregateCell.binding = { ...aggregateCell.binding!, type: 'min' }
    expect(reportRuntimeAggregateValue(groups[0]!.rows, aggregateCell, datasets)).toBe('10')
    aggregateCell.binding = { ...aggregateCell.binding!, type: 'count' }
    expect(reportRuntimeAggregateValue(groups[0]!.rows, aggregateCell, datasets)).toBe('2')
  })
})

describe('report sheet formulas', () => {
  const values: Record<string, unknown> = {
    '1:1': 10,
    '1:2': 5,
    '2:1': 20,
    '2:2': 15,
    '3:1': 30,
    '3:2': '',
  }
  const resolver = {
    cell: ({ row, col }: { row: number; col: number }) => values[`${row}:${col}`],
    range: (start: { row: number; col: number }, end: { row: number; col: number }) => {
      const result: unknown[] = []
      for (let row = Math.min(start.row, end.row); row <= Math.max(start.row, end.row); row += 1) {
        for (
          let col = Math.min(start.col, end.col);
          col <= Math.max(start.col, end.col);
          col += 1
        ) {
          result.push(values[`${row}:${col}`])
        }
      }
      return result
    },
  }

  it('calculates arithmetic, absolute references and parentheses', () => {
    expect(reportEvaluateFormula('=$A$1 + B1 * 2', resolver)).toEqual({ value: '20' })
    expect(reportEvaluateFormula('=(A2-B1)/3', resolver)).toEqual({ value: '5' })
  })

  it('calculates range and numeric functions', () => {
    expect(reportEvaluateFormula('=SUM(A1:A3)', resolver)).toEqual({ value: '60' })
    expect(reportEvaluateFormula('=AVG(A1:B2)', resolver)).toEqual({ value: '12.5' })
    expect(reportEvaluateFormula('=MIN(A1:B2)+MAX(A1:B2)', resolver)).toEqual({ value: '25' })
    expect(reportEvaluateFormula('=COUNT(A1:B3)', resolver)).toEqual({ value: '5' })
    expect(reportEvaluateFormula('=ROUND(10/3, 2)+ABS(-1)', resolver)).toEqual({ value: '4.33' })
  })

  it('reports invalid syntax and division by zero without evaluating arbitrary code', () => {
    expect(reportValidateFormula('=A1/0')).toBe('division_by_zero')
    expect(reportValidateFormula('=SUM(A1:A2')).toBe('syntax')
    expect(reportValidateFormula('=window.alert(1)')).toBe('syntax')
  })
})
