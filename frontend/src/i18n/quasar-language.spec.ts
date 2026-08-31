import { afterEach, describe, expect, it, vi } from 'vitest'

const setLanguage = vi.hoisted(() => vi.fn())

vi.mock('quasar', () => ({ Lang: { set: setLanguage } }))
vi.mock('quasar/lang/en-US', () => ({ default: { isoName: 'en-US' } }))
vi.mock('quasar/lang/zh-CN', () => ({ default: { isoName: 'zh-CN' } }))

import { applyQuasarLanguage } from './quasar-language'

describe('Quasar language synchronization', () => {
  afterEach(() => {
    setLanguage.mockReset()
    document.documentElement.removeAttribute('lang')
  })

  it('selects the matching Quasar pack and document language', () => {
    applyQuasarLanguage('en-US')
    expect(setLanguage).toHaveBeenLastCalledWith({ isoName: 'en-US' })
    expect(document.documentElement.lang).toBe('en-US')

    applyQuasarLanguage('zh-CN')
    expect(setLanguage).toHaveBeenLastCalledWith({ isoName: 'zh-CN' })
    expect(document.documentElement.lang).toBe('zh-CN')
  })
})
