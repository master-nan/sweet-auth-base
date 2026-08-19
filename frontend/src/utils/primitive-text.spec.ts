import { describe, expect, it } from 'vitest'
import { primitiveText } from './primitive-text'

describe('primitiveText', () => {
  it('formats primitive values without Object default stringification', () => {
    expect(primitiveText('text')).toBe('text')
    expect(primitiveText(12)).toBe('12')
    expect(primitiveText(false)).toBe('false')
    expect(primitiveText({ value: 1 }, '-')).toBe('-')
    expect(primitiveText(null, '-')).toBe('-')
  })
})
