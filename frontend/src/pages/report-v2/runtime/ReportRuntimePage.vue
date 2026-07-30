<template>
  <div class="report-v2-runtime-page">
    <q-card v-if="reportLoading" flat bordered class="runtime-state-card">
      <q-skeleton type="text" width="260px" />
      <q-skeleton type="rect" height="180px" />
    </q-card>

    <q-card v-else-if="loadError" flat bordered class="runtime-state-card">
      <q-icon name="error_outline" color="negative" size="40px" />
      <div class="text-subtitle1 text-weight-medium">报表加载失败</div>
      <div class="text-body2 text-grey-7">{{ loadError }}</div>
      <q-btn outline color="primary" icon="arrow_back" label="返回工作台" @click="goWorkbench" />
    </q-card>

    <report-runtime-shell
      v-else
      :title="report?.report_name || '报表运行页'"
      :report-code="runtimeReportCode"
      :status="report?.status || ''"
      :source-type="sourceType"
      :menu-name="runtimeMenuName"
      :runtime-entry-type="runtimeEntryType"
      :menu-id="runtimeMenuId"
      :permission-table-code="runtimePermissionTableCode"
      :description="report?.description || '通用报表运行页面骨架'"
      :version-no="displayVersionNo"
      :keyword="keyword"
      :parameters="runtimeParameters"
      :parameter-values="parameterValues"
      :control-metas="controlMetas"
      :parameter-loading="parameterControlsLoading"
      :rows="rows"
      :columns="tableColumns"
      :total="total"
      :page="page"
      :page-size="pageSize"
      :loading="loading"
      :exporting="exporting"
      @back="goWorkbench"
      @run="handleSearch"
      @reset="handleReset"
      @export="handleExport"
      @update:keyword="keyword = $event"
      @update:parameter-values="setParameterValues"
      @update:page="updatePage"
      @update:page-size="updatePageSize"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import type { QTableProps } from 'quasar'
import { useReportApi, type Report, type ReportField } from 'src/api/services/report'
import ReportRuntimeShell from '../components/ReportRuntimeShell.vue'
import { useReportExport } from '../composables/useReportExport'
import { useReportParameterControls } from '../composables/useReportParameterControls'
import { useReportRuntime, type ReportRuntimeContext } from '../composables/useReportRuntime'

type RuntimeEntryType = 'menu' | 'development'

const route = useRoute()
const router = useRouter()
const $q = useQuasar()
const reportApi = useReportApi()

const report = ref<Report | null>(null)
const reportLoading = ref(false)
const loadError = ref('')

const {
  loading,
  keyword,
  parameterValues,
  rows,
  columns,
  total,
  page,
  pageSize,
  versionNo,
  resolveParameters,
  initRuntime,
  buildRunReq,
  runReport,
  setParameterValues,
  resetRuntimeFilters,
  resetRuntime,
} = useReportRuntime()
const { exporting, buildExportReq, exportReport } = useReportExport()
const {
  loading: parameterControlsLoading,
  controlMetas,
  defaultParameterValues,
  loadControls,
} = useReportParameterControls()

const runtimeParameters = computed(() => resolveParameters(report.value))
const sourceType = computed(() => report.value?.source_type || report.value?.data_source_name || '-')
const displayVersionNo = computed(() => versionNo.value || report.value?.published_version_no || 0)
const reportId = computed(() => firstNumber(route.meta.reportId, route.params.id, route.query.report_id))
const routeMenuId = computed(() => firstNumber(route.meta.menuId, route.query.menu_id))
const runtimeMenuId = computed(() =>
  firstNumber(route.meta.menuId, route.query.menu_id, report.value?.permission_menu_id),
)
const runtimeReportCode = computed(() =>
  firstString(route.meta.reportCode, route.params.code, route.query.report_code, report.value?.report_code),
)
const runtimePermissionTableCode = computed(() =>
  firstString(route.meta.permissionTableCode, report.value?.permission_table_code),
)
const isMenuRuntime = computed(() => route.meta.pageType === 'report' || !!routeMenuId.value)
const runtimeEntryType = computed<RuntimeEntryType>(() =>
  isMenuRuntime.value ? 'menu' : 'development',
)
const runtimeMenuName = computed(() => {
  if (isMenuRuntime.value) return firstString(route.meta.title, report.value?.report_name) || '报表菜单'
  return '开发运行页'
})
const runtimeContext = computed<ReportRuntimeContext>(() => ({
  ...(runtimeMenuId.value ? { menuId: runtimeMenuId.value } : {}),
}))

const tableColumns = computed<QTableProps['columns']>(() => {
  if (columns.value.length) return columns.value.map(toTableColumn)
  const firstRow = rows.value[0]
  if (!firstRow) return []
  return Object.keys(firstRow).map((key) => ({
    name: key,
    label: key,
    field: key,
    align: 'left',
  }))
})

onMounted(() => {
  void loadReport()
})

watch(reportId, () => {
  void loadReport()
})

async function loadReport() {
  const id = reportId.value
  if (!id) {
    report.value = null
    resetRuntime()
    loadError.value = '缺少报表 ID'
    return
  }
  reportLoading.value = true
  loadError.value = ''
  resetRuntime()
  try {
    const loadedReport = await reportApi.queryReportById(id).then((res) => res.data)
    report.value = loadedReport
    initRuntime(loadedReport)
    await loadControls(loadedReport, resolveParameters(loadedReport))
    resetRuntimeFilters(loadedReport, defaultParameterValues.value)
    if (loadedReport.status === 'published') {
      await handleRun()
    }
  } catch (error) {
    report.value = null
    loadError.value = error instanceof Error && error.message ? error.message : '报表详情加载失败'
    $q.notify({ type: 'negative', message: loadError.value })
  } finally {
    reportLoading.value = false
  }
}

async function handleRun() {
  if (!report.value) return
  try {
    warnMissingMenuId('运行')
    await runReport(report.value, runtimeParameters.value, runtimeContext.value)
  } catch (error) {
    $q.notify({
      type: 'negative',
      message: error instanceof Error ? error.message : '报表运行失败',
    })
  }
}

function handleSearch() {
  page.value = 1
  void handleRun()
}

function handleReset() {
  resetRuntimeFilters(report.value, defaultParameterValues.value)
  void handleRun()
}

async function handleExport() {
  if (!report.value) return
  try {
    warnMissingMenuId('导出')
    const runReq = buildRunReq(report.value, runtimeParameters.value, runtimeContext.value)
    await exportReport(report.value, buildExportReq(report.value, runReq, total.value))
    $q.notify({ type: 'positive', message: '报表导出成功' })
  } catch (error) {
    $q.notify({
      type: 'negative',
      message: error instanceof Error ? error.message : '导出失败',
    })
  }
}

function updatePage(value: number) {
  page.value = value
  void handleRun()
}

function updatePageSize(value: number) {
  pageSize.value = value
  page.value = 1
  void handleRun()
}

function goWorkbench() {
  router.push({ name: 'report_v2_workbench' })
}

function firstNumber(...values: unknown[]): number | undefined {
  for (const value of values) {
    const normalized = Array.isArray(value) ? value[0] : value
    if (normalized === '' || normalized === null || normalized === undefined) continue
    const numberValue = Number(normalized)
    if (Number.isFinite(numberValue) && numberValue > 0) return numberValue
  }
  return undefined
}

function firstString(...values: unknown[]): string {
  for (const value of values) {
    const normalized = Array.isArray(value) ? value[0] : value
    if (normalized === '' || normalized === null || normalized === undefined) continue
    return String(normalized)
  }
  return ''
}

function warnMissingMenuId(action: string) {
  if (!isMenuRuntime.value || runtimeMenuId.value) return
  console.warn(`报表菜单${action}缺少 menu_id，将以无菜单上下文执行`, {
    path: route.fullPath,
    reportId: reportId.value,
    reportCode: runtimeReportCode.value,
  })
}

function toTableColumn(field: ReportField): NonNullable<QTableProps['columns']>[number] {
  return {
    name: field.code || field.name,
    label: field.name || field.code,
    field: field.code || field.name,
    align: field.role === 'metric' ? 'right' : 'left',
  }
}
</script>

<style scoped>
.report-v2-runtime-page {
  padding: 20px;
}

.runtime-state-card {
  min-height: 240px;
  border-radius: 8px;
  padding: 24px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
}
</style>
