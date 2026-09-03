import { describe, expect, it, vi } from 'vitest'
import { assertControlledRuntimePath } from './generalization'

vi.mock('@/boot/axios', () => ({ instance: { request: vi.fn() } }))

describe('controlled runtime action path', () => {
  it('accepts only same-origin admin paths', () => {
    expect(assertControlledRuntimePath('/admin/generalization/export?format=csv')).toBe(
      '/admin/generalization/export?format=csv',
    )
    expect(() => assertControlledRuntimePath('https://example.com/admin/export')).toThrow()
    expect(() => assertControlledRuntimePath('//example.com/admin/export')).toThrow()
    expect(() => assertControlledRuntimePath('/admin/../secret')).toThrow()
  })
})
