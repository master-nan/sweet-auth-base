import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import OrganizationReadOnlyDetail from './OrganizationReadOnlyDetail.vue'

const SlotStub = defineComponent({
  name: 'SlotStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.())
  },
})

describe('OrganizationReadOnlyDetail', () => {
  it('renders all business fields in one continuous detail area', () => {
    const wrapper = mount(OrganizationReadOnlyDetail, {
      props: {
        groups: [
          {
            key: 'basic',
            title: '基础信息',
            icon: 'domain',
            fields: [
              { key: 'name', label: '组织名称', value: '财务中心' },
              { key: 'code', label: '组织编码', value: 'FIN', kind: 'code' as const },
            ],
          },
          {
            key: 'status',
            title: '状态信息',
            icon: 'event_available',
            fields: [
              {
                key: 'status',
                label: '状态',
                value: '启用',
                kind: 'status' as const,
                color: 'positive',
              },
            ],
          },
        ],
      },
      global: {
        stubs: {
          QInnerLoading: SlotStub,
          QSpinner: true,
          QBanner: SlotStub,
          QIcon: true,
          QChip: SlotStub,
        },
      },
    })

    expect(wrapper.text()).toContain('财务中心')
    expect(wrapper.text()).toContain('FIN')
    expect(wrapper.text()).toContain('启用')
    expect(wrapper.findAll('.organization-detail-grid')).toHaveLength(1)
    expect(wrapper.findAll('.organization-detail-field')).toHaveLength(3)
    expect(wrapper.find('.organization-detail-group').exists()).toBe(false)
    expect(wrapper.find('h3').exists()).toBe(false)
  })
})
