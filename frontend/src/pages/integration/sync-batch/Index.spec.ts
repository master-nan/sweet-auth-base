import { defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  querySyncBatches: vi.fn(),
  getSyncBatch: vi.fn(),
  queryExecutions: vi.fn(),
}))
const permissions = vi.hoisted(() => ({
  values: ['integration_sync_batch_query', 'integration_sync_batch_detail'],
}))
const tableApi = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
const schemeMocks = vi.hoisted(() => ({ initialize: vi.fn() }))
vi.mock('quasar', () => ({ useQuasar: () => ({ screen: { lt: { md: false } } }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/boot/axios', () => ({ instance: {} }))
vi.mock('@/api/services/integration', () => ({ useIntegrationApi: () => api }))
vi.mock('@/api/services/sys-table', () => ({ useTableApi: () => tableApi }))
vi.mock('@/composables/query-scheme-page', async () => {
  const { createQuerySchemePageStub } = await import('@/test/query-scheme-page-stub')
  return {
    useQuerySchemePage: () => createQuerySchemePageStub(schemeMocks.initialize),
  }
})
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ buttons: permissions.values }) }))
vi.mock('@/stores/loading', () => ({ useLoadingStore: () => ({ loading: false }) }))
vi.mock('@/components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))
vi.mock('@/components/Table/TablePagination.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/components/FormDialog/FormDialogShell.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))
import Page from './Index.vue'

const Slot = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const Table = defineComponent({
  props: { rows: Array },
  setup(props, { slots }) {
    return () => h('div', { 'data-count': props.rows?.length }, [slots.top?.(), slots.bottom?.()])
  },
})

describe('sync batch query page', () => {
  beforeEach(() => {
    permissions.values = ['integration_sync_batch_query', 'integration_sync_batch_detail']
    Object.values(api).forEach((mock) => mock.mockReset())
    schemeMocks.initialize.mockReset()
    schemeMocks.initialize.mockResolvedValue(false)
    tableApi.queryRuntimeTableByCode.mockReset()
    tableApi.queryRuntimeTableByCode.mockResolvedValue({
      success: true,
      data: {
        table_fields: [
          {
            field_code: 'status',
            field_name: '状态',
            field_type: 'varchar',
            is_advanced_search: true,
            sequence: 1,
          },
        ],
      },
    })
  })
  it('does not preload batch data without query permission', async () => {
    permissions.values = ['integration_sync_batch_detail']
    shallowMount(Page, {
      global: {
        stubs: {
          BaseContent: Slot,
          QTable: Table,
          QInput: true,
          QSelect: true,
          QBtn: true,
          QIcon: true,
          QChip: true,
          QTd: Slot,
          QSpace: true,
          TablePagination: true,
          FormDialogShell: true,
        },
      },
    })
    await flushPromises()
    expect(api.querySyncBatches).not.toHaveBeenCalled()
  })
  it('exposes no batch mutation command and does not request executions without permission', async () => {
    api.querySyncBatches.mockResolvedValue({ data: [], total: 0 })
    const wrapper = shallowMount(Page, {
      global: {
        stubs: {
          BaseContent: Slot,
          QTable: Table,
          QInput: true,
          QSelect: true,
          QBtn: true,
          QIcon: true,
          QChip: true,
          QTd: Slot,
          QSpace: true,
          TablePagination: true,
          FormDialogShell: true,
        },
      },
    })
    await flushPromises()
    expect(api.querySyncBatches).toHaveBeenCalledOnce()
    expect(tableApi.queryRuntimeTableByCode).toHaveBeenCalledWith('integration_sync_batch')
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
    expect(schemeMocks.initialize.mock.invocationCallOrder[0]).toBeLessThan(
      api.querySyncBatches.mock.invocationCallOrder[0]!,
    )
    expect(wrapper.text()).not.toMatch(/取消批次|修改 Checkpoint|补数|Dry Run|重新运行/)
    const vm = wrapper.vm as unknown as { openDetail: (id: number) => Promise<void> }
    api.getSyncBatch.mockResolvedValue({
      data: {
        id: 1,
        batch_no: 'SYNC-1',
        task_code: 'task',
        task_version: 1,
        trigger_type: 'manual',
        status: 'created',
        interface_code: 'employees',
        interface_version: 1,
        consumer_code: 'test',
        consumer_version: 1,
        checkpoint_mode: 'timestamp',
        planned_slice_count: 2,
        current_slice_no: 0,
        execution_count: 0,
        technical_success_count: 0,
        technical_failed_count: 0,
        business_success_count: 0,
        business_failed_count: 0,
      },
    })
    await vm.openDetail(1)
    expect(api.queryExecutions).not.toHaveBeenCalled()
    await (wrapper.vm as unknown as { fetchData: () => Promise<void> }).fetchData()
    expect(schemeMocks.initialize).toHaveBeenCalledOnce()
  })

  it('loads batch executions through the protected execution query', async () => {
    permissions.values = [
      'integration_sync_batch_query',
      'integration_sync_batch_detail',
      'integration_execution_query',
      'integration_execution_detail',
    ]
    api.querySyncBatches.mockResolvedValue({ data: [], total: 0 })
    api.getSyncBatch.mockResolvedValue({
      data: {
        id: 1,
        batch_no: 'SYNC-1',
        task_code: 'task',
        task_version: 1,
        trigger_type: 'manual',
        status: 'running',
        interface_code: 'employees',
        interface_version: 1,
        consumer_code: 'test',
        consumer_version: 1,
        checkpoint_mode: 'timestamp',
        planned_slice_count: 2,
        current_slice_no: 1,
        execution_count: 1,
        technical_success_count: 0,
        technical_failed_count: 0,
        business_success_count: 0,
        business_failed_count: 0,
      },
    })
    api.queryExecutions.mockResolvedValue({
      data: [
        {
          id: 9,
          execution_no: 'EX-9',
          status: 'running',
          sync_source: { batch_id: 1, slice_no: 1 },
        },
      ],
      total: 1,
    })
    const wrapper = shallowMount(Page, {
      global: {
        stubs: {
          BaseContent: Slot,
          QTable: Table,
          QInput: true,
          QSelect: true,
          QBtn: true,
          QIcon: true,
          QChip: true,
          QTd: Slot,
          QSpace: true,
          QSeparator: true,
          QList: Slot,
          QItem: Slot,
          QItemSection: Slot,
          QItemLabel: Slot,
          TablePagination: true,
          FormDialogShell: true,
        },
      },
    })
    await flushPromises()
    const vm = wrapper.vm as unknown as { openDetail: (id: number) => Promise<void> }
    await vm.openDetail(1)
    expect(api.queryExecutions).toHaveBeenCalledWith(expect.objectContaining({ sync_batch_id: 1 }))
  })
})
