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

    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="currentEditData?.id ? t('ui.editUser') : t('ui.addUser')"
      :fields="tableFields"
      :submit-btn-text="currentEditData?.id ? t('ui.save') : t('ui.createRecord')"
      @submit="handleFormSubmit"
    />

    <q-dialog v-model="showRoleDialog" @hide="clearRoleDialog">
      <q-card style="min-width: 520px; max-width: 90vw">
        <q-card-section>
          <div class="text-h6">{{ t('ui.assignRoles') }}</div>
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
            :label="t('ui.role')"
            outlined
            dense
            multiple
            emit-value
            map-options
            use-chips
            options-dense
            option-label="name"
            option-value="id"
            :rules="[(val) => (val && val.length > 0) || t('ui.selectAtLeastOneCharacter')]"
          />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat :label="t('ui.cancel')" v-close-popup />
          <q-btn
            color="primary"
            icon="save"
            :label="t('ui.save')"
            :loading="roleDialogLoading"
            :disable="selectedRoleIds.length === 0"
            @click="saveUserRoles"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="showResetPasswordDialog" @hide="clearResetPassword">
      <q-card style="min-width: 520px; max-width: 90vw">
        <q-card-section class="text-h6">{{ t('ui.temporaryPasswordGenerated') }}</q-card-section>
        <q-separator />
        <q-card-section>
          <div class="row items-center no-wrap q-gutter-sm q-mb-xs">
            <div class="text-body1">{{ t('ui.temporaryPassword') }}{{ resetPasswordValue }}</div>
            <q-btn
              flat
              dense
              round
              color="primary"
              icon="content_copy"
              :disable="!resetPasswordValue"
              @click="copyResetPassword"
            >
              <q-tooltip>{{ t('ui.copy') }}</q-tooltip>
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
          <div class="text-caption text-grey-7">
            {{ t('ui.theUserMustChangeThePasswordForNextLogin') }}
          </div>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn color="primary" :label="t('ui.close')" v-close-popup />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'system_user' })

import BaseContent from '@/components/BaseContent/BaseContent.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
import DynamicFormDialog from '@/components/FormDialog/DynamicFormDialog.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from '@/components/Table/TableColumnSelector.vue'

import { computed, ref, watch, onMounted } from 'vue'
import { copyToClipboard, type QTableProps, useQuasar } from 'quasar'

import type { Query } from '@/types/global'
import {
  useSysUserApi,
  type User,
  type UserCreateReq,
  type UserUpdateReq,
} from '@/api/services/sys-user'
import { useRoleApi, type Role } from '@/api/services/sys-role'

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

const $q = useQuasar()
const { confirmAction, confirmDanger } = useConfirmDialog($q)
const dictStore = useDictStore()
const router = useRouter()

const loading = ref(false)
const loadError = ref('')

const sysUserApi = useSysUserApi()
const roleApi = useRoleApi()

const rows = ref<User[]>([])
const total = ref(0)
const selected = ref([])

const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('system_user')

const action_handlers: PageActionHandlers<User> = {
  create: () => openAddDialog(),
  update: (row) => {
    if (row) openEditDialog(row)
  },
  detail: (row) => row && openRecordDetail('sys_user', row.id),
  delete: (row) => row && confirmDelete(row),
  reset_password: (row) => row && confirmResetPassword(row),
  unlock_login: (row) => row && confirmUnlockLogin(row),
  assign_role: (row) => {
    if (row) void openRoleDialog(row)
  },
}

const handleButtonClick = (btn: MenuButton, row?: User) => {
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
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: table_fields_advanced,
  formFields: tableFields,
  loadMetadata,
} = useRuntimeTableMetadata('sys_user')
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())

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

const queryState = useTableQueryState<Query>({
  createInitialQuery: () => ({
    page: 1,
    num: 20,
    order: { field: 'user_name', is_asc: true },
    table_code: 'sys_user',
    expressions: emptyAdvancedQuery().expressions,
    quick_query: { keyword: '' },
    include_deleted: false,
  }),
})
const { query, keyword, appliedAdvanced: appliedAdvancedQuery } = queryState

const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: false,
})

const resetAndFetch = () => {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  fetchData()
}

const schemePage = useQuerySchemePage('system_user', queryState, resetAndFetch)

const handleBasicSearch = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await sysUserApi.queryUser(query.value)
    rows.value = res.data
    total.value = res.total || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.loadingUserListFailed')
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
    get message() {
      return t('ui.areYouSureYouWantToDeleteUser', { value1: row.user_name })
    },
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
    get title() {
      return t('ui.confirmThatThePasswordIsBeingReset')
    },
    get message() {
      return t('ui.areYouSureYouWantToResetTheUserPassword', { value1: row.user_name })
    },
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
    $q.notify({
      type: 'positive',
      position: 'top-right',
      get message() {
        return t('ui.copiedToClipboard')
      },
    })
  } catch (error) {
    console.error('复制失败', error)
    $q.notify({
      type: 'negative',
      position: 'top-right',
      get message() {
        return t('ui.copyFailed')
      },
    })
  }
}

const confirmUnlockLogin = (row: User) => {
  confirmAction({
    get title() {
      return t('ui.unlockLogin')
    },
    get message() {
      return t('ui.determinesTheLoginFailureAndLockStatusOfTheUserToBe', { value1: row.user_name })
    },
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await sysUserApi.unlockLogin(row.id)
      if (result.success) {
        $q.notify({
          type: 'positive',
          position: 'top-right',
          get message() {
            return t('ui.loginLockedUnlocked')
          },
        })
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
    $q.notify({
      type: 'negative',
      position: 'top-right',
      get message() {
        return t('ui.failedToLoadUserRoles')
      },
    })
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
      $q.notify({
        type: 'positive',
        position: 'top-right',
        get message() {
          return t('ui.roleSaved')
        },
      })
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
        ...(formPayload.data.state === undefined ? {} : { state: formPayload.data.state }),
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
