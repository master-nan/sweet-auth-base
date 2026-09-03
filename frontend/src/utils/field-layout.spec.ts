import { describe, expect, it } from 'vitest'
import type { TableField } from '@/api/services/sys-table'
import { isDetailFieldVisible } from './field-layout'

describe('detail field visibility', () => {
  it('shows ordinary and explicitly visible fields but hides managed defaults', () => {
    expect(isDetailFieldVisible({ field_code: 'name' } as TableField)).toBe(true)
    expect(isDetailFieldVisible({ field_code: 'gmt_create', detail_visible: false } as TableField)).toBe(false)
    expect(isDetailFieldVisible({ field_code: 'gmt_create', detail_visible: true } as TableField)).toBe(true)
  })
})
