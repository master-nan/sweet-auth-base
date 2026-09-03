<template>
  <q-drawer v-model="visible" side="right" overlay bordered :width="Math.min(520, $q.screen.width)">
    <div class="column fit">
      <q-toolbar>
        <q-toolbar-title>{{ t('ui.programmeDetails') }}</q-toolbar-title>
        <q-btn
          flat
          round
          dense
          icon="refresh"
          :aria-label="t('ui.refreshProgramDetails')"
          :loading="loading"
          @click="load"
        >
          <q-tooltip>{{ t('ui.refresh') }}</q-tooltip>
        </q-btn>
        <q-btn
          flat
          round
          dense
          icon="close"
          :aria-label="t('ui.closeProgramDetails')"
          @click="visible = false"
        >
          <q-tooltip>{{ t('ui.close') }}</q-tooltip>
        </q-btn>
      </q-toolbar>
      <q-separator />
      <q-scroll-area class="col">
        <div v-if="loading" class="row justify-center q-pa-xl">
          <q-spinner color="primary" size="32px" />
        </div>
        <div v-else-if="error" class="q-pa-lg text-negative">{{ error }}</div>
        <div v-else-if="detail" class="q-pa-md q-gutter-md">
          <div>
            <div class="text-h6">{{ detail.name }}</div>
            <div class="text-caption text-grey-7">{{ t(detail.scope_label) }}</div>
          </div>
          <div class="row q-gutter-sm">
            <status-chip :label="QUERY_SCHEME_TYPE_LABELS[detail.type]" color="primary" />
            <status-chip :label="statusLabel" :color="statusColor" />
            <status-chip v-if="detail.is_default" :label="t('ui.defaultScheme')" color="amber-8" />
            <status-chip
              :label="detail.enabled ? t('ui.activatedStatus') : t('ui.deactivatedStatus')"
              :color="detail.enabled ? 'positive' : 'grey'"
            />
          </div>
          <q-list bordered separator>
            <q-item>
              <q-item-section
                ><q-item-label caption>{{ t('ui.createdBy') }}</q-item-label
                ><q-item-label>{{
                  detail.creator_display_name || '-'
                }}</q-item-label></q-item-section
              >
            </q-item>
            <q-item>
              <q-item-section
                ><q-item-label caption>{{ t('ui.updatedAt') }}</q-item-label
                ><q-item-label>{{ detail.updated_at || '-' }}</q-item-label></q-item-section
              >
            </q-item>
            <q-item v-if="detail.type === QuerySchemeType.ROLE">
              <q-item-section
                ><q-item-label caption>{{ t('ui.roleScope') }}</q-item-label
                ><q-item-label>{{ roleNames }}</q-item-label></q-item-section
              >
            </q-item>
          </q-list>
          <div>
            <div class="text-subtitle1 q-mb-sm">{{ t('ui.queryConditions') }}</div>
            <query-scheme-preview :payload="detail.query_payload" :fields="fields" />
          </div>
          <q-banner v-if="detail.issues.length" class="bg-warning text-dark rounded-borders">
            <div class="text-weight-medium q-mb-xs">
              {{ t('ui.theProgrammeContainsConditionsToBeAddressed') }}
            </div>
            <div v-for="issue in detail.issues" :key="`${issue.path}-${issue.code}`">
              {{ issue.message }}
            </div>
          </q-banner>
        </div>
      </q-scroll-area>
      <q-separator />
      <div v-if="detail" class="row justify-end q-gutter-sm q-pa-md">
        <q-btn
          v-if="detail.type !== QuerySchemeType.PERSONAL"
          outline
          color="primary"
          :label="t('ui.copyToMySchemes')"
          @click="$emit('copy', detail)"
        />
        <q-btn
          v-if="editable"
          color="primary"
          :label="t('ui.edit')"
          @click="$emit('edit', detail)"
        />
      </div>
    </div>
  </q-drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useI18n } from 'vue-i18n'
import { useQuerySchemeApi } from '@/api/services/query-scheme'
import { useTableApi, type TableField } from '@/api/services/sys-table'
import { useRoleApi, type Role } from '@/api/services/sys-role'
import StatusChip from '@/components/Display/StatusChip.vue'
import QuerySchemePreview from '@/components/QueryScheme/QuerySchemePreview.vue'
import {
  QUERY_SCHEME_TYPE_LABELS,
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeDetail,
} from '@/modules/query-scheme/types'

const props = withDefaults(
  defineProps<{ modelValue: boolean; schemeId?: number; editable?: boolean }>(),
  { schemeId: 0, editable: false },
)
const $q = useQuasar()
const { t } = useI18n()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  edit: [detail: QuerySchemeDetail]
  copy: [detail: QuerySchemeDetail]
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const api = useQuerySchemeApi()
const tableApi = useTableApi()
const roleApi = useRoleApi()
const detail = ref<QuerySchemeDetail | null>(null)
const fields = ref<TableField[]>([])
const roles = ref<Role[]>([])
const loading = ref(false)
const error = ref('')
const statusLabel = computed(() =>
  detail.value?.status === QuerySchemeValidationStatus.VALID
    ? t('ui.available')
    : detail.value?.status === QuerySchemeValidationStatus.DEGRADED
      ? t('ui.needsAttention')
      : t('ui.unavailable'),
)
const statusColor = computed(() =>
  detail.value?.status === QuerySchemeValidationStatus.VALID
    ? 'positive'
    : detail.value?.status === QuerySchemeValidationStatus.DEGRADED
      ? 'warning'
      : 'negative',
)
const roleNames = computed(() => {
  const ids = new Set(detail.value?.role_ids || [])
  return (
    roles.value
      .filter((role) => ids.has(role.id))
      .map((role) => role.name)
      .join('、') || '-'
  )
})

const load = async () => {
  if (!props.schemeId) return
  loading.value = true
  error.value = ''
  try {
    const response = await api.detail(props.schemeId)
    detail.value = response.data || null
    if (!detail.value) throw new Error(t('ui.noDetailsOfTheProgrammeExist'))
    const scope = await api.getScopeConfig(detail.value.scope_code)
    const tableCode = scope.data?.table_code
    fields.value = tableCode
      ? (await tableApi.queryRuntimeTableByCode(tableCode)).data?.table_fields || []
      : []
    if (detail.value.type === QuerySchemeType.ROLE) {
      const results = await Promise.allSettled(
        (detail.value.role_ids || []).map((id) => roleApi.queryRoleById(id)),
      )
      roles.value = results.flatMap((result) =>
        result.status === 'fulfilled' && result.value.data ? [result.value.data] : [],
      )
    }
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : t('ui.failedToLoadTheDetailsOfTheProgramme')
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.modelValue, props.schemeId] as const,
  ([open]) => {
    if (open) void load()
  },
)
</script>
