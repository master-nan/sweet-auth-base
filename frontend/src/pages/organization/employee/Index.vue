<template>
  <base-content :scrollable="showDetailDialog && detailMode === 'page'" class="q-pa-sm">
    <q-table
      v-if="!showDetailDialog || detailMode === 'dialog'"
      class="fit sticky-header-table"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :loading="loading"
      :pagination="{ rowsPerPage: 0 }"
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
              placeholder="搜索关键词"
              @keyup.enter="search"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="search" />
            <q-btn outline color="primary" icon="tune" @click="openAdvancedQuery">
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <div class="row q-gutter-xs">
            <q-btn
              v-for="button in refreshButtons"
              :key="button.id || button.code"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :disable="loading"
              @click="fetchData"
            />
          </div>
        </div>
      </template>

      <template #body-cell-employment_status="props">
        <q-td :props="props">
          <q-chip
            dense
            square
            outline
            :color="organizationStatusColor(props.row.employment_status)"
          >
            {{ dictLabel('org_employment_status', props.row.employment_status) }}
          </q-chip>
        </q-td>
      </template>

      <template #body-cell-binding_status="props">
        <q-td :props="props">
          <q-chip
            dense
            square
            outline
            :color="props.row.binding_status === 'bound' ? 'positive' : 'grey-7'"
          >
            {{ dictLabel('org_user_binding_status', props.row.binding_status) }}
          </q-chip>
        </q-td>
      </template>

      <template #body-cell-validity="props">
        <q-td :props="props">
          {{ formatOrganizationDate(props.row.valid_from) }}
          <span class="text-grey-6 q-mx-xs">至</span>
          {{ formatOrganizationDate(props.row.valid_to, '长期') }}
        </q-td>
      </template>

      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="button in visibleRowButtons(props.row)"
            :key="button.id || button.code"
            flat
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            size="sm"
            @click="handleRowAction(button, props.row)"
          >
            <q-tooltip>{{ button.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template #no-data>
        <div class="full-width row flex-center q-pa-xl text-grey-7">
          {{ loadError || '暂无人员数据' }}
        </div>
      </template>

      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="advancedFields"
      title="人员高级查询"
      @search="applyAdvancedQuery"
    />

    <organization-record-detail-dialog
      v-model="showDetailDialog"
      :title="employeeDetail?.name || '人员详情'"
      :subtitle="employeeDetail?.employee_no || ''"
      :sections="employeeDetailSections"
      icon="badge"
      :avatar-label="employeeDetail?.name?.slice(0, 1) || ''"
      :status-label="
        employeeDetail
          ? dictLabel('org_employment_status', employeeDetail.employment_status)
          : ''
      "
      :status-color="
        employeeDetail ? organizationStatusColor(employeeDetail.employment_status) : 'positive'
      "
      :loading="detailLoading"
      :error="detailError"
      :mode="detailMode"
      :top-buttons="record_detail_top_buttons"
      :bottom-buttons="record_detail_bottom_buttons"
      :record-context="employeeDetail"
      @button-click="handleDetailAction"
    >
      <template #section="{ sectionKey }">
        <template v-if="sectionKey === 'assignments'">
          <div class="assignment-section-toolbar q-mb-md">
            <div class="text-caption text-grey-7">按时间范围查看员工全部任职，不自动选取主任职</div>
            <assignment-scope-switch
              v-model="assignmentScope"
              class="assignment-scope-control"
              :loading="assignmentLoading"
              @update:model-value="loadAssignments"
            />
          </div>
          <q-table
            flat
            bordered
            separator="cell"
            :rows="assignments"
            :columns="assignmentColumns"
            row-key="id"
            :loading="assignmentLoading"
            :pagination="{ rowsPerPage: 0 }"
            hide-bottom
          >
            <template #body-cell-assignment_type="props">
              <q-td :props="props">
                {{ dictLabel('org_assignment_type', props.row.assignment_type) }}
              </q-td>
            </template>
            <template #body-cell-validity="props">
              <q-td :props="props">
                {{ formatOrganizationDate(props.row.valid_from) }} 至
                {{ formatOrganizationDate(props.row.valid_to, '长期') }}
              </q-td>
            </template>
            <template #no-data>
              <div class="full-width text-center text-grey-7 q-pa-lg">
                当前范围暂无任职记录
              </div>
            </template>
          </q-table>
        </template>
      </template>
    </organization-record-detail-dialog>

    <q-dialog v-model="showBindDialog">
      <q-card style="width: 480px; max-width: 92vw">
        <q-card-section>
          <div class="text-h6">绑定平台账号</div>
          <div class="text-caption text-grey-7">{{ currentEmployee?.name }}</div>
        </q-card-section>
        <q-separator />
        <q-card-section>
          <q-select
            v-model="bindingUserId"
            outlined
            dense
            clearable
            use-input
            fill-input
            hide-selected
            emit-value
            map-options
            input-debounce="300"
            option-value="value"
            option-label="label"
            option-disable="disabled"
            label="平台账号"
            hint="输入账号名称搜索"
            :options="bindingUserOptions"
            :loading="bindingUserOptionsLoading"
            @filter="filterBindingUsers"
            @virtual-scroll="loadMoreBindingUsers"
          />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn v-close-popup flat label="取消" />
          <q-btn
            color="primary"
            icon="link"
            label="绑定"
            :loading="bindingLoading"
            :disable="!bindingUserId || bindingUserId <= 0"
            @click="submitBinding"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_employee' })

import cloneDeep from 'lodash/cloneDeep'
import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import {
  bindEmployeeUser,
  getEmployeeDetail,
  queryAssignments,
  queryEmployees,
  queryEmployeeUserOptions,
  unbindEmployeeUser,
  type AssignmentListItem,
  type AssignmentTimeScope,
  type EmployeeDetail,
  type EmployeeListItem,
  type EmployeeQueryRequest,
  type EmployeeUserOption,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useDictStore } from 'src/stores/dict'
import { useUserStore } from 'src/stores/user'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import OrganizationRecordDetailDialog from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import AssignmentScopeSwitch from 'src/pages/organization/employee/AssignmentScopeSwitch.vue'
import type { OrganizationDetailSection } from 'src/pages/organization/components/organization-record-detail'
import { useOrganizationDetailMode } from 'src/pages/organization/use-organization-detail-mode'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDate,
  formatOrganizationDateTime,
  organizationStatusColor,
} from 'src/pages/organization/organization-list-page'

const $q = useQuasar()
const router = useRouter()
const { confirmAction } = useConfirmDialog($q)
const dictStore = useDictStore()
const userStore = useUserStore()
const {
  line_buttons,
  top_buttons,
  record_detail_top_buttons,
  record_detail_bottom_buttons,
} = usePageButtons('organization_employee')
const detailMode = useOrganizationDetailMode('organization_employee', 'dialog')

const rows = ref<EmployeeListItem[]>([])
const canQueryEmployees = computed(() => userStore.buttons.includes('organization_employee_query'))
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const showAdvancedQuery = ref(false)
const query = ref<EmployeeQueryRequest>({
  ...createOrganizationQuery('org_employee'),
  only_effective: true,
  bound_status: 'all',
})
const tempAdvancedQuery = ref<EmployeeQueryRequest>(cloneDeep(query.value))

const showDetailDialog = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const employeeDetail = ref<EmployeeDetail | null>(null)
const currentEmployee = ref<EmployeeListItem | null>(null)
const assignments = ref<AssignmentListItem[]>([])
const assignmentLoading = ref(false)
const assignmentScope = ref<AssignmentTimeScope>('current')

const showBindDialog = ref(false)
const bindingUserId = ref<number | null>(null)
const bindingLoading = ref(false)
const bindingUserOptions = ref<EmployeeUserOption[]>([])
const bindingUserOptionsLoading = ref(false)
const bindingUserKeyword = ref('')
const bindingUserPage = ref(1)
const bindingUserTotal = ref(0)
let bindingUserRequestId = 0

const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const columns: QTableProps['columns'] = [
  { name: 'employee_no', field: 'employee_no', label: '员工编号', align: 'left', sortable: true },
  { name: 'name', field: 'name', label: '姓名', align: 'left', sortable: true },
  {
    name: 'employment_status',
    field: 'employment_status',
    label: '人员状态',
    align: 'center',
  },
  { name: 'binding_status', field: 'binding_status', label: '账号绑定', align: 'center' },
  {
    name: 'bound_account',
    field: (row: EmployeeListItem) => row.bound_account?.user_name || '-',
    label: '平台账号',
    align: 'left',
  },
  { name: 'validity', field: 'validity', label: '有效期', align: 'left' },
  { name: 'actions', field: 'actions', label: '操作', align: 'center' },
]

const assignmentColumns: QTableProps['columns'] = [
  {
    name: 'legal_entity',
    field: (row: AssignmentListItem) => row.legal_entity?.name || '-',
    label: '法人主体',
    align: 'left',
  },
  {
    name: 'org_unit',
    field: (row: AssignmentListItem) => row.org_unit?.name || '-',
    label: '组织',
    align: 'left',
  },
  {
    name: 'position',
    field: (row: AssignmentListItem) => row.position?.name || '-',
    label: '岗位',
    align: 'left',
  },
  { name: 'assignment_type', field: 'assignment_type', label: '任职类型', align: 'center' },
  { name: 'validity', field: 'validity', label: '有效期', align: 'left' },
]

const advancedFields = [
  createOrganizationField('员工编号', 'employee_no'),
  createOrganizationField('姓名', 'name'),
  createOrganizationField('人员状态', 'employment_status', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_employment_status',
  }),
  createOrganizationField('账号绑定', 'user_id', SysTableFieldType.BIGINT, {
    inputType: SysTableFieldInputType.INPUT_NUMBER,
  }),
  createOrganizationField('有效期开始', 'valid_from', SysTableFieldType.DATETIME, {
    inputType: SysTableFieldInputType.DATE_PICKER,
  }),
  createOrganizationField('有效期结束', 'valid_to', SysTableFieldType.DATETIME, {
    inputType: SysTableFieldInputType.DATE_PICKER,
  }),
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || String(value || '-')

const employeeDetailSections = computed<OrganizationDetailSection[]>(() => {
  const detail = employeeDetail.value
  const basicItems = detail
    ? [
        { label: '员工编号', value: detail.employee_no },
        { label: '姓名', value: detail.name },
        { label: '手机号', value: detail.mobile_masked ?? null },
        { label: '邮箱', value: detail.email_masked ?? null },
        {
          label: '人员状态',
          value: dictLabel('org_employment_status', detail.employment_status),
          chip: true,
          color: organizationStatusColor(detail.employment_status),
        },
        {
          label: '有效期',
          value: `${formatOrganizationDate(detail.valid_from)} 至 ${formatOrganizationDate(detail.valid_to, '长期')}`,
        },
      ]
    : []
  return [
    {
      key: 'basic',
      label: '基本资料',
      caption: '人员身份与联系方式',
      icon: 'badge',
      items: basicItems,
    },
    {
      key: 'assignments',
      label: '任职记录',
      caption: '当前、历史与未来任职',
      icon: 'work_history',
      count: assignments.value.length,
      items: [],
    },
    {
      key: 'account',
      label: '账号信息',
      caption: '当前平台账号绑定',
      icon: 'manage_accounts',
      items: detail
        ? [
            {
              label: '绑定状态',
              value: dictLabel('org_user_binding_status', detail.binding_status),
              chip: true,
              color: detail.binding_status === 'bound' ? 'positive' : 'grey-7',
            },
            {
              label: '平台账号',
              value: detail.bound_account?.user_name || '未绑定',
            },
          ]
        : [],
    },
    {
      key: 'mirror',
      label: '镜像信息',
      caption: '同步时间与平台备注',
      icon: 'sync',
      items: detail
        ? [
            { label: '最近更新时间', value: formatOrganizationDateTime(detail.gmt_modify) },
            { label: '平台备注', value: detail.local_note, fullWidth: true },
          ]
        : [],
    },
  ]
})

const visibleRowButtons = (row?: EmployeeListItem) => {
  void row
  return line_buttons.value.filter((button) => button.event_action === 'detail')
}

const search = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}

const openAdvancedQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
  showAdvancedQuery.value = true
}

const applyAdvancedQuery = () => {
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  showAdvancedQuery.value = false
  search()
}

const fetchData = async () => {
  if (!canQueryEmployees.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await queryEmployees(query.value)
    rows.value = result.items
    total.value = result.total
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '人员数据加载失败'
  } finally {
    loading.value = false
  }
}

const openDetail = async (row: EmployeeListItem) => {
  currentEmployee.value = row
  employeeDetail.value = null
  assignments.value = []
  detailError.value = ''
  detailLoading.value = true
  assignmentScope.value = 'current'
  showDetailDialog.value = true
  try {
    employeeDetail.value = await getEmployeeDetail(row.id)
    await loadAssignments()
  } catch {
    detailError.value = '人员详情加载失败'
  } finally {
    detailLoading.value = false
  }
}

const loadAssignments = async () => {
  if (!currentEmployee.value) return
  assignmentLoading.value = true
  try {
    const result = await queryAssignments({
      ...createOrganizationQuery('org_assignment'),
      employee_id: currentEmployee.value.id,
      time_scope: assignmentScope.value,
      order: { field: 'valid_from', is_asc: assignmentScope.value === 'future' },
      num: 100,
    })
    assignments.value = result.items
  } catch {
    assignments.value = []
  } finally {
    assignmentLoading.value = false
  }
}

const openBind = (row: EmployeeListItem) => {
  currentEmployee.value = row
  bindingUserId.value = null
  bindingUserOptions.value = []
  bindingUserKeyword.value = ''
  bindingUserPage.value = 1
  bindingUserTotal.value = 0
  showBindDialog.value = true
  void loadBindingUserOptions('', 1, false)
}

const loadBindingUserOptions = async (keyword: string, page: number, append: boolean) => {
  const requestId = ++bindingUserRequestId
  bindingUserOptionsLoading.value = true
  try {
    const normalizedKeyword = keyword.trim()
    const result = await queryEmployeeUserOptions(normalizedKeyword, page, 20)
    if (requestId !== bindingUserRequestId) return
    bindingUserKeyword.value = normalizedKeyword
    bindingUserPage.value = page
    bindingUserTotal.value = result.total
    if (append) {
      const existingIds = new Set(bindingUserOptions.value.map((item) => item.value))
      bindingUserOptions.value = [
        ...bindingUserOptions.value,
        ...result.items.filter((item) => !existingIds.has(item.value)),
      ]
    } else {
      bindingUserOptions.value = result.items
    }
  } catch {
    if (requestId === bindingUserRequestId && !append) {
      bindingUserOptions.value = []
      bindingUserTotal.value = 0
    }
  } finally {
    if (requestId === bindingUserRequestId) bindingUserOptionsLoading.value = false
  }
}

const filterBindingUsers = (
  value: string,
  update: (callback: () => void) => void,
  abort: () => void,
) => {
  const requestId = ++bindingUserRequestId
  const keyword = value.trim()
  bindingUserOptionsLoading.value = true
  queryEmployeeUserOptions(keyword, 1, 20)
    .then((result) => {
      update(() => {
        if (requestId !== bindingUserRequestId) return
        bindingUserKeyword.value = keyword
        bindingUserPage.value = 1
        bindingUserTotal.value = result.total
        bindingUserOptions.value = result.items
      })
    })
    .catch(() => {
      abort()
    })
    .finally(() => {
      if (requestId === bindingUserRequestId) {
        bindingUserOptionsLoading.value = false
      }
    })
}

const loadMoreBindingUsers = (details: { to: number }) => {
  if (
    bindingUserOptionsLoading.value ||
    bindingUserOptions.value.length >= bindingUserTotal.value ||
    details.to < bindingUserOptions.value.length - 1
  ) {
    return
  }
  void loadBindingUserOptions(bindingUserKeyword.value, bindingUserPage.value + 1, true)
}

const submitBinding = async () => {
  if (!currentEmployee.value || !bindingUserId.value) return
  bindingLoading.value = true
  try {
    await bindEmployeeUser(currentEmployee.value.id, bindingUserId.value)
    showBindDialog.value = false
    $q.notify({ type: 'positive', position: 'top-right', message: '账号绑定成功' })
    await fetchData()
    if (showDetailDialog.value) employeeDetail.value = await getEmployeeDetail(currentEmployee.value.id)
  } finally {
    bindingLoading.value = false
  }
}

const unbind = (row: Pick<EmployeeListItem, 'id' | 'name'>) => {
  confirmAction({
    title: '解绑平台账号',
    message: `确认解除 ${row.name} 与当前平台账号的绑定吗？`,
  }).onOk(() => {
    void (async () => {
      await unbindEmployeeUser(row.id)
      $q.notify({ type: 'positive', position: 'top-right', message: '账号解绑成功' })
      await fetchData()
      if (showDetailDialog.value) employeeDetail.value = await getEmployeeDetail(row.id)
    })()
  })
}

const handleDetailAction = (button: MenuButton) => {
  if (!currentEmployee.value) return
  if (button.event_action === 'bind_user') openBind(currentEmployee.value)
  if (button.event_action === 'unbind_user') unbind(currentEmployee.value)
  if (button.event_action === 'view_sync') {
    void router.push({
      name: 'organization_sync_error',
      query: { object_type: 'employee', local_id: String(currentEmployee.value.id) },
    })
  }
}

const handleRowAction = (button: MenuButton, row: EmployeeListItem) => {
  switch (button.event_action) {
    case 'detail':
      void openDetail(row)
      break
    case 'bind_user':
      openBind(row)
      break
    case 'unbind_user':
      unbind(row)
      break
    case 'view_sync':
      void router.push({
        name: 'organization_sync_error',
        query: { object_type: 'employee', local_id: String(row.id) },
      })
      break
  }
}

watch(
  () => [query.value.page, query.value.num],
  () => void fetchData(),
)

onMounted(async () => {
  await dictStore.loadDicts([
    'org_employment_status',
    'org_user_binding_status',
    'org_assignment_type',
  ])
  await fetchData()
})
</script>

<style scoped>
.assignment-section-toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
}

.assignment-scope-control {
  margin-left: auto;
}

@media (max-width: 720px) {
  .assignment-section-toolbar {
    align-items: stretch;
    flex-direction: column;
    gap: 10px;
  }

  .assignment-scope-control {
    margin-left: 0;
  }
}
</style>
