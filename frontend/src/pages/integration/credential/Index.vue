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
          <template #right-actions
            ><q-btn
              v-for="button in top_buttons"
              :key="button.id"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :disable="loading"
              @click="handleButtonClick(button)"
          /></template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-external_system="props"
        ><q-td :props="props"
          ><div class="text-weight-bold">{{ props.row.external_system.name }}</div>
          <div class="text-caption text-grey-7">
            {{ props.row.external_system.system_code }}
          </div></q-td
        ></template
      >
      <template #body-cell-credential_code="props"
        ><q-td :props="props"
          ><div class="text-weight-bold">{{ props.row.name }}</div>
          <div class="text-caption text-mono text-grey-7">
            {{ props.row.credential_code }}
          </div></q-td
        ></template
      >
      <template #body-cell-credential_type="props"
        ><q-td :props="props">{{ typeLabels[props.row.credential_type] }}</q-td></template
      >
      <template #body-cell-effective_status="props"
        ><q-td :props="props"
          ><status-chip
            :color="statusMeta[props.row.effective_status]?.color || 'grey'"
            :label="
              statusMeta[props.row.effective_status]?.label || props.row.effective_status
            " /></q-td
      ></template>
      <template #body-cell-expires_at="props"
        ><q-td :props="props">{{ props.row.expires_at || '长期有效' }}</q-td></template
      >
      <template #body-cell-version="props"
        ><q-td :props="props"
          ><span class="text-mono">v{{ props.row.version }}</span>
          <div class="text-caption text-grey-7">{{ props.row.fingerprint_summary }}</div></q-td
        ></template
      >
      <template #body-cell-rotated_at="props"
        ><q-td :props="props">{{ props.row.rotated_at || '尚未轮换' }}</q-td></template
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
            @click="handleButtonClick(button, props.row)"
            ><q-tooltip>{{ button.name }}</q-tooltip></q-btn
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

    <credential-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :systems="systems"
      :rotate-mode="rotateMode"
      :loading="loading"
      @submit="handleFormSubmit"
    />
    <credential-detail-dialog v-model="showDetailDialog" :id="currentDetailId" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_credential' })

import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from 'src/components/Table/TableColumnSelector.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import CredentialFormDialog from './CredentialFormDialog.vue'
import CredentialDetailDialog from './CredentialDetailDialog.vue'
import {
  type CredentialCreateRequest,
  type CredentialDetail,
  type CredentialListItem,
  type CredentialQuery,
  type CredentialSecret,
  type CredentialUpdateRequest,
  type ExternalSystemListItem,
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
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-handlers'
import { resolveRuntimeColumns } from 'src/utils/column-format'
import { resolveTableEmptyMessage } from 'src/utils/table-state'

const $q = useQuasar()
const route = useRoute()
const api = useIntegrationApi()
const loading = ref(false)
const loadError = ref('')
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('integration_credential')
const rows = ref<CredentialListItem[]>([])
const systems = ref<ExternalSystemListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const rotateMode = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<CredentialDetail | null>(null)
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: metadataAdvancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('integration_credential')
const advancedFields = computed(() => metadataAdvancedFields.value)
const typeLabels: Record<string, string> = {
  basic: 'Basic',
  api_key: 'API Key',
  bearer_token: 'Bearer Token',
  oauth_client: 'OAuth Client',
}
const statusMeta: Record<string, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'grey-7' },
  active: { label: '已启用', color: 'positive' },
  disabled: { label: '已停用', color: 'warning' },
  revoked: { label: '已吊销', color: 'negative' },
  expired: { label: '已过期', color: 'negative' },
}
const columns = ref<TableColumn<CredentialListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const routeSystemID = Number(route.query.external_system_id)
const hasRouteSystemContext = Number.isSafeInteger(routeSystemID) && routeSystemID > 0
const queryState = useTableQueryState<CredentialQuery>({
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
    const response = await api.queryCredentials(query.value)
    rows.value = response.data || []
    total.value = response.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '集成凭证加载失败'
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
const fetchMetadata = async () => {
  if (!(await loadMetadata())) return
  const resolution = resolveRuntimeColumns<CredentialListItem>(metadataFields.value, {
    context: { getDictLabel: () => '' },
    overrides: [
      { fieldCode: 'external_system_id', visible: false },
      { fieldCode: 'name', visible: false },
      { fieldCode: 'status', visible: false },
      { fieldCode: 'credential_code', label: '集成凭证', order: 2 },
      { fieldCode: 'credential_type', order: 3 },
      { fieldCode: 'expires_at', order: 5 },
      { fieldCode: 'rotated_at', order: 7 },
    ],
    virtualColumns: [
      {
        name: 'external_system',
        label: '所属系统',
        field: (row) => row.external_system.name,
        order: 1,
      },
      {
        name: 'effective_status',
        label: '状态',
        field: 'effective_status',
        align: 'center',
        order: 4,
      },
      {
        name: 'version',
        label: '版本 / 指纹',
        field: 'version',
        order: 6,
        serverSortField: 'version',
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
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('integration_credential', queryState, resetAndFetch)
const handleBasicSearch = () => schemePage.runQueryChange(queryState.submitQuickSearch)
const availableLineButtons = (row: CredentialListItem) =>
  line_buttons.value.filter((button) => {
    if (row.status === 'revoked') return button.event_action === 'detail'
    if (button.event_action === 'enable') return row.status === 'draft' || row.status === 'disabled'
    if (button.event_action === 'disable') return row.status === 'active'
    return true
  })
const loadDetail = async (row: CredentialListItem) => (await api.getCredential(row.id)).data || null
const openDetail = (row: CredentialListItem) => {
  currentDetailId.value = row.id
  showDetailDialog.value = true
}
const openEdit = async (row: CredentialListItem, rotate = false) => {
  currentEditData.value = await loadDetail(row)
  rotateMode.value = rotate
  showFormDialog.value = true
}
const changeState = (row: CredentialListItem, target: 'active' | 'disabled' | 'revoked') => {
  const labels = { active: '启用', disabled: '停用', revoked: '吊销' }
  confirmAction({
    title: `确认${labels[target]}`,
    message: `${labels[target]}凭证“${row.name}”${target === 'revoked' ? '？吊销后不可恢复。' : '？'}`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const response =
        target === 'active'
          ? await api.enableCredential(row.id, row.revision)
          : target === 'disabled'
            ? await api.disableCredential(row.id, row.revision)
            : await api.revokeCredential(row.id, row.revision)
      if (response.success) await fetchData()
    })()
  })
}
const actionHandlers: PageActionHandlers<CredentialListItem> = {
  create: () => {
    currentEditData.value = null
    rotateMode.value = false
    showFormDialog.value = true
  },
  detail: (row) => row && openDetail(row),
  update: (row) => row && void openEdit(row),
  rotate: (row) => row && void openEdit(row, true),
  enable: (row) => row && changeState(row, 'active'),
  disable: (row) => row && changeState(row, 'disabled'),
  revoke: (row) => row && changeState(row, 'revoked'),
}
const handleButtonClick = (button: MenuButton, row?: CredentialListItem) => {
  dispatchPageAction(button, actionHandlers, row)
}
const toAPIDate = (value: string) => new Date(value.replace(' ', 'T')).toISOString()
const handleFormSubmit = async (form: {
  external_system_id: number | null
  credential_code: string
  name: string
  credential_type: CredentialCreateRequest['credential_type']
  expires_at: string
  description: string
  secret: CredentialSecret
}) => {
  if (rotateMode.value && currentEditData.value) {
    if (
      (
        await api.rotateCredential(
          currentEditData.value.id,
          form.secret,
          currentEditData.value.revision,
        )
      ).success
    )
      showFormDialog.value = false
  } else if (currentEditData.value) {
    const request: CredentialUpdateRequest = {
      name: form.name,
      description: form.description,
      revision: currentEditData.value.revision,
    }
    if (form.expires_at) request.expires_at = toAPIDate(form.expires_at)
    else request.clear_expires_at = true
    if ((await api.updateCredential(currentEditData.value.id, request)).success)
      showFormDialog.value = false
  } else {
    const request: CredentialCreateRequest = {
      external_system_id: form.external_system_id!,
      credential_code: form.credential_code,
      name: form.name,
      credential_type: form.credential_type,
      secret: form.secret,
      description: form.description,
    }
    if (form.expires_at) request.expires_at = toAPIDate(form.expires_at)
    if ((await api.createCredential(request)).success) showFormDialog.value = false
  }
  await fetchData()
}
onMounted(async () => {
  await Promise.all([fetchMetadata(), fetchSystems()])
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
