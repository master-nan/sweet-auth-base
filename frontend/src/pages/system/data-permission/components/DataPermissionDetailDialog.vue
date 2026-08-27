<template>
  <form-dialog-shell
    v-model="visible"
    :title="title"
    :subtitle="subtitle"
    :icon="icon"
    readonly
    :show-preview="false"
  >
    <div class="q-pa-lg">
      <q-inner-loading :showing="loading">
        <q-spinner color="primary" size="40px" />
      </q-inner-loading>

      <template v-if="detail">
        <q-list bordered separator>
          <template v-if="kind === 'resource'">
            <q-item>
              <q-item-section>
                <q-item-label caption>资源编码</q-item-label>
                <q-item-label>{{ resourceDetail.resource_code }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>资源名称</q-item-label>
                <q-item-label>{{ resourceDetail.name }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>资源类型</q-item-label>
                <q-item-label>{{ resourceTypeLabel(resourceDetail.resource_type) }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>数据权限</q-item-label>
                <q-item-label>
                  <q-badge
                    :color="resourceDetail.permission_enabled ? 'positive' : 'grey-6'"
                    outline
                  >
                    {{ resourceDetail.permission_enabled ? '已启用' : '未启用' }}
                  </q-badge>
                </q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>支持操作</q-item-label>
                <q-item-label>
                  <q-chip
                    v-for="operation in resourceOperations"
                    :key="operation.id"
                    dense
                    square
                    color="primary"
                    text-color="white"
                  >
                    {{ operationLabel(operation.operation) }}
                  </q-chip>
                  <span v-if="resourceOperations.length === 0">-</span>
                </q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>归属定义</q-item-label>
                <q-item-label>{{ resourceOwnerships.length }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>关联策略</q-item-label>
                <q-item-label>{{ resourcePolicyCount }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>授权数量</q-item-label>
                <q-item-label>{{ resourceGrants.length }}</q-item-label>
              </q-item-section>
            </q-item>
          </template>

          <template v-else-if="kind === 'ownership'">
            <q-item>
              <q-item-section>
                <q-item-label caption>归属编码</q-item-label>
                <q-item-label>{{ ownershipDetail.ownership_code }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>数据资源</q-item-label>
                <q-item-label>{{ ownershipDetail.resource?.name || '-' }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>数据维度</q-item-label>
                <q-item-label>{{ ownershipDetail.dimension?.name || '-' }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>绑定类型</q-item-label>
                <q-item-label>{{ bindingTypeLabel(ownershipDetail.binding_type) }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>值类型</q-item-label>
                <q-item-label>{{ valueTypeLabel(ownershipDetail.value_type) }}</q-item-label>
              </q-item-section>
            </q-item>
          </template>

          <template v-else-if="kind === 'policy'">
            <q-item>
              <q-item-section>
                <q-item-label caption>策略编码</q-item-label>
                <q-item-label>{{ policyDetail.policy_code }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>策略名称</q-item-label>
                <q-item-label>{{ policyDetail.name }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>规则数量</q-item-label>
                <q-item-label>{{ policyDetail.rules?.length || 0 }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>状态</q-item-label>
                <q-item-label>
                  <q-badge :color="policyDetail.state ? 'positive' : 'grey-6'" outline>
                    {{ policyDetail.state ? '启用' : '停用' }}
                  </q-badge>
                </q-item-label>
              </q-item-section>
            </q-item>
          </template>

          <template v-else>
            <q-item>
              <q-item-section>
                <q-item-label caption>授权主体</q-item-label>
                <q-item-label>{{ grantSubjectLabel }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>数据资源</q-item-label>
                <q-item-label>{{ grantDetail.resource?.name || '数据资源不可用' }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>资源操作</q-item-label>
                <q-item-label>{{ operationLabel(grantDetail.operation) }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>权限策略</q-item-label>
                <q-item-label>{{ grantDetail.policy?.name || '权限策略不可用' }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>状态</q-item-label>
                <q-item-label>
                  <q-badge :color="grantDetail.state ? 'positive' : 'grey-6'" outline>
                    {{ grantDetail.state ? '启用' : '停用' }}
                  </q-badge>
                </q-item-label>
              </q-item-section>
            </q-item>
          </template>
        </q-list>

        <div v-if="kind === 'policy'" class="text-subtitle1 text-weight-medium q-mt-lg q-mb-sm">
          策略规则
        </div>
        <q-table
          v-if="kind === 'policy'"
          :rows="policyDetail.rules || []"
          :columns="ruleColumns"
          row-key="id"
          flat
          bordered
          separator="cell"
          hide-pagination
          :pagination="{ rowsPerPage: 0 }"
          no-data-label="暂无策略规则"
        />
      </template>
    </div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import {
  type DataGrant,
  type DataOwnership,
  type DataPolicy,
  type DataResource,
  type DataResourceOperationItem,
  useDataPermissionConfigApi,
} from 'src/api/services/data-permission-config'
import type { Query } from 'src/types/global'

type DetailKind = 'resource' | 'ownership' | 'policy' | 'grant'

const props = defineProps<{
  modelValue: boolean
  kind: DetailKind
  id: number
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
}>()

const api = useDataPermissionConfigApi()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const loading = ref(false)
const detail = ref<DataResource | DataOwnership | DataPolicy | DataGrant | null>(null)
const resourceOperations = ref<DataResourceOperationItem[]>([])
const resourceOwnerships = ref<DataOwnership[]>([])
const resourceGrants = ref<DataGrant[]>([])

const resourceDetail = computed(() => detail.value as DataResource)
const ownershipDetail = computed(() => detail.value as DataOwnership)
const policyDetail = computed(() => detail.value as DataPolicy)
const grantDetail = computed(() => detail.value as DataGrant)
const grantSubjectLabel = computed(() => {
  const subject = grantDetail.value.subject
  if (!subject) return grantDetail.value.subject_type === 'role' ? '角色不可用' : '用户不可用'
  return subject.code ? `${subject.name} · ${subject.code}` : subject.name
})
const resourcePolicyCount = computed(
  () => new Set(resourceGrants.value.map((grant) => grant.policy_id)).size,
)

const title = computed(
  () =>
    ({
      resource: '数据资源详情',
      ownership: '归属定义详情',
      policy: '权限策略详情',
      grant: '权限授权详情',
    })[props.kind],
)
const subtitle = computed(() => {
  if (!detail.value) return '正在读取配置'
  if (props.kind === 'resource') return resourceDetail.value.resource_code
  if (props.kind === 'ownership') return ownershipDetail.value.ownership_code
  if (props.kind === 'policy') return policyDetail.value.policy_code
  return grantSubjectLabel.value
})
const icon = computed(
  () =>
    ({
      resource: 'dataset',
      ownership: 'account_tree',
      policy: 'policy',
      grant: 'verified_user',
    })[props.kind],
)

const ruleColumns: QTableProps['columns'] = [
  { name: 'sequence', label: '顺序', field: 'sequence', align: 'left' },
  { name: 'ownership_code', label: '归属编码', field: 'ownership_code', align: 'left' },
  {
    name: 'dimension',
    label: '数据维度',
    field: (row) => row.dimension?.name || '数据维度不可用',
    align: 'left',
  },
  {
    name: 'scope_source',
    label: '范围来源',
    field: (row) => scopeSourceLabel(row.scope_source),
    align: 'left',
  },
  {
    name: 'relation',
    label: '关系',
    field: (row) => relationLabel(row.relation),
    align: 'left',
  },
  {
    name: 'operator',
    label: '操作符',
    field: (row) => operatorLabel(row.operator),
    align: 'left',
  },
]

const baseQuery = (resourceId: number): Query & { resource_id: number } => ({
  page: 1,
  num: 500,
  order: { field: '', is_asc: true },
  expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
  quick_query: { keyword: '' },
  resource_id: resourceId,
})

const loadDetail = async () => {
  if (!props.id) return
  loading.value = true
  try {
    resourceOperations.value = []
    resourceOwnerships.value = []
    resourceGrants.value = []
    if (props.kind === 'resource') {
      const [resource, operations, ownerships, grants] = await Promise.all([
        api.getResource(props.id),
        api.listResourceOperations(props.id),
        api.listResourceOwnerships(props.id),
        api.queryGrants(baseQuery(props.id)),
      ])
      detail.value = resource.data
      resourceOperations.value = operations.data || []
      resourceOwnerships.value = ownerships.data || []
      resourceGrants.value = grants.data || []
    } else if (props.kind === 'ownership') {
      detail.value = (await api.getOwnership(props.id)).data
    } else if (props.kind === 'policy') {
      detail.value = (await api.getPolicy(props.id)).data
    } else {
      detail.value = (await api.getGrant(props.id)).data
    }
  } finally {
    loading.value = false
  }
}

const resourceTypeLabel = (value: string) =>
  ({ low_code_table: '低代码数据表', business_service: '业务服务', report: '报表' })[value] || value
const operationLabel = (value: string) =>
  ({
    query: '查询',
    detail: '详情',
    create: '新增',
    update: '修改',
    delete: '删除',
    export: '导出',
    run: '运行',
  })[value] || value
const bindingTypeLabel = (value: string) =>
  ({ metadata_field: '元数据字段', registered_field: '注册字段' })[value] || value
const valueTypeLabel = (value: string) =>
  ({ bigint: '数字ID', string: '字符串编码' })[value] || value
const scopeSourceLabel = (value: string) =>
  ({
    effective_legal_entities: '当前有效法人',
    effective_org_units: '当前有效组织',
    current_employee: '当前员工',
    specified_values: '指定值',
  })[value] || value
const relationLabel = (value: string) =>
  ({ exact: '精确匹配', self_and_descendants: '本级及下级' })[value] || value
const operatorLabel = (value: string) => ({ eq: '等于', in: '包含于' })[value] || value

watch(
  () => [props.modelValue, props.kind, props.id],
  ([open]) => {
    if (open) void loadDetail()
  },
)
</script>
