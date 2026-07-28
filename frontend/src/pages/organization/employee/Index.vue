<template>
  <base-content class="q-pa-sm">
    <q-table
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
        <div class="row q-col-gutter-sm items-center full-width">
          <div class="col-12 col-md">
            <q-input
              v-model="query.quick_query!.keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索员工编号或姓名"
              @keyup.enter="search"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
          </div>
          <div class="col-6 col-sm-auto">
            <q-select
              v-model="query.employment_status"
              dense
              outlined
              clearable
              emit-value
              map-options
              :options="employmentStatusOptions"
              label="人员状态"
            />
          </div>
          <div class="col-6 col-sm-auto">
            <q-select
              v-model="query.bound_status"
              dense
              outlined
              emit-value
              map-options
              :options="bindingStatusOptions"
              label="账号绑定"
            />
          </div>
          <div class="col-auto">
            <q-btn color="primary" icon="search" label="查询" :disable="loading" @click="search" />
          </div>
          <div class="col-auto">
            <q-btn flat round color="primary" icon="tune" @click="openAdvancedQuery">
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <div class="col-auto">
            <q-btn
              v-for="button in refreshButtons"
              :key="button.id || button.code"
              flat
              round
              :icon="button.icon || 'refresh'"
              :color="button.color || 'primary'"
              :loading="loading"
              @click="fetchData"
            >
              <q-tooltip>{{ button.name }}</q-tooltip>
            </q-btn>
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
            dense
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
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

    <q-dialog v-model="showDetailDialog">
      <q-card style="width: 960px; max-width: 94vw">
        <q-card-section class="row items-start no-wrap">
          <div>
            <div class="text-h6">{{ employeeDetail?.name || '人员详情' }}</div>
            <div class="text-caption text-grey-7">{{ employeeDetail?.employee_no || '' }}</div>
          </div>
          <q-space />
          <q-btn v-close-popup flat round dense icon="close"><q-tooltip>关闭</q-tooltip></q-btn>
        </q-card-section>
        <q-separator />

        <q-card-section v-if="detailLoading" class="row justify-center q-pa-xl">
          <q-spinner color="primary" size="32px" />
        </q-card-section>
        <q-banner v-else-if="detailError" class="text-negative">{{ detailError }}</q-banner>
        <template v-else-if="employeeDetail">
          <q-list separator>
            <q-item>
              <q-item-section>
                <q-item-label caption>人员状态</q-item-label>
                <q-item-label>{{
                  dictLabel('org_employment_status', employeeDetail.employment_status)
                }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>账号绑定</q-item-label>
                <q-item-label>
                  {{
                    employeeDetail.bound_account?.user_name ||
                    dictLabel('org_user_binding_status', employeeDetail.binding_status)
                  }}
                </q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>手机号</q-item-label>
                <q-item-label>{{ employeeDetail.mobile_masked || '-' }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>邮箱</q-item-label>
                <q-item-label>{{ employeeDetail.email_masked || '-' }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>有效期</q-item-label>
                <q-item-label>
                  {{ formatOrganizationDate(employeeDetail.valid_from) }} 至
                  {{ formatOrganizationDate(employeeDetail.valid_to, '长期') }}
                </q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>平台备注</q-item-label>
                <q-item-label>{{ employeeDetail.local_note || '-' }}</q-item-label>
              </q-item-section>
            </q-item>
          </q-list>

          <q-separator />
          <q-card-section class="row items-center q-gutter-sm">
            <div class="text-subtitle1 text-weight-medium">任职记录</div>
            <q-space />
            <q-btn-toggle
              v-model="assignmentScope"
              dense
              unelevated
              toggle-color="primary"
              :options="assignmentScopeOptions"
              @update:model-value="loadAssignments"
            />
          </q-card-section>
          <q-table
            flat
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
              <div class="full-width text-center text-grey-7 q-pa-lg">当前范围暂无任职记录</div>
            </template>
          </q-table>
        </template>
        <q-card-actions align="right">
          <q-btn v-close-popup flat color="primary" label="关闭" />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="showBindDialog">
      <q-card style="width: 480px; max-width: 92vw">
        <q-card-section>
          <div class="text-h6">绑定平台账号</div>
          <div class="text-caption text-grey-7">{{ currentEmployee?.name }}</div>
        </q-card-section>
        <q-separator />
        <q-card-section>
          <q-input
            v-model.number="bindingUserId"
            type="number"
            min="1"
            outlined
            dense
            label="平台账号ID"
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
  unbindEmployeeUser,
  type AssignmentListItem,
  type AssignmentTimeScope,
  type EmployeeDetail,
  type EmployeeListItem,
  type EmployeeQueryRequest,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useDictStore } from 'src/stores/dict'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDate,
  organizationStatusColor,
} from 'src/pages/organization/organization-list-page'

const $q = useQuasar()
const router = useRouter()
const { confirmAction } = useConfirmDialog($q)
const dictStore = useDictStore()
const { line_buttons, top_buttons } = usePageButtons('organization_employee')

const rows = ref<EmployeeListItem[]>([])
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

const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const employmentStatusOptions = computed(() =>
  dictStore.getDictOptions('org_employment_status'),
)
const bindingStatusOptions = computed(() => [
  { label: '全部', value: 'all' },
  ...dictStore.getDictOptions('org_user_binding_status').filter((item) =>
    ['bound', 'unbound'].includes(String(item.value)),
  ),
])
const assignmentScopeOptions = [
  { label: '当前', value: 'current' },
  { label: '历史', value: 'history' },
  { label: '未来', value: 'future' },
  { label: '时间轴', value: 'timeline' },
]

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
  createOrganizationField('有效期开始', 'valid_from', SysTableFieldType.DATETIME, {
    inputType: SysTableFieldInputType.DATE_PICKER,
  }),
  createOrganizationField('有效期结束', 'valid_to', SysTableFieldType.DATETIME, {
    inputType: SysTableFieldInputType.DATE_PICKER,
  }),
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || String(value || '-')

const visibleRowButtons = (row: EmployeeListItem) =>
  line_buttons.value.filter((button) => {
    if (button.event_action === 'bind_user') return row.binding_status !== 'bound'
    if (button.event_action === 'unbind_user') return row.binding_status === 'bound'
    return ['detail', 'bind_user', 'unbind_user', 'view_sync'].includes(button.event_action || '')
  })

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
  showBindDialog.value = true
}

const submitBinding = async () => {
  if (!currentEmployee.value || !bindingUserId.value) return
  bindingLoading.value = true
  try {
    await bindEmployeeUser(currentEmployee.value.id, bindingUserId.value)
    showBindDialog.value = false
    $q.notify({ type: 'positive', position: 'top-right', message: '账号绑定成功' })
    await fetchData()
  } finally {
    bindingLoading.value = false
  }
}

const unbind = (row: EmployeeListItem) => {
  confirmAction({
    title: '解绑平台账号',
    message: `确认解除 ${row.name} 与当前平台账号的绑定吗？`,
  }).onOk(() => {
    void (async () => {
      await unbindEmployeeUser(row.id)
      $q.notify({ type: 'positive', position: 'top-right', message: '账号解绑成功' })
      await fetchData()
    })()
  })
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
