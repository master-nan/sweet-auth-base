<template>
  <form-dialog-shell
    v-model="visible"
    title="接口定义详情"
    :subtitle="detail ? `${detail.interface_code} · v${detail.version}` : '正在读取技术契约'"
    icon="api"
    readonly
    :loading="loading"
    width="min(960px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="interface-detail">
      <section>
        <div class="interface-detail__title">基础信息</div>
        <div class="interface-detail__grid">
          <div v-for="item in basicItems" :key="item.label" class="interface-detail__item">
            <div class="interface-detail__label">{{ item.label }}</div>
            <div class="interface-detail__value">{{ item.value ?? '-' }}</div>
          </div>
        </div>
      </section>
      <q-separator />
      <section>
        <div class="interface-detail__title">技术契约</div>
        <div class="interface-detail__grid">
          <div v-for="item in contractItems" :key="item.label" class="interface-detail__item">
            <div class="interface-detail__label">{{ item.label }}</div>
            <div class="interface-detail__value" :class="{ 'text-mono': item.mono }">{{ item.value ?? '-' }}</div>
          </div>
          <div class="interface-detail__item interface-detail__wide">
            <div class="interface-detail__label">描述</div>
            <div class="interface-detail__value">{{ detail.description || '-' }}</div>
          </div>
        </div>
      </section>
    </div>
    <div v-else class="interface-detail__loading"><q-spinner-dots color="primary" size="36px" /></div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { type InterfaceDefinitionDetail, useIntegrationApi } from 'src/api/services/integration'

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const detail = ref<InterfaceDefinitionDetail | null>(null)
const loading = ref(false)
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const statusLabels = { draft: '草稿', enabled: '已启用', disabled: '已停用' }
const basicItems = computed(() => detail.value ? [
  { label: '所属系统', value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）` },
  { label: '接口编码', value: detail.value.interface_code },
  { label: '接口名称', value: detail.value.name },
  { label: '版本', value: `v${detail.value.version}` },
  { label: '状态', value: statusLabels[detail.value.status] },
  { label: '更新时间', value: detail.value.gmt_modify },
] : [])
const contractItems = computed(() => detail.value ? [
  { label: '协议', value: detail.value.protocol.toUpperCase(), mono: false },
  { label: 'HTTP Method', value: detail.value.http_method, mono: true },
  { label: '相对路径', value: detail.value.relative_path, mono: true },
  { label: '超时', value: `${detail.value.timeout_seconds} 秒`, mono: false },
  { label: '响应大小限制', value: `${(detail.value.response_limit / 1024).toLocaleString()} KiB`, mono: false },
  { label: '认证引用', value: detail.value.credential ? `${detail.value.credential.name}（${detail.value.credential.credential_code}）` : '未配置', mono: false },
  { label: '凭证状态', value: detail.value.credential?.effective_status || '-', mono: false },
  { label: '重试策略', value: detail.value.retry_policy_id ? '已配置重试策略' : '未配置', mono: false },
] : [])

const load = async () => {
  if (!props.id) return
  loading.value = true
  try { detail.value = (await api.getInterfaceDefinition(props.id)).data } finally { loading.value = false }
}
watch(() => [props.modelValue, props.id] as const, ([open]) => { if (open) void load(); else detail.value = null }, { immediate: true })
</script>

<style scoped lang="scss">
.interface-detail { display: grid; gap: 24px; padding: 4px 6px 20px; }
.interface-detail__title { margin-bottom: 16px; font-size: 16px; font-weight: 700; }
.interface-detail__grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px 36px; }
.interface-detail__item { min-width: 0; padding-bottom: 12px; border-bottom: 1px solid rgba(15, 23, 42, 0.08); }
.interface-detail__wide { grid-column: 1 / -1; }
.interface-detail__label { margin-bottom: 7px; color: #8290a8; font-size: 12px; }
.interface-detail__value { overflow-wrap: anywhere; color: #172033; font-weight: 600; }
.interface-detail__loading { min-height: 260px; display: grid; place-items: center; }
.body--dark .interface-detail__item { border-color: rgba(255, 255, 255, 0.1); }
.body--dark .interface-detail__value { color: #e7ebf5; }
@media (max-width: 700px) { .interface-detail__grid { grid-template-columns: 1fr; } .interface-detail__wide { grid-column: auto; } }
</style>
