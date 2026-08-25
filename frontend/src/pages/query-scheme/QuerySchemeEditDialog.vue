<template>
  <q-dialog v-model="visible" :maximized="$q.screen.lt.md" persistent>
    <q-card style="width: 900px; max-width: 100%">
      <q-card-section class="row items-center">
        <div class="text-h6">{{ detail ? '编辑查询方案' : '新建查询方案' }}</div>
        <q-space />
        <q-btn v-close-popup flat round dense icon="close" aria-label="关闭方案编辑窗口"><q-tooltip>关闭</q-tooltip></q-btn>
      </q-card-section>
      <q-separator />
      <q-card-section class="q-gutter-md">
        <div class="row q-col-gutter-md">
          <q-select class="col-12 col-md-6" v-model="scopeCode" outlined dense emit-value map-options :options="scopeOptions" label="所属页面" :disable="!!detail" @update:model-value="loadScope" />
          <q-input class="col-12 col-md-6" v-model="name" outlined dense label="方案名称" maxlength="64" />
        </div>
        <div class="row items-center q-gutter-lg">
          <q-checkbox v-if="schemeType === QuerySchemeType.PERSONAL || schemeType === QuerySchemeType.PAGE_DEFAULT" v-model="isDefault" :label="schemeType === QuerySchemeType.PERSONAL ? '设为我的默认方案' : '设为页面默认方案'" />
        </div>
        <role-select
          v-if="schemeType === QuerySchemeType.ROLE"
          v-model="roleIds"
          label="适用角色（最多32个）"
          :rules="[validateRoleIds]"
        />
        <q-banner v-if="scopeError" class="bg-warning text-dark rounded-borders">{{ scopeError }}</q-banner>
        <q-btn outline color="primary" icon="tune" label="编辑查询条件" :disable="!scopeConfig" @click="openConditionEditor" />
        <query-scheme-preview v-if="scopeConfig" :payload="payload" :fields="fields" :menu-id="scopeConfig.menu_id" />
      </q-card-section>
      <q-card-actions align="right" class="q-pa-md">
        <q-btn v-close-popup flat label="取消" />
        <q-btn color="primary" label="保存" :loading="loading" :disable="!valid" @click="submit" />
      </q-card-actions>
    </q-card>
  </q-dialog>

  <advanced-query
    v-model="showQuery"
    v-model:query-model="conditionDraft"
    v-model:bindings="conditionBindingDraft"
    :fields="fields"
    :menu-id="scopeConfig?.menu_id || 0"
    title="编辑查询条件"
    usage="scheme-condition-editor"
    @confirm="confirmConditionEdit"
    @cancel="showQuery = false"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import cloneDeep from 'lodash/cloneDeep'
import { useQuasar } from 'quasar'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import QuerySchemePreview from 'src/components/QueryScheme/QuerySchemePreview.vue'
import RoleSelect from 'src/components/Select/RoleSelect.vue'
import { useQuerySchemeApi } from 'src/api/services/query-scheme'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import { normalizeQuerySchemePayload } from 'src/utils/query-state'
import { ExpressionLogic } from 'src/types/enum'
import type { Query } from 'src/types/global'
import {
  QuerySchemeType,
  type QueryScopeConfig,
  type QuerySchemeBinding,
  type QuerySchemeDetail,
  type QuerySchemeType as SchemeType,
} from 'src/modules/query-scheme/types'

const props = defineProps<{
  modelValue: boolean
  schemeType: SchemeType
  detail?: QuerySchemeDetail | null
  scopeOptions: Array<{ label: string; value: string }>
}>()
const $q = useQuasar()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [] }>()
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const api = useQuerySchemeApi()
const tableApi = useTableApi()
const name = ref('')
const scopeCode = ref('')
const isDefault = ref(false)
const roleIds = ref<number[]>([])
const fields = ref<TableField[]>([])
const scopeConfig = ref<QueryScopeConfig | null>(null)
const scopeError = ref('')
const loading = ref(false)
const showQuery = ref(false)
const bindings = ref<QuerySchemeBinding[]>([])
const query = ref<Query>({ page: 1, num: 20, order: { field: '', is_asc: false }, quick_query: { keyword: '' }, expressions: [{ logic: ExpressionLogic.AND, rules: [{ field: '', value: null }], nested: [] }] })
const conditionDraft = ref<Query>(cloneDeep(query.value))
const conditionBindingDraft = ref<QuerySchemeBinding[]>([])
const payload = computed(() => normalizeQuerySchemePayload(query.value, bindings.value))
const valid = computed(() => !!scopeCode.value && !!name.value.trim() && (props.schemeType !== QuerySchemeType.ROLE || (roleIds.value.length > 0 && roleIds.value.length <= 32)))
const validateRoleIds = (value: number[]) =>
  (value.length > 0 && value.length <= 32) || '请选择1至32个角色'

const loadScope = async () => {
  scopeConfig.value = null
  fields.value = []
  scopeError.value = ''
  if (!scopeCode.value) return
  try {
    const scope = await api.getScopeConfig(scopeCode.value)
    scopeConfig.value = scope.data || null
    if (!scopeConfig.value) throw new Error('查询范围不可用')
    fields.value = (await tableApi.queryRuntimeTableByCode(scopeConfig.value.table_code)).data?.table_fields || []
  } catch (cause) {
    scopeError.value = cause instanceof Error ? cause.message : '查询范围加载失败'
  }
}

const reset = async () => {
  const detail = props.detail
  name.value = detail?.name || ''
  scopeCode.value = detail?.scope_code || props.scopeOptions[0]?.value || ''
  isDefault.value = detail?.is_default ?? props.schemeType === QuerySchemeType.PAGE_DEFAULT
  roleIds.value = [...(detail?.role_ids || [])]
  const source = detail?.query_payload
  query.value = { page: 1, num: 20, order: source?.order || { field: '', is_asc: false }, quick_query: source?.quick_query || { keyword: '' }, expressions: source?.expressions || [{ logic: ExpressionLogic.AND, rules: [{ field: '', value: null }], nested: [] }] }
  bindings.value = [...(source?.bindings || [])]
  await loadScope()
}

const openConditionEditor = () => {
  conditionDraft.value = cloneDeep(query.value)
  conditionBindingDraft.value = cloneDeep(bindings.value)
  showQuery.value = true
}

const confirmConditionEdit = () => {
  query.value = cloneDeep(conditionDraft.value)
  bindings.value = cloneDeep(conditionBindingDraft.value)
  showQuery.value = false
}

const submit = async () => {
  if (!valid.value) return
  loading.value = true
  try {
    if (props.schemeType === QuerySchemeType.PERSONAL) {
      if (!props.detail) throw new Error('请从业务页面新建个人方案')
      await api.updatePersonal(props.detail.id, { name: name.value.trim(), query_payload: payload.value, is_default: isDefault.value, revision: props.detail.revision })
    } else if (props.detail) {
      await api.updateShared(props.detail.id, { name: name.value.trim(), query_payload: payload.value, is_default: props.schemeType === QuerySchemeType.PAGE_DEFAULT ? isDefault.value : false, role_ids: props.schemeType === QuerySchemeType.ROLE ? roleIds.value : [], revision: props.detail.revision })
    } else {
      await api.createShared({ name: name.value.trim(), scope_code: scopeCode.value, scheme_type: props.schemeType as Exclude<SchemeType, 'PERSONAL'>, query_payload: payload.value, is_default: props.schemeType === QuerySchemeType.PAGE_DEFAULT ? isDefault.value : false, enabled: true, role_ids: props.schemeType === QuerySchemeType.ROLE ? roleIds.value : [] })
    }
    visible.value = false
    emit('saved')
  } catch {
    // 安全的用户提示由共享HTTP拦截器统一处理，Dialog不展示后端技术正文。
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (open) => { if (open) void reset() })
</script>
