import { describe, expect, it } from 'vitest'
import type { Menu } from 'src/api/services/sys-menu'
import { findMenuPathByTableCode } from './menu-context'

const menu = (value: Partial<Menu>): Menu =>
  ({
    id: 1,
    pid: 0,
    name: 'menu',
    path: '',
    component: '',
    title: '',
    is_hidden: false,
    sequence: 0,
    option: '',
    ...value,
  }) as Menu

describe('findMenuPathByTableCode', () => {
  it('uses the actual dynamic menu hierarchy instead of a fixed parent path', () => {
    const menus = [
      menu({
        id: 10,
        path: 'tms-demo',
        children: [
          menu({
            id: 11,
            pid: 10,
            path: 'company',
            page_type: 'low_code',
            table_code: 'company',
          }),
        ],
      }),
    ]

    expect(findMenuPathByTableCode(menus, 'company')).toBe('/admin/tms-demo/company')
  })
})
