<template>
  <div class="runtime-shell">
    <section class="runtime-head">
      <div>
        <div class="menu-name">{{ menuName || '开发运行页' }}</div>
        <div class="runtime-title">{{ title }}</div>
        <div class="runtime-caption">
          <q-chip v-if="versionNo" dense square color="blue-1" text-color="primary" :label="`当前版本 V${versionNo}`" />
          <q-chip dense square :color="statusMeta.color" text-color="white" :label="statusMeta.label" />
          <q-chip dense square color="blue-grey-1" text-color="blue-grey-8" :label="runtimeEntryTypeLabel" />
          <span v-if="reportCode">编码：{{ reportCode }}</span>
          <span v-if="menuId">menu_id：{{ menuId }}</span>
          <span v-if="permissionTableCode">权限表：{{ permissionTableCode }}</span>
          <span v-if="sourceType">数据来源：{{ sourceType }}</span>
          <span>{{ description }}</span>
        </div>
      </div>
      <div class="runtime-actions">
        <q-btn flat color="primary" icon="arrow_back" label="返回工作台" @click="$emit('back')" />
        <q-btn outline color="primary" icon="refresh" label="查询" :loading="loading" @click="$emit('run')" />
        <q-btn color="primary" unelevated icon="file_download" label="导出 CSV" :loading="exporting" @click="$emit('export')" />
      </div>
    </section>

    <section class="runtime-filter-panel">
      <div class="panel-title">查询参数</div>
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
      <div class="panel-title">运行结果</div>
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
import { computed } from 'vue'
import type { QTableProps } from 'quasar'
import type { ReportParameter } from 'src/api/services/report'
import type { ReportRuntimeParameterValue } from '../composables/useReportRuntime'
import type { ReportParameterControlMeta } from '../composables/useReportParameterControls'
import ReportDataTable from './ReportDataTable.vue'
import ReportParameterForm from './ReportParameterForm.vue'

type ReportStatus = 'draft' | 'published' | 'disabled' | string
type RuntimeEntryType = 'menu' | 'development'

const props = withDefaults(defineProps<{
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
}>(), {
  reportCode: '',
  status: '',
  sourceType: '',
  menuName: '',
  runtimeEntryType: 'development',
  permissionTableCode: '',
  description: '通用报表运行页面骨架',
  versionNo: 0,
  keyword: '',
  parameters: () => [],
  parameterValues: () => ({}),
  controlMetas: () => ({}),
  parameterLoading: false,
  loading: false,
  exporting: false,
})

const statusMeta = computed(() => {
  if (props.status === 'published') return { label: '已发布', color: 'positive' }
  if (props.status === 'disabled') return { label: '已停用', color: 'negative' }
  return { label: props.status ? '草稿' : '未知状态', color: 'grey-7' }
})

const runtimeEntryTypeLabel = computed(() =>
  props.runtimeEntryType === 'menu' ? '菜单运行' : '开发运行',
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
