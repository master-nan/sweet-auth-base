<template>
  <form-dialog-shell
    v-model="visible"
    :title="t('ui.detailsOfTheExternalSystem')"
    :subtitle="detail?.system_code || t('ui.readingConfiguration')"
    icon="dns"
    readonly
    :loading="loading"
    width="min(920px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="external-system-detail">
      <section>
        <div class="external-system-detail__section-title">{{ t('ui.basicInfo') }}</div>
        <div class="external-system-detail__grid">
          <div v-for="item in basicItems" :key="item.label" class="external-system-detail__item">
            <div class="external-system-detail__label">{{ item.label }}</div>
            <div class="external-system-detail__value">{{ item.value || '-' }}</div>
          </div>
        </div>
      </section>
      <q-separator />
      <section>
        <div class="external-system-detail__section-title">{{ t('ui.connectAndManage') }}</div>
        <div class="external-system-detail__grid">
          <div class="external-system-detail__item external-system-detail__item--wide">
            <div class="external-system-detail__label">{{ t('ui.basicAddress') }}</div>
            <div class="external-system-detail__value text-mono">{{ detail.base_url }}</div>
          </div>
          <div class="external-system-detail__item">
            <div class="external-system-detail__label">{{ t('ui.head') }}</div>
            <div class="external-system-detail__value">{{ detail.owner_name }}</div>
          </div>
          <div class="external-system-detail__item">
            <div class="external-system-detail__label">{{ t('ui.responsibleIdentification') }}</div>
            <div class="external-system-detail__value text-mono">{{ detail.owner_identifier }}</div>
          </div>
          <div class="external-system-detail__item external-system-detail__item--wide">
            <div class="external-system-detail__label">{{ t('ui.description') }}</div>
            <div class="external-system-detail__value">{{ detail.description || '-' }}</div>
          </div>
        </div>
      </section>
    </div>
    <div v-else class="external-system-detail__loading">
      <q-spinner-dots color="primary" size="36px" />
    </div>
    <template #footer-actions>
      <q-btn
        v-if="detail"
        flat
        color="primary"
        icon="api"
        :label="t('ui.viewInterfaces')"
        @click="emit('show-interfaces', detail.id)"
      />
      <q-btn
        v-if="detail"
        flat
        color="primary"
        icon="key"
        :label="t('ui.viewCertificates')"
        @click="emit('show-credentials', detail.id)"
      />
    </template>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import FormDialogShell from '@/components/FormDialog/FormDialogShell.vue'
import { type ExternalSystemDetail, useIntegrationApi } from '@/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

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
  get hr() {
    return t('ui.humanResourcesSystem')
  },
  get erp() {
    return t('ui.enterpriseResourcePlanning')
  },
  get tms() {
    return t('ui.transportationManagementSystem')
  },
  get wms() {
    return t('ui.warehouseManagementSystem')
  },
  get other() {
    return t('ui.otherSystem')
  },
}
const statusLabels: Record<string, string> = {
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
const basicItems = computed(() => [
  {
    get label() {
      return t('ui.systemEncoding')
    },
    value: detail.value?.system_code,
  },
  {
    get label() {
      return t('ui.systemName')
    },
    value: detail.value?.name,
  },
  {
    get label() {
      return t('ui.systemType')
    },
    value: typeLabels[detail.value?.system_type || ''],
  },
  {
    get label() {
      return t('ui.status')
    },
    value: statusLabels[detail.value?.status || ''],
  },
  {
    get label() {
      return t('ui.version')
    },
    value: detail.value?.revision,
  },
  {
    get label() {
      return t('ui.updatedAt')
    },
    value: detail.value?.gmt_modify,
  },
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
