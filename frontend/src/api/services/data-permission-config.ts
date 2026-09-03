import { instance } from '@/boot/axios'
import type { Basic, Query, ResponseData } from '@/types/global'

export type DataPermissionResourceType = 'low_code_table' | 'business_service' | 'report'
export type DataPermissionBindingType = 'metadata_field' | 'registered_field'
export type DataPermissionValueType = 'bigint' | 'string'
export type DataPermissionSubjectType = 'role' | 'user'
export type DataPermissionOperation =
  | 'query'
  | 'detail'
  | 'create'
  | 'update'
  | 'delete'
  | 'export'
  | 'run'

export interface DataPermissionReferenceSummary {
  id: number
  code: string
  name: string
}

export interface DataPermissionDimension extends Basic {
  dimension_code: string
  name: string
  category: string
  value_type: DataPermissionValueType
  provider_code: string
  selector_type?: string
}

export interface DataResourceTarget {
  type?: DataPermissionResourceType
  reference_id?: number
  reference_code?: string
}

export interface DataResource extends Basic {
  resource_code: string
  name: string
  resource_type: DataPermissionResourceType
  permission_enabled: boolean
  adapter_code?: string
  target?: DataResourceTarget
}

export interface DataResourceOperationItem extends Basic {
  resource_id: number
  operation: DataPermissionOperation
  permission_enabled: boolean
  description?: string
}

export interface DataOwnership extends Basic {
  resource_id: number
  ownership_code: string
  dimension_id: number
  binding_type: DataPermissionBindingType
  value_type: DataPermissionValueType
  binding_target?: {
    type?: DataPermissionBindingType
    reference_id?: number
    reference_code?: string
  }
  resource?: DataPermissionReferenceSummary
  dimension?: DataPermissionReferenceSummary
}

export interface DataPolicyRule extends Basic {
  policy_id: number
  sequence: number
  dimension_id: number
  ownership_code: string
  scope_source: string
  relation: string
  operator: string
  specified_values?: unknown[]
  structure_code?: string
  dimension?: DataPermissionReferenceSummary
}

export interface DataPolicy extends Basic {
  policy_code: string
  name: string
  policy_type: string
  rules?: DataPolicyRule[]
}

export interface DataGrant extends Basic {
  subject_type: DataPermissionSubjectType
  subject_id: number
  subject?: DataPermissionReferenceSummary
  resource_id: number
  operation: DataPermissionOperation
  policy_id: number
  valid_from?: string | null
  valid_to?: string | null
  resource?: DataPermissionReferenceSummary
  policy?: DataPermissionReferenceSummary
}

export interface ValidationError {
  code: string
  message: string
  object_type: string
  object_id: number
}

export interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
}

export interface DataResourceSaveReq {
  id?: number
  resource_code: string
  name: string
  resource_type: DataPermissionResourceType
  target: DataResourceTarget
  adapter_code: string
  description?: string
  state?: boolean
  operations?: Array<{
    operation: DataPermissionOperation
    description?: string
    state?: boolean
  }>
}

export interface DataOwnershipSaveReq {
  id?: number
  resource_id: number
  ownership_code: string
  dimension_id: number
  binding_type: DataPermissionBindingType
  binding_target: {
    reference_id?: number
    reference_code?: string
  }
  value_type: DataPermissionValueType
  state?: boolean
}

export interface DataPolicyRuleSaveReq {
  sequence: number
  dimension_id: number
  ownership_code: string
  scope_source: string
  relation: string
  operator: string
  specified_values?: unknown[]
  structure_code?: string
  description?: string
  state?: boolean
}

export interface DataPolicySaveReq {
  id?: number
  policy_code: string
  name: string
  description?: string
  state?: boolean
  rules?: DataPolicyRuleSaveReq[]
}

export interface DataGrantSaveReq {
  subject_type: DataPermissionSubjectType
  subject_id: number
  resource_id: number
  operation: DataPermissionOperation
  policy_id: number
  valid_from?: string | null
  valid_to?: string | null
  description?: string
  state?: boolean
}

export interface DataPermissionConfigQuery extends Query {
  resource_type?: string
  permission_enabled?: boolean
  state?: boolean
  resource_id?: number
  dimension_id?: number
  binding_type?: string
  policy_type?: string
  policy_id?: number
  subject_type?: string
  subject_id?: number
  operation?: string
}

const postQuery = <T>(url: string, params: DataPermissionConfigQuery) =>
  instance.post<ResponseData<T[]>>(url, params).then((response) => response.data)

export const useDataPermissionConfigApi = () => {
  const queryDimensions = (params: DataPermissionConfigQuery) =>
    postQuery<DataPermissionDimension>('/admin/data-permission/config/dimension/query', params)

  const queryResources = (params: DataPermissionConfigQuery) =>
    postQuery<DataResource>('/admin/data-permission/config/resource/query', params)

  const getResource = (id: number) =>
    instance
      .get<ResponseData<DataResource>>(`/admin/data-permission/config/resource/${id}`)
      .then((response) => response.data)

  const createResource = (request: DataResourceSaveReq) =>
    instance
      .post<ResponseData<DataResource>>('/admin/data-permission/config/resource', request)
      .then((response) => response.data)

  const updateResource = (request: DataResourceSaveReq) =>
    instance
      .put<
        ResponseData<DataResource>
      >(`/admin/data-permission/config/resource/${request.id}`, request)
      .then((response) => response.data)

  const listResourceOperations = (resourceId: number) =>
    instance
      .get<
        ResponseData<DataResourceOperationItem[]>
      >(`/admin/data-permission/config/resource/${resourceId}/operations`)
      .then((response) => response.data)

  const replaceResourceOperations = (
    resourceId: number,
    operations: DataResourceOperationItem['operation'][],
  ) =>
    instance
      .put<ResponseData<DataResourceOperationItem[]>>(
        `/admin/data-permission/config/resource/${resourceId}/operations`,
        {
          resource_id: resourceId,
          items: operations.map((operation) => ({ operation, state: true })),
        },
      )
      .then((response) => response.data)

  const setResourcePermission = (id: number, enabled: boolean) =>
    instance
      .put<
        ResponseData<ValidationResult>
      >(`/admin/data-permission/config/resource/${id}/permission`, { permission_enabled: enabled })
      .then((response) => response.data)

  const queryOwnerships = (params: DataPermissionConfigQuery) =>
    postQuery<DataOwnership>('/admin/data-permission/config/ownership/query', params)

  const listResourceOwnerships = (resourceId: number) =>
    instance
      .get<
        ResponseData<DataOwnership[]>
      >(`/admin/data-permission/config/resource/${resourceId}/ownerships`)
      .then((response) => response.data)

  const getOwnership = (id: number) =>
    instance
      .get<ResponseData<DataOwnership>>(`/admin/data-permission/config/ownership/${id}`)
      .then((response) => response.data)

  const createOwnership = (request: DataOwnershipSaveReq) =>
    instance
      .post<ResponseData<DataOwnership>>('/admin/data-permission/config/ownership', request)
      .then((response) => response.data)

  const updateOwnership = (request: DataOwnershipSaveReq) =>
    instance
      .put<
        ResponseData<DataOwnership>
      >(`/admin/data-permission/config/ownership/${request.id}`, request)
      .then((response) => response.data)

  const disableOwnership = (id: number) =>
    instance
      .put<ResponseData<DataOwnership>>(`/admin/data-permission/config/ownership/${id}`, {
        id,
        state: false,
      })
      .then((response) => response.data)

  const queryPolicies = (params: DataPermissionConfigQuery) =>
    postQuery<DataPolicy>('/admin/data-permission/config/policy/query', params)

  const getPolicy = (id: number) =>
    instance
      .get<ResponseData<DataPolicy>>(`/admin/data-permission/config/policy/${id}`)
      .then((response) => response.data)

  const queryPolicyRules = (params: DataPermissionConfigQuery) =>
    postQuery<DataPolicyRule>('/admin/data-permission/config/policy/rule/query', params)

  const createPolicy = (request: DataPolicySaveReq) =>
    instance
      .post<ResponseData<DataPolicy>>('/admin/data-permission/config/policy', request)
      .then((response) => response.data)

  const updatePolicy = (request: DataPolicySaveReq) =>
    instance
      .put<ResponseData<DataPolicy>>(`/admin/data-permission/config/policy/${request.id}`, request)
      .then((response) => response.data)

  const replacePolicyRules = (policyId: number, rules: DataPolicyRuleSaveReq[]) =>
    instance
      .put<
        ResponseData<DataPolicyRule[]>
      >(`/admin/data-permission/config/policy/${policyId}/rules`, { policy_id: policyId, items: rules })
      .then((response) => response.data)

  const setPolicyState = (id: number, enabled: boolean) =>
    instance
      .put<ResponseData<ValidationResult>>(`/admin/data-permission/config/policy/${id}/state`, {
        state: enabled,
      })
      .then((response) => response.data)

  const queryGrants = (params: DataPermissionConfigQuery) =>
    postQuery<DataGrant>('/admin/data-permission/config/grant/query', params)

  const getGrant = (id: number) =>
    instance
      .get<ResponseData<DataGrant>>(`/admin/data-permission/config/grant/${id}`)
      .then((response) => response.data)

  const createGrant = (request: DataGrantSaveReq) =>
    instance
      .post<ResponseData<DataGrant>>('/admin/data-permission/config/grant', request)
      .then((response) => response.data)

  const setGrantState = (id: number, enabled: boolean) =>
    instance
      .put<ResponseData<ValidationResult>>(`/admin/data-permission/config/grant/${id}/state`, {
        state: enabled,
      })
      .then((response) => response.data)

  const preflight = (type: 'resource' | 'policy' | 'grant', id: number) =>
    instance
      .get<ResponseData<ValidationResult>>(`/admin/data-permission/config/preflight/${type}/${id}`)
      .then((response) => response.data)

  return {
    queryDimensions,
    queryResources,
    getResource,
    createResource,
    updateResource,
    listResourceOperations,
    replaceResourceOperations,
    setResourcePermission,
    queryOwnerships,
    listResourceOwnerships,
    getOwnership,
    createOwnership,
    updateOwnership,
    disableOwnership,
    queryPolicies,
    getPolicy,
    queryPolicyRules,
    createPolicy,
    updatePolicy,
    replacePolicyRules,
    setPolicyState,
    queryGrants,
    getGrant,
    createGrant,
    setGrantState,
    preflight,
  }
}
