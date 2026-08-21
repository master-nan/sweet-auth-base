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
                  placeholder="搜索策略编码或名称"
                  @keyup.enter="handleBasicSearch"
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
              </template>
            </query-scheme-controls>
          </template>
          <template #column-selector>
            <q-select
              v-model="visibleColumns"
              multiple
              outlined
              dense
              options-dense
              emit-value
              map-options
              :display-value="compactSelectionDisplay(visibleColumns, columns, 2, '列')"
              :options="columns"
              option-value="name"
              options-cover
            />
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

      <template #body-cell-policy_code="props">
        <q-td :props="props"
          ><div class="text-weight-bold">{{ props.row.policy_name }}</div>
          <div class="text-caption text-mono text-grey-7">
            {{ props.row.policy_code }} · v{{ props.row.version }}
          </div></q-td
        >
      </template>
      <template #body-cell-status="props"
        ><q-td :props="props"
          ><status-chip
            :color="policyStatusMeta(props.row).color"
            :label="policyStatusMeta(props.row).label" /></q-td
      ></template>
      <template #body-cell-backoff_type="props"
        ><q-td :props="props">{{ policyBackoffLabel(props.row) }}</q-td></template
      >
      <template #body-cell-initial_delay_ms="props"
        ><q-td :props="props">{{ formatDuration(props.row.initial_delay_ms) }}</q-td></template
      >
      <template #body-cell-max_delay_ms="props"
        ><q-td :props="props">{{ formatDuration(props.row.max_delay_ms) }}</q-td></template
      >
      <template #body-cell-retry_window_ms="props"
        ><q-td :props="props">{{ formatDuration(props.row.retry_window_ms) }}</q-td></template
      >
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs no-wrap">
          <q-btn
            v-for="button in availableLineButtons(props.row)"
            :key="button.id"
            flat
            dense
            size="sm"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            @click="handleButtonClick(button, props.row)"
            ><q-tooltip>{{ button.name }}</q-tooltip></q-btn
          >
        </q-td>
      </template>
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="query.page"
          v-model:page-size="query.num"
          :total="total"
      /></template>
      <template #no-data
        ><div class="full-width row flex-center q-pa-xl text-grey-7">
          {{ emptyMessage }}
        </div></template
      >
    </q-table>

    <retry-policy-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :loading="loading"
      @submit="handleFormSubmit"
    />
    <retry-policy-detail-dialog v-model="showDetailDialog" :id="currentDetailId" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_retry_policy' })

import { onMounted, ref, watch, computed } from 'vue'
import { useQuasar } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import RetryPolicyFormDialog, { type RetryPolicyFormValue } from './RetryPolicyFormDialog.vue'
import RetryPolicyDetailDialog from './RetryPolicyDetailDialog.vue'
import {
  type RetryPolicyCreateRequest,
  type RetryPolicyDetail,
  type RetryPolicyListItem,
  type RetryPolicyQuery,
  type RetryPolicyStatus,
  type RetryBackoffType,
  type RetryPolicyUpdateRequest,
  useIntegrationApi,
} from 'src/api/services/integration'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import type { MenuButton } from 'src/api/services/sys-menu'
import type { TableColumn } from 'src/types/global'
import { countEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { compactSelectionDisplay } from 'src/utils/select-display'
import { resolveRuntimeColumns } from 'src/utils/column-format'
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-handlers'
import { resolveTableEmptyMessage } from 'src/utils/table-state'

const $q = useQuasar()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('integration_retry_policy')
const rows = ref<RetryPolicyListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<RetryPolicyDetail | null>(null)
const {
  fields: metadataFields,
  advancedSearchFields: advancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('integration_retry_policy')
const statusMeta: Record<RetryPolicyStatus, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'grey-7' },
  enabled: { label: '已启用', color: 'positive' },
  disabled: { label: '已停用', color: 'warning' },
}
const backoffLabels: Record<RetryBackoffType, string> = {
  fixed: '固定间隔',
  exponential: '指数退避',
}
const columns = ref<TableColumn<RetryPolicyListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const queryState = useTableQueryState<RetryPolicyQuery>({
  createInitialQuery: () => ({
    page: 1,
    num: 15,
    order: { field: '', is_asc: false },
    quick_query: { keyword: '' },
    expressions: emptyExpressions(),
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: true,
    error: loadError.value,
    hasQuery: !!keyword.value || activeFilterCount.value > 0,
  }),
)

const formatDuration = (milliseconds: number) =>
  milliseconds >= 86400000 && milliseconds % 86400000 === 0
    ? `${milliseconds / 86400000} 天`
    : milliseconds >= 60000 && milliseconds % 60000 === 0
      ? `${milliseconds / 60000} 分钟`
      : `${milliseconds / 1000} 秒`
const policyStatusMeta = (row: RetryPolicyListItem) => statusMeta[row.status]
const policyBackoffLabel = (row: RetryPolicyListItem) => backoffLabels[row.backoff_type]
const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.queryRetryPolicies(query.value)
    rows.value = response.data || []
    total.value = response.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '重试策略加载失败'
  } finally {
    loading.value = false
  }
}
const fetchMetadata = async () => {
  if (!(await loadMetadata())) return
  const resolution = resolveRuntimeColumns<RetryPolicyListItem>(metadataFields.value, {
    context: { getDictLabel: () => '' },
    overrides: [
      { fieldCode: 'policy_code', label: '重试策略', order: 1 },
      { fieldCode: 'policy_name', visible: false, order: 2 },
      { fieldCode: 'status', align: 'center', order: 3 },
      { fieldCode: 'max_attempts', align: 'center', order: 4 },
      { fieldCode: 'backoff_type', align: 'center', order: 5 },
      { fieldCode: 'initial_delay_ms', align: 'right', order: 6 },
      { fieldCode: 'max_delay_ms', align: 'right', order: 7 },
      { fieldCode: 'retry_window_ms', align: 'right', order: 8 },
    ],
    virtualColumns: [
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
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('integration_retry_policy', queryState, resetAndFetch)
const handleBasicSearch = () => schemePage.runQueryChange(queryState.submitQuickSearch)
const availableLineButtons = (row: RetryPolicyListItem) =>
  line_buttons.value.filter((button) => {
    if (button.event_action === 'update') return row.status === 'draft'
    if (button.event_action === 'create_version') return row.status !== 'draft'
    if (button.event_action === 'enable') return row.status !== 'enabled'
    if (button.event_action === 'disable') return row.status === 'enabled'
    return true
  })
const openDetail = (row: RetryPolicyListItem) => {
  currentDetailId.value = row.id
  showDetailDialog.value = true
}
const openEdit = async (row: RetryPolicyListItem) => {
  currentEditData.value = (await api.getRetryPolicy(row.id)).data || null
  showFormDialog.value = true
}
const createVersion = (row: RetryPolicyListItem) => {
  confirmAction({
    title: '创建策略版本',
    message: `基于“${row.policy_name}”v${row.version} 创建下一草稿版本？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const response = await api.createRetryPolicyVersion(row.id, row.revision)
      if (response.success) {
        await fetchData()
        if (response.data) {
          currentEditData.value = response.data
          showFormDialog.value = true
        }
      }
    })()
  })
}
const changeState = (row: RetryPolicyListItem, enable: boolean) => {
  confirmAction({
    title: enable ? '确认启用' : '确认停用',
    message: `${enable ? '启用' : '停用'}策略“${row.policy_name}”v${row.version}？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const response = enable
        ? await api.enableRetryPolicy(row.id, row.revision)
        : await api.disableRetryPolicy(row.id, row.revision)
      if (response.success) await fetchData()
    })()
  })
}
const actionHandlers: PageActionHandlers<RetryPolicyListItem> = {
  create: () => {
    currentEditData.value = null
    showFormDialog.value = true
  },
  detail: (row) => row && openDetail(row),
  update: (row) => row && void openEdit(row),
  create_version: (row) => row && createVersion(row),
  enable: (row) => row && changeState(row, true),
  disable: (row) => row && changeState(row, false),
}
const handleButtonClick = (button: MenuButton, row?: RetryPolicyListItem) => {
  dispatchPageAction(button, actionHandlers, row)
}
const handleFormSubmit = async (form: RetryPolicyFormValue) => {
  if (currentEditData.value) {
    const request: RetryPolicyUpdateRequest = { ...form, revision: currentEditData.value.revision }
    if ((await api.updateRetryPolicy(currentEditData.value.id, request)).success)
      showFormDialog.value = false
  } else if ((await api.createRetryPolicy(form as RetryPolicyCreateRequest)).success)
    showFormDialog.value = false
  await fetchData()
}

onMounted(async () => {
  await fetchMetadata()
  await schemePage.initialize()
  await fetchData()
  initialized.value = true
})
watch(
  () => [query.value.page, query.value.num] as const,
  ([page]) => {
    if (!initialized.value) return
    pagination.value.page = page
    void fetchData()
  },
)
watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([sortBy, descending], previous) => {
    if (!initialized.value || (sortBy === previous[0] && descending === previous[1])) return
    if (!queryState.applySorting(sortBy || '', descending, sortableFields.value)) return
    resetAndFetch()
  },
)
</script>
