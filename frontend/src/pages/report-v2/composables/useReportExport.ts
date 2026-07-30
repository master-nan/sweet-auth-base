import { ref } from 'vue'
import {
  useReportApi,
  type Report,
  type ReportExportReq,
  type ReportPreviewReq,
} from 'src/api/services/report'
import { downloadBlob } from 'src/utils/download'

export function useReportExport() {
  const reportApi = useReportApi()
  const exporting = ref(false)

  function buildExportReq(report: Report, runReq: ReportPreviewReq, total = 0): ReportExportReq {
    return {
      format: 'csv',
      dataset_id: runReq.dataset_id,
      menu_id: runReq.menu_id || 0,
      parameters: runReq.parameters || {},
      query: {
        page: 1,
        num: Math.max(total || 0, 5000),
        table_code: String(runReq.data_source_id || report.data_source_id || ''),
        expressions: [],
        quick_query: {
          keyword: runReq.keyword || '',
        },
        filters: {},
        include_deleted: false,
      },
    }
  }

  async function exportReport(report: Report, req: ReportExportReq) {
    exporting.value = true
    try {
      const file = await reportApi.exportReport(report.id, req)
      downloadBlob(file.blob, file.filename)
      return file
    } finally {
      exporting.value = false
    }
  }

  return {
    exporting,
    buildExportReq,
    exportReport,
  }
}
