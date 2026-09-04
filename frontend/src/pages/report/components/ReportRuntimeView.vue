<template>
  <div class="runtime-view">
    <div class="runtime-head">
      <div>
        <div class="report-title">
          {{ runtimeConfig?.report_name || runtimeReport?.report_name || runtimeTitle }}
        </div>
        <div class="report-caption">{{ runtimeSourceName }} · {{ runtimeDescription }}</div>
        <q-chip
          v-if="runtimeVersionNo"
          dense
          square
          color="primary"
          text-color="white"
          class="runtime-version-chip"
        >
          {{ t('ui.currentVersionV') }}{{ runtimeVersionNo }}
        </q-chip>
      </div>
      <q-space />
      <q-btn
        v-if="allowExport"
        outline
        color="primary"
        icon="download"
        :label="t('ui.exportCsv')"
        :disable="!runtimeRows.length || exporting"
        :loading="exporting"
        @click="exportCurrentRuntime"
      />
      <q-btn v-if="showClose" flat round icon="close" @click="$emit('close')" />
    </div>
    <div class="runtime-filters">
      <q-input
        v-model="runtimeKeyword"
        dense
        outlined
        clearable
        :label="t('ui.keyword')"
        class="runtime-filter"
        @keyup.enter="loadRuntimePreview"
      />
      <template v-for="param in runtimeParameters" :key="param.id">
        <sweet-date-time-picker
          v-if="param.type === 'date'"
          :model-value="runtimeScalarValue(param.id)"
          type="date"
          dense
          :label="param.label"
          class="runtime-filter"
          @update:model-value="runtimeFilterValues[param.id] = $event"
        />
        <div v-else-if="param.type === 'date_range'" class="runtime-range-filter">
          <sweet-date-time-picker
            :model-value="runtimeRangeValue(param.id, 0)"
            type="date"
            dense
            :label="t('ui.rangeStart', { label: param.label })"
            class="runtime-filter"
            @update:model-value="setRuntimeRangeValue(param.id, 0, $event)"
          />
          <sweet-date-time-picker
            :model-value="runtimeRangeValue(param.id, 1)"
            type="date"
            dense
            :label="t('ui.rangeEnd', { label: param.label })"
            class="runtime-filter"
            @update:model-value="setRuntimeRangeValue(param.id, 1, $event)"
          />
        </div>
        <q-input
          v-else
          :model-value="runtimeScalarValue(param.id)"
          dense
          outlined
          clearable
          :type="param.type === 'number' ? 'number' : 'text'"
          :label="param.label"
          :placeholder="param.placeholder"
          class="runtime-filter"
          @update:model-value="runtimeFilterValues[param.id] = $event"
          @keyup.enter="loadRuntimePreview"
        />
      </template>
      <q-select
        dense
        outlined
        :label="t('ui.scopeOfCompetence')"
        :model-value="t('ui.inheritCurrentMenuDataPrivileges')"
        class="runtime-filter"
        :options="[t('ui.inheritCurrentMenuDataPrivileges')]"
      />
      <q-btn color="primary" icon="search" :label="t('ui.query')" @click="loadRuntimePreview" />
      <q-btn
        outline
        color="primary"
        icon="restart_alt"
        :label="t('ui.reset')"
        @click="resetRuntimeFilters"
      />
    </div>
    <div class="runtime-body">
      <report-sheet-preview
        :sheet="runtimeSheet"
        :datasets="runtimeDatasets"
        :preview-data="runtimeData"
        :loading="runtimeLoading"
        :report-kind="runtimeConfig?.kind || runtimeReport?.report_kind || 'detail'"
      />
      <div v-if="runtimeDisplayMode === 'paged'" class="runtime-pagination">
        <table-pagination
          v-model:page="runtimePagination.page"
          v-model:pageSize="runtimePagination.rowsPerPage"
          :total="runtimePagination.rowsNumber"
          @update:page="loadRuntimePreview"
        />
      </div>
      <div v-else class="runtime-pagination">
        <span>{{ t('ui.total') }} {{ runtimePagination.rowsNumber }} {{ t('ui.okay') }}</span>
        <q-badge color="primary" :label="t('ui.showAll')" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import TablePagination from '@/components/Table/TablePagination.vue'
import SweetDateTimePicker from '@/components/DateTime/SweetDateTimePicker.vue'
import type { Report } from '@/api/services/report'
import ReportSheetPreview from './ReportSheetPreview.vue'
import { useReportRuntime } from '../composables/useReportRuntime'
import { useReportExport } from '../composables/useReportExport'

const { t } = useI18n({ useScope: 'global' })

const props = withDefaults(
  defineProps<{
    active?: boolean
    report: Report | null
    defaultPageSize?: number | undefined
    allowExport?: boolean
    showClose?: boolean
    mode?: 'center' | 'manage'
    menuId?: number
  }>(),
  {
    active: true,
    allowExport: true,
    showClose: false,
    mode: 'center',
    menuId: 0,
  },
)

defineEmits<{
  close: []
}>()

const {
  runtimeReport,
  runtimeData,
  runtimeLoading,
  runtimeKeyword,
  runtimeFilterValues,
  runtimePagination,
  runtimeRows,
  runtimeConfig,
  runtimeDatasets,
  runtimeSheet,
  runtimeDisplayMode,
  runtimeMenuId,
  runtimeVersionNo,
  runtimeParameters,
  openRuntime,
  clearRuntime,
  loadRuntimePreview,
  resetRuntimeFilters,
  buildRuntimeParameterValues,
  runtimeScalarValue,
  runtimeRangeValue,
  setRuntimeRangeValue,
} = useReportRuntime()

const { exporting, exportRuntimeCsv } = useReportExport()

const runtimeTitle = computed(() =>
  props.mode === 'manage' ? t('ui.reportRunPreview') : t('ui.reportRun'),
)
const runtimeSourceName = computed(
  () =>
    runtimeDatasets.value.find((dataset) => dataset.primary)?.source_code ||
    runtimeDatasets.value[0]?.source_code ||
    runtimeReport.value?.data_source_name ||
    '-',
)
const runtimeDescription = computed(() => {
  if (runtimeConfig.value) {
    return runtimeConfig.value.description || t('ui.runPreviewSessionsToApplyBackendDataPrivileges')
  }
  return runtimeReport.value?.description || t('ui.runPreviewSessionsToApplyBackendDataPrivileges')
})

watch(
  () => [props.active, props.report?.id, props.menuId] as const,
  ([active]) => {
    if (active && props.report) {
      openRuntime(props.report, props.defaultPageSize, props.menuId)
      void loadRuntimePreview()
      return
    }
    clearRuntime()
  },
  { immediate: true },
)

async function exportCurrentRuntime() {
  await exportRuntimeCsv(runtimeReport.value, {
    menuId: runtimeMenuId.value,
    keyword: runtimeKeyword.value,
    parameters: buildRuntimeParameterValues(),
    total: runtimePagination.value.rowsNumber,
    rowCount: runtimeRows.value.length,
    pageSize: runtimePagination.value.rowsPerPage,
  })
}
</script>

<style scoped lang="scss">
.runtime-view {
  display: flex;
  height: 100%;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  background: #fff;
}

.runtime-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-height: 72px;
  padding: 12px 20px;
  border-bottom: 1px solid #dfe5f2;
  background: #fff;
}

.runtime-filters {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 12px 20px;
  border-bottom: 1px solid #dfe5f2;
  background: #fff;
}

.report-title {
  color: #172033;
  font-size: 20px;
  font-weight: 800;
}

.report-caption {
  margin-top: 4px;
  color: #71809a;
  line-height: 1.5;
}

.runtime-filter {
  width: 220px;
}

.runtime-version-chip {
  margin-top: 8px;
}

.runtime-range-filter {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.runtime-body {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  overflow: hidden;
  background: #fff;
}

.runtime-body :deep(.report-sheet-preview) {
  min-height: 0;
  flex: 1;
  border: 0;
  border-radius: 0;
}

.runtime-body :deep(.report-sheet-preview__scroll) {
  height: 100%;
  max-height: none;
  padding: 32px 40px;
  background: #fff;
}

.runtime-pagination {
  display: flex;
  min-height: 58px;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding: 10px 20px;
  border-top: 1px solid #dfe5f2;
  background: #fff;
  color: #71809a;
}

body.body--dark {
  .runtime-view,
  .runtime-body {
    background: var(--app-dark-page);
  }

  .runtime-head,
  .runtime-filters,
  .runtime-pagination {
    border-color: var(--app-dark-border);
    background: var(--app-dark-surface);
  }

  .runtime-body :deep(.report-sheet-preview__scroll),
  .runtime-body :deep(.report-sheet-preview__grid) {
    background: var(--app-dark-surface);
  }

  .report-title {
    color: #f4f6fb;
  }

  .report-caption,
  .runtime-pagination {
    color: #aeb8ca;
  }
}
</style>
