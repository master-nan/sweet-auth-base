package response

import (
	"backend/model"
	"encoding/json"
	"time"
)

type IntegrationExecutionSystemRes struct {
	Id         int    `json:"id"`
	SystemCode string `json:"system_code"`
	Name       string `json:"name"`
}

type IntegrationExecutionInterfaceRes struct {
	Id            int    `json:"id"`
	InterfaceCode string `json:"interface_code"`
	Name          string `json:"name"`
	Version       int    `json:"version"`
}

type IntegrationExecutionListRes struct {
	Id              int                              `json:"id"`
	ExecutionNo     string                           `json:"execution_no"`
	ExternalSystem  IntegrationExecutionSystemRes    `json:"external_system"`
	Interface       IntegrationExecutionInterfaceRes `json:"interface"`
	TriggerSource   string                           `json:"trigger_source"`
	Status          string                           `json:"status"`
	CurrentAttempt  int                              `json:"current_attempt"`
	MaxAttempts     int                              `json:"max_attempts"`
	NextRunAt       *time.Time                       `json:"next_run_at,omitempty"`
	RetryReasonCode string                           `json:"retry_reason_code,omitempty"`
	Revision        int                              `json:"revision"`
	GmtCreate       model.CustomTime                 `json:"gmt_create"`
	GmtModify       model.CustomTime                 `json:"gmt_modify"`
	StartedAt       *time.Time                       `json:"started_at,omitempty"`
	CompletedAt     *time.Time                       `json:"completed_at,omitempty"`
	DurationMs      int64                            `json:"duration_ms"`
	ErrorCategory   string                           `json:"error_category,omitempty"`
}

type IntegrationLogListRes struct {
	Id              int        `json:"id"`
	ExecutionNo     string     `json:"execution_no"`
	AttemptNo       int        `json:"attempt_no"`
	SystemCode      string     `json:"system_code"`
	InterfaceCode   string     `json:"interface_code"`
	WorkerIDSummary string     `json:"worker_id_summary,omitempty"`
	Status          string     `json:"status"`
	HTTPStatus      *int       `json:"http_status,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	DurationMs      int64      `json:"duration_ms"`
	ErrorCategory   string     `json:"error_category,omitempty"`
	ResultCertainty string     `json:"result_certainty"`
}

type IntegrationLogDetailRes struct {
	IntegrationLogListRes
	ErrorCode                    string     `json:"error_code,omitempty"`
	ResultSummary                string     `json:"result_summary,omitempty"`
	ResultSizeBytes              int64      `json:"result_size_bytes"`
	ResultHash                   string     `json:"result_hash,omitempty"`
	ResponseContentType          string     `json:"response_content_type,omitempty"`
	CredentialCode               string     `json:"credential_code,omitempty"`
	CredentialVersion            string     `json:"credential_version,omitempty"`
	CredentialFingerprintSummary string     `json:"credential_fingerprint_summary,omitempty"`
	RequestID                    string     `json:"request_id,omitempty"`
	TraceID                      string     `json:"trace_id,omitempty"`
	Retryable                    bool       `json:"retryable"`
	RetryReasonCode              string     `json:"retry_reason_code,omitempty"`
	RetryDelayMs                 int64      `json:"retry_delay_ms"`
	RetryScheduledAt             *time.Time `json:"retry_scheduled_at,omitempty"`
	RetryAfterSource             string     `json:"retry_after_source"`
}

func integrationWorkerSummary(value string) string {
	if len(value) <= 16 {
		return value
	}
	return value[:8] + "..." + value[len(value)-4:]
}

func integrationIdentifierSummary(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func integrationExecutionDuration(value model.IntegrationExecution) int64 {
	if value.StartedAt == nil || value.CompletedAt == nil {
		return 0
	}
	return value.CompletedAt.Sub(*value.StartedAt).Milliseconds()
}

func NewIntegrationLogListRes(value model.IntegrationLog) IntegrationLogListRes {
	return IntegrationLogListRes{
		Id: value.Id, ExecutionNo: value.Execution.ExecutionNo, AttemptNo: value.AttemptNo,
		SystemCode: value.Execution.ExternalSystemCode, InterfaceCode: value.Execution.InterfaceCode,
		WorkerIDSummary: integrationWorkerSummary(value.WorkerID), Status: value.Status, HTTPStatus: value.HTTPStatus,
		StartedAt: value.StartedAt, EndedAt: value.EndedAt, DurationMs: value.DurationMs,
		ErrorCategory: value.ErrorCategory, ResultCertainty: value.ResultCertainty,
	}
}

func NewIntegrationLogDetailRes(value model.IntegrationLog) IntegrationLogDetailRes {
	return IntegrationLogDetailRes{
		IntegrationLogListRes:        NewIntegrationLogListRes(value),
		ErrorCode:                    value.ErrorCode,
		ResultSummary:                value.ResultSummary,
		ResultSizeBytes:              value.ResultSizeBytes,
		ResultHash:                   value.ResultHash,
		ResponseContentType:          value.ResponseContentType,
		CredentialCode:               value.CredentialCode,
		CredentialVersion:            value.CredentialVersion,
		CredentialFingerprintSummary: value.CredentialFingerprintSummary,
		RequestID:                    value.RequestID,
		TraceID:                      value.TraceID,
		Retryable:                    value.Retryable,
		RetryReasonCode:              value.RetryReasonCode,
		RetryDelayMs:                 value.RetryDelayMs,
		RetryScheduledAt:             value.RetryScheduledAt,
		RetryAfterSource:             value.RetryAfterSource,
	}
}

type IntegrationExecutionDetailRes struct {
	IntegrationExecutionListRes
	IdempotencyScope  string                                     `json:"idempotency_scope"`
	IdempotencyKey    string                                     `json:"idempotency_key"`
	InputHash         string                                     `json:"input_hash"`
	InputSummary      IntegrationExecutionInputSummaryRes        `json:"input_summary"`
	RetryPolicy       *IntegrationExecutionRetryPolicySummaryRes `json:"retry_policy,omitempty"`
	AttemptsRemaining int                                        `json:"attempts_remaining"`
	ResultHTTPStatus  *int                                       `json:"result_http_status,omitempty"`
	ResultSizeBytes   int64                                      `json:"result_size_bytes"`
	ResultHash        string                                     `json:"result_hash,omitempty"`
	ResultSummary     string                                     `json:"result_summary,omitempty"`
	SyncBusiness      *IntegrationExecutionSyncBusinessRes       `json:"sync_business,omitempty"`
	LeaseOwnerSummary string                                     `json:"lease_owner_summary,omitempty"`
	LeaseExpiresAt    *time.Time                                 `json:"lease_expires_at,omitempty"`
	CancelledAt       *time.Time                                 `json:"cancelled_at,omitempty"`
	LastAttemptAt     *time.Time                                 `json:"last_attempt_at,omitempty"`
}

type IntegrationExecutionSyncBusinessRes struct {
	Status       string `json:"status"`
	ReasonCode   string `json:"reason_code,omitempty"`
	SuccessCount int    `json:"success_count"`
	FailedCount  int    `json:"failed_count"`
	Reference    string `json:"reference,omitempty"`
}

type IntegrationExecutionRetryPolicySummaryRes struct {
	PolicyCode    string `json:"policy_code"`
	PolicyVersion int    `json:"policy_version"`
	MaxAttempts   int    `json:"max_attempts"`
}

type IntegrationExecutionInputSummaryRes struct {
	SnapshotVersion int  `json:"snapshot_version"`
	SizeBytes       int  `json:"size_bytes"`
	PathCount       int  `json:"path_count"`
	QueryCount      int  `json:"query_count"`
	HeaderCount     int  `json:"header_count"`
	HasBody         bool `json:"has_body"`
}

func NewIntegrationExecutionListRes(value model.IntegrationExecution) IntegrationExecutionListRes {
	policy := integrationExecutionRetryPolicySummary(value)
	maxAttempts := 1
	if policy != nil {
		maxAttempts = policy.MaxAttempts
	}
	return IntegrationExecutionListRes{
		Id: value.Id, ExecutionNo: value.ExecutionNo,
		ExternalSystem: IntegrationExecutionSystemRes{
			Id: value.ExternalSystemID, SystemCode: value.ExternalSystemCode, Name: value.ExternalSystemName,
		},
		Interface: IntegrationExecutionInterfaceRes{
			Id: value.InterfaceDefinitionID, InterfaceCode: value.InterfaceCode,
			Name: value.InterfaceName, Version: value.InterfaceVersion,
		},
		TriggerSource: value.TriggerSource, Status: value.Status, CurrentAttempt: value.CurrentAttempt,
		MaxAttempts: maxAttempts, NextRunAt: value.NextRunAt, RetryReasonCode: value.RetryReasonCode,
		Revision: value.Revision, GmtCreate: value.GmtCreate, GmtModify: value.GmtModify,
		StartedAt: value.StartedAt, CompletedAt: value.CompletedAt, DurationMs: integrationExecutionDuration(value),
		ErrorCategory: value.ErrorCategory,
	}
}

func NewIntegrationExecutionDetailRes(value model.IntegrationExecution) IntegrationExecutionDetailRes {
	inputSummary := integrationExecutionInputSummary(value)
	policy := integrationExecutionRetryPolicySummary(value)
	result := IntegrationExecutionDetailRes{
		IntegrationExecutionListRes: NewIntegrationExecutionListRes(value),
		IdempotencyScope:            value.IdempotencyScope, IdempotencyKey: integrationIdentifierSummary(value.IdempotencyKey),
		InputHash: value.InputHash,
		InputSummary: IntegrationExecutionInputSummaryRes{
			SnapshotVersion: inputSummary.SnapshotVersion, SizeBytes: inputSummary.SizeBytes,
			PathCount: inputSummary.PathCount, QueryCount: inputSummary.QueryCount,
			HeaderCount: inputSummary.HeaderCount, HasBody: inputSummary.HasBody,
		},
		ResultHTTPStatus: value.ResultHTTPStatus,
		ResultSizeBytes:  value.ResultSizeBytes, ResultHash: value.ResultHash,
		ResultSummary:     value.ResultSummary,
		LeaseOwnerSummary: integrationWorkerSummary(value.LeaseOwner), LeaseExpiresAt: value.LeaseExpiresAt,
		CancelledAt: value.CancelledAt, LastAttemptAt: value.LastAttemptAt,
	}
	if value.SyncBatchID != nil {
		result.SyncBusiness = &IntegrationExecutionSyncBusinessRes{
			Status: value.SyncBusinessStatus, ReasonCode: value.SyncBusinessReasonCode,
			SuccessCount: value.SyncBusinessSuccessCount, FailedCount: value.SyncBusinessFailedCount,
			Reference: value.SyncBusinessReference,
		}
	}
	result.RetryPolicy = policy
	result.AttemptsRemaining = max(0, result.MaxAttempts-value.CurrentAttempt)
	return result
}

func integrationExecutionRetryPolicySummary(value model.IntegrationExecution) *IntegrationExecutionRetryPolicySummaryRes {
	if value.RetryPolicySnapshotVersion <= 0 || len(value.RetryPolicySnapshot) == 0 {
		return nil
	}
	var snapshot struct {
		PolicyCode    string `json:"policy_code"`
		PolicyVersion int    `json:"policy_version"`
		MaxAttempts   int    `json:"max_attempts"`
	}
	if json.Unmarshal(value.RetryPolicySnapshot, &snapshot) != nil || snapshot.PolicyCode == "" || snapshot.MaxAttempts < 1 {
		return nil
	}
	return &IntegrationExecutionRetryPolicySummaryRes{
		PolicyCode: snapshot.PolicyCode, PolicyVersion: snapshot.PolicyVersion, MaxAttempts: snapshot.MaxAttempts,
	}
}

func integrationExecutionInputSummary(value model.IntegrationExecution) IntegrationExecutionInputSummaryRes {
	summary := IntegrationExecutionInputSummaryRes{
		SnapshotVersion: value.InputSnapshotVersion,
		SizeBytes:       value.InputSnapshotSize,
	}
	if value.InputSnapshotVersion <= 0 || len(value.InputSnapshot) == 0 {
		return summary
	}
	var snapshot struct {
		PathParams  map[string]string   `json:"path_params"`
		QueryParams map[string][]string `json:"query_params"`
		Headers     map[string][]string `json:"headers"`
		JSONBody    json.RawMessage     `json:"json_body"`
	}
	if json.Unmarshal(value.InputSnapshot, &snapshot) != nil {
		return summary
	}
	summary.PathCount = len(snapshot.PathParams)
	summary.QueryCount = len(snapshot.QueryParams)
	summary.HeaderCount = len(snapshot.Headers)
	summary.HasBody = len(snapshot.JSONBody) > 0
	return summary
}
