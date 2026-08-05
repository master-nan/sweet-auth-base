package response

import (
	"backend/model"
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
	Id             int                              `json:"id"`
	ExecutionNo    string                           `json:"execution_no"`
	ExternalSystem IntegrationExecutionSystemRes    `json:"external_system"`
	Interface      IntegrationExecutionInterfaceRes `json:"interface"`
	TriggerSource  string                           `json:"trigger_source"`
	Status         string                           `json:"status"`
	CurrentAttempt int                              `json:"current_attempt"`
	Revision       int                              `json:"revision"`
	GmtCreate      model.CustomTime                 `json:"gmt_create"`
	GmtModify      model.CustomTime                 `json:"gmt_modify"`
	StartedAt      *time.Time                       `json:"started_at,omitempty"`
	CompletedAt    *time.Time                       `json:"completed_at,omitempty"`
}

type IntegrationLogSummaryRes struct {
	AttemptNo       int        `json:"attempt_no"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	DurationMs      int64      `json:"duration_ms"`
	HTTPStatus      *int       `json:"http_status,omitempty"`
	ErrorCategory   string     `json:"error_category,omitempty"`
	ErrorCode       string     `json:"error_code,omitempty"`
	ResultSummary   string     `json:"result_summary,omitempty"`
	ResultSizeBytes int64      `json:"result_size_bytes"`
	ResultHash      string     `json:"result_hash,omitempty"`
	ResultCertainty string     `json:"result_certainty"`
	RequestID       string     `json:"request_id,omitempty"`
	TraceID         string     `json:"trace_id,omitempty"`
}

type IntegrationExecutionDetailRes struct {
	IntegrationExecutionListRes
	IdempotencyScope string                     `json:"idempotency_scope"`
	IdempotencyKey   string                     `json:"idempotency_key"`
	InputHash        string                     `json:"input_hash"`
	ResultHTTPStatus *int                       `json:"result_http_status,omitempty"`
	ResultSizeBytes  int64                      `json:"result_size_bytes"`
	ResultHash       string                     `json:"result_hash,omitempty"`
	ResultSummary    string                     `json:"result_summary,omitempty"`
	ErrorCategory    string                     `json:"error_category,omitempty"`
	NextRunAt        *time.Time                 `json:"next_run_at,omitempty"`
	CancelledAt      *time.Time                 `json:"cancelled_at,omitempty"`
	Attempts         []IntegrationLogSummaryRes `json:"attempts"`
}

func NewIntegrationExecutionListRes(value model.IntegrationExecution) IntegrationExecutionListRes {
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
		Revision: value.Revision, GmtCreate: value.GmtCreate, GmtModify: value.GmtModify,
		StartedAt: value.StartedAt, CompletedAt: value.CompletedAt,
	}
}

func NewIntegrationLogSummaryRes(value model.IntegrationLog) IntegrationLogSummaryRes {
	return IntegrationLogSummaryRes{
		AttemptNo: value.AttemptNo, Status: value.Status, StartedAt: value.StartedAt,
		EndedAt: value.EndedAt, DurationMs: value.DurationMs, HTTPStatus: value.HTTPStatus,
		ErrorCategory: value.ErrorCategory, ErrorCode: value.ErrorCode, ResultSummary: value.ResultSummary,
		ResultSizeBytes: value.ResultSizeBytes, ResultHash: value.ResultHash,
		ResultCertainty: value.ResultCertainty, RequestID: value.RequestID, TraceID: value.TraceID,
	}
}

func NewIntegrationExecutionDetailRes(
	value model.IntegrationExecution,
	logs []model.IntegrationLog,
) IntegrationExecutionDetailRes {
	attempts := make([]IntegrationLogSummaryRes, 0, len(logs))
	for _, log := range logs {
		attempts = append(attempts, NewIntegrationLogSummaryRes(log))
	}
	return IntegrationExecutionDetailRes{
		IntegrationExecutionListRes: NewIntegrationExecutionListRes(value),
		IdempotencyScope:            value.IdempotencyScope, IdempotencyKey: value.IdempotencyKey,
		InputHash: value.InputHash, ResultHTTPStatus: value.ResultHTTPStatus,
		ResultSizeBytes: value.ResultSizeBytes, ResultHash: value.ResultHash,
		ResultSummary: value.ResultSummary, ErrorCategory: value.ErrorCategory,
		NextRunAt: value.NextRunAt, CancelledAt: value.CancelledAt, Attempts: attempts,
	}
}
