import { instance } from 'boot/axios'
import type { Query, ResponseData } from 'src/types/global'
import { localLoadingRequestConfig } from 'src/api/request-config'

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
export type InterfaceInputLocation = 'path' | 'query' | 'header' | 'body'
export type InterfaceInputDataType = 'string' | 'integer' | 'number' | 'boolean' | 'object' | 'array'

export interface InterfaceInputParameter {
  code: string
  name?: string
  location: InterfaceInputLocation
  data_type: InterfaceInputDataType
  required: boolean
  allow_multiple: boolean
  sensitive: boolean
  max_length?: number
}

export interface InterfaceInputContract {
  version: number
  parameters: InterfaceInputParameter[]
}

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
  input_contract: InterfaceInputContract
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
  retry_policy?: RetryPolicySummary
  description: string
  gmt_create: string
}

export type RetryPolicyStatus = 'draft' | 'enabled' | 'disabled'
export type RetryBackoffType = 'fixed' | 'exponential'
export type RetryJitterType = 'none' | 'full'
export type RetryErrorCategory = 'network' | 'timeout' | 'remote'
export type RetryHTTPStatus = 429 | 502 | 503 | 504

export interface RetryPolicySummary {
  id: number
  policy_code: string
  policy_name: string
  version: number
  status: RetryPolicyStatus
}

export interface RetryPolicyListItem extends RetryPolicySummary {
  max_attempts: number
  backoff_type: RetryBackoffType
  initial_delay_ms: number
  max_delay_ms: number
  retry_window_ms: number
  revision: number
  gmt_modify: string
}

export interface RetryPolicyDetail extends RetryPolicyListItem {
  description: string
  backoff_multiplier: number
  jitter_type: RetryJitterType
  jitter_ratio: number
  retryable_error_categories: RetryErrorCategory[]
  retryable_http_statuses: RetryHTTPStatus[]
  respect_retry_after: boolean
  gmt_create: string
}

export interface RetryPolicyCreateRequest {
  policy_code: string
  policy_name: string
  description?: string
  max_attempts: number
  initial_delay_ms: number
  max_delay_ms: number
  backoff_type: RetryBackoffType
  backoff_multiplier: number
  jitter_type: RetryJitterType
  jitter_ratio: number
  retry_window_ms: number
  retryable_error_categories: RetryErrorCategory[]
  retryable_http_statuses: RetryHTTPStatus[]
  respect_retry_after: boolean
}

export interface RetryPolicyUpdateRequest extends Omit<RetryPolicyCreateRequest, 'policy_code'> {
  revision: number
}

export interface RetryPolicyQuery extends Query {
  status?: RetryPolicyStatus | ''
  backoff_type?: RetryBackoffType | ''
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
  input_contract: InterfaceInputContract
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
  input_contract?: InterfaceInputContract
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
  max_attempts: number
  next_run_at?: string
  retry_reason_code?: string
  revision: number
  gmt_create: string
  started_at?: string
  completed_at?: string
  duration_ms: number
  error_category?: string
  sync_source?: {
    batch_id: number
    slice_no: number
    window_start?: string
    window_end?: string
  }
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
  retryable: boolean
  retry_reason_code?: string
  retry_delay_ms: number
  retry_scheduled_at?: string
  retry_after_source: string
}

export interface IntegrationExecutionDetail extends IntegrationExecutionListItem {
  idempotency_scope: string
  idempotency_key: string
  input_hash: string
  input_summary: {
    snapshot_version: number
    size_bytes: number
    path_count: number
    query_count: number
    header_count: number
    has_body: boolean
  }
  retry_policy?: {
    policy_code: string
    policy_version: number
    max_attempts: number
  }
  attempts_remaining: number
  sync_business?: {
    status: 'pending' | 'succeeded' | 'failed'
    reason_code?: string
    success_count: number
    failed_count: number
    reference?: string
  }
  result_http_status?: number
  result_size_bytes: number
  result_hash?: string
  result_summary?: string
  error_category?: string
  lease_owner_summary?: string
  lease_expires_at?: string
  cancelled_at?: string
  last_attempt_at?: string
}

export interface IntegrationExecutionQuery extends Query {
  external_system_id?: number
  interface_definition_id?: number
  sync_batch_id?: number
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
  started_at?: string | null
  last_poll_at?: string | null
  last_success_at?: string | null
  last_error_category: string
  active_execution_count: number
  claimed_total: number
  completed_total: number
  failed_total: number
  recovered_total: number
}

export type SyncTaskStatus = 'draft' | 'enabled' | 'disabled'
export type SyncScheduleType = 'none' | 'cron'
export type SyncCheckpointMode = 'none' | 'timestamp'
export type SyncTimeFormat = 'rfc3339' | 'unix_seconds' | 'unix_milliseconds' | 'local_datetime_seconds'
export type SyncWindowMode = 'bounded_window' | 'lower_bound_only'

export interface SyncWindowBinding {
  location: InterfaceInputLocation
  code: string
  format: SyncTimeFormat
}

export interface SyncStaticInput {
  path_params: Record<string, string>
  query_params: Record<string, string[]>
  headers: Record<string, string[]>
  json_body?: Record<string, unknown>
}

export interface SyncExecutionInputPlan {
  version: 1 | 2
  window_mode?: SyncWindowMode
  static_input: SyncStaticInput
  window_start_binding?: SyncWindowBinding
  window_end_binding?: SyncWindowBinding
}

export interface SyncConsumerMetadata {
  code: string
  version: number
  name: string
  content_types: string[]
  max_response_bytes: number
  max_duration_ms: number
  checkpoint_modes: SyncCheckpointMode[]
}

export interface SyncTaskListItem {
  id: number
  task_code: string
  task_name: string
  version: number
  status: SyncTaskStatus
  external_system: { id: number; code: string; name: string }
  interface_definition: { id: number; code: string; name: string; version: number }
  consumer: { code: string; name: string; version: number }
  schedule_type: SyncScheduleType
  cron_summary?: string
  timezone: string
  checkpoint_mode: SyncCheckpointMode
  checkpoint_at?: string
  lookback_seconds: number
  window_slice_seconds: number
  input_plan_summary: { version: number; static_parameter_count: number; has_window_bindings: boolean; window_mode: SyncWindowMode; response_bounded: boolean }
  revision: number
  gmt_modify: string
}

export interface SyncTaskDetail extends SyncTaskListItem {
  description: string
  initial_checkpoint_at?: string
  next_scheduled_at?: string
  last_scheduled_at?: string
  gmt_create: string
}

export interface SyncTaskEdit extends SyncTaskDetail {
  input_plan: SyncExecutionInputPlan
}

export interface SyncTaskCreateRequest {
  task_code: string
  task_name: string
  description?: string
  external_system_id: number
  interface_definition_id: number
  consumer_code: string
  consumer_version: number
  schedule_type: SyncScheduleType
  cron_expression: string
  timezone: string
  checkpoint_mode: SyncCheckpointMode
  initial_checkpoint_at?: string
  lookback_seconds: number
  window_slice_seconds: number
  input_plan: SyncExecutionInputPlan
}

export interface SyncTaskUpdateRequest extends Omit<SyncTaskCreateRequest, 'task_code'> {
  revision: number
  clear_initial_checkpoint?: boolean
}

export interface SyncTaskQuery extends Query {
  status?: SyncTaskStatus | ''
  schedule_type?: SyncScheduleType | ''
  checkpoint_mode?: SyncCheckpointMode | ''
  external_system_id?: number
}

export type SyncBatchStatus = 'created' | 'running' | 'succeeded' | 'failed'
export type SyncTriggerType = 'manual' | 'scheduled'

export interface SyncBatchListItem {
  id: number
  batch_no: string
  sync_task_id: number
  task_code: string
  task_name: string
  task_version: number
  trigger_type: SyncTriggerType
  status: SyncBatchStatus
  window_start?: string
  window_end?: string
  checkpoint_before?: string
  checkpoint_after?: string
  planned_slice_count: number
  current_slice_no: number
  execution_count: number
  technical_success_count: number
  technical_failed_count: number
  business_success_count: number
  business_failed_count: number
  reason_code?: string
  started_at?: string
  completed_at?: string
  gmt_create: string
}

export interface SyncBatchDetail extends SyncBatchListItem {
  system_code: string
  interface_code: string
  interface_version: number
  consumer_code: string
  consumer_version: number
  checkpoint_mode: SyncCheckpointMode
  lookback_seconds: number
  window_slice_seconds: number
  result_summary?: string
  revision: number
}

export interface SyncBatchQuery extends Query {
  status?: SyncBatchStatus | ''
  trigger_type?: SyncTriggerType | ''
  sync_task_id?: number
}

export const useIntegrationApi = () => ({
  queryExternalSystems: (query: ExternalSystemQuery) =>
    instance
      .post<
        ResponseData<ExternalSystemListItem[]>
      >('/admin/integration/external-system/query', query, localLoadingRequestConfig)
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
  queryRetryPolicies: (query: RetryPolicyQuery) =>
    instance
      .post<ResponseData<RetryPolicyListItem[]>>(
        '/admin/integration/retry-policy/query',
        query,
        localLoadingRequestConfig,
      )
      .then((response) => response.data),
  getRetryPolicy: (id: number) =>
    instance
      .get<ResponseData<RetryPolicyDetail>>(`/admin/integration/retry-policy/${id}`)
      .then((response) => response.data),
  createRetryPolicy: (request: RetryPolicyCreateRequest) =>
    instance
      .post<ResponseData<RetryPolicyDetail>>('/admin/integration/retry-policy', request)
      .then((response) => response.data),
  updateRetryPolicy: (id: number, request: RetryPolicyUpdateRequest) =>
    instance
      .put<ResponseData<RetryPolicyDetail>>(`/admin/integration/retry-policy/${id}`, request)
      .then((response) => response.data),
  createRetryPolicyVersion: (id: number, revision: number) =>
    instance
      .post<ResponseData<RetryPolicyDetail>>(`/admin/integration/retry-policy/${id}/versions`, {
        revision,
      })
      .then((response) => response.data),
  enableRetryPolicy: (id: number, revision: number) =>
    instance
      .put<ResponseData<RetryPolicyDetail>>(`/admin/integration/retry-policy/${id}/enable`, {
        revision,
      })
      .then((response) => response.data),
  disableRetryPolicy: (id: number, revision: number) =>
    instance
      .put<ResponseData<RetryPolicyDetail>>(`/admin/integration/retry-policy/${id}/disable`, {
        revision,
      })
      .then((response) => response.data),
  querySyncTasks: (query: SyncTaskQuery) =>
    instance.post<ResponseData<SyncTaskListItem[]>>('/admin/integration/sync-task/query', query).then((response) => response.data),
  getSyncTask: (id: number) =>
    instance.get<ResponseData<SyncTaskDetail>>(`/admin/integration/sync-task/${id}`).then((response) => response.data),
  getSyncTaskForEdit: (id: number) =>
    instance.get<ResponseData<SyncTaskEdit>>(`/admin/integration/sync-task/${id}/edit`).then((response) => response.data),
  listSyncConsumers: () =>
    instance.get<ResponseData<SyncConsumerMetadata[]>>('/admin/integration/sync-task/consumers').then((response) => response.data),
  createSyncTask: (request: SyncTaskCreateRequest) =>
    instance.post<ResponseData<SyncTaskDetail>>('/admin/integration/sync-task', request).then((response) => response.data),
  updateSyncTask: (id: number, request: SyncTaskUpdateRequest) =>
    instance.put<ResponseData<SyncTaskDetail>>(`/admin/integration/sync-task/${id}`, request).then((response) => response.data),
  createSyncTaskVersion: (id: number, revision: number) =>
    instance.post<ResponseData<SyncTaskDetail>>(`/admin/integration/sync-task/${id}/versions`, { revision }).then((response) => response.data),
  enableSyncTask: (id: number, revision: number) =>
    instance.put<ResponseData<SyncTaskDetail>>(`/admin/integration/sync-task/${id}/enable`, { revision }).then((response) => response.data),
  disableSyncTask: (id: number, revision: number) =>
    instance.put<ResponseData<SyncTaskDetail>>(`/admin/integration/sync-task/${id}/disable`, { revision }).then((response) => response.data),
  runSyncTask: (id: number, revision: number) =>
    instance.post<ResponseData<SyncBatchDetail>>(`/admin/integration/sync-task/${id}/run`, { revision }).then((response) => response.data),
  querySyncBatches: (query: SyncBatchQuery) =>
    instance.post<ResponseData<SyncBatchListItem[]>>('/admin/integration/sync-batch/query', query).then((response) => response.data),
  getSyncBatch: (id: number) =>
    instance.get<ResponseData<SyncBatchDetail>>(`/admin/integration/sync-batch/${id}`).then((response) => response.data),
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
