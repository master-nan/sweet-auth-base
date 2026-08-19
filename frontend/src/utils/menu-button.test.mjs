import assert from 'node:assert/strict'
import test from 'node:test'

import { hasGrantedActionCapability, resolvePageButtons } from './menu-button.ts'

const pageButton = (overrides = {}) => ({
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

test('resolvePageButtons returns granted visible buttons for a nested menu in sequence order', () => {
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
  ]

  const buttons = resolvePageButtons(menus, 'example_page')

  assert.deepEqual(
    buttons.map((button) => button.code),
    ['query', 'update'],
  )
})

test('resolvePageButtons returns an empty list when the current route has no granted menu', () => {
  assert.deepEqual(resolvePageButtons([], 'missing_page'), [])
})

test('granted action capability includes hidden API permissions in nested menus', () => {
  const menus = [{
    name: 'system',
    children: [{
      name: 'query_scheme_capabilities',
      menu_buttons: [pageButton({
        code: 'query_scheme_shared_manage_create',
        event_action: 'query_scheme_shared_manage',
        is_button: false,
        is_hidden: true,
      })],
    }],
  }]

  assert.equal(hasGrantedActionCapability(menus, 'query_scheme_shared_manage'), true)
  assert.equal(hasGrantedActionCapability(menus, 'query_scheme_unknown'), false)
})
