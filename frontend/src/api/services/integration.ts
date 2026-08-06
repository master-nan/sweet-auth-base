import { instance } from 'boot/axios'
import type { Query, ResponseData } from 'src/types/global'

export type ExternalSystemStatus = 'draft' | 'enabled' | 'disabled'
export type ExternalSystemType = 'hr' | 'erp' | 'tms' | 'wms' | 'other'

export interface ExternalSystemListItem {
  id: number
  system_code: string
  name: string
  system_type: ExternalSystemType
  base_url_summary: string
  owner_identifier: string
  owner_name: string
  status: ExternalSystemStatus
  revision: number
  gmt_modify: string
}

export interface ExternalSystemDetail extends ExternalSystemListItem {
  base_url: string
  description: string
  gmt_create: string
}

export interface ExternalSystemCreateRequest {
  system_code: string
  name: string
  system_type: ExternalSystemType
  base_url: string
  owner_identifier: string
  owner_name: string
  description?: string
}

export interface ExternalSystemUpdateRequest {
  name?: string
  system_type?: ExternalSystemType
  base_url?: string
  owner_identifier?: string
  owner_name?: string
  description?: string
  revision: number
}

export interface ExternalSystemQuery extends Query {
  system_type?: ExternalSystemType | ''
  status?: ExternalSystemStatus | ''
  owner?: string
}

export type InterfaceDefinitionStatus = 'draft' | 'enabled' | 'disabled'
export type InterfaceDefinitionEffectiveStatus = InterfaceDefinitionStatus | 'unavailable'
export type InterfaceProtocol = 'http' | 'https'
export type InterfaceHTTPMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface InterfaceSystemSummary {
  id: number
  system_code: string
  name: string
}

export interface InterfaceDefinitionListItem {
  id: number
  external_system: InterfaceSystemSummary
  interface_code: string
  name: string
  version: number
  protocol: InterfaceProtocol
  http_method: InterfaceHTTPMethod
  path_summary: string
  status: InterfaceDefinitionStatus
  effective_status: InterfaceDefinitionEffectiveStatus
  revision: number
  gmt_modify: string
}

export interface InterfaceDefinitionDetail extends InterfaceDefinitionListItem {
  relative_path: string
  credential_id?: number
  credential?: {
    id: number
    credential_code: string
    name: string
    credential_type: CredentialType
    effective_status: CredentialEffectiveStatus
  }
  timeout_seconds: number
  response_limit: number
  retry_policy_id?: number
  description: string
  gmt_create: string
}

export interface InterfaceDefinitionCreateRequest {
  external_system_id: number
  interface_code: string
  name: string
  protocol: InterfaceProtocol
  http_method: InterfaceHTTPMethod
  relative_path: string
  credential_id?: number
  timeout_seconds: number
  response_limit: number
  retry_policy_id?: number
  description?: string
}

export interface InterfaceDefinitionUpdateRequest {
  name?: string
  protocol?: InterfaceProtocol
  http_method?: InterfaceHTTPMethod
  relative_path?: string
  credential_id?: number
  clear_credential?: boolean
  timeout_seconds?: number
  response_limit?: number
  retry_policy_id?: number
  clear_retry_policy?: boolean
  description?: string
  revision: number
}

export interface InterfaceDefinitionQuery extends Query {
  external_system_id?: number
  http_method?: InterfaceHTTPMethod | ''
  status?: InterfaceDefinitionStatus | ''
}

export type CredentialType = 'basic' | 'api_key' | 'bearer_token' | 'oauth_client'
export type CredentialStatus = 'draft' | 'active' | 'disabled' | 'revoked'
export type CredentialEffectiveStatus = CredentialStatus | 'expired'
export type CredentialSecret = Record<string, string>

export interface CredentialListItem {
  id: number
  external_system: InterfaceSystemSummary
  credential_code: string
  name: string
  credential_type: CredentialType
  status: CredentialStatus
  effective_status: CredentialEffectiveStatus
  fingerprint_summary: string
  expires_at?: string
  version: number
  rotated_at?: string
  revision: number
  gmt_modify: string
}

export interface CredentialDetail extends CredentialListItem {
  description: string
  gmt_create: string
}

export interface CredentialCreateRequest {
  external_system_id: number
  credential_code: string
  name: string
  credential_type: CredentialType
  secret: CredentialSecret
  expires_at?: string
  description?: string
}

export interface CredentialUpdateRequest {
  name?: string
  expires_at?: string
  clear_expires_at?: boolean
  description?: string
  revision: number
}

export interface CredentialQuery extends Query {
  external_system_id?: number
  credential_type?: CredentialType | ''
  status?: CredentialEffectiveStatus | ''
}

export type IntegrationExecutionStatus =
  | 'created'
  | 'running'
  | 'retry_waiting'
  | 'succeeded'
  | 'failed'
  | 'cancelled'
export type IntegrationResultCertainty = 'confirmed' | 'unknown'

export interface IntegrationExecutionListItem {
  id: number
  execution_no: string
  external_system: InterfaceSystemSummary
  interface: InterfaceSystemSummary & { version: number }
  trigger_source: string
  status: IntegrationExecutionStatus
  current_attempt: number
  revision: number
  gmt_create: string
  started_at?: string
  completed_at?: string
  duration_ms: number
  error_category?: string
}

export interface IntegrationLogListItem {
  id: number
  execution_no: string
  attempt_no: number
  system_code: string
  interface_code: string
  worker_id_summary?: string
  status: string
  http_status?: number
  started_at: string
  ended_at?: string
  duration_ms: number
  error_category?: string
  result_certainty: IntegrationResultCertainty
}

export interface IntegrationLogDetail extends IntegrationLogListItem {
  error_code?: string
  result_summary?: string
  result_size_bytes: number
  result_hash?: string
  response_content_type?: string
  credential_code?: string
  credential_version?: string
  credential_fingerprint_summary?: string
  request_id?: string
  trace_id?: string
}

export interface IntegrationExecutionDetail extends IntegrationExecutionListItem {
  idempotency_scope: string
  idempotency_key: string
  input_hash: string
  result_http_status?: number
  result_size_bytes: number
  result_hash?: string
  result_summary?: string
  error_category?: string
  lease_owner_summary?: string
  lease_expires_at?: string
  next_run_at?: string
  cancelled_at?: string
  attempts: Array<{
    id: number
    attempt_no: number
    status: string
    started_at: string
    ended_at?: string
    duration_ms: number
    http_status?: number
    error_category?: string
    error_code?: string
    result_summary?: string
    result_size_bytes: number
    result_hash?: string
    result_certainty: IntegrationResultCertainty
    request_id?: string
    trace_id?: string
  }>
}

export interface IntegrationExecutionQuery extends Query {
  external_system_id?: number
  interface_definition_id?: number
  trigger_source?: string
  status?: IntegrationExecutionStatus | ''
  created_from?: string
  created_to?: string
}

export interface IntegrationLogQuery extends Query {
  execution_id?: number
  execution_no?: string
  external_system_id?: number
  interface_definition_id?: number
  attempt_no?: number
  status?: string
  error_category?: string
  started_from?: string
  started_to?: string
}

export interface IntegrationWorkerStatus {
  enabled: boolean
  running: boolean
  worker_id: string
  started_at: string
  last_poll_at: string
  last_success_at: string
  last_error_category: string
  active_execution_count: number
  claimed_total: number
  completed_total: number
  failed_total: number
  recovered_total: number
}

export const useIntegrationApi = () => ({
  queryExternalSystems: (query: ExternalSystemQuery) =>
    instance
      .post<
        ResponseData<ExternalSystemListItem[]>
      >('/admin/integration/external-system/query', query)
      .then((response) => response.data),
  getExternalSystem: (id: number) =>
    instance
      .get<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}`)
      .then((response) => response.data),
  createExternalSystem: (request: ExternalSystemCreateRequest) =>
    instance
      .post<ResponseData<ExternalSystemDetail>>('/admin/integration/external-system', request)
      .then((response) => response.data),
  updateExternalSystem: (id: number, request: ExternalSystemUpdateRequest) =>
    instance
      .put<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}`, request)
      .then((response) => response.data),
  enableExternalSystem: (id: number, revision: number) =>
    instance
      .put<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}/enable`, {
        revision,
      })
      .then((response) => response.data),
  disableExternalSystem: (id: number, revision: number) =>
    instance
      .put<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}/disable`, {
        revision,
      })
      .then((response) => response.data),
  queryInterfaceDefinitions: (query: InterfaceDefinitionQuery) =>
    instance
      .post<
        ResponseData<InterfaceDefinitionListItem[]>
      >('/admin/integration/interface-definition/query', query)
      .then((response) => response.data),
  getInterfaceDefinition: (id: number) =>
    instance
      .get<ResponseData<InterfaceDefinitionDetail>>(`/admin/integration/interface-definition/${id}`)
      .then((response) => response.data),
  createInterfaceDefinition: (request: InterfaceDefinitionCreateRequest) =>
    instance
      .post<
        ResponseData<InterfaceDefinitionDetail>
      >('/admin/integration/interface-definition', request)
      .then((response) => response.data),
  updateInterfaceDefinition: (id: number, request: InterfaceDefinitionUpdateRequest) =>
    instance
      .put<
        ResponseData<InterfaceDefinitionDetail>
      >(`/admin/integration/interface-definition/${id}`, request)
      .then((response) => response.data),
  createInterfaceDefinitionVersion: (id: number, revision: number) =>
    instance
      .post<
        ResponseData<InterfaceDefinitionDetail>
      >(`/admin/integration/interface-definition/${id}/versions`, { revision })
      .then((response) => response.data),
  enableInterfaceDefinition: (id: number, revision: number) =>
    instance
      .put<
        ResponseData<InterfaceDefinitionDetail>
      >(`/admin/integration/interface-definition/${id}/enable`, { revision })
      .then((response) => response.data),
  disableInterfaceDefinition: (id: number, revision: number) =>
    instance
      .put<
        ResponseData<InterfaceDefinitionDetail>
      >(`/admin/integration/interface-definition/${id}/disable`, { revision })
      .then((response) => response.data),
  queryCredentials: (query: CredentialQuery) =>
    instance
      .post<ResponseData<CredentialListItem[]>>('/admin/integration/credential/query', query)
      .then((response) => response.data),
  getCredential: (id: number) =>
    instance
      .get<ResponseData<CredentialDetail>>(`/admin/integration/credential/${id}`)
      .then((response) => response.data),
  createCredential: (request: CredentialCreateRequest) =>
    instance
      .post<ResponseData<CredentialDetail>>('/admin/integration/credential', request)
      .then((response) => response.data),
  updateCredential: (id: number, request: CredentialUpdateRequest) =>
    instance
      .put<ResponseData<CredentialDetail>>(`/admin/integration/credential/${id}`, request)
      .then((response) => response.data),
  rotateCredential: (id: number, secret: CredentialSecret, revision: number) =>
    instance
      .post<
        ResponseData<CredentialDetail>
      >(`/admin/integration/credential/${id}/rotate`, { secret, revision })
      .then((response) => response.data),
  enableCredential: (id: number, revision: number) =>
    instance
      .put<
        ResponseData<CredentialDetail>
      >(`/admin/integration/credential/${id}/enable`, { revision })
      .then((response) => response.data),
  disableCredential: (id: number, revision: number) =>
    instance
      .put<
        ResponseData<CredentialDetail>
      >(`/admin/integration/credential/${id}/disable`, { revision })
      .then((response) => response.data),
  revokeCredential: (id: number, revision: number) =>
    instance
      .put<
        ResponseData<CredentialDetail>
      >(`/admin/integration/credential/${id}/revoke`, { revision })
      .then((response) => response.data),
  queryExecutions: (query: IntegrationExecutionQuery) =>
    instance
      .post<
        ResponseData<IntegrationExecutionListItem[]>
      >('/admin/integration/execution/query', query)
      .then((response) => response.data),
  getExecution: (id: number) =>
    instance
      .get<ResponseData<IntegrationExecutionDetail>>(`/admin/integration/execution/${id}`)
      .then((response) => response.data),
  cancelExecution: (id: number, revision: number) =>
    instance
      .put<
        ResponseData<IntegrationExecutionDetail>
      >(`/admin/integration/execution/${id}/cancel`, { revision })
      .then((response) => response.data),
  queryLogs: (query: IntegrationLogQuery) =>
    instance
      .post<ResponseData<IntegrationLogListItem[]>>('/admin/integration/log/query', query)
      .then((response) => response.data),
  getLog: (id: number) =>
    instance
      .get<ResponseData<IntegrationLogDetail>>(`/admin/integration/log/${id}`)
      .then((response) => response.data),
  getWorkerStatus: () =>
    instance
      .get<ResponseData<IntegrationWorkerStatus>>('/admin/integration/worker/status')
      .then((response) => response.data),
})
