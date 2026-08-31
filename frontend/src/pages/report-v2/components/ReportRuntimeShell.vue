<template>
  <div class="runtime-shell">
    <section class="runtime-head">
      <div>
        <div class="menu-name">{{ menuName || t('ui.developRunpage') }}</div>
        <div class="runtime-title">{{ title }}</div>
        <div class="runtime-caption">
          <q-chip
            v-if="versionNo"
            dense
            square
            color="blue-1"
            text-color="primary"
            :label="t('ui.currentVersionNumber', { version: versionNo })"
          />
          <q-chip
            dense
            square
            :color="statusMeta.color"
            text-color="white"
            :label="statusMeta.label"
          />
          <q-chip
            dense
            square
            color="blue-grey-1"
            text-color="blue-grey-8"
            :label="runtimeEntryTypeLabel"
          />
          <span v-if="reportCode">{{ t('ui.code') }}{{ reportCode }}</span>
          <span v-if="menuId">menu_id：{{ menuId }}</span>
          <span v-if="permissionTableCode"
            >{{ t('ui.listOfCompetence') }}{{ permissionTableCode }}</span
          >
          <span v-if="sourceType">{{ t('ui.dataSource') }}{{ sourceType }}</span>
          <span>{{ displayDescription }}</span>
        </div>
      </div>
      <div class="runtime-actions">
        <q-btn
          flat
          color="primary"
          icon="arrow_back"
          :label="t('ui.backToTheDesk')"
          @click="$emit('back')"
        />
        <q-btn
          outline
          color="primary"
          icon="refresh"
          :label="t('ui.query')"
          :loading="loading"
          @click="$emit('run')"
        />
        <q-btn
          color="primary"
          unelevated
          icon="file_download"
          :label="t('ui.exportCsv')"
          :loading="exporting"
          @click="$emit('export')"
        />
      </div>
    </section>

    <section class="runtime-filter-panel">
      <div class="panel-title">{{ t('ui.queryParameters') }}</div>
      <report-parameter-form
        :keyword="props.keyword"
        :parameters="props.parameters"
        :model-value="props.parameterValues"
        :control-metas="props.controlMetas"
        :loading="props.parameterLoading"
        :disabled="loading ?? false"
        @update:keyword="$emit('update:keyword', $event)"
        @update:model-value="$emit('update:parameter-values', $event)"
        @search="$emit('run')"
        @reset="$emit('reset')"
      />
    </section>

    <section class="runtime-result-panel">
      <div class="panel-title">{{ t('ui.runResults') }}</div>
      <report-data-table
        row-key="order_no"
        :rows="rows"
        :columns="columns"
        :loading="loading ?? false"
        :page="page"
        :page-size="pageSize"
        :total="total"
        @update:page="$emit('update:page', $event)"
        @update:page-size="$emit('update:pageSize', $event)"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed } from 'vue'
import type { QTableProps } from 'quasar'
import type { ReportParameter } from 'src/api/services/report'
import type { ReportRuntimeParameterValue } from '../composables/useReportRuntime'
import type { ReportParameterControlMeta } from '../composables/useReportParameterControls'
import ReportDataTable from './ReportDataTable.vue'
import ReportParameterForm from './ReportParameterForm.vue'

const { t } = useI18n({ useScope: 'global' })

type ReportStatus = 'draft' | 'published' | 'disabled' | string
type RuntimeEntryType = 'menu' | 'development'

const props = withDefaults(
  defineProps<{
    title: string
    reportCode?: string
    status?: ReportStatus
    sourceType?: string
    menuName?: string
    runtimeEntryType?: RuntimeEntryType
    menuId?: number | undefined
    permissionTableCode?: string
    description?: string
    versionNo?: number
    keyword?: string
    parameters?: ReportParameter[]
    parameterValues?: Record<string, ReportRuntimeParameterValue>
    controlMetas?: Record<string, ReportParameterControlMeta>
    parameterLoading?: boolean
    rows: unknown[]
    columns: QTableProps['columns']
    total: number
    page: number
    pageSize: number
    loading?: boolean
    exporting?: boolean
  }>(),
  {
    reportCode: '',
    status: '',
    sourceType: '',
    menuName: '',
    runtimeEntryType: 'development',
    permissionTableCode: '',
    description: '',
    versionNo: 0,
    keyword: '',
    parameters: () => [],
    parameterValues: () => ({}),
    controlMetas: () => ({}),
    parameterLoading: false,
    loading: false,
    exporting: false,
  },
)

const displayDescription = computed(
  () => props.description || t('ui.genericReportRunningPageSkeleton'),
)

const statusMeta = computed(() => {
  if (props.status === 'published')
    return {
      get label() {
        return t('ui.published')
      },
      color: 'positive',
    }
  if (props.status === 'disabled')
    return {
      get label() {
        return t('ui.deactivatedStatus')
      },
      color: 'negative',
    }
  return {
    get label() {
      return props.status ? t('ui.draft') : t('ui.unknownStatus')
    },
    color: 'grey-7',
  }
})

const runtimeEntryTypeLabel = computed(() =>
  props.runtimeEntryType === 'menu' ? t('ui.menuRun') : t('ui.developmentRun'),
)

defineEmits<{
  back: []
  run: []
  reset: []
  export: []
  'update:keyword': [value: string]
  'update:parameter-values': [value: Record<string, ReportRuntimeParameterValue>]
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()
</script>

<style scoped lang="scss">
.runtime-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.runtime-head,
.runtime-filter-panel,
.runtime-result-panel {
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;
  padding: 14px;
}

.runtime-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.menu-name {
  margin-bottom: 4px;
  color: #2f6fed;
  font-size: 13px;
  font-weight: 700;
}

.runtime-title {
  color: #172033;
  font-size: 22px;
  font-weight: 700;
}

.runtime-caption,
.runtime-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 6px;
  color: #667085;
  font-size: 13px;
}

.panel-title {
  margin-bottom: 12px;
  color: #172033;
  font-weight: 700;
}
</style>
