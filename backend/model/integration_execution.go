package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	IntegrationExecutionInputSnapshotVersion = 1

	IntegrationExecutionStatusCreated      = "created"
	IntegrationExecutionStatusRunning      = "running"
	IntegrationExecutionStatusRetryWaiting = "retry_waiting"
	IntegrationExecutionStatusSucceeded    = "succeeded"
	IntegrationExecutionStatusFailed       = "failed"
	IntegrationExecutionStatusCancelled    = "cancelled"

	IntegrationTriggerSourceManual      = "manual"
	IntegrationTriggerSourceSystemEvent = "system_event"
	IntegrationTriggerSourceScheduled   = "scheduled"

	IntegrationLogStatusRunning   = "running"
	IntegrationLogStatusSucceeded = "succeeded"
	IntegrationLogStatusFailed    = "failed"
	IntegrationLogStatusCancelled = "cancelled"

	IntegrationResultCertaintyConfirmed = "confirmed"
	IntegrationResultCertaintyUnknown   = "unknown"

	IntegrationErrorCategoryConfiguration = "configuration"
	IntegrationErrorCategoryCredential    = "credential"
	IntegrationErrorCategoryNetwork       = "network"
	IntegrationErrorCategoryTimeout       = "timeout"
	IntegrationErrorCategoryRemote        = "remote"
	IntegrationErrorCategoryResponse      = "response"
	IntegrationErrorCategoryBusiness      = "business"
	IntegrationErrorCategoryConcurrency   = "concurrency"
	IntegrationErrorCategorySystem        = "system"
)

// IntegrationExecution 表示一次逻辑集成调用，只保存经过契约校验的非敏感输入快照。
type IntegrationExecution struct {
	Basic

	ExecutionNo           string         `gorm:"size:64;not null;uniqueIndex:uni_integration_execution_no" json:"execution_no"`
	ExternalSystemID      int            `gorm:"type:bigint;not null;index:idx_integration_execution_system" json:"external_system_id"`
	ExternalSystemCode    string         `gorm:"size:64;not null;index:idx_integration_execution_system_code" json:"external_system_code"`
	ExternalSystemName    string         `gorm:"size:128;not null" json:"external_system_name"`
	InterfaceDefinitionID int            `gorm:"type:bigint;not null;index:idx_integration_execution_interface;uniqueIndex:uni_integration_execution_idempotency,priority:1" json:"interface_definition_id"`
	InterfaceCode         string         `gorm:"size:64;not null;index:idx_integration_execution_interface_code" json:"interface_code"`
	InterfaceName         string         `gorm:"size:128;not null" json:"interface_name"`
	InterfaceVersion      int            `gorm:"not null;uniqueIndex:uni_integration_execution_idempotency,priority:2" json:"interface_version"`
	TriggerSource         string         `gorm:"size:32;not null;index:idx_integration_execution_trigger" json:"trigger_source"`
	Status                string         `gorm:"size:32;not null;default:created;index:idx_integration_execution_status" json:"status"`
	IdempotencyScope      string         `gorm:"size:64;not null;uniqueIndex:uni_integration_execution_idempotency,priority:3" json:"idempotency_scope"`
	IdempotencyKey        string         `gorm:"size:128;not null;uniqueIndex:uni_integration_execution_idempotency,priority:4" json:"idempotency_key"`
	InputHash             string         `gorm:"size:64;not null" json:"input_hash"`
	InputSnapshot         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	InputSnapshotVersion  int            `gorm:"not null;default:0" json:"-"`
	InputSnapshotSize     int            `gorm:"not null;default:0" json:"-"`
	CurrentAttempt        int            `gorm:"not null;default:0" json:"current_attempt"`
	LeaseOwner            string         `gorm:"size:128;index:idx_integration_execution_lease_owner" json:"-"`
	LeaseExpiresAt        *time.Time     `gorm:"type:timestamp;index:idx_integration_execution_lease_expires_at" json:"-"`
	ResultHTTPStatus      *int           `gorm:"index:idx_integration_execution_http_status" json:"result_http_status"`
	ResultSizeBytes       int64          `gorm:"not null;default:0" json:"result_size_bytes"`
	ResultHash            string         `gorm:"size:64" json:"result_hash"`
	ResultSummary         string         `gorm:"size:1024" json:"result_summary"`
	ErrorCategory         string         `gorm:"size:32;index:idx_integration_execution_error_category" json:"error_category"`
	StartedAt             *time.Time     `gorm:"type:timestamp;index:idx_integration_execution_started_at" json:"started_at"`
	CompletedAt           *time.Time     `gorm:"type:timestamp;index:idx_integration_execution_completed_at" json:"completed_at"`
	NextRunAt             *time.Time     `gorm:"type:timestamp;index:idx_integration_execution_next_run_at" json:"next_run_at"`
	CancelledAt           *time.Time     `gorm:"type:timestamp" json:"cancelled_at"`
	Revision              int            `gorm:"not null;default:1" json:"revision"`

	ExternalSystem      ExternalSystem      `gorm:"foreignKey:ExternalSystemID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	InterfaceDefinition InterfaceDefinition `gorm:"foreignKey:InterfaceDefinitionID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

func (IntegrationExecution) TableName() string {
	return "integration_execution"
}

// IntegrationLog 表示一次真实调用 Attempt；记录只能追加，不能覆盖历史 Attempt。
type IntegrationLog struct {
	Basic

	ExecutionID                  int        `gorm:"type:bigint;not null;index:idx_integration_log_execution;uniqueIndex:uni_integration_log_attempt,priority:1" json:"execution_id"`
	AttemptNo                    int        `gorm:"not null;uniqueIndex:uni_integration_log_attempt,priority:2" json:"attempt_no"`
	Status                       string     `gorm:"size:32;not null;index:idx_integration_log_status" json:"status"`
	StartedAt                    time.Time  `gorm:"type:timestamp;not null;index:idx_integration_log_started_at" json:"started_at"`
	EndedAt                      *time.Time `gorm:"type:timestamp" json:"ended_at"`
	DurationMs                   int64      `gorm:"not null;default:0" json:"duration_ms"`
	HTTPStatus                   *int       `gorm:"index:idx_integration_log_http_status" json:"http_status"`
	ErrorCategory                string     `gorm:"size:32;index:idx_integration_log_error_category" json:"error_category"`
	ErrorCode                    string     `gorm:"size:64" json:"error_code"`
	ResultSummary                string     `gorm:"size:1024" json:"result_summary"`
	ResultSizeBytes              int64      `gorm:"not null;default:0" json:"result_size_bytes"`
	ResultHash                   string     `gorm:"size:64" json:"result_hash"`
	ResultCertainty              string     `gorm:"size:16;not null;default:unknown" json:"result_certainty"`
	RequestID                    string     `gorm:"size:128;index:idx_integration_log_request_id" json:"request_id"`
	TraceID                      string     `gorm:"size:128;index:idx_integration_log_trace_id" json:"trace_id"`
	WorkerID                     string     `gorm:"size:128;index:idx_integration_log_worker_id" json:"-"`
	ResponseContentType          string     `gorm:"size:128" json:"-"`
	CredentialCode               string     `gorm:"size:64" json:"-"`
	CredentialVersion            string     `gorm:"size:32" json:"-"`
	CredentialFingerprintSummary string     `gorm:"size:32" json:"-"`

	Execution IntegrationExecution `gorm:"foreignKey:ExecutionID;references:Id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}

func (IntegrationLog) TableName() string {
	return "integration_log"
}
