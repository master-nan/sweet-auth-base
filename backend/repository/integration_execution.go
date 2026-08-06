package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrIntegrationExecutionClaimUnavailable 表示候选执行在领取事务内已被其他 Worker 领取或状态变化。
	ErrIntegrationExecutionClaimUnavailable = errors.New("integration execution claim unavailable")
	// ErrIntegrationExecutionLeaseLost 表示完成阶段已不再持有有效租约。
	ErrIntegrationExecutionLeaseLost = errors.New("integration execution lease lost")
	// ErrIntegrationAttemptAlreadyCompleted 表示 Attempt 已经被其他流程收敛。
	ErrIntegrationAttemptAlreadyCompleted = errors.New("integration attempt already completed")
)

// IntegrationExecutionClaimRequest 描述一个短事务内的批量领取请求。
// AttemptIDs 由调用方预先生成，避免 Repository 依赖具体 ID 生成策略。
type IntegrationExecutionClaimRequest struct {
	WorkerID       string
	LeaseExpiresAt time.Time
	StartedAt      time.Time
	RequestID      string
	TraceID        string
	AttemptIDs     []int
}

// ClaimedIntegrationExecution 是领取事务提交后的受控执行上下文。
type ClaimedIntegrationExecution struct {
	Execution model.IntegrationExecution
	Attempt   model.IntegrationLog
}

// IntegrationAttemptCompletion 是 Engine 已经完成结果判定后的原子持久化命令。
// Repository 只校验租约、revision 与 Attempt 状态，不决定重试语义。
type IntegrationAttemptCompletion struct {
	ExecutionID                  int
	AttemptID                    int
	AttemptNo                    int
	WorkerID                     string
	ExpectedRevision             int
	ExecutionStatus              string
	CompletedAt                  time.Time
	HTTPStatus                   *int
	ResultSizeBytes              int64
	ResultHash                   string
	ResultSummary                string
	ErrorCategory                string
	ErrorCode                    string
	ResultCertainty              string
	ResponseContentType          string
	CredentialCode               string
	CredentialVersion            string
	CredentialFingerprintSummary string
}

// ExpiredExecutionRecovery 用于将已过期租约的执行安全收敛为未知失败。
type ExpiredExecutionRecovery struct {
	ExecutionID int
	RecoveredAt time.Time
}

type IntegrationExecutionRepository interface {
	BasicRepository[model.IntegrationExecution]
	GetIntegrationExecutionList(context.Context, *request.Basic, model.SysTable, GeneralizationPermission) (response.ListResult[model.IntegrationExecution], error)
	FindByIDWithPermission(context.Context, int, model.SysTable, GeneralizationPermission) (model.IntegrationExecution, error)
	FindByIdempotency(*gorm.DB, int, int, string, string) (model.IntegrationExecution, error)
	ListCandidatesByStatus(context.Context, []string, int) ([]model.IntegrationExecution, error)
	ClaimCreatedExecutions(context.Context, IntegrationExecutionClaimRequest) ([]ClaimedIntegrationExecution, error)
	CompleteAttemptAndExecution(context.Context, IntegrationAttemptCompletion) (model.IntegrationExecution, error)
	FindExpiredRunningExecutions(context.Context, time.Time, int) ([]model.IntegrationExecution, error)
	RecoverExpiredExecution(context.Context, ExpiredExecutionRecovery) (bool, error)
}

type IntegrationLogRepository interface {
	BasicRepository[model.IntegrationLog]
	GetIntegrationLogList(context.Context, request.IntegrationLogQueryReq, model.SysTable, GeneralizationPermission) (response.ListResult[model.IntegrationLog], error)
	FindByIDWithPermission(context.Context, int, model.SysTable, GeneralizationPermission) (model.IntegrationLog, error)
}
