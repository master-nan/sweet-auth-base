<template>
  <base-content class="q-pa-sm">
    <q-table
      v-model:pagination="pagination"
      class="fit sticky-header-table"
      color="primary"
      :dense="$q.screen.lt.md"
      separator="cell"
      flat bordered
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      row-key="id"
      :loading="loading"
    >
      <template #top>
        <div class="row q-gutter-xs full-width items-center">
          <div class="col-grow row q-gutter-xs items-center">
            <q-input v-model="query.quick_query!.keyword" dense outlined debounce="300" placeholder="搜索接口编码、名称或路径" @keyup.enter="handleBasicSearch">
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-select
              v-model="query.external_system_id"
              dense outlined emit-value map-options clearable
              :options="systemOptions"
              label="所属系统"
              style="min-width: 220px"
              @update:model-value="resetAndFetch"
            />
            <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
            <q-select v-model="visibleColumns" multiple outlined dense options-dense emit-value map-options :display-value="compactSelectionDisplay(visibleColumns, columns, 2, '列')" :options="columns" option-value="name" options-cover />
            <q-btn outline icon="tune" color="primary" :aria-label="activeFilterCount ? `高级查询，已启用 ${activeFilterCount} 个条件` : '高级查询'" @click="showAdvancedQuery = true">
              <q-badge v-if="activeFilterCount" floating color="red">{{ activeFilterCount }}</q-badge>
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <div class="row q-gutter-xs">
            <q-btn v-for="button in top_buttons" :key="button.id" v-bind="menuButtonDisplayProps(button)" :color="button.color || 'primary'" :disable="loading" @click="handleButtonClick(button)" />
          </div>
        </div>
      </template>

      <template #body-cell-external_system="props">
        <q-td :props="props"><div class="text-weight-bold">{{ props.row.external_system.name }}</div><div class="text-caption text-grey-7">{{ props.row.external_system.system_code }}</div></q-td>
      </template>
      <template #body-cell-interface_code="props">
        <q-td :props="props"><div class="text-weight-bold">{{ props.row.name }}</div><div class="text-caption text-mono text-grey-7">{{ props.row.interface_code }} · v{{ props.row.version }}</div></q-td>
      </template>
      <template #body-cell-http_method="props"><q-td :props="props"><q-chip dense square color="primary" text-color="white" :label="props.row.http_method" /></q-td></template>
      <template #body-cell-path_summary="props"><q-td :props="props"><span class="text-mono">{{ props.row.path_summary }}</span></q-td></template>
      <template #body-cell-status="props"><q-td :props="props"><q-chip dense square outline :color="statusMeta[props.row.effective_status]?.color || 'grey'" :label="statusMeta[props.row.effective_status]?.label || props.row.effective_status" /></q-td></template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs no-wrap">
          <q-btn v-for="button in availableLineButtons(props.row)" :key="button.id" flat dense size="sm" v-bind="menuButtonDisplayProps(button)" :color="button.color || 'primary'" @click="handleButtonClick(button, props.row)"><q-tooltip>{{ button.name }}</q-tooltip></q-btn>
        </q-td>
      </template>
      <template #bottom><q-space /><table-pagination v-model:page="query.page" v-model:page-size="query.num" :total="total" /></template>
    </q-table>

    <advanced-query v-model="showAdvancedQuery" v-model:query-model="tempAdvancedQuery" :fields="advancedFields" @search="handleAdvancedSearch" />
    <interface-definition-form-dialog v-model="showFormDialog" :edit-data="currentEditData" :systems="systems" :credentials="credentials" :loading="loading" @submit="handleFormSubmit" />
    <interface-definition-detail-dialog v-model="showDetailDialog" :id="currentDetailId" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_interface_definition' })

import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useRoute } from 'vue-router'
import cloneDeep from 'lodash/cloneDeep'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
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
  useIntegrationApi,
} from 'src/api/services/integration'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import type { MenuButton } from 'src/api/services/sys-menu'
import type { Query } from 'src/types/global'
import { countEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { compactSelectionDisplay } from 'src/utils/select-display'

const $q = useQuasar()
const route = useRoute()
const api = useIntegrationApi()
const tableApi = useTableApi()
const { loading } = storeToRefs(useLoadingStore())
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('integration_interface_definition')
const rows = ref<InterfaceDefinitionListItem[]>([])
const systems = ref<ExternalSystemListItem[]>([])
const credentials = ref<CredentialListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showAdvancedQuery = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<InterfaceDefinitionDetail | null>(null)
const advancedFields = ref<TableField[]>([])
const statusMeta: Record<string, { label: string; color: string }> = { draft: { label: '草稿', color: 'grey-7' }, enabled: { label: '已启用', color: 'positive' }, disabled: { label: '已停用', color: 'warning' }, unavailable: { label: '当前不可用', color: 'negative' } }
const columns: QTableProps['columns'] = [
  { name: 'external_system', label: '所属系统', field: (row) => row.external_system.name, align: 'left' },
  { name: 'interface_code', label: '接口定义', field: 'interface_code', align: 'left', sortable: true },
  { name: 'http_method', label: 'Method', field: 'http_method', align: 'center', sortable: true },
  { name: 'path_summary', label: '相对路径', field: 'path_summary', align: 'left' },
  { name: 'protocol', label: '协议', field: 'protocol', align: 'center', sortable: true },
  { name: 'status', label: '状态', field: 'status', align: 'center', sortable: true },
  { name: 'gmt_modify', label: '更新时间', field: 'gmt_modify', align: 'left', sortable: true },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]
const visibleColumns = ref(columns.map((column) => column.name))
const systemOptions = computed(() => systems.value.map((item) => ({ label: `${item.name}（${item.system_code}）`, value: item.id })))
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const routeSystemID = Number(route.query.external_system_id)
const initialQuery: InterfaceDefinitionQuery = { page: 1, num: 15, order: { field: '', is_asc: false }, quick_query: { keyword: '' }, expressions: emptyExpressions() }
if (Number.isSafeInteger(routeSystemID) && routeSystemID > 0) initialQuery.external_system_id = routeSystemID
const query = ref<InterfaceDefinitionQuery>(initialQuery)
const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref<Query>(cloneDeep(query.value))
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })

const fetchData = async () => { const response = await api.queryInterfaceDefinitions(query.value); rows.value = response.data || []; total.value = response.total || 0 }
const fetchSystems = async () => { const response = await api.queryExternalSystems({ page: 1, num: 500, order: { field: 'name', is_asc: true }, quick_query: { keyword: '' }, expressions: [] }); systems.value = response.data || [] }
const fetchCredentials = async () => { const response = await api.queryCredentials({ page: 1, num: 500, order: { field: 'credential_code', is_asc: true }, quick_query: { keyword: '' }, expressions: [] }); credentials.value = response.data || [] }
const fetchMetadata = async () => { const response = await tableApi.queryTableByCode('integration_interface_definition'); advancedFields.value = (response.data?.table_fields || []).filter((field) => field.is_advanced_search && field.field_code !== 'external_system_id') }
const resetAndFetch = () => { if (query.value.page !== 1) query.value.page = 1; else void fetchData() }
const handleBasicSearch = () => { query.value.expressions = emptyExpressions(); appliedAdvancedQuery.value = cloneDeep(query.value); resetAndFetch() }
const handleAdvancedSearch = () => { query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions); appliedAdvancedQuery.value = cloneDeep(query.value); showAdvancedQuery.value = false; resetAndFetch() }
const availableLineButtons = (row: InterfaceDefinitionListItem) => line_buttons.value.filter((button) => {
  if (button.event_action === 'update') return row.status === 'draft'
  if (button.event_action === 'create_version') return row.status !== 'draft'
  if (button.event_action === 'enable') return row.status !== 'enabled'
  if (button.event_action === 'disable') return row.status === 'enabled'
  return true
})
const openDetail = (row: InterfaceDefinitionListItem) => { currentDetailId.value = row.id; showDetailDialog.value = true }
const openEdit = async (row: InterfaceDefinitionListItem) => { currentEditData.value = (await api.getInterfaceDefinition(row.id)).data; showFormDialog.value = true }
const changeState = (row: InterfaceDefinitionListItem, enable: boolean) => {
  confirmAction({ title: enable ? '确认启用' : '确认停用', message: `${enable ? '启用' : '停用'}接口“${row.name}”v${row.version}？`, loading: loading.value, disable: loading.value }).onOk(() => { void (async () => { const response = enable ? await api.enableInterfaceDefinition(row.id, row.revision) : await api.disableInterfaceDefinition(row.id, row.revision); if (response.success) await fetchData() })() })
}
const createVersion = (row: InterfaceDefinitionListItem) => {
  confirmAction({ title: '创建新版本', message: `基于“${row.name}”v${row.version}创建下一草稿版本？`, loading: loading.value, disable: loading.value }).onOk(() => { void (async () => { const response = await api.createInterfaceDefinitionVersion(row.id, row.revision); if (response.success) { await fetchData(); if (response.data) { currentEditData.value = response.data; showFormDialog.value = true } } })() })
}
const actionHandlers: Record<string, (row?: InterfaceDefinitionListItem) => void> = {
  create: () => { currentEditData.value = null; showFormDialog.value = true },
  detail: (row) => row && openDetail(row), update: (row) => row && void openEdit(row), create_version: (row) => row && createVersion(row),
  enable: (row) => row && changeState(row, true), disable: (row) => row && changeState(row, false),
}
const handleButtonClick = (button: MenuButton, row?: InterfaceDefinitionListItem) => { actionHandlers[button.event_action]?.(row) }
const handleFormSubmit = async (form: { external_system_id: number | null; interface_code: string; name: string; protocol: 'http' | 'https'; http_method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'; relative_path: string; credential_id: number | null; timeout_seconds: number; response_limit: number; description: string }) => {
  if (currentEditData.value) {
    const request: InterfaceDefinitionUpdateRequest = { name: form.name, protocol: form.protocol, http_method: form.http_method, relative_path: form.relative_path, timeout_seconds: form.timeout_seconds, response_limit: form.response_limit, description: form.description, revision: currentEditData.value.revision }
    if (form.credential_id) request.credential_id = form.credential_id
    else request.clear_credential = true
    if ((await api.updateInterfaceDefinition(currentEditData.value.id, request)).success) showFormDialog.value = false
  } else {
    const request: InterfaceDefinitionCreateRequest = { external_system_id: form.external_system_id!, interface_code: form.interface_code, name: form.name, protocol: form.protocol, http_method: form.http_method, relative_path: form.relative_path, timeout_seconds: form.timeout_seconds, response_limit: form.response_limit, description: form.description }
    if (form.credential_id) request.credential_id = form.credential_id
    if ((await api.createInterfaceDefinition(request)).success) showFormDialog.value = false
  }
  await fetchData()
}
onMounted(async () => { await Promise.all([fetchMetadata(), fetchSystems(), fetchCredentials(), fetchData()]); if (!has_line_buttons.value) visibleColumns.value = visibleColumns.value.filter((name) => name !== 'actions'); initialized.value = true })
watch(() => [query.value.page, query.value.num] as const, ([page]) => { if (!initialized.value) return; pagination.value.page = page; void fetchData() })
watch(() => [pagination.value.sortBy, pagination.value.descending] as const, ([sortBy, descending], previous) => { if (!initialized.value || (sortBy === previous[0] && descending === previous[1])) return; query.value.order = { field: sortBy || '', is_asc: sortBy ? !descending : false }; resetAndFetch() })
watch(showAdvancedQuery, (open) => { if (open) tempAdvancedQuery.value = cloneDeep(query.value) })
</script>
