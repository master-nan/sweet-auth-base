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
                  :placeholder="quickSearchPlaceholder"
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

      <template #body-cell-external_system="props">
        <q-td :props="props"
          ><div class="text-weight-bold">{{ props.row.external_system.name }}</div>
          <div class="text-caption text-grey-7">
            {{ props.row.external_system.system_code }}
          </div></q-td
        >
      </template>
      <template #body-cell-interface_code="props">
        <q-td :props="props"
          ><div class="text-weight-bold">{{ props.row.name }}</div>
          <div class="text-caption text-mono text-grey-7">
            {{ props.row.interface_code }} · v{{ props.row.version }}
          </div></q-td
        >
      </template>
      <template #body-cell-http_method="props"
        ><q-td :props="props"
          ><q-chip
            dense
            square
            color="primary"
            text-color="white"
            :label="props.row.http_method" /></q-td
      ></template>
      <template #body-cell-path_summary="props"
        ><q-td :props="props"
          ><span class="text-mono">{{ props.row.path_summary }}</span></q-td
        ></template
      >
      <template #body-cell-status="props"
        ><q-td :props="props"
          ><status-chip
            :color="statusMeta[props.row.effective_status]?.color || 'grey'"
            :label="
              statusMeta[props.row.effective_status]?.label || props.row.effective_status
            " /></q-td
      ></template>
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
      <template #no-data
        ><div class="full-width row flex-center q-pa-xl text-grey-7">
          {{ emptyMessage }}
        </div></template
      >
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="query.page"
          v-model:page-size="query.num"
          :total="total"
      /></template>
    </q-table>

    <interface-definition-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :systems="systems"
      :credentials="credentials"
      :retry-policies="retryPolicies"
      :loading="loading"
      @submit="handleFormSubmit"
    />
    <interface-definition-detail-dialog v-model="showDetailDialog" :id="currentDetailId" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_interface_definition' })

import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import InterfaceDefinitionFormDialog from './InterfaceDefinitionFormDialog.vue'
import InterfaceDefinitionDetailDialog from './InterfaceDefinitionDetailDialog.vue'
import {
  type ExternalSystemListItem,
  type CredentialListItem,
  type InterfaceDefinitionCreateRequest,
  type InterfaceDefinitionDetail,
  type InterfaceDefinitionListItem,
  type InterfaceDefinitionQuery,
  type InterfaceDefinitionUpdateRequest,
  type RetryPolicyListItem,
  useIntegrationApi,
} from 'src/api/services/integration'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import type { MenuButton } from 'src/api/services/sys-menu'
import type { TableColumn } from 'src/types/global'
import { ExpressionType } from 'src/types/enum'
import { countEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { compactSelectionDisplay } from 'src/utils/select-display'
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-handlers'
import { resolveRuntimeColumns } from 'src/utils/column-format'
import { resolveTableEmptyMessage } from 'src/utils/table-state'

const $q = useQuasar()
const route = useRoute()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons, hasGrantedCapability } = usePageButtons(
  'integration_interface_definition',
)
const rows = ref<InterfaceDefinitionListItem[]>([])
const systems = ref<ExternalSystemListItem[]>([])
const credentials = ref<CredentialListItem[]>([])
const retryPolicies = ref<RetryPolicyListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<InterfaceDefinitionDetail | null>(null)
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: metadataAdvancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('integration_interface_definition')
const advancedFields = computed(() => metadataAdvancedFields.value)
const statusMeta: Record<string, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'grey-7' },
  enabled: { label: '已启用', color: 'positive' },
  disabled: { label: '已停用', color: 'warning' },
  unavailable: { label: '当前不可用', color: 'negative' },
}
const columns = ref<TableColumn<InterfaceDefinitionListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const routeSystemID = Number(route.query.external_system_id)
const hasRouteSystemContext = Number.isSafeInteger(routeSystemID) && routeSystemID > 0
const queryState = useTableQueryState<InterfaceDefinitionQuery>({
  createInitialQuery: () => ({
    page: 1,
    num: 15,
    order: { field: '', is_asc: false },
    quick_query: { keyword: '' },
    expressions: hasRouteSystemContext
      ? [
          {
            rules: [
              {
                field: 'external_system_id',
                expression_type: ExpressionType.EQ,
                value: routeSystemID,
              },
            ],
            nested: [],
          },
        ]
      : emptyExpressions(),
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: true,
    error: loadError.value,
    hasQuery: !!keyword.value || activeFilterCount.value > 0,
  }),
)
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })

const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.queryInterfaceDefinitions(query.value)
    rows.value = response.data || []
    total.value = response.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '接口定义加载失败'
  } finally {
    loading.value = false
  }
}
const fetchSystems = async () => {
  const response = await api.queryExternalSystems({
    page: 1,
    num: 500,
    order: { field: 'name', is_asc: true },
    quick_query: { keyword: '' },
    expressions: [],
  })
  systems.value = response.data || []
}
const fetchCredentials = async () => {
  const response = await api.queryCredentials({
    page: 1,
    num: 500,
    order: { field: 'credential_code', is_asc: true },
    quick_query: { keyword: '' },
    expressions: [],
  })
  credentials.value = response.data || []
}
const fetchRetryPolicies = async () => {
  const response = await api.queryRetryPolicies({
    page: 1,
    num: 500,
    order: { field: 'policy_code', is_asc: true },
    quick_query: { keyword: '' },
    expressions: [],
    status: 'enabled',
  })
  retryPolicies.value = response.data || []
}
const fetchMetadata = async () => {
  if (!(await loadMetadata())) return
  const resolution = resolveRuntimeColumns<InterfaceDefinitionListItem>(metadataFields.value, {
    context: { getDictLabel: () => '' },
    overrides: [
      { fieldCode: 'external_system_id', visible: false },
      { fieldCode: 'name', visible: false },
      { fieldCode: 'relative_path', visible: false },
      { fieldCode: 'interface_code', label: '接口定义', order: 2 },
      { fieldCode: 'http_method', label: 'Method', align: 'center', order: 3 },
      { fieldCode: 'protocol', align: 'center', order: 5 },
      { fieldCode: 'status', align: 'center', order: 6 },
    ],
    virtualColumns: [
      {
        name: 'external_system',
        label: '所属系统',
        field: (row) => row.external_system.name,
        order: 1,
      },
      { name: 'path_summary', label: '相对路径', field: 'path_summary', order: 4 },
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
const schemePage = useQuerySchemePage('integration_interface_definition', queryState, resetAndFetch)
const handleBasicSearch = () => schemePage.runQueryChange(queryState.submitQuickSearch)
const availableLineButtons = (row: InterfaceDefinitionListItem) =>
  line_buttons.value.filter((button) => {
    if (button.event_action === 'update') return row.status === 'draft'
    if (button.event_action === 'create_version') return row.status !== 'draft'
    if (button.event_action === 'enable') return row.status !== 'enabled'
    if (button.event_action === 'disable') return row.status === 'enabled'
    return true
  })
const openDetail = (row: InterfaceDefinitionListItem) => {
  currentDetailId.value = row.id
  showDetailDialog.value = true
}
const openEdit = async (row: InterfaceDefinitionListItem) => {
  currentEditData.value = (await api.getInterfaceDefinition(row.id)).data
  showFormDialog.value = true
}
const changeState = (row: InterfaceDefinitionListItem, enable: boolean) => {
  confirmAction({
    title: enable ? '确认启用' : '确认停用',
    message: `${enable ? '启用' : '停用'}接口“${row.name}”v${row.version}？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const response = enable
        ? await api.enableInterfaceDefinition(row.id, row.revision)
        : await api.disableInterfaceDefinition(row.id, row.revision)
      if (response.success) await fetchData()
    })()
  })
}
const createVersion = (row: InterfaceDefinitionListItem) => {
  confirmAction({
    title: '创建新版本',
    message: `基于“${row.name}”v${row.version}创建下一草稿版本？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const response = await api.createInterfaceDefinitionVersion(row.id, row.revision)
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
const actionHandlers: PageActionHandlers<InterfaceDefinitionListItem> = {
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
const handleButtonClick = (button: MenuButton, row?: InterfaceDefinitionListItem) => {
  dispatchPageAction(button, actionHandlers, row)
}
const handleFormSubmit = async (form: {
  external_system_id: number | null
  interface_code: string
  name: string
  protocol: 'http' | 'https'
  http_method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  relative_path: string
  credential_id: number | null
  retry_policy_id: number | null
  timeout_seconds: number
  response_limit: number
  description: string
}) => {
  if (currentEditData.value) {
    const request: InterfaceDefinitionUpdateRequest = {
      name: form.name,
      protocol: form.protocol,
      http_method: form.http_method,
      relative_path: form.relative_path,
      timeout_seconds: form.timeout_seconds,
      response_limit: form.response_limit,
      description: form.description,
      revision: currentEditData.value.revision,
    }
    if (form.credential_id) request.credential_id = form.credential_id
    else request.clear_credential = true
    if (form.retry_policy_id) request.retry_policy_id = form.retry_policy_id
    else request.clear_retry_policy = true
    if ((await api.updateInterfaceDefinition(currentEditData.value.id, request)).success)
      showFormDialog.value = false
  } else {
    const request: InterfaceDefinitionCreateRequest = {
      external_system_id: form.external_system_id!,
      interface_code: form.interface_code,
      name: form.name,
      protocol: form.protocol,
      http_method: form.http_method,
      relative_path: form.relative_path,
      timeout_seconds: form.timeout_seconds,
      response_limit: form.response_limit,
      description: form.description,
    }
    if (form.credential_id) request.credential_id = form.credential_id
    if (form.retry_policy_id) request.retry_policy_id = form.retry_policy_id
    if ((await api.createInterfaceDefinition(request)).success) showFormDialog.value = false
  }
  await fetchData()
}
onMounted(async () => {
  const requests = [fetchMetadata(), fetchSystems(), fetchCredentials()]
  if (hasGrantedCapability('integration_retry_policy_query')) requests.push(fetchRetryPolicies())
  await Promise.all(requests)
  await schemePage.initialize({ preserveInitialQuery: hasRouteSystemContext })
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
