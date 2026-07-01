import { instance } from 'boot/axios'
import type { Basic, Query, ResponseData } from 'src/types/global'

export type ReportStatus = 'draft' | 'published' | 'archived'
export type ReportKind = 'detail' | 'summary' | 'chart' | 'pivot'
export type ReportWidgetType = 'filter' | 'table' | 'pivot' | 'bar' | 'line' | 'metric'

export interface ReportField {
  name: string
  code: string
  type: string
  role?: 'dimension' | 'metric' | 'time' | 'text'
  aggregate?: 'none' | 'count' | 'sum' | 'avg' | 'max' | 'min'
  selected?: boolean
}

export interface ReportParameter {
  id: string
  label: string
  field: string
  type: 'text' | 'select' | 'date' | 'date_range' | 'number'
  operator: 'eq' | 'like' | 'between' | 'gte' | 'lte'
  placeholder?: string
}

export interface ReportWidget {
  id: string
  type: ReportWidgetType
  title: string
  datasetCode?: string
  fields: string[]
  groupBy?: string[]
  metrics?: Array<{
    field: string
    aggregate: ReportField['aggregate']
    label?: string
  }>
}

export interface ReportLayoutConfig {
  view: 'sheet'
  title?: string
  subtitle?: string
  kind?: ReportKind
  parameters: ReportParameter[]
  widgets: ReportWidget[]
}

export interface ReportQueryConfig {
  fields: ReportField[]
  parameters: ReportParameter[]
}

export interface Report extends Basic {
  code?: string
  name?: string
  source_code?: string
  source_type?: string
  permission_menu_id?: number
  permission_table_code?: string
  query_config?: ReportQueryConfig
  layout_config?: ReportLayoutConfig
  report_name: string
  report_code: string
  report_kind: ReportKind
  category?: string
  description?: string
  data_source_id?: number | string
  data_source_name?: string
  status: ReportStatus
  owner?: string
  updated_at?: string
}

export interface ReportDataSource {
  id: number | string
  name: string
  code: string
  type?: string
  description?: string
  fields: ReportField[]
}

export interface ReportSaveReq {
  id?: number | undefined
  report_name: string
  report_code: string
  report_kind: ReportKind
  category?: string
  description?: string
  data_source_id?: number | string | undefined
  permission_menu_id?: number | undefined
  permission_table_code?: string | undefined
  fields: ReportField[]
  parameters: ReportParameter[]
  widgets: ReportWidget[]
}

export interface ReportPreviewReq {
  report_id?: number | undefined
  menu_id?: number | undefined
  data_source_id?: number | string | undefined
  fields?: ReportField[]
  filters?: Record<string, unknown>
  page?: number
  num?: number
  keyword?: string
  expressions?: Query['expressions']
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
  query_config?: Partial<ReportQueryConfig>
  layout_config?: Partial<ReportLayoutConfig>
  remark?: string
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

type BackendReportWidget = Omit<ReportWidget, 'fields'> & {
  fields?: Array<string | BackendReportColumn | ReportField>
}

const defaultLayout = (report?: Partial<Report>): ReportLayoutConfig => ({
  view: 'sheet',
  title: report?.report_name || report?.name || '未命名报表',
  subtitle: report?.description || '',
  kind: report?.report_kind || 'detail',
  parameters: [],
  widgets: [
    {
      id: 'detail_table',
      type: 'table',
      title: '明细表',
      fields: [],
    },
  ],
})

const normalizeKind = (value: unknown): ReportKind => {
  if (value === 'summary' || value === 'chart' || value === 'pivot') return value
  return 'detail'
}

const toField = (field: BackendReportColumn | ReportField): ReportField => {
  const source = field as BackendReportColumn & Partial<ReportField>
  const code = source.code || source.field || source.name || ''
  const name = source.label || source.name || code
  const result: ReportField = {
    name,
    code,
    type: source.type || 'string',
    role: source.role || guessFieldRole(code, source.type || 'string'),
    aggregate: source.aggregate || 'none',
  }
  if (source.selected !== undefined) result.selected = source.selected
  return result
}

const guessFieldRole = (code: string, type: string): NonNullable<ReportField['role']> => {
  const lower = code.toLowerCase()
  if (lower.includes('date') || lower.includes('time') || type === 'datetime') return 'time'
  if (['number', 'decimal', 'float', 'int'].some((item) => type.includes(item))) return 'metric'
  if (lower.endsWith('_id') || lower.includes('name') || lower.includes('status')) return 'dimension'
  return 'text'
}

const toReport = (item: BackendReport): Report => {
  const layout = {
    ...defaultLayout({ name: item.name, description: item.description || '' }),
    ...(item.layout_config || {}),
  } as ReportLayoutConfig
  const queryConfig = {
    fields: (item.query_config?.fields || []).map(toField),
    parameters: item.query_config?.parameters || layout.parameters || [],
  }
  const kind = normalizeKind(layout.kind)
  const widgets = ((layout.widgets || []) as BackendReportWidget[]).map((widget) => ({
    ...widget,
    fields: normalizeWidgetFields(widget.fields || []),
  }))

  return {
    ...item,
    report_name: item.name,
    report_code: item.code,
    report_kind: kind,
    data_source_id: item.source_code,
    data_source_name: item.source_code,
    query_config: queryConfig,
    layout_config: {
      ...layout,
      kind,
      parameters: queryConfig.parameters,
      widgets: widgets.length ? widgets : defaultLayout().widgets,
    },
    status: item.state === false ? 'archived' : item.remark === 'draft' ? 'draft' : 'published',
    owner: item.modify_user_name || item.create_user_name || '',
    updated_at: item.gmt_modify || '',
  }
}

const toBackendReport = (req: ReportSaveReq) => {
  const sourceCode = String(req.data_source_id || '')
  const layoutConfig: ReportLayoutConfig = {
    view: 'sheet',
    title: req.report_name,
    subtitle: req.description || '',
    kind: req.report_kind,
    parameters: req.parameters || [],
    widgets: req.widgets || [],
  }
  return {
    id: req.id,
    code: req.report_code,
    name: req.report_name,
    category: req.category || '',
    description: req.description || '',
    source_type: 'table',
    source_code: sourceCode,
    permission_menu_id: req.permission_menu_id || 0,
    permission_table_code: req.permission_table_code || sourceCode,
    query_config: {
      fields: req.fields || [],
      parameters: req.parameters || [],
    },
    layout_config: layoutConfig,
    remark: '',
    state: true,
  }
}

const normalizeWidgetFields = (fields: Array<string | BackendReportColumn | ReportField>): string[] =>
  fields
    .map((field) => {
      if (typeof field === 'string') return field
      if ('code' in field && field.code) return field.code
      if ('field' in field && field.field) return field.field
      return field.name || ''
    })
    .filter(Boolean)

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
      .then((res) => res.data)
  }

  const deleteReport = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/report/${id}`).then((res) => res.data)
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
            type: item.type,
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
          page: req.page || 1,
          num: req.num || 50,
          table_code: String(req.data_source_id || ''),
          expressions: req.expressions || [],
          quick_query: {
            keyword: req.keyword || '',
          },
          params: req.filters || {},
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
    return report.query_config?.fields || []
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
