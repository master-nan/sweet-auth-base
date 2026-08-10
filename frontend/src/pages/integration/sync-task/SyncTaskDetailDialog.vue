<template>
  <form-dialog-shell v-model="visible" title="同步任务详情" :subtitle="detail ? `${detail.task_code} · v${detail.version}` : '正在读取任务配置'" icon="sync_alt" readonly :loading="loading" width="min(980px, calc(100vw - 48px))">
    <div v-if="detail" class="sync-task-detail">
      <div v-for="item in items" :key="item.label" class="sync-task-detail__item"><div class="text-caption text-grey-7">{{ item.label }}</div><div class="text-body1">{{ item.value }}</div></div>
      <div class="sync-task-detail__item sync-task-detail__wide"><div class="text-caption text-grey-7">输入计划摘要</div><div class="text-body1">版本 {{ detail.input_plan_summary.version }} · 静态参数 {{ detail.input_plan_summary.static_parameter_count }} 个 · {{ detail.input_plan_summary.has_window_bindings ? '含窗口绑定' : '无窗口绑定' }}</div></div>
      <div class="sync-task-detail__item sync-task-detail__wide"><div class="text-caption text-grey-7">描述</div><div class="text-body1">{{ detail.description || '-' }}</div></div>
    </div>
    <div v-else class="row justify-center q-pa-xl"><q-spinner-dots color="primary" size="36px" /></div>
  </form-dialog-shell>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { type SyncTaskDetail, useIntegrationApi } from 'src/api/services/integration'
const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi(); const detail = ref<SyncTaskDetail | null>(null); const loading = ref(false)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const status = { draft: '草稿', enabled: '已启用', disabled: '已停用' }
const items = computed(() => detail.value ? [
  { label: '任务名称', value: detail.value.task_name }, { label: '状态', value: status[detail.value.status] },
  { label: '外部系统', value: `${detail.value.external_system.name} (${detail.value.external_system.code})` },
  { label: '接口版本', value: `${detail.value.interface_definition.name} · v${detail.value.interface_definition.version}` },
  { label: 'Consumer', value: `${detail.value.consumer.code} · v${detail.value.consumer.version}` },
  { label: '调度', value: detail.value.schedule_type === 'cron' ? `${detail.value.cron_summary} · ${detail.value.timezone}` : '仅手工触发' },
  { label: 'Checkpoint', value: detail.value.checkpoint_mode === 'timestamp' ? detail.value.checkpoint_at || detail.value.initial_checkpoint_at || '-' : '无' },
  { label: 'Lookback / 切片', value: detail.value.checkpoint_mode === 'timestamp' ? `${detail.value.lookback_seconds} 秒 / ${detail.value.window_slice_seconds} 秒` : '-' },
] : [])
watch(() => [props.modelValue, props.id] as const, async ([open, id]) => { if (!open || !id) return; loading.value = true; try { detail.value = (await api.getSyncTask(id)).data || null } finally { loading.value = false } }, { immediate: true })
</script>
<style scoped lang="scss">
.sync-task-detail { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 20px 24px; padding: 8px 4px 20px; }
.sync-task-detail__item { min-width: 0; padding-bottom: 12px; border-bottom: 1px solid var(--app-border-color); }
.sync-task-detail__wide { grid-column: 1 / -1; }
@media (max-width: 760px) { .sync-task-detail { grid-template-columns: 1fr; } .sync-task-detail__wide { grid-column: auto; } }
</style>
