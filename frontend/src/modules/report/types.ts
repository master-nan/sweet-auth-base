import type { Basic, Query } from 'src/types/global'

export type ReportStatus = 'draft' | 'published' | 'disabled'
export type ReportKind = 'detail' | 'summary'
export type ReportDatasetType = 'table' | 'sql'
export type ReportCellBindingType = 'static' | 'field' | 'group' | 'sum' | 'count' | 'formula'
export type ReportParameterType = 'text' | 'select' | 'date' | 'date_range' | 'number'
export type ReportParameterOperator = 'eq' | 'like' | 'between' | 'gte' | 'lte'
export type ReportDatasetJoinType = 'left' | 'inner'
export type ReportRuntimeDisplayMode = 'paged' | 'all'
export type ReportRuntimeType = 'design_preview' | 'runtime_run' | 'runtime_export'
export type ReportExportFormat = 'csv' | 'xlsx'
export type ReportQuery = Partial<Query>
export type ReportDesignerMode = 'layout' | 'legacy_sheet'
export type ReportLayoutAreaType =
  | 'title'
  | 'parameter_summary'
  | 'header'
  | 'detail'
  | 'group'
  | 'summary'
  | 'footer'
export type ReportLayoutAreaItemType = 'static_text' | 'parameter' | 'field' | 'summary' | 'runtime'
export type ReportLayoutAreaItemAlign = 'left' | 'center' | 'right'
export type ReportLayoutAreaItemFormat =
  | 'text'
  | 'number'
  | 'amount'
  | 'date'
  | 'datetime'
  | 'dict'
export type ReportLayoutAreaAggregate = 'none' | 'sum' | 'count'

export interface ReportField {
  name: string
  code: string
  type: string
  role?: 'dimension' | 'metric' | 'time' | 'text'
  aggregate?: 'none' | 'count' | 'sum' | 'avg' | 'max' | 'min'
  selected?: boolean
}

export interface ReportDataset {
  id: string
  name: string
  type: ReportDatasetType
  source_code?: string
  sql?: string
  fields: ReportField[]
  primary?: boolean
}

export interface ReportDatasetJoin {
  id: string
  left_dataset_id: string
  left_field: string
  right_dataset_id: string
  right_field: string
  join_type: ReportDatasetJoinType
}

export interface ReportParameter {
  id: string
  label: string
  dataset_id?: string
  field: string
  type: ReportParameterType
  operator: ReportParameterOperator
  placeholder?: string
  default_value?: string | number | Array<string | number> | undefined
}

export interface ReportCellStyle {
  bold?: boolean
  italic?: boolean
  align?: 'left' | 'center' | 'right'
  background?: string
  color?: string
}

export interface ReportCellBinding {
  type: ReportCellBindingType
  dataset_id?: string
  field?: string
  formula?: string
}

export interface ReportSheetCell {
  id: string
  row: number
  col: number
  value: string
  binding?: ReportCellBinding | undefined
  style?: ReportCellStyle | undefined
  colspan?: number | undefined
  rowspan?: number | undefined
}

export interface ReportSheetConfig {
  rows: number
  cols: number
  scale?: number | undefined
  active_cell?: string | undefined
  detail_rows?: number[] | undefined
  summary_rows?: number[] | undefined
  cells: ReportSheetCell[]
}

export interface ReportLayoutAreaItem {
  id: string
  type: ReportLayoutAreaItemType
  label?: string
  dataset_id?: string
  field?: string
  parameter_id?: string
  value?: string
  width?: number
  align?: ReportLayoutAreaItemAlign
  format?: ReportLayoutAreaItemFormat
  aggregate?: ReportLayoutAreaAggregate
  visible?: boolean
  order?: number
}

export interface ReportLayoutArea {
  id: string
  type: ReportLayoutAreaType
  title: string
  visible: boolean
  order: number
  items: ReportLayoutAreaItem[]
}

export interface ReportLayoutConfig {
  version: number
  view: 'sheet'
  title?: string
  subtitle?: string
  kind?: ReportKind
  designer_mode?: ReportDesignerMode
  layout_areas?: ReportLayoutArea[]
  datasets: ReportDataset[]
  dataset_joins?: ReportDatasetJoin[] | undefined
  parameters: ReportParameter[]
  sheet: ReportSheetConfig
  runtime_display?: ReportRuntimeDisplayMode
  runtime_page_size?: number
}

export interface ReportQueryConfig {
  version: number
  datasets: ReportDataset[]
  dataset_joins?: ReportDatasetJoin[] | undefined
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
  published_version_id?: number
  published_version_no?: number
  menu_id?: number
  menu_name?: string
  menu_title?: string
  menu_path?: string
  menu_component?: string
  menu_page_type?: string
  menu_visible?: boolean
  published_to_menu?: boolean
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
  datasets: ReportDataset[]
  dataset_joins?: ReportDatasetJoin[] | undefined
  parameters: ReportParameter[]
  sheet: ReportSheetConfig
  runtime_display?: ReportRuntimeDisplayMode
  runtime_page_size?: number
  status?: ReportStatus
  query_config?: ReportQueryConfig
  layout_config?: ReportLayoutConfig
}

export interface ReportPublishReq {
  change_log?: string
}

export interface ReportPublishRes {
  report_id: number
  version_id: number
  version_no: number
  status: ReportStatus
  published_at?: string
}

export interface ReportPublishMenuReq {
  parent_menu_id: number
  title: string
  path?: string
  icon?: string
  sort?: number
  visible?: boolean
  permission_role_ids?: number[]
}

export interface ReportPublishMenuRes {
  report_id: number
  report_code: string
  menu_id: number
  menu_name: string
  menu_title: string
  path: string
  component: string
  page_type: string
  visible: boolean
  published_to_menu: boolean
}

export interface ReportVersion {
  id: number
  report_id: number
  version_no: number
  status: 'published' | 'archived' | 'draft' | string
  state?: boolean
  published_at?: string
  published_by?: number | string
  published_name?: string
  published_by_name?: string
  change_log?: string
  created_at?: string
  updated_at?: string
  is_current?: boolean
}

export interface ReportPreviewReq {
  report_id?: number | undefined
  dataset_id?: string | undefined
  menu_id?: number | undefined
  data_source_id?: number | string | undefined
  fields?: ReportField[]
  filters?: Record<string, unknown>
  parameters?: Record<string, unknown>
  page?: number
  num?: number
  keyword?: string
  expressions?: Query['expressions']
}

export interface ReportExportReq {
  format?: ReportExportFormat
  menu_id?: number | undefined
  dataset_id?: string | undefined
  parameters?: Record<string, unknown>
  params?: Record<string, unknown>
  query?: ReportQuery
  max_rows?: number
}

export interface ReportExportFile {
  blob: Blob
  filename: string
  contentType: string
}

export interface ReportPreviewMeta {
  report_id?: number
  report_code?: string
  source_code?: string
  dataset_id?: string
  dataset_type?: ReportDatasetType | string
  applied_menu_id?: number
  runtime_type?: ReportRuntimeType
  version_id?: number
  version_no?: number
}

export interface ReportPreviewRes {
  columns: ReportField[]
  rows: Record<string, unknown>[]
  total?: number
  datasets?: ReportDataset[]
  joins?: ReportDatasetJoin[]
  meta?: ReportPreviewMeta
}
