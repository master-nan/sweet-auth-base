import { translate as t } from 'src/boot/i18n'
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
  {
    get label() {
      return t('ui.lineUp')
    },
    value: 'detail',
  },
  {
    get label() {
      return t('ui.fixedSummaryRow')
    },
    value: 'summary',
  },
]

export const reportDatasetTypeOptions: Array<{ label: string; value: ReportDatasetType }> = [
  {
    get label() {
      return t('ui.currentTable')
    },
    value: 'table',
  },
  { label: 'SQL', value: 'sql' },
]

export const reportBindingTypeOptions: Array<{ label: string; value: ReportCellBindingType }> = [
  {
    get label() {
      return t('ui.staticText')
    },
    value: 'static',
  },
  {
    get label() {
      return t('ui.detailedFields')
    },
    value: 'field',
  },
  {
    get label() {
      return t('ui.groupFields')
    },
    value: 'group',
  },
  {
    get label() {
      return t('ui.peace')
    },
    value: 'sum',
  },
  {
    get label() {
      return t('ui.count')
    },
    value: 'count',
  },
  {
    get label() {
      return t('ui.averageValue')
    },
    value: 'avg',
  },
  {
    get label() {
      return t('ui.maximumValue')
    },
    value: 'max',
  },
  {
    get label() {
      return t('ui.minimumValue')
    },
    value: 'min',
  },
  {
    get label() {
      return t('ui.formula')
    },
    value: 'formula',
  },
]

export const reportAlignOptions: Array<{ label: string; value: 'left' | 'center' | 'right' }> = [
  {
    get label() {
      return t('ui.leftAlignment')
    },
    value: 'left',
  },
  {
    get label() {
      return t('ui.centred')
    },
    value: 'center',
  },
  {
    get label() {
      return t('ui.rightAlignment')
    },
    value: 'right',
  },
]

export const reportParameterTypeOptions: Array<{ label: string; value: ReportParameterType }> = [
  {
    get label() {
      return t('ui.inputBox')
    },
    value: 'text',
  },
  {
    get label() {
      return t('ui.dropdownSelection')
    },
    value: 'select',
  },
  {
    get label() {
      return t('ui.date')
    },
    value: 'date',
  },
  {
    get label() {
      return t('ui.dateRange')
    },
    value: 'date_range',
  },
  {
    get label() {
      return t('ui.number')
    },
    value: 'number',
  },
]

export const reportParameterOperatorOptions: Array<{
  label: string
  value: ReportParameterOperator
}> = [
  {
    get label() {
      return t('ui.equals')
    },
    value: 'eq',
  },
  {
    get label() {
      return t('ui.containsOperator')
    },
    value: 'like',
  },
  {
    get label() {
      return t('ui.area')
    },
    value: 'between',
  },
  {
    get label() {
      return t('ui.greaterThanOrEqualTo')
    },
    value: 'gte',
  },
  {
    get label() {
      return t('ui.lessThanOrEqualTo')
    },
    value: 'lte',
  },
]

export const reportDatasetJoinTypeOptions: Array<{ label: string; value: ReportDatasetJoinType }> =
  [
    {
      get label() {
        return t('ui.leftAssociation')
      },
      value: 'left',
    },
    {
      get label() {
        return t('ui.inline')
      },
      value: 'inner',
    },
  ]

export const reportRuntimeDisplayOptions: Array<{
  label: string
  value: ReportRuntimeDisplayMode
}> = [
  {
    get label() {
      return t('ui.pageBreakPresentation')
    },
    value: 'paged',
  },
  {
    get label() {
      return t('ui.showAll')
    },
    value: 'all',
  },
]
