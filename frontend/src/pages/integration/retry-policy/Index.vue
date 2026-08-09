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
            <q-input v-model="query.quick_query!.keyword" dense outlined debounce="300" placeholder="搜索策略编码或名称" @keyup.enter="handleBasicSearch">
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-select v-model="query.status" dense outlined emit-value map-options clearable :options="statusOptions" label="状态" style="min-width: 150px" @update:model-value="resetAndFetch" />
            <q-select v-model="query.backoff_type" dense outlined emit-value map-options clearable :options="backoffOptions" label="退避方式" style="min-width: 160px" @update:model-value="resetAndFetch" />
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

      <template #body-cell-policy_code="props">
        <q-td :props="props"><div class="text-weight-bold">{{ props.row.policy_name }}</div><div class="text-caption text-mono text-grey-7">{{ props.row.policy_code }} · v{{ props.row.version }}</div></q-td>
      </template>
      <template #body-cell-status="props"><q-td :props="props"><q-chip dense square outline :color="policyStatusMeta(props.row).color" :label="policyStatusMeta(props.row).label" /></q-td></template>
      <template #body-cell-backoff_type="props"><q-td :props="props">{{ policyBackoffLabel(props.row) }}</q-td></template>
      <template #body-cell-initial_delay_ms="props"><q-td :props="props">{{ formatDuration(props.row.initial_delay_ms) }}</q-td></template>
      <template #body-cell-max_delay_ms="props"><q-td :props="props">{{ formatDuration(props.row.max_delay_ms) }}</q-td></template>
      <template #body-cell-retry_window_ms="props"><q-td :props="props">{{ formatDuration(props.row.retry_window_ms) }}</q-td></template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs no-wrap">
          <q-btn v-for="button in availableLineButtons(props.row)" :key="button.id" flat dense size="sm" v-bind="menuButtonDisplayProps(button)" :color="button.color || 'primary'" @click="handleButtonClick(button, props.row)"><q-tooltip>{{ button.name }}</q-tooltip></q-btn>
        </q-td>
      </template>
      <template #bottom><q-space /><table-pagination v-model:page="query.page" v-model:page-size="query.num" :total="total" /></template>
    </q-table>

    <advanced-query v-model="showAdvancedQuery" v-model:query-model="tempAdvancedQuery" :fields="advancedFields" @search="handleAdvancedSearch" />
    <retry-policy-form-dialog v-model="showFormDialog" :edit-data="currentEditData" :loading="loading" @submit="handleFormSubmit" />
    <retry-policy-detail-dialog v-model="showDetailDialog" :id="currentDetailId" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_retry_policy' })

import { onMounted, ref, watch, computed } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
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
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('integration_retry_policy')
const rows = ref<RetryPolicyListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showAdvancedQuery = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<RetryPolicyDetail | null>(null)
const advancedFields = ref<TableField[]>([])
const statusMeta: Record<RetryPolicyStatus, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'grey-7' }, enabled: { label: '已启用', color: 'positive' }, disabled: { label: '已停用', color: 'warning' },
}
const backoffLabels: Record<RetryBackoffType, string> = { fixed: '固定间隔', exponential: '指数退避' }
const statusOptions = Object.entries(statusMeta).map(([value, item]) => ({ label: item.label, value }))
const backoffOptions = Object.entries(backoffLabels).map(([value, label]) => ({ label, value }))
const columns: QTableProps['columns'] = [
  { name: 'policy_code', label: '重试策略', field: 'policy_code', align: 'left', sortable: true },
  { name: 'status', label: '状态', field: 'status', align: 'center', sortable: true },
  { name: 'max_attempts', label: '最大尝试次数', field: 'max_attempts', align: 'center', sortable: true },
  { name: 'backoff_type', label: '退避方式', field: 'backoff_type', align: 'center', sortable: true },
  { name: 'initial_delay_ms', label: '初始延迟', field: 'initial_delay_ms', align: 'right', sortable: true },
  { name: 'max_delay_ms', label: '最大延迟', field: 'max_delay_ms', align: 'right', sortable: true },
  { name: 'retry_window_ms', label: '重试窗口', field: 'retry_window_ms', align: 'right', sortable: true },
  { name: 'gmt_modify', label: '更新时间', field: 'gmt_modify', align: 'left', sortable: true },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]
const visibleColumns = ref(columns.map((column) => column.name))
const emptyExpressions = () => [{ rules: [{ field: '', value: null }], nested: [] }]
const query = ref<RetryPolicyQuery>({ page: 1, num: 15, order: { field: '', is_asc: false }, quick_query: { keyword: '' }, expressions: emptyExpressions() })
const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref<Query>(cloneDeep(query.value))
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })

const formatDuration = (milliseconds: number) => milliseconds >= 86400000 && milliseconds % 86400000 === 0
  ? `${milliseconds / 86400000} 天`
  : milliseconds >= 60000 && milliseconds % 60000 === 0
    ? `${milliseconds / 60000} 分钟`
    : `${milliseconds / 1000} 秒`
const policyStatusMeta = (row: RetryPolicyListItem) => statusMeta[row.status]
const policyBackoffLabel = (row: RetryPolicyListItem) => backoffLabels[row.backoff_type]
const fetchData = async () => { const response = await api.queryRetryPolicies(query.value); rows.value = response.data || []; total.value = response.total || 0 }
const fetchMetadata = async () => { const response = await tableApi.queryTableByCode('integration_retry_policy'); advancedFields.value = response.data?.table_fields || [] }
const resetAndFetch = () => { if (query.value.page !== 1) query.value.page = 1; else void fetchData() }
const handleBasicSearch = () => { query.value.expressions = emptyExpressions(); appliedAdvancedQuery.value = cloneDeep(query.value); resetAndFetch() }
const handleAdvancedSearch = () => { query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions); appliedAdvancedQuery.value = cloneDeep(query.value); showAdvancedQuery.value = false; resetAndFetch() }
const availableLineButtons = (row: RetryPolicyListItem) => line_buttons.value.filter((button) => {
  if (button.event_action === 'update') return row.status === 'draft'
  if (button.event_action === 'create_version') return row.status !== 'draft'
  if (button.event_action === 'enable') return row.status !== 'enabled'
  if (button.event_action === 'disable') return row.status === 'enabled'
  return true
})
const openDetail = (row: RetryPolicyListItem) => { currentDetailId.value = row.id; showDetailDialog.value = true }
const openEdit = async (row: RetryPolicyListItem) => { currentEditData.value = (await api.getRetryPolicy(row.id)).data || null; showFormDialog.value = true }
const createVersion = (row: RetryPolicyListItem) => {
  confirmAction({ title: '创建策略版本', message: `基于“${row.policy_name}”v${row.version} 创建下一草稿版本？`, loading: loading.value, disable: loading.value }).onOk(() => { void (async () => {
    const response = await api.createRetryPolicyVersion(row.id, row.revision)
    if (response.success) { await fetchData(); if (response.data) { currentEditData.value = response.data; showFormDialog.value = true } }
  })() })
}
const changeState = (row: RetryPolicyListItem, enable: boolean) => {
  confirmAction({ title: enable ? '确认启用' : '确认停用', message: `${enable ? '启用' : '停用'}策略“${row.policy_name}”v${row.version}？`, loading: loading.value, disable: loading.value }).onOk(() => { void (async () => {
    const response = enable ? await api.enableRetryPolicy(row.id, row.revision) : await api.disableRetryPolicy(row.id, row.revision)
    if (response.success) await fetchData()
  })() })
}
const actionHandlers: Record<string, (row?: RetryPolicyListItem) => void> = {
  create: () => { currentEditData.value = null; showFormDialog.value = true },
  detail: (row) => row && openDetail(row), update: (row) => row && void openEdit(row),
  create_version: (row) => row && createVersion(row), enable: (row) => row && changeState(row, true), disable: (row) => row && changeState(row, false),
}
const handleButtonClick = (button: MenuButton, row?: RetryPolicyListItem) => { actionHandlers[button.event_action]?.(row) }
const handleFormSubmit = async (form: RetryPolicyFormValue) => {
  if (currentEditData.value) {
    const request: RetryPolicyUpdateRequest = { ...form, revision: currentEditData.value.revision }
    if ((await api.updateRetryPolicy(currentEditData.value.id, request)).success) showFormDialog.value = false
  } else if ((await api.createRetryPolicy(form as RetryPolicyCreateRequest)).success) showFormDialog.value = false
  await fetchData()
}

onMounted(async () => { await Promise.all([fetchMetadata(), fetchData()]); if (!has_line_buttons.value) visibleColumns.value = visibleColumns.value.filter((name) => name !== 'actions'); initialized.value = true })
watch(() => [query.value.page, query.value.num] as const, ([page]) => { if (!initialized.value) return; pagination.value.page = page; void fetchData() })
watch(() => [pagination.value.sortBy, pagination.value.descending] as const, ([sortBy, descending], previous) => { if (!initialized.value || (sortBy === previous[0] && descending === previous[1])) return; query.value.order = { field: sortBy || '', is_asc: sortBy ? !descending : false }; resetAndFetch() })
watch(showAdvancedQuery, (open) => { if (open) tempAdvancedQuery.value = cloneDeep(query.value) })
</script>
