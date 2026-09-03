<template>
  <form-dialog-shell
    v-model="visible"
    :title="t('ui.interfaceDefinitionDetails')"
    :subtitle="
      detail ? `${detail.interface_code} · v${detail.version}` : t('ui.readingTechnologyCompacts')
    "
    icon="api"
    readonly
    :loading="loading"
    width="min(960px, calc(100vw - 48px))"
  >
    <div v-if="detail" class="interface-detail">
      <section>
        <div class="interface-detail__title">{{ t('ui.basicInfo') }}</div>
        <detail-field-grid :items="basicItems" />
      </section>
      <q-separator />
      <section>
        <div class="interface-detail__title">{{ t('ui.technicalCompact') }}</div>
        <detail-field-grid :items="contractItems" />
      </section>
      <q-separator />
      <section>
        <div class="interface-detail__title">{{ t('ui.contractForRequestedParameters') }}</div>
        <div class="interface-detail__hint">
          {{ t('ui.thisDisplayShowsTheNameLocationAndTypeOfParameter') }}
        </div>
        <q-markup-table
          v-if="detail.input_contract.parameters.length"
          flat
          bordered
          dense
          class="interface-detail__parameters"
        >
          <thead>
            <tr>
              <th class="text-left">{{ t('ui.parameters') }}</th>
              <th class="text-left">{{ t('ui.position') }}</th>
              <th class="text-left">{{ t('ui.type') }}</th>
              <th class="text-center">{{ t('ui.required') }}</th>
              <th class="text-center">{{ t('ui.allowMultipleValues') }}</th>
              <th class="text-right">{{ t('ui.maximumLength') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="parameter in detail.input_contract.parameters"
              :key="`${parameter.location}:${parameter.code}`"
            >
              <td>
                <div class="text-weight-medium">{{ parameter.name || parameter.code }}</div>
                <div v-if="parameter.name" class="text-caption text-grey-7 text-mono">
                  {{ parameter.code }}
                </div>
              </td>
              <td>{{ locationLabels[parameter.location] }}</td>
              <td>{{ dataTypeLabels[parameter.data_type] }}</td>
              <td class="text-center">{{ parameter.required ? t('ui.yes') : t('ui.no') }}</td>
              <td class="text-center">{{ parameter.allow_multiple ? t('ui.yes') : t('ui.no') }}</td>
              <td class="text-right">{{ parameter.max_length || '-' }}</td>
            </tr>
          </tbody>
        </q-markup-table>
        <div v-else class="interface-detail__empty">
          {{ t('ui.theInterfaceDoesNotDeclareVariableRequestParameters') }}
        </div>
      </section>
      <q-separator />
      <section>
        <div class="interface-detail__title">{{ t('ui.responseProcessing') }}</div>
        <div class="interface-detail__hint">
          {{ t('ui.thePlatformReadsTheResultsAccordingToTheResponseSize') }}
        </div>
      </section>
    </div>
    <div v-else class="interface-detail__loading">
      <q-spinner-dots color="primary" size="36px" />
    </div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import FormDialogShell from '@/components/FormDialog/FormDialogShell.vue'
import DetailFieldGrid from '@/components/Detail/DetailFieldGrid.vue'
import type { DetailFieldItem } from '@/components/Detail/types'
import {
  type InterfaceDefinitionDetail,
  type InterfaceInputDataType,
  type InterfaceInputLocation,
  useIntegrationApi,
} from '@/api/services/integration'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{ modelValue: boolean; id: number }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const api = useIntegrationApi()
const detail = ref<InterfaceDefinitionDetail | null>(null)
const loading = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const statusLabels = {
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
const locationLabels: Record<InterfaceInputLocation, string> = {
  path: 'Path',
  query: 'Query',
  header: 'Header',
  body: 'JSON Body',
}
const dataTypeLabels: Record<InterfaceInputDataType, string> = {
  get string() {
    return t('ui.string')
  },
  get integer() {
    return t('ui.integer')
  },
  get number() {
    return t('ui.valueLabel')
  },
  get boolean() {
    return t('ui.boolean')
  },
  get object() {
    return t('ui.object')
  },
  get array() {
    return t('ui.array')
  },
}
const basicItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        {
          get label() {
            return t('ui.owningSystem')
          },
          value: `${detail.value.external_system.name}（${detail.value.external_system.system_code}）`,
        },
        {
          get label() {
            return t('ui.apiCode')
          },
          value: detail.value.interface_code,
        },
        {
          get label() {
            return t('ui.apiName')
          },
          value: detail.value.name,
        },
        {
          get label() {
            return t('ui.version')
          },
          value: `v${detail.value.version}`,
        },
        {
          get label() {
            return t('ui.status')
          },
          value: statusLabels[detail.value.status],
        },
        {
          get label() {
            return t('ui.updatedAt')
          },
          value: detail.value.gmt_modify,
        },
      ]
    : [],
)
const contractItems = computed<DetailFieldItem[]>(() =>
  detail.value
    ? [
        {
          get label() {
            return t('ui.protocolLabel')
          },
          value: detail.value.protocol.toUpperCase(),
        },
        { label: 'HTTP Method', value: detail.value.http_method },
        {
          get label() {
            return t('ui.relativePathLabel')
          },
          value: detail.value.relative_path,
        },
        {
          get label() {
            return t('ui.timeout')
          },
          value: t('ui.secondsValue', { value1: detail.value.timeout_seconds }),
        },
        {
          get label() {
            return t('ui.responseSizeLimit')
          },
          value: `${(detail.value.response_limit / 1024).toLocaleString()} KiB`,
        },
        {
          get label() {
            return t('ui.authenticationReference')
          },
          value: detail.value.credential
            ? `${detail.value.credential.name}（${detail.value.credential.credential_code}）`
            : t('ui.notConfigured'),
        },
        {
          get label() {
            return t('ui.documentStatus')
          },
          value: detail.value.credential?.effective_status || '-',
        },
        {
          get label() {
            return t('ui.retryPolicy')
          },
          value: detail.value.retry_policy
            ? `${detail.value.retry_policy.policy_name}（${detail.value.retry_policy.policy_code} · v${detail.value.retry_policy.version}）`
            : t('ui.notConfigured'),
        },
        {
          get label() {
            return t('ui.policyStatus')
          },
          value: detail.value.retry_policy ? statusLabels[detail.value.retry_policy.status] : '-',
        },
        {
          get label() {
            return t('ui.description')
          },
          value: detail.value.description || '-',
          fullWidth: true,
        },
      ]
    : [],
)

const load = async () => {
  if (!props.id) return
  loading.value = true
  try {
    detail.value = (await api.getInterfaceDefinition(props.id)).data
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
.interface-detail {
  display: grid;
  gap: 24px;
  padding: 4px 6px 20px;
}
.interface-detail__title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}
.interface-detail__hint {
  margin: -6px 0 14px;
  color: var(--app-text-muted);
  font-size: 13px;
  line-height: 1.7;
}
.interface-detail__parameters {
  background: var(--app-surface);
}
.interface-detail__empty {
  padding: 18px;
  border: 1px dashed var(--app-border);
  color: var(--app-text-muted);
  text-align: center;
}
.interface-detail__loading {
  min-height: 260px;
  display: grid;
  place-items: center;
}
</style>
