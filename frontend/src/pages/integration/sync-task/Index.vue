<template>
  <base-content class="q-pa-sm">
    <q-table
      v-model:pagination="pagination"
      class="fit sticky-header-table"
      color="primary"
      :dense="$q.screen.lt.md"
      separator="cell"
      flat
      bordered
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      row-key="id"
      :loading="loading"
      :no-data-label="emptyMessage"
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #query-controls>
            <query-scheme-controls
              :controller="schemePage"
              :query-state="queryState"
              :fields="advancedFields"
            >
              <template #quick-search>
                <q-input
                  v-model="keyword"
                  dense
                  outlined
                  debounce="300"
                  :placeholder="quickSearchPlaceholder"
                  @keyup.enter="handleBasicSearch"
                  ><template #append><q-icon name="search" /></template
                ></q-input>
                <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
              </template>
            </query-scheme-controls>
          </template>
          <template #column-selector>
            <table-column-selector v-model="visibleColumns" :columns="columns" />
          </template>
          <template #right-actions>
            <q-btn
              v-for="button in top_buttons"
              :key="button.id"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :disable="loading"
              @click="handleButtonClick(button)"
            />
          </template>
        </standard-table-toolbar>
      </template>
      <template #body-cell-task_code="props"
        ><q-td :props="props"
          ><div class="text-weight-bold">{{ props.row.task_name }}</div>
          <div class="text-caption text-mono text-grey-7">
            {{ props.row.task_code }} · v{{ props.row.version }}
          </div></q-td
        ></template
      >
      <template #body-cell-status="props"
        ><q-td :props="props"
          ><status-chip
            :color="statusFor(props.row).color"
            :label="statusFor(props.row).label" /></q-td
      ></template>
      <template #body-cell-interface="props"
        ><q-td :props="props"
          >{{ props.row.interface_definition.name }}
          <div class="text-caption text-grey-7">
            {{ props.row.interface_definition.code }} · v{{
              props.row.interface_definition.version
            }}
          </div></q-td
        ></template
      >
      <template #body-cell-consumer="props"
        ><q-td :props="props">
          <span class="text-mono">{{ props.row.consumer.code }}@{{ props.row.consumer.version }}</span>
          <status-chip
            v-if="consumerMetadataLoaded && !isConsumerAvailable(props.row)"
            class="q-ml-sm"
            color="negative"
            label="当前服务未开放"
          >
            <q-tooltip>后端没有注册这个 Consumer，任务暂时不能启用或运行</q-tooltip>
          </status-chip>
        </q-td
        ></template
      >
      <template #body-cell-schedule="props"
        ><q-td :props="props"
          >{{ props.row.schedule_type === 'cron' ? props.row.cron_summary : '仅手工' }}
          <div class="text-caption text-grey-7">{{ props.row.timezone }}</div></q-td
        ></template
      >
      <template #body-cell-checkpoint="props"
        ><q-td :props="props">{{
          props.row.checkpoint_mode === 'timestamp'
            ? props.row.checkpoint_at
              ? formatRuntimeDateTime(props.row.checkpoint_at)
              : '待首次启用'
            : '无'
        }}</q-td></template
      >
      <template #body-cell-actions="props"
        ><q-td :props="props" class="q-gutter-xs no-wrap"
          ><q-btn
            v-for="button in availableLineButtons(props.row)"
            :key="button.id"
            flat
            dense
            size="sm"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            :disable="isLineButtonDisabled(button, props.row)"
            @click="handleButtonClick(button, props.row)"
            ><q-tooltip>{{ lineButtonTooltip(button, props.row) }}</q-tooltip></q-btn
          ></q-td
        ></template
      >
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="query.page"
          v-model:page-size="query.num"
          :total="total"
      /></template>
    </q-table>
    <sync-task-form-dialog
      v-model="showForm"
      :edit-data="editData"
      :systems="systems"
      :interfaces="interfaces"
      :consumers="consumers"
      :loading="loading"
      @submit="handleSubmit"
    />
    <sync-task-detail-dialog v-model="showDetail" :id="detailID" />
  </base-content>
</template>
<script setup lang="ts">
defineOptions({ name: 'integration_sync_task' })
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from 'src/components/Table/TableColumnSelector.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import SyncTaskFormDialog, { type SyncTaskFormValue } from './SyncTaskFormDialog.vue'
import SyncTaskDetailDialog from './SyncTaskDetailDialog.vue'
import {
  type ExternalSystemListItem,
  type InterfaceDefinitionListItem,
  type SyncConsumerMetadata,
  type SyncTaskEdit,
  type SyncTaskListItem,
  type SyncTaskQuery,
  type SyncTaskUpdateRequest,
  useIntegrationApi,
} from 'src/api/services/integration'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import type { MenuButton } from 'src/api/services/sys-menu'
import type { TableColumn } from 'src/types/global'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-handlers'
import { resolveRuntimeColumns } from 'src/utils/column-format'
import { resolveTableEmptyMessage } from 'src/utils/table-state'
import { countEffectiveQueryRules } from 'src/utils/query-state'
import { formatRuntimeDateTime } from 'src/pages/integration/runtime-display'

const $q = useQuasar()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons, hasGrantedCapability } =
  usePageButtons('integration_sync_task')
const rows = ref<SyncTaskListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showForm = ref(false)
const showDetail = ref(false)
const detailID = ref(0)
const editData = ref<SyncTaskEdit | null>(null)
const systems = ref<ExternalSystemListItem[]>([])
const interfaces = ref<InterfaceDefinitionListItem[]>([])
const consumers = ref<SyncConsumerMetadata[]>([])
const consumerMetadataLoaded = ref(false)
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: advancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('integration_sync_task')
const canQueryTasks = computed(() => hasGrantedCapability('integration_sync_task_query'))
const statusMeta = {
  draft: { label: '草稿', color: 'grey-7' },
  enabled: { label: '已启用', color: 'positive' },
  disabled: { label: '已停用', color: 'warning' },
}
const statusFor = (row: SyncTaskListItem) => statusMeta[row.status]
const columns = ref<TableColumn<SyncTaskListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const queryState = useTableQueryState<SyncTaskQuery>({
  createInitialQuery: () => ({
    page: 1,
    num: 20,
    order: { field: '', is_asc: false },
    quick_query: { keyword: '' },
    expressions: emptyExpressions(),
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryTasks.value,
    error: loadError.value,
    hasQuery: !!keyword.value || activeFilterCount.value > 0,
  }),
)
const fetchData = async () => {
  if (!canQueryTasks.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await api.querySyncTasks(query.value)
    rows.value = result.data || []
    total.value = result.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '同步任务加载失败'
  } finally {
    loading.value = false
  }
}
const fetchMetadata = async () => {
  if (!(await loadMetadata())) return
  const resolution = resolveRuntimeColumns<SyncTaskListItem>(metadataFields.value, {
    context: { getDictLabel: () => '' },
    overrides: [
      { fieldCode: 'task_name', visible: false },
      { fieldCode: 'interface_definition_id', visible: false },
      { fieldCode: 'consumer_code', visible: false },
      { fieldCode: 'consumer_version', visible: false },
      { fieldCode: 'schedule_type', visible: false },
      { fieldCode: 'checkpoint_at', visible: false },
      { fieldCode: 'task_code', label: '同步任务', order: 1 },
      { fieldCode: 'status', align: 'center', order: 2 },
      { fieldCode: 'window_slice_seconds', label: '切片（秒）', align: 'right', order: 7 },
    ],
    virtualColumns: [
      { name: 'interface', label: '接口版本', field: 'interface_definition', order: 3 },
      { name: 'consumer', label: 'Consumer', field: 'consumer', order: 4 },
      {
        name: 'schedule',
        label: '调度',
        field: 'schedule_type',
        order: 5,
        serverSortField: 'schedule_type',
      },
      {
        name: 'checkpoint',
        label: 'Checkpoint',
        field: 'checkpoint_at',
        order: 6,
        serverSortField: 'checkpoint_at',
      },
      {
        name: 'actions',
        label: '操作',
        field: 'actions',
        align: 'center',
        order: 100,
        defaultVisible: has_line_buttons.value,
      },
    ],
  })
  columns.value = resolution.columns
  visibleColumns.value = resolution.visibleColumns
  sortableFields.value = resolution.sortableFields
}
const fetchReferences = async () => {
  const requests: Promise<void>[] = []
  if (hasGrantedCapability('integration_external_system_query'))
    requests.push(
      api
        .queryExternalSystems({
          page: 1,
          num: 500,
          order: { field: 'name', is_asc: true },
          expressions: emptyExpressions(),
          status: 'enabled',
        })
        .then((result) => {
          systems.value = result.data || []
        }),
    )
  if (hasGrantedCapability('integration_interface_definition_query'))
    requests.push(
      api
        .queryInterfaceDefinitions({
          page: 1,
          num: 500,
          order: { field: 'interface_code', is_asc: true },
          expressions: emptyExpressions(),
          status: 'enabled',
        })
        .then((result) => {
          interfaces.value = result.data || []
        }),
    )
  if (hasGrantedCapability('integration_sync_task_consumer_metadata'))
    requests.push(
      api.listSyncConsumers().then((result) => {
        consumers.value = result.data || []
        consumerMetadataLoaded.value = true
      }),
    )
  await Promise.all(requests)
}
const isConsumerAvailable = (row: SyncTaskListItem) =>
  consumers.value.some(
    (consumer) => consumer.code === row.consumer.code && consumer.version === row.consumer.version,
  )
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('integration_sync_task', queryState, resetAndFetch)
const handleBasicSearch = () => schemePage.runQueryChange(queryState.submitQuickSearch)
const availableLineButtons = (row: SyncTaskListItem) =>
  line_buttons.value.filter((button) =>
    button.event_action === 'update'
      ? row.status === 'draft'
      : button.event_action === 'create_version'
        ? row.status !== 'draft'
        : button.event_action === 'enable'
          ? row.status !== 'enabled'
          : button.event_action === 'disable' || button.event_action === 'run'
            ? row.status === 'enabled'
            : true,
  )
const isLineButtonDisabled = (button: MenuButton, row: SyncTaskListItem) =>
  consumerMetadataLoaded.value &&
  !isConsumerAvailable(row) &&
  (button.event_action === 'enable' || button.event_action === 'run')
const lineButtonTooltip = (button: MenuButton, row: SyncTaskListItem) =>
  isLineButtonDisabled(button, row)
    ? `${button.name}不可用：当前服务未注册该 Consumer`
    : button.name
const openCreate = async () => {
  await fetchReferences()
  editData.value = null
  showForm.value = true
}
const openEdit = async (row: SyncTaskListItem) => {
  await fetchReferences()
  editData.value = (await api.getSyncTaskForEdit(row.id)).data || null
  showForm.value = true
}
const createVersion = (row: SyncTaskListItem) =>
  confirmAction({
    title: '创建任务版本',
    message: `基于“${row.task_name}”v${row.version} 创建下一草稿版本？`,
  }).onOk(() => {
    void (async () => {
      const result = await api.createSyncTaskVersion(row.id, row.revision)
      if (result.success && result.data) {
        await fetchReferences()
        editData.value = await api
          .getSyncTaskForEdit(result.data.id)
          .then((item) => item.data || null)
        showForm.value = true
        await fetchData()
      }
    })()
  })
const changeState = (row: SyncTaskListItem, enable: boolean) => {
  if (enable && consumerMetadataLoaded.value && !isConsumerAvailable(row)) {
    $q.notify({ type: 'warning', message: '当前服务未开放该同步 Consumer，不能启用任务' })
    return
  }
  confirmAction({
    title: enable ? '确认启用' : '确认停用',
    message: `${enable ? '启用' : '停用'}任务“${row.task_name}”v${row.version}？`,
  }).onOk(() => {
    void (async () => {
      const result = enable
        ? await api.enableSyncTask(row.id, row.revision)
        : await api.disableSyncTask(row.id, row.revision)
      if (result.success) await fetchData()
    })()
  })
}
const runTask = (row: SyncTaskListItem) => {
  if (consumerMetadataLoaded.value && !isConsumerAvailable(row)) {
    $q.notify({ type: 'warning', message: '当前服务未开放该同步 Consumer，不能运行任务' })
    return
  }
  confirmAction({
    title: '运行一次',
    message: `按当前 Checkpoint 运行任务“${row.task_name}”v${row.version}？`,
  }).onOk(() => {
    void (async () => {
      const result = await api.runSyncTask(row.id, row.revision)
      if (result.success && result.data) {
        $q.notify({ type: 'positive', message: `已创建同步批次 ${result.data.batch_no}` })
        await fetchData()
      }
    })()
  })
}
const handlers: PageActionHandlers<SyncTaskListItem> = {
  create: () => {
    void openCreate()
  },
  detail: (row) => {
    if (row) {
      detailID.value = row.id
      showDetail.value = true
    }
  },
  update: (row) => row && void openEdit(row),
  create_version: (row) => row && createVersion(row),
  enable: (row) => row && changeState(row, true),
  disable: (row) => row && changeState(row, false),
  run: (row) => row && runTask(row),
}
const handleButtonClick = (button: MenuButton, row?: SyncTaskListItem) => {
  dispatchPageAction(button, handlers, row)
}
const handleSubmit = async (value: SyncTaskFormValue) => {
  const result = editData.value
    ? await api.updateSyncTask(editData.value.id, {
        ...value,
        revision: editData.value.revision,
      } as SyncTaskUpdateRequest)
    : await api.createSyncTask(value)
  if (result.success) {
    showForm.value = false
    await fetchData()
  }
}
onMounted(async () => {
  await fetchMetadata()
  if (hasGrantedCapability('integration_sync_task_consumer_metadata')) await fetchReferences()
  await schemePage.initialize()
  await fetchData()
  initialized.value = true
})
watch(
  () => [query.value.page, query.value.num] as const,
  () => {
    if (initialized.value) void fetchData()
  },
)
watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([field, descending]) => {
    if (
      !initialized.value ||
      !queryState.applySorting(field || '', descending, sortableFields.value)
    )
      return
    resetAndFetch()
  },
)
</script>
