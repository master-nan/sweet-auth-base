import { describe, expect, it, vi } from 'vitest'

const menus = vi.hoisted(() => [
  {
    name: 'demo_page',
    menu_buttons: [
      { id: 1, code: 'demo_create', event_action: 'create', position: 2, is_button: true, state: true },
      { id: 2, code: 'demo_detail', event_action: 'detail', position: 1, is_button: true, state: true },
      { id: 3, code: 'demo_approve', event_action: 'approve', position: 6, is_button: true, state: true },
      { id: 4, code: 'demo_history', event_action: 'history', position: 7, is_button: true, state: true },
    ],
  },
])

vi.mock('src/stores/user', () => ({
  useUserStore: () => ({ menus, buttons: ['demo_query', 'demo_metadata'] }),
}))

import { usePageButtons } from './page-buttons'

describe('usePageButtons capabilities', () => {
  it('keeps business capabilities and detail positions under the parent page', () => {
    const buttons = usePageButtons('demo_page')
    expect(buttons.hasCapability('demo_create')).toBe(true)
    expect(buttons.hasActionCapability('detail')).toBe(true)
    expect(buttons.record_detail_top_buttons.value.map((button) => button.code)).toEqual([
      'demo_approve',
    ])
    expect(buttons.record_detail_bottom_buttons.value.map((button) => button.code)).toEqual([
      'demo_history',
    ])
    expect(buttons.hasActionCapability('refresh')).toBe(false)
    expect(buttons.hasGrantedCapability('demo_query')).toBe(true)
    expect(buttons.hasGrantedCapability('demo_delete')).toBe(false)
  })
})
