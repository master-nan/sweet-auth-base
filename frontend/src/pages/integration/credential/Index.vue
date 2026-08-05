<template>
  <base-content class="q-pa-sm">
    <q-table v-model:pagination="pagination" class="fit sticky-header-table" color="primary" :dense="$q.screen.lt.md" separator="cell" flat bordered :rows="rows" :columns="columns" :visible-columns="visibleColumns" row-key="id" :loading="loading">
      <template #top>
        <div class="row q-gutter-xs full-width items-center">
          <div class="col-grow row q-gutter-xs items-center">
            <q-input v-model="query.quick_query!.keyword" dense outlined debounce="300" placeholder="搜索凭证编码或名称" @keyup.enter="handleBasicSearch"><template #append><q-icon name="search" /></template></q-input>
            <q-select v-model="query.external_system_id" dense outlined emit-value map-options clearable :options="systemOptions" label="所属系统" style="min-width: 220px" @update:model-value="resetAndFetch" />
            <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
            <q-select v-model="visibleColumns" multiple outlined dense options-dense emit-value map-options :display-value="compactSelectionDisplay(visibleColumns, columns, 2, '列')" :options="columns" option-value="name" options-cover />
            <q-btn outline icon="tune" color="primary" :aria-label="activeFilterCount ? `高级查询，已启用 ${activeFilterCount} 个条件` : '高级查询'" @click="showAdvancedQuery = true"><q-badge v-if="activeFilterCount" floating color="red">{{ activeFilterCount }}</q-badge><q-tooltip>高级查询</q-tooltip></q-btn>
          </div>
          <q-space />
          <div class="row q-gutter-xs"><q-btn v-for="button in top_buttons" :key="button.id" v-bind="menuButtonDisplayProps(button)" :color="button.color || 'primary'" :disable="loading" @click="handleButtonClick(button)" /></div>
        </div>
      </template>

      <template #body-cell-external_system="props"><q-td :props="props"><div class="text-weight-bold">{{ props.row.external_system.name }}</div><div class="text-caption text-grey-7">{{ props.row.external_system.system_code }}</div></q-td></template>
      <template #body-cell-credential_code="props"><q-td :props="props"><div class="text-weight-bold">{{ props.row.name }}</div><div class="text-caption text-mono text-grey-7">{{ props.row.credential_code }}</div></q-td></template>
      <template #body-cell-credential_type="props"><q-td :props="props">{{ typeLabels[props.row.credential_type] }}</q-td></template>
      <template #body-cell-effective_status="props"><q-td :props="props"><q-chip dense square outline :color="statusMeta[props.row.effective_status]?.color || 'grey'" :label="statusMeta[props.row.effective_status]?.label || props.row.effective_status" /></q-td></template>
      <template #body-cell-expires_at="props"><q-td :props="props">{{ props.row.expires_at || '长期有效' }}</q-td></template>
      <template #body-cell-version="props"><q-td :props="props"><span class="text-mono">v{{ props.row.version }}</span><div class="text-caption text-grey-7">{{ props.row.fingerprint_summary }}</div></q-td></template>
      <template #body-cell-rotated_at="props"><q-td :props="props">{{ props.row.rotated_at || '尚未轮换' }}</q-td></template>
      <template #body-cell-actions="props"><q-td :props="props" class="q-gutter-xs no-wrap"><q-btn v-for="button in availableLineButtons(props.row)" :key="button.id" flat dense size="sm" v-bind="menuButtonDisplayProps(button)" :color="button.color || 'primary'" @click="handleButtonClick(button, props.row)"><q-tooltip>{{ button.name }}</q-tooltip></q-btn></q-td></template>
      <template #bottom><q-space /><table-pagination v-model:page="query.page" v-model:page-size="query.num" :total="total" /></template>
    </q-table>

    <advanced-query v-model="showAdvancedQuery" v-model:query-model="tempAdvancedQuery" :fields="advancedFields" @search="handleAdvancedSearch" />
    <credential-form-dialog v-model="showFormDialog" :edit-data="currentEditData" :systems="systems" :rotate-mode="rotateMode" :loading="loading" @submit="handleFormSubmit" />
    <credential-detail-dialog v-model="showDetailDialog" :id="currentDetailId" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_credential' })

import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import CredentialFormDialog from './CredentialFormDialog.vue'
import CredentialDetailDialog from './CredentialDetailDialog.vue'
import { type CredentialCreateRequest, type CredentialDetail, type CredentialListItem, type CredentialQuery, type CredentialSecret, type CredentialUpdateRequest, type ExternalSystemListItem, useIntegrationApi } from 'src/api/services/integration'
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
const api = useIntegrationApi()
const tableApi = useTableApi()
const { loading } = storeToRefs(useLoadingStore())
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('integration_credential')
const rows = ref<CredentialListItem[]>([])
const systems = ref<ExternalSystemListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showAdvancedQuery = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const rotateMode = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<CredentialDetail | null>(null)
const advancedFields = ref<TableField[]>([])
const typeLabels: Record<string, string> = { basic: 'Basic', api_key: 'API Key', bearer_token: 'Bearer Token', oauth_client: 'OAuth Client' }
const statusMeta: Record<string, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'grey-7' }, active: { label: '已启用', color: 'positive' },
  disabled: { label: '已停用', color: 'warning' }, revoked: { label: '已吊销', color: 'negative' }, expired: { label: '已过期', color: 'negative' },
}
const columns: QTableProps['columns'] = [
  { name: 'external_system', label: '所属系统', field: (row) => row.external_system.name, align: 'left' },
  { name: 'credential_code', label: '集成凭证', field: 'credential_code', align: 'left', sortable: true },
  { name: 'credential_type', label: '类型', field: 'credential_type', align: 'left', sortable: true },
  { name: 'effective_status', label: '状态', field: 'effective_status', align: 'center' },
  { name: 'expires_at', label: '有效期', field: 'expires_at', align: 'left', sortable: true },
  { name: 'version', label: '版本 / 指纹', field: 'version', align: 'left', sortable: true },
  { name: 'rotated_at', label: '轮换时间', field: 'rotated_at', align: 'left', sortable: true },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]
const visibleColumns = ref(columns.map((column) => column.name))
const systemOptions = computed(() => systems.value.map((item) => ({ label: `${item.name}（${item.system_code}）`, value: item.id })))
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const query = ref<CredentialQuery>({ page: 1, num: 15, order: { field: '', is_asc: false }, quick_query: { keyword: '' }, expressions: emptyExpressions() })
const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref<Query>(cloneDeep(query.value))
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })

const fetchData = async () => { const response = await api.queryCredentials(query.value); rows.value = response.data || []; total.value = response.total || 0 }
const fetchSystems = async () => { const response = await api.queryExternalSystems({ page: 1, num: 500, order: { field: 'name', is_asc: true }, quick_query: { keyword: '' }, expressions: [] }); systems.value = response.data || [] }
const fetchMetadata = async () => { const response = await tableApi.queryTableByCode('integration_credential'); advancedFields.value = (response.data?.table_fields || []).filter((field) => field.is_advanced_search && field.field_code !== 'external_system_id') }
const resetAndFetch = () => { if (query.value.page !== 1) query.value.page = 1; else void fetchData() }
const handleBasicSearch = () => { query.value.expressions = emptyExpressions(); appliedAdvancedQuery.value = cloneDeep(query.value); resetAndFetch() }
const handleAdvancedSearch = () => { query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions); appliedAdvancedQuery.value = cloneDeep(query.value); showAdvancedQuery.value = false; resetAndFetch() }
const availableLineButtons = (row: CredentialListItem) => line_buttons.value.filter((button) => {
  if (row.status === 'revoked') return button.event_action === 'detail'
  if (button.event_action === 'enable') return row.status === 'draft' || row.status === 'disabled'
  if (button.event_action === 'disable') return row.status === 'active'
  return true
})
const loadDetail = async (row: CredentialListItem) => (await api.getCredential(row.id)).data || null
const openDetail = (row: CredentialListItem) => { currentDetailId.value = row.id; showDetailDialog.value = true }
const openEdit = async (row: CredentialListItem, rotate = false) => { currentEditData.value = await loadDetail(row); rotateMode.value = rotate; showFormDialog.value = true }
const changeState = (row: CredentialListItem, target: 'active' | 'disabled' | 'revoked') => {
  const labels = { active: '启用', disabled: '停用', revoked: '吊销' }
  confirmAction({ title: `确认${labels[target]}`, message: `${labels[target]}凭证“${row.name}”${target === 'revoked' ? '？吊销后不可恢复。' : '？'}`, loading: loading.value, disable: loading.value }).onOk(() => { void (async () => {
    const response = target === 'active' ? await api.enableCredential(row.id, row.revision) : target === 'disabled' ? await api.disableCredential(row.id, row.revision) : await api.revokeCredential(row.id, row.revision)
    if (response.success) await fetchData()
  })() })
}
const actionHandlers: Record<string, (row?: CredentialListItem) => void> = {
  create: () => { currentEditData.value = null; rotateMode.value = false; showFormDialog.value = true },
  detail: (row) => row && openDetail(row), update: (row) => row && void openEdit(row), rotate: (row) => row && void openEdit(row, true),
  enable: (row) => row && changeState(row, 'active'), disable: (row) => row && changeState(row, 'disabled'), revoke: (row) => row && changeState(row, 'revoked'),
}
const handleButtonClick = (button: MenuButton, row?: CredentialListItem) => { actionHandlers[button.event_action]?.(row) }
const toAPIDate = (value: string) => new Date(value).toISOString()
const handleFormSubmit = async (form: { external_system_id: number | null; credential_code: string; name: string; credential_type: CredentialCreateRequest['credential_type']; expires_at: string; description: string; secret: CredentialSecret }) => {
  if (rotateMode.value && currentEditData.value) {
    if ((await api.rotateCredential(currentEditData.value.id, form.secret, currentEditData.value.revision)).success) showFormDialog.value = false
  } else if (currentEditData.value) {
    const request: CredentialUpdateRequest = { name: form.name, description: form.description, revision: currentEditData.value.revision }
    if (form.expires_at) request.expires_at = toAPIDate(form.expires_at)
    else request.clear_expires_at = true
    if ((await api.updateCredential(currentEditData.value.id, request)).success) showFormDialog.value = false
  } else {
    const request: CredentialCreateRequest = { external_system_id: form.external_system_id!, credential_code: form.credential_code, name: form.name, credential_type: form.credential_type, secret: form.secret, description: form.description }
    if (form.expires_at) request.expires_at = toAPIDate(form.expires_at)
    if ((await api.createCredential(request)).success) showFormDialog.value = false
  }
  await fetchData()
}
onMounted(async () => { await Promise.all([fetchMetadata(), fetchSystems(), fetchData()]); if (!has_line_buttons.value) visibleColumns.value = visibleColumns.value.filter((name) => name !== 'actions'); initialized.value = true })
watch(() => [query.value.page, query.value.num] as const, ([page]) => { if (!initialized.value) return; pagination.value.page = page; void fetchData() })
watch(() => [pagination.value.sortBy, pagination.value.descending] as const, ([sortBy, descending], previous) => { if (!initialized.value || (sortBy === previous[0] && descending === previous[1])) return; query.value.order = { field: sortBy || '', is_asc: sortBy ? !descending : false }; resetAndFetch() })
watch(showAdvancedQuery, (open) => { if (open) tempAdvancedQuery.value = cloneDeep(query.value) })
</script>
