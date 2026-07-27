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
  it('renders business groups instead of a flat field collection', () => {
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

    const groups = wrapper.findAll('.organization-detail-group')
    expect(groups).toHaveLength(2)
    expect(groups.map((group) => group.find('h3').text())).toEqual([
      '基础信息',
      '状态信息',
    ])
    expect(wrapper.text()).toContain('财务中心')
    expect(wrapper.text()).toContain('FIN')
    expect(wrapper.text()).toContain('启用')
  })
})
