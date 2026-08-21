export const organizationSelectorTypes = [
  'legal_entity',
  'org_unit',
  'employee',
  'position',
] as const

export type OrganizationSelectorType = (typeof organizationSelectorTypes)[number]

export interface OrganizationSelectorFieldMetadata {
  input_type?: string | number | null
  selector_type?: string | null
  linkage_config?: string | Record<string, unknown> | null
  dict_code?: string | null
  field_type?: string | number | null
  multiple?: boolean | string | number | null
  include_history?: boolean | string | number | null
  disabled?: boolean | string | number | null
}

export interface OrganizationSelectorRuntimeConfig {
  selectorType: OrganizationSelectorType
  multiple: boolean
  includeHistory: boolean
  disabled: boolean
}
