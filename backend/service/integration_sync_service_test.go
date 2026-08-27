package service

import (
	"backend/config"
	"backend/dto/request"
	"backend/internal/audit"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/integration"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSyncTaskServiceLifecycleVersionCheckpointAndActiveBatch(t *testing.T) {
	svc, db, writer := newSyncTaskTestSubject(t)
	initial := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	created, err := svc.CreateSyncTask(context.Background(), syncTaskCreateRequest(initial))
	if err != nil || created.Status != model.IntegrationSyncTaskStatusDraft || created.Version != 1 {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	name := "人员增量同步 v1"
	updated, err := svc.UpdateDraftSyncTask(context.Background(), created.ID, request.SyncTaskUpdateReq{TaskName: &name, Revision: created.Revision})
	if err != nil || updated.TaskName != name || updated.Revision != 2 {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	enabled, err := svc.EnableSyncTask(context.Background(), updated.ID, updated.Revision)
	if err != nil || enabled.Status != model.IntegrationSyncTaskStatusEnabled || enabled.CheckpointAt == nil || !enabled.CheckpointAt.Equal(initial) {
		t.Fatalf("enable=%+v err=%v", enabled, err)
	}
	if _, err := svc.UpdateDraftSyncTask(context.Background(), enabled.ID, request.SyncTaskUpdateReq{Revision: enabled.Revision}); !errors.Is(err, myerrors.ErrSyncTaskFieldImmutable) {
		t.Fatalf("enabled edit error=%v", err)
	}
	v2, err := svc.CreateSyncTaskVersion(context.Background(), enabled.ID, enabled.Revision)
	if err != nil || v2.Version != 2 || v2.CheckpointAt != nil || v2.Status != model.IntegrationSyncTaskStatusDraft {
		t.Fatalf("version=%+v err=%v", v2, err)
	}
	latest := initial.Add(48 * time.Hour)
	if err := db.Model(&model.IntegrationSyncTask{}).Where("id = ?", enabled.ID).Update("checkpoint_at", latest).Error; err != nil {
		t.Fatal(err)
	}
	enabledV2, err := svc.EnableSyncTask(context.Background(), v2.ID, v2.Revision)
	if err != nil || enabledV2.CheckpointAt == nil || !enabledV2.CheckpointAt.Equal(latest) {
		t.Fatalf("enable v2=%+v err=%v", enabledV2, err)
	}
	var old model.IntegrationSyncTask
	if err := db.First(&old, enabled.ID).Error; err != nil || old.Status != model.IntegrationSyncTaskStatusDisabled {
		t.Fatalf("old version=%+v err=%v", old, err)
	}
	batch := syncBatchFixture(9901, enabledV2.ID, enabledV2.TaskCode)
	testutil.MustCreate(t, db, &batch)
	if _, err := svc.DisableSyncTask(context.Background(), enabledV2.ID, enabledV2.Revision); !errors.Is(err, myerrors.ErrSyncTaskActiveBatch) {
		t.Fatalf("active batch disable=%v", err)
	}
	if len(writer.records) != 5 {
		t.Fatalf("audit records=%d want 5", len(writer.records))
	}
}

func TestSyncTaskServiceValidationEditDTOAndBatchQuery(t *testing.T) {
	svc, db, _ := newSyncTaskTestSubject(t)
	initial := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	bad := syncTaskCreateRequest(initial)
	bad.CronExpression = "* * * * * *"
	bad.ScheduleType = "cron"
	if _, err := svc.CreateSyncTask(context.Background(), bad); !errors.Is(err, myerrors.ErrSyncScheduleInvalid) {
		t.Fatalf("six-field cron=%v", err)
	}
	bad = syncTaskCreateRequest(initial)
	bad.Timezone = "Mars/Olympus"
	if _, err := svc.CreateSyncTask(context.Background(), bad); !errors.Is(err, myerrors.ErrSyncTimezoneInvalid) {
		t.Fatalf("timezone=%v", err)
	}
	bad = syncTaskCreateRequest(initial)
	bad.ConsumerCode = "missing"
	if _, err := svc.CreateSyncTask(context.Background(), bad); !errors.Is(err, myerrors.ErrSyncConsumerNotRegistered) {
		t.Fatalf("consumer=%v", err)
	}
	created, err := svc.CreateSyncTask(context.Background(), syncTaskCreateRequest(initial))
	if err != nil {
		t.Fatal(err)
	}
	edit, err := svc.GetSyncTaskForEdit(context.Background(), created.ID)
	if err != nil || len(edit.InputPlan) == 0 {
		t.Fatalf("edit=%+v err=%v", edit, err)
	}
	payload, _ := json.Marshal(created)
	for _, forbidden := range []string{"input_plan\"", "static_input", "gmt_delete", "create_user", "next_scheduled_at\":null"} {
		if stringContainsFold(string(payload), forbidden) {
			t.Fatalf("detail leaked %q: %s", forbidden, payload)
		}
	}
	batchSvc := NewSyncBatchService(impl.NewIntegrationSyncBatchRepositoryImpl(&database.PrimaryDB{DB: db}))
	batch := syncBatchFixture(9911, created.ID, created.TaskCode)
	batch.Status = model.IntegrationSyncBatchStatusSucceeded
	testutil.MustCreate(t, db, &batch)
	page, err := batchSvc.PageSyncBatch(context.Background(), request.SyncBatchQueryReq{Page: 1, Num: 10, Status: model.IntegrationSyncBatchStatusSucceeded})
	if err != nil || page.Total != 1 || len(page.Data) != 1 || page.Data[0].BatchNo != batch.BatchNo {
		t.Fatalf("batch page=%+v err=%v", page, err)
	}
}

func TestSyncTaskServiceRevisionDisableAndImmutableVersionRules(t *testing.T) {
	svc, _, _ := newSyncTaskTestSubject(t)
	initial := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	created, err := svc.CreateSyncTask(context.Background(), syncTaskCreateRequest(initial))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateDraftSyncTask(context.Background(), created.ID, request.SyncTaskUpdateReq{Revision: created.Revision + 1}); !errors.Is(err, myerrors.ErrSyncTaskRevisionConflict) {
		t.Fatalf("draft revision conflict=%v", err)
	}
	enabled, err := svc.EnableSyncTask(context.Background(), created.ID, created.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DisableSyncTask(context.Background(), enabled.ID, enabled.Revision+1); !errors.Is(err, myerrors.ErrSyncTaskRevisionConflict) {
		t.Fatalf("disable revision conflict=%v", err)
	}
	disabled, err := svc.DisableSyncTask(context.Background(), enabled.ID, enabled.Revision)
	if err != nil || disabled.Status != model.IntegrationSyncTaskStatusDisabled {
		t.Fatalf("disable=%+v err=%v", disabled, err)
	}
	if _, err := svc.UpdateDraftSyncTask(context.Background(), disabled.ID, request.SyncTaskUpdateReq{Revision: disabled.Revision}); !errors.Is(err, myerrors.ErrSyncTaskFieldImmutable) {
		t.Fatalf("disabled edit=%v", err)
	}
	reenabled, err := svc.EnableSyncTask(context.Background(), disabled.ID, disabled.Revision)
	if err != nil || reenabled.Status != model.IntegrationSyncTaskStatusEnabled || reenabled.CheckpointAt == nil || !reenabled.CheckpointAt.Equal(initial) {
		t.Fatalf("reenable=%+v err=%v", reenabled, err)
	}
}

func TestSyncTaskServiceNoneCheckpointClearsWindowConfiguration(t *testing.T) {
	svc, _, _ := newSyncTaskTestSubject(t)
	req := syncTaskCreateRequest(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	req.TaskCode = "employee_full_sync"
	req.CheckpointMode = model.IntegrationSyncCheckpointNone
	req.InputPlan.WindowStartBinding, req.InputPlan.WindowEndBinding = nil, nil
	req.InputPlan.StaticInput.QueryParams = map[string][]string{"updated_from": {"2026-08-01T00:00:00Z"}, "updated_to": {"2026-08-02T00:00:00Z"}}
	created, err := svc.CreateSyncTask(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if created.InitialCheckpointAt != nil || created.CheckpointAt != nil || created.LookbackSeconds != 0 || created.WindowSliceSeconds != 0 {
		t.Fatalf("none checkpoint was not normalized: %+v", created)
	}
}

func TestSyncTaskServiceManualRunUsesDatabaseWindowAndAuditSubject(t *testing.T) {
	svc, db, writer := newSyncTaskTestSubject(t)
	initial := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	created, err := svc.CreateSyncTask(context.Background(), syncTaskCreateRequest(initial))
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := svc.EnableSyncTask(context.Background(), created.ID, created.Revision)
	if err != nil {
		t.Fatal(err)
	}
	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(88, "sync-admin"))
	batch, err := svc.RunSyncTask(ctx, enabled.ID, enabled.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if batch.TriggerType != model.IntegrationSyncTriggerManual || batch.Status != model.IntegrationSyncBatchStatusCreated || batch.WindowStart == nil || batch.WindowEnd == nil || batch.CheckpointBefore == nil || !batch.WindowStart.Equal(initial) || !batch.CheckpointBefore.Equal(initial) || !batch.WindowEnd.After(initial) {
		t.Fatalf("manual batch=%+v", batch)
	}
	var stored model.IntegrationSyncBatch
	if err := db.First(&stored, batch.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ScheduledFor != nil || stored.TriggeredByUserID == nil || *stored.TriggeredByUserID != 88 || stored.TriggeredByUserName != "sync-admin" || stored.TriggerKey == "" || stored.TaskRevision != enabled.Revision {
		t.Fatalf("stored manual batch=%+v", stored)
	}
	if _, err := svc.RunSyncTask(ctx, enabled.ID, enabled.Revision); !errors.Is(err, myerrors.ErrSyncTaskActiveBatch) {
		t.Fatalf("second manual run=%v", err)
	}
	if len(writer.records) != 3 || writer.records[len(writer.records)-1].Action != syncTaskAuditRun {
		t.Fatalf("audit records=%+v", writer.records)
	}
}

func TestNextSyncScheduleHandlesDSTTransitions(t *testing.T) {
	springFrom := time.Date(2026, 3, 8, 6, 0, 0, 0, time.UTC)
	spring, err := nextSyncSchedule("30 2 * * *", "America/New_York", springFrom)
	if err != nil || spring == nil || !spring.After(springFrom) {
		t.Fatalf("spring schedule=%v err=%v", spring, err)
	}
	location, _ := time.LoadLocation("America/New_York")
	if local := spring.In(location); local.Hour() != 2 || local.Minute() != 30 {
		t.Fatalf("spring schedule selected nonexistent/incorrect local time: %v", local)
	}
	fallFrom := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)
	fall, err := nextSyncSchedule("30 1 * * *", "America/New_York", fallFrom)
	if err != nil || fall == nil || !fall.After(fallFrom) {
		t.Fatalf("fall schedule=%v err=%v", fall, err)
	}
	if local := fall.In(location); local.Hour() != 1 || local.Minute() != 30 {
		t.Fatalf("fall local time=%v", local)
	}
}

func newSyncTaskTestSubject(t *testing.T) (*SyncTaskService, *gorm.DB, *externalSystemAuditWriter) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}, &model.IntegrationSyncTask{}, &model.IntegrationSyncBatch{})
	primary := &database.PrimaryDB{DB: db}
	system := model.ExternalSystem{Basic: model.Basic{Id: 9701, State: true}, SystemCode: "sync_hr", Name: "Sync HR", SystemType: model.ExternalSystemTypeHR, BaseURL: "https://example.com", OwnerIdentifier: "ops", OwnerName: "Ops", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	contract, _ := json.Marshal(integration.InterfaceInputContract{Version: 1, Parameters: []integration.InputParameterDefinition{{Code: "updated_from", Location: "query", DataType: "string", Required: true, MaxLength: 64}, {Code: "updated_to", Location: "query", DataType: "string", Required: true, MaxLength: 64}, {Code: "X-Correlation-ID", Location: "header", DataType: "string", MaxLength: 64}}})
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: 9702, State: true}, ExternalSystemID: system.Id, InterfaceCode: "employees", Name: "Employees", Version: 3, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/employees", TimeoutSeconds: 30, ResponseLimit: 1024 * 1024, InputContract: contract, IdempotencyMode: model.InterfaceIdempotencyModeNone, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	testutil.MustCreate(t, db, &system)
	testutil.MustCreate(t, db, &definition)
	registry, err := integration.NewStaticSyncConsumerRegistry(integration.SyncConsumerRegistration{Metadata: integration.SyncConsumerMetadata{Code: "test_sync_consumer", Version: 1, Name: "Test Consumer", Status: integration.SyncConsumerStatusEnabled, ContentTypes: []string{"application/json"}, MaxResponseBytes: 2 * 1024 * 1024, MaxDuration: 10 * time.Second, CheckpointModes: []string{"none", "timestamp"}}, Consumer: integration.SyncResultConsumerFunc(func(context.Context, integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
		return integration.NewSyncConsumptionResult(true, "", 1, 0, "")
	})})
	if err != nil {
		t.Fatal(err)
	}
	sf, _ := utils.NewSnowflake(1)
	writer := &externalSystemAuditWriter{}
	svc := NewSyncTaskService(impl.NewIntegrationSyncTaskRepositoryImpl(primary), impl.NewIntegrationSyncBatchRepositoryImpl(primary), impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), impl.NewRetryPolicyRepositoryImpl(primary), registry, sf, writer, &config.Server{})
	return svc, db, writer
}

func syncTaskCreateRequest(initial time.Time) request.SyncTaskCreateReq {
	return request.SyncTaskCreateReq{TaskCode: "employee_sync", TaskName: "人员增量同步", ExternalSystemID: 9701, InterfaceDefinitionID: 9702, ConsumerCode: "test_sync_consumer", ConsumerVersion: 1, ScheduleType: "none", Timezone: "UTC", CheckpointMode: "timestamp", InitialCheckpointAt: &initial, LookbackSeconds: 60, WindowSliceSeconds: 3600, InputPlan: request.SyncExecutionInputPlanReq{Version: 1, StaticInput: request.SyncStaticInputReq{Headers: map[string][]string{"X-Correlation-ID": {"sweet"}}}, WindowStartBinding: &request.SyncWindowBindingReq{Location: "query", Code: "updated_from", Format: "rfc3339"}, WindowEndBinding: &request.SyncWindowBindingReq{Location: "query", Code: "updated_to", Format: "rfc3339"}}}
}

func syncBatchFixture(id, taskID int, taskCode string) model.IntegrationSyncBatch {
	return model.IntegrationSyncBatch{Basic: model.Basic{Id: id, State: true}, BatchNo: "SYNC-" + taskCode, SyncTaskID: taskID, TaskCode: taskCode, TaskName: "Task", TaskVersion: 2, SystemCode: "sync_hr", InterfaceCode: "employees", InterfaceVersion: 3, ConsumerCode: "test_sync_consumer", ConsumerVersion: 1, TriggerType: model.IntegrationSyncTriggerManual, TriggerKey: "manual:" + taskCode, Status: model.IntegrationSyncBatchStatusCreated, CheckpointMode: model.IntegrationSyncCheckpointTimestamp, CheckpointBefore: timePointer(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)), LookbackSeconds: 60, WindowSliceSeconds: 3600, Revision: 1}
}

func timePointer(value time.Time) *time.Time { return &value }
func stringContainsFold(value, expected string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(expected))
}
