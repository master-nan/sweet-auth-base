<template>
  <base-content class="q-pa-sm">
    <q-table v-model:pagination="pagination" class="fit sticky-header-table" color="primary" :dense="$q.screen.lt.md" separator="cell" flat bordered :rows="rows" :columns="columns" :visible-columns="visibleColumns" row-key="id" :loading="loading">
      <template #top>
        <div class="row q-gutter-xs full-width items-center">
          <q-input v-model="query.quick_query!.keyword" dense outlined debounce="300" placeholder="搜索任务编码或名称" @keyup.enter="resetAndFetch"><template #append><q-icon name="search" /></template></q-input>
          <q-select v-model="query.status" dense outlined emit-value map-options clearable :options="statusOptions" label="状态" style="min-width: 140px" @update:model-value="resetAndFetch" />
          <q-select v-model="query.schedule_type" dense outlined emit-value map-options clearable :options="scheduleOptions" label="调度方式" style="min-width: 150px" @update:model-value="resetAndFetch" />
          <q-btn color="primary" label="搜索" :disable="loading" @click="resetAndFetch" />
          <q-btn v-if="canUseAdvancedQuery" outline icon="tune" color="primary" aria-label="高级查询" @click="showAdvancedQuery = true"><q-tooltip>高级查询</q-tooltip></q-btn>
          <q-space />
          <q-btn v-for="button in top_buttons" :key="button.id" v-bind="menuButtonDisplayProps(button)" :color="button.color || 'primary'" :disable="loading" @click="handleButtonClick(button)" />
        </div>
      </template>
      <template #body-cell-task_code="props"><q-td :props="props"><div class="text-weight-bold">{{ props.row.task_name }}</div><div class="text-caption text-mono text-grey-7">{{ props.row.task_code }} · v{{ props.row.version }}</div></q-td></template>
      <template #body-cell-status="props"><q-td :props="props"><q-chip dense square outline :color="statusFor(props.row).color" :label="statusFor(props.row).label" /></q-td></template>
      <template #body-cell-interface="props"><q-td :props="props">{{ props.row.interface_definition.name }}<div class="text-caption text-grey-7">{{ props.row.interface_definition.code }} · v{{ props.row.interface_definition.version }}</div></q-td></template>
      <template #body-cell-consumer="props"><q-td :props="props"><span class="text-mono">{{ props.row.consumer.code }}@{{ props.row.consumer.version }}</span></q-td></template>
      <template #body-cell-schedule="props"><q-td :props="props">{{ props.row.schedule_type === 'cron' ? props.row.cron_summary : '仅手工' }}<div class="text-caption text-grey-7">{{ props.row.timezone }}</div></q-td></template>
      <template #body-cell-checkpoint="props"><q-td :props="props">{{ props.row.checkpoint_mode === 'timestamp' ? props.row.checkpoint_at || '待首次启用' : '无' }}</q-td></template>
      <template #body-cell-actions="props"><q-td :props="props" class="q-gutter-xs no-wrap"><q-btn v-for="button in availableLineButtons(props.row)" :key="button.id" flat dense size="sm" v-bind="menuButtonDisplayProps(button)" :color="button.color || 'primary'" @click="handleButtonClick(button, props.row)"><q-tooltip>{{ button.name }}</q-tooltip></q-btn></q-td></template>
      <template #bottom><q-space /><table-pagination v-model:page="query.page" v-model:page-size="query.num" :total="total" /></template>
    </q-table>
    <advanced-query v-model="showAdvancedQuery" v-model:query-model="tempAdvancedQuery" :fields="advancedFields" @search="handleAdvancedSearch" />
    <sync-task-form-dialog v-model="showForm" :edit-data="editData" :systems="systems" :interfaces="interfaces" :consumers="consumers" :loading="loading" @submit="handleSubmit" />
    <sync-task-detail-dialog v-model="showDetail" :id="detailID" />
  </base-content>
</template>
<script setup lang="ts">
defineOptions({ name: 'integration_sync_task' })
import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import SyncTaskFormDialog, { type SyncTaskFormValue } from './SyncTaskFormDialog.vue'
import SyncTaskDetailDialog from './SyncTaskDetailDialog.vue'
import { type ExternalSystemListItem, type InterfaceDefinitionListItem, type SyncConsumerMetadata, type SyncTaskEdit, type SyncTaskListItem, type SyncTaskQuery, type SyncTaskUpdateRequest, useIntegrationApi } from 'src/api/services/integration'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useLoadingStore } from 'src/stores/loading'
import { useUserStore } from 'src/stores/user'
import { storeToRefs } from 'pinia'
import type { MenuButton } from 'src/api/services/sys-menu'
import type { Query } from 'src/types/global'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const $q = useQuasar(); const api = useIntegrationApi(); const tableApi = useTableApi(); const userStore = useUserStore(); const { loading } = storeToRefs(useLoadingStore()); const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('integration_sync_task')
const rows = ref<SyncTaskListItem[]>([]); const total = ref(0); const initialized = ref(false); const showAdvancedQuery = ref(false); const showForm = ref(false); const showDetail = ref(false); const detailID = ref(0); const editData = ref<SyncTaskEdit | null>(null)
const systems = ref<ExternalSystemListItem[]>([]); const interfaces = ref<InterfaceDefinitionListItem[]>([]); const consumers = ref<SyncConsumerMetadata[]>([]); const advancedFields = ref<TableField[]>([])
const canQueryTasks = computed(() => userStore.buttons.includes('integration_sync_task_query'))
const canUseAdvancedQuery = computed(() => userStore.buttons.includes('integration_sync_task_metadata'))
const statusMeta = { draft: { label: '草稿', color: 'grey-7' }, enabled: { label: '已启用', color: 'positive' }, disabled: { label: '已停用', color: 'warning' } }
const statusFor = (row: SyncTaskListItem) => statusMeta[row.status]
const statusOptions = Object.entries(statusMeta).map(([value, item]) => ({ label: item.label, value })); const scheduleOptions = [{ label: '仅手工', value: 'none' }, { label: 'Cron', value: 'cron' }]
const columns: QTableProps['columns'] = [
  { name: 'task_code', label: '同步任务', field: 'task_code', align: 'left', sortable: true }, { name: 'status', label: '状态', field: 'status', align: 'center', sortable: true },
  { name: 'interface', label: '接口版本', field: 'interface_definition', align: 'left' }, { name: 'consumer', label: 'Consumer', field: 'consumer', align: 'left' },
  { name: 'schedule', label: '调度', field: 'schedule_type', align: 'left', sortable: true }, { name: 'checkpoint', label: 'Checkpoint', field: 'checkpoint_at', align: 'left', sortable: true },
  { name: 'window_slice_seconds', label: '切片（秒）', field: 'window_slice_seconds', align: 'right', sortable: true }, { name: 'gmt_modify', label: '更新时间', field: 'gmt_modify', align: 'left', sortable: true }, { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]
const visibleColumns = ref(columns.map((item) => item.name)); const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const query = ref<SyncTaskQuery>({ page: 1, num: 15, order: { field: '', is_asc: false }, quick_query: { keyword: '' }, expressions: emptyExpressions() }); const tempAdvancedQuery = ref<Query>(cloneDeep(query.value)); const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const fetchData = async () => { if (!canQueryTasks.value) return; const result = await api.querySyncTasks(query.value); rows.value = result.data || []; total.value = result.total || 0 }
const fetchMetadata = async () => { if (!userStore.buttons.includes('integration_sync_task_metadata')) return; const metadata = await tableApi.queryTableByCode('integration_sync_task'); advancedFields.value = metadata.data?.table_fields || [] }
const fetchReferences = async () => {
  const requests: Promise<void>[] = []
  if (userStore.buttons.includes('integration_external_system_query')) requests.push(api.queryExternalSystems({ page: 1, num: 500, order: { field: 'name', is_asc: true }, expressions: emptyExpressions(), status: 'enabled' }).then((result) => { systems.value = result.data || [] }))
  if (userStore.buttons.includes('integration_interface_definition_query')) requests.push(api.queryInterfaceDefinitions({ page: 1, num: 500, order: { field: 'interface_code', is_asc: true }, expressions: emptyExpressions(), status: 'enabled' }).then((result) => { interfaces.value = result.data || [] }))
  if (userStore.buttons.includes('integration_sync_task_consumer_metadata')) requests.push(api.listSyncConsumers().then((result) => { consumers.value = result.data || [] }))
  await Promise.all(requests)
}
const resetAndFetch = () => { if (query.value.page !== 1) query.value.page = 1; else void fetchData() }
const handleAdvancedSearch = () => { query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions); showAdvancedQuery.value = false; resetAndFetch() }
const availableLineButtons = (row: SyncTaskListItem) => line_buttons.value.filter((button) => button.event_action === 'update' ? row.status === 'draft' : button.event_action === 'create_version' ? row.status !== 'draft' : button.event_action === 'enable' ? row.status !== 'enabled' : button.event_action === 'disable' || button.event_action === 'run' ? row.status === 'enabled' : true)
const openCreate = async () => { await fetchReferences(); editData.value = null; showForm.value = true }
const openEdit = async (row: SyncTaskListItem) => { await fetchReferences(); editData.value = (await api.getSyncTaskForEdit(row.id)).data || null; showForm.value = true }
const createVersion = (row: SyncTaskListItem) => confirmAction({ title: '创建任务版本', message: `基于“${row.task_name}”v${row.version} 创建下一草稿版本？` }).onOk(() => { void (async () => { const result = await api.createSyncTaskVersion(row.id, row.revision); if (result.success && result.data) { await fetchReferences(); editData.value = await api.getSyncTaskForEdit(result.data.id).then((item) => item.data || null); showForm.value = true; await fetchData() } })() })
const changeState = (row: SyncTaskListItem, enable: boolean) => confirmAction({ title: enable ? '确认启用' : '确认停用', message: `${enable ? '启用' : '停用'}任务“${row.task_name}”v${row.version}？` }).onOk(() => { void (async () => { const result = enable ? await api.enableSyncTask(row.id, row.revision) : await api.disableSyncTask(row.id, row.revision); if (result.success) await fetchData() })() })
const runTask = (row: SyncTaskListItem) => confirmAction({ title: '运行一次', message: `按当前 Checkpoint 运行任务“${row.task_name}”v${row.version}？` }).onOk(() => { void (async () => { const result = await api.runSyncTask(row.id, row.revision); if (result.success && result.data) { $q.notify({ type: 'positive', message: `已创建同步批次 ${result.data.batch_no}` }); await fetchData() } })() })
const handlers: Record<string, (row?: SyncTaskListItem) => void> = { create: () => { void openCreate() }, detail: (row) => { if (row) { detailID.value = row.id; showDetail.value = true } }, update: (row) => row && void openEdit(row), create_version: (row) => row && createVersion(row), enable: (row) => row && changeState(row, true), disable: (row) => row && changeState(row, false), run: (row) => row && runTask(row) }
const handleButtonClick = (button: MenuButton, row?: SyncTaskListItem) => handlers[button.event_action]?.(row)
const handleSubmit = async (value: SyncTaskFormValue) => { const result = editData.value ? await api.updateSyncTask(editData.value.id, { ...value, revision: editData.value.revision } as SyncTaskUpdateRequest) : await api.createSyncTask(value); if (result.success) { showForm.value = false; await fetchData() } }
onMounted(async () => { await Promise.all([fetchMetadata(), fetchData()]); if (!has_line_buttons.value) visibleColumns.value = visibleColumns.value.filter((item) => item !== 'actions'); initialized.value = true })
watch(() => [query.value.page, query.value.num] as const, () => { if (initialized.value) void fetchData() }); watch(() => [pagination.value.sortBy, pagination.value.descending] as const, ([field, descending]) => { if (!initialized.value) return; query.value.order = { field: field || '', is_asc: field ? !descending : false }; resetAndFetch() }); watch(showAdvancedQuery, (open) => { if (open) tempAdvancedQuery.value = cloneDeep(query.value) })
</script>
