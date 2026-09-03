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
                <q-btn
                  color="primary"
                  :label="t('ui.search')"
                  :disable="loading"
                  @click="handleBasicSearch"
                />
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

    <!-- 高级查询对话框 -->

    <!-- 通用表单对话框 - 用于新增和编辑角色 -->
    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="currentEditData?.id ? t('ui.editRole') : t('ui.addRole')"
      :fields="tableFields"
      :submit-btn-text="currentEditData?.id ? t('ui.save') : t('ui.createRecord')"
      @submit="handleFormSubmit"
    />

    <!-- 权限分配对话框 - 这个需要特殊处理 -->
    <permission-dialog
      v-model:open="showPermissionDialog"
      :role="currentRole"
      @save="savePermission"
    />
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'system_role' })
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from '@/components/Table/TableColumnSelector.vue'
import PermissionDialog from './PermissionDialog.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useRoleApi, type Role } from '@/api/services/sys-role'
import type { Query } from '@/types/global'
import DynamicFormDialog from '@/components/FormDialog/DynamicFormDialog.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
import { useDictStore } from '@/stores/dict'
import { buildTableColumns, buildRelationLookups } from '@/utils/column-format'
import { usePageButtons } from '@/composables/page-buttons'
import { useRuntimeTableMetadata } from '@/composables/runtime-table-metadata'
import { useTableQueryState } from '@/composables/table-query-state'
import { useQuerySchemePage } from '@/composables/query-scheme-page'
import type { MenuButton } from '@/api/services/sys-menu'
import { hasEffectiveQueryRules } from '@/utils/query-state'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'
import { useConfirmDialog } from '@/composables/confirm-dialog'
import { useRouter } from 'vue-router'
import { dispatchPageAction, type PageActionHandlers } from '@/utils/button-handlers'
import { resolveTableEmptyMessage } from '@/utils/table-state'

const { t } = useI18n({ useScope: 'global' })

// 加载状态
const loading = ref(false)
const loadError = ref('')
const dictStore = useDictStore()

const $q = useQuasar()
const { confirmDanger } = useConfirmDialog($q)
const router = useRouter()
const roleApi = useRoleApi()
const rows = ref<Role[]>([])
const total = ref(0)
const selected = ref([])

const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('system_role')

const action_handlers: PageActionHandlers<Role> = {
  create: () => openAddDialog(),
  update: (row) => {
    if (row) openEditDialog(row)
  },
  detail: (row) => row && openRecordDetail('sys_role', row.id),
  delete: (row) => row && confirmDelete(row),
  assign_permission: (row) => {
    if (row) openPermissionDialog(row)
  },
}

const handleButtonClick = (btn: MenuButton, row?: Role) => {
  dispatchPageAction(btn, action_handlers, row)
}

const openRecordDetail = (tableCode: string, id: number) => {
  void router.push({
    name: 'record_detail',
    params: {
      source: 'generalization',
      table_code: tableCode,
      id,
    },
  })
}

// 表单对话框相关
const showFormDialog = ref(false)
const currentEditData = ref<Role | null>(null)

// 权限对话框相关
const showPermissionDialog = ref(false)
const currentRole = ref<Role>({
  id: 0,
  name: '',
  memo: '',
})

// 表格列定义
const columns = ref<QTableProps['columns']>([])
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: table_fields_advanced,
  formFields: tableFields,
  loadMetadata,
} = useRuntimeTableMetadata('sys_role')
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
    table_code: 'sys_role',
    expressions: emptyAdvancedQuery().expressions,
    quick_query: { keyword: '' },
    include_deleted: false,
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState

// 判断是否存在已应用的高级查询条件
const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

// 分页设置
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

const schemePage = useQuerySchemePage('system_role', queryState, resetAndFetch)

const handleBasicSearch = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

// 获取角色列表数据
const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await roleApi.queryRole(query.value)
    rows.value = res.data
    total.value = res.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.loadingOfTheRoleListFailed')
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

// 获取表结构信息
const fetchTableFields = async () => {
  if (await loadMetadata()) {
    // 加载字典数据
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

// 打开添加对话框
const openAddDialog = () => {
  currentEditData.value = null
  showFormDialog.value = true
}

// 打开编辑对话框
const openEditDialog = async (row: Role) => {
  try {
    const res = await roleApi.queryRoleById(row.id)
    if (res.success && res.data) {
      currentEditData.value = res.data
      showFormDialog.value = true
    }
  } catch (error) {
    console.error('获取角色详情失败', error)
  }
}

// 打开权限分配对话框
const openPermissionDialog = async (role: Role) => {
  const res = await roleApi.queryRoleById(role.id)
  if (res.success && res.data) {
    currentRole.value = res.data
    showPermissionDialog.value = true
  }
}

// 确认删除
const confirmDelete = (row: Role) => {
  confirmDanger({
    get message() {
      return t('ui.areYouSureYouWantToDeleteTheCharacter', { value1: row.name })
    },
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await roleApi.deleteRole(row.id)
      if (result.success) {
        fetchData() // 刷新列表
      }
    })()
  })
}

// 处理表单提交
const handleFormSubmit = async (formPayload: { data: Role; isEdit: boolean; id?: number }) => {
  try {
    if (formPayload.isEdit && formPayload.id) {
      // 编辑模式
      const result = await roleApi.updateRole({
        id: formPayload.id,
        name: formPayload.data.name,
        memo: formPayload.data.memo || '',
      })

      if (result.success) {
        showFormDialog.value = false
        fetchData() // 刷新列表
      }
    } else {
      // 新建模式
      const result = await roleApi.createRole({
        name: formPayload.data.name,
        memo: formPayload.data.memo || '',
      })

      if (result.success) {
        showFormDialog.value = false
        fetchData() // 刷新列表
      }
    }
  } catch (error) {
    console.error('角色表单提交失败', error)
  }
}

// 保存权限设置
const savePermission = async (permissionData: {
  roleId: number
  menuIds: number[]
  buttonIds: number[]
}) => {
  try {
    const result = await roleApi.assignPermissions(
      permissionData.roleId,
      permissionData.menuIds,
      permissionData.buttonIds,
    )

    if (result.success) {
      showPermissionDialog.value = false
    }
  } catch (error) {
    console.error('权限分配失败', error)
  }
}

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
</script>
