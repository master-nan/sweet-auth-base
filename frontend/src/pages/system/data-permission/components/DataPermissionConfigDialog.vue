<template>
  <form-dialog-shell
    v-model="visible"
    :title="title"
    :subtitle="subtitle"
    :icon="dialogIcon"
    :loading="saving"
    :submit-text="editData ? '保存' : '创建'"
    :show-preview="false"
    @submit="submit"
  >
    <q-form ref="formRef" class="q-pa-lg q-gutter-md">
      <template v-if="kind === 'resource'">
        <div class="row q-col-gutter-md">
          <q-input
            v-model="resourceForm.resource_code"
            class="col-12 col-md-6"
            outlined
            dense
            label="资源编码"
            :disable="Boolean(editData)"
            :rules="requiredRules"
          />
          <q-input
            v-model="resourceForm.name"
            class="col-12 col-md-6"
            outlined
            dense
            label="资源名称"
            :rules="requiredRules"
          />
          <q-select
            v-model="resourceForm.resource_type"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            label="资源类型"
            :disable="Boolean(editData)"
            :options="resourceTypeOptions"
            :rules="requiredRules"
            @update:model-value="clearResourceTarget"
          />
          <q-input
            v-model="resourceForm.adapter_code"
            class="col-12 col-md-6"
            outlined
            dense
            label="适配器编码"
            :disable="Boolean(editData)"
            :rules="requiredRules"
          />
          <q-select
            v-if="resourceForm.resource_type === 'low_code_table'"
            v-model="resourceForm.target.reference_id"
            class="col-12"
            outlined
            dense
            emit-value
            map-options
            use-input
            label="元数据表"
            :disable="Boolean(editData)"
            :options="tableOptions"
            :rules="requiredRules"
            @filter="filterTableOptions"
          />
          <q-input
            v-else-if="resourceForm.resource_type === 'business_service'"
            v-model="resourceForm.target.reference_code"
            class="col-12"
            outlined
            dense
            label="业务服务编码"
            :disable="Boolean(editData)"
            :rules="requiredRules"
          />
          <q-input
            v-else
            v-model.number="resourceForm.target.reference_id"
            class="col-12"
            outlined
            dense
            type="number"
            label="报表定义ID"
            :disable="Boolean(editData)"
            :rules="positiveIdRules"
          />
          <q-select
            v-model="resourceForm.operations"
            class="col-12"
            outlined
            dense
            multiple
            emit-value
            map-options
            use-chips
            label="支持操作"
            :options="operationOptions"
            :rules="atLeastOneRules"
          />
          <q-input
            v-if="!editData"
            v-model="resourceForm.description"
            class="col-12"
            outlined
            dense
            type="textarea"
            autogrow
            label="说明"
          />
          <q-toggle v-model="resourceForm.state" label="配置状态启用" />
        </div>
      </template>

      <template v-else-if="kind === 'ownership'">
        <div class="row q-col-gutter-md">
          <q-select
            v-model="ownershipForm.resource_id"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            use-input
            label="数据资源"
            :disable="Boolean(editData)"
            :options="resourceOptions"
            :rules="requiredRules"
            @filter="filterResourceOptions"
            @update:model-value="loadMetadataFields"
          />
          <q-input
            v-model="ownershipForm.ownership_code"
            class="col-12 col-md-6"
            outlined
            dense
            label="归属编码"
            :disable="Boolean(editData)"
            :rules="requiredRules"
          />
          <q-select
            v-model="ownershipForm.dimension_id"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            label="数据维度"
            :disable="Boolean(editData)"
            :options="dimensionOptions"
            :rules="requiredRules"
            @update:model-value="syncOwnershipValueType"
          />
          <q-select
            v-model="ownershipForm.binding_type"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            label="绑定类型"
            :disable="Boolean(editData)"
            :options="bindingTypeOptions"
            :rules="requiredRules"
          />
          <q-select
            v-if="ownershipForm.binding_type === 'metadata_field'"
            v-model="ownershipForm.binding_target.reference_id"
            class="col-12"
            outlined
            dense
            emit-value
            map-options
            label="元数据字段"
            :disable="Boolean(editData)"
            :loading="metadataFieldLoading"
            :options="metadataFieldOptions"
            :rules="requiredRules"
          />
          <q-input
            v-else
            v-model="ownershipForm.binding_target.reference_code"
            class="col-12"
            outlined
            dense
            label="注册字段编码"
            hint="仅接受服务端已经注册的稳定字段编码"
            :disable="Boolean(editData)"
            :rules="requiredRules"
          />
          <q-select
            v-model="ownershipForm.value_type"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            label="值类型"
            :disable="Boolean(editData)"
            :options="valueTypeOptions"
            :rules="requiredRules"
          />
          <div class="col-12 col-md-6 row items-center">
            <q-toggle v-model="ownershipForm.state" label="配置状态启用" />
          </div>
        </div>
      </template>

      <template v-else-if="kind === 'policy'">
        <div class="row q-col-gutter-md">
          <q-input
            v-model="policyForm.policy_code"
            class="col-12 col-md-6"
            outlined
            dense
            label="策略编码"
            :disable="Boolean(editData)"
            :rules="requiredRules"
          />
          <q-input
            v-model="policyForm.name"
            class="col-12 col-md-6"
            outlined
            dense
            label="策略名称"
            :rules="requiredRules"
          />
          <q-input
            v-if="!editData"
            v-model="policyForm.description"
            class="col-12"
            outlined
            dense
            type="textarea"
            autogrow
            label="说明"
          />
          <div class="col-12">
            <div class="row items-center q-mb-sm">
              <div class="text-subtitle1 text-weight-medium">策略规则</div>
              <q-space />
              <q-btn
                color="primary"
                icon="add"
                label="新增规则"
                :disable="policyForm.rules.length >= 8"
                @click="addPolicyRule"
              />
            </div>
            <q-list bordered separator>
              <q-item v-for="(rule, index) in policyForm.rules" :key="index">
                <q-item-section>
                  <div class="row q-col-gutter-sm">
                    <q-select
                      v-model="rule.ownership_key"
                      class="col-12 col-lg-4"
                      outlined
                      dense
                      emit-value
                      map-options
                      label="归属定义"
                      :options="ownershipIdentityOptions"
                      :rules="requiredRules"
                      @update:model-value="syncPolicyRuleOwnership(rule)"
                    />
                    <q-select
                      v-model="rule.scope_source"
                      class="col-12 col-md-4 col-lg-2"
                      outlined
                      dense
                      emit-value
                      map-options
                      label="范围来源"
                      :options="scopeSourceOptionsFor(rule.dimension_id)"
                      :rules="requiredRules"
                      @update:model-value="syncPolicyRuleOperator(rule)"
                    />
                    <q-select
                      v-model="rule.relation"
                      class="col-12 col-md-4 col-lg-2"
                      outlined
                      dense
                      emit-value
                      map-options
                      label="关系"
                      :options="relationOptions"
                      :rules="requiredRules"
                    />
                    <q-select
                      v-model="rule.operator"
                      class="col-12 col-md-4 col-lg-2"
                      outlined
                      dense
                      emit-value
                      map-options
                      label="操作符"
                      :options="operatorOptions"
                      :rules="requiredRules"
                    />
                    <div class="col-auto row items-center">
                      <q-btn
                        flat
                        round
                        dense
                        color="negative"
                        icon="delete"
                        @click="removePolicyRule(index)"
                      >
                        <q-tooltip>删除规则</q-tooltip>
                      </q-btn>
                    </div>
                    <q-input
                      v-if="rule.scope_source === 'specified_values'"
                      v-model="rule.specified_values_text"
                      class="col-12 col-md-8"
                      outlined
                      dense
                      label="指定值"
                      hint="多个值使用逗号分隔"
                      :rules="requiredRules"
                    />
                    <q-input
                      v-if="rule.relation === 'self_and_descendants'"
                      v-model="rule.structure_code"
                      class="col-12 col-md-4"
                      outlined
                      dense
                      label="组织视图编码"
                      :rules="requiredRules"
                    />
                  </div>
                </q-item-section>
              </q-item>
              <q-item v-if="policyForm.rules.length === 0">
                <q-item-section class="text-grey-7">至少配置一条策略规则</q-item-section>
              </q-item>
            </q-list>
          </div>
          <q-toggle v-model="policyForm.state" label="配置状态启用" />
        </div>
      </template>

      <template v-else>
        <div class="row q-col-gutter-md">
          <q-select
            v-model="grantForm.subject_type"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            label="授权主体类型"
            :options="subjectTypeOptions"
            :rules="requiredRules"
            @update:model-value="loadSubjectOptions"
          />
          <q-select
            v-model="grantForm.subject_id"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            use-input
            input-debounce="300"
            label="授权主体"
            :options="subjectOptions"
            :rules="requiredRules"
            @filter="filterSubjectOptions"
          />
          <q-select
            v-model="grantForm.resource_id"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            use-input
            label="数据资源"
            :options="resourceOptions"
            :rules="requiredRules"
            @filter="filterResourceOptions"
            @update:model-value="loadGrantOperations"
          />
          <q-select
            v-model="grantForm.operation"
            class="col-12 col-md-6"
            outlined
            dense
            emit-value
            map-options
            label="资源操作"
            :options="grantOperationOptions"
            :rules="requiredRules"
          />
          <q-select
            v-model="grantForm.policy_id"
            class="col-12"
            outlined
            dense
            emit-value
            map-options
            use-input
            label="权限策略"
            :options="policyOptions"
            :rules="requiredRules"
            @filter="filterPolicyOptions"
          />
          <q-input
            v-model="grantForm.valid_from"
            class="col-12 col-md-6"
            outlined
            dense
            type="date"
            label="有效期开始"
          />
          <q-input
            v-model="grantForm.valid_to"
            class="col-12 col-md-6"
            outlined
            dense
            type="date"
            label="有效期结束"
          />
          <q-input
            v-model="grantForm.description"
            class="col-12"
            outlined
            dense
            type="textarea"
            autogrow
            label="说明"
          />
        </div>
      </template>
    </q-form>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { type QForm, useQuasar } from 'quasar'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import {
  type DataGrant,
  type DataGrantSaveReq,
  type DataOwnership,
  type DataOwnershipSaveReq,
  type DataPermissionBindingType,
  type DataPermissionDimension,
  type DataPermissionOperation,
  type DataPermissionResourceType,
  type DataPermissionValueType,
  type DataPolicy,
  type DataPolicyRuleSaveReq,
  type DataPolicySaveReq,
  type DataResource,
  type DataResourceSaveReq,
  useDataPermissionConfigApi,
} from 'src/api/services/data-permission-config'
import { useTableApi, type Table, type TableField } from 'src/api/services/sys-table'
import { useRoleApi, type Role } from 'src/api/services/sys-role'
import { useSysUserApi, type User } from 'src/api/services/sys-user'
import type { Query } from 'src/types/global'

type DialogKind = 'resource' | 'ownership' | 'policy' | 'grant'
type EditValue = DataResource | DataOwnership | DataPolicy | DataGrant | null

interface SelectOption<T = string | number> {
  label: string
  value: T
}

interface EditablePolicyRule extends DataPolicyRuleSaveReq {
  ownership_key: string
  specified_values_text: string
}

type EditableResourceForm = Omit<DataResourceSaveReq, 'operations'> & {
  operations: DataPermissionOperation[]
}

type EditablePolicyForm = Omit<DataPolicySaveReq, 'rules'> & {
  rules: EditablePolicyRule[]
}

const props = defineProps<{
  modelValue: boolean
  kind: DialogKind
  editData?: EditValue
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'saved'): void
}>()

const api = useDataPermissionConfigApi()
const tableApi = useTableApi()
const roleApi = useRoleApi()
const userApi = useSysUserApi()
const $q = useQuasar()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const formRef = ref<QForm | null>(null)
const saving = ref(false)
const metadataFieldLoading = ref(false)

const requiredRules = [
  (value: unknown) => (value !== null && value !== undefined && value !== '') || '必填',
]
const positiveIdRules = [(value: number) => Number(value) > 0 || '请输入有效ID']
const atLeastOneRules = [(value: unknown[]) => value?.length > 0 || '至少选择一项']

const resourceTypeOptions: SelectOption<DataPermissionResourceType>[] = [
  { label: '低代码数据表', value: 'low_code_table' },
  { label: '业务服务', value: 'business_service' },
  { label: '报表', value: 'report' },
]
const bindingTypeOptions: SelectOption<DataPermissionBindingType>[] = [
  { label: '元数据字段', value: 'metadata_field' },
  { label: '注册字段', value: 'registered_field' },
]
const valueTypeOptions: SelectOption<DataPermissionValueType>[] = [
  { label: '数字ID', value: 'bigint' },
  { label: '字符串编码', value: 'string' },
]
const operationOptions: SelectOption<DataPermissionOperation>[] = [
  { label: '查询', value: 'query' },
  { label: '详情', value: 'detail' },
  { label: '新增', value: 'create' },
  { label: '修改', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '导出', value: 'export' },
  { label: '运行', value: 'run' },
]
const relationOptions = [
  { label: '精确匹配', value: 'exact' },
  { label: '本级及下级', value: 'self_and_descendants' },
]
const operatorOptions = [
  { label: '等于', value: 'eq' },
  { label: '包含于', value: 'in' },
]
const subjectTypeOptions = [
  { label: '角色', value: 'role' },
  { label: '用户', value: 'user' },
]

const resourceForm = ref<EditableResourceForm>({
  resource_code: '',
  name: '',
  resource_type: 'business_service',
  target: {},
  adapter_code: '',
  description: '',
  state: true,
  operations: ['query', 'detail'],
})
const ownershipForm = ref<DataOwnershipSaveReq>({
  resource_id: 0,
  ownership_code: '',
  dimension_id: 0,
  binding_type: 'metadata_field',
  binding_target: {},
  value_type: 'bigint',
  state: true,
})
const policyForm = ref<EditablePolicyForm>({
  policy_code: '',
  name: '',
  description: '',
  state: true,
  rules: [],
})
const grantForm = ref<DataGrantSaveReq>({
  subject_type: 'role',
  subject_id: 0,
  resource_id: 0,
  operation: 'query',
  policy_id: 0,
  valid_from: null,
  valid_to: null,
  description: '',
  state: true,
})

const resources = ref<DataResource[]>([])
const dimensions = ref<DataPermissionDimension[]>([])
const policies = ref<DataPolicy[]>([])
const ownerships = ref<DataOwnership[]>([])
const tables = ref<Table[]>([])
const roles = ref<Role[]>([])
const users = ref<User[]>([])
const resourceOptions = ref<SelectOption<number>[]>([])
const policyOptions = ref<SelectOption<number>[]>([])
const tableOptions = ref<SelectOption<number>[]>([])
const subjectOptions = ref<SelectOption<number>[]>([])
const metadataFieldOptions = ref<SelectOption<number>[]>([])
const grantOperationOptions = ref<SelectOption<DataPermissionOperation>[]>([])

const title = computed(() => {
  const names = {
    resource: '数据资源',
    ownership: '归属定义',
    policy: '权限策略',
    grant: '权限授权',
  }
  return `${props.editData ? '编辑' : '新增'}${names[props.kind]}`
})
const subtitle = computed(() =>
  props.kind === 'ownership'
    ? '归属定义只描述资源记录的业务归属，不保存SQL或字段表达式'
    : '按照数据权限配置边界维护当前记录',
)
const dialogIcon = computed(
  () =>
    ({
      resource: 'dataset',
      ownership: 'account_tree',
      policy: 'policy',
      grant: 'verified_user',
    })[props.kind],
)

const dimensionOptions = computed<SelectOption<number>[]>(() =>
  dimensions.value.map((item) => ({
    label: `${item.dimension_code} · ${item.name}`,
    value: item.id,
  })),
)

const ownershipIdentityOptions = computed<SelectOption<string>[]>(() => {
  const unique = new Map<string, SelectOption<string>>()
  ownerships.value
    .filter((item) => item.state !== false)
    .forEach((item) => {
      const key = `${item.ownership_code}:${item.dimension_id}`
      const dimension = dimensions.value.find((candidate) => candidate.id === item.dimension_id)
      unique.set(key, {
        label: `${item.ownership_code} · ${dimension?.name || item.dimension_id}`,
        value: key,
      })
    })
  return [...unique.values()]
})

const baseQuery = (keyword = '', num = 200): Query => ({
  page: 1,
  num,
  order: { field: '', is_asc: true },
  expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
  quick_query: { keyword },
})

const loadBaseOptions = async () => {
  const [resourceResult, dimensionResult, policyResult, ownershipResult] = await Promise.all([
    api.queryResources(baseQuery('', 500)),
    api.queryDimensions(baseQuery('', 500)),
    api.queryPolicies(baseQuery('', 500)),
    api.queryOwnerships(baseQuery('', 500)),
  ])
  resources.value = resourceResult.data || []
  dimensions.value = dimensionResult.data || []
  policies.value = policyResult.data || []
  ownerships.value = ownershipResult.data || []
  resourceOptions.value = toResourceOptions(resources.value)
  policyOptions.value = toPolicyOptions(policies.value)
}

const loadTables = async (keyword = '') => {
  const result = await tableApi.queryTable(baseQuery(keyword, 100))
  tables.value = result.data || []
  tableOptions.value = tables.value.map((item) => ({
    label: `${item.table_name} · ${item.table_code}`,
    value: item.id,
  }))
}

const filterTableOptions = (value: string, update: (callback: () => void) => void) => {
  void loadTables(value).then(() => update(() => undefined))
}

const toResourceOptions = (items: DataResource[], keyword = '') =>
  items
    .filter((item) =>
      `${item.resource_code} ${item.name}`.toLowerCase().includes(keyword.toLowerCase()),
    )
    .map((item) => ({ label: `${item.resource_code} · ${item.name}`, value: item.id }))

const toPolicyOptions = (items: DataPolicy[], keyword = '') =>
  items
    .filter((item) =>
      `${item.policy_code} ${item.name}`.toLowerCase().includes(keyword.toLowerCase()),
    )
    .map((item) => ({ label: `${item.policy_code} · ${item.name}`, value: item.id }))

const filterResourceOptions = (value: string, update: (callback: () => void) => void) =>
  update(() => {
    resourceOptions.value = toResourceOptions(resources.value, value)
  })

const filterPolicyOptions = (value: string, update: (callback: () => void) => void) =>
  update(() => {
    policyOptions.value = toPolicyOptions(policies.value, value)
  })

const loadSubjectOptions = async () => {
  grantForm.value.subject_id = 0
  if (grantForm.value.subject_type === 'role') {
    const result = await roleApi.queryRole(baseQuery('', 100))
    roles.value = result.data || []
    subjectOptions.value = roles.value.map((item) => ({ label: item.name, value: item.id }))
    return
  }
  const result = await userApi.queryUser(baseQuery('', 100))
  users.value = result.data || []
  subjectOptions.value = users.value.map((item) => ({ label: item.user_name, value: item.id }))
}

const filterSubjectOptions = (value: string, update: (callback: () => void) => void) => {
  const query = value.toLowerCase()
  update(() => {
    subjectOptions.value =
      grantForm.value.subject_type === 'role'
        ? roles.value
            .filter((item) => item.name.toLowerCase().includes(query))
            .map((item) => ({ label: item.name, value: item.id }))
        : users.value
            .filter((item) => item.user_name.toLowerCase().includes(query))
            .map((item) => ({ label: item.user_name, value: item.id }))
  })
}

const clearResourceTarget = () => {
  resourceForm.value.target = {}
}

const loadMetadataFields = async () => {
  ownershipForm.value.binding_target = {}
  const resourceId = ownershipForm.value.resource_id
  if (!resourceId) return
  metadataFieldLoading.value = true
  try {
    const resource = await api.getResource(resourceId)
    const tableId = resource.data?.target?.reference_id
    if (!tableId || resource.data.resource_type !== 'low_code_table') {
      metadataFieldOptions.value = []
      return
    }
    const result = await tableApi.queryTableById(tableId)
    metadataFieldOptions.value = (result.data?.table_fields || [])
      .filter(isOwnershipCandidateField)
      .map((field) => ({
        label: `${field.field_name} · ${field.field_code}`,
        value: field.id,
      }))
  } finally {
    metadataFieldLoading.value = false
  }
}

const isOwnershipCandidateField = (field: TableField) => {
  const code = field.field_code.toLowerCase()
  return (
    field.state !== false &&
    !field.is_primary_key &&
    !['id', 'name', 'path', 'level', 'parent_id', 'parent_node_id'].includes(code) &&
    !['gmt_', 'source_', 'create_', 'modify_', 'delete_'].some((prefix) => code.startsWith(prefix))
  )
}

const syncOwnershipValueType = () => {
  const dimension = dimensions.value.find((item) => item.id === ownershipForm.value.dimension_id)
  if (dimension) ownershipForm.value.value_type = dimension.value_type
}

const addPolicyRule = () => {
  policyForm.value.rules.push({
    sequence: policyForm.value.rules.length + 1,
    dimension_id: 0,
    ownership_code: '',
    ownership_key: '',
    scope_source: '',
    relation: 'exact',
    operator: 'in',
    specified_values_text: '',
    state: true,
  })
}

const removePolicyRule = (index: number) => {
  policyForm.value.rules.splice(index, 1)
  policyForm.value.rules.forEach((rule, ruleIndex) => {
    rule.sequence = ruleIndex + 1
  })
}

const syncPolicyRuleOwnership = (rule: EditablePolicyRule) => {
  const [ownershipCode, dimensionId] = rule.ownership_key.split(':')
  rule.ownership_code = ownershipCode || ''
  rule.dimension_id = Number(dimensionId || 0)
  rule.scope_source = scopeSourceOptionsFor(rule.dimension_id)[0]?.value || ''
  syncPolicyRuleOperator(rule)
}

const scopeSourceOptionsFor = (dimensionId: number) => {
  const dimension = dimensions.value.find((item) => item.id === dimensionId)
  const specified = { label: '指定值', value: 'specified_values' }
  if (dimension?.dimension_code === 'legal_entity') {
    return [{ label: '当前有效法人', value: 'effective_legal_entities' }, specified]
  }
  if (['management_org', 'management_organization'].includes(dimension?.dimension_code || '')) {
    return [{ label: '当前有效组织', value: 'effective_org_units' }, specified]
  }
  if (['employee', 'enterprise_employee'].includes(dimension?.dimension_code || '')) {
    return [{ label: '当前员工', value: 'current_employee' }, specified]
  }
  return [specified]
}

const syncPolicyRuleOperator = (rule: EditablePolicyRule) => {
  if (rule.scope_source === 'current_employee') rule.operator = 'eq'
  if (['effective_legal_entities', 'effective_org_units'].includes(rule.scope_source)) {
    rule.operator = 'in'
  }
}

const loadGrantOperations = async () => {
  grantForm.value.operation = 'query'
  if (!grantForm.value.resource_id) {
    grantOperationOptions.value = []
    return
  }
  const result = await api.listResourceOperations(grantForm.value.resource_id)
  const operations = (result.data || []).filter((item) => item.state !== false)
  grantOperationOptions.value = operationOptions.filter((option) =>
    operations.some((item) => item.operation === option.value),
  )
  grantForm.value.operation = grantOperationOptions.value[0]?.value || 'query'
}

const resetForms = () => {
  resourceForm.value = {
    resource_code: '',
    name: '',
    resource_type: 'business_service',
    target: {},
    adapter_code: '',
    description: '',
    state: true,
    operations: ['query', 'detail'],
  }
  ownershipForm.value = {
    resource_id: 0,
    ownership_code: '',
    dimension_id: 0,
    binding_type: 'metadata_field',
    binding_target: {},
    value_type: 'bigint',
    state: true,
  }
  policyForm.value = {
    policy_code: '',
    name: '',
    description: '',
    state: true,
    rules: [],
  }
  grantForm.value = {
    subject_type: 'role',
    subject_id: 0,
    resource_id: 0,
    operation: 'query',
    policy_id: 0,
    valid_from: null,
    valid_to: null,
    description: '',
    state: true,
  }
}

const initializeEditData = async () => {
  resetForms()
  if (!props.editData) {
    if (props.kind === 'grant') await loadSubjectOptions()
    return
  }
  if (props.kind === 'resource') {
    const value = (await api.getResource((props.editData as DataResource).id)).data
    const operations = await api.listResourceOperations(value.id)
    resourceForm.value = {
      id: value.id,
      resource_code: value.resource_code,
      name: value.name,
      resource_type: value.resource_type,
      target: value.target || {},
      adapter_code: value.adapter_code || '',
      state: value.state !== false,
      operations: (operations.data || [])
        .filter((item) => item.state !== false)
        .map((item) => item.operation),
    }
  } else if (props.kind === 'ownership') {
    const value = (await api.getOwnership((props.editData as DataOwnership).id)).data
    const bindingTarget: DataOwnershipSaveReq['binding_target'] = {}
    if (value.binding_target?.reference_id) {
      bindingTarget.reference_id = value.binding_target.reference_id
    }
    if (value.binding_target?.reference_code) {
      bindingTarget.reference_code = value.binding_target.reference_code
    }
    ownershipForm.value = {
      id: value.id,
      resource_id: value.resource_id,
      ownership_code: value.ownership_code,
      dimension_id: value.dimension_id,
      binding_type: value.binding_type,
      binding_target: bindingTarget,
      value_type: value.value_type,
      state: value.state !== false,
    }
  } else if (props.kind === 'policy') {
    const value = (await api.getPolicy((props.editData as DataPolicy).id)).data
    policyForm.value = {
      id: value.id,
      policy_code: value.policy_code,
      name: value.name,
      state: value.state !== false,
      rules: (value.rules || []).map((rule) => ({
        ...rule,
        ownership_key: `${rule.ownership_code}:${rule.dimension_id}`,
        specified_values_text: Array.isArray(rule.specified_values)
          ? rule.specified_values.join(',')
          : '',
      })),
    }
  }
}

const policyRuleRequest = (rule: EditablePolicyRule): DataPolicyRuleSaveReq => {
  const dimension = dimensions.value.find((item) => item.id === rule.dimension_id)
  const specifiedValues =
    rule.scope_source === 'specified_values'
      ? rule.specified_values_text
          .split(',')
          .map((value) => value.trim())
          .filter(Boolean)
          .map((value) => (dimension?.value_type === 'bigint' ? Number(value) : value))
      : undefined
  const request: DataPolicyRuleSaveReq = {
    sequence: rule.sequence,
    dimension_id: rule.dimension_id,
    ownership_code: rule.ownership_code,
    scope_source: rule.scope_source,
    relation: rule.relation,
    operator: rule.operator,
  }
  if (specifiedValues) request.specified_values = specifiedValues
  if (rule.relation === 'self_and_descendants' && rule.structure_code) {
    request.structure_code = rule.structure_code
  }
  if (rule.state !== undefined) request.state = rule.state
  return request
}

const submit = async () => {
  const valid = await formRef.value?.validate()
  if (!valid) return
  if (props.kind === 'policy' && policyForm.value.rules.length === 0) {
    $q.notify({ type: 'warning', message: '至少配置一条策略规则' })
    return
  }
  saving.value = true
  try {
    if (props.kind === 'resource') {
      const { operations, ...request } = resourceForm.value
      if (props.editData) {
        await api.updateResource(request)
        await api.replaceResourceOperations(request.id!, operations)
      } else {
        await api.createResource({
          ...request,
          operations: operations.map((operation) => ({ operation, state: true })),
        })
      }
    } else if (props.kind === 'ownership') {
      if (props.editData) await api.updateOwnership(ownershipForm.value)
      else await api.createOwnership(ownershipForm.value)
    } else if (props.kind === 'policy') {
      const { rules: editableRules, ...policyRequest } = policyForm.value
      const rules = editableRules.map(policyRuleRequest)
      if (props.editData) {
        await api.updatePolicy(policyRequest)
        await api.replacePolicyRules(policyRequest.id!, rules)
      } else {
        await api.createPolicy({ ...policyRequest, rules })
      }
    } else {
      await api.createGrant(grantForm.value)
    }
    $q.notify({ type: 'positive', message: '保存成功' })
    visible.value = false
    emit('saved')
  } finally {
    saving.value = false
  }
}

watch(
  () => props.modelValue,
  async (open) => {
    if (!open) return
    await loadBaseOptions()
    if (props.kind === 'resource') await loadTables()
    await initializeEditData()
    await nextTick()
    formRef.value?.resetValidation()
  },
)
</script>
