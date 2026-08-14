export interface TableStateInput {
  canRead: boolean
  error?: string
  hasQuery: boolean
}

export const resolveTableEmptyMessage = ({ canRead, error, hasQuery }: TableStateInput) => {
  if (!canRead) return '无权读取当前数据'
  if (error) return error
  if (hasQuery) return '没有符合当前查询条件的数据'
  return '暂无数据'
}
