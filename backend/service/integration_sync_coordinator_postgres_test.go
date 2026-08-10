package service

import (
	"backend/internal/database"
	"backend/internal/integration"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestIntegrationSyncPostgreSQLSkipLockedCreatesOneBatch(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	first := seedPostgreSQLSyncCoordinator(t, db, "pg_schedule_once")
	second := newPostgreSQLSyncCoordinator(t, db, integration.SyncBusinessResultSucceeded)
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, coordinator := range []*IntegrationSyncCoordinator{first, second} {
		group.Add(1)
		go func(value *IntegrationSyncCoordinator) {
			defer group.Done()
			<-start
			_, err := value.ScheduleDueTasks(context.Background(), 1)
			results <- err
		}(coordinator)
	}
	close(start)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("schedule due task: %v", err)
		}
	}
	var count int64
	if err := db.Model(&model.IntegrationSyncBatch{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("batch count=%d err=%v", count, err)
	}
	var task model.IntegrationSyncTask
	if err := db.Where("task_code = ?", "pg_schedule_once").First(&task).Error; err != nil || task.LastScheduledAt == nil || task.NextScheduledAt == nil || !task.NextScheduledAt.After(*task.LastScheduledAt) {
		t.Fatalf("scheduled task=%+v err=%v", task, err)
	}
}

func TestIntegrationSyncPostgreSQLRunnerSequentialE2E(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	coordinator := seedPostgreSQLSyncCoordinator(t, db, "pg_runner_e2e")
	runner, err := integration.NewIntegrationSyncRunner(coordinator, integration.SyncRunnerConfig{
		Enabled: true, RunnerID: "pg-sync-runner", PollInterval: time.Second,
		ScheduleBatchSize: 4, CoordinateBatchSize: 4, ShutdownTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := runner.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runner.Stop(context.Background()) })

	first := waitForPostgreSQLSyncExecution(t, db, 1, 5*time.Second)
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	markSyncExecution(t, db, first.Id, model.IntegrationExecutionStatusSucceeded)
	runner, err = integration.NewIntegrationSyncRunner(coordinator, integration.SyncRunnerConfig{
		Enabled: true, RunnerID: "pg-sync-runner-restarted", PollInterval: time.Second,
		ScheduleBatchSize: 4, CoordinateBatchSize: 4, ShutdownTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(ctx); err != nil {
		t.Fatal(err)
	}
	second := waitForPostgreSQLSyncExecution(t, db, 2, 5*time.Second)
	if second.SyncWindowStart == nil || first.SyncWindowEnd == nil || !second.SyncWindowStart.Equal(*first.SyncWindowEnd) {
		t.Fatalf("non-contiguous slices first=%+v second=%+v", first, second)
	}
	markSyncExecution(t, db, second.Id, model.IntegrationExecutionStatusSucceeded)
	waitForPostgreSQLSyncBatchStatus(t, db, model.IntegrationSyncBatchStatusSucceeded, 5*time.Second)
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	var batch model.IntegrationSyncBatch
	if err := db.First(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if batch.ExecutionCount != 2 || batch.TechnicalSuccessCount != 2 || batch.CheckpointAfter == nil || !batch.CheckpointAfter.Equal(*batch.WindowEnd) {
		t.Fatalf("final batch=%+v", batch)
	}
	var task model.IntegrationSyncTask
	if err := db.First(&task, batch.SyncTaskID).Error; err != nil || task.CheckpointAt == nil || !task.CheckpointAt.Equal(*batch.WindowEnd) {
		t.Fatalf("final task=%+v err=%v", task, err)
	}
}

func openSyncCoordinatorPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schemaName := fmt.Sprintf("integration_sync_coordinator_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf("CREATE SCHEMA %q", schemaName)).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schemaName)).Error })
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	query.Set("TimeZone", "UTC")
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalSystem{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}, &model.IntegrationSyncTask{}, &model.IntegrationSyncBatch{}, &model.IntegrationExecution{}, &model.IntegrationLog{}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE UNIQUE INDEX uni_integration_sync_batch_active ON integration_sync_batch (task_code) WHERE status IN ('created','running') AND gmt_delete IS NULL`,
		`CREATE UNIQUE INDEX uni_integration_sync_batch_scheduled ON integration_sync_batch (task_code, scheduled_for) WHERE trigger_type = 'scheduled' AND gmt_delete IS NULL`,
		`CREATE UNIQUE INDEX uni_integration_execution_sync_slice ON integration_execution (sync_batch_id, sync_slice_no) WHERE sync_batch_id IS NOT NULL AND gmt_delete IS NULL`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func seedPostgreSQLSyncCoordinator(t *testing.T, db *gorm.DB, taskCode string) *IntegrationSyncCoordinator {
	t.Helper()
	system := model.ExternalSystem{Basic: model.Basic{Id: nextSyncTestID(), State: true}, SystemCode: taskCode, Name: "PG Sync", SystemType: model.ExternalSystemTypeHR, BaseURL: "https://example.com", OwnerIdentifier: "ops", OwnerName: "Ops", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	contract, _ := json.Marshal(integration.InterfaceInputContract{Version: 1, Parameters: []integration.InputParameterDefinition{
		{Code: "updated_from", Location: "query", DataType: "string", Required: true, MaxLength: 64},
		{Code: "updated_to", Location: "query", DataType: "string", Required: true, MaxLength: 64},
	}})
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id, InterfaceCode: taskCode, Name: "PG Sync", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/employees", TimeoutSeconds: 30, ResponseLimit: 1024 * 1024, InputContract: contract, IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	var databaseNow time.Time
	if err := db.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&databaseNow).Error; err != nil {
		t.Fatal(err)
	}
	databaseNow = databaseNow.UTC()
	checkpoint := databaseNow.Add(-2 * time.Hour)
	due := databaseNow.Add(-time.Minute)
	planRaw, _ := json.Marshal(integration.SyncExecutionInputPlan{Version: 1, StaticInput: integration.ExecutionInputValues{}, WindowStartBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}})
	task := model.IntegrationSyncTask{Basic: model.Basic{Id: nextSyncTestID(), State: true}, TaskCode: taskCode, TaskName: "PG Sync", Version: 1, Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id, ConsumerCode: "test_sync_consumer", ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleCron, CronExpression: "* * * * *", Timezone: "UTC", NextScheduledAt: &due, CheckpointMode: model.IntegrationSyncCheckpointTimestamp, InitialCheckpointAt: &checkpoint, CheckpointAt: &checkpoint, LookbackSeconds: 60, WindowSliceSeconds: 3600, InputPlan: datatypes.JSON(planRaw), Revision: 1}
	if err := db.Create(&system).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&definition).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	return newPostgreSQLSyncCoordinator(t, db, integration.SyncBusinessResultSucceeded)
}

func newPostgreSQLSyncCoordinator(t *testing.T, db *gorm.DB, businessStatus string) *IntegrationSyncCoordinator {
	t.Helper()
	primary := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(3)
	if err != nil {
		t.Fatal(err)
	}
	executionRepo := impl.NewIntegrationExecutionRepositoryImpl(primary)
	application := NewIntegrationExecutionService(executionRepo, impl.NewIntegrationLogRepositoryImpl(primary), impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), impl.NewRetryPolicyRepositoryImpl(primary), sf, &integrationExecutionAuditWriter{})
	return NewIntegrationSyncCoordinator(impl.NewIntegrationSyncTaskRepositoryImpl(primary), impl.NewIntegrationSyncBatchRepositoryImpl(primary), executionRepo, impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), application, &syncBusinessResultStub{status: businessStatus}, sf)
}

func waitForPostgreSQLSyncExecution(t *testing.T, db *gorm.DB, sliceNo int, timeout time.Duration) model.IntegrationExecution {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var value model.IntegrationExecution
		if err := db.Where("sync_slice_no = ?", sliceNo).First(&value).Error; err == nil {
			return value
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("slice %d was not created", sliceNo)
	return model.IntegrationExecution{}
}

func waitForPostgreSQLSyncBatchStatus(t *testing.T, db *gorm.DB, status string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var value model.IntegrationSyncBatch
		if err := db.First(&value).Error; err == nil && value.Status == status {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("batch did not reach %s", status)
}

var syncTestIDMu sync.Mutex
var syncTestIDValue = 910000

func nextSyncTestID() int {
	syncTestIDMu.Lock()
	defer syncTestIDMu.Unlock()
	syncTestIDValue++
	return syncTestIDValue
}
