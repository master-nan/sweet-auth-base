import { defineComponent, h, type Component } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getLegalEntityTree: vi.fn(),
  getLegalEntityDetail: vi.fn(),
  queryStructureOptions: vi.fn(),
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

vi.mock('quasar', async () => {
  const actual = await vi.importActual('quasar')
  return {
    ...actual,
    useQuasar: () => ({ dark: { isActive: false } }),
  }
})

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
  setup(_, { slots }) {
    return () =>
      h('div', { 'data-testid': 'master-detail' }, [
        slots['master-actions']?.(),
        slots['master-toolbar']?.(),
        slots['master-content']?.(),
        slots['detail-context']?.(),
        slots['detail-content']?.(),
      ])
  },
})

const TreeTableStub = defineComponent({
  name: 'TreeTable',
  props: {
    data: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['node-selected'],
  setup(props) {
    return () => h('div', { 'data-testid': 'tree-table' }, String(props.data.length))
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
  emits: ['update:modelValue', 'filter'],
  setup(props) {
    return () => h('div', { 'data-testid': 'structure-select' }, props.modelValue)
  },
})

const mountPage = (component: Component) =>
  shallowMount(component, {
    global: {
      stubs: {
        BaseContent: SlotHostStub,
        MasterDetailPage: MasterDetailPageStub,
        TreeTable: TreeTableStub,
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
      },
    },
  })

describe('Organization read-only center', () => {
  beforeEach(() => {
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
  })

  it('loads the legal-entity tree and source-safe detail without edit actions', async () => {
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
        children: [],
      },
    ])
    apiMocks.getLegalEntityDetail.mockResolvedValue({
      id: 10,
      code: 'LE-10',
      name: '集团',
      short_name: '集团',
      entity_type: 'group',
      unified_social_credit_code: '91310000TEST',
      accounting_code: 'AC-10',
      status: 'enabled',
      valid_from: '2026-01-01',
      valid_to: null,
      local_note: '',
      local_handling_status: '',
    })

    const wrapper = mountPage(LegalEntityPage)
    await flushPromises()

    expect(apiMocks.getLegalEntityTree).toHaveBeenCalledWith({ only_effective: true })
    expect(apiMocks.getLegalEntityDetail).toHaveBeenCalledWith(10, {
      only_effective: true,
    })
    expect(wrapper.findComponent(TreeTableStub).props('data')).toHaveLength(1)
    expect(wrapper.find('[data-testid="organization-detail"]').text()).toContain('LE-10')
    expect(wrapper.text()).not.toMatch(/新增|编辑|删除|调岗|离职/)
  })

  it('switches management trees by structure_code and keeps the page read-only', async () => {
    apiMocks.queryStructureOptions.mockResolvedValue({
      items: [
        {
          value: 20,
          code: 'MGMT-A',
          name: '行政管理架构',
          label: 'MGMT-A - 行政管理架构',
          disabled: false,
        },
        {
          value: 21,
          code: 'MGMT-B',
          name: '经营管理架构',
          label: 'MGMT-B - 经营管理架构',
          disabled: false,
        },
      ],
      total: 2,
    })
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
      valid_from: '2026-01-01',
      valid_to: null,
      local_note: '',
      local_handling_status: '',
    }))

    const wrapper = mountPage(StructurePage)
    await flushPromises()

    expect(apiMocks.getStructureOrgTree).toHaveBeenCalledWith({
      structure_id: 20,
      only_effective: true,
    })

    wrapper.findComponent(QSelectStub).vm.$emit('update:modelValue', 'MGMT-B')
    await flushPromises()

    expect(apiMocks.getStructureOrgTree).toHaveBeenLastCalledWith({
      structure_id: 21,
      only_effective: true,
    })
    expect(apiMocks.getOrgUnitDetail).toHaveBeenLastCalledWith(121, {
      only_effective: true,
    })
    expect(wrapper.text()).not.toMatch(/新增|编辑|删除|组织调整/)
  })
})
