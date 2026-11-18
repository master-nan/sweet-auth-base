import { describe, expect, it } from 'vitest'
import { computed, reactive } from 'vue'
import type { TableRelation } from '@/api/services/sys-table'
import { SysTableRelationType } from '@/types/enum'
import { documentRelations, documentTotals, draftLines, lineChanges } from './document-lines'

function relation(
  id: number,
  document_tab: string,
  relation_type = SysTableRelationType.ONE_TO_MANY,
): TableRelation {
  return {
    id,
    table_id: 1,
    related_table_id: id,
    reference_key: 'id',
    foreign_key: 'document_id',
    many_table_code: '',
    relation_type,
    document_tab,
  }
}

describe('documentRelations', () => {
  it('keeps all named one-to-many tabs in relation ID order', () => {
    const fees = relation(11, 'Fees')
    const goods = relation(2, 'Goods')
    expect(
      documentRelations([
        fees,
        relation(3, 'Header', SysTableRelationType.ONE_TO_ONE),
        relation(4, '  '),
        goods,
        relation(5, 'Customer', SysTableRelationType.MANY_TO_ONE),
        relation(6, 'Tags', SysTableRelationType.MANY_TO_MANY),
      ]),
    ).toEqual([goods, fees])
  })
})

describe('lineChanges', () => {
  it('keeps adds, edits and deletes scoped to their tab, even for matching row IDs', () => {
    const goods = draftLines([
      { id: 7, qty: 1 },
      { id: 8, qty: 2 },
    ])
    const fees = draftLines([{ id: 8, qty: 5 }])
    goods.lines[0]!.data.qty = 3
    goods.lines.splice(1, 1)
    goods.lines.push({ key: 'new-item', data: { qty: 4 } })
    fees.lines[0]!.data.qty = 6

    expect(lineChanges(2, goods, ['qty'], ['qty'])).toEqual({
      relation_id: 2,
      added: [{ qty: 4 }],
      updated: [{ id: 7, data: { qty: 3 } }],
      deleted: [8],
    })
    expect(lineChanges(11, fees, ['qty'], ['qty'])).toEqual({
      relation_id: 11,
      added: [],
      updated: [{ id: 8, data: { qty: 6 } }],
      deleted: [],
    })
  })

  it('drops reverted edits and discarded new rows without changing loaded records', () => {
    const records = [{ id: 7, details: { unit: 'box' } }]
    const draft = draftLines(records)
    const row = draft.lines[0]!.data
    row.details.unit = 'case'
    expect(records[0]!.details.unit).toBe('box')
    expect(lineChanges(2, draft, [], ['details']).updated).toEqual([
      { id: 7, data: { details: { unit: 'case' } } },
    ])

    row.details = { unit: 'box' }
    draft.lines.push({ key: 'temporary', data: { details: { unit: 'case' } } })
    draft.lines.pop()
    expect(lineChanges(2, draft, ['details'], ['details'])).toEqual({
      relation_id: 2,
      added: [],
      updated: [],
      deleted: [],
    })
  })

  it('uses separate insert and update allowlists and sends only changed writable values', () => {
    const draft = draftLines([
      { id: 7, sku: 'A', qty: 4, note: 'before', active: true, unit: 'box', total: 24 },
      { id: 8, total: 10 },
    ])
    Object.assign(draft.lines[0]!.data, { sku: 'B', qty: 0, note: null, active: false, total: 0 })
    draft.lines[1]!.data.total = 20
    draft.lines.push({
      key: 'new-item',
      data: { sku: 'C', qty: 0, note: 'update only', unit: undefined, total: 99 },
    })

    expect(
      lineChanges(2, draft, ['sku', 'qty', 'unit'], ['qty', 'note', 'active', 'unit']),
    ).toStrictEqual({
      relation_id: 2,
      added: [{ sku: 'C', qty: 0 }],
      updated: [{ id: 7, data: { qty: 0, note: null, active: false } }],
      deleted: [],
    })
  })
})

describe('documentTotals', () => {
  it('按各自的字段映射精确汇总，分别响应明细的增删改', () => {
    const goods = reactive(draftLines([{ id: 7, units: '0.3', price: '0.2' }]))
    const fees = reactive(
      draftLines([{ id: 8, count: '2', rate: '7', charge: '9007199254740993.001' }]),
    )
    const totals = computed(() => documentTotals({ quantity_field: 'units', price_field: 'price' }, goods.lines))
    const charges = computed(() => documentTotals({ quantity_field: 'count', price_field: 'rate', amount_field: 'charge' }, fees.lines))

    expect(totals.value).toEqual({ quantity: '0.3', amount: '0.06' })
    expect(charges.value).toEqual({ quantity: '2', amount: '9007199254740993.001' })
    goods.lines[0]!.data.units = '0.35'
    expect(totals.value).toEqual({ quantity: '0.35', amount: '0.07' })
    goods.lines.push({ key: 'new-item', data: { units: '0.05', price: '0.2' } })
    expect(totals.value).toEqual({ quantity: '0.4', amount: '0.08' })
    fees.lines.splice(0, 1)
    expect(charges.value).toEqual({ quantity: '0', amount: '0.00' })
    expect(totals.value).toEqual({ quantity: '0.4', amount: '0.08' })
    goods.lines.splice(0)
    expect(totals.value).toEqual({ quantity: '0', amount: '0.00' })
  })
})
