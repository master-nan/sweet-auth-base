<template>
  <form-dialog-shell
    v-model="visible"
    title="外部系统详情"
    :subtitle="detail?.system_code || '正在读取配置'"
    icon="dns"
    readonly
    :loading="loading"
    width="min(920px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="external-system-detail">
      <section>
        <div class="external-system-detail__section-title">基础信息</div>
        <div class="external-system-detail__grid">
          <div v-for="item in basicItems" :key="item.label" class="external-system-detail__item">
            <div class="external-system-detail__label">{{ item.label }}</div>
            <div class="external-system-detail__value">{{ item.value || '-' }}</div>
          </div>
        </div>
      </section>
      <q-separator />
      <section>
        <div class="external-system-detail__section-title">连接与管理</div>
        <div class="external-system-detail__grid">
          <div class="external-system-detail__item external-system-detail__item--wide">
            <div class="external-system-detail__label">基础地址</div>
            <div class="external-system-detail__value text-mono">{{ detail.base_url }}</div>
          </div>
          <div class="external-system-detail__item">
            <div class="external-system-detail__label">负责人</div>
            <div class="external-system-detail__value">{{ detail.owner_name }}</div>
          </div>
          <div class="external-system-detail__item">
            <div class="external-system-detail__label">负责人标识</div>
            <div class="external-system-detail__value text-mono">{{ detail.owner_identifier }}</div>
          </div>
          <div class="external-system-detail__item external-system-detail__item--wide">
            <div class="external-system-detail__label">描述</div>
            <div class="external-system-detail__value">{{ detail.description || '-' }}</div>
          </div>
        </div>
      </section>
    </div>
    <div v-else class="external-system-detail__loading">
      <q-spinner-dots color="primary" size="36px" />
    </div>
    <template #footer-actions>
      <q-btn v-if="detail" flat color="primary" icon="api" label="查看接口" @click="emit('show-interfaces', detail.id)" />
      <q-btn v-if="detail" flat color="primary" icon="key" label="查看凭证" @click="emit('show-credentials', detail.id)" />
    </template>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import {
  type ExternalSystemDetail,
  useIntegrationApi,
} from 'src/api/services/integration'

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'show-interfaces', id: number): void
  (event: 'show-credentials', id: number): void
}>()
const api = useIntegrationApi()
const loading = ref(false)
const detail = ref<ExternalSystemDetail | null>(null)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const typeLabels: Record<string, string> = {
  hr: '人力资源系统',
  erp: '企业资源计划',
  tms: '运输管理系统',
  wms: '仓储管理系统',
  other: '其他系统',
}
const statusLabels: Record<string, string> = {
  draft: '草稿',
  enabled: '已启用',
  disabled: '已停用',
}
const basicItems = computed(() => [
  { label: '系统编码', value: detail.value?.system_code },
  { label: '系统名称', value: detail.value?.name },
  { label: '系统类型', value: typeLabels[detail.value?.system_type || ''] },
  { label: '状态', value: statusLabels[detail.value?.status || ''] },
  { label: '版本', value: detail.value?.revision },
  { label: '更新时间', value: detail.value?.gmt_modify },
])

const load = async () => {
  if (!props.id) return
  loading.value = true
  try {
    const response = await api.getExternalSystem(props.id)
    detail.value = response.data
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.modelValue, props.id] as const,
  ([open]) => {
    if (open) void load()
    else detail.value = null
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.external-system-detail {
  display: grid;
  gap: 24px;
  padding: 4px 6px 20px;
}

.external-system-detail__section-title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}

.external-system-detail__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px 36px;
}

.external-system-detail__item {
  min-width: 0;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--app-border);
}

.external-system-detail__item--wide {
  grid-column: 1 / -1;
}

.external-system-detail__label {
  margin-bottom: 7px;
  color: var(--app-text-muted);
  font-size: 12px;
}

.external-system-detail__value {
  overflow-wrap: anywhere;
  color: var(--app-text-strong);
  font-weight: 600;
}

.external-system-detail__loading {
  min-height: 260px;
  display: grid;
  place-items: center;
}

@media (max-width: 700px) {
  .external-system-detail__grid {
    grid-template-columns: 1fr;
  }

  .external-system-detail__item--wide {
    grid-column: auto;
  }
}
</style>
