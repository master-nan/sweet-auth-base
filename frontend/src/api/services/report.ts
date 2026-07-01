import { instance } from 'boot/axios'
import type { Basic, Query, ResponseData } from 'src/types/global'

export type ReportStatus = 'draft' | 'published' | 'archived'

export interface Report extends Basic {
  code?: string
  name?: string
  source_code?: string
  source_type?: string
  query_config?: Record<string, unknown>
  layout_config?: Record<string, unknown>
  report_name: string
  report_code: string
  category?: string
  description?: string
  data_source_id?: number | string
  data_source_name?: string
  status: ReportStatus
  owner?: string
  updated_at?: string
}

export interface ReportField {
  name: string
  code: string
  type: string
  selected?: boolean
}

export interface ReportDataSource {
  id: number | string
  name: string
  code: string
  description?: string
  fields: ReportField[]
}

export interface ReportSaveReq {
  id?: number | undefined
  report_name: string
  report_code: string
  category?: string
  description?: string
  data_source_id?: number | string | undefined
  fields: ReportField[]
}

export interface ReportPreviewReq {
  report_id?: number | undefined
  menu_id?: number | undefined
  data_source_id?: number | string | undefined
  fields: ReportField[]
  filters?: Record<string, unknown>
}

export interface ReportPreviewRes {
  columns: ReportField[]
  rows: Record<string, unknown>[]
  total?: number
}

interface BackendReport extends Basic {
  code: string
  name: string
  category?: string
  description?: string
  source_type: string
  source_code: string
  permission_menu_id?: number
  permission_table_code?: string
  query_config?: {
    fields?: ReportField[]
    [key: string]: unknown
  }
  layout_config?: Record<string, unknown>
  remark?: string
}

interface BackendReportQueryConfig {
  fields?: ReportField[]
  [key: string]: unknown
}

interface BackendReportColumn {
  name?: string
  field?: string
  label?: string
  type?: string
}

interface BackendReportPreview {
  columns: BackendReportColumn[]
  rows: Record<string, unknown>[]
  total?: number
}

interface BackendReportDataSource {
  id: number | string
  name: string
  code: string
  type?: string
  description?: string
  fields: BackendReportColumn[]
}

const toReport = (item: BackendReport): Report => ({
  ...item,
  report_name: item.name,
  report_code: item.code,
  data_source_id: item.source_code,
  data_source_name: item.source_code,
  status: item.state === false ? 'archived' : 'published',
  owner: item.modify_user_name || item.create_user_name || '',
  updated_at: item.gmt_modify || '',
})

const toBackendReport = (req: ReportSaveReq) => ({
  id: req.id,
  code: req.report_code,
  name: req.report_name,
  category: req.category || '',
  description: req.description || '',
  source_type: 'table',
  source_code: String(req.data_source_id || ''),
  permission_table_code: String(req.data_source_id || ''),
  query_config: {
    fields: req.fields || [],
  },
  layout_config: {},
  remark: '',
  state: true,
})

const toField = (field: BackendReportColumn): ReportField => ({
  name: field.label || field.name || field.field || '',
  code: field.field || field.name || '',
  type: field.type || 'string',
})

const toPreview = (data: BackendReportPreview): ReportPreviewRes => ({
  columns: (data.columns || []).map(toField),
  rows: data.rows || [],
  total: data.total ?? (data.rows || []).length,
})

export const useReportApi = () => {
  const queryReports = async (params: Query) => {
    return instance.post<ResponseData<BackendReport[]>>('/admin/report/query', params).then((res) => {
      return {
        ...res.data,
        data: (res.data.data || []).map(toReport),
      } as ResponseData<Report[]>
    })
  }

  const queryReportById = async (id: number) => {
    return instance.get<ResponseData<BackendReport>>(`/admin/report/${id}`).then((res) => {
      return {
        ...res.data,
        data: toReport(res.data.data),
      } as ResponseData<Report>
    })
  }

  const createReport = async (req: ReportSaveReq) => {
    return instance.post<ResponseData<number>>('/admin/report', toBackendReport(req)).then((res) => {
      return res.data
    })
  }

  const updateReport = async (req: ReportSaveReq) => {
    return instance
      .put<ResponseData<number>>(`/admin/report/${req.id}`, toBackendReport(req))
      .then((res) => {
        return res.data
      })
  }

  const deleteReport = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/report/${id}`).then((res) => {
      return res.data
    })
  }

  const queryDataSources = async () => {
    return instance
      .get<ResponseData<BackendReportDataSource[]>>('/admin/report/data-sources')
      .then((res) => {
        return {
          ...res.data,
          data: (res.data.data || []).map((item) => ({
            id: item.code,
            name: item.name,
            code: item.code,
            description: item.description,
            fields: (item.fields || []).map(toField),
          })),
        } as ResponseData<ReportDataSource[]>
      })
  }

  const previewReport = async (req: ReportPreviewReq) => {
    if (!req.report_id) {
      throw new Error('report_id is required for backend preview')
    }
    return instance
      .post<ResponseData<BackendReportPreview>>(`/admin/report/${req.report_id}/preview`, {
        menu_id: req.menu_id || 0,
        query: {
          page: 1,
          num: 20,
          table_code: String(req.data_source_id || ''),
          expressions: [],
          quick_query: {
            keyword: '',
          },
          include_deleted: false,
        },
      })
      .then((res) => {
        return {
          ...res.data,
          data: toPreview(res.data.data),
        } as ResponseData<ReportPreviewRes>
      })
  }

  const getSelectedFields = (report: Report): ReportField[] => {
    const config = report.query_config as BackendReportQueryConfig | undefined
    return config?.fields || []
  }

  return {
    queryReports,
    queryReportById,
    createReport,
    updateReport,
    deleteReport,
    queryDataSources,
    previewReport,
    getSelectedFields,
  }
}
