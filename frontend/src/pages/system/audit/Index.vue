<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      color="primary"
      :dense="$q.screen.lt.md"
      flat
      bordered
      separator="cell"
      row-key="id"
      v-model:pagination="pagination"
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      :loading="loading"
      hide-pagination
    >
      <template #top>
        <div class="row q-gutter-xs full-width">
          <div class="col-grow row q-gutter-xs">
            <q-input
              v-model="query.quick_query!.keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索用户、动作、资源、路径、IP"
              @keyup.enter="handleBasicSearch"
            >
              <template #append>
                <q-icon name="search" />
              </template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
            <q-select
              v-model="visibleColumns"
              multiple
              outlined
              dense
              options-dense
              :display-value="compactSelectionDisplay(visibleColumns, columns, 2, '列')"
              emit-value
              map-options
              :options="columns"
              option-value="name"
              options-cover
            />
            <q-btn
              outline
              icon="tune"
              color="primary"
              :aria-label="
                hasAppliedAdvancedFilters ? `高级查询，已启用 ${activeFilterCount} 个条件` : '高级查询'
              "
              @click="showAdvancedQuery = true"
            >
              <q-badge v-if="activeFilterCount > 0" floating color="red">{{
                activeFilterCount
              }}</q-badge>
              <q-tooltip>{{
                hasAppliedAdvancedFilters
                  ? `高级查询，已启用 ${activeFilterCount} 个条件`
                  : '高级查询'
              }}</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <q-btn flat color="primary" icon="refresh" :disable="loading" @click="fetchData">
            <q-tooltip>刷新</q-tooltip>
          </q-btn>
        </div>
      </template>

      <template #body-cell-success="props">
        <q-td :props="props">
          <q-chip
            dense
            square
            :color="props.row.success ? 'positive' : 'negative'"
            text-color="white"
          >
            {{ props.row.success ? '成功' : '失败' }}
          </q-chip>
        </q-td>
      </template>

      <template #body-cell-duration_ms="props">
        <q-td :props="props">{{ props.row.duration_ms }} ms</q-td>
      </template>

      <template #body-cell-actions="props">
        <q-td :props="props">
          <q-btn
            v-for="btn in lineButtons"
            :key="btn.id || btn.code"
            flat
            v-bind="menuButtonDisplayProps(btn)"
            :color="btn.color || 'primary'"
            size="sm"
            data-testid="audit-open-detail"
            :disable="loading"
            @click="handleLineButtonClick(btn, props.row)"
          >
            <q-tooltip>{{ btn.name }}</q-tooltip>
          </q-btn>
          <q-btn
            v-if="lineButtons.length === 0"
            flat
            round
            color="primary"
            icon="visibility"
            size="sm"
            data-testid="audit-open-detail"
            @click="openDetail(props.row)"
          >
            <q-tooltip>查看详情</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="auditAdvancedFields"
      :enable-nested="false"
      title="审计日志查询"
      @search="handleAdvancedSearch"
    />

  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_audit' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import cloneDeep from 'lodash/cloneDeep'
import { useAccessLogApi, type AccessLog } from 'src/api/services/access-log'
import type { TableField } from 'src/api/services/sys-table'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import type { Query } from 'src/types/global'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import type { MenuButton } from 'src/api/services/sys-menu'
import { countEffectiveQueryRules, hasEffectiveQueryRules } from 'src/utils/query-state'
import { metadataDictDefault } from 'src/utils/field-metadata'
import { compactSelectionDisplay } from 'src/utils/select-display'
import { usePageButtons } from 'src/composables/page-buttons'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { executeButtonAction, type ButtonActionContext } from 'src/utils/button-handlers'

const $q = useQuasar()
const router = useRouter()
const accessLogApi = useAccessLogApi()
const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const { line_buttons: lineButtons } = usePageButtons('system_audit')

const rows = ref<AccessLog[]>([])
const total = ref(0)
const showAdvancedQuery = ref(false)

const emptyAdvancedQuery = (): Query => ({
  page: 1,
  num: 20,
  expressions: [
    {
      rules: [{ field: '', value: null }],
      nested: [],
    },
  ],
})

const query = ref<Query>({
  page: 1,
  num: 20,
  order: {
    field: 'gmt_create',
    is_asc: false,
  },
  table_code: 'access_log',
  expressions: emptyAdvancedQuery().expressions,
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref<Query>(cloneDeep(emptyAdvancedQuery()))

const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))

const columns = computed<QTableProps['columns']>(() => [
  { name: 'gmt_create', field: 'gmt_create', label: '时间', align: 'left', sortable: true },
  { name: 'user_name', field: 'user_name', label: '用户', align: 'left', sortable: true },
  { name: 'action', field: 'action', label: '动作', align: 'left', sortable: true },
  {
    name: 'resource_code',
    field: 'resource_code',
    label: '资源编码',
    align: 'left',
    sortable: true,
  },
  { name: 'method', field: 'method', label: '方法', align: 'center', sortable: true },
  { name: 'url', field: 'url', label: '路径', align: 'left' },
  { name: 'status_code', field: 'status_code', label: '状态码', align: 'right', sortable: true },
  { name: 'success', field: 'success', label: '结果', align: 'center', sortable: true },
  { name: 'duration_ms', field: 'duration_ms', label: '耗时', align: 'right', sortable: true },
  { name: 'ip', field: 'ip', label: 'IP', align: 'left' },
  { name: 'actions', field: 'actions', label: '操作', align: 'center' },
])

const visibleColumns = ref([
  'gmt_create',
  'user_name',
  'action',
  'resource_code',
  'method',
  'url',
  'status_code',
  'success',
  'duration_ms',
  'ip',
  'actions',
])

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: 'gmt_create',
  descending: true,
})

const auditAdvancedFields: TableField[] = [
  buildAuditField(
    '时间',
    'gmt_create',
    SysTableFieldType.DATETIME,
    SysTableFieldInputType.DATETIME_PICKER,
  ),
  buildAuditField('用户', 'user_name', SysTableFieldType.VARCHAR),
  buildAuditField('动作', 'action', SysTableFieldType.VARCHAR),
  buildAuditField('资源类型', 'resource_type', SysTableFieldType.VARCHAR),
  buildAuditField('资源编码', 'resource_code', SysTableFieldType.VARCHAR),
  buildAuditField('资源ID', 'resource_id', SysTableFieldType.VARCHAR),
  buildAuditField('方法', 'method', SysTableFieldType.VARCHAR),
  buildAuditField('路径', 'url', SysTableFieldType.VARCHAR),
  buildAuditField('IP', 'ip', SysTableFieldType.VARCHAR),
  buildAuditField(
    '状态码',
    'status_code',
    SysTableFieldType.INT,
    SysTableFieldInputType.INPUT_NUMBER,
  ),
  buildAuditField('结果', 'success', SysTableFieldType.BOOLEAN, SysTableFieldInputType.BOOLEAN),
  buildAuditField(
    '耗时',
    'duration_ms',
    SysTableFieldType.BIGINT,
    SysTableFieldInputType.INPUT_NUMBER,
  ),
]

function buildAuditField(
  fieldName: string,
  fieldCode: string,
  fieldType: SysTableFieldType,
  inputType: SysTableFieldInputType = SysTableFieldInputType.INPUT,
): TableField {
  const dictCode = metadataDictDefault(fieldCode, fieldType)
  return {
    id: 0,
    table_id: 0,
    field_name: fieldName,
    field_code: fieldCode,
    field_type: fieldType,
    field_length: 0,
    field_decimal_length: 0,
    input_type: inputType,
    default_value: '',
    dict_code: dictCode,
    is_primary_key: fieldCode === 'id',
    is_index: false,
    is_quick_search: ['user_name', 'action', 'resource_code', 'url', 'ip'].includes(fieldCode),
    is_advanced_search: true,
    is_sort: ['gmt_create', 'status_code', 'success', 'duration_ms'].includes(fieldCode),
    is_null: true,
    is_list_show: true,
    is_insert_show: false,
    is_update_show: false,
    sequence: 0,
    original_field_id: 0,
    binding: '',
  }
}

const initTempQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
}

const resetToFirstPageOrFetch = () => {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  void fetchData()
}

const handleBasicSearch = () => {
  query.value.expressions = emptyAdvancedQuery().expressions
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
}

const handleAdvancedSearch = () => {
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
  showAdvancedQuery.value = false
}

const fetchData = async () => {
  const response = await accessLogApi.queryAccessLogs(query.value)
  if (response.success) {
    rows.value = response.data || []
    total.value = response.total || 0
  }
}

const openDetail = async (row: AccessLog) => {
  await router.push({
    name: 'record_detail',
    params: {
      source: 'audit',
      table_code: 'access_log',
      id: row.id,
    },
  })
}

const handleLineButtonClick = async (button: MenuButton, row: AccessLog) => {
  const ctx: ButtonActionContext = {
    table_code: 'access_log',
    row,
    onOpenDetail: (dataRow) => openDetail(dataRow as AccessLog),
    onRefresh: fetchData,
  }
  await executeButtonAction(button.event_action || '', ctx)
}

watch(
  () => [query.value.page, query.value.num] as const,
  ([page]) => {
    pagination.value.page = page
    void fetchData()
  },
)

watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([sortBy, descending], [prevSortBy, prevDescending]) => {
    if (sortBy === prevSortBy && descending === prevDescending) return
    query.value.order = query.value.order ?? { field: 'gmt_create', is_asc: false }
    query.value.order.field = sortBy || 'gmt_create'
    query.value.order.is_asc = sortBy ? !descending : false
    resetToFirstPageOrFetch()
  },
)

watch(
  () => showAdvancedQuery.value,
  (isOpen) => {
    if (isOpen) {
      initTempQuery()
    }
  },
)

onMounted(fetchData)
</script>
