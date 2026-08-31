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

      <template #body-cell-system_code="props">
        <q-td :props="props">
          <div class="text-weight-bold">{{ props.row.system_code }}</div>
          <div class="text-caption text-grey-7">{{ props.row.name }}</div>
        </q-td>
      </template>

      <template #body-cell-system_type="props">
        <q-td :props="props">{{ typeLabels[props.row.system_type] || props.row.system_type }}</q-td>
      </template>

      <template #body-cell-owner_name="props">
        <q-td :props="props">
          <div>{{ props.row.owner_name }}</div>
          <div class="text-caption text-grey-7">{{ props.row.owner_identifier }}</div>
        </q-td>
      </template>

      <template #body-cell-status="props">
        <q-td :props="props">
          <status-chip
            :color="statusMeta[props.row.status]?.color || 'grey'"
            :label="statusMeta[props.row.status]?.label || props.row.status"
          />
        </q-td>
      </template>

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
          >
            <q-tooltip>{{ button.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:page-size="query.num" :total="total" />
      </template>
    </q-table>

    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="currentEditData ? t('ui.editExternalSystem') : t('ui.addExternalSystem')"
      :fields="formFields"
      :submit-btn-text="currentEditData ? t('ui.save') : t('ui.createRecord')"
      @submit="handleFormSubmit"
    />

    <external-system-detail-dialog
      v-model="showDetailDialog"
      :id="currentDetailId"
      @show-interfaces="openRelatedInterfaces"
      @show-credentials="openRelatedCredentials"
    />
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'integration_external_system' })

import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from 'src/components/Table/TableColumnSelector.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import ExternalSystemDetailDialog from './ExternalSystemDetailDialog.vue'
import {
  type ExternalSystemCreateRequest,
  type ExternalSystemDetail,
  type ExternalSystemListItem,
  type ExternalSystemQuery,
  type ExternalSystemUpdateRequest,
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
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-handlers'
import { resolveRuntimeColumns } from 'src/utils/column-format'
import { resolveTableEmptyMessage } from 'src/utils/table-state'

const { t } = useI18n({ useScope: 'global' })

const $q = useQuasar()
const router = useRouter()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons(
  'integration_external_system',
)

const rows = ref<ExternalSystemListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<ExternalSystemDetail | null>(null)
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: advancedFields,
  formFields,
  loadMetadata,
} = useRuntimeTableMetadata('integration_external_system')

const typeLabels: Record<string, string> = {
  get hr() {
    return t('ui.humanResourcesSystem')
  },
  get erp() {
    return t('ui.enterpriseResourcePlanning')
  },
  get tms() {
    return t('ui.transportationManagementSystem')
  },
  get wms() {
    return t('ui.warehouseManagementSystem')
  },
  get other() {
    return t('ui.otherSystem')
  },
}
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
}

const columns = ref<TableColumn<ExternalSystemListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())

const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const queryState = useTableQueryState<ExternalSystemQuery>({
  createInitialQuery: () => ({
    page: 1,
    num: 20,
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

const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.queryExternalSystems(query.value)
    rows.value = response.data || []
    total.value = response.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.externalSystemLoadFailed')
  } finally {
    loading.value = false
  }
}

const fetchMetadata = async () => {
  if (!(await loadMetadata())) return
  const resolution = resolveRuntimeColumns<ExternalSystemListItem>(metadataFields.value, {
    context: { getDictLabel: () => '' },
    overrides: [
      {
        fieldCode: 'system_code',
        get label() {
          return t('ui.externalSystemLabel')
        },
        order: 1,
      },
      { fieldCode: 'name', visible: false, order: 2 },
      {
        fieldCode: 'system_type',
        get label() {
          return t('ui.type')
        },
        order: 3,
      },
      { fieldCode: 'owner_name', order: 5 },
      { fieldCode: 'status', align: 'center', order: 6 },
    ],
    virtualColumns: [
      {
        name: 'base_url_summary',
        get label() {
          return t('ui.addressSummary')
        },
        field: 'base_url_summary',
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

const schemePage = useQuerySchemePage('integration_external_system', queryState, resetAndFetch)

const handleBasicSearch = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

const availableLineButtons = (row: ExternalSystemListItem) =>
  line_buttons.value.filter((button) => {
    if (button.event_action === 'enable') return row.status !== 'enabled'
    if (button.event_action === 'disable') return row.status === 'enabled'
    return true
  })

const openDetail = (row: ExternalSystemListItem) => {
  currentDetailId.value = row.id
  showDetailDialog.value = true
}
const openRelatedInterfaces = (id: number) => {
  showDetailDialog.value = false
  void router.push({
    name: 'integration_interface_definition',
    query: { external_system_id: String(id) },
  })
}
const openRelatedCredentials = (id: number) => {
  showDetailDialog.value = false
  void router.push({ name: 'integration_credential', query: { external_system_id: String(id) } })
}

const openEdit = async (row: ExternalSystemListItem) => {
  const response = await api.getExternalSystem(row.id)
  currentEditData.value = response.data
  showFormDialog.value = true
}

const changeState = (row: ExternalSystemListItem, enable: boolean) => {
  confirmAction({
    get title() {
      return enable ? t('ui.confirmEnable') : t('ui.confirmDisable')
    },
    get message() {
      return t('ui.confirmExternalSystemAction', {
        value1: enable ? t('ui.enabled') : t('ui.disabled'),
        value2: row.name,
      })
    },
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const response = enable
        ? await api.enableExternalSystem(row.id, row.revision)
        : await api.disableExternalSystem(row.id, row.revision)
      if (response.success) await fetchData()
    })()
  })
}

const actionHandlers: PageActionHandlers<ExternalSystemListItem> = {
  create: () => {
    currentEditData.value = null
    showFormDialog.value = true
  },
  detail: (row) => row && openDetail(row),
  update: (row) => row && void openEdit(row),
  enable: (row) => row && changeState(row, true),
  disable: (row) => row && changeState(row, false),
}

const handleButtonClick = (button: MenuButton, row?: ExternalSystemListItem) => {
  dispatchPageAction(button, actionHandlers, row)
}

const handleFormSubmit = async (payload: {
  data: ExternalSystemDetail
  isEdit: boolean
  id?: number
}) => {
  if (payload.isEdit && payload.id) {
    const request: ExternalSystemUpdateRequest = {
      name: payload.data.name,
      system_type: payload.data.system_type,
      base_url: payload.data.base_url,
      owner_identifier: payload.data.owner_identifier,
      owner_name: payload.data.owner_name,
      description: payload.data.description || '',
      revision: payload.data.revision,
    }
    const response = await api.updateExternalSystem(payload.id, request)
    if (response.success) showFormDialog.value = false
  } else {
    const request: ExternalSystemCreateRequest = {
      system_code: payload.data.system_code,
      name: payload.data.name,
      system_type: payload.data.system_type,
      base_url: payload.data.base_url,
      owner_identifier: payload.data.owner_identifier,
      owner_name: payload.data.owner_name,
      description: payload.data.description || '',
    }
    const response = await api.createExternalSystem(request)
    if (response.success) showFormDialog.value = false
  }
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
