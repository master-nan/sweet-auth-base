package main

import (
	testutil "backend/internal/test"
	"backend/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestIntegrationSyncSchemaSQLiteIsIdempotent(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		if err := migrateIntegrationSyncSchema(db); err != nil {
			t.Fatalf("sync migration run %d: %v", run+1, err)
		}
	}
	if !db.Migrator().HasTable(&model.IntegrationSyncTask{}) || !db.Migrator().HasTable(&model.IntegrationSyncBatch{}) || !db.Migrator().HasColumn(&model.IntegrationExecution{}, "sync_batch_id") {
		t.Fatal("sync schema is incomplete")
	}
}

func TestIntegrationSyncSchemaPostgreSQLConstraints(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("integration_sync_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })
	db, err := gorm.Open(postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now})
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationConfigurationSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := migrateIntegrationRuntimeSchema(db); err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		if err := migrateIntegrationSyncSchema(db); err != nil {
			t.Fatalf("sync migration run %d: %v", run+1, err)
		}
	}

	system, definition := integrationRuntimeMigrationFixtures(t, db)
	task := validSyncTaskMigrationFixture(8801, "employee_sync", 1, system.Id, definition.Id)
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	duplicateVersion := validSyncTaskMigrationFixture(8802, task.TaskCode, 1, system.Id, definition.Id)
	if err := db.Create(&duplicateVersion).Error; err == nil {
		t.Fatal("expected task code/version unique violation")
	}
	enabledConflict := validSyncTaskMigrationFixture(8803, task.TaskCode, 2, system.Id, definition.Id)
	if err := db.Create(&enabledConflict).Error; err == nil {
		t.Fatal("expected one enabled task version")
	}
	badSchedule := validSyncTaskMigrationFixture(8804, "bad_schedule", 1, system.Id, definition.Id)
	badSchedule.Status = model.IntegrationSyncTaskStatusDraft
	badSchedule.State = false
	badSchedule.ScheduleType = model.IntegrationSyncScheduleCron
	badSchedule.CronExpression = ""
	if err := db.Create(&badSchedule).Error; err == nil {
		t.Fatal("expected schedule check violation")
	}
	badStatus := validSyncTaskMigrationFixture(8805, "bad_status", 1, system.Id, definition.Id)
	badStatus.Status = "paused"
	if err := db.Create(&badStatus).Error; err == nil {
		t.Fatal("expected task status check violation")
	}
	badCheckpoint := validSyncTaskMigrationFixture(8806, "bad_checkpoint", 1, system.Id, definition.Id)
	badCheckpoint.Status, badCheckpoint.State = model.IntegrationSyncTaskStatusDraft, false
	badCheckpoint.CheckpointMode, badCheckpoint.WindowSliceSeconds = model.IntegrationSyncCheckpointTimestamp, 59
	if err := db.Create(&badCheckpoint).Error; err == nil {
		t.Fatal("expected checkpoint range check violation")
	}
	badPlan := validSyncTaskMigrationFixture(8807, "bad_plan", 1, system.Id, definition.Id)
	badPlan.Status, badPlan.State = model.IntegrationSyncTaskStatusDraft, false
	badPlan.InputPlan = datatypes.JSON([]byte(`{"version":2,"static_input":{}}`))
	if err := db.Create(&badPlan).Error; err == nil {
		t.Fatal("expected input plan version check violation")
	}
	badRevision := validSyncTaskMigrationFixture(8808, "bad_revision", 1, system.Id, definition.Id)
	badRevision.Status, badRevision.State = model.IntegrationSyncTaskStatusDraft, false
	if err := db.Create(&badRevision).Error; err != nil {
		t.Fatalf("create revision fixture: %v", err)
	}
	if err := db.Model(&model.IntegrationSyncTask{}).Where("id = ?", badRevision.Id).Update("revision", 0).Error; err == nil {
		t.Fatal("expected task revision check violation")
	}

	batch := validSyncBatchMigrationFixture(8901, task)
	if err := db.Create(&batch).Error; err != nil {
		t.Fatalf("create batch: %v", err)
	}
	activeConflict := validSyncBatchMigrationFixture(8902, task)
	activeConflict.BatchNo = "SYNC-8902"
	activeConflict.TriggerKey = "manual:8902"
	if err := db.Create(&activeConflict).Error; err == nil {
		t.Fatal("expected active batch unique violation")
	}
	duplicateBatchNo := validSyncBatchMigrationFixture(8904, task)
	duplicateBatchNo.Status, duplicateBatchNo.TriggerKey = model.IntegrationSyncBatchStatusSucceeded, "manual:8904"
	if err := db.Create(&duplicateBatchNo).Error; err == nil {
		t.Fatal("expected batch number unique violation")
	}
	duplicateTriggerKey := validSyncBatchMigrationFixture(8905, task)
	duplicateTriggerKey.Status, duplicateTriggerKey.BatchNo = model.IntegrationSyncBatchStatusSucceeded, "SYNC-8905"
	if err := db.Create(&duplicateTriggerKey).Error; err == nil {
		t.Fatal("expected trigger key unique violation")
	}
	scheduledFor := time.Date(2026, 8, 10, 9, 30, 0, 0, time.UTC)
	scheduled := validSyncBatchMigrationFixture(8906, task)
	scheduled.Status, scheduled.BatchNo, scheduled.TriggerKey = model.IntegrationSyncBatchStatusSucceeded, "SYNC-8906", "schedule:8906"
	scheduled.TriggerType, scheduled.ScheduledFor = model.IntegrationSyncTriggerScheduled, &scheduledFor
	if err := db.Create(&scheduled).Error; err != nil {
		t.Fatalf("create scheduled batch: %v", err)
	}
	scheduledConflict := validSyncBatchMigrationFixture(8907, task)
	scheduledConflict.Status, scheduledConflict.BatchNo, scheduledConflict.TriggerKey = model.IntegrationSyncBatchStatusSucceeded, "SYNC-8907", "schedule:8907"
	scheduledConflict.TriggerType, scheduledConflict.ScheduledFor = model.IntegrationSyncTriggerScheduled, &scheduledFor
	if err := db.Create(&scheduledConflict).Error; err == nil {
		t.Fatal("expected task/scheduled time unique violation")
	}
	badBatchCounts := validSyncBatchMigrationFixture(8908, task)
	badBatchCounts.Status, badBatchCounts.BatchNo, badBatchCounts.TriggerKey, badBatchCounts.ExecutionCount = model.IntegrationSyncBatchStatusSucceeded, "SYNC-8908", "manual:8908", -1
	if err := db.Create(&badBatchCounts).Error; err == nil {
		t.Fatal("expected batch count check violation")
	}
	badFK := validSyncBatchMigrationFixture(8903, task)
	badFK.BatchNo = "SYNC-8903"
	badFK.TriggerKey = "manual:8903"
	badFK.SyncTaskID = 999999
	badFK.TaskCode = "other"
	if err := db.Create(&badFK).Error; err == nil {
		t.Fatal("expected sync task foreign key violation")
	}

	execution := validIntegrationExecutionFixture(8951, "INT-SYNC-8951", system, definition, "sync-8951")
	slice := 1
	consumerVersion := 1
	execution.SyncBatchID, execution.SyncSliceNo, execution.SyncConsumerCode, execution.SyncConsumerVersion = &batch.Id, &slice, "test_sync_consumer", &consumerVersion
	if err := db.Create(&execution).Error; err != nil {
		t.Fatalf("create sync execution: %v", err)
	}
	duplicateSlice := validIntegrationExecutionFixture(8952, "INT-SYNC-8952", system, definition, "sync-8952")
	duplicateSlice.SyncBatchID, duplicateSlice.SyncSliceNo, duplicateSlice.SyncConsumerCode, duplicateSlice.SyncConsumerVersion = &batch.Id, &slice, "test_sync_consumer", &consumerVersion
	if err := db.Create(&duplicateSlice).Error; err == nil {
		t.Fatal("expected batch/slice unique violation")
	}
	invalidGroup := validIntegrationExecutionFixture(8953, "INT-SYNC-8953", system, definition, "sync-8953")
	invalidGroup.SyncBatchID = &batch.Id
	if err := db.Create(&invalidGroup).Error; err == nil {
		t.Fatal("expected sync source group check violation")
	}
	plainExecution := validIntegrationExecutionFixture(8954, "INT-SYNC-8954", system, definition, "sync-8954")
	if err := db.Create(&plainExecution).Error; err != nil {
		t.Fatalf("ordinary execution must allow empty sync source: %v", err)
	}

	for _, index := range []string{"uni_integration_sync_task_enabled", "uni_integration_sync_batch_active", "uni_integration_sync_batch_scheduled", "uni_integration_execution_sync_slice"} {
		var count int64
		if err := db.Raw(`SELECT COUNT(*) FROM pg_indexes WHERE schemaname = ? AND indexname = ?`, schemaName, index).Scan(&count).Error; err != nil || count != 1 {
			t.Fatalf("index %s count=%d err=%v", index, count, err)
		}
	}
}

func validSyncTaskMigrationFixture(id int, code string, version, systemID, interfaceID int) model.IntegrationSyncTask {
	return model.IntegrationSyncTask{Basic: model.Basic{Id: id, State: true}, TaskCode: code, TaskName: "Task", Version: version, Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: systemID, InterfaceDefinitionID: interfaceID, ConsumerCode: "test_sync_consumer", ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleNone, Timezone: "UTC", CheckpointMode: model.IntegrationSyncCheckpointNone, InputPlan: datatypes.JSON([]byte(`{"version":1,"static_input":{}}`)), Revision: 1}
}

func validSyncBatchMigrationFixture(id int, task model.IntegrationSyncTask) model.IntegrationSyncBatch {
	return model.IntegrationSyncBatch{Basic: model.Basic{Id: id, State: true}, BatchNo: "SYNC-8901", SyncTaskID: task.Id, TaskCode: task.TaskCode, TaskName: task.TaskName, TaskVersion: task.Version, SystemCode: "system", InterfaceCode: "interface", InterfaceVersion: 1, ConsumerCode: task.ConsumerCode, ConsumerVersion: task.ConsumerVersion, TriggerType: model.IntegrationSyncTriggerManual, TriggerKey: "manual:8901", Status: model.IntegrationSyncBatchStatusCreated, CheckpointMode: model.IntegrationSyncCheckpointNone, Revision: 1}
}
