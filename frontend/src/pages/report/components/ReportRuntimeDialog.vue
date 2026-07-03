<template>
  <q-dialog
    :model-value="modelValue"
    maximized
    @hide="handleDialogValue(false)"
    @update:model-value="handleDialogValue"
  >
    <q-card class="runtime-dialog">
      <q-card-section class="runtime-head">
        <div>
          <div class="report-title">{{ runtimeReport?.report_name || runtimeTitle }}</div>
          <div class="report-caption">
            {{ runtimeReport?.data_source_name || '-' }} ·
            {{ runtimeReport?.description || '运行预览会应用后端数据权限' }}
          </div>
          <q-chip
            v-if="runtimeVersionNo"
            dense
            square
            color="primary"
            text-color="white"
            class="runtime-version-chip"
          >
            当前版本：V{{ runtimeVersionNo }}
          </q-chip>
        </div>
        <q-space />
        <q-btn
          v-if="allowExport"
          outline
          color="primary"
          icon="download"
          label="导出 CSV"
          :disable="!runtimeRows.length || exporting"
          :loading="exporting"
          @click="exportCurrentRuntime"
        />
        <q-btn flat round icon="close" @click="handleDialogValue(false)" />
      </q-card-section>
      <q-separator />
      <q-card-section class="runtime-filters">
        <q-input
          v-model="runtimeKeyword"
          dense
          outlined
          clearable
          label="关键词"
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
              :label="`${param.label}开始`"
              class="runtime-filter"
              @update:model-value="setRuntimeRangeValue(param.id, 0, $event)"
            />
            <sweet-date-time-picker
              :model-value="runtimeRangeValue(param.id, 1)"
              type="date"
              dense
              :label="`${param.label}结束`"
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
          label="权限范围"
          model-value="继承当前菜单数据权限"
          class="runtime-filter"
          :options="['继承当前菜单数据权限']"
        />
        <q-btn color="primary" icon="search" label="查询" @click="loadRuntimePreview" />
        <q-btn
          outline
          color="primary"
          icon="restart_alt"
          label="重置"
          @click="resetRuntimeFilters"
        />
      </q-card-section>
      <q-card-section class="runtime-body">
        <report-sheet-preview
          :sheet="runtimeSheet"
          :datasets="runtimeDatasets"
          :preview-data="runtimeData"
          :loading="runtimeLoading"
          :report-kind="runtimeReport?.report_kind || 'detail'"
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
          <span>共 {{ runtimePagination.rowsNumber }} 行</span>
          <q-badge color="primary" label="全部展示" />
        </div>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import TablePagination from 'components/Table/TablePagination.vue'
import SweetDateTimePicker from 'components/DateTime/SweetDateTimePicker.vue'
import type { Report } from 'src/api/services/report'
import ReportSheetPreview from './ReportSheetPreview.vue'
import { useReportRuntime } from '../composables/useReportRuntime'
import { useReportExport } from '../composables/useReportExport'

const props = withDefaults(defineProps<{
  modelValue: boolean
  report: Report | null
  defaultPageSize?: number | undefined
  allowExport?: boolean
  mode?: 'center' | 'manage'
}>(), {
  allowExport: true,
  mode: 'center',
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const {
  runtimeReport,
  runtimeData,
  runtimeLoading,
  runtimeKeyword,
  runtimeFilterValues,
  runtimePagination,
  runtimeRows,
  runtimeDatasets,
  runtimeSheet,
  runtimeDisplayMode,
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

const runtimeTitle = computed(() => (
  props.mode === 'manage' ? '报表运行预览' : '报表运行'
))

watch(
  () => props.modelValue,
  (visible) => {
    if (visible && props.report) {
      openRuntime(props.report, props.defaultPageSize)
      void loadRuntimePreview()
      return
    }
    if (!visible) clearRuntime()
  },
  { immediate: true },
)

watch(
  () => props.report?.id,
  () => {
    if (props.modelValue && props.report) {
      openRuntime(props.report, props.defaultPageSize)
      void loadRuntimePreview()
    }
  },
)

function handleDialogValue(value: boolean) {
  emit('update:modelValue', value)
  if (!value) clearRuntime()
}

async function exportCurrentRuntime() {
  await exportRuntimeCsv(runtimeReport.value, {
    keyword: runtimeKeyword.value,
    parameters: buildRuntimeParameterValues(),
    total: runtimePagination.value.rowsNumber,
    rowCount: runtimeRows.value.length,
    pageSize: runtimePagination.value.rowsPerPage,
  })
}
</script>

<style scoped lang="scss">
.runtime-dialog {
  display: flex;
  flex-direction: column;
}

.runtime-head,
.runtime-filters {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.report-title {
  font-size: 22px;
  font-weight: 800;
  color: #172033;
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
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.runtime-pagination {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  color: #71809a;
}
</style>
