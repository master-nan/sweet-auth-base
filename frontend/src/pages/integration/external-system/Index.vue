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
        <div class="row q-gutter-xs full-width items-center">
          <div class="col-grow row q-gutter-xs items-center">
            <q-input
              v-model="query.quick_query!.keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索关键词"
              @keyup.enter="handleBasicSearch"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
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
            <q-btn
              outline
              icon="tune"
              color="primary"
              :aria-label="activeFilterCount ? `高级查询，已启用 ${activeFilterCount} 个条件` : '高级查询'"
              @click="showAdvancedQuery = true"
            >
              <q-badge v-if="activeFilterCount" floating color="red">{{ activeFilterCount }}</q-badge>
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <div class="row q-gutter-xs">
            <q-btn
              v-for="button in top_buttons"
              :key="button.id"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :disable="loading"
              @click="handleButtonClick(button)"
            />
          </div>
        </div>
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
          <q-chip
            dense
            square
            outline
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

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:query-model="tempAdvancedQuery"
      :fields="advancedFields"
      @search="handleAdvancedSearch"
    />

    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="currentEditData ? '编辑外部系统' : '新增外部系统'"
      :fields="formFields"
      :submit-btn-text="currentEditData ? '保存' : '创建'"
      @submit="handleFormSubmit"
    />

    <external-system-detail-dialog
      v-model="showDetailDialog"
      :id="currentDetailId"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_external_system' })

import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import ExternalSystemDetailDialog from './ExternalSystemDetailDialog.vue'
import {
  type ExternalSystemCreateRequest,
  type ExternalSystemDetail,
  type ExternalSystemListItem,
  type ExternalSystemQuery,
  type ExternalSystemUpdateRequest,
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
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('integration_external_system')

const rows = ref<ExternalSystemListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showAdvancedQuery = ref(false)
const showFormDialog = ref(false)
const showDetailDialog = ref(false)
const currentDetailId = ref(0)
const currentEditData = ref<ExternalSystemDetail | null>(null)
const formFields = ref<TableField[]>([])
const advancedFields = ref<TableField[]>([])

const typeLabels: Record<string, string> = {
  hr: '人力资源系统',
  erp: '企业资源计划',
  tms: '运输管理系统',
  wms: '仓储管理系统',
  other: '其他系统',
}
const statusMeta: Record<string, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'grey-7' },
  enabled: { label: '已启用', color: 'positive' },
  disabled: { label: '已停用', color: 'warning' },
}

const columns: QTableProps['columns'] = [
  { name: 'system_code', label: '外部系统', field: 'system_code', align: 'left', sortable: true },
  { name: 'system_type', label: '类型', field: 'system_type', align: 'left', sortable: true },
  { name: 'base_url_summary', label: '地址摘要', field: 'base_url_summary', align: 'left' },
  { name: 'owner_name', label: '负责人', field: 'owner_name', align: 'left', sortable: true },
  { name: 'status', label: '状态', field: 'status', align: 'center', sortable: true },
  { name: 'gmt_modify', label: '更新时间', field: 'gmt_modify', align: 'left', sortable: true },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]
const visibleColumns = ref(columns.map((column) => column.name))

const emptyExpressions = () => [
  { rules: [{ field: '', value: null }], nested: [] },
]
const query = ref<ExternalSystemQuery>({
  page: 1,
  num: 15,
  order: { field: '', is_asc: false },
  quick_query: { keyword: '' },
  expressions: emptyExpressions(),
})
const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref<Query>(cloneDeep(query.value))
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })

const fetchData = async () => {
  const response = await api.queryExternalSystems(query.value)
  rows.value = response.data || []
  total.value = response.total || 0
}

const fetchMetadata = async () => {
  const response = await tableApi.queryTableByCode('integration_external_system')
  const fields = response.data?.table_fields || []
  formFields.value = fields
  advancedFields.value = fields.filter((field) => field.is_advanced_search)
}

const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}

const handleBasicSearch = () => {
  query.value.expressions = emptyExpressions()
  appliedAdvancedQuery.value = cloneDeep(query.value)
  resetAndFetch()
}

const handleAdvancedSearch = () => {
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  appliedAdvancedQuery.value = cloneDeep(query.value)
  showAdvancedQuery.value = false
  resetAndFetch()
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

const openEdit = async (row: ExternalSystemListItem) => {
  const response = await api.getExternalSystem(row.id)
  currentEditData.value = response.data
  showFormDialog.value = true
}

const changeState = (row: ExternalSystemListItem, enable: boolean) => {
  confirmAction({
    title: enable ? '确认启用' : '确认停用',
    message: `${enable ? '启用' : '停用'}外部系统“${row.name}”？`,
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

const actionHandlers: Record<string, (row?: ExternalSystemListItem) => void> = {
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
  actionHandlers[button.event_action]?.(row)
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
  await Promise.all([fetchMetadata(), fetchData()])
  if (!has_line_buttons.value) {
    visibleColumns.value = visibleColumns.value.filter((name) => name !== 'actions')
  }
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
    query.value.order = { field: sortBy || '', is_asc: sortBy ? !descending : false }
    resetAndFetch()
  },
)

watch(showAdvancedQuery, (open) => {
  if (open) tempAdvancedQuery.value = cloneDeep(query.value)
})
</script>
