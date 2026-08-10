package integration

import (
	"backend/model"
	"context"
)

const (
	SyncBusinessResultPending   = "pending"
	SyncBusinessResultSucceeded = "succeeded"
	SyncBusinessResultFailed    = "failed"
)

// SyncBusinessResult 是 Consumer 层向协调器提供的安全完成标记，不包含响应正文。
type SyncBusinessResult struct {
	Status       string
	SuccessCount int
	FailedCount  int
	ReasonCode   string
	Summary      string
}

// SyncBusinessResultProvider 只读取已落库的业务处理结果；它不执行 Consumer。
type SyncBusinessResultProvider interface {
	Result(context.Context, model.IntegrationExecution) (SyncBusinessResult, error)
}

// PendingSyncBusinessResultProvider 是生产默认边界。INT-005C-2 接入真实 Consumer 前，
// 技术成功不会被伪造为业务成功，也不会推进 Checkpoint。
type PendingSyncBusinessResultProvider struct{}

func NewPendingSyncBusinessResultProvider() *PendingSyncBusinessResultProvider {
	return &PendingSyncBusinessResultProvider{}
}

func (*PendingSyncBusinessResultProvider) Result(context.Context, model.IntegrationExecution) (SyncBusinessResult, error) {
	return SyncBusinessResult{Status: SyncBusinessResultPending}, nil
}

// PersistedSyncBusinessResultProvider 读取 Engine 原子落库的安全 Consumer 结果摘要。
// 它不读取响应正文，也不会再次调用 Consumer。
type PersistedSyncBusinessResultProvider struct{}

func NewPersistedSyncBusinessResultProvider() *PersistedSyncBusinessResultProvider {
	return &PersistedSyncBusinessResultProvider{}
}

func (*PersistedSyncBusinessResultProvider) Result(_ context.Context, execution model.IntegrationExecution) (SyncBusinessResult, error) {
	status := execution.SyncBusinessStatus
	if status == "" {
		status = SyncBusinessResultPending
	}
	return SyncBusinessResult{
		Status: status, SuccessCount: execution.SyncBusinessSuccessCount, FailedCount: execution.SyncBusinessFailedCount,
		ReasonCode: execution.SyncBusinessReasonCode,
	}, nil
}
