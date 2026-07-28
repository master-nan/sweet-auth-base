import { defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getLegalEntityTree: vi.fn(),
  getLegalEntityDetail: vi.fn(),
  queryStructures: vi.fn(),
  getStructureOrgTree: vi.fn(),
  getOrgUnitDetail: vi.fn(),
}))

const routerMocks = vi.hoisted(() => ({
  route: { query: {} as Record<string, string> },
  replace: vi.fn(),
}))

vi.mock('src/api/services/org', () => apiMocks)

vi.mock('vue-router', () => ({
  useRoute: () => routerMocks.route,
  useRouter: () => ({ replace: routerMocks.replace }),
}))

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
  usePageButtons: () => ({
    top_buttons: {
      value: [
        {
          id: 2,
          code: 'organization_structure_refresh',
          name: '刷新',
          icon: 'refresh',
          event_action: 'refresh',
        },
      ],
    },
  }),
}))

import StructurePage from 'src/pages/organization/structure/Index.vue'

const SlotHostStub = defineComponent({
  name: 'SlotHostStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const QBtnStub = defineComponent({
  name: 'QBtn',
  props: {
    icon: {
      type: String,
      default: '',
    },
    label: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'button',
        {
          'data-icon': props.icon,
          'data-label': props.label,
        },
        slots.default?.(),
      )
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
    groups: {
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
        JSON.stringify(props.groups),
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
    optionLabel: {
      type: String,
      default: 'label',
    },
    label: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props) {
    const labels = () =>
      props.options.map((option) => {
        if (typeof option !== 'object' || option === null) return String(option)
        const label = (option as Record<string, unknown>)[props.optionLabel]
        return typeof label === 'string' || typeof label === 'number' ? String(label) : ''
      })
    return () =>
      h(
        'div',
        {
          'data-testid': `select-${props.label}`,
          'data-model-value': props.modelValue,
        },
        labels().join(','),
      )
  },
})

const mountPage = () =>
  shallowMount(StructurePage, {
    global: {
      stubs: {
        BaseContent: SlotHostStub,
        OrganizationReadOnlyTree: OrganizationTreeStub,
        OrganizationReadOnlyDetail: OrganizationDetailStub,
        QSelect: QSelectStub,
        QInput: SlotHostStub,
        QBtn: QBtnStub,
        QCard: SlotHostStub,
        QCardSection: SlotHostStub,
        QSeparator: true,
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
    routerMocks.route.query = {}
    routerMocks.replace.mockReset()
    routerMocks.replace.mockResolvedValue(undefined)
  })

  it('uses one organization page and hides the management-view selector for one Structure', async () => {
    mockSingleStructure()
    mockStructureTreeAndDetail()

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('h1').text()).toBe('组织架构')
    expect(wrapper.find('.organization-page-heading p').text()).toBe(
      '统一浏览管理组织与法人主体镜像',
    )
    expect(selectByLabel(wrapper, '架构类型').text()).toContain('管理架构')
    expect(selectByLabel(wrapper, '架构类型').text()).toContain('法人架构')
    expect(wrapper.find('[data-testid="select-管理视图"]').exists()).toBe(false)
    expect(wrapper.find('.organization-panel-title').text()).toBe('管理组织树')
    const refreshButton = wrapper.find('[data-icon="refresh"]')
    expect(refreshButton.exists()).toBe(true)
    expect(refreshButton.attributes('data-label')).toBe('')
    expect(wrapper.text()).not.toMatch(/详情按钮|操作列|新增|编辑|删除|调岗|离职/)
    expect(wrapper.find('[data-icon="visibility"]').exists()).toBe(false)
    expect(apiMocks.getStructureOrgTree).toHaveBeenCalledWith({
      structure_id: 20,
      only_effective: true,
    })
    expect(apiMocks.getOrgUnitDetail).toHaveBeenCalledWith(120, {
      only_effective: true,
    })
    expect(
      (wrapper.findComponent(OrganizationDetailStub).props('groups') as unknown[]),
    ).toHaveLength(1)
  })

  it('shows backend management-view names when multiple Structures exist', async () => {
    apiMocks.queryStructures.mockResolvedValue({
      items: [
        structure(20, 'GROUP', '集团组织视图', true),
        structure(21, 'REGION', '区域协作视图', false),
      ],
      total: 2,
    })
    mockStructureTreeAndDetail()

    const wrapper = mountPage()
    await flushPromises()

    const selector = selectByLabel(wrapper, '管理视图')
    expect(selector.exists()).toBe(true)
    expect(selector.text()).toContain('集团组织视图')
    expect(selector.text()).toContain('区域协作视图')
    expect(selector.text()).not.toMatch(/行政架构|经营架构/)

    selector.vm.$emit('update:modelValue', 'REGION')
    await flushPromises()

    expect(apiMocks.getStructureOrgTree).toHaveBeenLastCalledWith({
      structure_id: 21,
      only_effective: true,
    })
    expect(apiMocks.getOrgUnitDetail).toHaveBeenLastCalledWith(121, {
      only_effective: true,
    })
  })

  it('switches the same page from management architecture to legal architecture', async () => {
    mockSingleStructure()
    mockStructureTreeAndDetail()
    mockLegalTreeAndDetail()

    const wrapper = mountPage()
    await flushPromises()

    const architectureSelector = selectByLabel(wrapper, '架构类型')
    architectureSelector.vm.$emit('update:modelValue', 'legal')
    await flushPromises()

    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: { architecture: 'legal' },
    })
    expect(apiMocks.getLegalEntityTree).toHaveBeenCalledWith({
      only_effective: true,
    })
    expect(apiMocks.getLegalEntityDetail).toHaveBeenCalledWith(10, {
      only_effective: true,
    })
    expect(wrapper.find('.organization-panel-title').text()).toBe('法人树')
    expect(wrapper.find('[data-testid="select-管理视图"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="organization-tree"]').text()).toContain('集团法人')
    expect(wrapper.find('[data-testid="organization-detail"]').text()).toContain(
      '统一社会信用代码',
    )

    wrapper.findComponent(OrganizationTreeStub).vm.$emit('select', 11)
    await flushPromises()

    expect(apiMocks.getLegalEntityDetail).toHaveBeenLastCalledWith(11, {
      only_effective: true,
    })
  })

  it('opens legal architecture directly for the legacy hidden route', async () => {
    routerMocks.route.query = { architecture: 'legal' }
    mockLegalTreeAndDetail()

    const wrapper = mountPage()
    await flushPromises()

    expect(apiMocks.queryStructures).not.toHaveBeenCalled()
    expect(apiMocks.getLegalEntityTree).toHaveBeenCalledTimes(1)
    expect(selectByLabel(wrapper, '架构类型').attributes('data-model-value')).toBe('legal')
    expect(wrapper.find('.organization-panel-title').text()).toBe('法人树')
  })

  it('keeps technical identifiers out of both detail modes', async () => {
    mockSingleStructure()
    mockStructureTreeAndDetail()
    mockLegalTreeAndDetail()

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('[data-testid="organization-detail"]').text()).not.toMatch(
      /structure_node_id|parent_node_id|path|source_id|level/,
    )

    selectByLabel(wrapper, '架构类型').vm.$emit('update:modelValue', 'legal')
    await flushPromises()

    expect(wrapper.find('[data-testid="organization-detail"]').text()).not.toMatch(
      /structure_node_id|parent_node_id|path|source_id|level/,
    )
  })
})

function selectByLabel(
  wrapper: ReturnType<typeof mountPage>,
  label: string,
) {
  const selector = wrapper
    .findAllComponents(QSelectStub)
    .find((select) => select.props('label') === label)
  if (!selector) throw new Error(`missing select: ${label}`)
  return selector
}

function structure(id: number, code: string, name: string, isDefault: boolean) {
  return {
    id,
    code,
    name,
    structure_type: 'management',
    status: 'enabled',
    is_default: isDefault,
  }
}

function mockSingleStructure() {
  apiMocks.queryStructures.mockResolvedValue({
    items: [structure(20, 'GROUP', '集团组织视图', true)],
    total: 1,
  })
}

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
      name: '集团法人',
    },
    valid_from: '2026-01-01',
    valid_to: null,
    local_note: '',
    local_handling_status: '',
  }))
}

function mockLegalTreeAndDetail() {
  apiMocks.getLegalEntityTree.mockResolvedValue([
    {
      id: 10,
      legal_entity_id: 10,
      value: 10,
      label: 'LE-10 - 集团法人',
      code: 'LE-10',
      name: '集团法人',
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
    name: id === 10 ? '集团法人' : '子公司',
    short_name: id === 10 ? '集团' : '子公司',
    entity_type: id === 10 ? 'group' : 'legal_company',
    parent_id: id === 11 ? 10 : null,
    unified_social_credit_code: '91310000TEST',
    accounting_code: `AC-${id}`,
    status: 'enabled',
    valid_from: '2026-01-01',
    valid_to: null,
    local_note: '',
    local_handling_status: '',
  }))
}
