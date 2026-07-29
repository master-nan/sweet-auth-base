import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { MenuButton } from 'src/api/services/sys-menu'
import { SysMenuButtonPosition } from 'src/types/enum'
import OrganizationRecordDetailContent from './OrganizationRecordDetailContent.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const ButtonStub = defineComponent({
  inheritAttrs: false,
  emits: ['click'],
  setup(_, { attrs, emit, slots }) {
    return () => {
      const label = typeof attrs.label === 'string' ? attrs.label : ''
      const icon = typeof attrs.icon === 'string' ? attrs.icon : ''
      return h(
        'button',
        {
          disabled: Boolean(attrs.disable),
          onClick: () => emit('click'),
        },
        label || icon || slots.default?.(),
      )
    }
  },
})

const detailButton = {
  id: 1,
  code: 'organization_employee_bind_user',
  name: '绑定账号',
  event_action: 'bind_user',
  icon: 'link',
  color: 'primary',
  position: SysMenuButtonPosition.DETAIL_TOP,
  disable_when: '{"field":"row.user_id","op":"not_empty"}',
} as unknown as MenuButton

describe('OrganizationRecordDetailContent', () => {
  it('renders configured detail actions and applies record disable conditions', () => {
    const wrapper = mount(OrganizationRecordDetailContent, {
      props: {
        mode: 'page',
        title: '人员详情',
        items: [{ label: '员工编号', value: 'E001' }],
        topButtons: [detailButton],
        recordContext: { user_id: 9 },
      },
      global: {
        stubs: {
          QCard: SlotStub,
          QCardSection: SlotStub,
          QCardActions: SlotStub,
          QBanner: SlotStub,
          QSeparator: true,
          QSpinner: true,
          QChip: SlotStub,
          QIcon: true,
          QSpace: true,
          QTooltip: SlotStub,
          QBtn: ButtonStub,
        },
      },
    })

    expect(wrapper.text()).toContain('人员详情')
    expect(wrapper.text()).toContain('E001')
    const bindingButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('绑定账号'))
    expect(bindingButton?.attributes('disabled')).toBeDefined()
  })
})
