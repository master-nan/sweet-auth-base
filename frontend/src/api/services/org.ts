import { instance } from 'boot/axios'
import type { ResponseData } from 'src/types/global'

export type OrganizationSelectorType = 'legal_entity' | 'org_unit' | 'employee' | 'position'

export interface OrganizationSelectorOption {
  value: number
  label: string
  code: string
  name: string
  disabled: boolean
}

export interface OrganizationOptionsRequest {
  page: number
  num: number
  keyword?: string
  selected_ids?: number[]
  only_effective?: boolean
  include_history?: boolean
}

export interface OrganizationOptionsResult {
  items: OrganizationSelectorOption[]
  total: number
}

export const organizationOptionsEndpoints: Record<OrganizationSelectorType, string> = {
  legal_entity: '/admin/org/legal-entity/options',
  org_unit: '/admin/org/unit/options',
  employee: '/admin/org/employee/options',
  position: '/admin/org/position/options',
}

const normalizeOption = (option: OrganizationSelectorOption): OrganizationSelectorOption | null => {
  if (!Number.isInteger(option.value) || option.value <= 0) return null

  return {
    value: option.value,
    label: String(option.label || ''),
    code: String(option.code || ''),
    name: String(option.name || ''),
    disabled: Boolean(option.disabled),
  }
}

export const queryOrganizationOptions = async (
  selectorType: OrganizationSelectorType,
  request: OrganizationOptionsRequest,
): Promise<OrganizationOptionsResult> => {
  const response = await instance.post<ResponseData<OrganizationSelectorOption[]>>(
    organizationOptionsEndpoints[selectorType],
    request,
    {
      headers: {
        'X-Skip-Global-Loading': 'true',
      },
    },
  )

  return {
    items: (response.data.data || [])
      .map(normalizeOption)
      .filter((option): option is OrganizationSelectorOption => option !== null),
    total: response.data.total || 0,
  }
}
