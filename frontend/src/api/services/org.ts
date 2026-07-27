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

export interface OrganizationReadScopeRequest {
  only_effective?: boolean
  include_disabled?: boolean
  include_history?: boolean
  as_of_date?: string
}

export interface OrganizationBaseRecord {
  id: number
  gmt_create?: string
  gmt_modify?: string
  state?: boolean
}

export interface LegalEntityTreeRequest extends OrganizationReadScopeRequest {
  root_id?: number
}

export interface LegalEntityTreeNode {
  id: number
  legal_entity_id: number
  value: number
  label: string
  code: string
  name: string
  short_name: string
  entity_type: string
  parent_id?: number | null
  status: string
  disabled: boolean
  orphan?: boolean
  children?: LegalEntityTreeNode[]
}

export interface LegalEntityDetail extends OrganizationBaseRecord {
  code: string
  name: string
  short_name: string
  entity_type: string
  parent_id?: number | null
  unified_social_credit_code: string
  accounting_code: string
  status: string
  valid_from?: string | null
  valid_to?: string | null
  local_note: string
  display_order?: number | null
  local_handling_status: string
}

export interface StructureOptionsRequest extends OrganizationReadScopeRequest {
  page: number
  num: number
  keyword?: string
  legal_entity_id?: number
  selected_ids?: number[]
}

export interface StructureOrgTreeRequest extends OrganizationReadScopeRequest {
  structure_id: number
  root_node_id?: number
  root_org_unit_id?: number
  keyword?: string
}

export interface StructureOrgTreeNode {
  id: number
  structure_node_id: number
  structure_id: number
  org_unit_id: number
  parent_node_id?: number | null
  code: string
  name: string
  unit_type: string
  status: string
  node_status: string
  level: number
  sort: number
  disabled: boolean
  orphan?: boolean
  children?: StructureOrgTreeNode[]
}

export interface OrganizationReferenceSummary {
  id: number
  code: string
  name: string
}

export interface OrgUnitDetail extends OrganizationBaseRecord {
  code: string
  name: string
  unit_type: string
  primary_legal_entity_id?: number | null
  primary_legal_entity?: OrganizationReferenceSummary | null
  status: string
  valid_from?: string | null
  valid_to?: string | null
  display_order?: number | null
  local_note: string
  local_handling_status: string
}

export const organizationOptionsEndpoints: Record<OrganizationSelectorType, string> = {
  legal_entity: '/admin/org/legal-entity/options',
  org_unit: '/admin/org/unit/options',
  employee: '/admin/org/employee/options',
  position: '/admin/org/position/options',
}

const organizationReadRequestConfig = {
  headers: {
    'X-Skip-Global-Loading': 'true',
  },
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
    organizationReadRequestConfig,
  )

  return {
    items: (response.data.data || [])
      .map(normalizeOption)
      .filter((option): option is OrganizationSelectorOption => option !== null),
    total: response.data.total || 0,
  }
}

const normalizeLegalEntityTree = (nodes: LegalEntityTreeNode[]): LegalEntityTreeNode[] =>
  (nodes || [])
    .filter((node) => Number.isInteger(node.legal_entity_id) && node.legal_entity_id > 0)
    .map((node) => ({
      ...node,
      id: node.legal_entity_id,
      value: node.legal_entity_id,
      children: normalizeLegalEntityTree(node.children || []),
    }))

const normalizeStructureOrgTree = (nodes: StructureOrgTreeNode[]): StructureOrgTreeNode[] =>
  (nodes || [])
    .filter(
      (node) =>
        Number.isInteger(node.structure_node_id) &&
        node.structure_node_id > 0 &&
        Number.isInteger(node.org_unit_id) &&
        node.org_unit_id > 0,
    )
    .map((node) => ({
      ...node,
      id: node.structure_node_id,
      children: normalizeStructureOrgTree(node.children || []),
    }))

export const getLegalEntityTree = async (
  request: LegalEntityTreeRequest,
): Promise<LegalEntityTreeNode[]> => {
  const response = await instance.post<ResponseData<LegalEntityTreeNode[]>>(
    '/admin/org/legal-entity/tree',
    request,
    organizationReadRequestConfig,
  )
  return normalizeLegalEntityTree(response.data.data || [])
}

export const getLegalEntityDetail = async (
  legalEntityId: number,
  request: OrganizationReadScopeRequest = {},
): Promise<LegalEntityDetail> => {
  const response = await instance.get<ResponseData<LegalEntityDetail>>(
    `/admin/org/legal-entity/${legalEntityId}`,
    {
      ...organizationReadRequestConfig,
      params: request,
    },
  )
  return response.data.data
}

export const queryStructureOptions = async (
  request: StructureOptionsRequest,
): Promise<OrganizationOptionsResult> => {
  const response = await instance.post<ResponseData<OrganizationSelectorOption[]>>(
    '/admin/org/structure/options',
    request,
    organizationReadRequestConfig,
  )
  return {
    items: (response.data.data || [])
      .map(normalizeOption)
      .filter((option): option is OrganizationSelectorOption => option !== null),
    total: response.data.total || 0,
  }
}

export const getStructureOrgTree = async (
  request: StructureOrgTreeRequest,
): Promise<StructureOrgTreeNode[]> => {
  const response = await instance.post<ResponseData<StructureOrgTreeNode[]>>(
    '/admin/org/unit/tree',
    request,
    organizationReadRequestConfig,
  )
  return normalizeStructureOrgTree(response.data.data || [])
}

export const getOrgUnitDetail = async (
  orgUnitId: number,
  request: OrganizationReadScopeRequest = {},
): Promise<OrgUnitDetail> => {
  const response = await instance.get<ResponseData<OrgUnitDetail>>(`/admin/org/unit/${orgUnitId}`, {
    ...organizationReadRequestConfig,
    params: request,
  })
  return response.data.data
}
