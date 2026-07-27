import { defineComponent, h, type Component } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getLegalEntityTree: vi.fn(),
  getLegalEntityDetail: vi.fn(),
  queryStructures: vi.fn(),
  getStructureOrgTree: vi.fn(),
  getOrgUnitDetail: vi.fn(),
}))

vi.mock('src/api/services/org', () => apiMocks)

vi.mock('src/stores/dict', () => ({
  useDictStore: () => ({
    loadDicts: vi.fn().mockResolvedValue(undefined),
    getDictLabel: (_dictCode: string, value: unknown) =>
      typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
        ? String(value)
        : '',
  }),
}))

vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => {
    const detail = {
      id: 1,
      code: 'detail',
      name: '详情',
      icon: 'visibility',
      event_action: 'detail',
    }
    const refresh = {
      id: 2,
      code: 'refresh',
      name: '刷新',
      icon: 'refresh',
      event_action: 'refresh',
    }
    return {
      all_buttons: { value: [detail, refresh] },
      top_buttons: { value: [refresh] },
      line_buttons: { value: [detail] },
    }
  },
}))

vi.mock('src/utils/menu-button-display', () => ({
  menuButtonDisplayProps: (button: { icon?: string }) => ({ icon: button.icon }),
}))

import LegalEntityPage from 'src/pages/organization/legal-entity/Index.vue'
import StructurePage from 'src/pages/organization/structure/Index.vue'

const SlotHostStub = defineComponent({
  name: 'SlotHostStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const MasterDetailPageStub = defineComponent({
  name: 'MasterDetailPage',
  props: {
    masterTitle: {
      type: String,
      default: '',
    },
    masterSubtitle: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-testid': 'master-detail' }, [
        h('div', { 'data-testid': 'master-title' }, props.masterTitle),
        h('div', { 'data-testid': 'master-subtitle' }, props.masterSubtitle),
        slots['master-actions']?.(),
        slots['master-toolbar']?.(),
        slots['master-content']?.(),
        slots['detail-context']?.(),
        slots['detail-content']?.(),
      ])
  },
})

const OrganizationTreeStub = defineComponent({
  name: 'OrganizationReadOnlyTree',
  props: {
    nodes: {
      type: Array,
      default: () => [],
    },
    selectedId: {
      type: Number,
      default: null,
    },
  },
  emits: ['select'],
  setup(props) {
    return () => h('div', { 'data-testid': 'organization-tree' }, JSON.stringify(props.nodes))
  },
})

const OrganizationDetailStub = defineComponent({
  name: 'OrganizationReadOnlyDetail',
  props: {
    fields: {
      type: Array,
      default: () => [],
    },
    error: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    return () =>
      h('div', { 'data-testid': 'organization-detail' }, [
        props.error,
        JSON.stringify(props.fields),
      ])
  },
})

const QSelectStub = defineComponent({
  name: 'QSelect',
  props: {
    modelValue: {
      type: String,
      default: null,
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props) {
    return () =>
      h(
        'div',
        { 'data-testid': 'structure-select' },
        `${props.modelValue || ''}:${JSON.stringify(props.options)}`,
      )
  },
})

const mountPage = (component: Component) =>
  shallowMount(component, {
    global: {
      stubs: {
        BaseContent: SlotHostStub,
        MasterDetailPage: MasterDetailPageStub,
        OrganizationReadOnlyTree: OrganizationTreeStub,
        OrganizationReadOnlyDetail: OrganizationDetailStub,
        QSelect: QSelectStub,
        QInput: SlotHostStub,
        QBtn: SlotHostStub,
        QTooltip: SlotHostStub,
        QIcon: true,
        QChip: SlotHostStub,
        QBanner: SlotHostStub,
        QItem: SlotHostStub,
        QItemSection: SlotHostStub,
        QItemLabel: SlotHostStub,
      },
    },
  })

describe('Organization read-only center', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
  })

  it('renders the legal-entity hierarchy as 法人主体 and opens detail from node selection', async () => {
    apiMocks.getLegalEntityTree.mockResolvedValue([
      {
        id: 10,
        legal_entity_id: 10,
        value: 10,
        label: 'LE-10 - 集团',
        code: 'LE-10',
        name: '集团',
        short_name: '集团',
        entity_type: 'group',
        status: 'enabled',
        disabled: false,
        children: [
          {
            id: 11,
            legal_entity_id: 11,
            value: 11,
            label: 'LE-11 - 子公司',
            code: 'LE-11',
            name: '子公司',
            short_name: '子公司',
            entity_type: 'legal_company',
            parent_id: 10,
            status: 'enabled',
            disabled: false,
            children: [],
          },
        ],
      },
    ])
    apiMocks.getLegalEntityDetail.mockImplementation((id: number) => ({
      id,
      code: `LE-${id}`,
      name: id === 10 ? '集团' : '子公司',
      short_name: '',
      entity_type: id === 10 ? 'group' : 'legal_company',
      parent_id: id === 11 ? 10 : null,
      unified_social_credit_code: '',
      accounting_code: '',
      status: 'enabled',
      valid_from: '2026-01-01',
      valid_to: null,
      local_note: '',
      local_handling_status: '',
    }))

    const wrapper = mountPage(LegalEntityPage)
    await flushPromises()

    expect(wrapper.findComponent(MasterDetailPageStub).props('masterTitle')).toBe('法人主体')
    expect(wrapper.text()).not.toContain('法人架构')
    expect(apiMocks.getLegalEntityTree).toHaveBeenCalledWith({ only_effective: true })
    expect(apiMocks.getLegalEntityDetail).toHaveBeenCalledWith(10, {
      only_effective: true,
    })

    wrapper.findComponent(OrganizationTreeStub).vm.$emit('select', 11)
    await flushPromises()

    expect(apiMocks.getLegalEntityDetail).toHaveBeenLastCalledWith(11, {
      only_effective: true,
    })
    expect(wrapper.find('[data-testid="organization-detail"]').text()).toContain(
      'LE-10 - 集团',
    )
    expect(wrapper.text()).not.toMatch(/新增|编辑|删除|调岗|离职/)
  })

  it('automatically loads a single Structure and hides the organization-view switcher', async () => {
    apiMocks.queryStructures.mockResolvedValue({
      items: [
        {
          id: 20,
          code: 'GROUP',
          name: '集团组织视图',
          structure_type: 'management',
          status: 'enabled',
          is_default: true,
        },
      ],
      total: 1,
    })
    mockStructureTreeAndDetail()

    const wrapper = mountPage(StructurePage)
    await flushPromises()

    expect(wrapper.findComponent(MasterDetailPageStub).props('masterTitle')).toBe('组织架构')
    expect(wrapper.findComponent(QSelectStub).exists()).toBe(false)
    expect(wrapper.find('[data-testid="master-subtitle"]').text()).toContain('集团组织视图')
    expect(apiMocks.getStructureOrgTree).toHaveBeenCalledWith({
      structure_id: 20,
      only_effective: true,
    })
  })

  it('shows backend Structure names for multiple views and switches tree context', async () => {
    apiMocks.queryStructures.mockResolvedValue({
      items: [
        {
          id: 20,
          code: 'GROUP',
          name: '集团组织视图',
          structure_type: 'management',
          status: 'enabled',
          is_default: true,
        },
        {
          id: 21,
          code: 'REGION',
          name: '区域协作视图',
          structure_type: 'management',
          status: 'enabled',
          is_default: false,
        },
      ],
      total: 2,
    })
    mockStructureTreeAndDetail()

    const wrapper = mountPage(StructurePage)
    await flushPromises()

    const selector = wrapper.findComponent(QSelectStub)
    expect(selector.exists()).toBe(true)
    expect(selector.props('options')).toEqual([
      expect.objectContaining({ code: 'GROUP', name: '集团组织视图' }),
      expect.objectContaining({ code: 'REGION', name: '区域协作视图' }),
    ])
    expect(apiMocks.getStructureOrgTree).toHaveBeenCalledWith({
      structure_id: 20,
      only_effective: true,
    })

    selector.vm.$emit('update:modelValue', 'REGION')
    await flushPromises()

    expect(apiMocks.getStructureOrgTree).toHaveBeenLastCalledWith({
      structure_id: 21,
      only_effective: true,
    })
    expect(apiMocks.getOrgUnitDetail).toHaveBeenLastCalledWith(121, {
      only_effective: true,
    })
    expect(wrapper.text()).not.toMatch(/行政架构|经营架构|新增|编辑|删除|组织调整/)
  })

  it('keeps technical structure-node fields out of organization detail', async () => {
    apiMocks.queryStructures.mockResolvedValue({
      items: [
        {
          id: 20,
          code: 'GROUP',
          name: '集团组织视图',
          structure_type: 'management',
          status: 'enabled',
          is_default: true,
        },
      ],
      total: 1,
    })
    mockStructureTreeAndDetail()

    const wrapper = mountPage(StructurePage)
    await flushPromises()

    const detailText = wrapper.find('[data-testid="organization-detail"]').text()
    expect(detailText).not.toMatch(
      /structure_node_id|parent_node_id|path|source_id|架构层级|节点状态/,
    )
    expect(detailText).toContain('组织视图')
    expect(detailText).toContain('主要法人')
  })
})

function mockStructureTreeAndDetail() {
  apiMocks.getStructureOrgTree.mockImplementation(
    ({ structure_id }: { structure_id: number }) => [
      {
        id: structure_id * 10,
        structure_node_id: structure_id * 10,
        structure_id,
        org_unit_id: structure_id + 100,
        code: `OU-${structure_id}`,
        name: `组织-${structure_id}`,
        unit_type: 'department',
        status: 'enabled',
        node_status: 'enabled',
        level: 1,
        sort: 1,
        disabled: false,
        children: [],
      },
    ],
  )
  apiMocks.getOrgUnitDetail.mockImplementation((orgUnitId: number) => ({
    id: orgUnitId,
    code: `OU-${orgUnitId}`,
    name: `组织-${orgUnitId}`,
    unit_type: 'department',
    status: 'enabled',
    primary_legal_entity: {
      id: 10,
      code: 'LE-10',
      name: '集团',
    },
    valid_from: '2026-01-01',
    valid_to: null,
    local_note: '',
    local_handling_status: '',
  }))
}
