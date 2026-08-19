import { describe, expect, it, vi } from 'vitest'
import { resolveRouteTitle } from './route-title'

describe('resolveRouteTitle', () => {
  it('translates registered route keys', () => {
    const translate = vi.fn(() => '用户管理')

    expect(resolveRouteTitle('router.system.user', translate)).toBe('用户管理')
    expect(translate).toHaveBeenCalledWith('router.system.user')
  })

  it('keeps backend dynamic menu titles without invoking i18n', () => {
    const translate = vi.fn()

    expect(resolveRouteTitle('TMS运单', translate)).toBe('TMS运单')
    expect(translate).not.toHaveBeenCalled()
  })
})
