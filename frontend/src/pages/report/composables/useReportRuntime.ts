import { translate as t } from 'src/boot/i18n'
import { computed, ref } from 'vue'
import { useQuasar } from 'quasar'
import {
  defaultReportSheet,
  useReportApi,
  type Report,
  type ReportDataset,
  type ReportParameter,
  type ReportPreviewReq,
  type ReportPreviewRes,
  type ReportSheetConfig,
} from 'src/api/services/report'

export type ReportRuntimeFilterValue = string | number | Array<string | number> | null | undefined

export function useReportRuntime() {
  const $q = useQuasar()
  const reportApi = useReportApi()

  const runtimeReport = ref<Report | null>(null)
  const runtimeData = ref<ReportPreviewRes>({ columns: [], rows: [], total: 0 })
  const runtimeLoading = ref(false)
  const runtimeKeyword = ref('')
  const runtimeFilterValues = ref<Record<string, ReportRuntimeFilterValue>>({})
  const runtimeMenuIdOverride = ref(0)
  const runtimePagination = ref({
    page: 1,
    rowsPerPage: 20,
    rowsNumber: 0,
  })

  const runtimeRows = computed(() => runtimeData.value.rows)
  const runtimeDatasets = computed<ReportDataset[]>(() =>
    runtimeReport.value?.layout_config?.datasets?.length
      ? runtimeReport.value.layout_config.datasets
      : runtimeData.value.datasets || [],
  )
  const runtimeSheet = computed<ReportSheetConfig>(
    () => runtimeReport.value?.layout_config?.sheet || defaultReportSheet(),
  )
  const runtimeDisplayMode = computed(
    () => runtimeReport.value?.layout_config?.runtime_display || 'paged',
  )
  const runtimeConfiguredPageSize = computed(() =>
    Number(runtimeReport.value?.layout_config?.runtime_page_size || 20),
  )
  const runtimePrimaryDataset = computed(() => {
    const datasets = runtimeDatasets.value
    return datasets.find((dataset) => dataset.primary) || datasets[0]
  })
  const runtimeDatasetId = computed(() => runtimePrimaryDataset.value?.id || '')
  const runtimeSourceCode = computed(() =>
    String(runtimePrimaryDataset.value?.source_code || runtimeReport.value?.data_source_id || ''),
  )
  const runtimeMenuId = computed(
    () => runtimeMenuIdOverride.value || runtimeReport.value?.permission_menu_id || 0,
  )
  const runtimeVersionNo = computed(() => runtimeData.value.meta?.version_no)
  const runtimeParameters = computed<ReportParameter[]>(() => {
    const report = runtimeReport.value
    return report?.layout_config?.parameters?.length
      ? report.layout_config.parameters
      : report?.query_config?.parameters || []
  })

  function openRuntime(report: Report, defaultPageSize?: number, menuId = 0) {
    runtimeReport.value = report
    runtimeMenuIdOverride.value = menuId
    runtimeKeyword.value = ''
    runtimeFilterValues.value = buildRuntimeDefaultFilters()
    runtimeData.value = { columns: [], rows: [], total: 0 }
    runtimePagination.value.page = 1
    runtimePagination.value.rowsPerPage = Number(
      defaultPageSize || runtimeConfiguredPageSize.value || 20,
    )
    runtimePagination.value.rowsNumber = 0
  }

  function clearRuntime() {
    runtimeReport.value = null
    runtimeMenuIdOverride.value = 0
    runtimeKeyword.value = ''
    runtimeFilterValues.value = {}
    runtimeData.value = { columns: [], rows: [], total: 0 }
    runtimePagination.value.page = 1
    runtimePagination.value.rowsPerPage = 20
    runtimePagination.value.rowsNumber = 0
    runtimeLoading.value = false
  }

  async function loadRuntimePreview() {
    if (!runtimeReport.value?.id) return false
    runtimeLoading.value = true
    try {
      const res = await reportApi.runReport(runtimeReport.value.id, buildRuntimePreviewReq())
      runtimeData.value = res
      runtimePagination.value.rowsNumber = res.total ?? res.rows.length
      return true
    } catch (error) {
      runtimeData.value = { columns: [], rows: [], total: 0 }
      runtimePagination.value.rowsNumber = 0
      const message =
        error instanceof Error && error.message
          ? error.message
          : t('ui.reportRunningFailedCheckReportConfigurationDataPrivilegesOrBackendInterface')
      $q.notify({ type: 'negative', message })
      return false
    } finally {
      runtimeLoading.value = false
    }
  }

  function resetRuntimeFilters() {
    runtimeKeyword.value = ''
    runtimeFilterValues.value = buildRuntimeDefaultFilters()
    runtimePagination.value.page = 1
    void loadRuntimePreview()
  }

  function buildRuntimeDefaultFilters() {
    const values: Record<string, ReportRuntimeFilterValue> = {}
    runtimeParameters.value.forEach((param) => {
      if (
        param.default_value === null ||
        param.default_value === undefined ||
        param.default_value === ''
      )
        return
      if (Array.isArray(param.default_value)) {
        const next = param.default_value.filter(
          (item) => item !== '' && item !== null && item !== undefined,
        )
        if (next.length) values[param.id] = next
        return
      }
      values[param.id] = param.default_value
    })
    return values
  }

  function buildRuntimeParameterValues() {
    const values: Record<string, unknown> = {}
    runtimeParameters.value.forEach((param) => {
      const value = runtimeFilterValues.value[param.id]
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

  function buildRuntimePreviewReq(): ReportPreviewReq {
    return {
      report_id: runtimeReport.value?.id,
      dataset_id: runtimeDatasetId.value,
      menu_id: runtimeMenuId.value,
      data_source_id: runtimeSourceCode.value,
      page: runtimeDisplayMode.value === 'all' ? 1 : runtimePagination.value.page,
      num: runtimeDisplayMode.value === 'all' ? 10000 : runtimePagination.value.rowsPerPage,
      keyword: runtimeKeyword.value,
      parameters: buildRuntimeParameterValues(),
    }
  }

  function runtimeScalarValue(id: string) {
    const value = runtimeFilterValues.value[id]
    return Array.isArray(value)
      ? String(value[0] || '')
      : value === undefined
        ? null
        : String(value)
  }

  function runtimeRangeValue(id: string, index: number) {
    const value = runtimeFilterValues.value[id]
    return Array.isArray(value) ? String(value[index] || '') : ''
  }

  function setRuntimeRangeValue(id: string, index: number, value: string | null) {
    const current = Array.isArray(runtimeFilterValues.value[id])
      ? [...(runtimeFilterValues.value[id] as Array<string | number>)]
      : ['', '']
    current[index] = value || ''
    runtimeFilterValues.value[id] = current
  }

  return {
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
    runtimeConfiguredPageSize,
    runtimeDatasetId,
    runtimeSourceCode,
    runtimeMenuId,
    runtimeVersionNo,
    runtimeParameters,
    openRuntime,
    clearRuntime,
    loadRuntimePreview,
    resetRuntimeFilters,
    buildRuntimeDefaultFilters,
    buildRuntimeParameterValues,
    buildRuntimePreviewReq,
    runtimeScalarValue,
    runtimeRangeValue,
    setRuntimeRangeValue,
  }
}
