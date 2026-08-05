<template>
  <form-dialog-shell v-model="visible" title="集成凭证详情" :subtitle="detail?.credential_code || '正在读取凭证元数据'" icon="key" readonly :loading="loading" width="min(900px, calc(100vw - 48px))">
    <div v-if="detail" class="credential-detail">
      <section>
        <div class="credential-detail__title">基础信息</div>
        <div class="credential-detail__grid">
          <div v-for="item in basicItems" :key="item.label" class="credential-detail__item"><div class="credential-detail__label">{{ item.label }}</div><div class="credential-detail__value">{{ item.value || '-' }}</div></div>
        </div>
      </section>
      <q-separator />
      <section>
        <div class="credential-detail__title">安全与轮换</div>
        <div class="credential-detail__grid">
          <div v-for="item in securityItems" :key="item.label" class="credential-detail__item"><div class="credential-detail__label">{{ item.label }}</div><div class="credential-detail__value">{{ item.value || '-' }}</div></div>
          <div class="credential-detail__item credential-detail__wide"><div class="credential-detail__label">描述</div><div class="credential-detail__value">{{ detail.description || '-' }}</div></div>
        </div>
        <q-banner class="credential-detail__notice q-mt-md" rounded>轮换历史已写入平台审计日志；本页面不保存或展示历史秘密。</q-banner>
      </section>
    </div>
    <div v-else class="credential-detail__loading"><q-spinner-dots color="primary" size="36px" /></div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { type CredentialDetail, useIntegrationApi } from 'src/api/services/integration'

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const visible = computed({ get: () => props.modelValue, set: (value) => emit('update:modelValue', value) })
const detail = ref<CredentialDetail | null>(null)
const loading = ref(false)
const typeLabels: Record<string, string> = { basic: 'Basic', api_key: 'API Key', bearer_token: 'Bearer Token', oauth_client: 'OAuth Client' }
const statusLabels: Record<string, string> = { draft: '草稿', active: '已启用', disabled: '已停用', revoked: '已吊销', expired: '已过期' }
const basicItems = computed(() => detail.value ? [
  { label: '所属系统', value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）` },
  { label: '凭证编码', value: detail.value.credential_code }, { label: '凭证名称', value: detail.value.name },
  { label: '类型', value: typeLabels[detail.value.credential_type] }, { label: '状态', value: statusLabels[detail.value.effective_status] },
  { label: '创建时间', value: detail.value.gmt_create },
] : [])
const securityItems = computed(() => detail.value ? [
  { label: '秘密版本', value: `v${detail.value.version}` }, { label: '指纹摘要', value: detail.value.fingerprint_summary },
  { label: '有效期', value: detail.value.expires_at || '长期有效' }, { label: '最近轮换', value: detail.value.rotated_at || '尚未轮换' },
] : [])
watch(() => [props.modelValue, props.id] as const, async ([open]) => {
  if (!open || !props.id) { detail.value = null; return }
  loading.value = true
  try { detail.value = (await api.getCredential(props.id)).data } finally { loading.value = false }
}, { immediate: true })
</script>

<style scoped lang="scss">
.credential-detail { display: grid; gap: 24px; padding: 4px 6px 20px; }
.credential-detail__title { margin-bottom: 16px; font-size: 16px; font-weight: 700; }
.credential-detail__grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px 36px; }
.credential-detail__item { min-width: 0; padding-bottom: 12px; border-bottom: 1px solid rgba(15, 23, 42, 0.08); }
.credential-detail__wide { grid-column: 1 / -1; }
.credential-detail__label { margin-bottom: 7px; color: #8290a8; font-size: 12px; }
.credential-detail__value { overflow-wrap: anywhere; color: #172033; font-weight: 600; }
.credential-detail__notice { background: var(--app-primary-soft); color: inherit; }
.credential-detail__loading { min-height: 240px; display: grid; place-items: center; }
.body--dark .credential-detail__item { border-color: rgba(255, 255, 255, 0.1); }
.body--dark .credential-detail__value { color: #e7ebf5; }
@media (max-width: 700px) { .credential-detail__grid { grid-template-columns: 1fr; } .credential-detail__wide { grid-column: auto; } }
</style>
