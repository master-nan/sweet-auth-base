import { ref } from 'vue'
import { useQuasar } from 'quasar'
import {
  useReportApi,
  type Report,
  type ReportExportReq,
} from 'src/api/services/report'
import { downloadBlob } from 'src/utils/download'

type ReportExportOptions = {
  keyword?: string
  parameters?: Record<string, unknown>
  total?: number
  rowCount?: number
  pageSize?: number
}

export function useReportExport() {
  const $q = useQuasar()
  const reportApi = useReportApi()
  const exporting = ref(false)
  const exportingReportId = ref<number | null>(null)

  function resolvePrimaryDataset(report: Report) {
    const datasets = report.layout_config?.datasets?.length
      ? report.layout_config.datasets
      : report.query_config?.datasets || []
    return datasets.find((dataset) => dataset.primary) || datasets[0]
  }

  function buildReportExportReq(
    report: Report,
    options: ReportExportOptions = {},
  ): ReportExportReq {
    const dataset = resolvePrimaryDataset(report)
    const sourceCode = String(dataset?.source_code || report.data_source_id || '')
    const querySize = Math.max(
      options.total || 0,
      options.rowCount || 0,
      options.pageSize || 0,
      5000,
    )
    return {
      format: 'csv',
      dataset_id: dataset?.id || '',
      menu_id: report.permission_menu_id || 0,
      parameters: options.parameters || {},
      query: {
        page: 1,
        num: querySize,
        table_code: sourceCode,
        expressions: [],
        quick_query: {
          keyword: options.keyword || '',
        },
        filters: {},
        include_deleted: false,
      },
    }
  }

  async function exportReportWithReq(report: Report, req: ReportExportReq) {
    const file = await reportApi.exportReport(report.id, req)
    downloadBlob(file.blob, file.filename)
    $q.notify({ type: 'positive', message: '报表导出成功' })
  }

  async function exportRuntimeCsv(report: Report | null, options: ReportExportOptions = {}) {
    if (!report?.id || exporting.value) return false
    exporting.value = true
    try {
      await exportReportWithReq(report, buildReportExportReq(report, options))
      return true
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : '报表导出失败'
      $q.notify({ type: 'negative', message })
      return false
    } finally {
      exporting.value = false
    }
  }

  async function exportReportRow(row: Report) {
    if (row.status !== 'published' || exportingReportId.value) return false
    exportingReportId.value = row.id
    try {
      await exportReportWithReq(row, buildReportExportReq(row))
      return true
    } catch (error) {
      const message = error instanceof Error && error.message ? error.message : '报表导出失败'
      $q.notify({ type: 'negative', message })
      return false
    } finally {
      exportingReportId.value = null
    }
  }

  return {
    exporting,
    exportingReportId,
    buildReportExportReq,
    exportRuntimeCsv,
    exportReportRow,
  }
}
