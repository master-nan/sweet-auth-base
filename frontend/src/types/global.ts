import type { ExpressionLogic, ExpressionType, SysTableFieldType } from '@/types/enum'

export interface Basic {
  id: number
  gmt_create?: string
  gmt_modify?: string
  create_user_name?: string
  modify_user_name?: string
  state?: boolean
}

export interface ResponseData<T> {
  total?: number
  success: boolean
  code?: number
  message?: string
  error_code?: number
  error_message?: string
  data: T
}

export interface Order {
  field?: string
  is_asc?: boolean
}

export interface QuickQuery {
  keyword: string
}

export interface QueryRule {
  field: string
  expression_type?: ExpressionType // 比较器类型，如EQ, LT等
  value: any
  type?: SysTableFieldType // 字段类型
}

export interface ExpressionGroup {
  logic?: ExpressionLogic
  rules: Array<QueryRule>
  nested?: Array<ExpressionGroup> // 嵌套的表达式组
}

export interface Query {
  page: number
  num: number
  order?: Order
  table_code?: string
  expressions: Array<ExpressionGroup>
  quick_query?: QuickQuery
  include_deleted?: boolean
  filters?: Record<string, any>
  menu_id?: number
}

// 定义表格列的接口
export interface TableColumn<T = Record<string, unknown>> {
  name: string
  label: string
  field: string | ((row: T) => unknown)
  required?: boolean
  align?: 'left' | 'right' | 'center'
  sortable?: boolean
  sort?: (a: unknown, b: unknown, rowA: T, rowB: T) => number
  format?: (val: unknown, row: T) => unknown
  style?: string | ((row: T) => string)
  classes?: string | ((row: T) => string)
  headerStyle?: string
  headerClasses?: string
}
