import { describe, expect, it } from 'vitest'
import type { TableField } from '@/api/services/sys-table'
import { fieldAccess, isDetailFieldVisible } from './field-layout'

describe('detail field visibility', () => {
  it('inherits form visibility when unset and respects explicit overrides', () => {
    expect(isDetailFieldVisible({ is_insert_show: true } as TableField)).toBe(true)
    expect(isDetailFieldVisible({ is_list_show: true, is_insert_show: false, is_update_show: false } as TableField)).toBe(false)
    expect(isDetailFieldVisible({ field_code: 'gmt_create', detail_visible: false } as TableField)).toBe(false)
    expect(isDetailFieldVisible({ field_code: 'gmt_create', detail_visible: true } as TableField)).toBe(true)
  })
  it('readonly only applies to visible fields in its own scene', () => {
    const flags = { is_insert_show: false, is_update_show: true, detail_visible: true, is_update_readonly: true }
    expect(fieldAccess(flags, 'insert', true)).toBe('hidden')
    expect(fieldAccess(flags, 'update')).toBe('readonly')
    expect(fieldAccess(flags, 'detail')).toBe('readonly')
    expect(fieldAccess({ ...flags, is_insert_show: true }, 'insert')).toBe('editable')
    expect(fieldAccess({ ...flags, detail_visible: false }, 'detail')).toBe('hidden')
  })
})
