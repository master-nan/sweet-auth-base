package service

import (
	"backend/internal/integration"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	myerrors "backend/internal/errors"

	"gorm.io/gorm"
)

const syncBatchReasonExecutionFailed = "sync_execution_failed"
const syncBatchReasonBusinessFailed = "sync_business_failed"
const syncBatchReasonExecutionCreateFailed = "sync_execution_create_failed"
const syncBatchReasonWindowInvalid = "sync_window_invalid"

// SyncExecutionApplication 是 Sync 唯一允许使用的 Execution 创建端口。
type SyncExecutionApplication interface {
	CreateSyncExecution(context.Context, SyncExecutionCreateCommand) (model.IntegrationExecution, error)
}

// SyncRunSummary 是 Runner 可观测的单轮安全摘要。
type SyncRunSummary = integration.SyncRuntimeSummary

// IntegrationSyncCoordinator 只创建和协调 Batch/Execution，不执行 HTTP、Credential 或 Retry。
type IntegrationSyncCoordinator struct {
	tasks       repository.IntegrationSyncTaskRepository
	batches     repository.IntegrationSyncBatchRepository
	executions  repository.IntegrationExecutionRepository
	systems     repository.ExternalSystemRepository
	interfaces  repository.InterfaceDefinitionRepository
	application SyncExecutionApplication
	business    integration.SyncBusinessResultProvider
	sf          *utils.Snowflake
}

func NewIntegrationSyncCoordinator(
	tasks repository.IntegrationSyncTaskRepository,
	batches repository.IntegrationSyncBatchRepository,
	executions repository.IntegrationExecutionRepository,
	systems repository.ExternalSystemRepository,
	interfaces repository.InterfaceDefinitionRepository,
	application *IntegrationExecutionService,
	business integration.SyncBusinessResultProvider,
	sf *utils.Snowflake,
) *IntegrationSyncCoordinator {
	if business == nil {
		business = integration.NewPendingSyncBusinessResultProvider()
	}
	return &IntegrationSyncCoordinator{tasks: tasks, batches: batches, executions: executions, systems: systems, interfaces: interfaces, application: application, business: business, sf: sf}
}

// RunOnce 先创建到期批次，再协调所有活动批次；两个阶段互不持有对方事务。
func (s *IntegrationSyncCoordinator) RunOnce(ctx context.Context, scheduleLimit, coordinateLimit int) (SyncRunSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var summary SyncRunSummary
	scheduled, scheduleErr := s.ScheduleDueTasks(ctx, scheduleLimit)
	summary.ScheduledBatches = scheduled
	coordinateErr := s.coordinateActiveBatches(ctx, coordinateLimit, &summary)
	return summary, errors.Join(scheduleErr, coordinateErr)
}

// ScheduleDueTasks 在一个短事务内使用数据库时间和 SKIP LOCKED 创建唯一批次。
func (s *IntegrationSyncCoordinator) ScheduleDueTasks(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		return 0, myerrors.ErrSyncRunnerInvalidConfig
	}
	created := 0
	err := RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		databaseNow, err := s.batches.CurrentDatabaseTime(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		candidates, err := s.batches.FindScheduledCandidates(tx, limit)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		for _, task := range candidates {
			active, err := s.batches.CountActiveByTaskCode(tx, task.TaskCode)
			if err != nil {
				return myerrors.WrapDatabaseError(err)
			}
			if active > 0 {
				continue
			}
			if task.NextScheduledAt == nil {
				continue
			}
			scheduledFor := task.NextScheduledAt.UTC()
			batch, err := s.newScheduledBatch(tx, task, scheduledFor, databaseNow)
			if err != nil {
				return err
			}
			next, err := nextSyncSchedule(task.CronExpression, task.Timezone, databaseNow)
			if err != nil {
				return err
			}
			batch.TaskRevision = task.Revision + 1
			if err := s.batches.Create(tx, &batch); err != nil {
				if isSyncDuplicate(err) {
					return myerrors.ErrSyncBatchConflict
				}
				return myerrors.WrapDatabaseError(err)
			}
			updated, err := s.tasks.UpdateFieldsByRevision(tx, task.Id, task.Revision, map[string]any{
				"last_scheduled_at": &scheduledFor, "next_scheduled_at": next, "revision": task.Revision + 1,
			})
			if err != nil {
				return myerrors.WrapDatabaseError(err)
			}
			if !updated {
				return myerrors.ErrSyncSchedulerClaimFailed
			}
			created++
		}
		return nil
	})
	return created, err
}

func (s *IntegrationSyncCoordinator) newScheduledBatch(tx *gorm.DB, task model.IntegrationSyncTask, scheduledFor, databaseNow time.Time) (model.IntegrationSyncBatch, error) {
	system, err := s.systems.FindByIdWithDB(tx, task.ExternalSystemID)
	if err != nil {
		return model.IntegrationSyncBatch{}, myerrors.ErrSyncInterfaceInvalid
	}
	definition, err := s.interfaces.FindByIdWithDB(tx, task.InterfaceDefinitionID)
	if err != nil || definition.ExternalSystemID != system.Id {
		return model.IntegrationSyncBatch{}, myerrors.ErrSyncInterfaceInvalid
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return model.IntegrationSyncBatch{}, myerrors.WrapSystemError(err)
	}
	return newSyncBatchSnapshot(task, system, definition, int(id), syncBatchTrigger{
		triggerType:  model.IntegrationSyncTriggerScheduled,
		triggerKey:   fmt.Sprintf("schedule:%s:%d", task.TaskCode, scheduledFor.Unix()),
		scheduledFor: &scheduledFor,
	}, databaseNow)
}

type syncBatchTrigger struct {
	triggerType  string
	triggerKey   string
	scheduledFor *time.Time
	userID       *int
	userName     string
}

func newSyncBatchSnapshot(task model.IntegrationSyncTask, system model.ExternalSystem, definition model.InterfaceDefinition, id int, trigger syncBatchTrigger, databaseNow time.Time) (model.IntegrationSyncBatch, error) {
	batch := model.IntegrationSyncBatch{
		Basic: model.Basic{Id: int(id), State: true}, BatchNo: fmt.Sprintf("SYNC-%d", id), SyncTaskID: task.Id,
		TaskCode: task.TaskCode, TaskName: task.TaskName, TaskVersion: task.Version, TaskRevision: task.Revision,
		SystemCode: system.SystemCode, InterfaceCode: definition.InterfaceCode, InterfaceVersion: definition.Version,
		ConsumerCode: task.ConsumerCode, ConsumerVersion: task.ConsumerVersion,
		TriggerType: trigger.triggerType, TriggerKey: trigger.triggerKey, ScheduledFor: trigger.scheduledFor,
		TriggeredByUserID: trigger.userID, TriggeredByUserName: trigger.userName,
		Status: model.IntegrationSyncBatchStatusCreated, CheckpointMode: task.CheckpointMode,
		LookbackSeconds: task.LookbackSeconds, WindowSliceSeconds: task.WindowSliceSeconds, Revision: 1,
	}
	if task.CheckpointMode == model.IntegrationSyncCheckpointTimestamp {
		if task.CheckpointAt == nil {
			return model.IntegrationSyncBatch{}, myerrors.ErrSyncCheckpointInvalid
		}
		start := task.CheckpointAt.UTC()
		end := databaseNow
		if trigger.scheduledFor != nil && trigger.scheduledFor.Before(end) {
			end = trigger.scheduledFor.UTC()
		}
		batch.WindowStart, batch.WindowEnd, batch.CheckpointBefore = &start, &end, &start
		if end.Before(start) {
			batch.Status = model.IntegrationSyncBatchStatusFailed
			batch.CompletedAt = &databaseNow
			batch.ReasonCode = syncBatchReasonWindowInvalid
			batch.ResultSummary = "同步窗口结束时间早于 Checkpoint"
			return batch, nil
		}
		plan, err := integration.DecodeSyncExecutionInputPlan(task.InputPlan)
		if err != nil {
			return model.IntegrationSyncBatch{}, err
		}
		if plan.WindowMode == integration.SyncWindowModeLowerBoundOnly {
			if end.After(start) {
				batch.PlannedSliceCount = 1
			}
		} else {
			batch.PlannedSliceCount = syncSliceCount(start, end, time.Duration(task.WindowSliceSeconds)*time.Second)
		}
	} else {
		batch.PlannedSliceCount = 1
	}
	return batch, nil
}

func syncSliceCount(start, end time.Time, duration time.Duration) int {
	if !end.After(start) || duration <= 0 {
		return 0
	}
	span := end.Sub(start)
	return int((span + duration - 1) / duration)
}

func (s *IntegrationSyncCoordinator) coordinateActiveBatches(ctx context.Context, limit int, summary *SyncRunSummary) error {
	if limit <= 0 {
		return myerrors.ErrSyncRunnerInvalidConfig
	}
	values, err := s.batches.FindActiveBatches(ctx, limit)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	var joined error
	for _, batch := range values {
		if ctx.Err() != nil {
			return errors.Join(joined, ctx.Err())
		}
		if err := s.CoordinateBatch(ctx, batch.Id); err != nil {
			joined = errors.Join(joined, err)
			continue
		}
		summary.Coordinated++
		latest, loadErr := s.batches.WithContext(ctx).FindById(batch.Id)
		if loadErr == nil {
			switch latest.Status {
			case model.IntegrationSyncBatchStatusSucceeded:
				summary.Succeeded++
			case model.IntegrationSyncBatchStatusFailed:
				summary.Failed++
			}
		}
	}
	return joined
}

// CoordinateBatch 幂等推进一个批次，最多创建一个新的切片 Execution。
func (s *IntegrationSyncCoordinator) CoordinateBatch(ctx context.Context, batchID int) error {
	batch, err := s.batches.WithContext(ctx).FindById(batchID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrSyncBatchNotFound
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if batch.Status != model.IntegrationSyncBatchStatusCreated && batch.Status != model.IntegrationSyncBatchStatusRunning {
		return nil
	}
	if batch.Status == model.IntegrationSyncBatchStatusCreated {
		if err := s.markBatchRunning(ctx, batch); err != nil {
			return err
		}
		batch, err = s.batches.WithContext(ctx).FindById(batchID)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	if batch.CheckpointMode == model.IntegrationSyncCheckpointTimestamp && batch.PlannedSliceCount == 0 {
		return s.completeEmptyBatch(ctx, batch)
	}

	sliceNo := batch.CurrentSliceNo
	var execution model.IntegrationExecution
	if sliceNo > 0 {
		execution, err = s.executions.FindBySyncBatchSlice(s.executions.DBWithContext(ctx), batch.Id, sliceNo)
	}
	if sliceNo == 0 || errors.Is(err, gorm.ErrRecordNotFound) || syncSliceAlreadyProcessed(batch, execution) {
		return s.createNextSlice(ctx, batch)
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	switch execution.Status {
	case model.IntegrationExecutionStatusCreated, model.IntegrationExecutionStatusRunning, model.IntegrationExecutionStatusRetryWaiting:
		return nil
	case model.IntegrationExecutionStatusSucceeded:
		result, err := s.business.Result(ctx, execution)
		if err != nil {
			return err
		}
		switch result.Status {
		case integration.SyncBusinessResultPending:
			return nil
		case integration.SyncBusinessResultFailed:
			return s.failBatch(ctx, batch, execution, syncBatchReasonBusinessFailed, result)
		case integration.SyncBusinessResultSucceeded:
			return s.advanceSuccessfulSlice(ctx, batch, execution, result)
		default:
			return myerrors.ErrSyncBusinessResultPending
		}
	case model.IntegrationExecutionStatusFailed, model.IntegrationExecutionStatusCancelled:
		if execution.SyncBusinessStatus == model.IntegrationSyncBusinessStatusFailed {
			result, resultErr := s.business.Result(ctx, execution)
			if resultErr != nil {
				return resultErr
			}
			return s.failBatch(ctx, batch, execution, syncBatchReasonBusinessFailed, result)
		}
		return s.failBatch(ctx, batch, execution, syncBatchReasonExecutionFailed, integration.SyncBusinessResult{})
	default:
		return myerrors.ErrSyncBatchStateInvalid
	}
}

func (s *IntegrationSyncCoordinator) markBatchRunning(ctx context.Context, batch model.IntegrationSyncBatch) error {
	return RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		now, err := s.batches.CurrentDatabaseTime(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		updated, err := s.batches.UpdateFieldsByRevision(tx, batch.Id, batch.Revision, map[string]any{
			"status": model.IntegrationSyncBatchStatusRunning, "started_at": &now, "revision": batch.Revision + 1,
		})
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrSyncBatchConflict
		}
		return nil
	})
}

func (s *IntegrationSyncCoordinator) createNextSlice(ctx context.Context, batch model.IntegrationSyncBatch) error {
	task, err := s.tasks.WithContext(ctx).FindById(batch.SyncTaskID)
	if err != nil {
		return s.failBatchWithoutExecution(ctx, batch, syncBatchReasonExecutionCreateFailed)
	}
	nextSlice := batch.CurrentSliceNo + 1
	if nextSlice > batch.PlannedSliceCount {
		return s.finishBatch(ctx, batch)
	}
	plan, err := integration.DecodeSyncExecutionInputPlan(task.InputPlan)
	if err != nil {
		return s.failBatchWithoutExecution(ctx, batch, syncBatchReasonExecutionCreateFailed)
	}
	windowStart, windowEnd, requestStart, err := syncSliceWindow(batch, nextSlice, plan.WindowMode)
	if err != nil {
		return s.failBatchWithoutExecution(ctx, batch, syncBatchReasonExecutionCreateFailed)
	}
	location, err := time.LoadLocation(task.Timezone)
	if err != nil {
		return s.failBatchWithoutExecution(ctx, batch, syncBatchReasonExecutionCreateFailed)
	}
	input, err := integration.MaterializeSyncExecutionInputPlanInLocation(task.InputPlan, requestStart, windowEnd, location)
	if err != nil {
		return s.failBatchWithoutExecution(ctx, batch, syncBatchReasonExecutionCreateFailed)
	}
	execution, err := s.application.CreateSyncExecution(ctx, SyncExecutionCreateCommand{
		ExternalSystemID: task.ExternalSystemID, InterfaceDefinitionID: task.InterfaceDefinitionID,
		BatchID: batch.Id, SliceNo: nextSlice, WindowStart: windowStart, WindowEnd: windowEnd,
		ConsumerCode: batch.ConsumerCode, ConsumerVersion: batch.ConsumerVersion, Input: input,
	})
	if err != nil {
		return s.failBatchWithoutExecution(ctx, batch, syncBatchReasonExecutionCreateFailed)
	}
	return RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.batches.FindByIdForUpdate(tx, batch.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status != model.IntegrationSyncBatchStatusRunning {
			return myerrors.ErrSyncBatchConflict
		}
		if current.CurrentSliceNo >= nextSlice {
			return nil
		}
		if execution.SyncBatchID == nil || *execution.SyncBatchID != current.Id {
			return myerrors.ErrSyncExecutionCreateFailed
		}
		updated, err := s.batches.UpdateFieldsByRevision(tx, current.Id, current.Revision, map[string]any{
			"current_slice_no": nextSlice, "execution_count": current.ExecutionCount + 1, "revision": current.Revision + 1,
		})
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrSyncBatchConflict
		}
		return nil
	})
}

func syncSliceWindow(batch model.IntegrationSyncBatch, sliceNo int, windowMode string) (*time.Time, *time.Time, *time.Time, error) {
	if batch.CheckpointMode == model.IntegrationSyncCheckpointNone {
		return nil, nil, nil, nil
	}
	if batch.WindowStart == nil || batch.WindowEnd == nil || batch.WindowSliceSeconds <= 0 || sliceNo < 1 {
		return nil, nil, nil, myerrors.ErrSyncCheckpointInvalid
	}
	start := batch.WindowStart.Add(time.Duration(sliceNo-1) * time.Duration(batch.WindowSliceSeconds) * time.Second).UTC()
	end := batch.WindowEnd.UTC()
	if windowMode != integration.SyncWindowModeLowerBoundOnly {
		end = start.Add(time.Duration(batch.WindowSliceSeconds) * time.Second)
		if end.After(*batch.WindowEnd) {
			end = batch.WindowEnd.UTC()
		}
	} else if sliceNo != 1 {
		return nil, nil, nil, myerrors.ErrSyncCheckpointInvalid
	}
	if !end.After(start) {
		return nil, nil, nil, myerrors.ErrSyncCheckpointInvalid
	}
	requestStart := start
	if sliceNo == 1 && batch.LookbackSeconds > 0 {
		requestStart = start.Add(-time.Duration(batch.LookbackSeconds) * time.Second)
		epoch := time.Unix(0, 0).UTC()
		if requestStart.Before(epoch) {
			requestStart = epoch
		}
	}
	return &start, &end, &requestStart, nil
}

func syncSliceAlreadyProcessed(batch model.IntegrationSyncBatch, execution model.IntegrationExecution) bool {
	return batch.CheckpointMode == model.IntegrationSyncCheckpointTimestamp && batch.CheckpointAfter != nil && execution.SyncWindowEnd != nil && !batch.CheckpointAfter.Before(*execution.SyncWindowEnd)
}

func (s *IntegrationSyncCoordinator) advanceSuccessfulSlice(ctx context.Context, batch model.IntegrationSyncBatch, execution model.IntegrationExecution, result integration.SyncBusinessResult) error {
	return RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.batches.FindByIdForUpdate(tx, batch.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if execution.SyncSliceNo == nil || current.Status != model.IntegrationSyncBatchStatusRunning || current.CurrentSliceNo != *execution.SyncSliceNo {
			return myerrors.ErrSyncBatchConflict
		}
		storedExecution, err := s.executions.FindBySyncBatchSlice(tx, current.Id, current.CurrentSliceNo)
		if err != nil || storedExecution.Status != model.IntegrationExecutionStatusSucceeded {
			return myerrors.ErrSyncBatchConflict
		}
		if current.CheckpointMode == model.IntegrationSyncCheckpointTimestamp && current.CheckpointAfter != nil && storedExecution.SyncWindowEnd != nil && !current.CheckpointAfter.Before(*storedExecution.SyncWindowEnd) {
			return nil
		}
		task, err := s.tasks.FindByIdForUpdate(tx, current.SyncTaskID)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if task.Revision != current.TaskRevision {
			return myerrors.ErrSyncCheckpointConflict
		}
		now, err := s.batches.CurrentDatabaseTime(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		batchUpdates := map[string]any{
			"technical_success_count": current.TechnicalSuccessCount + 1,
			"business_success_count":  current.BusinessSuccessCount + result.SuccessCount,
			"business_failed_count":   current.BusinessFailedCount + result.FailedCount,
			"result_summary":          safeSyncSummary(result.Summary), "reason_code": "", "revision": current.Revision + 1,
		}
		taskUpdates := map[string]any{"revision": task.Revision + 1}
		if current.CheckpointMode == model.IntegrationSyncCheckpointTimestamp {
			if storedExecution.SyncWindowEnd == nil {
				return myerrors.ErrSyncCheckpointConflict
			}
			checkpoint := storedExecution.SyncWindowEnd.UTC()
			batchUpdates["checkpoint_after"] = &checkpoint
			taskUpdates["checkpoint_at"] = &checkpoint
		}
		final := current.CurrentSliceNo >= current.PlannedSliceCount
		if final {
			batchUpdates["status"] = model.IntegrationSyncBatchStatusSucceeded
			batchUpdates["completed_at"] = &now
		}
		updatedTask, err := s.tasks.UpdateFieldsByRevision(tx, task.Id, task.Revision, taskUpdates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updatedTask {
			return myerrors.ErrSyncCheckpointConflict
		}
		batchUpdates["task_revision"] = task.Revision + 1
		updatedBatch, err := s.batches.UpdateFieldsByRevision(tx, current.Id, current.Revision, batchUpdates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updatedBatch {
			return myerrors.ErrSyncBatchConflict
		}
		return nil
	})
}

func (s *IntegrationSyncCoordinator) completeEmptyBatch(ctx context.Context, batch model.IntegrationSyncBatch) error {
	return RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.batches.FindByIdForUpdate(tx, batch.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		task, err := s.tasks.FindByIdForUpdate(tx, current.SyncTaskID)
		if err != nil || task.Revision != current.TaskRevision {
			return myerrors.ErrSyncCheckpointConflict
		}
		now, err := s.batches.CurrentDatabaseTime(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		updated, err := s.batches.UpdateFieldsByRevision(tx, current.Id, current.Revision, map[string]any{
			"status": model.IntegrationSyncBatchStatusSucceeded, "checkpoint_after": current.CheckpointBefore,
			"completed_at": &now, "revision": current.Revision + 1,
		})
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrSyncBatchConflict
		}
		return nil
	})
}

func (s *IntegrationSyncCoordinator) finishBatch(ctx context.Context, batch model.IntegrationSyncBatch) error {
	return RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		now, err := s.batches.CurrentDatabaseTime(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		updated, err := s.batches.UpdateFieldsByRevision(tx, batch.Id, batch.Revision, map[string]any{
			"status": model.IntegrationSyncBatchStatusSucceeded, "completed_at": &now, "revision": batch.Revision + 1,
		})
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrSyncBatchConflict
		}
		return nil
	})
}

func (s *IntegrationSyncCoordinator) failBatch(ctx context.Context, batch model.IntegrationSyncBatch, execution model.IntegrationExecution, reason string, result integration.SyncBusinessResult) error {
	return RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.batches.FindByIdForUpdate(tx, batch.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status != model.IntegrationSyncBatchStatusRunning {
			return nil
		}
		now, err := s.batches.CurrentDatabaseTime(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		updates := map[string]any{
			"status": model.IntegrationSyncBatchStatusFailed, "completed_at": &now, "reason_code": reason,
			"result_summary": safeSyncSummary(result.Summary), "revision": current.Revision + 1,
		}
		if execution.SyncBusinessStatus == model.IntegrationSyncBusinessStatusFailed {
			updates["technical_success_count"] = current.TechnicalSuccessCount + 1
			updates["business_failed_count"] = current.BusinessFailedCount + maxInt(1, result.FailedCount)
		} else if execution.Status == model.IntegrationExecutionStatusFailed || execution.Status == model.IntegrationExecutionStatusCancelled {
			updates["technical_failed_count"] = current.TechnicalFailedCount + 1
		} else {
			updates["technical_success_count"] = current.TechnicalSuccessCount + 1
			updates["business_failed_count"] = current.BusinessFailedCount + maxInt(1, result.FailedCount)
		}
		updated, err := s.batches.UpdateFieldsByRevision(tx, current.Id, current.Revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrSyncBatchConflict
		}
		return nil
	})
}

func (s *IntegrationSyncCoordinator) failBatchWithoutExecution(ctx context.Context, batch model.IntegrationSyncBatch, reason string) error {
	return RunInTransaction(ctx, s.batches.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.batches.FindByIdForUpdate(tx, batch.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status != model.IntegrationSyncBatchStatusRunning {
			return nil
		}
		now, err := s.batches.CurrentDatabaseTime(tx)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		updated, err := s.batches.UpdateFieldsByRevision(tx, current.Id, current.Revision, map[string]any{
			"status": model.IntegrationSyncBatchStatusFailed, "completed_at": &now, "reason_code": reason,
			"result_summary": "同步 Execution 创建失败", "revision": current.Revision + 1,
		})
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !updated {
			return myerrors.ErrSyncBatchConflict
		}
		return nil
	})
}

func safeSyncSummary(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 1024 {
		return value[:1024]
	}
	return value
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
