import { translate as t } from 'src/boot/i18n'
export interface TableStateInput {
  canRead: boolean
  error?: string
  hasQuery: boolean
}

export const resolveTableEmptyMessage = ({ canRead, error, hasQuery }: TableStateInput) => {
  if (!canRead) return t('ui.notEntitledToReadCurrentData')
  if (error) return error
  if (hasQuery) return t('ui.noDataMatchingTheCurrentQueryCondition')
  return t('ui.noData')
}
