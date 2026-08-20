import { describe, expect, it } from 'vitest'
import type { Menu, MenuButton } from 'src/api/services/sys-menu'
import { hasGrantedActionCapability, resolvePageButtons } from './menu-button'

const pageButton = (overrides: Partial<MenuButton> = {}): MenuButton => ({
  id: 1,
  menu_id: 10,
  name: '查询',
  code: 'query',
  icon: 'search',
  color: 'primary',
  sequence: 1,
  memo: '',
  position: 1,
  event_type: 'api',
  event_action: 'query',
  api_path: '/admin/example/query',
  http_method: 'POST',
  disable_when: '',
  params_schema: '',
  confirm_text: '',
  is_button: true,
  is_disabled: false,
  state: true,
  ...overrides,
})

describe('menu button capabilities', () => {
  it('returns granted visible nested buttons in sequence order', () => {
    const menus = [
      {
        id: 1,
        name: 'platform',
        children: [
          {
            id: 10,
            name: 'example_page',
            menu_buttons: [
              pageButton({ id: 2, code: 'update', sequence: 20 }),
              pageButton({ id: 1, code: 'query', sequence: 10 }),
              pageButton({ id: 3, code: 'hidden', is_hidden: true }),
              pageButton({ id: 4, code: 'disabled', is_disabled: true }),
              pageButton({ id: 5, code: 'inactive', state: false }),
              pageButton({ id: 6, code: 'api_only', is_button: false }),
            ],
          },
        ],
      },
    ] as Menu[]

    expect(resolvePageButtons(menus, 'example_page').map((button) => button.code)).toEqual([
      'query',
      'update',
    ])
  })

  it('returns an empty list when the current route has no granted menu', () => {
    expect(resolvePageButtons([], 'missing_page')).toEqual([])
  })

  it('includes hidden API permissions in granted action capabilities', () => {
    const menus = [
      {
        name: 'system',
        children: [
          {
            name: 'query_scheme_capabilities',
            menu_buttons: [
              pageButton({
                code: 'query_scheme_shared_manage_create',
                event_action: 'query_scheme_shared_manage',
                is_button: false,
                is_hidden: true,
              }),
            ],
          },
        ],
      },
    ] as Menu[]

    expect(hasGrantedActionCapability(menus, 'query_scheme_shared_manage')).toBe(true)
    expect(hasGrantedActionCapability(menus, 'query_scheme_unknown')).toBe(false)
  })
})
