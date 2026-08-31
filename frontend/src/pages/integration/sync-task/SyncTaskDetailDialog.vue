<template>
  <form-dialog-shell
    v-model="visible"
    :title="t('ui.synchroniseTaskDetails')"
    :subtitle="
      detail ? `${detail.task_code} · v${detail.version}` : t('ui.readingTaskConfiguration')
    "
    icon="sync_alt"
    readonly
    :loading="loading"
    width="min(980px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="sync-task-detail">
      <div v-for="item in items" :key="item.label" class="sync-task-detail__item">
        <div class="text-caption text-grey-7">{{ item.label }}</div>
        <div class="text-body1">{{ item.value }}</div>
      </div>
      <div class="sync-task-detail__item sync-task-detail__wide">
        <div class="text-caption text-grey-7">{{ t('ui.enterASummaryOfThePlan') }}</div>
        <div class="text-body1">
          {{ t('ui.version') }} {{ detail.input_plan_summary.version }}
          {{ t('ui.staticParametersCaption') }}
          {{ detail.input_plan_summary.static_parameter_count }} {{ t('ui.individual') }}
          {{
            detail.input_plan_summary.has_window_bindings
              ? t('ui.includeWindowBinding')
              : t('ui.noWindowBound')
          }}
          ·
          {{
            detail.input_plan_summary.window_mode === 'lower_bound_only'
              ? t('ui.onlyLowerBoundsNoUpperBounds')
              : t('ui.fullStartWindow')
          }}
        </div>
      </div>
      <div class="sync-task-detail__item sync-task-detail__wide">
        <div class="text-caption text-grey-7">{{ t('ui.description') }}</div>
        <div class="text-body1">{{ detail.description || '-' }}</div>
      </div>
    </div>
    <div v-else class="row justify-center q-pa-xl">
      <q-spinner-dots color="primary" size="36px" />
    </div>
  </form-dialog-shell>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { type SyncTaskDetail, useIntegrationApi } from 'src/api/services/integration'
import { formatRuntimeDateTime } from 'src/pages/integration/runtime-display'

const { t } = useI18n({ useScope: 'global' })
const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const detail = ref<SyncTaskDetail | null>(null)
const loading = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const status = {
  get draft() {
    return t('ui.draft')
  },
  get enabled() {
    return t('ui.activatedStatus')
  },
  get disabled() {
    return t('ui.deactivatedStatus')
  },
}
const items = computed(() =>
  detail.value
    ? [
        {
          get label() {
            return t('ui.taskNameLabel')
          },
          value: detail.value.task_name,
        },
        {
          get label() {
            return t('ui.status')
          },
          value: status[detail.value.status],
        },
        {
          get label() {
            return t('ui.externalSystemLabel')
          },
          value: `${detail.value.external_system.name} (${detail.value.external_system.code})`,
        },
        {
          get label() {
            return t('ui.apiVersion')
          },
          value: `${detail.value.interface_definition.name} · v${detail.value.interface_definition.version}`,
        },
        {
          label: 'Consumer',
          value: `${detail.value.consumer.code} · v${detail.value.consumer.version}`,
        },
        {
          get label() {
            return t('ui.scheduling')
          },
          value:
            detail.value.schedule_type === 'cron'
              ? `${detail.value.cron_summary} · ${detail.value.timezone}`
              : t('ui.manualTriggerOnly'),
        },
        {
          get label() {
            return t('ui.nextAutomaticRun')
          },
          value:
            detail.value.schedule_type === 'cron'
              ? formatRuntimeDateTime(detail.value.next_scheduled_at)
              : '-',
        },
        {
          get label() {
            return t('ui.lastAutomaticRun')
          },
          value:
            detail.value.schedule_type === 'cron'
              ? formatRuntimeDateTime(detail.value.last_scheduled_at)
              : '-',
        },
        {
          label: 'Checkpoint',
          value:
            detail.value.checkpoint_mode === 'timestamp'
              ? formatRuntimeDateTime(
                  detail.value.checkpoint_at || detail.value.initial_checkpoint_at,
                )
              : t('ui.none'),
        },
        {
          get label() {
            return t('ui.lookbackSlicing')
          },
          value:
            detail.value.checkpoint_mode === 'timestamp'
              ? t('ui.secSec', {
                  value1: detail.value.lookback_seconds,
                  value2: detail.value.window_slice_seconds,
                })
              : '-',
        },
      ]
    : [],
)
watch(
  () => [props.modelValue, props.id] as const,
  async ([open, id]) => {
    if (!open || !id) return
    loading.value = true
    try {
      detail.value = (await api.getSyncTask(id)).data || null
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
</script>
<style scoped lang="scss">
.sync-task-detail {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 20px 24px;
  padding: 8px 4px 20px;
}
.sync-task-detail__item {
  min-width: 0;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--app-border);
}
.sync-task-detail__wide {
  grid-column: 1 / -1;
}
@media (max-width: 760px) {
  .sync-task-detail {
    grid-template-columns: 1fr;
  }
  .sync-task-detail__wide {
    grid-column: auto;
  }
}
</style>
