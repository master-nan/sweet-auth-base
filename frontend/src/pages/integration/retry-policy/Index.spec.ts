import { computed, defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  queryRetryPolicies: vi.fn(),
  getRetryPolicy: vi.fn(),
  createRetryPolicy: vi.fn(),
  updateRetryPolicy: vi.fn(),
  createRetryPolicyVersion: vi.fn(),
  enableRetryPolicy: vi.fn(),
  disableRetryPolicy: vi.fn(),
}))
const tableApiMocks = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
const schemeMocks = vi.hoisted(() => ({ initialize: vi.fn() }))
const buttons = vi.hoisted(() => ({
  top: [{ id: 1, name: '新增', event_action: 'create', icon: 'add', color: 'primary' }],
  line: [
    { id: 2, name: '详情', event_action: 'detail' },
    { id: 3, name: '编辑', event_action: 'update' },
    { id: 4, name: '创建版本', event_action: 'create_version' },
    { id: 5, name: '启用', event_action: 'enable' },
    { id: 6, name: '停用', event_action: 'disable' },
  ],
}))
vi.mock('quasar', () => ({ useQuasar: () => ({ screen: { lt: { md: false } } }) }))
vi.mock('boot/axios', () => ({ instance: {} }))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ menus: [], buttons: [] }) }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), currentRoute: { value: { query: {} } } }),
}))
vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => apiMocks }))
vi.mock('src/composables/query-scheme-page', async () => {
  const { ref } = await import('vue')
  return {
    useQuerySchemePage: () => ({
      runtime: {
        schemes: ref([]),
        currentLabel: ref('查询方案'),
        loading: ref(false),
        error: ref(''),
        scope: { config: ref(null) },
        loadAvailable: vi.fn(),
      },
      showSaveDialog: ref(false),
      saving: ref(false),
      initialize: schemeMocks.initialize,
      selectScheme: vi.fn(),
      applyPreset: vi.fn(),
      restoreCurrent: vi.fn(),
      resetDefault: vi.fn(),
      openManager: vi.fn(),
      savePersonal: vi.fn(),
    }),
  }
})
vi.mock('src/api/services/sys-table', () => ({ useTableApi: () => tableApiMocks }))
vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    top_buttons: computed(() => buttons.top),
    line_buttons: computed(() => buttons.line),
    has_line_buttons: computed(() => true),
  }),
}))
vi.mock('src/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({ confirmAction: vi.fn(() => ({ onOk: vi.fn() })) }),
}))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))
vi.mock('src/components/Table/TablePagination.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/Query/AdvancedQuery.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./RetryPolicyFormDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./RetryPolicyDetailDialog.vue', () => ({ default: { template: '<div />' } }))

import RetryPolicyPage from './Index.vue'
import type { RetryPolicyListItem } from 'src/api/services/integration'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const ToolbarStub = defineComponent({
  setup(_, { slots }) {
    return () =>
      h(
        'div',
        Object.values(slots).flatMap((slot) => slot?.() || []),
      )
  },
})
const TableStub = defineComponent({
  props: { rows: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-testid': 'table', 'data-row-count': props.rows.length }, [
        slots.top?.(),
        slots.bottom?.(),
      ])
  },
})
const mountPage = () =>
  shallowMount(RetryPolicyPage, {
    global: {
      plugins: [createPinia()],
      stubs: {
        BaseContent: SlotStub,
        QTable: TableStub,
        QInput: true,
        QSelect: true,
        QBtn: true,
        QIcon: true,
        QSpace: true,
        QBadge: true,
        QTooltip: true,
        QChip: true,
        QTd: SlotStub,
        TablePagination: true,
        AdvancedQuery: true,
        RetryPolicyFormDialog: true,
        RetryPolicyDetailDialog: true,
        StandardTableToolbar: ToolbarStub,
        StatusChip: true,
      },
    },
  })
const row: RetryPolicyListItem = {
  id: 91,
  policy_code: 'hr_retry',
  policy_name: 'HR 重试',
  version: 1,
  status: 'draft',
  max_attempts: 3,
  backoff_type: 'exponential',
  initial_delay_ms: 5000,
  max_delay_ms: 300000,
  retry_window_ms: 86400000,
  revision: 1,
  gmt_modify: '',
}

describe('retry policy management page', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
    schemeMocks.initialize.mockReset()
    schemeMocks.initialize.mockResolvedValue(false)
    apiMocks.queryRetryPolicies.mockResolvedValue({ data: [row], total: 1 })
    tableApiMocks.queryRuntimeTableByCode.mockResolvedValue({
      success: true,
      data: { table_fields: [] },
    })
  })

  it('loads metadata and the policy page through the dynamic permission surface', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(tableApiMocks.queryRuntimeTableByCode).toHaveBeenCalledWith('integration_retry_policy')
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
    expect(schemeMocks.initialize.mock.invocationCallOrder[0]).toBeLessThan(
      apiMocks.queryRetryPolicies.mock.invocationCallOrder[0]!,
    )
    expect(apiMocks.queryRetryPolicies).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, num: 15 }),
    )
    expect(wrapper.find('[data-testid="table"]').attributes('data-row-count')).toBe('1')
    await (wrapper.vm as unknown as { fetchData: () => Promise<void> }).fetchData()
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
  })

  it('enforces version and state action visibility without delete or runtime actions', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const vm = wrapper.vm as unknown as {
      availableLineButtons: (item: typeof row) => Array<{ event_action: string }>
    }
    expect(vm.availableLineButtons(row).map((item) => item.event_action)).toEqual([
      'detail',
      'update',
      'enable',
    ])
    expect(
      vm.availableLineButtons({ ...row, status: 'enabled' }).map((item) => item.event_action),
    ).toEqual(['detail', 'create_version', 'disable'])
    expect(buttons.line.map((item) => item.event_action)).not.toContain('delete')
    expect(buttons.line.map((item) => item.event_action)).not.toContain('retry_now')
    expect(buttons.line.map((item) => item.event_action)).not.toContain('replay')
  })
})
