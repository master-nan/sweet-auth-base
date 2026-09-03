import { afterEach, describe, expect, it, vi } from 'vitest'

const menus = vi.hoisted(() => [
  {
    name: 'demo_page',
    menu_buttons: [
      {
        id: 1,
        name: '自定义新增',
        code: 'demo_create',
        event_action: 'create',
        position: 2,
        is_button: true,
        state: true,
      },
      {
        id: 2,
        name: '自定义详情',
        code: 'demo_detail',
        event_action: 'detail',
        position: 1,
        is_button: true,
        state: true,
      },
      {
        id: 3,
        name: '审批',
        code: 'demo_approve',
        event_action: 'approve',
        position: 6,
        is_button: true,
        state: true,
      },
      {
        id: 4,
        name: '历史',
        code: 'demo_history',
        event_action: 'history',
        position: 7,
        is_button: true,
        state: true,
      },
      {
        id: 5,
        name: '新增资源',
        code: 'system_data_permission_config_resource_create',
        event_action: 'create_resource',
        position: 2,
        is_button: true,
        state: true,
      },
    ],
  },
])

vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ menus, buttons: ['demo_query', 'demo_metadata'] }),
}))

import { usePageButtons } from './page-buttons'
import { i18n } from '@/boot/i18n'

describe('usePageButtons capabilities', () => {
  afterEach(() => {
    i18n.global.locale.value = 'zh-CN'
  })

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

  it('translates built-in buttons and preserves custom metadata names', () => {
    const buttons = usePageButtons('demo_page')
    expect(
      buttons.all_buttons.value.find(
        (button) => button.code === 'system_data_permission_config_resource_create',
      )?.name,
    ).toBe('新增资源')

    i18n.global.locale.value = 'en-US'
    const names = Object.fromEntries(
      buttons.all_buttons.value.map((button) => [button.code, button.name]),
    )

    expect(names.system_data_permission_config_resource_create).toBe('Add Resource')
    expect(names.demo_create).toBe('自定义新增')
  })
})
