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
                  v-model="query.quick_query!.keyword"
                  placeholder="搜索关键词"
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
      <template #no-data>
        <div class="full-width row flex-center q-pa-xl text-grey-7">{{ emptyMessage }}</div>
      </template>
    </q-table>

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
import StandardTableToolbar from 'components/Table/StandardTableToolbar.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { copyToClipboard, useQuasar } from 'quasar'
import {
  useApplicationApi,
  type Application,
  type ApplicationSecretRes,
} from 'src/api/services/application'
import type { Query } from 'src/types/global'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import QuerySchemeControls from 'src/components/QueryScheme/QuerySchemeControls.vue'
import { useDictStore } from 'src/stores/dict'
import { buildRelationLookups, resolveRuntimeColumns } from 'src/utils/column-format'
import { usePageButtons } from 'src/composables/page-buttons'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import type { MenuButton } from 'src/api/services/sys-menu'
import { countEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { compactSelectionDisplay } from 'src/utils/select-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { dispatchPageAction, type PageActionHandlers } from 'src/utils/button-actions'
import { resolveTableEmptyMessage } from 'src/utils/table-state'
import type { TableColumn } from 'src/types/global'

const loading = ref(false)
const loadError = ref('')
const dictStore = useDictStore()

const $q = useQuasar()
const { confirmAction, confirmDanger } = useConfirmDialog($q)
const applicationApi = useApplicationApi()
const rows = ref<Application[]>([])
const total = ref(0)
const selected = ref([])
const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('system_application')

const action_handlers: PageActionHandlers<Application> = {
  create: () => openAddDialog(),
  update: (row) => {
    if (row) openEditDialog(row)
  },
  rotate_secret: (row) => row && confirmRotateSecret(row),
  delete: (row) => row && confirmDelete(row),
}

const handleButtonClick = (btn: MenuButton, row?: Application) => {
  dispatchPageAction(btn, action_handlers, row)
}

// 表单对话框相关
const showFormDialog = ref(false)
const currentEditData = ref<Application | null>(null)
const showApplicationSecretDialog = ref(false)
const applicationSecretValue = ref<ApplicationSecretRes | null>(null)
let applicationSecretClearTimer: ReturnType<typeof setTimeout> | null = null

// 表格列定义
const columns = ref<TableColumn<Application>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const {
  fields: metadataFields,
  advancedSearchFields: table_fields_advanced,
  formFields: tableFields,
  loadMetadata,
} = useRuntimeTableMetadata('application')

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
const queryState = useTableQueryState<Query>({
  createInitialQuery: () => ({
    page: 1,
    num: 15,
    order: { field: '', is_asc: false },
    table_code: 'application',
    expressions: emptyAdvancedQuery().expressions,
    quick_query: { keyword: '' },
    include_deleted: false,
  }),
})
const { query, appliedAdvanced: appliedAdvancedQuery } = queryState

// 计算活跃的筛选条件数量
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: true,
    error: loadError.value,
    hasQuery: !!query.value.quick_query?.keyword || activeFilterCount.value > 0,
  }),
)

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: true,
})

const resetAndFetch = () => {
  if (pagination.value.page !== 1) {
    pagination.value.page = 1
    return
  }
  fetchData()
}

const schemePage = useQuerySchemePage('system_application', queryState, resetAndFetch)

// 基础查询处理
const handleBasicSearch = () => {
  // 基本查询时重置高级查询部分，保留基本的关键字查询
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

// 获取应用列表数据
const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await applicationApi.queryApplication(query.value)
    rows.value = res.data
    total.value = res.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '应用列表加载失败'
  } finally {
    loading.value = false
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

    const resolution = resolveRuntimeColumns<Application>(metadataFields.value, {
      context: { getDictLabel: dictStore.getDictLabel, relationLookups },
      virtualColumns: [
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
}

const initialized = ref(false)

onMounted(async () => {
  await fetchTableFields()
  await schemePage.initialize()
  await fetchData()
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
