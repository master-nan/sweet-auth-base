import type {
  ReportCellBindingType,
  ReportDatasetJoinType,
  ReportDatasetType,
  ReportKind,
  ReportParameterOperator,
  ReportParameterType,
  ReportRuntimeDisplayMode,
} from './types'

export const reportKindOptions: Array<{ label: string; value: ReportKind; disable?: boolean }> = [
  { label: '明细模板', value: 'detail' },
  { label: '汇总模板', value: 'summary' },
]

export const reportDatasetTypeOptions: Array<{ label: string; value: ReportDatasetType }> = [
  { label: '现有表', value: 'table' },
  { label: 'SQL', value: 'sql' },
]

export const reportBindingTypeOptions: Array<{ label: string; value: ReportCellBindingType }> = [
  { label: '静态文本', value: 'static' },
  { label: '明细字段', value: 'field' },
  { label: '分组字段', value: 'group' },
  { label: '求和', value: 'sum' },
  { label: '计数', value: 'count' },
  { label: '公式', value: 'formula' },
]

export const reportAlignOptions: Array<{ label: string; value: 'left' | 'center' | 'right' }> = [
  { label: '左对齐', value: 'left' },
  { label: '居中', value: 'center' },
  { label: '右对齐', value: 'right' },
]

export const reportParameterTypeOptions: Array<{ label: string; value: ReportParameterType }> = [
  { label: '输入框', value: 'text' },
  { label: '下拉选择', value: 'select' },
  { label: '日期', value: 'date' },
  { label: '日期范围', value: 'date_range' },
  { label: '数字', value: 'number' },
]

export const reportParameterOperatorOptions: Array<{ label: string; value: ReportParameterOperator }> = [
  { label: '等于', value: 'eq' },
  { label: '包含', value: 'like' },
  { label: '区间', value: 'between' },
  { label: '大于等于', value: 'gte' },
  { label: '小于等于', value: 'lte' },
]

export const reportDatasetJoinTypeOptions: Array<{ label: string; value: ReportDatasetJoinType }> = [
  { label: '左关联', value: 'left' },
  { label: '内关联', value: 'inner' },
]

export const reportRuntimeDisplayOptions: Array<{ label: string; value: ReportRuntimeDisplayMode }> = [
  { label: '分页展示', value: 'paged' },
  { label: '全部展示', value: 'all' },
]
