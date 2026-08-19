import { computed, defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
const api = vi.hoisted(() => ({ querySyncTasks: vi.fn(), queryExternalSystems: vi.fn(), queryInterfaceDefinitions: vi.fn(), listSyncConsumers: vi.fn(), getSyncTaskForEdit: vi.fn(), createSyncTask: vi.fn(), updateSyncTask: vi.fn(), createSyncTaskVersion: vi.fn(), enableSyncTask: vi.fn(), disableSyncTask: vi.fn(), runSyncTask: vi.fn() }))
const tableApi = vi.hoisted(() => ({ queryRuntimeTableByCode: vi.fn() }))
const buttons = vi.hoisted(() => ({ top: [{ id: 1, name: '新增', event_action: 'create' }], line: [{ id: 2, name: '详情', event_action: 'detail' }, { id: 3, name: '编辑', event_action: 'update' }, { id: 4, name: '创建版本', event_action: 'create_version' }, { id: 5, name: '启用', event_action: 'enable' }, { id: 6, name: '停用', event_action: 'disable' }, { id: 7, name: '运行一次', event_action: 'run' }] }))
const permissions = vi.hoisted(() => ({ values: ['integration_sync_task_query', 'integration_sync_task_metadata', 'integration_external_system_query', 'integration_interface_definition_query', 'integration_sync_task_consumer_metadata'] }))
const notify = vi.hoisted(() => vi.fn())
vi.mock('quasar', () => ({ useQuasar: () => ({ screen: { lt: { md: false } }, notify }) })); vi.mock('boot/axios', () => ({ instance: {} })); vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => api })); vi.mock('src/api/services/sys-table', () => ({ useTableApi: () => tableApi })); vi.mock('src/stores/user', () => ({ useUserStore: () => ({ buttons: permissions.values }) })); vi.mock('src/composables/page-buttons', () => ({ usePageButtons: () => ({ top_buttons: computed(() => buttons.top), line_buttons: computed(() => buttons.line), has_line_buttons: computed(() => true), hasGrantedCapability: (code: string) => permissions.values.includes(code) }) })); vi.mock('src/composables/confirm-dialog', () => ({ useConfirmDialog: () => ({ confirmAction: () => ({ onOk: (callback: () => void) => callback() }) }) })); vi.mock('src/components/BaseContent/BaseContent.vue', () => ({ default: { template: '<div><slot /></div>' } })); vi.mock('src/components/Table/TablePagination.vue', () => ({ default: { template: '<div />' } })); vi.mock('src/components/Query/AdvancedQuery.vue', () => ({ default: { template: '<div />' } })); vi.mock('./SyncTaskFormDialog.vue', () => ({ default: { template: '<div />' } })); vi.mock('./SyncTaskDetailDialog.vue', () => ({ default: { template: '<div />' } }))
import Page from './Index.vue'
import type { SyncTaskListItem } from 'src/api/services/integration'
const Slot = defineComponent({ setup(_, { slots }) { return () => h('div', slots.default?.()) } }); const Table = defineComponent({ props: { rows: Array }, setup(props, { slots }) { return () => h('div', { 'data-count': props.rows?.length }, [slots.top?.(), slots.bottom?.()]) } })
const row: SyncTaskListItem = { id: 1, task_code: 'employee_sync', task_name: '人员同步', version: 1, status: 'draft', external_system: { id: 1, code: 'hr', name: 'HR' }, interface_definition: { id: 2, code: 'employee', name: '人员', version: 1 }, consumer: { code: 'org_employee', name: '', version: 1 }, schedule_type: 'none', timezone: 'UTC', checkpoint_mode: 'timestamp', lookback_seconds: 0, window_slice_seconds: 3600, input_plan_summary: { version: 1, static_parameter_count: 0, has_window_bindings: true, window_mode: 'bounded_window', response_bounded: true }, revision: 1, gmt_modify: '' }
describe('sync task page permissions', () => {
  beforeEach(() => { setActivePinia(createPinia()); permissions.values = ['integration_sync_task_query', 'integration_sync_task_metadata', 'integration_external_system_query', 'integration_interface_definition_query', 'integration_sync_task_consumer_metadata']; Object.values(api).forEach((mock) => mock.mockReset()); notify.mockReset(); tableApi.queryRuntimeTableByCode.mockReset(); api.querySyncTasks.mockResolvedValue({ data: [row], total: 1 }); api.queryExternalSystems.mockResolvedValue({ data: [] }); api.queryInterfaceDefinitions.mockResolvedValue({ data: [] }); api.listSyncConsumers.mockResolvedValue({ data: [] }); tableApi.queryRuntimeTableByCode.mockResolvedValue({ data: { table_fields: [] } }) })
  it('loads metadata and only exposes run for enabled tasks', async () => {
    const wrapper = shallowMount(Page, { global: { plugins: [createPinia()], stubs: { BaseContent: Slot, QTable: Table, QInput: true, QSelect: true, QBtn: true, QIcon: true, QChip: true, QTd: Slot, QTooltip: true, QSpace: true, TablePagination: true, AdvancedQuery: true, SyncTaskFormDialog: true, SyncTaskDetailDialog: true } } }); await flushPromises()
    expect(tableApi.queryRuntimeTableByCode).toHaveBeenCalledWith('integration_sync_task'); expect(api.querySyncTasks).toHaveBeenCalled()
    const vm = wrapper.vm as unknown as { availableLineButtons: (value: SyncTaskListItem) => Array<{ event_action: string }> }
    expect(vm.availableLineButtons(row).map((item) => item.event_action)).toEqual(['detail', 'update', 'enable'])
    expect(vm.availableLineButtons({ ...row, status: 'enabled' }).map((item) => item.event_action)).toEqual(['detail', 'create_version', 'disable', 'run'])
    expect(buttons.line.map((item) => item.event_action)).not.toEqual(expect.arrayContaining(['cancel', 'checkpoint']))
  })

  it('creates a manual batch through the controlled run command', async () => {
    api.runSyncTask.mockResolvedValue({ success: true, data: { batch_no: 'SYNC-100' } })
    const wrapper = shallowMount(Page, { global: { plugins: [createPinia()], stubs: { BaseContent: Slot, QTable: Table, QInput: true, QSelect: true, QBtn: true, QIcon: true, QChip: true, QTd: Slot, QTooltip: true, QSpace: true, TablePagination: true, AdvancedQuery: true, SyncTaskFormDialog: true, SyncTaskDetailDialog: true } } }); await flushPromises()
    const vm = wrapper.vm as unknown as { handleButtonClick: (button: { event_action: string }, value: SyncTaskListItem) => void }
    vm.handleButtonClick({ event_action: 'run' }, { ...row, status: 'enabled' })
    await flushPromises()
    expect(api.runSyncTask).toHaveBeenCalledWith(row.id, row.revision)
    expect(notify).toHaveBeenCalledWith(expect.objectContaining({ message: expect.stringContaining('SYNC-100') }))
  })

  it('uses safe runtime metadata without loading protected form dependencies for a query-only account', async () => {
    permissions.values = ['integration_sync_task_query']
    shallowMount(Page, { global: { plugins: [createPinia()], stubs: { BaseContent: Slot, QTable: Table, QInput: true, QSelect: true, QBtn: true, QIcon: true, QChip: true, QTd: Slot, QTooltip: true, QSpace: true, TablePagination: true, AdvancedQuery: true, SyncTaskFormDialog: true, SyncTaskDetailDialog: true } } })
    await flushPromises()
    expect(api.querySyncTasks).toHaveBeenCalled()
    expect(tableApi.queryRuntimeTableByCode).toHaveBeenCalledWith('integration_sync_task')
    expect(api.queryExternalSystems).not.toHaveBeenCalled()
    expect(api.queryInterfaceDefinitions).not.toHaveBeenCalled()
    expect(api.listSyncConsumers).not.toHaveBeenCalled()
  })

  it('does not preload tasks without query permission', async () => {
    permissions.values = []
    shallowMount(Page, { global: { plugins: [createPinia()], stubs: { BaseContent: Slot, QTable: Table, QInput: true, QSelect: true, QBtn: true, QIcon: true, QChip: true, QTd: Slot, QTooltip: true, QSpace: true, TablePagination: true, AdvancedQuery: true, SyncTaskFormDialog: true, SyncTaskDetailDialog: true } } })
    await flushPromises()
    expect(api.querySyncTasks).not.toHaveBeenCalled()
  })
})
