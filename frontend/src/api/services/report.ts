import { translate as t } from 'src/i18n/runtime/instance'
import { instance } from 'boot/axios'
import type { Basic, Query, ResponseData } from 'src/types/global'
import { parseBlobJsonError, parseContentDispositionFilename } from 'src/utils/download'
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
  ReportExportFile,
  ReportExportReq,
  ReportField,
  ReportLayoutConfig,
  ReportPublishMenuReq,
  ReportPublishMenuRes,
  ReportPublishReq,
  ReportPublishRes,
  ReportPreviewReq,
  ReportPreviewRes,
  ReportQueryConfig,
  ReportSaveReq,
  ReportVersion,
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
  ReportDesignerMode,
  ReportExportFile,
  ReportExportFormat,
  ReportExportReq,
  ReportField,
  ReportKind,
  ReportLayoutArea,
  ReportLayoutAreaAggregate,
  ReportLayoutAreaItem,
  ReportLayoutAreaItemAlign,
  ReportLayoutAreaItemFormat,
  ReportLayoutAreaItemType,
  ReportLayoutAreaType,
  ReportLayoutConfig,
  ReportParameter,
  ReportParameterOperator,
  ReportParameterType,
  ReportPreviewMeta,
  ReportPreviewReq,
  ReportPreviewRes,
  ReportPublishMenuReq,
  ReportPublishMenuRes,
  ReportPublishReq,
  ReportPublishRes,
  ReportQuery,
  ReportQueryConfig,
  ReportRuntimeDisplayMode,
  ReportRuntimeType,
  ReportSaveReq,
  ReportSheetCell,
  ReportSheetConfig,
  ReportStatus,
  ReportVersion,
} from 'src/modules/report/types'

interface BackendReport extends Basic {
  code: string
  name: string
  category?: string
  status?: string
  description?: string
  source_type: string
  source_code: string
  permission_menu_id?: number
  permission_table_code?: string
  menu_id?: number
  menu_name?: string
  menu_title?: string
  menu_path?: string
  menu_component?: string
  menu_page_type?: string
  menu_visible?: boolean
  published_to_menu?: boolean
  path?: string
  published_version_id?: number
  published_version_no?: number
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
    return ensureOnePrimaryDataset(
      layoutDatasets.map((dataset, index) => ({
        ...dataset,
        id: dataset.id || `dataset_${index + 1}`,
        fields: (dataset.fields || []).map(toField),
        primary: dataset.primary || index === 0,
      })),
    )
  }
  return [
    {
      id: 'main',
      name: item.source_code || t('ui.primaryData'),
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
      runtime_display: layout.runtime_display || 'paged',
      runtime_page_size: Number(layout.runtime_page_size || 20),
      dataset_joins: item.query_config?.dataset_joins || layout.dataset_joins || [],
      parameters: item.query_config?.parameters || layout.parameters || [],
    },
    status: normalizeReportStatus(item.status || (item.remark === 'draft' ? 'draft' : 'published')),
    ...(item.published_version_id !== undefined
      ? { published_version_id: item.published_version_id }
      : {}),
    ...(item.published_version_no !== undefined
      ? { published_version_no: item.published_version_no }
      : {}),
    ...(item.menu_id !== undefined ? { menu_id: item.menu_id } : {}),
    ...(item.menu_name !== undefined ? { menu_name: item.menu_name } : {}),
    ...(item.menu_title !== undefined ? { menu_title: item.menu_title } : {}),
    ...(item.menu_path !== undefined || item.path !== undefined
      ? { menu_path: item.menu_path || item.path }
      : {}),
    ...(item.menu_component !== undefined ? { menu_component: item.menu_component } : {}),
    ...(item.menu_page_type !== undefined ? { menu_page_type: item.menu_page_type } : {}),
    ...(item.menu_visible !== undefined ? { menu_visible: item.menu_visible } : {}),
    ...(item.published_to_menu !== undefined ? { published_to_menu: item.published_to_menu } : {}),
    owner: item.modify_user_name || item.create_user_name || '',
    updated_at: item.gmt_modify || '',
  }
}

const normalizeReportStatus = (status: string): Report['status'] => {
  if (status === 'published' || status === 'disabled') return status
  return 'draft'
}

const toBackendReport = (req: ReportSaveReq) => {
  const datasets = ensureOnePrimaryDataset(req.datasets || [])
  const primary = primaryTableDataset(datasets)
  const sourceCode = primary?.source_code || String(req.data_source_id || '')
  const sourceLayout = req.layout_config
  const datasetJoins =
    req.dataset_joins || req.query_config?.dataset_joins || sourceLayout?.dataset_joins || []
  const parameters =
    req.parameters || req.query_config?.parameters || sourceLayout?.parameters || []
  const layoutConfig: ReportLayoutConfig = {
    ...(sourceLayout || {}),
    version: sourceLayout?.version || REPORT_SCHEMA_VERSION,
    view: sourceLayout?.view || 'sheet',
    title: sourceLayout?.title || req.report_name,
    subtitle: sourceLayout?.subtitle || req.description || '',
    kind: sourceLayout?.kind || req.report_kind,
    datasets,
    dataset_joins: datasetJoins,
    parameters,
    sheet: normalizeReportSheet(req.sheet || sourceLayout?.sheet || defaultReportSheet()),
    runtime_display: req.runtime_display || sourceLayout?.runtime_display || 'paged',
    runtime_page_size: Number(req.runtime_page_size || sourceLayout?.runtime_page_size || 20),
  }
  return {
    id: req.id,
    code: req.report_code,
    name: req.report_name,
    category: req.category || '',
    status: req.status || 'draft',
    description: req.description || '',
    source_type: primary?.type || 'table',
    source_code: sourceCode,
    permission_menu_id: req.permission_menu_id || 0,
    permission_table_code: req.permission_table_code || sourceCode,
    query_config: {
      ...(req.query_config || {}),
      version: req.query_config?.version || REPORT_SCHEMA_VERSION,
      datasets,
      dataset_joins: datasetJoins,
      fields: req.fields || [],
      parameters,
    },
    layout_config: layoutConfig,
    remark: req.status === 'draft' ? 'draft' : '',
    state: true,
  }
}

const toPreviewPayload = (req: ReportPreviewReq) => ({
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

const getResponseHeader = (headers: unknown, name: string): string => {
  const source = headers as
    | {
        get?: (header: string) => unknown
        [key: string]: unknown
      }
    | undefined
  const value =
    source?.get?.(name) ??
    source?.get?.(name.toLowerCase()) ??
    source?.[name] ??
    source?.[name.toLowerCase()] ??
    source?.[name.toUpperCase()]

  if (Array.isArray(value)) return value.map(String).join('; ')
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value)
  }
  if (value === undefined || value === null) return ''
  return ''
}

const isBlob = (value: unknown): value is Blob =>
  typeof Blob !== 'undefined' && value instanceof Blob

const toBlob = (value: unknown, contentType: string): Blob => {
  if (isBlob(value)) return value
  return new Blob([value as BlobPart], contentType ? { type: contentType } : undefined)
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
    return instance
      .post<ResponseData<BackendReport[]>>('/admin/report/query', params)
      .then((res) => {
        return {
          ...res.data,
          data: (res.data.data || []).map(toReport),
        } as ResponseData<Report[]>
      })
  }

  const queryReportById = async (id: number, menuId = 0) => {
    return instance
      .get<ResponseData<BackendReport>>(`/admin/report/${id}`, {
        params: menuId > 0 ? { menu_id: menuId } : undefined,
      })
      .then((res) => {
        return {
          ...res.data,
          data: toReport(res.data.data),
        } as ResponseData<Report>
      })
  }

  const createReport = async (req: ReportSaveReq) => {
    return instance
      .post<ResponseData<number>>('/admin/report', toBackendReport(req))
      .then((res) => {
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

  const updateReportStatus = async (id: number, status: Report['status']) => {
    return instance
      .post<ResponseData<number>>(`/admin/report/${id}/status`, { status })
      .then((res) => res.data)
  }

  const publishReport = async (id: number, req?: ReportPublishReq): Promise<ReportPublishRes> => {
    return instance
      .post<ResponseData<ReportPublishRes>>(`/admin/report/${id}/publish`, req || {})
      .then((res) => res.data.data)
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
      .post<
        ResponseData<BackendReportPreview>
      >(`/admin/report/${req.report_id}/preview`, toPreviewPayload(req))
      .then((res) => {
        return {
          ...res.data,
          data: toPreview(res.data.data),
        } as ResponseData<ReportPreviewRes>
      })
  }

  const designPreviewReport = async (
    id: number,
    req: ReportPreviewReq,
  ): Promise<ReportPreviewRes> => {
    return instance
      .post<
        ResponseData<BackendReportPreview>
      >(`/admin/report/${id}/design-preview`, toPreviewPayload(req))
      .then((res) => toPreview(res.data.data))
  }

  const runReport = async (id: number, req: ReportPreviewReq): Promise<ReportPreviewRes> => {
    return instance
      .post<ResponseData<BackendReportPreview>>(`/admin/report/${id}/run`, toPreviewPayload(req))
      .then((res) => toPreview(res.data.data))
  }

  const exportReport = async (id: number, req: ReportExportReq): Promise<ReportExportFile> => {
    const format = req.format || 'csv'
    const fallbackFilename = `report_${id}.csv`

    try {
      const res = await instance.post<Blob>(
        `/admin/report/${id}/export`,
        {
          ...req,
          format,
        },
        {
          responseType: 'blob',
        },
      )
      const contentType =
        getResponseHeader(res.headers, 'content-type') || res.data.type || 'text/csv;charset=utf-8'
      const blob = toBlob(res.data, contentType)

      if (contentType.toLowerCase().includes('json')) {
        const errorMessage = await parseBlobJsonError(blob)
        throw new Error(errorMessage || t('ui.exportFailed'))
      }

      const filename =
        parseContentDispositionFilename(getResponseHeader(res.headers, 'content-disposition')) ||
        fallbackFilename

      return {
        blob,
        filename,
        contentType,
      }
    } catch (error) {
      const response = (error as { response?: { data?: unknown; headers?: unknown } }).response
      if (response?.data) {
        const contentType = getResponseHeader(response.headers, 'content-type')
        const blob = toBlob(response.data, contentType)
        const errorMessage = await parseBlobJsonError(blob)
        if (errorMessage) {
          throw new Error(errorMessage)
        }
      }
      throw error
    }
  }

  const queryReportVersions = async (id: number): Promise<ReportVersion[]> => {
    return instance
      .get<ResponseData<ReportVersion[]>>(`/admin/report/${id}/versions`)
      .then((res) => res.data.data || [])
  }

  const publishReportMenu = async (
    id: number,
    req: ReportPublishMenuReq,
  ): Promise<ReportPublishMenuRes> => {
    return instance
      .post<ResponseData<ReportPublishMenuRes>>(`/admin/report/${id}/publish-menu`, req)
      .then((res) => res.data.data)
  }

  const unpublishReportMenu = async (id: number): Promise<ReportPublishMenuRes> => {
    return instance
      .delete<ResponseData<ReportPublishMenuRes>>(`/admin/report/${id}/publish-menu`)
      .then((res) => res.data.data)
  }

  const inferSqlFields = async (sql: string) => {
    return instance
      .post<ResponseData<BackendReportColumn[]>>('/admin/report/sql-fields', { sql })
      .then((res) => {
        return {
          ...res.data,
          data: (res.data.data || []).map(toField),
        } as ResponseData<ReportField[]>
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
    updateReportStatus,
    publishReport,
    deleteReport,
    queryDataSources,
    previewReport,
    designPreviewReport,
    runReport,
    exportReport,
    queryReportVersions,
    publishReportMenu,
    unpublishReportMenu,
    inferSqlFields,
    getSelectedFields,
  }
}
