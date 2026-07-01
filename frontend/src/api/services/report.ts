import { instance } from 'boot/axios'
import type { Basic, Query, ResponseData } from 'src/types/global'
import {
  createReportLayout,
  defaultReportSheet,
  ensureOnePrimaryDataset,
  guessReportFieldRole,
  normalizeReportKind,
  normalizeReportSheet,
  primaryTableDataset,
  REPORT_SCHEMA_VERSION,
} from 'src/modules/report/schema'
import type {
  Report,
  ReportDataSource,
  ReportDataset,
  ReportDatasetJoin,
  ReportField,
  ReportLayoutConfig,
  ReportPreviewReq,
  ReportPreviewRes,
  ReportQueryConfig,
  ReportSaveReq,
} from 'src/modules/report/types'

export {
  createBlankReportSheet as createBlankSheet,
  defaultReportSheet,
  makeReportCellId,
} from 'src/modules/report/schema'
export type {
  Report,
  ReportCellBinding,
  ReportCellBindingType,
  ReportCellStyle,
  ReportDataSource,
  ReportDataset,
  ReportDatasetJoin,
  ReportDatasetJoinType,
  ReportDatasetType,
  ReportField,
  ReportKind,
  ReportLayoutConfig,
  ReportParameter,
  ReportParameterOperator,
  ReportParameterType,
  ReportPreviewReq,
  ReportPreviewRes,
  ReportQueryConfig,
  ReportSaveReq,
  ReportSheetCell,
  ReportSheetConfig,
  ReportStatus,
} from 'src/modules/report/types'

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
  layout_config?: Partial<ReportLayoutConfig> & {
    widgets?: Array<{ fields?: Array<string | BackendReportColumn | ReportField> }>
  }
  remark?: string
}

interface BackendReportColumn {
  name?: string
  field?: string
  label?: string
  type?: string
  code?: string
}

interface BackendReportPreview {
  columns: BackendReportColumn[]
  rows: Record<string, unknown>[]
  total?: number
  datasets?: Array<ReportDataset & { source_code?: string }>
  joins?: ReportDatasetJoin[]
  meta?: ReportPreviewRes['meta']
}

interface BackendReportDataSource {
  id: number | string
  name: string
  code: string
  type?: string
  description?: string
  fields: BackendReportColumn[]
}

const toField = (field: BackendReportColumn | ReportField): ReportField => {
  const source = field as BackendReportColumn & Partial<ReportField>
  const code = source.code || source.field || source.name || ''
  const name = source.label || source.name || code
  const result: ReportField = {
    name,
    code,
    type: source.type || 'string',
    role: source.role || guessReportFieldRole(code, source.type || 'string'),
    aggregate: source.aggregate || 'none',
  }
  if (source.selected !== undefined) result.selected = source.selected
  return result
}

const normalizeWidgetFields = (
  widgets: Array<{ fields?: Array<string | BackendReportColumn | ReportField> }> = [],
) =>
  widgets
    .flatMap((widget) => widget.fields || [])
    .map((field) => {
      if (typeof field === 'string') return field
      if ('code' in field && field.code) return field.code
      if ('field' in field && field.field) return field.field
      return field.name || ''
    })
    .filter(Boolean)

const resolveDatasets = (item: BackendReport, fields: ReportField[]): ReportDataset[] => {
  const layoutDatasets = item.layout_config?.datasets || item.query_config?.datasets || []
  if (layoutDatasets.length) {
    return ensureOnePrimaryDataset(layoutDatasets.map((dataset, index) => ({
      ...dataset,
      id: dataset.id || `dataset_${index + 1}`,
      fields: (dataset.fields || []).map(toField),
      primary: dataset.primary || index === 0,
    })))
  }
  return [
    {
      id: 'main',
      name: item.source_code || '主数据',
      type: 'table',
      source_code: item.source_code,
      fields,
      primary: true,
    },
  ]
}

const toReport = (item: BackendReport): Report => {
  const fields = (item.query_config?.fields || []).map(toField)
  const fallbackWidgetFields = normalizeWidgetFields(item.layout_config?.widgets || [])
  const queryFields = fields.length
    ? fields
    : fallbackWidgetFields.map((code) => toField({ field: code, label: code }))
  const datasets = resolveDatasets(item, queryFields)
  const layout = {
    ...createReportLayout({ name: item.name, description: item.description || '' }),
    ...(item.layout_config || {}),
    datasets,
    sheet: normalizeReportSheet(item.layout_config?.sheet),
  } as ReportLayoutConfig
  const kind = normalizeReportKind(layout.kind)

  return {
    ...item,
    report_name: item.name,
    report_code: item.code,
    report_kind: kind,
    data_source_id: item.source_code,
    data_source_name: item.source_code,
    query_config: {
      version: item.query_config?.version || REPORT_SCHEMA_VERSION,
      datasets,
      dataset_joins: item.query_config?.dataset_joins || layout.dataset_joins || [],
      fields: queryFields,
      parameters: item.query_config?.parameters || layout.parameters || [],
    },
    layout_config: {
      ...layout,
      version: layout.version || REPORT_SCHEMA_VERSION,
      kind,
      dataset_joins: item.query_config?.dataset_joins || layout.dataset_joins || [],
      parameters: item.query_config?.parameters || layout.parameters || [],
    },
    status: item.state === false ? 'archived' : item.remark === 'draft' ? 'draft' : 'published',
    owner: item.modify_user_name || item.create_user_name || '',
    updated_at: item.gmt_modify || '',
  }
}

const toBackendReport = (req: ReportSaveReq) => {
  const datasets = ensureOnePrimaryDataset(req.datasets || [])
  const primary = primaryTableDataset(datasets)
  const sourceCode = primary?.source_code || String(req.data_source_id || '')
  const layoutConfig: ReportLayoutConfig = {
    version: REPORT_SCHEMA_VERSION,
    view: 'sheet',
    title: req.report_name,
    subtitle: req.description || '',
    kind: req.report_kind,
    datasets,
    dataset_joins: req.dataset_joins || [],
    parameters: req.parameters || [],
    sheet: req.sheet || defaultReportSheet(),
  }
  return {
    id: req.id,
    code: req.report_code,
    name: req.report_name,
    category: req.category || '',
    description: req.description || '',
    source_type: primary?.type || 'table',
    source_code: sourceCode,
    permission_menu_id: req.permission_menu_id || 0,
    permission_table_code: req.permission_table_code || sourceCode,
    query_config: {
      version: REPORT_SCHEMA_VERSION,
      datasets,
      dataset_joins: req.dataset_joins || [],
      fields: req.fields || [],
      parameters: req.parameters || [],
    },
    layout_config: layoutConfig,
    remark: req.status === 'draft' ? 'draft' : '',
    state: req.status !== 'archived',
  }
}

const toPreview = (data: BackendReportPreview): ReportPreviewRes => ({
  columns: (data.columns || []).map(toField),
  rows: data.rows || [],
  total: data.total ?? (data.rows || []).length,
  datasets: (data.datasets || []).map((dataset) => ({
    ...dataset,
    id: dataset.id,
    name: dataset.name,
    type: dataset.type,
    fields: (dataset.fields || []).map(toField),
    ...(dataset.source_code ? { source_code: dataset.source_code } : {}),
  })),
  joins: data.joins || [],
  ...(data.meta ? { meta: data.meta } : {}),
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
        dataset_id: req.dataset_id || '',
        parameters: req.parameters || {},
        query: {
          page: req.page || 1,
          num: req.num || 50,
          table_code: String(req.data_source_id || ''),
          expressions: req.expressions || [],
          quick_query: {
            keyword: req.keyword || '',
          },
          filters: req.filters || {},
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
