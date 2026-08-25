import { computed, defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  queryExternalSystems: vi.fn(),
  getExternalSystem: vi.fn(),
  createExternalSystem: vi.fn(),
  updateExternalSystem: vi.fn(),
  enableExternalSystem: vi.fn(),
  disableExternalSystem: vi.fn(),
}))
const routerPush = vi.hoisted(() => vi.fn())
const schemeMocks = vi.hoisted(() => ({ initialize: vi.fn() }))

const tableApiMocks = vi.hoisted(() => ({
  queryRuntimeTableByCode: vi.fn(),
}))

const permissionButtons = vi.hoisted(() => ({
  top: [{ id: 1, name: '新增', event_action: 'create', icon: 'add', color: 'primary' }],
  line: [
    { id: 2, name: '详情', event_action: 'detail', icon: 'visibility', color: 'primary' },
    { id: 3, name: '编辑', event_action: 'update', icon: 'edit', color: 'primary' },
    { id: 4, name: '启用', event_action: 'enable', icon: 'play_arrow', color: 'positive' },
    { id: 5, name: '停用', event_action: 'disable', icon: 'block', color: 'warning' },
  ],
}))

vi.mock('quasar', () => ({
  useQuasar: () => ({ screen: { lt: { md: false } } }),
}))

vi.mock('boot/axios', () => ({ instance: {} }))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ menus: [], buttons: [] }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: routerPush }) }))

vi.mock('src/api/services/integration', () => ({
  useIntegrationApi: () => apiMocks,
}))

vi.mock('src/composables/query-scheme-page', async () => {
  const { createQuerySchemePageStub } = await import('src/test/query-scheme-page-stub')
  return {
    useQuerySchemePage: () => createQuerySchemePageStub(schemeMocks.initialize),
  }
})

vi.mock('src/api/services/sys-table', () => ({
  useTableApi: () => tableApiMocks,
}))

vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    top_buttons: computed(() => permissionButtons.top),
    line_buttons: computed(() => permissionButtons.line),
    has_line_buttons: computed(() => permissionButtons.line.length > 0),
  }),
}))

vi.mock('src/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({
    confirmAction: vi.fn(() => ({ onOk: vi.fn() })),
  }),
}))

vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

vi.mock('src/components/Table/TablePagination.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('src/components/FormDialog/DynamicFormDialog.vue', () => ({
  default: { template: '<div />' },
}))

vi.mock('./ExternalSystemDetailDialog.vue', () => ({
  default: { template: '<div />' },
}))

import ExternalSystemPage from './Index.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const ToolbarStub = defineComponent({
  emits: ['refresh'],
  setup(_, { slots }) {
    return () =>
      h(
        'div',
        Object.values(slots).flatMap((slot) => slot?.() || []),
      )
  },
})

const ButtonStub = defineComponent({
  name: 'QBtn',
  inheritAttrs: false,
  props: {
    label: { type: String, default: '' },
    icon: { type: String, default: '' },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, [props.label, slots.default?.()])
  },
})

const TableStub = defineComponent({
  name: 'QTable',
  props: {
    rows: { type: Array, default: () => [] },
  },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-testid': 'table', 'data-row-count': props.rows.length }, [
        slots.top?.(),
        slots.bottom?.(),
      ])
  },
})

const FormDialogStub = defineComponent({
  name: 'DynamicFormDialog',
  props: {
    modelValue: Boolean,
    editData: { type: Object, default: null },
    title: { type: String, default: '' },
  },
  emits: ['submit', 'update:modelValue'],
  setup(props) {
    return () =>
      props.modelValue
        ? h('div', { 'data-testid': 'form-dialog', 'data-title': props.title })
        : null
  },
})

const mountPage = () =>
  shallowMount(ExternalSystemPage, {
    global: {
      plugins: [createPinia()],
      stubs: {
        BaseContent: SlotStub,
        QTable: TableStub,
        QInput: true,
        QSelect: true,
        QBtn: ButtonStub,
        QIcon: true,
        QSpace: true,
        QBadge: true,
        QTooltip: true,
        QChip: true,
        QTd: SlotStub,
        StandardTableToolbar: ToolbarStub,
        StatusChip: true,
        TablePagination: true,
        DynamicFormDialog: FormDialogStub,
        ExternalSystemDetailDialog: true,
      },
    },
  })

const row = {
  id: 12,
  system_code: 'demo_erp',
  name: 'Demo ERP',
  system_type: 'erp',
  base_url_summary: 'https://erp.example.com',
  owner_identifier: 'owner-1',
  owner_name: '实施负责人',
  status: 'draft',
  revision: 2,
  gmt_modify: '2026-08-04 10:00:00',
}

describe('external system management page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
    routerPush.mockReset()
    schemeMocks.initialize.mockReset()
    schemeMocks.initialize.mockResolvedValue(false)
    tableApiMocks.queryRuntimeTableByCode.mockReset()
    apiMocks.queryExternalSystems.mockResolvedValue({ data: [row], total: 1 })
    apiMocks.getExternalSystem.mockResolvedValue({
      success: true,
      data: { ...row, base_url: 'https://erp.example.com', description: '', gmt_create: '' },
    })
    apiMocks.createExternalSystem.mockResolvedValue({ success: true })
    apiMocks.updateExternalSystem.mockResolvedValue({ success: true })
    tableApiMocks.queryRuntimeTableByCode.mockResolvedValue({
      success: true,
      data: {
        table_fields: [
          {
            field_code: 'system_code',
            field_name: '系统编码',
            is_list_show: true,
            is_sort: true,
            sequence: 1,
          },
        ],
      },
    })
  })

  it('loads metadata and the paged list, then renders granted top actions', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(tableApiMocks.queryRuntimeTableByCode).toHaveBeenCalledWith(
      'integration_external_system',
    )
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
    expect(schemeMocks.initialize.mock.invocationCallOrder[0]).toBeLessThan(
      apiMocks.queryExternalSystems.mock.invocationCallOrder[0]!,
    )
    expect(apiMocks.queryExternalSystems).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, num: 20, quick_query: { keyword: '' } }),
    )
    expect(wrapper.find('[data-testid="table"]').attributes('data-row-count')).toBe('1')
    expect(wrapper.text()).toContain('新增')

    await (wrapper.vm as unknown as { fetchData: () => Promise<void> }).fetchData()
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
  })

  it('opens the create dialog only through the configured dynamic button', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const createButton = wrapper.findAll('button').find((button) => button.text() === '新增')
    await createButton?.trigger('click')

    expect(wrapper.find('[data-testid="form-dialog"]').attributes('data-title')).toBe(
      '新增外部系统',
    )
  })

  it('filters enable and disable actions by the current status', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as {
      availableLineButtons: (item: typeof row) => Array<{ event_action: string }>
    }

    expect(vm.availableLineButtons(row).map((button) => button.event_action)).toEqual([
      'detail',
      'update',
      'enable',
    ])
    expect(
      vm.availableLineButtons({ ...row, status: 'enabled' }).map((button) => button.event_action),
    ).toEqual(['detail', 'update', 'disable'])
  })

  it('navigates from detail to system-scoped interface and credential lists', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as {
      openRelatedInterfaces: (id: number) => void
      openRelatedCredentials: (id: number) => void
    }
    vm.openRelatedInterfaces(12)
    vm.openRelatedCredentials(12)
    expect(routerPush).toHaveBeenNthCalledWith(1, {
      name: 'integration_interface_definition',
      query: { external_system_id: '12' },
    })
    expect(routerPush).toHaveBeenNthCalledWith(2, {
      name: 'integration_credential',
      query: { external_system_id: '12' },
    })
  })

  it('submits an edit with the server revision and without the immutable code', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as {
      handleFormSubmit: (payload: Record<string, unknown>) => Promise<void>
    }

    await vm.handleFormSubmit({
      isEdit: true,
      id: 12,
      data: {
        ...row,
        base_url: 'https://erp.example.com',
        description: 'ERP 集成入口',
      },
    })

    expect(apiMocks.updateExternalSystem).toHaveBeenCalledWith(12, {
      name: 'Demo ERP',
      system_type: 'erp',
      base_url: 'https://erp.example.com',
      owner_identifier: 'owner-1',
      owner_name: '实施负责人',
      description: 'ERP 集成入口',
      revision: 2,
    })
    expect(apiMocks.updateExternalSystem.mock.calls[0]?.[1]).not.toHaveProperty('system_code')
  })
})
