<template>
  <form-dialog-shell
    v-model="visible"
    title="集成凭证详情"
    :subtitle="detail?.credential_code || (loadFailed ? '凭证详情读取失败' : '正在读取凭证元数据')"
    icon="key"
    readonly
    :loading="loading"
    width="min(900px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="credential-detail">
      <section>
        <div class="credential-detail__title">基础信息</div>
        <detail-field-grid :items="basicItems" />
      </section>
      <q-separator />
      <section>
        <div class="credential-detail__title">安全与轮换</div>
        <detail-field-grid :items="securityItems" />
        <q-banner class="credential-detail__notice q-mt-md" rounded
          >轮换历史已写入平台审计日志；本页面不保存或展示历史秘密。</q-banner
        >
      </section>
    </div>
    <div v-else-if="loading" class="credential-detail__loading">
      <q-spinner-dots color="primary" size="36px" />
    </div>
    <div v-else class="credential-detail__error">
      <q-icon name="error_outline" color="negative" size="42px" />
      <div class="text-subtitle1 text-weight-bold">无法读取凭证详情</div>
      <div class="text-body2 text-grey-7">凭证可能已失效，或其所属外部系统已被移除。</div>
      <q-btn outline color="primary" icon="refresh" label="重新加载" @click="load" />
    </div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import DetailFieldGrid from 'src/components/Detail/DetailFieldGrid.vue'
import type { DetailFieldItem } from 'src/components/Detail/types'
import { type CredentialDetail, useIntegrationApi } from 'src/api/services/integration'

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const detail = ref<CredentialDetail | null>(null)
const loading = ref(false)
const loadFailed = ref(false)
const typeLabels: Record<string, string> = {
  basic: 'Basic',
  api_key: 'API Key',
  bearer_token: 'Bearer Token',
  oauth_client: 'OAuth Client',
}
const statusLabels: Record<string, string> = {
  draft: '草稿',
  active: '已启用',
  disabled: '已停用',
  revoked: '已吊销',
  expired: '已过期',
}
const basicItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        {
          label: '所属系统',
          value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）`,
        },
        { label: '凭证编码', value: detail.value.credential_code },
        { label: '凭证名称', value: detail.value.name },
        {
          label: '类型',
          value: typeLabels[detail.value.credential_type] || detail.value.credential_type,
        },
        {
          label: '状态',
          value: statusLabels[detail.value.effective_status] || detail.value.effective_status,
        },
        { label: '创建时间', value: detail.value.gmt_create },
      ]
    : [],
)
const securityItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        { label: '秘密版本', value: `v${detail.value.version}` },
        { label: '指纹摘要', value: detail.value.fingerprint_summary || '-' },
        { label: '有效期', value: detail.value.expires_at || '长期有效' },
        { label: '最近轮换', value: detail.value.rotated_at || '尚未轮换' },
        { label: '描述', value: detail.value.description || '-', fullWidth: true },
      ]
    : [],
)
const load = async () => {
  if (!props.id) return
  loading.value = true
  loadFailed.value = false
  detail.value = null
  try {
    detail.value = (await api.getCredential(props.id)).data
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}
watch(
  () => [props.modelValue, props.id] as const,
  ([open]) => {
    if (!open || !props.id) {
      detail.value = null
      loadFailed.value = false
      return
    }
    void load()
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.credential-detail {
  display: grid;
  gap: 24px;
  padding: 4px 6px 20px;
}
.credential-detail__title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}
.credential-detail__notice {
  background: var(--app-primary-soft);
  color: inherit;
}
.credential-detail__loading,
.credential-detail__error {
  min-height: 240px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  text-align: center;
}
</style>
