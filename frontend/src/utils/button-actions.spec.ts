import { describe, expect, it, vi } from 'vitest'
import type { MenuButton } from 'src/api/services/sys-menu'
import { dispatchPageAction } from './button-actions'

const button = (action: string) => ({ event_action: action }) as MenuButton

describe('dispatchPageAction', () => {
  it('dispatches common actions only when the current page supplies a handler', () => {
    const edit = vi.fn()
    expect(dispatchPageAction(button('edit'), { edit }, { id: 7 })).toBe(true)
    expect(edit).toHaveBeenCalledWith({ id: 7 }, expect.objectContaining({ event_action: 'edit' }))
    expect(dispatchPageAction(button('domain_only'), { edit }, { id: 7 })).toBe(false)
  })
})
