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
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn
                  color="primary"
                  :label="t('ui.search')"
                  :disable="loading"
                  @click="handleBasicSearch"
                />
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
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'integration_interface_definition' })

import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from '@/components/Table/TableColumnSelector.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
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
} from '@/api/services/integration'
import { usePageButtons } from '@/composables/page-buttons'
import { useConfirmDialog } from '@/composables/confirm-dialog'
import { useRuntimeTableMetadata } from '@/composables/runtime-table-metadata'
import { useTableQueryState } from '@/composables/table-query-state'
import { useQuerySchemePage } from '@/composables/query-scheme-page'
import type { MenuButton } from '@/api/services/sys-menu'
import type { TableColumn } from '@/types/global'
import { ExpressionType } from '@/types/enum'
import { countEffectiveQueryRules } from '@/utils/query-state'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'
import { dispatchPageAction, type PageActionHandlers } from '@/utils/button-handlers'
import { resolveRuntimeColumns } from '@/utils/column-format'
import { resolveTableEmptyMessage } from '@/utils/table-state'

const { t } = useI18n({ useScope: 'global' })

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
  draft: {
    get label() {
      return t('ui.draft')
    },
    color: 'grey-7',
  },
  enabled: {
    get label() {
      return t('ui.activatedStatus')
    },
    color: 'positive',
  },
  disabled: {
    get label() {
      return t('ui.deactivatedStatus')
    },
    color: 'warning',
  },
  unavailable: {
    get label() {
      return t('ui.notAvailable')
    },
    color: 'negative',
  },
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
    num: 20,
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
    loadError.value = t('ui.interfaceDefinitionLoadFailed')
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
      {
        fieldCode: 'interface_code',
        get label() {
          return t('ui.interfaceDefinition')
        },
        order: 2,
      },
      { fieldCode: 'http_method', label: 'Method', align: 'center', order: 3 },
      { fieldCode: 'protocol', align: 'center', order: 5 },
      { fieldCode: 'status', align: 'center', order: 6 },
    ],
    virtualColumns: [
      {
        name: 'external_system',
        get label() {
          return t('ui.owningSystem')
        },
        field: (row) => row.external_system.name,
        order: 1,
      },
      {
        name: 'path_summary',
        get label() {
          return t('ui.relativePathLabel')
        },
        field: 'path_summary',
        order: 4,
      },
      {
        name: 'actions',
        get label() {
          return t('ui.actions')
        },
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
    get title() {
      return enable ? t('ui.confirmEnable') : t('ui.confirmDisable')
    },
    get message() {
      return t('ui.interfaceV', {
        value1: enable ? t('ui.enabled') : t('ui.disabled'),
        value2: row.name,
        value3: row.version,
      })
    },
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
    get title() {
      return t('ui.createANewVersion')
    },
    get message() {
      return t('ui.createTheNextDraftBasedOnV', { value1: row.name, value2: row.version })
    },
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
  input_contract: InterfaceDefinitionCreateRequest['input_contract']
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
      input_contract: form.input_contract,
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
      input_contract: form.input_contract,
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
