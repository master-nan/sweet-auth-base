package integration

import (
	"backend/internal/database"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestIntegrationExecutionEngineClaimRunAndConverge(t *testing.T) {
	engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusOK)
	defer closeServer()

	claimed, err := engine.ClaimReadyExecutions(context.Background())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim executions = %+v err=%v", claimed, err)
	}
	if claimed[0].Execution.Status != model.IntegrationExecutionStatusRunning || claimed[0].Attempt.AttemptNo != 1 || claimed[0].Attempt.WorkerID != "runtime-worker-1" {
		t.Fatalf("unexpected claim: %+v", claimed[0])
	}
	if claimed[0].Execution.LeaseExpiresAt == nil || claimed[0].Execution.LeaseExpiresAt.Sub(claimed[0].Attempt.StartedAt) < IntegrationMinimumLeaseDuration {
		t.Fatalf("claim lease does not include runtime safety margin: %+v", claimed[0].Execution.LeaseExpiresAt)
	}
	result, err := engine.RunExecution(context.Background(), claimed[0])
	if err != nil || !result.Succeeded || result.HTTPStatus == nil || *result.HTTPStatus != http.StatusOK {
		t.Fatalf("run result = %+v err=%v", result, err)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, execution.Id).Error; err != nil {
		t.Fatalf("load execution: %v", err)
	}
	if stored.Status != model.IntegrationExecutionStatusSucceeded || stored.LeaseOwner != "" || stored.LeaseExpiresAt != nil || stored.CurrentAttempt != 1 {
		t.Fatalf("stored execution not converged: %+v", stored)
	}
	var attempt model.IntegrationLog
	if err := db.Where("execution_id = ?", execution.Id).First(&attempt).Error; err != nil {
		t.Fatalf("load attempt: %v", err)
	}
	if attempt.Status != model.IntegrationLogStatusSucceeded || attempt.WorkerID != "runtime-worker-1" || attempt.ResultHash == "" || attempt.ResponseContentType != "application/json" ||
		attempt.CredentialCode != "runtime_token" || attempt.CredentialVersion != "v1" || len(attempt.CredentialFingerprintSummary) != 12 {
		t.Fatalf("stored attempt not completed safely: %+v", attempt)
	}
}

func TestIntegrationExecutionEngineRetryEligibilityAndLeaseRecovery(t *testing.T) {
	for _, statusCode := range []int{http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			engine, db, execution, closeServer := newExecutionEngineFixture(t, statusCode)
			defer closeServer()
			claimed, err := engine.ClaimReadyExecutions(context.Background())
			if err != nil || len(claimed) != 1 {
				t.Fatalf("claim = %+v err=%v", claimed, err)
			}
			result, err := engine.RunExecution(context.Background(), claimed[0])
			if err != nil || result.ExecutionStatus != model.IntegrationExecutionStatusRetryWaiting || result.Certainty != model.IntegrationResultCertaintyConfirmed {
				t.Fatalf("retry result = %+v err=%v", result, err)
			}
			var stored model.IntegrationExecution
			if err := db.First(&stored, execution.Id).Error; err != nil || stored.Status != model.IntegrationExecutionStatusRetryWaiting ||
				stored.NextRunAt == nil || stored.CompletedAt != nil || stored.RetryReasonCode != RetryReasonAllowed {
				t.Fatalf("retry waiting execution = %+v err=%v", stored, err)
			}
			var attempt model.IntegrationLog
			if err := db.Where("execution_id = ?", execution.Id).First(&attempt).Error; err != nil || !attempt.Retryable ||
				attempt.RetryScheduledAt == nil || attempt.RetryDelayMs != 1000 || attempt.RetryAfterSource != RetryAfterSourceLocal {
				t.Fatalf("retry diagnostics = %+v err=%v", attempt, err)
			}
		})
	}
	t.Run("500 is not a V1 retry status", func(t *testing.T) {
		engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusInternalServerError)
		defer closeServer()
		claimed, err := engine.ClaimReadyExecutions(context.Background())
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim = %+v err=%v", claimed, err)
		}
		result, err := engine.RunExecution(context.Background(), claimed[0])
		if err != nil || result.ExecutionStatus != model.IntegrationExecutionStatusFailed || result.RetryReasonCode != RetryReasonHTTPStatusNotAllowed {
			t.Fatalf("500 decision = %+v err=%v", result, err)
		}
		var stored model.IntegrationExecution
		if err := db.First(&stored, execution.Id).Error; err != nil || stored.Status != model.IntegrationExecutionStatusFailed || stored.NextRunAt != nil {
			t.Fatalf("500 execution = %+v err=%v", stored, err)
		}
	})
	t.Run("Retry-After is persisted as a safe schedule summary", func(t *testing.T) {
		engine, db, execution, closeServer := newExecutionEngineFixtureWithHandler(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			writer.Header().Set("Retry-After", "5")
			writer.WriteHeader(http.StatusServiceUnavailable)
			_, _ = writer.Write([]byte(`{"error":"busy"}`))
		}))
		defer closeServer()
		claimed, err := engine.ClaimReadyExecutions(context.Background())
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim=%+v err=%v", claimed, err)
		}
		result, err := engine.RunExecution(context.Background(), claimed[0])
		if err != nil || result.RetryDelay != 5*time.Second || result.RetryAfterSource != RetryAfterSourceHTTPDelta {
			t.Fatalf("retry-after result=%+v err=%v", result, err)
		}
		var attempt model.IntegrationLog
		if err := db.Where("execution_id = ?", execution.Id).First(&attempt).Error; err != nil ||
			attempt.RetryDelayMs != 5000 || attempt.RetryAfterSource != RetryAfterSourceHTTPDelta {
			t.Fatalf("retry-after attempt=%+v err=%v", attempt, err)
		}
	})
	t.Run("expired lease becomes unknown failed without HTTP retry", func(t *testing.T) {
		engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusOK)
		defer closeServer()
		claimed, err := engine.ClaimReadyExecutions(context.Background())
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim = %+v err=%v", claimed, err)
		}
		expired := time.Now().Add(-time.Second)
		if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).Update("lease_expires_at", expired).Error; err != nil {
			t.Fatalf("expire lease: %v", err)
		}
		recovered, err := engine.RecoverExpiredLease(context.Background(), 4)
		if err != nil || recovered != 1 {
			t.Fatalf("recover = %d err=%v", recovered, err)
		}
		var stored model.IntegrationExecution
		var attempt model.IntegrationLog
		if err := db.First(&stored, execution.Id).Error; err != nil {
			t.Fatalf("load recovered execution: %v", err)
		}
		if err := db.Where("execution_id = ?", execution.Id).First(&attempt).Error; err != nil {
			t.Fatalf("load recovered attempt: %v", err)
		}
		if stored.Status != model.IntegrationExecutionStatusFailed || attempt.Status != model.IntegrationLogStatusFailed || attempt.ResultCertainty != model.IntegrationResultCertaintyUnknown {
			t.Fatalf("recovery did not keep unknown failure: execution=%+v attempt=%+v", stored, attempt)
		}
	})
}

func TestIntegrationExecutionEngineConfigurationFailureDoesNotCallTransport(t *testing.T) {
	engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusOK)
	defer closeServer()
	claimed, err := engine.ClaimReadyExecutions(context.Background())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = %+v err=%v", claimed, err)
	}
	if err := db.Model(&model.ExternalSystem{}).Where("id = ?", execution.ExternalSystemID).Update("status", model.ExternalSystemStatusDisabled).Error; err != nil {
		t.Fatalf("disable system: %v", err)
	}
	result, err := engine.RunExecution(context.Background(), claimed[0])
	if err != nil || result.ErrorCategory != model.IntegrationErrorCategoryConfiguration {
		t.Fatalf("configuration failure result = %+v err=%v", result, err)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, execution.Id).Error; err != nil || stored.Status != model.IntegrationExecutionStatusFailed {
		t.Fatalf("configuration failure convergence = %+v err=%v", stored, err)
	}
}

func TestIntegrationExecutionEngineSyncConsumerDefinesBusinessSuccess(t *testing.T) {
	tests := []struct {
		name       string
		consumer   SyncResultConsumer
		wantStatus string
		wantBiz    string
	}{
		{name: "consumer success", consumer: SyncResultConsumerFunc(func(_ context.Context, request SyncConsumptionRequest) (SyncConsumptionResult, error) {
			if request.ExecutionNo() == "" || request.SyncBatchNo() != "SYNC-RUNTIME-1" || request.TaskCode() != "runtime_sync" || request.SliceNo() != 1 || string(request.Body()) != `{"result":"ok"}` {
				return SyncConsumptionResult{}, fmt.Errorf("unexpected controlled request")
			}
			return NewSyncConsumptionResult(true, "", 3, 0, "ORG-BATCH-1")
		}), wantStatus: model.IntegrationExecutionStatusSucceeded, wantBiz: model.IntegrationSyncBusinessStatusSucceeded},
		{name: "consumer failure is not retried", consumer: SyncResultConsumerFunc(func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error) {
			return NewSyncConsumptionResult(false, "org_record_invalid", 0, 2, "ORG-BATCH-2")
		}), wantStatus: model.IntegrationExecutionStatusFailed, wantBiz: model.IntegrationSyncBusinessStatusFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusOK)
			defer closeServer()
			configureSyncExecutionFixture(t, engine, db, &execution, test.consumer)
			claimed, err := engine.ClaimReadyExecutions(context.Background())
			if err != nil || len(claimed) != 1 {
				t.Fatalf("claim=%+v err=%v", claimed, err)
			}
			result, err := engine.RunExecution(context.Background(), claimed[0])
			if err != nil {
				t.Fatal(err)
			}
			var stored model.IntegrationExecution
			if err := db.First(&stored, execution.Id).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Status != test.wantStatus || stored.SyncBusinessStatus != test.wantBiz || stored.NextRunAt != nil {
				t.Fatalf("execution=%+v result=%+v", stored, result)
			}
			if test.wantBiz == model.IntegrationSyncBusinessStatusSucceeded && (stored.SyncBusinessSuccessCount != 3 || stored.SyncBusinessReference != "ORG-BATCH-1") {
				t.Fatalf("success summary=%+v", stored)
			}
			if test.wantBiz == model.IntegrationSyncBusinessStatusFailed {
				if result.ErrorCategory != model.IntegrationErrorCategoryBusiness || result.ReasonCode != SyncBusinessReasonProcessingFailed ||
					stored.SyncBusinessReasonCode != "org_record_invalid" || stored.SyncBusinessFailedCount != 2 {
					t.Fatalf("failure summary execution=%+v result=%+v", stored, result)
				}
			}
		})
	}
}

func configureSyncExecutionFixture(t *testing.T, engine *IntegrationExecutionEngine, db *gorm.DB, execution *model.IntegrationExecution, consumer SyncResultConsumer) {
	t.Helper()
	if err := db.AutoMigrate(&model.IntegrationSyncTask{}, &model.IntegrationSyncBatch{}); err != nil {
		t.Fatal(err)
	}
	task := model.IntegrationSyncTask{Basic: model.Basic{Id: 201, State: true}, TaskCode: "runtime_sync", TaskName: "Runtime Sync", Version: 1,
		Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: execution.ExternalSystemID, InterfaceDefinitionID: execution.InterfaceDefinitionID,
		ConsumerCode: "test_sync", ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleNone, Timezone: "UTC",
		CheckpointMode: model.IntegrationSyncCheckpointNone, InputPlan: datatypes.JSON([]byte(`{"version":1,"static_input":{}}`)), Revision: 1}
	batch := model.IntegrationSyncBatch{Basic: model.Basic{Id: 202, State: true}, BatchNo: "SYNC-RUNTIME-1", SyncTaskID: task.Id,
		TaskCode: task.TaskCode, TaskName: task.TaskName, TaskVersion: task.Version, TaskRevision: task.Revision,
		SystemCode: execution.ExternalSystemCode, InterfaceCode: execution.InterfaceCode, InterfaceVersion: execution.InterfaceVersion,
		ConsumerCode: "test_sync", ConsumerVersion: 1, TriggerType: model.IntegrationSyncTriggerManual, TriggerKey: "manual:runtime-sync:1",
		Status: model.IntegrationSyncBatchStatusRunning, CheckpointMode: model.IntegrationSyncCheckpointNone, PlannedSliceCount: 1, CurrentSliceNo: 1, Revision: 1}
	testutil.MustCreate(t, db, &task)
	testutil.MustCreate(t, db, &batch)
	slice := 1
	consumerVersion := 1
	updates := map[string]any{"sync_batch_id": batch.Id, "sync_slice_no": slice, "sync_consumer_code": "test_sync", "sync_consumer_version": consumerVersion,
		"sync_business_status": model.IntegrationSyncBusinessStatusPending}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).Updates(updates).Error; err != nil {
		t.Fatal(err)
	}
	execution.SyncBatchID, execution.SyncSliceNo, execution.SyncConsumerCode, execution.SyncConsumerVersion = &batch.Id, &slice, "test_sync", &consumerVersion
	registry, err := NewStaticSyncConsumerRegistry(SyncConsumerRegistration{Metadata: SyncConsumerMetadata{
		Code: "test_sync", Version: 1, Name: "Test Sync", Status: SyncConsumerStatusEnabled,
		ContentTypes: []string{"application/json"}, MaxResponseBytes: 1024, MaxDuration: time.Second,
		CheckpointModes: []string{model.IntegrationSyncCheckpointNone},
	}, Consumer: consumer})
	if err != nil {
		t.Fatal(err)
	}
	engine.syncConsumers = registry
}

func TestIntegrationExecutionEngineRejectsRuntimeIncompatibleDirtyConfiguration(t *testing.T) {
	engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusOK)
	defer closeServer()
	claimed, err := engine.ClaimReadyExecutions(context.Background())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim = %+v err=%v", claimed, err)
	}
	if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", execution.InterfaceDefinitionID).Update("timeout_seconds", 121).Error; err != nil {
		t.Fatalf("prepare dirty runtime configuration: %v", err)
	}
	result, err := engine.RunExecution(context.Background(), claimed[0])
	if err != nil || result.Succeeded || result.ReasonCode != "integration_execution_runtime_incompatible" {
		t.Fatalf("runtime compatibility result = %+v err=%v", result, err)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, execution.Id).Error; err != nil || stored.Status != model.IntegrationExecutionStatusFailed {
		t.Fatalf("runtime-incompatible convergence = %+v err=%v", stored, err)
	}
}

func TestIntegrationExecutionEngineUsesSupportedCredentialTypes(t *testing.T) {
	for _, credentialType := range []string{model.CredentialTypeBasic, model.CredentialTypeAPIKey, model.CredentialTypeBearerToken} {
		t.Run(credentialType, func(t *testing.T) {
			engine, _, _, closeServer := newExecutionEngineFixture(t, http.StatusOK, credentialType)
			defer closeServer()
			claimed, err := engine.ClaimReadyExecutions(context.Background())
			if err != nil || len(claimed) != 1 {
				t.Fatalf("claim = %+v err=%v", claimed, err)
			}
			result, err := engine.RunExecution(context.Background(), claimed[0])
			if err != nil || !result.Succeeded || result.CredentialCode != "runtime_token" {
				t.Fatalf("run with %s = %+v err=%v", credentialType, result, err)
			}
		})
	}
}

func TestIntegrationExecutionEngineRebuildsSnapshotBeforeCredentialAndTransport(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	restoreLogger := zap.ReplaceGlobals(zap.New(core))
	t.Cleanup(restoreLogger)
	requestSeen := make(chan struct{}, 1)
	engine, db, execution, closeServer := newExecutionEngineFixtureWithHandler(t, http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/runtime/10001" || req.URL.Query().Get("page") != "1" {
			t.Errorf("rebuilt URL = %s", req.URL.String())
		}
		if req.Header.Get("X-Correlation-ID") != "private-correlation-7b" {
			t.Errorf("rebuilt header = %q", req.Header.Get("X-Correlation-ID"))
		}
		if req.Header.Get("Authorization") != "Bearer execution-engine-token" {
			t.Errorf("credential was not injected last: %q", req.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body["active"] != true || body["name"] != "private-display-name-7b" {
			t.Errorf("rebuilt body = %#v err=%v", body, err)
		}
		requestSeen <- struct{}{}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"result":"ok"}`))
	}))
	defer closeServer()

	contract := snapshotTestContract()
	definitionUpdates := map[string]any{
		"http_method": "POST", "relative_path": "/api/runtime/{employee_id}", "input_contract": datatypes.JSON(contract),
		"idempotency_mode": model.InterfaceIdempotencyModeNone, "remote_idempotency_header": "",
	}
	if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", execution.InterfaceDefinitionID).Updates(definitionUpdates).Error; err != nil {
		t.Fatalf("update interface contract: %v", err)
	}
	_, snapshot, hash, err := BuildExecutionInputSnapshot(contract, "POST", "/api/runtime/{employee_id}", execution.InterfaceVersion, ExecutionInputValues{
		PathParams: map[string]string{"employee_id": "10001"}, QueryParams: map[string][]string{"page": {"1"}},
		Headers:  map[string][]string{"x-correlation-id": {"private-correlation-7b"}},
		JSONBody: json.RawMessage(`{"name":"private-display-name-7b","active":true,"items":[]}`),
	})
	if err != nil {
		t.Fatalf("build dynamic snapshot: %v", err)
	}
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).Updates(map[string]any{
		"input_snapshot": datatypes.JSON(snapshot), "input_snapshot_version": model.IntegrationExecutionInputSnapshotVersion,
		"input_snapshot_size": len(snapshot), "input_hash": hash,
		"remote_idempotency_mode": model.InterfaceIdempotencyModeNone, "remote_idempotency_header": "",
	}).Error; err != nil {
		t.Fatalf("update execution snapshot: %v", err)
	}
	claimed, err := engine.ClaimReadyExecutions(context.Background())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim dynamic execution = %+v err=%v", claimed, err)
	}
	result, err := engine.RunExecution(context.Background(), claimed[0])
	if err != nil || !result.Succeeded {
		t.Fatalf("run dynamic execution = %+v err=%v", result, err)
	}
	select {
	case <-requestSeen:
	default:
		t.Fatal("transport did not receive rebuilt request")
	}
	logs := fmt.Sprint(observed.All())
	for _, forbidden := range []string{"private-correlation-7b", "private-display-name-7b", "execution-engine-token"} {
		if strings.Contains(logs, forbidden) {
			t.Fatalf("runtime log leaked input value %q: %s", forbidden, logs)
		}
	}
}

func TestIntegrationExecutionEngineInjectsFrozenRemoteIdempotencyKey(t *testing.T) {
	var received string
	engine, db, execution, closeServer := newExecutionEngineFixtureWithHandler(t, http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		received = req.Header.Get(RemoteIdempotencyHeaderName)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte(`{"error":"busy"}`))
	}))
	defer closeServer()
	if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", execution.InterfaceDefinitionID).Updates(map[string]any{
		"http_method": model.InterfaceMethodPOST, "idempotency_mode": model.InterfaceIdempotencyModeRemoteKeyHeader,
		"remote_idempotency_header": RemoteIdempotencyHeaderName,
	}).Error; err != nil {
		t.Fatalf("update interface idempotency: %v", err)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, execution.Id).Error; err != nil {
		t.Fatalf("load execution: %v", err)
	}
	policy, err := ParseRetryPolicySnapshot(stored.RetryPolicySnapshot)
	if err != nil {
		t.Fatalf("parse retry policy: %v", err)
	}
	policy.IdempotencyMode = RemoteIdempotencyKeyHeader
	policy.RemoteIdempotencyHeader = RemoteIdempotencyHeaderName
	encoded, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal retry policy: %v", err)
	}
	remoteKey := "server-generated-remote-key"
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).Updates(map[string]any{
		"retry_policy_snapshot": datatypes.JSON(encoded), "remote_idempotency_mode": model.InterfaceIdempotencyModeRemoteKeyHeader,
		"remote_idempotency_header": RemoteIdempotencyHeaderName, "remote_idempotency_key": remoteKey,
	}).Error; err != nil {
		t.Fatalf("freeze execution idempotency: %v", err)
	}
	claimed, err := engine.ClaimReadyExecutions(context.Background())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	result, err := engine.RunExecution(context.Background(), claimed[0])
	if err != nil || received != remoteKey || result.ExecutionStatus != model.IntegrationExecutionStatusRetryWaiting {
		t.Fatalf("remote idempotency received=%q result=%+v err=%v", received, result, err)
	}
}

func TestIntegrationExecutionEngineRejectsMissingOrTamperedSnapshotWithoutHTTP(t *testing.T) {
	t.Run("missing snapshot is not claimable", func(t *testing.T) {
		var calls atomic.Int32
		engine, db, execution, closeServer := newExecutionEngineFixtureWithHandler(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			calls.Add(1)
		}))
		defer closeServer()
		if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).Updates(map[string]any{
			"input_snapshot": datatypes.JSON([]byte(`{}`)), "input_snapshot_version": 0, "input_snapshot_size": 0,
		}).Error; err != nil {
			t.Fatalf("remove snapshot: %v", err)
		}
		claimed, err := engine.ClaimReadyExecutions(context.Background())
		if err != nil || len(claimed) != 0 || calls.Load() != 0 {
			t.Fatalf("missing snapshot claim=%+v calls=%d err=%v", claimed, calls.Load(), err)
		}
	})

	t.Run("hash mismatch completes without transport", func(t *testing.T) {
		var calls atomic.Int32
		engine, db, execution, closeServer := newExecutionEngineFixtureWithHandler(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			calls.Add(1)
		}))
		defer closeServer()
		if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).Update("input_hash", strings.Repeat("0", 64)).Error; err != nil {
			t.Fatalf("tamper hash: %v", err)
		}
		claimed, err := engine.ClaimReadyExecutions(context.Background())
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim tampered execution=%+v err=%v", claimed, err)
		}
		result, err := engine.RunExecution(context.Background(), claimed[0])
		if err != nil || result.Succeeded || result.ReasonCode != "execution_input_hash_mismatch" || calls.Load() != 0 {
			t.Fatalf("tampered result=%+v calls=%d err=%v", result, calls.Load(), err)
		}
		var stored model.IntegrationExecution
		if err := db.First(&stored, execution.Id).Error; err != nil || stored.Status != model.IntegrationExecutionStatusFailed {
			t.Fatalf("tampered execution state=%+v err=%v", stored, err)
		}
	})
}

func TestInMemoryConcurrencyGuard(t *testing.T) {
	guard, err := NewInMemoryConcurrencyGuard(1, 1, 1)
	if err != nil {
		t.Fatalf("new guard: %v", err)
	}
	release, err := guard.Acquire(1, 1)
	if err != nil || release == nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := guard.Acquire(1, 1); err == nil {
		t.Fatal("expected bounded concurrency rejection")
	}
	release()
	if release, err := guard.Acquire(1, 1); err != nil {
		t.Fatalf("release did not restore quota: %v", err)
	} else {
		release()
	}
}

func newExecutionEngineFixture(t *testing.T, responseStatus int, credentialTypes ...string) (*IntegrationExecutionEngine, *gorm.DB, model.IntegrationExecution, func()) {
	return newExecutionEngineFixtureWithHandler(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(responseStatus)
		_, _ = writer.Write([]byte(`{"result":"ok"}`))
	}), credentialTypes...)
}

func newExecutionEngineFixtureWithHandler(t *testing.T, handler http.Handler, credentialTypes ...string) (*IntegrationExecutionEngine, *gorm.DB, model.IntegrationExecution, func()) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.Credential{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}, &model.IntegrationExecution{}, &model.IntegrationLog{})
	client, _, closeServer := newTestTransportClient(t, false, handler)
	protector, err := security.NewCredentialSecretProtectorWithKey("execution-engine-master-key")
	if err != nil {
		closeServer()
		t.Fatalf("new protector: %v", err)
	}
	credentialType := model.CredentialTypeBearerToken
	secretValues := map[string]string{"token": "execution-engine-token"}
	if len(credentialTypes) > 0 {
		credentialType = credentialTypes[0]
		switch credentialType {
		case model.CredentialTypeBasic:
			secretValues = map[string]string{"username": "runtime-user", "password": "runtime-password"}
		case model.CredentialTypeAPIKey:
			secretValues = map[string]string{"api_key": "execution-engine-api-key"}
		}
	}
	secret, err := json.Marshal(secretValues)
	if err != nil {
		closeServer()
		t.Fatalf("marshal secret: %v", err)
	}
	envelope, err := protector.Seal(secret)
	if err != nil {
		closeServer()
		t.Fatalf("seal secret: %v", err)
	}
	system := model.ExternalSystem{Basic: model.Basic{Id: 101, State: true}, SystemCode: "runtime_system", Name: "运行时系统", SystemType: model.ExternalSystemTypeHR,
		BaseURL: "https://api.integration.test", OwnerIdentifier: "runtime_owner", OwnerName: "运行负责人", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	credential := model.Credential{Basic: model.Basic{Id: 102, State: true}, ExternalSystemID: system.Id, CredentialCode: "runtime_token", Name: "运行时令牌",
		CredentialType: credentialType, Status: model.CredentialStatusActive, SecretStorageRef: envelope.StorageRef, SecretCiphertext: envelope.Ciphertext,
		SecretNonce: envelope.Nonce, SecretFingerprint: envelope.Fingerprint, Version: 1, Revision: 1}
	credentialID := credential.Id
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: 103, State: true}, ExternalSystemID: system.Id, InterfaceCode: "runtime_list", Name: "运行时列表", Version: 1,
		Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/api/runtime", CredentialID: &credentialID,
		TimeoutSeconds: 15, ResponseLimit: 1024, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1,
		IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod}
	policy := model.RetryPolicy{Basic: model.Basic{Id: 105, State: true}, PolicyCode: "runtime_retry", PolicyName: "运行时重试", Version: 1,
		Status: model.RetryPolicyStatusEnabled, MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 5000,
		BackoffType: model.RetryBackoffTypeFixed, BackoffMultiplier: 1, JitterType: model.RetryJitterTypeNone, JitterRatio: 0,
		RetryWindowMs: 60000, RetryableErrorCategories: datatypes.JSON([]byte(`["network","remote","timeout"]`)),
		RetryableHTTPStatuses: datatypes.JSON([]byte(`[429,502,503,504]`)), RespectRetryAfter: true, Revision: 1}
	retrySnapshot, err := BuildRetryPolicySnapshot(policy, RetryPolicySnapshotOptions{IdempotencyMode: RemoteIdempotencySafeMethod})
	if err != nil {
		closeServer()
		t.Fatalf("build retry snapshot: %v", err)
	}
	_, snapshot, inputHash, err := BuildExecutionInputSnapshot(
		definition.InputContract, definition.HTTPMethod, definition.RelativePath, definition.Version, ExecutionInputValues{},
	)
	if err != nil {
		closeServer()
		t.Fatalf("build execution input snapshot: %v", err)
	}
	execution := model.IntegrationExecution{Basic: model.Basic{Id: 104, State: true}, ExecutionNo: "INT-RUNTIME-104", ExternalSystemID: system.Id,
		ExternalSystemCode: system.SystemCode, ExternalSystemName: system.Name, InterfaceDefinitionID: definition.Id, InterfaceCode: definition.InterfaceCode,
		InterfaceName: definition.Name, InterfaceVersion: definition.Version, TriggerSource: model.IntegrationTriggerSourceManual,
		Status: model.IntegrationExecutionStatusCreated, IdempotencyScope: "runtime", IdempotencyKey: "execution-104",
		InputHash: inputHash, InputSnapshot: datatypes.JSON(snapshot),
		InputSnapshotVersion: model.IntegrationExecutionInputSnapshotVersion, InputSnapshotSize: len(snapshot),
		RetryPolicyID: &policy.Id, RetryPolicySnapshot: datatypes.JSON(retrySnapshot), RetryPolicySnapshotVersion: RetryPolicySnapshotVersion,
		RemoteIdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Revision: 1}
	for _, value := range []any{&system, &credential, &policy, &definition, &execution} {
		testutil.MustCreate(t, db, value)
	}
	primary := &database.PrimaryDB{DB: db}
	credentialRepository := impl.NewCredentialRepositoryImpl(primary)
	provider := NewCredentialProvider(credentialRepository, impl.NewInterfaceDefinitionRepositoryImpl(primary), protector)
	guard, err := NewInMemoryConcurrencyGuard(4, 2, 1)
	if err != nil {
		closeServer()
		t.Fatalf("new concurrency guard: %v", err)
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		closeServer()
		t.Fatalf("new snowflake: %v", err)
	}
	engine, err := NewIntegrationExecutionEngine(
		impl.NewIntegrationExecutionRepositoryImpl(primary), impl.NewExternalSystemRepositoryImpl(primary), impl.NewInterfaceDefinitionRepositoryImpl(primary), credentialRepository,
		impl.NewIntegrationSyncBatchRepositoryImpl(primary), provider, client, guard, NewSyncConsumerRegistry(), sf,
		ExecutionEngineOptions{WorkerID: "runtime-worker-1", LeaseDuration: IntegrationDefaultLeaseDuration, BatchSize: 2},
	)
	if err != nil {
		closeServer()
		t.Fatalf("new execution engine: %v", err)
	}
	return engine, db, execution, closeServer
}

var _ repository.CredentialRuntimeIdentity
