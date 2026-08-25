<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      color="primary"
      selection="multiple"
      v-model:selected="selected"
      :dense="$q.screen.lt.md"
      separator="cell"
      flat
      bordered
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      row-key="id"
      v-model:pagination="pagination"
      :loading="loading"
      :no-data-label="emptyMessage"
    >
      <template v-slot:top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #query-controls>
            <query-scheme-controls
              :controller="schemePage"
              :query-state="queryState"
              :fields="table_fields_advanced"
            >
              <template #quick-search>
                <q-input
                  dense
                  outlined
                  debounce="300"
                  v-model="keyword"
                  :placeholder="quickSearchPlaceholder"
                >
                  <template v-slot:append>
                    <q-icon name="search" />
                  </template>
                </q-input>
                <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
              </template>
            </query-scheme-controls>
          </template>

          <template #column-selector>
            <table-column-selector v-model="visibleColumns" :columns="columns" />
          </template>

          <template #right-actions>
            <q-btn
              v-for="btn in top_buttons"
              :key="btn.id"
              v-bind="menuButtonDisplayProps(btn)"
              :color="btn.color || 'primary'"
              :disable="loading"
              @click="handleButtonClick(btn)"
            />
          </template>
        </standard-table-toolbar>
      </template>
      <template v-slot:body-cell-template_params="props">
        <q-td :props="props">
          <div class="params-display">
            <q-chip
              v-for="(param, index) in formatTemplateParams(props.value)"
              :key="index"
              size="sm"
              outline
              color="primary"
              class="q-mr-xs"
            >
              {{ param }}
            </q-chip>
            <span v-if="!formatTemplateParams(props.value).length" class="text-grey">无参数</span>
          </div>
        </q-td>
      </template>
      <template v-slot:body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="btn in line_buttons"
            :key="btn.id"
            flat
            v-bind="menuButtonDisplayProps(btn)"
            :color="btn.color || 'primary'"
            size="sm"
            @click="handleButtonClick(btn, props.row)"
          >
            <q-tooltip>{{ btn.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template v-slot:bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="currentEditData ? '编辑短信模板' : '新增短信模板'"
      :fields="tableFields"
      :submit-btn-text="currentEditData ? '保存' : '创建'"
      @submit="handleFormSubmit"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_sms' })
import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from 'src/components/Table/TableColumnSelector.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useSmsApi, type SmsTemplate } from 'src/api/services/sms'
import type { Query } from 'src/types/global'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import { useDictStore } from 'src/stores/dict'
import { buildTableColumns, buildRelationLookups } from 'src/utils/column-format'
import { usePageButtons } from 'src/composables/page-buttons'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import type { MenuButton } from 'src/api/services/sys-menu'
import { hasEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-handlers'
import { resolveTableEmptyMessage } from 'src/utils/table-state'

const loading = ref(false)
const loadError = ref('')
const dictStore = useDictStore()

const $q = useQuasar()
const { confirmDanger } = useConfirmDialog($q)
const smsApi = useSmsApi()
const rows = ref<SmsTemplate[]>([])
const total = ref(0)
const selected = ref([])

const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('system_sms')

const action_handlers: PageActionHandlers<SmsTemplate> = {
  create: () => openAddDialog(),
  update: (row) => {
    if (row) openEditDialog(row)
  },
  delete: (row) => row && confirmDelete(row),
}

const handleButtonClick = (btn: MenuButton, row?: SmsTemplate) => {
  dispatchPageAction(btn, action_handlers, row)
}

// 表单对话框相关
const showFormDialog = ref(false)
const currentEditData = ref<SmsTemplate | null>(null)

// 表格列定义
const columns = ref<QTableProps['columns']>([])
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: table_fields_advanced,
  formFields: tableFields,
  loadMetadata,
} = useRuntimeTableMetadata('sms_template')
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())

// 默认空查询
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
// 查询参数
const queryState = useTableQueryState<Query>({
  createInitialQuery: () => ({
    page: 1,
    num: 20,
    order: { field: '', is_asc: false },
    table_code: 'sms_template',
    expressions: emptyAdvancedQuery().expressions,
    quick_query: { keyword: '' },
    include_deleted: false,
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState

// 判断是否存在已应用的高级查询条件
const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: true,
})

const resetAndFetch = () => {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  fetchData()
}

const schemePage = useQuerySchemePage('system_sms', queryState, resetAndFetch)

// 基础查询处理
const handleBasicSearch = () => {
  // 基本查询时重置高级查询部分，保留基本的关键字查询
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

// 获取短信模板列表数据
const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await smsApi.querySmsTemplate(query.value)
    rows.value = res.data
    total.value = res.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '短信模板加载失败'
  } finally {
    loading.value = false
  }
}
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: true,
    error: loadError.value,
    hasQuery: !!keyword.value || hasAppliedAdvancedFilters.value,
  }),
)

const formatTemplateParams = (value: any): string[] => {
  if (!value) return []

  try {
    // 如果是字符串，尝试解析为JSON
    if (typeof value === 'string') {
      try {
        const parsed = JSON.parse(value)
        return Array.isArray(parsed) ? parsed : []
      } catch {
        // 如果不是有效JSON，返回空数组
        return []
      }
    }
    // 如果已经是数组，直接使用
    else if (Array.isArray(value)) {
      return value
    }
    return []
  } catch (e) {
    console.error('Error formatting template params:', e)
    return []
  }
}

// 获取表结构信息
const fetchTableFields = async () => {
  if (await loadMetadata()) {
    const dictCodes = metadataFields.value.map((f) => f.dict_code).filter((c): c is string => !!c)
    const [, relationLookups] = await Promise.all([
      dictStore.loadDicts(dictCodes),
      buildRelationLookups(metadataFields.value),
    ])

    const { columns: cols } = buildTableColumns(metadataFields.value, {
      getDictLabel: dictStore.getDictLabel,
      relationLookups,
    })
    columns.value = cols
    visibleColumns.value = cols.map((c) => c.name)
    sortableFields.value = new Set(
      metadataFields.value.filter((field) => field.is_sort).map((field) => field.field_code),
    )
    if (!has_line_buttons.value) {
      visibleColumns.value = visibleColumns.value.filter((c) => c !== 'actions')
    }
  }
}

const initialized = ref(false)

onMounted(async () => {
  await fetchTableFields()
  await schemePage.initialize()
  await fetchData()
  initialized.value = true
})

// 监听分页变化
watch(
  () => [query.value.page, query.value.num] as const,
  ([page]) => {
    if (!initialized.value) return
    pagination.value.page = page
    fetchData()
  },
)

// 监听排序变化（表头点击会改变 pagination.sortBy/descending）
watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([sortBy, descending], [prevSortBy, prevDescending]) => {
    if (!initialized.value) return
    if (sortBy === prevSortBy && descending === prevDescending) return

    // 同步排序到 query.order
    if (!queryState.applySorting(sortBy || '', descending, sortableFields.value)) return

    // 排序变化时，自动回到第1页
    if (query.value.page !== 1) {
      query.value.page = 1
      // 回到第1页会触发上面的 watch，所以这里 return
      return
    }

    fetchData()
  },
)

// 打开添加对话框
const openAddDialog = () => {
  currentEditData.value = null
  showFormDialog.value = true
}

// 打开编辑对话框
const openEditDialog = async (row: SmsTemplate) => {
  // 获取完整的短信模板数据
  const response = await smsApi.querySmsTemplateById(row.id)
  if (response.data) {
    currentEditData.value = response.data
    showFormDialog.value = true
  }
}

// 确认删除
const confirmDelete = (row: SmsTemplate) => {
  confirmDanger({
    message: `确定要删除短信模板 "${row.template_name}" 吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await smsApi.deleteSmsTemplate(row.id)
      if (result.success) {
        fetchData() // 刷新列表
      }
    })()
  })
}

// 处理表单提交
const handleFormSubmit = async (formPayload: {
  data: SmsTemplate
  isEdit: boolean
  id?: number
}) => {
  if (!formPayload.data.template_params) {
    formPayload.data.template_params = []
  }
  if (formPayload.isEdit && formPayload.id) {
    const result = await smsApi.updateSmsTemplate(formPayload.id, {
      sign_name: formPayload.data.sign_name,
      template_code: formPayload.data.template_code,
      template_name: formPayload.data.template_name,
      template_params: formPayload.data.template_params,
    })
    if (result.success) {
      showFormDialog.value = false
      fetchData()
    }
  } else {
    const result = await smsApi.createSmsTemplate({
      sign_name: formPayload.data.sign_name,
      template_code: formPayload.data.template_code,
      template_name: formPayload.data.template_name,
      template_params: formPayload.data.template_params,
    })
    if (result.success) {
      showFormDialog.value = false
      fetchData()
    }
  }
}
</script>
