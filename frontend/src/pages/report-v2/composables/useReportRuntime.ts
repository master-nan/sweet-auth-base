import { computed, ref } from 'vue'
import {
  useReportApi,
  type Report,
  type ReportDataset,
  type ReportParameter,
  type ReportPreviewReq,
  type ReportPreviewRes,
} from 'src/api/services/report'

export type ReportRuntimeParameterValue =
  | string
  | number
  | boolean
  | Array<string | number>
  | null
  | undefined

export interface ReportRuntimeContext {
  menuId?: number
}

export function useReportRuntime() {
  const reportApi = useReportApi()
  const loading = ref(false)
  const previewData = ref<ReportPreviewRes>({ columns: [], rows: [], total: 0 })
  const keyword = ref('')
  const parameterValues = ref<Record<string, ReportRuntimeParameterValue>>({})
  const page = ref(1)
  const pageSize = ref(20)

  const rows = computed(() => previewData.value.rows || [])
  const columns = computed(() => previewData.value.columns || [])
  const total = computed(() => previewData.value.total || rows.value.length)
  const versionNo = computed(() => previewData.value.meta?.version_no || 0)

  function resolveParameters(report: Report | null): ReportParameter[] {
    if (!report) return []
    return report.layout_config?.parameters?.length
      ? report.layout_config.parameters
      : report.query_config?.parameters || []
  }

  function resolvePrimaryDataset(report: Report | null): ReportDataset | undefined {
    const datasets = report?.layout_config?.datasets?.length
      ? report.layout_config.datasets
      : report?.query_config?.datasets || []
    return datasets.find((dataset) => dataset.primary) || datasets[0]
  }

  function initRuntime(report: Report, defaultPageSize?: number) {
    keyword.value = ''
    parameterValues.value = buildDefaultParameterValues(resolveParameters(report))
    previewData.value = { columns: [], rows: [], total: 0 }
    page.value = 1
    pageSize.value = Number(defaultPageSize || report.layout_config?.runtime_page_size || 20)
  }

  function buildDefaultParameterValues(
    parameters: ReportParameter[],
    defaults: Record<string, ReportRuntimeParameterValue> = {},
  ) {
    const values: Record<string, ReportRuntimeParameterValue> = {}
    Object.entries(defaults).forEach(([key, value]) => {
      if (value === '' || value === null || value === undefined) return
      values[key] = value
    })
    parameters.forEach((param) => {
      if (values[param.id] !== undefined) return
      const value = param.default_value
      if (value === '' || value === null || value === undefined) return
      values[param.id] = value
    })
    return values
  }

  function buildParameterPayload(parameters: ReportParameter[]) {
    const values: Record<string, unknown> = {}
    parameters.forEach((param) => {
      const value = parameterValues.value[param.id]
      if (value === '' || value === null || value === undefined) return
      if (
        Array.isArray(value) &&
        !value.some((item) => item !== '' && item !== null && item !== undefined)
      )
        return
      values[param.id] = value
    })
    return values
  }

  function buildRunReq(
    report: Report,
    parameters: ReportParameter[] = resolveParameters(report),
    context: ReportRuntimeContext = {},
  ): ReportPreviewReq {
    const dataset = resolvePrimaryDataset(report)
    return {
      report_id: report.id,
      dataset_id: dataset?.id || '',
      menu_id: context.menuId ?? report.permission_menu_id ?? 0,
      data_source_id: String(dataset?.source_code || report.data_source_id || ''),
      page: page.value,
      num: pageSize.value,
      keyword: keyword.value,
      parameters: buildParameterPayload(parameters),
    }
  }

  async function runReport(
    report: Report,
    parameters?: ReportParameter[],
    context: ReportRuntimeContext = {},
  ) {
    loading.value = true
    try {
      previewData.value = await reportApi.runReport(report.id, buildRunReq(report, parameters, context))
      return previewData.value
    } finally {
      loading.value = false
    }
  }

  function setParameterValue(id: string, value: ReportRuntimeParameterValue) {
    parameterValues.value[id] = value
  }

  function setParameterValues(values: Record<string, ReportRuntimeParameterValue>) {
    parameterValues.value = { ...values }
  }

  function resetRuntimeFilters(
    report: Report | null,
    defaults: Record<string, ReportRuntimeParameterValue> = {},
  ) {
    keyword.value = ''
    parameterValues.value = buildDefaultParameterValues(resolveParameters(report), defaults)
    page.value = 1
  }

  function resetRuntime() {
    previewData.value = { columns: [], rows: [], total: 0 }
    keyword.value = ''
    parameterValues.value = {}
    page.value = 1
    pageSize.value = 20
  }

  return {
    loading,
    previewData,
    keyword,
    parameterValues,
    rows,
    columns,
    total,
    page,
    pageSize,
    versionNo,
    resolveParameters,
    resolvePrimaryDataset,
    initRuntime,
    buildParameterPayload,
    buildRunReq,
    runReport,
    setParameterValue,
    setParameterValues,
    resetRuntimeFilters,
    resetRuntime,
  }
}
