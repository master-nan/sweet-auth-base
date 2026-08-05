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
            />
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
            <q-btn
              color="primary"
              outline
              icon="refresh"
              label="刷新"
              :disable="loading"
              @click="fetchData"
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
      :title="currentEditData?.id ? '编辑用户' : '新增用户'"
      :fields="tableFields"
      :submit-btn-text="currentEditData?.id ? '保存' : '创建'"
      @submit="handleFormSubmit"
    />

    <q-dialog v-model="showRoleDialog" @hide="clearRoleDialog">
      <q-card style="min-width: 520px; max-width: 90vw">
        <q-card-section>
          <div class="text-h6">分配角色</div>
          <div class="text-caption text-grey-7">
            {{ currentRoleUser?.user_name || '' }}
          </div>
        </q-card-section>
        <q-separator />
        <q-card-section>
          <q-select
            v-model="selectedRoleIds"
            :options="roleOptions"
            :loading="roleDialogLoading"
            label="角色"
            outlined
            dense
            multiple
            emit-value
            map-options
            use-chips
            options-dense
            option-label="name"
            option-value="id"
            :rules="[(val) => (val && val.length > 0) || '至少选择一个角色']"
          />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat label="取消" v-close-popup />
          <q-btn
            color="primary"
            icon="save"
            label="保存"
            :loading="roleDialogLoading"
            :disable="selectedRoleIds.length === 0"
            @click="saveUserRoles"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="showResetPasswordDialog" @hide="clearResetPassword">
      <q-card style="min-width: 520px; max-width: 90vw">
        <q-card-section class="text-h6">临时密码已生成</q-card-section>
        <q-separator />
        <q-card-section>
          <div class="row items-center no-wrap q-gutter-sm q-mb-xs">
            <div class="text-body1">临时密码：{{ resetPasswordValue }}</div>
            <q-btn
              flat
              dense
              round
              color="primary"
              icon="content_copy"
              :disable="!resetPasswordValue"
              @click="copyResetPassword"
            >
              <q-tooltip>复制</q-tooltip>
            </q-btn>
          </div>
          <q-banner
            v-if="resetPasswordEmailMessage"
            dense
            rounded
            class="q-mt-sm"
            :class="
              resetPasswordEmailSent ? 'bg-green-1 text-positive' : 'bg-orange-1 text-warning'
            "
          >
            <template #avatar>
              <q-icon :name="resetPasswordEmailSent ? 'mark_email_read' : 'report_problem'" />
            </template>
            {{ resetPasswordEmailMessage }}
          </q-banner>
          <div class="text-caption text-grey-7">该用户下次登录必须修改密码</div>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn color="primary" label="关闭" v-close-popup />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_user' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import ScrollableTable from 'components/Table/ScrollableTable.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'

import { computed, ref, watch, onMounted } from 'vue'
import { copyToClipboard, type QTableProps, useQuasar } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'

import type { Query } from 'src/types/global'
import {
  useSysUserApi,
  type User,
  type UserCreateReq,
  type UserUpdateReq,
} from 'src/api/services/sys-user'
import { useRoleApi, type Role } from 'src/api/services/sys-role'
import { useTableApi, type TableField } from 'src/api/services/sys-table'

import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import { useDictStore } from 'src/stores/dict'
import { buildTableColumns, buildRelationLookups } from 'src/utils/column-format'
import { usePageButtons } from 'src/composables/page-buttons'
import type { MenuButton } from 'src/api/services/sys-menu'
import { countEffectiveQueryRules, hasEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { compactSelectionDisplay } from 'src/utils/select-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useRouter } from 'vue-router'

const $q = useQuasar()
const { confirmAction, confirmDanger } = useConfirmDialog($q)
const dictStore = useDictStore()
const router = useRouter()

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const sysUserApi = useSysUserApi()
const roleApi = useRoleApi()
const tableApi = useTableApi()

const rows = ref<User[]>([])
const total = ref(0)
const selected = ref([])

const showAdvancedQuery = ref(false)

const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('system_user')

const action_handlers: Record<string, (row?: User) => void> = {
  create: () => openAddDialog(),
  update: (row) => {
    if (row) openEditDialog(row)
  },
  detail: (row) => row && openRecordDetail('sys_user', row.id),
  delete: (row) => row && confirmDelete(row),
  reset_password: (row) => row && confirmResetPassword(row),
  unlock_login: (row) => row && confirmUnlockLogin(row),
  assign_role: (row) => row && openRoleDialog(row),
}

const handleButtonClick = (btn: MenuButton, row?: User) => {
  const action = (btn.event_action || '').trim()
  const handler = action_handlers[action]
  if (handler) handler(row)
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

const showFormDialog = ref(false)
const currentEditData = ref<User | null>(null)

const showResetPasswordDialog = ref(false)
const resetPasswordValue = ref('')
const resetPasswordEmailSent = ref(false)
const resetPasswordEmailMessage = ref('')
let resetPasswordClearTimer: ReturnType<typeof setTimeout> | null = null
const showRoleDialog = ref(false)
const roleDialogLoading = ref(false)
const currentRoleUser = ref<User | null>(null)
const roleOptions = ref<Role[]>([])
const selectedRoleIds = ref<number[]>([])

const columns = ref<QTableProps['columns']>([])
const table_fields_advanced = ref<TableField[]>([])
const visibleColumns = ref<string[]>([])
const tableFields = ref<TableField[]>([])

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

const query = ref<Query>({
  page: 1,
  num: 15,
  order: {
    field: '',
    is_asc: false,
  },
  table_code: 'sys_user',
  expressions: emptyAdvancedQuery().expressions,
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref(cloneDeep(emptyAdvancedQuery()))

const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: false,
})

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
  const res = await sysUserApi.queryUser(query.value)
  rows.value = res.data
  total.value = res.total || 0
}

const openAddDialog = () => {
  currentEditData.value = null
  showFormDialog.value = true
}

const openEditDialog = async (row: User) => {
  try {
    const res = await sysUserApi.queryUserById(row.id)
    if (res.success && res.data) {
      currentEditData.value = res.data
      showFormDialog.value = true
    }
  } catch (error) {
    console.error('获取用户详情失败', error)
  }
}

const confirmDelete = (row: User) => {
  confirmDanger({
    message: `确定要删除用户 "${row.user_name}" 吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await sysUserApi.deleteUser(row.id)
      if (result.success) {
        fetchData()
      }
    })()
  })
}

const confirmResetPassword = (row: User) => {
  confirmAction({
    title: '确认重置密码',
    message: `确定要重置用户 "${row.user_name}" 的密码吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await sysUserApi.resetPassword(row.id)
      if (result.success) {
        showTemporaryPassword(
          String(result.data?.temporary_password ?? ''),
          Boolean(result.data?.email_sent),
          String(result.data?.email_message ?? ''),
        )
        showResetPasswordDialog.value = true
        await fetchData()
      }
    })()
  })
}

const showTemporaryPassword = (password: string, emailSent: boolean, emailMessage: string) => {
  clearResetPasswordTimer()
  resetPasswordValue.value = password
  resetPasswordEmailSent.value = emailSent
  resetPasswordEmailMessage.value = emailMessage
  resetPasswordClearTimer = setTimeout(() => {
    showResetPasswordDialog.value = false
    clearResetPassword()
  }, 60 * 1000)
}

const clearResetPasswordTimer = () => {
  if (resetPasswordClearTimer) {
    clearTimeout(resetPasswordClearTimer)
    resetPasswordClearTimer = null
  }
}

const clearResetPassword = () => {
  clearResetPasswordTimer()
  resetPasswordValue.value = ''
  resetPasswordEmailSent.value = false
  resetPasswordEmailMessage.value = ''
}

const copyResetPassword = async () => {
  try {
    await copyToClipboard(resetPasswordValue.value)
    $q.notify({ type: 'positive', position: 'top-right', message: '已复制到剪贴板' })
  } catch (error) {
    console.error('复制失败', error)
    $q.notify({ type: 'negative', position: 'top-right', message: '复制失败' })
  }
}

const confirmUnlockLogin = (row: User) => {
  confirmAction({
    title: '解除登录锁定',
    message: `确定要清除用户 "${row.user_name}" 的登录失败次数和锁定状态吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await sysUserApi.unlockLogin(row.id)
      if (result.success) {
        $q.notify({ type: 'positive', position: 'top-right', message: '登录锁定已解除' })
        await fetchData()
      }
    })()
  })
}

const openRoleDialog = async (row: User) => {
  roleDialogLoading.value = true
  showRoleDialog.value = true
  try {
    currentRoleUser.value = row
    const [userRes, roleRes] = await Promise.all([
      sysUserApi.queryUserById(row.id),
      roleApi.queryRole({
        page: 1,
        num: 1000,
        table_code: 'sys_role',
        expressions: emptyAdvancedQuery().expressions,
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    ])
    if (roleRes.success) {
      roleOptions.value = roleRes.data || []
    }
    if (userRes.success && userRes.data) {
      currentRoleUser.value = userRes.data
      selectedRoleIds.value = (userRes.data.roles || []).map((role) => role.id)
    }
  } catch (error) {
    console.error('加载用户角色失败', error)
    $q.notify({ type: 'negative', position: 'top-right', message: '加载用户角色失败' })
  } finally {
    roleDialogLoading.value = false
  }
}

const saveUserRoles = async () => {
  if (!currentRoleUser.value) return
  roleDialogLoading.value = true
  try {
    const result = await sysUserApi.assignRoles(currentRoleUser.value.id, {
      role_ids: selectedRoleIds.value,
    })
    if (result.success) {
      $q.notify({ type: 'positive', position: 'top-right', message: '角色已保存' })
      showRoleDialog.value = false
      await fetchData()
    }
  } catch (error) {
    console.error('保存用户角色失败', error)
  } finally {
    roleDialogLoading.value = false
  }
}

const clearRoleDialog = () => {
  currentRoleUser.value = null
  selectedRoleIds.value = []
}

const handleFormSubmit = async (formPayload: { data: User; isEdit: boolean; id?: number }) => {
  try {
    if (formPayload.isEdit && formPayload.id) {
      const req: UserUpdateReq = {
        id: formPayload.id,
        user_name: formPayload.data.user_name,
        email: formPayload.data.email,
        phone_number: formPayload.data.phone_number,
        access_tokens: (formPayload.data as any).access_tokens,
        gmt_last_login: formPayload.data.gmt_last_login || null,
      }
      if (formPayload.data.is_reset !== undefined) {
        req.is_reset = formPayload.data.is_reset
      }
      const result = await sysUserApi.updateUser(req)
      if (result.success) {
        showFormDialog.value = false
        await fetchData()
      }
    } else {
      const req: UserCreateReq = {
        user_name: formPayload.data.user_name,
        password: (formPayload.data as any).password,
        email: formPayload.data.email,
        phone_number: formPayload.data.phone_number,
      }
      const result = await sysUserApi.createUser(req)
      if (result.success) {
        showFormDialog.value = false
        await fetchData()
      }
    }
  } catch (error) {
    console.error('用户表单提交失败', error)
  }
}

// 获取表结构信息
const fetchTableFields = async () => {
  const res = await tableApi.queryTableByCode('sys_user')
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

watch(
  () => showAdvancedQuery.value,
  (isOpen) => {
    if (isOpen) {
      initTempQuery()
    }
  },
)
</script>
