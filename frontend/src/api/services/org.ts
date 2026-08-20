import { instance } from 'boot/axios'
import type { Query, ResponseData } from 'src/types/global'

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

export interface StructureQueryRequest extends OrganizationReadScopeRequest {
  page: number
  num: number
  structure_type?: 'management' | 'legal'
}

export interface OrganizationStructure extends OrganizationBaseRecord {
  code: string
  name: string
  structure_type: string
  status: string
  is_default: boolean
  valid_from?: string | null
  valid_to?: string | null
}

export interface OrganizationStructureResult {
  items: OrganizationStructure[]
  total: number
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

export interface OrganizationListResult<T> {
  items: T[]
  total: number
}

export interface OrganizationListQuery extends Query, OrganizationReadScopeRequest {}

export interface EmployeeQueryRequest extends OrganizationListQuery {
  employment_status?: string
  primary_legal_entity_id?: number
  user_id?: number
  legal_entity_id?: number
  org_unit_id?: number
  position_id?: number
  bound_status?: 'all' | 'bound' | 'unbound'
}

export interface BoundUserSummary {
  user_id: number
  user_name: string
}

export interface EmployeeListItem extends OrganizationBaseRecord {
  employee_no: string
  name: string
  employment_status: string
	primary_legal_entity_id?: number | null
	primary_legal_entity?: OrganizationReferenceSummary | null
  user_id?: number | null
  binding_status: string
  bound_account?: BoundUserSummary | null
  valid_from?: string | null
  valid_to?: string | null
}

export interface EmployeeDetail extends EmployeeListItem {
  mobile_masked?: string
  email_masked?: string
  local_note: string
  local_tags?: unknown
}

export interface EmployeeUserBinding {
  employee_id: number
  user_id?: number | null
  binding_status: string
  bound_account?: BoundUserSummary | null
}

export interface EmployeeUserOption {
  value: number
  label: string
  disabled: boolean
}

export interface EmployeeUserOptionsResult {
  items: EmployeeUserOption[]
  total: number
}

export interface PositionQueryRequest extends OrganizationListQuery {
  legal_entity_id?: number
  org_unit_id?: number
  position_type?: string
  is_manager_position?: boolean
  status?: string
}

export interface PositionListItem extends OrganizationBaseRecord {
  code: string
  name: string
  org_unit_id: number
  position_type: string
  job_level: string
  is_manager_position: boolean
  status: string
  valid_from?: string | null
  valid_to?: string | null
}

export interface PositionDetail extends PositionListItem {
  org_unit?: OrganizationReferenceSummary | null
  legal_entity?: OrganizationReferenceSummary | null
  local_note: string
}

export type AssignmentTimeScope = 'current' | 'history' | 'future' | 'timeline'

export interface AssignmentQueryRequest extends Query {
  employee_id: number
  legal_entity_id?: number
  org_unit_id?: number
  position_id?: number
  assignment_type?: string
  is_primary?: boolean
  is_manager?: boolean
  status?: string
  time_scope?: AssignmentTimeScope
  as_of_date?: string
}

export interface AssignmentListItem extends OrganizationBaseRecord {
  employee_id: number
  legal_entity_id: number
  org_unit_id: number
  position_id?: number | null
  assignment_type: string
  is_primary: boolean
  is_manager: boolean
  valid_from?: string | null
  valid_to?: string | null
  status: string
  time_scope: AssignmentTimeScope
  legal_entity?: OrganizationReferenceSummary | null
  org_unit?: OrganizationReferenceSummary | null
  position?: OrganizationReferenceSummary | null
}

export type AssignmentDetail = AssignmentListItem

export interface EmployeeAssignmentSummary {
  employee_id: number
  as_of_date: string
  assignment_count: number
  legal_entities: OrganizationReferenceSummary[]
  org_units: OrganizationReferenceSummary[]
  positions: OrganizationReferenceSummary[]
}

export interface SyncBatchQueryRequest extends Query {
  execution_id?: number
  sync_type?: string
  object_scope?: string
  status?: string
}

export interface SyncBatchListItem extends OrganizationBaseRecord {
  batch_no: string
  execution_id?: number | null
  sync_type: string
  object_scope: string
  started_at?: string | null
  completed_at?: string | null
  total_count: number
  success_count: number
  failed_count: number
  skipped_count: number
  status: string
  has_error: boolean
}

export type SyncBatchDetail = SyncBatchListItem

export interface SyncBatchError {
  id: number
  error_summary: string
}

export interface SyncRecordQueryRequest extends Query {
  batch_id?: number
  execution_id?: number
  object_type?: string
  local_id?: number
  action?: string
  status?: string
  dependency_type?: string
  local_handling_status?: string
}

export interface SyncRecordListItem extends OrganizationBaseRecord {
  batch_id: number
  execution_id?: number | null
  object_type: string
  source_summary: string
  local_id?: number | null
  action: string
  status: string
  error_code: string
  dependency_type: string
  retry_count: number
  last_retry_at?: string | null
  local_handling_status: string
  has_error: boolean
}

export type SyncRecordDetail = SyncRecordListItem

export interface SyncRecordError {
  id: number
  error_code: string
  dependency_type: string
  dependency_summary: string
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

export const queryStructures = async (
  request: StructureQueryRequest,
): Promise<OrganizationStructureResult> => {
  const response = await instance.post<ResponseData<OrganizationStructure[]>>(
    '/admin/org/structure/query',
    request,
    organizationReadRequestConfig,
  )
  return {
    items: response.data.data || [],
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

const listResult = <T>(response: ResponseData<T[]>): OrganizationListResult<T> => ({
  items: response.data || [],
  total: response.total || 0,
})

export const queryEmployees = async (
  request: EmployeeQueryRequest,
): Promise<OrganizationListResult<EmployeeListItem>> => {
  const response = await instance.post<ResponseData<EmployeeListItem[]>>(
    '/admin/org/employee/query',
    request,
  )
  return listResult(response.data)
}

export const getEmployeeDetail = async (employeeId: number): Promise<EmployeeDetail> => {
  const response = await instance.get<ResponseData<EmployeeDetail>>(
    `/admin/org/employee/${employeeId}`,
  )
  return response.data.data
}

export const queryEmployeeUserOptions = async (
  keyword: string,
  page = 1,
  num = 20,
): Promise<EmployeeUserOptionsResult> => {
  const response = await instance.post<ResponseData<EmployeeUserOption[]>>(
    '/admin/org/employee/user-options',
    { keyword, page, num },
  )
  return {
    items: response.data.data || [],
    total: response.data.total || 0,
  }
}

export const bindEmployeeUser = async (
  employeeId: number,
  userId: number,
): Promise<EmployeeUserBinding> => {
  const response = await instance.post<ResponseData<EmployeeUserBinding>>(
    `/admin/org/employee/${employeeId}/bind-user`,
    { user_id: userId },
  )
  return response.data.data
}

export const unbindEmployeeUser = async (employeeId: number): Promise<EmployeeUserBinding> => {
  const response = await instance.post<ResponseData<EmployeeUserBinding>>(
    `/admin/org/employee/${employeeId}/unbind-user`,
  )
  return response.data.data
}

export const queryAssignments = async (
  request: AssignmentQueryRequest,
): Promise<OrganizationListResult<AssignmentListItem>> => {
  const response = await instance.post<ResponseData<AssignmentListItem[]>>(
    '/admin/org/assignment/query',
    request,
  )
  return listResult(response.data)
}

export const getAssignmentDetail = async (assignmentId: number): Promise<AssignmentDetail> => {
  const response = await instance.get<ResponseData<AssignmentDetail>>(
    `/admin/org/assignment/${assignmentId}`,
  )
  return response.data.data
}

export const getEmployeeAssignmentSummary = async (
  employeeId: number,
  asOfDate?: string,
): Promise<EmployeeAssignmentSummary> => {
  const response = await instance.get<ResponseData<EmployeeAssignmentSummary>>(
    `/admin/org/employee/${employeeId}/assignments/summary`,
    asOfDate ? { params: { as_of_date: asOfDate } } : undefined,
  )
  return response.data.data
}

export const queryPositions = async (
  request: PositionQueryRequest,
): Promise<OrganizationListResult<PositionListItem>> => {
  const response = await instance.post<ResponseData<PositionListItem[]>>(
    '/admin/org/position/query',
    request,
    organizationReadRequestConfig,
  )
  return listResult(response.data)
}

export const getPositionDetail = async (positionId: number): Promise<PositionDetail> => {
  const response = await instance.get<ResponseData<PositionDetail>>(
    `/admin/org/position/${positionId}`,
  )
  return response.data.data
}

export const querySyncBatches = async (
  request: SyncBatchQueryRequest,
): Promise<OrganizationListResult<SyncBatchListItem>> => {
  const response = await instance.post<ResponseData<SyncBatchListItem[]>>(
    '/admin/org/sync/batch/query',
    request,
  )
  return listResult(response.data)
}

export const getSyncBatchDetail = async (batchId: number): Promise<SyncBatchDetail> => {
  const response = await instance.get<ResponseData<SyncBatchDetail>>(
    `/admin/org/sync/batch/${batchId}`,
  )
  return response.data.data
}

export const getSyncBatchError = async (batchId: number): Promise<SyncBatchError> => {
  const response = await instance.get<ResponseData<SyncBatchError>>(
    `/admin/org/sync/batch/${batchId}/error`,
  )
  return response.data.data
}

export const querySyncRecords = async (
  request: SyncRecordQueryRequest,
): Promise<OrganizationListResult<SyncRecordListItem>> => {
  const response = await instance.post<ResponseData<SyncRecordListItem[]>>(
    '/admin/org/sync/record/query',
    request,
  )
  return listResult(response.data)
}

export const getSyncRecordDetail = async (recordId: number): Promise<SyncRecordDetail> => {
  const response = await instance.get<ResponseData<SyncRecordDetail>>(
    `/admin/org/sync/record/${recordId}`,
  )
  return response.data.data
}

export const getSyncRecordError = async (recordId: number): Promise<SyncRecordError> => {
  const response = await instance.get<ResponseData<SyncRecordError>>(
    `/admin/org/sync/record/${recordId}/error`,
  )
  return response.data.data
}
