import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { MenuButton } from 'src/api/services/sys-menu'
import { SysMenuButtonPosition } from 'src/types/enum'
import OrganizationRecordDetailContent from './OrganizationRecordDetailContent.vue'
import OrganizationRecordDetailDialog from './OrganizationRecordDetailDialog.vue'

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

const FormDialogShellStub = defineComponent({
  props: {
    title: String,
    subtitle: String,
  },
  emits: ['update:modelValue'],
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-testid': 'form-dialog-shell' }, [
        h('h2', props.title),
        h('p', props.subtitle),
        slots['title-extra']?.(),
        slots['header-actions']?.(),
        slots.navigation?.(),
        slots.default?.(),
        slots['footer-actions']?.(),
      ])
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
          QAvatar: SlotStub,
          QBanner: SlotStub,
          QList: SlotStub,
          QItem: SlotStub,
          QItemSection: SlotStub,
          QItemLabel: SlotStub,
          QBadge: SlotStub,
          QScrollArea: SlotStub,
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

  it('renders the configured dialog navigation and only the active section fields', async () => {
    const wrapper = mount(OrganizationRecordDetailDialog, {
      props: {
        modelValue: true,
        mode: 'dialog',
        title: '李娜',
        subtitle: 'DEV-E0002',
        avatarLabel: '李',
        statusLabel: '在职',
        sections: [
          {
            key: 'basic',
            label: '基本资料',
            items: [{ label: '员工编号', value: 'DEV-E0002' }],
          },
          {
            key: 'assignments',
            label: '任职记录',
            count: 2,
            items: [],
          },
          {
            key: 'account',
            label: '账号信息',
            items: [{ label: '平台账号', value: '未绑定' }],
          },
        ],
      },
      global: {
        stubs: {
          FormDialogShell: FormDialogShellStub,
          QBanner: SlotStub,
          QSpinner: true,
          QChip: SlotStub,
          QIcon: true,
          QBtn: ButtonStub,
        },
      },
    })

    expect(wrapper.text()).toContain('李娜')
    expect(wrapper.text()).toContain('在职')
    expect(wrapper.text()).toContain('基本资料')
    expect(wrapper.text()).toContain('任职记录')
    expect(wrapper.text()).toContain('账号信息')
    expect(wrapper.text()).toContain('员工编号')
    expect(wrapper.text()).not.toContain('平台账号')

    const accountButton = wrapper
      .findAll('.detail-section-navigation__item')
      .find((button) => button.text().includes('账号信息'))
    await accountButton?.trigger('click')
    expect(wrapper.text()).toContain('平台账号')
    expect(wrapper.text()).not.toContain('员工编号')
  })

  it('uses the platform detail page header and field grid in page mode', () => {
    const wrapper = mount(OrganizationRecordDetailContent, {
      props: {
        mode: 'page',
        title: '同步批次详情：BATCH-001',
        subtitle: '全量',
        sections: [
          {
            key: 'basic',
            label: '基础信息',
            items: [
              { label: '批次号', value: 'BATCH-001' },
              { label: '状态', value: '成功', chip: true, color: 'positive' },
            ],
          },
        ],
      },
      global: {
        stubs: {
          QChip: SlotStub,
          QIcon: true,
          QSpace: true,
          QTooltip: SlotStub,
          QBtn: ButtonStub,
          QBanner: SlotStub,
          QSpinner: true,
        },
      },
    })

    expect(wrapper.find('.organization-detail-page-header').exists()).toBe(true)
    expect(wrapper.find('.detail-field-grid--card').exists()).toBe(true)
    expect(wrapper.text()).toContain('同步批次详情：BATCH-001')
    expect(wrapper.text()).toContain('批次号')
  })
})
