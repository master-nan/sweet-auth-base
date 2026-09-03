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
      :visible-columns="visibleColumns"
      :loading="loading"
      :no-data-label="emptyMessage"
      v-model:pagination="pagination"
      hide-pagination
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #query-controls>
            <query-scheme-controls
              :controller="schemePage"
              :query-state="queryState"
              :fields="advancedFields"
              :advanced-title="t('ui.advancedEmployeeSearch')"
              :show-filter-count="false"
            >
              <template #quick-search>
                <q-input
                  v-model="keyword"
                  dense
                  outlined
                  debounce="300"
                  :placeholder="quickSearchPlaceholder"
                  @keyup.enter="search"
                >
                  <template #append><q-icon name="search" /></template>
                </q-input>
                <q-btn color="primary" :label="t('ui.search')" :disable="loading" @click="search" />
              </template>
            </query-scheme-controls>
          </template>

          <template #column-selector>
            <table-column-selector v-model="visibleColumns" :columns="columns" />
          </template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-employment_status="props">
        <q-td :props="props">
          <status-chip
            :color="organizationStatusColor(props.row.employment_status)"
            :label="dictLabel('org_employment_status', props.row.employment_status)"
          />
        </q-td>
      </template>

      <template #body-cell-binding_status="props">
        <q-td :props="props">
          <status-chip
            :color="props.row.binding_status === 'bound' ? 'positive' : 'grey-7'"
            :label="dictLabel('org_user_binding_status', props.row.binding_status)"
          />
        </q-td>
      </template>

      <template #body-cell-validity="props">
        <q-td :props="props">
          {{ formatOrganizationDate(props.row.valid_from) }}
          <span class="text-grey-6 q-mx-xs">{{ t('ui.to') }}</span>
          {{ formatOrganizationDate(props.row.valid_to, t('ui.longTerm')) }}
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

      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <organization-record-detail-dialog
      v-model="showDetailDialog"
      :title="employeeDetail?.name || t('ui.personnelDetails')"
      :subtitle="employeeDetail?.employee_no || ''"
      :sections="employeeDetailSections"
      icon="badge"
      :avatar-label="employeeDetail?.name?.slice(0, 1) || ''"
      :status-label="
        employeeDetail ? dictLabel('org_employment_status', employeeDetail.employment_status) : ''
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
            <div class="text-caption text-grey-7">
              {{ t('ui.viewAllStaffMembersByTimeFrameNotAutomaticallySelecting') }}
            </div>
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
                {{ formatOrganizationDate(props.row.valid_from) }} {{ t('ui.to') }}
                {{ formatOrganizationDate(props.row.valid_to, t('ui.longTerm')) }}
              </q-td>
            </template>
            <template #no-data>
              <div class="full-width column flex-center q-gutter-sm q-pa-lg text-grey-7">
                <q-icon name="inbox" color="grey-5" size="48px" />
                <span>{{ t('ui.currentRangeForTemporaryNoServiceRecord') }}</span>
              </div>
            </template>
          </q-table>
        </template>
      </template>
    </organization-record-detail-dialog>

    <q-dialog v-model="showBindDialog">
      <q-card style="width: 480px; max-width: 92vw">
        <q-card-section>
          <div class="text-h6">{{ t('ui.tiePlatformAccount') }}</div>
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
            :label="t('ui.platformAccount')"
            :hint="t('ui.enterAccountNameSearch')"
            :options="bindingUserOptions"
            :loading="bindingUserOptionsLoading"
            @filter="filterBindingUsers"
            @virtual-scroll="loadMoreBindingUsers"
          />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn v-close-popup flat :label="t('ui.cancel')" />
          <q-btn
            color="primary"
            icon="link"
            :label="t('ui.tie')"
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
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'organization_employee' })

import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import QuerySchemeControls from '@/components/QueryScheme/QuerySchemeControls.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import TableColumnSelector from '@/components/Table/TableColumnSelector.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
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
} from '@/api/services/org'
import type { MenuButton } from '@/api/services/sys-menu'
import { usePageButtons } from '@/composables/page-buttons'
import { useConfirmDialog } from '@/composables/confirm-dialog'
import { useRuntimeTableMetadata } from '@/composables/runtime-table-metadata'
import { useTableQueryState } from '@/composables/table-query-state'
import { useQuerySchemePage } from '@/composables/query-scheme-page'
import { useDictStore } from '@/stores/dict'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'
import { resolveRuntimeColumns } from '@/utils/column-format'
import { resolveTableEmptyMessage } from '@/utils/table-state'
import type { TableColumn } from '@/types/global'
import OrganizationRecordDetailDialog from '@/pages/organization/components/OrganizationRecordDetailDialog.vue'
import AssignmentScopeSwitch from '@/pages/organization/employee/AssignmentScopeSwitch.vue'
import type { OrganizationDetailSection } from '@/pages/organization/components/organization-record-detail'
import { useOrganizationDetailMode } from '@/pages/organization/use-organization-detail-mode'
import {
  createOrganizationQuery,
  formatOrganizationDate,
  formatOrganizationDateTime,
  formatOrganizationValue,
  organizationStatusColor,
} from '@/pages/organization/organization-list-page'

const { t } = useI18n({ useScope: 'global' })

const $q = useQuasar()
const router = useRouter()
const { confirmAction } = useConfirmDialog($q)
const dictStore = useDictStore()
const {
  line_buttons,
  record_detail_top_buttons,
  record_detail_bottom_buttons,
  has_line_buttons,
  hasGrantedCapability,
} = usePageButtons('organization_employee')
const detailMode = useOrganizationDetailMode('organization_employee', 'dialog')

const rows = ref<EmployeeListItem[]>([])
const canQueryEmployees = computed(() => hasGrantedCapability('organization_employee_query'))
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const queryState = useTableQueryState<EmployeeQueryRequest>({
  createInitialQuery: () => ({
    ...createOrganizationQuery('org_employee'),
    only_effective: true,
    bound_status: 'all',
  }),
})
const { query, keyword } = queryState
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('organization_employee', queryState, resetAndFetch)
const initialized = ref(false)
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })

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

const columns = ref<TableColumn<EmployeeListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const {
  fields: metadataFields,
  quickSearchPlaceholder,
  advancedSearchFields: advancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('org_employee')

const assignmentColumns: QTableProps['columns'] = [
  {
    name: 'legal_entity',
    field: (row: AssignmentListItem) => row.legal_entity?.name || '-',
    get label() {
      return t('ui.legalEntity')
    },
    align: 'left',
  },
  {
    name: 'org_unit',
    field: (row: AssignmentListItem) => row.org_unit?.name || '-',
    get label() {
      return t('ui.organizations')
    },
    align: 'left',
  },
  {
    name: 'position',
    field: (row: AssignmentListItem) => row.position?.name || '-',
    get label() {
      return t('ui.positionLabel')
    },
    align: 'left',
  },
  {
    name: 'assignment_type',
    field: 'assignment_type',
    get label() {
      return t('ui.assignmentType')
    },
    align: 'center',
  },
  {
    name: 'validity',
    field: 'validity',
    get label() {
      return t('ui.validityPeriod')
    },
    align: 'left',
  },
]

const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryEmployees.value,
    error: loadError.value,
    hasQuery: !!keyword.value || query.value.expressions.length > 0,
  }),
)

const fetchMetadata = async () => {
  if (!(await loadMetadata())) return
  const resolution = resolveRuntimeColumns<EmployeeListItem>(metadataFields.value, {
    context: { getDictLabel: dictLabel },
    overrides: [
      { fieldCode: 'employee_no', order: 1 },
      { fieldCode: 'name', order: 2 },
      { fieldCode: 'employment_status', align: 'center', order: 3 },
      { fieldCode: 'user_id', visible: false },
      { fieldCode: 'primary_legal_entity_id', visible: false },
      { fieldCode: 'valid_from', visible: false },
      { fieldCode: 'valid_to', visible: false },
    ],
    virtualColumns: [
      {
        name: 'primary_legal_entity',
        field: (row) => row.primary_legal_entity?.name || '-',
        get label() {
          return t('ui.primaryLegalEntity')
        },
        order: 4,
      },
      {
        name: 'binding_status',
        field: 'binding_status',
        get label() {
          return t('ui.accountBinding')
        },
        align: 'center',
        order: 5,
      },
      {
        name: 'bound_account',
        field: (row) => row.bound_account?.user_name || '-',
        get label() {
          return t('ui.platformAccount')
        },
        order: 6,
      },
      {
        name: 'validity',
        field: 'validity',
        get label() {
          return t('ui.validityPeriod')
        },
        order: 7,
      },
      {
        name: 'actions',
        field: 'actions',
        get label() {
          return t('ui.actions')
        },
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

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || formatOrganizationValue(value)

const employeeDetailSections = computed<OrganizationDetailSection[]>(() => {
  const detail = employeeDetail.value
  const basicItems = detail
    ? [
        {
          get label() {
            return t('ui.employeeNumber')
          },
          value: detail.employee_no,
        },
        {
          get label() {
            return t('ui.name')
          },
          value: detail.name,
        },
        {
          get label() {
            return t('ui.cellPhoneNumber')
          },
          value: detail.mobile_masked ?? null,
        },
        {
          get label() {
            return t('ui.mailbox')
          },
          value: detail.email_masked ?? null,
        },
        {
          get label() {
            return t('ui.primaryLegalEntity')
          },
          value: detail.primary_legal_entity?.name || '-',
        },
        {
          get label() {
            return t('ui.employeeStatus')
          },
          value: dictLabel('org_employment_status', detail.employment_status),
          chip: true,
          color: organizationStatusColor(detail.employment_status),
        },
        {
          get label() {
            return t('ui.validityPeriod')
          },
          value: t('ui.rangeFromTo', {
            value1: formatOrganizationDate(detail.valid_from),
            value2: formatOrganizationDate(detail.valid_to, t('ui.longTerm')),
          }),
        },
      ]
    : []
  return [
    {
      key: 'basic',
      get label() {
        return t('ui.profileInformation')
      },
      get caption() {
        return t('ui.identityAndContactInformation')
      },
      icon: 'badge',
      items: basicItems,
    },
    {
      key: 'assignments',
      get label() {
        return t('ui.jobIncumbencyRecords')
      },
      get caption() {
        return t('ui.currentHistoricalAndFuturePositions')
      },
      icon: 'work_history',
      count: assignments.value.length,
      items: [],
    },
    {
      key: 'account',
      get label() {
        return t('ui.accountInformation')
      },
      get caption() {
        return t('ui.currentPlatformAccountBind')
      },
      icon: 'manage_accounts',
      get items() {
        return detail
          ? [
              {
                label: t('ui.tieStatus'),
                value: dictLabel('org_user_binding_status', detail.binding_status),
                chip: true,
                color: detail.binding_status === 'bound' ? 'positive' : 'grey-7',
              },
              {
                label: t('ui.platformAccount'),
                value: detail.bound_account?.user_name || t('ui.unbound'),
              },
            ]
          : []
      },
    },
    {
      key: 'mirror',
      get label() {
        return t('ui.imageInformation')
      },
      get caption() {
        return t('ui.syncTimeAndPlatformRemarks')
      },
      icon: 'sync',
      get items() {
        return detail
          ? [
              { label: t('ui.recentUpdate'), value: formatOrganizationDateTime(detail.gmt_modify) },
              { label: t('ui.platformNotes'), value: detail.local_note, fullWidth: true },
            ]
          : []
      },
    },
  ]
})

const visibleRowButtons = (row?: EmployeeListItem) => {
  void row
  return line_buttons.value.filter((button) => button.event_action === 'detail')
}

const search = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
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
    loadError.value = t('ui.failedToLoadEmployees')
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
    detailError.value = t('ui.failedToLoadEmployeeDetails')
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
    $q.notify({
      type: 'positive',
      position: 'top-right',
      get message() {
        return t('ui.accountBindingSuccessful')
      },
    })
    await fetchData()
    if (showDetailDialog.value)
      employeeDetail.value = await getEmployeeDetail(currentEmployee.value.id)
  } finally {
    bindingLoading.value = false
  }
}

const unbind = (row: Pick<EmployeeListItem, 'id' | 'name'>) => {
  confirmAction({
    get title() {
      return t('ui.untiePlatformAccount')
    },
    get message() {
      return t('ui.confirmToUntieWithTheCurrentPlatformAccount', { value1: row.name })
    },
  }).onOk(() => {
    void (async () => {
      await unbindEmployeeUser(row.id)
      $q.notify({
        type: 'positive',
        position: 'top-right',
        get message() {
          return t('ui.accountUntied')
        },
      })
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
  () => {
    if (initialized.value) void fetchData()
  },
)

watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([field, descending]) => {
    if (!initialized.value) return
    if (!queryState.applySorting(field || '', descending, sortableFields.value)) return
    void fetchData()
  },
)

onMounted(async () => {
  await dictStore.loadDicts([
    'org_employment_status',
    'org_user_binding_status',
    'org_assignment_type',
  ])
  await fetchMetadata()
  await schemePage.initialize()
  await fetchData()
  initialized.value = true
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
