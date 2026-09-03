import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import { createBlankReportSheet, makeReportCellId } from '@/modules/report/schema'
import type { ReportDataset } from '@/modules/report/types'
import ReportSheetPreview from './ReportSheetPreview.vue'

describe('ReportSheetPreview', () => {
  it('keeps a static detail-marked row once and preserves its sheet dimensions', () => {
    const sheet = createBlankReportSheet(8, 6)
    sheet.detail_rows = [1]
    sheet.column_widths = { '1': 180, '2': 96 }
    sheet.row_heights = { '1': 54 }
    sheet.cells = [
      { id: makeReportCellId(1, 1), row: 1, col: 1, value: '静态一' },
      { id: makeReportCellId(1, 2), row: 1, col: 2, value: '静态二' },
    ]

    const wrapper = mount(ReportSheetPreview, {
      props: {
        sheet,
        datasets: [],
        reportKind: 'detail',
        previewData: {
          columns: [],
          rows: [{ id: 1 }, { id: 2 }, { id: 3 }],
          total: 3,
        },
      },
    })

    expect(wrapper.findAll('.report-sheet-preview__cell').map((cell) => cell.text())).toEqual([
      '静态一',
      '静态二',
    ])
    expect(wrapper.find('.report-sheet-preview__grid').attributes('style')).toContain('180px 96px')
    expect(wrapper.find('.report-sheet-preview__grid').attributes('style')).toContain('54px')
  })

  it('renders a subtotal for each data group and one global total', () => {
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
    const sheet = createBlankReportSheet(8, 6)
    sheet.detail_rows = [2]
    sheet.summary_rows = [3, 4]
    sheet.group_summary_rows = [3]
    sheet.cells = [
      {
        id: makeReportCellId(2, 1),
        row: 2,
        col: 1,
        value: '订单.G(部门)',
        binding: { type: 'group', dataset_id: 'orders', field: 'department' },
      },
      {
        id: makeReportCellId(2, 2),
        row: 2,
        col: 2,
        value: '订单.S(金额)',
        binding: { type: 'field', dataset_id: 'orders', field: 'amount' },
      },
      { id: makeReportCellId(3, 1), row: 3, col: 1, value: '小计' },
      {
        id: makeReportCellId(3, 2),
        row: 3,
        col: 2,
        value: '=SUM(B2:B2)',
        binding: { type: 'formula', formula: '=SUM(B2:B2)' },
      },
      { id: makeReportCellId(4, 1), row: 4, col: 1, value: '总计' },
      {
        id: makeReportCellId(4, 2),
        row: 4,
        col: 2,
        value: '=SUM(B2:B2)',
        binding: { type: 'formula', formula: '=SUM(B2:B2)' },
      },
    ]

    const wrapper = mount(ReportSheetPreview, {
      props: {
        sheet,
        datasets,
        reportKind: 'detail',
        previewData: {
          columns: datasets[0]!.fields,
          rows: [
            { orders__department: '研发', orders__amount: 10 },
            { orders__department: '研发', orders__amount: 20 },
            { orders__department: '销售', orders__amount: 5 },
          ],
          total: 3,
        },
      },
    })

    expect(wrapper.findAll('.report-sheet-preview__cell').map((cell) => cell.text())).toEqual([
      '研发',
      '10',
      '小计',
      '30',
      '销售',
      '5',
      '小计',
      '5',
      '总计',
      '35',
    ])
    expect(wrapper.findAll('.is-group-summary')).toHaveLength(4)
    expect(wrapper.findAll('.is-summary')).toHaveLength(6)
  })
})
