import { translate as t } from 'src/boot/i18n'
import type { ExpressionGroup, Order, QuickQuery } from 'src/types/global'

export const QuerySchemeType = {
  PERSONAL: 'PERSONAL',
  PUBLIC: 'PUBLIC',
  ROLE: 'ROLE',
  PAGE_DEFAULT: 'PAGE_DEFAULT',
} as const

export type QuerySchemeType = (typeof QuerySchemeType)[keyof typeof QuerySchemeType]

export const QuerySchemeValidationStatus = {
  VALID: 'VALID',
  DEGRADED: 'DEGRADED',
  INVALID: 'INVALID',
} as const

export type QuerySchemeValidationStatus =
  (typeof QuerySchemeValidationStatus)[keyof typeof QuerySchemeValidationStatus]

export const QuerySchemeBindingKind = {
  TODAY: 'TODAY',
  START_OF_WEEK: 'START_OF_WEEK',
  END_OF_WEEK: 'END_OF_WEEK',
  START_OF_MONTH: 'START_OF_MONTH',
  END_OF_MONTH: 'END_OF_MONTH',
  CURRENT_USER: 'CURRENT_USER',
  CURRENT_EMPLOYEE: 'CURRENT_EMPLOYEE',
} as const

export type QuerySchemeBindingKind =
  (typeof QuerySchemeBindingKind)[keyof typeof QuerySchemeBindingKind]

export interface QuerySchemeBindingParams {
  day_offset?: number
  week_offset?: number
  month_offset?: number
}

export interface QuerySchemeBinding {
  pointer: string
  kind: QuerySchemeBindingKind
  params?: QuerySchemeBindingParams
}

export interface QuerySchemePayloadV1 {
  expressions: ExpressionGroup[]
  quick_query: QuickQuery
  order: Order
  bindings: QuerySchemeBinding[]
}

export interface QueryScopeQuickPreset {
  code: string
  label: string
  payload: QuerySchemePayloadV1
}

export interface QueryScopeConfig {
  menu_id: number
  scope_code: string
  scope_label: string
  table_code: string
  quick_date_field?: string
  quick_presets: QueryScopeQuickPreset[]
  virtual_sort_fields: string[]
  dynamic_binding_kinds: QuerySchemeBindingKind[]
}

export interface QuerySchemeSummary {
  id: number
  name: string
  type: QuerySchemeType
  is_default: boolean
  status: QuerySchemeValidationStatus
}

export interface QuerySchemeIssue {
  code: string
  field_code?: string
  path?: string
  message: string
}

export interface QuerySchemeListItem extends QuerySchemeSummary {
  scope_code: string
  scope_label: string
  enabled: boolean
  creator_display_name?: string
  role_ids?: number[]
  revision: number
  updated_at: string
}

export interface QuerySchemeDetail extends QuerySchemeListItem {
  query_payload: QuerySchemePayloadV1
  issues: QuerySchemeIssue[]
}

export interface QuerySchemeSource {
  id: number
  name: string
  type: QuerySchemeType
  revision: number
  is_default: boolean
}

export interface QuerySchemeResolvedQuery {
  expressions: ExpressionGroup[]
  quick_query: QuickQuery
  order: Order
}

export interface QuerySchemeResolveResult {
  scheme: QuerySchemeSource
  validation_status: QuerySchemeValidationStatus
  issues: QuerySchemeIssue[]
  resolved_query?: QuerySchemeResolvedQuery
  bindings: QuerySchemeBinding[]
  binding_kinds: QuerySchemeBindingKind[]
}

export interface QuerySchemeManagementQuery {
  page: number
  num: number
  name?: string
  scope_code?: string
  scheme_type?: QuerySchemeType
  enabled?: boolean
}

export interface PersonalSchemeWrite {
  name: string
  scope_code: string
  query_payload: QuerySchemePayloadV1
  is_default: boolean
}

export interface PersonalSchemeUpdate {
  name: string
  query_payload: QuerySchemePayloadV1
  is_default: boolean
  revision: number
}

export interface SharedSchemeWrite extends PersonalSchemeWrite {
  scheme_type: Exclude<QuerySchemeType, 'PERSONAL'>
  enabled: boolean
  role_ids: number[]
}

export interface SharedSchemeUpdate {
  name: string
  query_payload: QuerySchemePayloadV1
  is_default: boolean
  role_ids: number[]
  revision: number
}

export const QUERY_SCHEME_TYPE_LABELS: Record<QuerySchemeType, string> = {
  get PERSONAL() {
    return t('ui.myPlan')
  },
  get PUBLIC() {
    return t('ui.publicSchemes')
  },
  get ROLE() {
    return t('ui.roleProgram')
  },
  get PAGE_DEFAULT() {
    return t('ui.pageDefault')
  },
}

export const QUERY_SCHEME_BINDING_LABELS: Record<QuerySchemeBindingKind, string> = {
  get TODAY() {
    return t('ui.today')
  },
  get START_OF_WEEK() {
    return t('ui.startingThisWeek')
  },
  get END_OF_WEEK() {
    return t('ui.endOfTheWeek')
  },
  get START_OF_MONTH() {
    return t('ui.startingThisMonth')
  },
  get END_OF_MONTH() {
    return t('ui.endOfMonth')
  },
  get CURRENT_USER() {
    return t('ui.currentUser')
  },
  get CURRENT_EMPLOYEE() {
    return t('ui.currentEmployee')
  },
}
