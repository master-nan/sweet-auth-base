import cloneDeep from 'lodash/cloneDeep'
import isEqual from 'lodash/isEqual'
import Decimal from 'decimal.js'
import type { DocumentChanges } from '@/api/services/generalization'
import type { DocumentTabSettings, TableRelation } from '@/api/services/sys-table'
import { SysTableRelationType } from '@/types/enum'

const Exact = Decimal.clone({ precision: 100 })
type Data = Record<string, any>
export type DocumentLine = { key: string; data: Data }
export type LineDraft = { lines: DocumentLine[]; originals: Record<number, Data> }
export const documentRelations = (relations: TableRelation[] = []) =>
  relations
    .filter((r) => r.relation_type === SysTableRelationType.ONE_TO_MANY && r.document_tab?.trim())
    .sort((a, b) => a.id - b.id)

export function draftLines(records: Data[]): LineDraft {
  return {
    lines: records.map((data) => ({ key: String(data.id), data: cloneDeep(data) })),
    originals: Object.fromEntries(records.map((row) => [row.id, row])),
  }
}

export function lineChanges(
  relationId: number,
  draft: LineDraft,
  insertFields: string[],
  updateFields: string[],
): DocumentChanges {
  const result: DocumentChanges = { relation_id: relationId, added: [], updated: [], deleted: [] }
  const retained = new Set<number>()
  for (const { data } of draft.lines) {
    if (!data.id) {
      result.added.push(
        Object.fromEntries(
          insertFields.filter((key) => data[key] !== undefined).map((key) => [key, data[key]]),
        ),
      )
      continue
    }
    const id = Number(data.id)
    retained.add(id)
    const patch = Object.fromEntries(
      updateFields
        .filter((key) => !isEqual(data[key], draft.originals[id]?.[key]))
        .map((key) => [key, data[key]]),
    )
    if (Object.keys(patch).length) result.updated.push({ id, data: patch })
  }
  result.deleted = Object.keys(draft.originals)
    .map(Number)
    .filter((id) => !retained.has(id))
  return result
}

function numeric(value: unknown) {
  if (typeof value !== 'string' && typeof value !== 'number') return new Exact(0)
  const text = String(value).trim()
  if (!/^[+-]?(?:\d+\.?\d*|\.\d+)$/.test(text)) return new Exact(0)
  return new Exact(text)
}

export function documentTotals(relation: DocumentTabSettings, lines: DocumentLine[]) {
  let quantity = new Exact(0),
    amount = new Exact(0)
  for (const { data } of lines) {
    const qty = numeric(data[relation.quantity_field || ''])
    quantity = quantity.plus(qty)
    amount = amount.plus(
      relation.amount_field
        ? numeric(data[relation.amount_field])
        : qty.times(numeric(data[relation.price_field || ''])),
    )
  }
  return {
    quantity: quantity.toFixed(),
    amount: amount.toFixed(Math.max(2, amount.decimalPlaces())),
  }
}
