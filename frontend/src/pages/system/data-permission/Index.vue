<template>
  <base-content class="q-pa-sm">
    <q-card class="fit column no-wrap" flat bordered>
      <q-card-section class="row items-center q-py-sm">
        <q-avatar color="primary" text-color="white" icon="rule" rounded />
        <div class="q-ml-md">
          <div class="text-h6">数据权限</div>
          <div class="text-caption text-grey-7">
            维护资源、归属、策略与授权，不执行运行时数据过滤
          </div>
        </div>
        <q-space />
        <q-btn outline color="primary" icon="refresh" label="刷新" @click="refreshActiveTab" />
      </q-card-section>

      <q-separator />

      <q-tabs
        v-model="activeTab"
        align="left"
        active-color="primary"
        indicator-color="primary"
        no-caps
        narrow-indicator
      >
        <q-tab name="resources" icon="dataset" label="数据资源" />
        <q-tab name="ownerships" icon="account_tree" label="归属定义" />
        <q-tab name="policies" icon="policy" label="权限策略" />
        <q-tab name="grants" icon="verified_user" label="权限授权" />
        <q-tab name="preflight" icon="fact_check" label="配置检查" />
      </q-tabs>

      <q-separator />

      <q-tab-panels v-model="activeTab" animated keep-alive class="col">
        <q-tab-panel
          v-for="tab in listTabs"
          :key="tab.name"
          :name="tab.name"
          class="fit column no-wrap q-pa-none"
        >
          <q-table
            class="col sticky-header-table"
            :rows="activeRows"
            :columns="activeColumns"
            row-key="id"
            flat
            separator="cell"
            :loading="loading"
            :pagination="{ rowsPerPage: 0 }"
            hide-pagination
            @row-click="(_, row) => openDetail(row)"
          >
            <template #top>
              <div class="row q-col-gutter-sm items-center full-width">
                <div class="col-12 col-md">
                  <q-input
                    v-model="activeQuery.quick_query!.keyword"
                    dense
                    outlined
                    debounce="300"
                    placeholder="搜索关键词"
                    @keyup.enter="searchActiveTab"
                  >
                    <template #append>
                      <q-icon name="search" />
                    </template>
                  </q-input>
                </div>
                <div class="col-auto">
                  <q-btn
                    color="primary"
                    icon="search"
                    label="查询"
                    :disable="loading"
                    @click="searchActiveTab"
                  />
                </div>
                <div class="col-auto">
                  <q-btn
                    outline
                    color="primary"
                    icon="tune"
                    aria-label="高级查询"
                    @click="openAdvancedQuery"
                  >
                    <q-tooltip>高级查询</q-tooltip>
                  </q-btn>
                </div>
                <q-space />
                <div class="col-auto row q-gutter-sm">
                  <q-btn
                    v-for="button in activeTopButtons"
                    :key="button.id"
                    v-bind="menuButtonDisplayProps(button)"
                    :color="button.color || 'primary'"
                    :disable="loading"
                    @click="handleButtonClick(button)"
                  />
                </div>
              </div>
            </template>

            <template #body-cell-permission_enabled="props">
              <q-td :props="props">
                <q-badge :color="props.value ? 'positive' : 'grey-6'" outline>
                  {{ props.value ? '已启用' : '未启用' }}
                </q-badge>
              </q-td>
            </template>

            <template #body-cell-state="props">
              <q-td :props="props">
                <q-badge :color="props.value ? 'positive' : 'grey-6'" outline>
                  {{ props.value ? '启用' : '停用' }}
                </q-badge>
              </q-td>
            </template>

            <template #body-cell-actions="props">
              <q-td :props="props" class="q-gutter-xs">
                <q-btn
                  v-for="button in activeLineButtons"
                  :key="button.id"
                  v-bind="menuButtonDisplayProps(button)"
                  :color="button.color || 'primary'"
                  flat
                  dense
                  size="sm"
                  @click.stop="handleButtonClick(button, props.row)"
                >
                  <q-tooltip>{{ button.name }}</q-tooltip>
                </q-btn>
              </q-td>
            </template>

            <template #bottom>
              <q-space />
              <table-pagination
                :page="activeQuery.page"
                :page-size="activeQuery.num"
                :total="activeTotal"
                @update:page="setActivePage"
                @update:page-size="setActivePageSize"
              />
            </template>
          </q-table>
        </q-tab-panel>

        <q-tab-panel name="preflight" class="fit column no-wrap">
          <div class="row q-col-gutter-md items-end">
            <q-select
              v-model="preflightType"
              class="col-12 col-md-3"
              outlined
              dense
              emit-value
              map-options
              label="检查对象"
              :options="preflightTypeOptions"
              @update:model-value="resetPreflightTarget"
            />
            <q-select
              v-model="preflightId"
              class="col-12 col-md"
              outlined
              dense
              emit-value
              map-options
              use-input
              input-debounce="200"
              label="选择配置"
              :options="preflightTargetOptions"
              @filter="filterPreflightOptions"
            />
            <div class="col-auto">
              <q-btn
                v-for="button in activeTopButtons"
                :key="button.id"
                v-bind="menuButtonDisplayProps(button)"
                :color="button.color || 'primary'"
                :disable="!preflightId || preflightLoading"
                :loading="preflightLoading"
                @click="handleButtonClick(button)"
              />
            </div>
          </div>

          <q-separator class="q-my-md" />

          <q-banner
            v-if="preflightResult"
            rounded
            :class="preflightResult.valid ? 'bg-green-1 text-positive' : 'bg-red-1 text-negative'"
          >
            <template #avatar>
              <q-icon :name="preflightResult.valid ? 'task_alt' : 'error_outline'" />
            </template>
            {{ preflightResult.valid ? '配置检查通过' : '配置检查未通过' }}
          </q-banner>

          <q-table
            class="col q-mt-md sticky-header-table"
            :rows="preflightResult?.errors || []"
            :columns="preflightColumns"
            row-key="code"
            flat
            bordered
            separator="cell"
            hide-pagination
            :pagination="{ rowsPerPage: 0 }"
            no-data-label="请选择对象并执行配置检查"
          />
        </q-tab-panel>
      </q-tab-panels>
    </q-card>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:query-model="tempAdvancedQuery"
      :fields="advancedQueryFields"
      :enable-nested="false"
      @search="applyAdvancedQuery"
    />

    <data-permission-config-dialog
      v-model="showConfigDialog"
      :kind="configDialogKind"
      :edit-data="currentEditData"
      @saved="handleSaved"
    />

    <data-permission-detail-dialog v-model="showDetailDialog" :kind="detailKind" :id="detailId" />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_data_permission' })

import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import DataPermissionConfigDialog from './components/DataPermissionConfigDialog.vue'
import DataPermissionDetailDialog from './components/DataPermissionDetailDialog.vue'
import {
  type DataGrant,
  type DataOwnership,
  type DataPermissionConfigQuery,
  type DataPermissionDimension,
  type DataPolicy,
  type DataPolicyRule,
  type DataResource,
  type ValidationResult,
  useDataPermissionConfigApi,
} from 'src/api/services/data-permission-config'
import type { MenuButton } from 'src/api/services/sys-menu'
import type { TableField } from 'src/api/services/sys-table'
import type { Query } from 'src/types/global'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { usePageButtons } from 'src/composables/page-buttons'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'

type ListTabName = 'resources' | 'ownerships' | 'policies' | 'grants'
type ActiveTabName = ListTabName | 'preflight'
type ConfigDialogKind = 'resource' | 'ownership' | 'policy' | 'grant'
type ConfigRow = DataResource | DataOwnership | DataPolicy | DataGrant
type PreflightType = 'resource' | 'policy' | 'grant'

const api = useDataPermissionConfigApi()
const $q = useQuasar()
const { confirmAction } = useConfirmDialog($q)
const { top_buttons: topButtons, line_buttons: lineButtons } =
  usePageButtons('system_data_permission')

const listTabs: Array<{ name: ListTabName }> = [
  { name: 'resources' },
  { name: 'ownerships' },
  { name: 'policies' },
  { name: 'grants' },
]
const activeTab = ref<ActiveTabName>('resources')
const loading = ref(false)
const showAdvancedQuery = ref(false)
const showConfigDialog = ref(false)
const configDialogKind = ref<ConfigDialogKind>('resource')
const currentEditData = ref<ConfigRow | null>(null)
const showDetailDialog = ref(false)
const detailKind = ref<ConfigDialogKind>('resource')
const detailId = ref(0)

const resources = ref<DataResource[]>([])
const ownerships = ref<DataOwnership[]>([])
const policies = ref<DataPolicy[]>([])
const grants = ref<DataGrant[]>([])
const dimensions = ref<DataPermissionDimension[]>([])
const policyRules = ref<DataPolicyRule[]>([])
const resourceLookup = ref<DataResource[]>([])
const policyLookup = ref<DataPolicy[]>([])

const totals = ref<Record<ListTabName, number>>({
  resources: 0,
  ownerships: 0,
  policies: 0,
  grants: 0,
})

const newQuery = (): DataPermissionConfigQuery => ({
  page: 1,
  num: 15,
  order: { field: '', is_asc: false },
  expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
  quick_query: { keyword: '' },
})
const queries = ref<Record<ListTabName, DataPermissionConfigQuery>>({
  resources: newQuery(),
  ownerships: newQuery(),
  policies: newQuery(),
  grants: newQuery(),
})
const tempAdvancedQuery = ref<Query>(cloneDeep(queries.value.resources))

const resourceTypeLabels: Record<string, string> = {
  low_code_table: '低代码数据表',
  business_service: '业务服务',
  report: '报表',
}
const bindingTypeLabels: Record<string, string> = {
  metadata_field: '元数据字段',
  registered_field: '注册字段',
}
const operationLabels: Record<string, string> = {
  query: '查询',
  detail: '详情',
  create: '新增',
  update: '修改',
  delete: '删除',
  export: '导出',
  run: '运行',
}

const resourceColumns: QTableProps['columns'] = [
  { name: 'resource_code', label: '资源编码', field: 'resource_code', align: 'left' },
  { name: 'name', label: '资源名称', field: 'name', align: 'left' },
  {
    name: 'resource_type',
    label: '资源类型',
    field: (row) => resourceTypeLabels[row.resource_type] || row.resource_type,
    align: 'left',
  },
  {
    name: 'permission_enabled',
    label: '数据权限',
    field: 'permission_enabled',
    align: 'center',
  },
  { name: 'state', label: '状态', field: 'state', align: 'center' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]

const ownershipColumns: QTableProps['columns'] = [
  { name: 'ownership_code', label: '归属编码', field: 'ownership_code', align: 'left' },
  {
    name: 'resource',
    label: '数据资源',
    field: (row) => resourceLabel(row.resource_id),
    align: 'left',
  },
  {
    name: 'dimension',
    label: '数据维度',
    field: (row) => dimensionLabel(row.dimension_id),
    align: 'left',
  },
  {
    name: 'binding_type',
    label: '绑定类型',
    field: (row) => bindingTypeLabels[row.binding_type] || row.binding_type,
    align: 'left',
  },
  { name: 'state', label: '状态', field: 'state', align: 'center' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]

const policyColumns: QTableProps['columns'] = [
  { name: 'policy_code', label: '策略编码', field: 'policy_code', align: 'left' },
  { name: 'name', label: '策略名称', field: 'name', align: 'left' },
  {
    name: 'rule_count',
    label: '规则数量',
    field: (row) => policyRules.value.filter((rule) => rule.policy_id === row.id).length,
    align: 'center',
  },
  { name: 'state', label: '状态', field: 'state', align: 'center' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]

const grantColumns: QTableProps['columns'] = [
  {
    name: 'subject',
    label: '授权主体',
    field: (row) => `${row.subject_type === 'role' ? '角色' : '用户'} #${row.subject_id}`,
    align: 'left',
  },
  {
    name: 'resource',
    label: '数据资源',
    field: (row) => resourceLabel(row.resource_id),
    align: 'left',
  },
  {
    name: 'operation',
    label: '资源操作',
    field: (row) => operationLabels[row.operation] || row.operation,
    align: 'left',
  },
  {
    name: 'policy',
    label: '权限策略',
    field: (row) => policyLabel(row.policy_id),
    align: 'left',
  },
  { name: 'state', label: '状态', field: 'state', align: 'center' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]

const preflightColumns: QTableProps['columns'] = [
  { name: 'code', label: '错误编码', field: 'code', align: 'left' },
  { name: 'message', label: '说明', field: 'message', align: 'left' },
  { name: 'object_type', label: '对象类型', field: 'object_type', align: 'left' },
  { name: 'object_id', label: '对象ID', field: 'object_id', align: 'left' },
]

const activeQuery = computed(() =>
  activeTab.value === 'preflight' ? queries.value.resources : queries.value[activeTab.value],
)
const activeRows = computed(() => {
  if (activeTab.value === 'resources') return resources.value
  if (activeTab.value === 'ownerships') return ownerships.value
  if (activeTab.value === 'policies') return policies.value
  if (activeTab.value === 'grants') return grants.value
  return []
})
const activeColumns = computed(() => {
  if (activeTab.value === 'resources') return resourceColumns
  if (activeTab.value === 'ownerships') return ownershipColumns
  if (activeTab.value === 'policies') return policyColumns
  return grantColumns
})
const activeTotal = computed(() =>
  activeTab.value === 'preflight' ? 0 : totals.value[activeTab.value],
)

const topActionByTab: Record<ActiveTabName, string[]> = {
  resources: ['create_resource'],
  ownerships: ['create_ownership'],
  policies: ['create_policy'],
  grants: ['create_grant'],
  preflight: ['preflight_resource', 'preflight_policy', 'preflight_grant'],
}
const lineActionByTab: Record<ListTabName, string[]> = {
  resources: ['update_resource', 'configure_operations', 'toggle_permission'],
  ownerships: ['update_ownership'],
  policies: ['update_policy', 'configure_rules', 'toggle_policy'],
  grants: ['toggle_grant'],
}
const activeTopButtons = computed(() =>
  topButtons.value.filter((button) => {
    if (!topActionByTab[activeTab.value].includes(button.event_action)) return false
    if (activeTab.value !== 'preflight') return true
    return button.event_action === `preflight_${preflightType.value}`
  }),
)
const activeLineButtons = computed(() => {
  const tab = activeTab.value
  if (tab === 'preflight') return []
  return lineButtons.value.filter((button) => lineActionByTab[tab].includes(button.event_action))
})

const field = (
  code: string,
  name: string,
  type: SysTableFieldType = SysTableFieldType.VARCHAR,
  inputType: SysTableFieldInputType = SysTableFieldInputType.INPUT,
): Partial<TableField> => ({
  id: 0,
  field_code: code,
  field_name: name,
  field_type: type,
  input_type: inputType,
  state: true,
})
const advancedFields: Record<ListTabName, Partial<TableField>[]> = {
  resources: [
    field('resource_code', '资源编码'),
    field('name', '资源名称'),
    field('resource_type', '资源类型'),
    field('permission_enabled', '数据权限', SysTableFieldType.BOOLEAN),
    field('state', '状态', SysTableFieldType.BOOLEAN),
  ],
  ownerships: [
    field('ownership_code', '归属编码'),
    field('resource_id', '资源ID', SysTableFieldType.BIGINT, SysTableFieldInputType.INPUT_NUMBER),
    field('dimension_id', '维度ID', SysTableFieldType.BIGINT, SysTableFieldInputType.INPUT_NUMBER),
    field('binding_type', '绑定类型'),
    field('state', '状态', SysTableFieldType.BOOLEAN),
  ],
  policies: [
    field('policy_code', '策略编码'),
    field('name', '策略名称'),
    field('policy_type', '策略类型'),
    field('state', '状态', SysTableFieldType.BOOLEAN),
  ],
  grants: [
    field('subject_type', '主体类型'),
    field('subject_id', '主体ID', SysTableFieldType.BIGINT, SysTableFieldInputType.INPUT_NUMBER),
    field('resource_id', '资源ID', SysTableFieldType.BIGINT, SysTableFieldInputType.INPUT_NUMBER),
    field('operation', '资源操作'),
    field('policy_id', '策略ID', SysTableFieldType.BIGINT, SysTableFieldInputType.INPUT_NUMBER),
    field('state', '状态', SysTableFieldType.BOOLEAN),
  ],
}
const advancedQueryFields = computed(() =>
  activeTab.value === 'preflight' ? [] : advancedFields[activeTab.value],
)

const preflightType = ref<PreflightType>('resource')
const preflightId = ref<number | null>(null)
const preflightLoading = ref(false)
const preflightResult = ref<ValidationResult | null>(null)
const preflightTypeOptions = [
  { label: '数据资源', value: 'resource' },
  { label: '权限策略', value: 'policy' },
  { label: '权限授权', value: 'grant' },
]
const preflightTargetOptions = ref<Array<{ label: string; value: number }>>([])

const resourceLabel = (id: number) => {
  const resource = resourceLookup.value.find((item) => item.id === id)
  return resource ? `${resource.resource_code} · ${resource.name}` : `#${id}`
}
const policyLabel = (id: number) => {
  const policy = policyLookup.value.find((item) => item.id === id)
  return policy ? `${policy.policy_code} · ${policy.name}` : `#${id}`
}
const dimensionLabel = (id: number) => {
  const dimension = dimensions.value.find((item) => item.id === id)
  return dimension ? `${dimension.dimension_code} · ${dimension.name}` : `#${id}`
}

const fetchActiveTab = async () => {
  if (activeTab.value === 'preflight') return
  loading.value = true
  try {
    const tab = activeTab.value
    const query = queries.value[tab]
    if (tab === 'resources') {
      const result = await api.queryResources(query)
      resources.value = result.data || []
      totals.value.resources = result.total || 0
    } else if (tab === 'ownerships') {
      const result = await api.queryOwnerships(query)
      ownerships.value = result.data || []
      totals.value.ownerships = result.total || 0
    } else if (tab === 'policies') {
      const [policyResult, ruleResult] = await Promise.all([
        api.queryPolicies(query),
        api.queryPolicyRules({ ...newQuery(), num: 500 }),
      ])
      policies.value = policyResult.data || []
      policyRules.value = ruleResult.data || []
      totals.value.policies = policyResult.total || 0
    } else {
      const result = await api.queryGrants(query)
      grants.value = result.data || []
      totals.value.grants = result.total || 0
    }
  } finally {
    loading.value = false
  }
}

const loadLookups = async () => {
  const lookupQuery = { ...newQuery(), num: 500 }
  const [resourceResult, policyResult, dimensionResult] = await Promise.all([
    api.queryResources(lookupQuery),
    api.queryPolicies(lookupQuery),
    api.queryDimensions(lookupQuery),
  ])
  resourceLookup.value = resourceResult.data || []
  policyLookup.value = policyResult.data || []
  dimensions.value = dimensionResult.data || []
}

const searchActiveTab = () => {
  if (activeTab.value === 'preflight') return
  queries.value[activeTab.value].page = 1
  void fetchActiveTab()
}
const refreshActiveTab = () => {
  if (activeTab.value === 'preflight') {
    resetPreflightTarget()
    return
  }
  void Promise.all([fetchActiveTab(), loadLookups()])
}
const setActivePage = (page: number) => {
  if (activeTab.value === 'preflight') return
  queries.value[activeTab.value].page = page
  void fetchActiveTab()
}
const setActivePageSize = (pageSize: number) => {
  if (activeTab.value === 'preflight') return
  queries.value[activeTab.value].num = pageSize
  queries.value[activeTab.value].page = 1
  void fetchActiveTab()
}
const openAdvancedQuery = () => {
  if (activeTab.value === 'preflight') return
  tempAdvancedQuery.value = cloneDeep(queries.value[activeTab.value])
  showAdvancedQuery.value = true
}
const applyAdvancedQuery = () => {
  if (activeTab.value === 'preflight') return
  queries.value[activeTab.value].expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  queries.value[activeTab.value].page = 1
  showAdvancedQuery.value = false
  void fetchActiveTab()
}

const openConfig = (kind: ConfigDialogKind, row: ConfigRow | null = null) => {
  configDialogKind.value = kind
  currentEditData.value = row
  showConfigDialog.value = true
}
const openDetail = (row: ConfigRow) => {
  if (activeTab.value === 'preflight') return
  detailKind.value = activeTab.value.slice(0, -1) as ConfigDialogKind
  detailId.value = row.id
  showDetailDialog.value = true
}
const handleSaved = () => {
  void Promise.all([fetchActiveTab(), loadLookups()])
}

const toggleResource = (row: DataResource) => {
  const enabled = !row.permission_enabled
  confirmAction({
    title: enabled ? '启用数据权限' : '停用数据权限',
    message: enabled
      ? `启用前将执行完整配置检查，确认启用“${row.name}”的数据权限？`
      : `确认停用“${row.name}”的数据权限？`,
  }).onOk(async () => {
    const result = await api.setResourcePermission(row.id, enabled)
    if (!result.data.valid) {
      $q.notify({
        type: 'negative',
        message: result.data.errors.map((item) => item.message).join('；'),
      })
      return
    }
    $q.notify({ type: 'positive', message: enabled ? '数据权限已启用' : '数据权限已停用' })
    await fetchActiveTab()
  })
}
const toggleOwnership = (row: DataOwnership) => {
  if (!row.state) return
  confirmAction({
    title: '停用归属定义',
    message: `确认停用“${row.ownership_code}”？`,
  }).onOk(async () => {
    await api.disableOwnership(row.id)
    $q.notify({ type: 'positive', message: '归属定义已停用' })
    await fetchActiveTab()
  })
}
const togglePolicy = (row: DataPolicy) => {
  const enabled = !row.state
  confirmAction({
    title: enabled ? '启用权限策略' : '停用权限策略',
    message: `确认${enabled ? '启用' : '停用'}“${row.name}”？`,
  }).onOk(async () => {
    const result = await api.setPolicyState(row.id, enabled)
    if (!result.data.valid) {
      $q.notify({
        type: 'negative',
        message: result.data.errors.map((item) => item.message).join('；'),
      })
      return
    }
    $q.notify({ type: 'positive', message: `权限策略已${enabled ? '启用' : '停用'}` })
    await fetchActiveTab()
  })
}
const toggleGrant = (row: DataGrant) => {
  const enabled = !row.state
  confirmAction({
    title: enabled ? '启用权限授权' : '停用权限授权',
    message: `确认${enabled ? '启用' : '停用'}当前授权？`,
  }).onOk(async () => {
    const result = await api.setGrantState(row.id, enabled)
    if (!result.data.valid) {
      $q.notify({
        type: 'negative',
        message: result.data.errors.map((item) => item.message).join('；'),
      })
      return
    }
    $q.notify({ type: 'positive', message: `权限授权已${enabled ? '启用' : '停用'}` })
    await fetchActiveTab()
  })
}

const actionHandlers: Record<string, (row?: ConfigRow) => void> = {
  create_resource: () => openConfig('resource'),
  update_resource: (row) => row && openConfig('resource', row),
  configure_operations: (row) => row && openConfig('resource', row),
  toggle_permission: (row) => row && void toggleResource(row as DataResource),
  create_ownership: () => openConfig('ownership'),
  update_ownership: (row) => row && void toggleOwnership(row as DataOwnership),
  create_policy: () => openConfig('policy'),
  update_policy: (row) => row && openConfig('policy', row),
  configure_rules: (row) => row && openConfig('policy', row),
  toggle_policy: (row) => row && void togglePolicy(row as DataPolicy),
  create_grant: () => openConfig('grant'),
  toggle_grant: (row) => row && void toggleGrant(row as DataGrant),
  preflight_resource: () => void runPreflight(),
  preflight_policy: () => void runPreflight(),
  preflight_grant: () => void runPreflight(),
}
const handleButtonClick = (button: MenuButton, row?: ConfigRow) => {
  actionHandlers[button.event_action]?.(row)
}

const targetOptions = (keyword = '') => {
  const normalized = keyword.toLowerCase()
  if (preflightType.value === 'resource') {
    return resourceLookup.value
      .filter((item) => `${item.resource_code} ${item.name}`.toLowerCase().includes(normalized))
      .map((item) => ({ label: `${item.resource_code} · ${item.name}`, value: item.id }))
  }
  if (preflightType.value === 'policy') {
    return policyLookup.value
      .filter((item) => `${item.policy_code} ${item.name}`.toLowerCase().includes(normalized))
      .map((item) => ({ label: `${item.policy_code} · ${item.name}`, value: item.id }))
  }
  return grants.value
    .filter((item) =>
      `${item.subject_type} ${item.subject_id} ${resourceLabel(item.resource_id)}`
        .toLowerCase()
        .includes(normalized),
    )
    .map((item) => ({
      label: `${item.subject_type} #${item.subject_id} · ${resourceLabel(item.resource_id)}`,
      value: item.id,
    }))
}
const resetPreflightTarget = () => {
  preflightId.value = null
  preflightResult.value = null
  preflightTargetOptions.value = targetOptions()
}
const filterPreflightOptions = (value: string, update: (callback: () => void) => void) =>
  update(() => {
    preflightTargetOptions.value = targetOptions(value)
  })
const runPreflight = async () => {
  if (!preflightId.value) return
  preflightLoading.value = true
  try {
    const result = await api.preflight(preflightType.value, preflightId.value)
    preflightResult.value = result.data
  } finally {
    preflightLoading.value = false
  }
}

watch(activeTab, async (tab) => {
  if (tab === 'preflight') {
    if (grants.value.length === 0) {
      const result = await api.queryGrants({ ...newQuery(), num: 500 })
      grants.value = result.data || []
    }
    resetPreflightTarget()
    return
  }
  tempAdvancedQuery.value = cloneDeep(queries.value[tab])
  await fetchActiveTab()
})

onMounted(async () => {
  await Promise.all([loadLookups(), fetchActiveTab()])
})
</script>
