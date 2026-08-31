import { translate as t } from 'src/i18n/runtime/instance'
import { date } from 'quasar'
import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import type { Query } from 'src/types/global'

export const emptyOrganizationExpressions = () => [
  {
    rules: [{ field: '', value: null }],
    nested: [],
  },
]

export const createOrganizationQuery = (tableCode: string): Query => ({
  page: 1,
  num: 20,
  order: { field: 'gmt_modify', is_asc: false },
  table_code: tableCode,
  expressions: emptyOrganizationExpressions(),
  quick_query: { keyword: '' },
  include_deleted: false,
})

export const createOrganizationField = (
  fieldName: string,
  fieldCode: string,
  fieldType: SysTableFieldType = SysTableFieldType.VARCHAR,
  options: {
    inputType?: SysTableFieldInputType
    dictCode?: string
    sortable?: boolean
  } = {},
): TableField => ({
  id: 0,
  table_id: 0,
  field_name: fieldName,
  field_code: fieldCode,
  field_type: fieldType,
  field_length: 0,
  field_decimal_length: 0,
  input_type: options.inputType || SysTableFieldInputType.INPUT,
  default_value: '',
  dict_code: options.dictCode || '',
  is_primary_key: fieldCode === 'id',
  is_index: false,
  is_quick_search: false,
  is_advanced_search: true,
  is_sort: options.sortable ?? true,
  is_null: true,
  is_list_show: true,
  is_insert_show: false,
  is_update_show: false,
  sequence: 0,
  original_field_id: 0,
  binding: '',
})

export const formatOrganizationDate = (value?: string | null, fallback = '-'): string =>
  value ? date.formatDate(value, 'YYYY-MM-DD') : fallback

export const formatOrganizationDateTime = (value?: string | null, fallback = '-'): string =>
  value ? date.formatDate(value, 'YYYY-MM-DD HH:mm:ss') : fallback

export const referenceLabel = (value?: { code: string; name: string } | null): string =>
  value ? `${value.code} - ${value.name}` : '-'

export const formatOrganizationValue = (value: unknown): string => {
  if (value === null || value === undefined || value === '') return '-'
  if (
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'bigint' ||
    typeof value === 'boolean'
  ) {
    return String(value)
  }
  return '-'
}

export const organizationStatusColor = (status?: string): string => {
  if (['enabled', 'active', 'success', 'synced'].includes(status || '')) return 'positive'
  if (['failed', 'disabled', 'resigned'].includes(status || '')) return 'negative'
  if (['pending', 'processing', 'dependency_waiting', 'probation'].includes(status || '')) {
    return 'warning'
  }
  return 'grey-7'
}

const organizationSyncObjectLabels: Record<string, string> = {
  get all() {
    return t('ui.allObjects')
  },
  get legal_entity() {
    return t('ui.legalEntity')
  },
  get management_company() {
    return t('ui.managementCompany')
  },
  get management_unit() {
    return t('ui.managementOrganization')
  },
  get legal_unit() {
    return t('ui.organizationOfLegalPersons')
  },
  get structure_node() {
    return t('ui.structureNodes')
  },
  get position() {
    return t('ui.positionLabel')
  },
  get employee() {
    return t('ui.employeeProfiles')
  },
  get assignment() {
    return t('ui.appointments')
  },
  get resigned_employee() {
    return t('ui.separations')
  },
}

export const organizationSyncObjectLabel = (value?: string | null): string => {
  const code = String(value || '').trim()
  if (!code) return '-'
  return organizationSyncObjectLabels[code] || code
}
