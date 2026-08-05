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
                hasAppliedAdvancedFilters
                  ? `高级查询，已启用 ${activeFilterCount} 个条件`
                  : '高级查询'
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
      :title="currentEditData ? '编辑应用' : '新增应用'"
      :fields="tableFields"
      :submit-btn-text="currentEditData ? '保存' : '创建'"
      @submit="handleFormSubmit"
    />

    <q-dialog v-model="showApplicationSecretDialog" @hide="clearApplicationSecret">
      <q-card class="application-secret-dialog">
        <q-card-section class="text-h6">应用密钥</q-card-section>
        <q-separator />
        <q-card-section class="q-gutter-md">
          <q-input
            :model-value="applicationSecretValue?.app_key || ''"
            label="App Key"
            outlined
            dense
            readonly
          >
            <template v-slot:append>
              <q-btn
                flat
                dense
                round
                color="primary"
                icon="content_copy"
                :disable="!applicationSecretValue?.app_key"
                @click="copyApplicationSecret(applicationSecretValue?.app_key)"
              >
                <q-tooltip>复制</q-tooltip>
              </q-btn>
            </template>
          </q-input>
          <q-input
            :model-value="applicationSecretValue?.app_secret || ''"
            label="App Secret"
            outlined
            dense
            readonly
          >
            <template v-slot:append>
              <q-btn
                flat
                dense
                round
                color="primary"
                icon="content_copy"
                :disable="!applicationSecretValue?.app_secret"
                @click="copyApplicationSecret(applicationSecretValue?.app_secret)"
              >
                <q-tooltip>复制</q-tooltip>
              </q-btn>
            </template>
          </q-input>
          <div class="text-caption text-grey-7">关闭后可通过轮换密钥重新生成</div>
        </q-card-section>
        <q-card-actions align="right">
          <q-btn color="primary" label="关闭" v-close-popup />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_application' })
import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import ScrollableTable from 'components/Table/ScrollableTable.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { copyToClipboard, type QTableProps, useQuasar } from 'quasar'
import {
  useApplicationApi,
  type Application,
  type ApplicationSecretRes,
} from 'src/api/services/application'
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
const { confirmAction, confirmDanger } = useConfirmDialog($q)
const applicationApi = useApplicationApi()
const tableApi = useTableApi()
const rows = ref<Application[]>([])
const total = ref(0)
const selected = ref([])
const showAdvancedQuery = ref(false)

const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('system_application')

const action_handlers: Record<string, (row?: Application) => void> = {
  create: () => openAddDialog(),
  update: (row) => {
    if (row) openEditDialog(row)
  },
  rotate_secret: (row) => row && confirmRotateSecret(row),
  delete: (row) => row && confirmDelete(row),
}

const handleButtonClick = (btn: MenuButton, row?: Application) => {
  const handler = action_handlers[btn.event_action]
  if (handler) handler(row)
}

// 表单对话框相关
const showFormDialog = ref(false)
const currentEditData = ref<Application | null>(null)
const showApplicationSecretDialog = ref(false)
const applicationSecretValue = ref<ApplicationSecretRes | null>(null)
let applicationSecretClearTimer: ReturnType<typeof setTimeout> | null = null

// 表格列定义
const columns = ref<QTableProps['columns']>([])
const table_fields_advanced = ref<TableField[]>([])
const visibleColumns = ref<string[]>([])

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
  table_code: 'application',
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

// 保存查询到的表字段，供表单使用
const tableFields = ref<TableField[]>([])

// 初始化临时查询
const initTempQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
}

const resetToFirstPageOrFetch = () => {
  if (pagination.value.page !== 1) {
    pagination.value.page = 1
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

// 获取应用列表数据
const fetchData = async () => {
  const res = await applicationApi.queryApplication(query.value)
  rows.value = res.data
  total.value = res.total || 0
}

// 获取表结构信息
const fetchTableFields = async () => {
  const res = await tableApi.queryTableByCode('application')
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

// 监听分页变化（底部分页组件会改变 query.page/query.num）
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
const openEditDialog = async (row: Application) => {
  // 获取完整的应用数据
  const response = await applicationApi.queryApplicationById(row.id)
  if (response.data) {
    currentEditData.value = response.data
    showFormDialog.value = true
  }
}

// 确认删除
const confirmDelete = (row: Application) => {
  confirmDanger({
    message: `确定要删除应用 "${row.name}" 吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await applicationApi.deleteApplication(row.id)
      if (result) {
        await fetchData() // 刷新列表
      }
    })()
  })
}

const confirmRotateSecret = (row: Application) => {
  confirmAction({
    title: '确认轮换密钥',
    message: `确定要轮换应用 "${row.name}" 的 App Secret 吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await applicationApi.rotateApplicationSecret(row.id)
      if (result.success) {
        showApplicationSecret(result.data)
        await fetchData()
      }
    })()
  })
}

const showApplicationSecret = (value: ApplicationSecretRes) => {
  clearApplicationSecretTimer()
  applicationSecretValue.value = value
  showApplicationSecretDialog.value = true
  applicationSecretClearTimer = setTimeout(
    () => {
      showApplicationSecretDialog.value = false
      clearApplicationSecret()
    },
    2 * 60 * 1000,
  )
}

const clearApplicationSecretTimer = () => {
  if (applicationSecretClearTimer) {
    clearTimeout(applicationSecretClearTimer)
    applicationSecretClearTimer = null
  }
}

const clearApplicationSecret = () => {
  clearApplicationSecretTimer()
  applicationSecretValue.value = null
}

const copyApplicationSecret = async (value?: string) => {
  if (!value) return
  try {
    await copyToClipboard(value)
    $q.notify({ type: 'positive', position: 'top-right', message: '已复制到剪贴板' })
  } catch (error) {
    console.error('复制失败', error)
    $q.notify({ type: 'negative', position: 'top-right', message: '复制失败' })
  }
}

// 处理表单提交
const handleFormSubmit = async (formPayload: {
  data: Application & { ding_secret?: string }
  isEdit: boolean
  id?: number
}) => {
  if (formPayload.isEdit && formPayload.id) {
    // 编辑模式
    const result = await applicationApi.updateApplication(formPayload.id, {
      name: formPayload.data.name,
      expiration: formPayload.data.expiration,
      ding_key: formPayload.data.ding_key || '',
      ding_secret: formPayload.data.ding_secret || '',
      ding_app_id: formPayload.data.ding_app_id || '',
      remark: formPayload.data.remark || '',
    })
    if (result.success) {
      showFormDialog.value = false
      await fetchData() // 刷新列表
    }
  } else {
    // 新建模式
    const result = await applicationApi.createApplication({
      name: formPayload.data.name,
      expiration: formPayload.data.expiration || 0,
      ding_key: formPayload.data.ding_key || '',
      ding_secret: formPayload.data.ding_secret || '',
      ding_app_id: formPayload.data.ding_app_id || '',
      remark: formPayload.data.remark || '',
    })
    if (result.success) {
      showFormDialog.value = false
      showApplicationSecret(result.data)
      await fetchData() // 刷新列表
    }
  }
}
</script>

<style scoped lang="scss">
.application-secret-dialog {
  min-width: min(520px, 92vw);
}
</style>
