import { describe, expect, it } from 'vitest'
import {
  QUERY_SCHEME_NAME_MAX_CODE_POINTS,
  buildQuerySchemeCopyName,
  isValidQuerySchemeName,
  normalizeQuerySchemeName,
  querySchemeNameLength,
  truncateQuerySchemeName,
} from './name'

describe('query scheme name rules', () => {
  it('counts Unicode code points like the backend rune limit', () => {
    const name = '😀'.repeat(QUERY_SCHEME_NAME_MAX_CODE_POINTS)

    expect(name.length).toBe(QUERY_SCHEME_NAME_MAX_CODE_POINTS * 2)
    expect(querySchemeNameLength(name)).toBe(QUERY_SCHEME_NAME_MAX_CODE_POINTS)
    expect(isValidQuerySchemeName(name)).toBe(true)
    expect(isValidQuerySchemeName(`${name}😀`)).toBe(false)
  })

  it('normalizes surrounding whitespace and rejects blank names', () => {
    expect(normalizeQuerySchemeName('  常用方案  ')).toBe('常用方案')
    expect(isValidQuerySchemeName('   ')).toBe(false)
  })

  it('truncates by code point and reserves room for the copy suffix', () => {
    const source = '🚀'.repeat(QUERY_SCHEME_NAME_MAX_CODE_POINTS)
    const copyName = buildQuerySchemeCopyName(source)

    expect(truncateQuerySchemeName(`${source}x`)).toBe(source)
    expect(copyName.endsWith(' 副本')).toBe(true)
    expect(querySchemeNameLength(copyName)).toBe(QUERY_SCHEME_NAME_MAX_CODE_POINTS)
    expect(isValidQuerySchemeName(copyName)).toBe(true)
  })
})
