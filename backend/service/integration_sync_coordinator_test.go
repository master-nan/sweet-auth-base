package service

import (
	"backend/internal/database"
	"backend/internal/integration"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	myerrors "backend/internal/errors"
	testutil "backend/internal/test"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type syncBusinessResultStub struct {
	status string
}

func (s *syncBusinessResultStub) Result(context.Context, model.IntegrationExecution) (integration.SyncBusinessResult, error) {
	return integration.SyncBusinessResult{Status: s.status, SuccessCount: 1, Summary: "business result accepted"}, nil
}

type syncCoordinatorTestEnv struct {
	db          *gorm.DB
	coordinator *IntegrationSyncCoordinator
	task        model.IntegrationSyncTask
	audit       *integrationExecutionAuditWriter
}

func TestIntegrationSyncCoordinatorSequentialSlicesAndCheckpoint(t *testing.T) {
	env := newSyncCoordinatorTestEnv(t, integration.SyncBusinessResultSucceeded)
	summary, err := env.coordinator.RunOnce(context.Background(), 4, 4)
	if err != nil || summary.ScheduledBatches != 1 {
		t.Fatalf("first run summary=%+v err=%v", summary, err)
	}
	batch := loadOnlySyncBatch(t, env.db)
	if batch.Status != model.IntegrationSyncBatchStatusRunning || batch.CurrentSliceNo != 1 || batch.ExecutionCount != 1 || batch.PlannedSliceCount != 2 {
		t.Fatalf("first slice batch=%+v", batch)
	}
	if len(env.audit.records) != 0 {
		t.Fatalf("automatic sync execution wrote administrator audit: %d", len(env.audit.records))
	}
	first := loadSyncExecution(t, env.db, batch.Id, 1)
	if first.SyncWindowStart == nil || first.SyncWindowEnd == nil || first.Status != model.IntegrationExecutionStatusCreated {
		t.Fatalf("first execution=%+v", first)
	}
	if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
		t.Fatal(err)
	}
	batch = loadOnlySyncBatch(t, env.db)
	if batch.ExecutionCount != 1 {
		t.Fatalf("duplicate coordination created another execution: %+v", batch)
	}
	markSyncExecution(t, env.db, first.Id, model.IntegrationExecutionStatusSucceeded)
	if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
		t.Fatal(err)
	}
	batch = loadOnlySyncBatch(t, env.db)
	if batch.CheckpointAfter == nil || !batch.CheckpointAfter.Equal(*first.SyncWindowEnd) || batch.TechnicalSuccessCount != 1 {
		t.Fatalf("checkpoint after first=%+v", batch)
	}
	if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
		t.Fatal(err)
	}
	batch = loadOnlySyncBatch(t, env.db)
	if batch.CurrentSliceNo != 2 || batch.ExecutionCount != 2 {
		t.Fatalf("second slice batch=%+v", batch)
	}
	second := loadSyncExecution(t, env.db, batch.Id, 2)
	markSyncExecution(t, env.db, second.Id, model.IntegrationExecutionStatusSucceeded)
	if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
		t.Fatal(err)
	}
	batch = loadOnlySyncBatch(t, env.db)
	if batch.Status != model.IntegrationSyncBatchStatusSucceeded || batch.TechnicalSuccessCount != 2 || batch.ExecutionCount != 2 {
		t.Fatalf("final batch=%+v", batch)
	}
	var task model.IntegrationSyncTask
	if err := env.db.First(&task, env.task.Id).Error; err != nil || task.CheckpointAt == nil || !task.CheckpointAt.Equal(*batch.WindowEnd) {
		t.Fatalf("task=%+v err=%v", task, err)
	}
}

func TestIntegrationSyncCoordinatorFailureRetryWaitingAndRestartRecovery(t *testing.T) {
	t.Run("failed slice stops batch", func(t *testing.T) {
		env := newSyncCoordinatorTestEnv(t, integration.SyncBusinessResultSucceeded)
		if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
			t.Fatal(err)
		}
		batch := loadOnlySyncBatch(t, env.db)
		first := loadSyncExecution(t, env.db, batch.Id, 1)
		markSyncExecution(t, env.db, first.Id, model.IntegrationExecutionStatusFailed)
		if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
			t.Fatal(err)
		}
		batch = loadOnlySyncBatch(t, env.db)
		if batch.Status != model.IntegrationSyncBatchStatusFailed || batch.ExecutionCount != 1 {
			t.Fatalf("failed batch=%+v", batch)
		}
		var count int64
		env.db.Model(&model.IntegrationExecution{}).Where("sync_batch_id = ?", batch.Id).Count(&count)
		if count != 1 {
			t.Fatalf("execution count=%d", count)
		}
	})

	t.Run("retry waiting and pending business do not advance", func(t *testing.T) {
		env := newSyncCoordinatorTestEnv(t, integration.SyncBusinessResultPending)
		if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
			t.Fatal(err)
		}
		batch := loadOnlySyncBatch(t, env.db)
		first := loadSyncExecution(t, env.db, batch.Id, 1)
		markSyncExecution(t, env.db, first.Id, model.IntegrationExecutionStatusRetryWaiting)
		if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
			t.Fatal(err)
		}
		batch = loadOnlySyncBatch(t, env.db)
		if batch.CheckpointAfter != nil || batch.CurrentSliceNo != 1 {
			t.Fatalf("retry waiting advanced batch=%+v", batch)
		}
		markSyncExecution(t, env.db, first.Id, model.IntegrationExecutionStatusSucceeded)
		if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
			t.Fatal(err)
		}
		batch = loadOnlySyncBatch(t, env.db)
		if batch.CheckpointAfter != nil {
			t.Fatalf("pending business advanced checkpoint=%+v", batch)
		}
	})
}

func TestIntegrationSyncCoordinatorRevisionConflictAndIdempotentExecution(t *testing.T) {
	env := newSyncCoordinatorTestEnv(t, integration.SyncBusinessResultSucceeded)
	if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
		t.Fatal(err)
	}
	batch := loadOnlySyncBatch(t, env.db)
	first := loadSyncExecution(t, env.db, batch.Id, 1)
	markSyncExecution(t, env.db, first.Id, model.IntegrationExecutionStatusSucceeded)
	if err := env.db.Model(&model.IntegrationSyncTask{}).Where("id = ?", env.task.Id).Update("revision", gorm.Expr("revision + 1")).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); !errors.Is(err, myerrors.ErrSyncCheckpointConflict) {
		t.Fatalf("revision conflict=%v", err)
	}
	batch = loadOnlySyncBatch(t, env.db)
	if batch.CheckpointAfter != nil || batch.Status != model.IntegrationSyncBatchStatusRunning {
		t.Fatalf("conflict advanced batch=%+v", batch)
	}
}

func TestSyncWindowAndSliceBoundaries(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)
	if got := syncSliceCount(start, end, time.Hour); got != 2 {
		t.Fatalf("slice count=%d", got)
	}
	batch := model.IntegrationSyncBatch{CheckpointMode: model.IntegrationSyncCheckpointTimestamp, WindowStart: &start, WindowEnd: &end, WindowSliceSeconds: 3600, LookbackSeconds: 60}
	logicalStart, logicalEnd, requestStart, err := syncSliceWindow(batch, 1)
	if err != nil || !logicalStart.Equal(start) || !logicalEnd.Equal(start.Add(time.Hour)) || !requestStart.Equal(start.Add(-time.Minute)) {
		t.Fatalf("first window start=%v end=%v request=%v err=%v", logicalStart, logicalEnd, requestStart, err)
	}
	_, logicalEnd, requestStart, err = syncSliceWindow(batch, 2)
	if err != nil || !logicalEnd.Equal(end) || !requestStart.Equal(start.Add(time.Hour)) {
		t.Fatalf("second window end=%v request=%v err=%v", logicalEnd, requestStart, err)
	}
}

func TestIntegrationSyncCoordinatorEmptyAndInvalidWindows(t *testing.T) {
	t.Run("empty window succeeds without execution", func(t *testing.T) {
		env := newSyncCoordinatorTestEnv(t, integration.SyncBusinessResultSucceeded)
		var task model.IntegrationSyncTask
		if err := env.db.First(&task, env.task.Id).Error; err != nil {
			t.Fatal(err)
		}
		checkpoint := task.NextScheduledAt.UTC()
		if err := env.db.Model(&task).Updates(map[string]any{"checkpoint_at": &checkpoint, "initial_checkpoint_at": &checkpoint}).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
			t.Fatal(err)
		}
		batch := loadOnlySyncBatch(t, env.db)
		if batch.Status != model.IntegrationSyncBatchStatusSucceeded || batch.ExecutionCount != 0 || batch.PlannedSliceCount != 0 {
			t.Fatalf("empty batch=%+v", batch)
		}
	})

	t.Run("invalid window fails once and advances schedule", func(t *testing.T) {
		env := newSyncCoordinatorTestEnv(t, integration.SyncBusinessResultSucceeded)
		var task model.IntegrationSyncTask
		if err := env.db.First(&task, env.task.Id).Error; err != nil {
			t.Fatal(err)
		}
		checkpoint := task.NextScheduledAt.Add(time.Minute).UTC()
		if err := env.db.Model(&task).Updates(map[string]any{"checkpoint_at": &checkpoint, "initial_checkpoint_at": &checkpoint}).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := env.coordinator.RunOnce(context.Background(), 4, 4); err != nil {
			t.Fatal(err)
		}
		batch := loadOnlySyncBatch(t, env.db)
		if batch.Status != model.IntegrationSyncBatchStatusFailed || batch.ReasonCode != syncBatchReasonWindowInvalid || batch.ExecutionCount != 0 {
			t.Fatalf("invalid batch=%+v", batch)
		}
		var refreshed model.IntegrationSyncTask
		if err := env.db.First(&refreshed, task.Id).Error; err != nil || refreshed.NextScheduledAt == nil || !refreshed.NextScheduledAt.After(*task.NextScheduledAt) {
			t.Fatalf("task schedule=%+v err=%v", refreshed, err)
		}
	})
}

func newSyncCoordinatorTestEnv(t *testing.T, businessStatus string) syncCoordinatorTestEnv {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}, &model.IntegrationSyncTask{}, &model.IntegrationSyncBatch{}, &model.IntegrationExecution{}, &model.IntegrationLog{})
	primary := &database.PrimaryDB{DB: db}
	system := model.ExternalSystem{Basic: model.Basic{Id: 8101, State: true}, SystemCode: "sync_runtime", Name: "Sync Runtime", SystemType: model.ExternalSystemTypeHR, BaseURL: "https://example.com", OwnerIdentifier: "ops", OwnerName: "Ops", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	contract, _ := json.Marshal(integration.InterfaceInputContract{Version: 1, Parameters: []integration.InputParameterDefinition{
		{Code: "updated_from", Location: "query", DataType: "string", Required: true, MaxLength: 64},
		{Code: "updated_to", Location: "query", DataType: "string", Required: true, MaxLength: 64},
	}})
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: 8102, State: true}, ExternalSystemID: system.Id, InterfaceCode: "sync_employees", Name: "Sync Employees", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/employees", TimeoutSeconds: 30, ResponseLimit: 1024 * 1024, InputContract: contract, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	now := time.Now().UTC().Truncate(time.Minute)
	checkpoint := now.Add(-2 * time.Hour)
	due := now.Add(-time.Minute)
	plan := integration.SyncExecutionInputPlan{Version: 1, StaticInput: integration.ExecutionInputValues{}, WindowStartBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}}
	planRaw, _ := json.Marshal(plan)
	task := model.IntegrationSyncTask{Basic: model.Basic{Id: 8103, State: true}, TaskCode: "runtime_sync", TaskName: "Runtime Sync", Version: 1, Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id, ConsumerCode: "test_sync_consumer", ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleCron, CronExpression: "* * * * *", Timezone: "UTC", NextScheduledAt: &due, CheckpointMode: model.IntegrationSyncCheckpointTimestamp, InitialCheckpointAt: &checkpoint, CheckpointAt: &checkpoint, LookbackSeconds: 60, WindowSliceSeconds: 3600, InputPlan: datatypes.JSON(planRaw), Revision: 1}
	testutil.MustCreate(t, db, &system)
	testutil.MustCreate(t, db, &definition)
	testutil.MustCreate(t, db, &task)
	sf, _ := utils.NewSnowflake(1)
	executionRepo := impl.NewIntegrationExecutionRepositoryImpl(primary)
	auditWriter := &integrationExecutionAuditWriter{}
	executionService := NewIntegrationExecutionService(executionRepo, impl.NewIntegrationLogRepositoryImpl(primary), impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), impl.NewRetryPolicyRepositoryImpl(primary), sf, auditWriter)
	coordinator := NewIntegrationSyncCoordinator(impl.NewIntegrationSyncTaskRepositoryImpl(primary), impl.NewIntegrationSyncBatchRepositoryImpl(primary), executionRepo, impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), executionService, &syncBusinessResultStub{status: businessStatus}, sf)
	return syncCoordinatorTestEnv{db: db, coordinator: coordinator, task: task, audit: auditWriter}
}

func loadOnlySyncBatch(t *testing.T, db *gorm.DB) model.IntegrationSyncBatch {
	t.Helper()
	var value model.IntegrationSyncBatch
	if err := db.Order("id").First(&value).Error; err != nil {
		t.Fatal(err)
	}
	return value
}

func loadSyncExecution(t *testing.T, db *gorm.DB, batchID, sliceNo int) model.IntegrationExecution {
	t.Helper()
	var value model.IntegrationExecution
	if err := db.Where("sync_batch_id = ? AND sync_slice_no = ?", batchID, sliceNo).First(&value).Error; err != nil {
		t.Fatal(err)
	}
	return value
}

func markSyncExecution(t *testing.T, db *gorm.DB, id int, status string) {
	t.Helper()
	updates := map[string]any{"status": status, "revision": gorm.Expr("revision + 1")}
	if status == model.IntegrationExecutionStatusSucceeded || status == model.IntegrationExecutionStatusFailed {
		now := time.Now().UTC()
		updates["completed_at"] = &now
	}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		t.Fatal(err)
	}
}
