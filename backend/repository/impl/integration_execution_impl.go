package impl

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/database"
	"backend/model"
	"backend/repository"
	queryutil "backend/repository/util"
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IntegrationExecutionRepositoryImpl struct {
	*BasicRepositoryImpl[model.IntegrationExecution]
}

func NewIntegrationExecutionRepositoryImpl(primaryDB *database.PrimaryDB) *IntegrationExecutionRepositoryImpl {
	return &IntegrationExecutionRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.IntegrationExecution{}),
	}
}

func (r *IntegrationExecutionRepositoryImpl) GetIntegrationExecutionList(
	ctx context.Context,
	basic *request.Basic,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (response.ListResult[model.IntegrationExecution], error) {
	var values []model.IntegrationExecution
	query := queryutil.ExecuteQuery(r.DBWithContext(ctx).Table(table.TableCode), basic, table)
	query, err := queryutil.ApplyGeneralizationPermission(query, permission, table)
	if err != nil {
		return response.ListResult[model.IntegrationExecution]{}, err
	}
	total, err := r.PaginateAndCountQuery(query, &values)
	return response.ListResult[model.IntegrationExecution]{Data: values, Total: int(total)}, err
}

func (r *IntegrationExecutionRepositoryImpl) FindByIDWithPermission(
	ctx context.Context,
	id int,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (model.IntegrationExecution, error) {
	var value model.IntegrationExecution
	query, err := queryutil.ApplyGeneralizationPermission(
		r.DBWithContext(ctx).Table(table.TableCode), permission, table,
	)
	if err != nil {
		return value, err
	}
	err = query.Where(table.TableCode+".id = ?", id).Take(&value).Error
	return value, err
}

func (r *IntegrationExecutionRepositoryImpl) FindByIdempotency(
	db *gorm.DB,
	interfaceDefinitionID int,
	interfaceVersion int,
	scope string,
	key string,
) (model.IntegrationExecution, error) {
	var value model.IntegrationExecution
	err := db.Model(&model.IntegrationExecution{}).
		Where(
			"interface_definition_id = ? AND interface_version = ? AND idempotency_scope = ? AND idempotency_key = ?",
			interfaceDefinitionID,
			interfaceVersion,
			scope,
			key,
		).
		First(&value).Error
	return value, err
}

func (r *IntegrationExecutionRepositoryImpl) ListCandidatesByStatus(
	ctx context.Context,
	statuses []string,
	limit int,
) ([]model.IntegrationExecution, error) {
	if len(statuses) == 0 || limit <= 0 {
		return []model.IntegrationExecution{}, nil
	}
	var values []model.IntegrationExecution
	err := r.DBWithContext(ctx).
		Model(&model.IntegrationExecution{}).
		Where("status IN ?", statuses).
		Order("gmt_create ASC, id ASC").
		Limit(limit).
		Find(&values).Error
	return values, err
}

// ClaimCreatedExecutions 在一个短事务中使用行锁领取 created Execution 并创建 running Attempt。
// PostgreSQL 会生成 FOR UPDATE SKIP LOCKED；SQLite 测试仅验证状态条件和原子回滚，不替代行锁专项验证。
func (r *IntegrationExecutionRepositoryImpl) ClaimCreatedExecutions(
	ctx context.Context,
	request repository.IntegrationExecutionClaimRequest,
) ([]repository.ClaimedIntegrationExecution, error) {
	if strings.TrimSpace(request.WorkerID) == "" || request.LeaseExpiresAt.IsZero() || request.StartedAt.IsZero() ||
		!request.LeaseExpiresAt.After(request.StartedAt) || len(request.AttemptIDs) == 0 {
		return nil, repository.ErrIntegrationExecutionClaimUnavailable
	}
	claimed := make([]repository.ClaimedIntegrationExecution, 0, len(request.AttemptIDs))
	err := r.ExecuteTx(ctx, func(tx *gorm.DB) error {
		var executions []model.IntegrationExecution
		query := tx.Model(&model.IntegrationExecution{}).
			Where(
				"status = ? AND input_snapshot_version = ? AND input_snapshot_size > 0",
				model.IntegrationExecutionStatusCreated,
				model.IntegrationExecutionInputSnapshotVersion,
			).
			Order("gmt_create ASC, id ASC").
			Limit(len(request.AttemptIDs)).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		if err := query.Find(&executions).Error; err != nil {
			return err
		}
		for index := range executions {
			execution := executions[index]
			nextAttempt := execution.CurrentAttempt + 1
			updates := map[string]any{
				"status":           model.IntegrationExecutionStatusRunning,
				"lease_owner":      request.WorkerID,
				"lease_expires_at": request.LeaseExpiresAt,
				"started_at":       request.StartedAt,
				"current_attempt":  nextAttempt,
				"revision":         execution.Revision + 1,
			}
			result := tx.Model(&model.IntegrationExecution{}).
				Where("id = ? AND status = ? AND revision = ?", execution.Id, model.IntegrationExecutionStatusCreated, execution.Revision).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return repository.ErrIntegrationExecutionClaimUnavailable
			}
			attempt := model.IntegrationLog{
				Basic:       model.Basic{Id: request.AttemptIDs[index], State: true},
				ExecutionID: execution.Id, AttemptNo: nextAttempt,
				Status: model.IntegrationLogStatusRunning, StartedAt: request.StartedAt,
				ResultCertainty: model.IntegrationResultCertaintyUnknown,
				RequestID:       request.RequestID, TraceID: request.TraceID, WorkerID: request.WorkerID,
			}
			if err := tx.Create(&attempt).Error; err != nil {
				return err
			}
			execution.Status = model.IntegrationExecutionStatusRunning
			execution.LeaseOwner = request.WorkerID
			execution.LeaseExpiresAt = &request.LeaseExpiresAt
			execution.StartedAt = &request.StartedAt
			execution.CurrentAttempt = nextAttempt
			execution.Revision++
			claimed = append(claimed, repository.ClaimedIntegrationExecution{Execution: execution, Attempt: attempt})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// CompleteAttemptAndExecution 在新的短事务内验证租约并原子收敛 Attempt 和 Execution。
func (r *IntegrationExecutionRepositoryImpl) CompleteAttemptAndExecution(
	ctx context.Context,
	completion repository.IntegrationAttemptCompletion,
) (model.IntegrationExecution, error) {
	var completed model.IntegrationExecution
	if completion.ExecutionID <= 0 || completion.AttemptID <= 0 || completion.AttemptNo <= 0 ||
		completion.ExpectedRevision <= 0 || strings.TrimSpace(completion.WorkerID) == "" || completion.CompletedAt.IsZero() {
		return completed, repository.ErrIntegrationExecutionLeaseLost
	}
	err := r.ExecuteTx(ctx, func(tx *gorm.DB) error {
		execution, err := r.FindByIdForUpdate(tx, completion.ExecutionID)
		if err != nil {
			return err
		}
		if execution.Status != model.IntegrationExecutionStatusRunning || execution.Revision != completion.ExpectedRevision ||
			execution.LeaseOwner != completion.WorkerID || execution.LeaseExpiresAt == nil {
			return repository.ErrIntegrationExecutionLeaseLost
		}
		var attempt model.IntegrationLog
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND execution_id = ? AND attempt_no = ?", completion.AttemptID, completion.ExecutionID, completion.AttemptNo).
			First(&attempt).Error; err != nil {
			return err
		}
		if attempt.Status != model.IntegrationLogStatusRunning {
			return repository.ErrIntegrationAttemptAlreadyCompleted
		}
		duration := completion.CompletedAt.Sub(attempt.StartedAt).Milliseconds()
		if duration < 0 {
			duration = 0
		}
		attemptUpdates := map[string]any{
			"status":                         attemptTerminalStatus(completion.ExecutionStatus),
			"ended_at":                       completion.CompletedAt,
			"duration_ms":                    duration,
			"http_status":                    completion.HTTPStatus,
			"error_category":                 completion.ErrorCategory,
			"error_code":                     completion.ErrorCode,
			"result_summary":                 completion.ResultSummary,
			"result_size_bytes":              completion.ResultSizeBytes,
			"result_hash":                    completion.ResultHash,
			"result_certainty":               completion.ResultCertainty,
			"response_content_type":          completion.ResponseContentType,
			"credential_code":                completion.CredentialCode,
			"credential_version":             completion.CredentialVersion,
			"credential_fingerprint_summary": completion.CredentialFingerprintSummary,
			"retryable":                      completion.Retryable,
			"retry_reason_code":              completion.RetryReasonCode,
			"retry_delay_ms":                 completion.RetryDelayMs,
			"retry_scheduled_at":             completion.RetryScheduledAt,
			"retry_after_source":             completion.RetryAfterSource,
		}
		if result := tx.Model(&model.IntegrationLog{}).Where("id = ? AND status = ?", attempt.Id, model.IntegrationLogStatusRunning).Updates(attemptUpdates); result.Error != nil {
			return result.Error
		} else if result.RowsAffected != 1 {
			return repository.ErrIntegrationAttemptAlreadyCompleted
		}
		var completedAt any = completion.CompletedAt
		if completion.ExecutionStatus == model.IntegrationExecutionStatusRetryWaiting {
			completedAt = nil
		}
		executionUpdates := map[string]any{
			"status":             completion.ExecutionStatus,
			"lease_owner":        "",
			"lease_expires_at":   nil,
			"result_http_status": completion.HTTPStatus,
			"result_size_bytes":  completion.ResultSizeBytes,
			"result_hash":        completion.ResultHash,
			"result_summary":     completion.ResultSummary,
			"error_category":     completion.ErrorCategory,
			"completed_at":       completedAt,
			"next_run_at":        completion.RetryScheduledAt,
			"last_attempt_at":    completion.CompletedAt,
			"retry_reason_code":  completion.RetryReasonCode,
			"revision":           execution.Revision + 1,
		}
		result := tx.Model(&model.IntegrationExecution{}).
			Where("id = ? AND status = ? AND revision = ? AND lease_owner = ? AND lease_expires_at > ?", execution.Id, model.IntegrationExecutionStatusRunning, execution.Revision, completion.WorkerID, completion.CompletedAt).
			Updates(executionUpdates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return repository.ErrIntegrationExecutionLeaseLost
		}
		completed = execution
		completed.Status = completion.ExecutionStatus
		completed.LeaseOwner = ""
		completed.LeaseExpiresAt = nil
		completed.ResultHTTPStatus = completion.HTTPStatus
		completed.ResultSizeBytes = completion.ResultSizeBytes
		completed.ResultHash = completion.ResultHash
		completed.ResultSummary = completion.ResultSummary
		completed.ErrorCategory = completion.ErrorCategory
		completed.CompletedAt = &completion.CompletedAt
		if completion.ExecutionStatus == model.IntegrationExecutionStatusRetryWaiting {
			completed.CompletedAt = nil
		}
		completed.NextRunAt = completion.RetryScheduledAt
		completed.LastAttemptAt = &completion.CompletedAt
		completed.RetryReasonCode = completion.RetryReasonCode
		completed.Revision++
		return nil
	})
	return completed, err
}

func (r *IntegrationExecutionRepositoryImpl) FindExpiredRunningExecutions(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]model.IntegrationExecution, error) {
	if limit <= 0 {
		return []model.IntegrationExecution{}, nil
	}
	var values []model.IntegrationExecution
	err := r.DBWithContext(ctx).Model(&model.IntegrationExecution{}).
		Where("status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?", model.IntegrationExecutionStatusRunning, now).
		Order("lease_expires_at ASC, id ASC").Limit(limit).Find(&values).Error
	return values, err
}

// RecoverExpiredExecution 不重新发起远端调用，只将遗留 Attempt 和 Execution 收敛为未知失败。
func (r *IntegrationExecutionRepositoryImpl) RecoverExpiredExecution(
	ctx context.Context,
	recovery repository.ExpiredExecutionRecovery,
) (bool, error) {
	if recovery.ExecutionID <= 0 || recovery.RecoveredAt.IsZero() {
		return false, repository.ErrIntegrationExecutionLeaseLost
	}
	recovered := false
	err := r.ExecuteTx(ctx, func(tx *gorm.DB) error {
		var execution model.IntegrationExecution
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ?", recovery.ExecutionID, model.IntegrationExecutionStatusRunning, recovery.RecoveredAt).
			First(&execution).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var attempt model.IntegrationLog
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("execution_id = ? AND attempt_no = ?", execution.Id, execution.CurrentAttempt).
			First(&attempt).Error; err != nil {
			return err
		}
		if attempt.Status != model.IntegrationLogStatusRunning {
			return repository.ErrIntegrationAttemptAlreadyCompleted
		}
		duration := recovery.RecoveredAt.Sub(attempt.StartedAt).Milliseconds()
		if duration < 0 {
			duration = 0
		}
		if result := tx.Model(&model.IntegrationLog{}).Where("id = ? AND status = ?", attempt.Id, model.IntegrationLogStatusRunning).Updates(map[string]any{
			"status": model.IntegrationLogStatusFailed, "ended_at": recovery.RecoveredAt, "duration_ms": duration,
			"error_category": model.IntegrationErrorCategoryConcurrency, "error_code": "lease_expired",
			"result_summary": "执行租约已过期，远端处理结果无法确认", "result_certainty": model.IntegrationResultCertaintyUnknown,
		}); result.Error != nil {
			return result.Error
		} else if result.RowsAffected != 1 {
			return repository.ErrIntegrationAttemptAlreadyCompleted
		}
		result := tx.Model(&model.IntegrationExecution{}).
			Where("id = ? AND status = ? AND revision = ? AND lease_expires_at <= ?", execution.Id, model.IntegrationExecutionStatusRunning, execution.Revision, recovery.RecoveredAt).
			Updates(map[string]any{
				"status": model.IntegrationExecutionStatusFailed, "lease_owner": "", "lease_expires_at": nil,
				"completed_at": recovery.RecoveredAt, "next_run_at": nil,
				"error_category": model.IntegrationErrorCategoryConcurrency,
				"result_summary": "执行租约已过期，远端处理结果无法确认", "revision": execution.Revision + 1,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return repository.ErrIntegrationExecutionLeaseLost
		}
		recovered = true
		return nil
	})
	return recovered, err
}

func attemptTerminalStatus(executionStatus string) string {
	if executionStatus == model.IntegrationExecutionStatusSucceeded {
		return model.IntegrationLogStatusSucceeded
	}
	return model.IntegrationLogStatusFailed
}

type IntegrationLogRepositoryImpl struct {
	*BasicRepositoryImpl[model.IntegrationLog]
}

func NewIntegrationLogRepositoryImpl(primaryDB *database.PrimaryDB) *IntegrationLogRepositoryImpl {
	return &IntegrationLogRepositoryImpl{
		BasicRepositoryImpl: NewBasicRepositoryImpl(primaryDB.DB, &model.IntegrationLog{}),
	}
}

func (r *IntegrationLogRepositoryImpl) GetIntegrationLogList(
	ctx context.Context,
	req request.IntegrationLogQueryReq,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (response.ListResult[model.IntegrationLog], error) {
	var values []model.IntegrationLog
	basic := req.ToBasic()
	query := queryutil.ExecuteQuery(r.DBWithContext(ctx).Table(table.TableCode), &basic, table)
	keyword := ""
	if req.QuickQuery != nil {
		keyword = strings.TrimSpace(req.QuickQuery.Keyword)
	}
	if req.ExecutionNo != "" || req.ExternalSystemID > 0 || req.InterfaceDefinitionID > 0 || keyword != "" {
		query = query.Joins("JOIN integration_execution ON integration_execution.id = integration_log.execution_id")
	}
	if req.ExecutionNo != "" {
		query = query.Where("integration_execution.execution_no = ?", strings.TrimSpace(req.ExecutionNo))
	}
	if req.ExternalSystemID > 0 {
		query = query.Where("integration_execution.external_system_id = ?", req.ExternalSystemID)
	}
	if req.InterfaceDefinitionID > 0 {
		query = query.Where("integration_execution.interface_definition_id = ?", req.InterfaceDefinitionID)
	}
	if keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("(integration_execution.execution_no LIKE ? OR integration_execution.external_system_code LIKE ? OR integration_execution.external_system_name LIKE ? OR integration_execution.interface_code LIKE ? OR integration_execution.interface_name LIKE ? OR integration_log.worker_id LIKE ?)", pattern, pattern, pattern, pattern, pattern, pattern)
	}
	query, err := queryutil.ApplyGeneralizationPermission(query, permission, table)
	if err != nil {
		return response.ListResult[model.IntegrationLog]{}, err
	}
	total, err := r.Count(query.Session(&gorm.Session{}))
	if err != nil {
		return response.ListResult[model.IntegrationLog]{}, err
	}
	if err = query.Preload("Execution").Find(&values).Error; err != nil {
		return response.ListResult[model.IntegrationLog]{}, err
	}
	return response.ListResult[model.IntegrationLog]{Data: values, Total: int(total)}, nil
}

func (r *IntegrationLogRepositoryImpl) FindByIDWithPermission(
	ctx context.Context,
	id int,
	table model.SysTable,
	permission repository.GeneralizationPermission,
) (model.IntegrationLog, error) {
	var value model.IntegrationLog
	query, err := queryutil.ApplyGeneralizationPermission(r.DBWithContext(ctx).Table(table.TableCode), permission, table)
	if err != nil {
		return value, err
	}
	err = query.Preload("Execution").Where(table.TableCode+".id = ?", id).Take(&value).Error
	return value, err
}

var _ repository.IntegrationExecutionRepository = (*IntegrationExecutionRepositoryImpl)(nil)
var _ repository.IntegrationLogRepository = (*IntegrationLogRepositoryImpl)(nil)
