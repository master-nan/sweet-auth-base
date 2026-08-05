<template>
  <base-content class="q-pa-sm">
    <scrollable-table
      class="fit"
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
    >
      <template v-slot:top>
        <div class="row q-gutter-xs full-width">
          <div class="col-grow row q-gutter-xs">
            <q-input
              dense
              outlined
              debounce="300"
              v-model="query.quick_query!.keyword"
              placeholder="搜索关键词"
            >
              <template v-slot:append>
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
            ></q-select>
            <q-btn
              outline
              icon="tune"
              color="primary"
              class="q-ml-xs"
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

          <div class="row q-gutter-xs">
            <q-btn
              v-for="btn in top_buttons"
              :key="btn.id"
              v-bind="menuButtonDisplayProps(btn)"
              :color="btn.color || 'primary'"
              :disable="loading"
              @click="handleButtonClick(btn)"
            />
          </div>
        </div>
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
    </scrollable-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="table_fields_advanced"
      @search="handleAdvancedSearch"
    />

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
import ScrollableTable from 'components/Table/ScrollableTable.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useSmsApi, type SmsTemplate } from 'src/api/services/sms'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import type { Query } from 'src/types/global'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import cloneDeep from 'lodash/cloneDeep'
import { useDictStore } from 'src/stores/dict'
import { buildTableColumns, buildRelationLookups } from 'src/utils/column-format'
import { usePageButtons } from 'src/composables/page-buttons'
import type { MenuButton } from 'src/api/services/sys-menu'
import { countEffectiveQueryRules, hasEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { compactSelectionDisplay } from 'src/utils/select-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)
const dictStore = useDictStore()

const $q = useQuasar()
const { confirmDanger } = useConfirmDialog($q)
const smsApi = useSmsApi()
const tableApi = useTableApi()
const rows = ref<SmsTemplate[]>([])
const total = ref(0)
const selected = ref([])
const showAdvancedQuery = ref(false)

const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('system_sms')

const action_handlers: Record<string, (row?: SmsTemplate) => void> = {
  create: () => openAddDialog(),
  update: (row) => {
    if (row) openEditDialog(row)
  },
  delete: (row) => row && confirmDelete(row),
}

const handleButtonClick = (btn: MenuButton, row?: SmsTemplate) => {
  const handler = action_handlers[btn.event_action]
  if (handler) handler(row)
}

// 表单对话框相关
const showFormDialog = ref(false)
const currentEditData = ref<SmsTemplate | null>(null)

// 表格列定义
const columns = ref<QTableProps['columns']>([])
const table_fields_advanced = ref<TableField[]>([])
const visibleColumns = ref<string[]>([])
// 保存查询到的表字段，供表单使用
const tableFields = ref<TableField[]>([])

// 默认空查询
const emptyAdvancedQuery = (): Query => ({
  page: 1,
  num: 15,
  expressions: [
    {
      rules: [{ field: '', value: null }],
      nested: [],
    },
  ],
})
// 查询参数
const query = ref<Query>({
  page: 1,
  num: 15,
  order: {
    field: '',
    is_asc: false,
  },
  table_code: 'sms_template',
  expressions: emptyAdvancedQuery().expressions,
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

// 临时高级查询条件（在对话框中编辑但未应用）
const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))

// 跟踪已应用的高级查询条件
const appliedAdvancedQuery = ref(cloneDeep(emptyAdvancedQuery()))

// 判断是否存在已应用的高级查询条件
const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

// 计算活跃的筛选条件数量
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: true,
})

// 初始化临时查询
const initTempQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
}

const resetToFirstPageOrFetch = () => {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  fetchData()
}

// 基础查询处理
const handleBasicSearch = () => {
  // 基本查询时重置高级查询部分，保留基本的关键字查询
  query.value.expressions = emptyAdvancedQuery().expressions
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
}

// 高级查询处理
const handleAdvancedSearch = () => {
  // 应用临时查询条件到实际查询
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)

  // 更新已应用的高级查询状态
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
  showAdvancedQuery.value = false
}

// 获取短信模板列表数据
const fetchData = async () => {
  const res = await smsApi.querySmsTemplate(query.value)
  rows.value = res.data
  total.value = res.total || 0
}

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
  const res = await tableApi.queryTableByCode('sms_template')
  if (res.data && res.data.table_fields) {
    tableFields.value = res.data.table_fields
    const dictCodes = res.data.table_fields.map((f) => f.dict_code).filter((c): c is string => !!c)
    const [, relationLookups] = await Promise.all([
      dictStore.loadDicts(dictCodes),
      buildRelationLookups(res.data.table_fields),
    ])

    const { columns: cols, advancedFields } = buildTableColumns(res.data.table_fields, {
      getDictLabel: dictStore.getDictLabel,
      relationLookups,
    })
    columns.value = cols
    table_fields_advanced.value = advancedFields
    visibleColumns.value = cols.map((c) => c.name)
    if (!has_line_buttons.value) {
      visibleColumns.value = visibleColumns.value.filter((c) => c !== 'actions')
    }
    await fetchData()
  }
}

const initialized = ref(false)

onMounted(async () => {
  await fetchTableFields()
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
    query.value.order = query.value.order ?? { field: '', is_asc: false }
    query.value.order.field = sortBy || ''
    query.value.order.is_asc = sortBy ? !descending : false

    // 排序变化时，自动回到第1页
    if (query.value.page !== 1) {
      query.value.page = 1
      // 回到第1页会触发上面的 watch，所以这里 return
      return
    }

    fetchData()
  },
)

// 监听高级查询对话框打开状态，打开时初始化临时查询
watch(
  () => showAdvancedQuery.value,
  (isOpen) => {
    if (isOpen) {
      initTempQuery()
    }
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
