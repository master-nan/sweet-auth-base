<template>
  <q-drawer v-model="visible" side="right" overlay bordered :width="Math.min(520, $q.screen.width)">
    <div class="column fit">
      <q-toolbar>
        <q-toolbar-title>方案详情</q-toolbar-title>
        <q-btn flat round dense icon="refresh" aria-label="刷新方案详情" :loading="loading" @click="load">
          <q-tooltip>刷新</q-tooltip>
        </q-btn>
        <q-btn flat round dense icon="close" aria-label="关闭方案详情" @click="visible = false">
          <q-tooltip>关闭</q-tooltip>
        </q-btn>
      </q-toolbar>
      <q-separator />
      <q-scroll-area class="col">
        <div v-if="loading" class="row justify-center q-pa-xl"><q-spinner color="primary" size="32px" /></div>
        <div v-else-if="error" class="q-pa-lg text-negative">{{ error }}</div>
        <div v-else-if="detail" class="q-pa-md q-gutter-md">
          <div>
            <div class="text-h6">{{ detail.name }}</div>
            <div class="text-caption text-grey-7">{{ t(detail.scope_label) }}</div>
          </div>
          <div class="row q-gutter-sm">
            <status-chip :label="QUERY_SCHEME_TYPE_LABELS[detail.type]" color="primary" />
            <status-chip :label="statusLabel" :color="statusColor" />
            <status-chip v-if="detail.is_default" label="默认方案" color="amber-8" />
            <status-chip :label="detail.enabled ? '已启用' : '已停用'" :color="detail.enabled ? 'positive' : 'grey'" />
          </div>
          <q-list bordered separator>
            <q-item>
              <q-item-section><q-item-label caption>创建人</q-item-label><q-item-label>{{ detail.creator_display_name || '-' }}</q-item-label></q-item-section>
            </q-item>
            <q-item>
              <q-item-section><q-item-label caption>更新时间</q-item-label><q-item-label>{{ detail.updated_at || '-' }}</q-item-label></q-item-section>
            </q-item>
            <q-item v-if="detail.type === QuerySchemeType.ROLE">
              <q-item-section><q-item-label caption>角色范围</q-item-label><q-item-label>{{ roleNames }}</q-item-label></q-item-section>
            </q-item>
          </q-list>
          <div>
            <div class="text-subtitle1 q-mb-sm">查询条件</div>
            <query-scheme-preview :payload="detail.query_payload" :fields="fields" />
          </div>
          <q-banner v-if="detail.issues.length" class="bg-warning text-dark rounded-borders">
            <div class="text-weight-medium q-mb-xs">该方案包含需要处理的条件</div>
            <div v-for="issue in detail.issues" :key="`${issue.path}-${issue.code}`">{{ issue.message }}</div>
          </q-banner>
        </div>
      </q-scroll-area>
      <q-separator />
      <div v-if="detail" class="row justify-end q-gutter-sm q-pa-md">
        <q-btn v-if="detail.type !== QuerySchemeType.PERSONAL" outline color="primary" label="复制为我的方案" @click="$emit('copy', detail)" />
        <q-btn v-if="editable" color="primary" label="编辑" @click="$emit('edit', detail)" />
      </div>
    </div>
  </q-drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useI18n } from 'vue-i18n'
import { useQuerySchemeApi } from 'src/api/services/query-scheme'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import { useRoleApi, type Role } from 'src/api/services/sys-role'
import StatusChip from 'src/components/Display/StatusChip.vue'
import QuerySchemePreview from 'src/components/QueryScheme/QuerySchemePreview.vue'
import {
  QUERY_SCHEME_TYPE_LABELS,
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeDetail,
} from 'src/modules/query-scheme/types'

const props = withDefaults(defineProps<{ modelValue: boolean; schemeId?: number; editable?: boolean }>(), { schemeId: 0, editable: false })
const $q = useQuasar()
const { t } = useI18n()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  edit: [detail: QuerySchemeDetail]
  copy: [detail: QuerySchemeDetail]
}>()
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const api = useQuerySchemeApi()
const tableApi = useTableApi()
const roleApi = useRoleApi()
const detail = ref<QuerySchemeDetail | null>(null)
const fields = ref<TableField[]>([])
const roles = ref<Role[]>([])
const loading = ref(false)
const error = ref('')
const statusLabel = computed(() => detail.value?.status === QuerySchemeValidationStatus.VALID ? '可用' : detail.value?.status === QuerySchemeValidationStatus.DEGRADED ? '需要修复' : '不可用')
const statusColor = computed(() => detail.value?.status === QuerySchemeValidationStatus.VALID ? 'positive' : detail.value?.status === QuerySchemeValidationStatus.DEGRADED ? 'warning' : 'negative')
const roleNames = computed(() => {
  const ids = new Set(detail.value?.role_ids || [])
  return roles.value.filter((role) => ids.has(role.id)).map((role) => role.name).join('、') || '-'
})

const load = async () => {
  if (!props.schemeId) return
  loading.value = true
  error.value = ''
  try {
    const response = await api.detail(props.schemeId)
    detail.value = response.data || null
    if (!detail.value) throw new Error('方案详情不存在')
    const scope = await api.getScopeConfig(detail.value.scope_code)
    const tableCode = scope.data?.table_code
    fields.value = tableCode ? (await tableApi.queryRuntimeTableByCode(tableCode)).data?.table_fields || [] : []
    if (detail.value.type === QuerySchemeType.ROLE) {
      const results = await Promise.allSettled(
        (detail.value.role_ids || []).map((id) => roleApi.queryRoleById(id)),
      )
      roles.value = results.flatMap((result) =>
        result.status === 'fulfilled' && result.value.data ? [result.value.data] : [],
      )
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '方案详情加载失败'
  } finally {
    loading.value = false
  }
}

watch(() => [props.modelValue, props.schemeId] as const, ([open]) => { if (open) void load() })
</script>
