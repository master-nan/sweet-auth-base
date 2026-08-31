import { describe, expect, it } from 'vitest'
import { legacyEnglishMessages } from './legacy-en'
import { translateLegacyText } from './legacy-localizer'

describe('legacy UI localization', () => {
  it('covers the existing frontend UI vocabulary with English copy', () => {
    expect(Object.keys(legacyEnglishMessages).length).toBeGreaterThan(1_600)
    expect(legacyEnglishMessages['搜索']).toBe('Search')
    expect(legacyEnglishMessages['通知详情加载失败']).toBe(
      'Failed to load notification details',
    )
    expect(Object.values(legacyEnglishMessages).some((value) => /\p{Script=Han}/u.test(value))).toBe(
      false,
    )
  })

  it('translates exact UI copy while preserving surrounding whitespace', () => {
    expect(translateLegacyText('  搜索  ', 'en-US')).toBe('  Search  ')
    expect(translateLegacyText('搜索', 'zh-CN')).toBe('搜索')
  })

  it('supports dynamic count summaries used by existing pages', () => {
    expect(translateLegacyText('共 21 条通知', 'en-US')).toBe('21 notifications')
    expect(translateLegacyText('共 52 行', 'en-US')).toBe('52 rows')
    expect(translateLegacyText('当前版本：V3', 'en-US')).toBe('Current version: V3')
  })

  it('does not translate arbitrary backend business data', () => {
    expect(translateLegacyText('华东运输项目', 'en-US')).toBe('华东运输项目')
  })
})
