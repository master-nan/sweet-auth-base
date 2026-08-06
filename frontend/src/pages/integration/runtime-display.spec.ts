import { describe, expect, it } from 'vitest'
import { formatRuntimeDateTime } from './runtime-display'

describe('formatRuntimeDateTime', () => {
  it('hides empty and Go zero timestamps', () => {
    expect(formatRuntimeDateTime()).toBe('-')
    expect(formatRuntimeDateTime('0001-01-01T00:00:00Z')).toBe('-')
  })

  it('formats valid timestamps and rejects invalid values', () => {
    expect(formatRuntimeDateTime('2026-08-06T12:30:00Z')).not.toBe('-')
    expect(formatRuntimeDateTime('invalid')).toBe('-')
  })
})
