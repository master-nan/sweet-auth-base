package service

import (
	"backend/internal/database"
	"backend/internal/integration"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type syncConsumerTestResolver map[string][]net.IP

func (r syncConsumerTestResolver) LookupIP(_ context.Context, host string) ([]net.IP, error) {
	values, ok := r[host]
	if !ok {
		return nil, fmt.Errorf("host not found")
	}
	return append([]net.IP(nil), values...), nil
}

type syncConsumerTestRoundTripper struct {
	target *url.URL
	base   http.RoundTripper
}

func (r syncConsumerTestRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	copyRequest := request.Clone(request.Context())
	copyURL := *request.URL
	copyURL.Scheme, copyURL.Host = r.target.Scheme, r.target.Host
	copyRequest.URL, copyRequest.Host = &copyURL, ""
	return r.base.RoundTrip(copyRequest)
}

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

func TestIntegrationSyncPostgreSQLRunnerTransportConsumerCheckpointE2E(t *testing.T) {
	db := openSyncCoordinatorPostgreSQL(t)
	var httpCalls atomic.Int32
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		httpCalls.Add(1)
		if request.URL.Path != "/employees" || request.URL.Query().Get("updated_from") == "" || request.URL.Query().Get("updated_to") == "" {
			t.Errorf("unexpected sync request: %s", request.URL.String())
		}
		if request.Header.Get("Authorization") != "Bearer sync-consumer-token" {
			t.Errorf("credential was not injected")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"employees":[{"id":"10001"}]}`))
	}))
	defer tlsServer.Close()
	target, err := url.Parse(tlsServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := integration.NewEndpointPolicy(false, nil, syncConsumerTestResolver{"sync-consumer.test": {net.ParseIP("93.184.216.34")}})
	if err != nil {
		t.Fatal(err)
	}
	transport, err := integration.NewHTTPTransportClient(policy, integration.TransportClientOptions{RoundTripper: syncConsumerTestRoundTripper{target: target, base: tlsServer.Client().Transport}})
	if err != nil {
		t.Fatal(err)
	}
	consumerCalled := make(chan integration.SyncConsumptionRequest, 1)
	registry, err := integration.NewStaticSyncConsumerRegistry(integration.SyncConsumerRegistration{
		Metadata: integration.SyncConsumerMetadata{Code: "test_sync_consumer", Version: 1, Name: "Test Consumer", Status: integration.SyncConsumerStatusEnabled,
			ContentTypes: []string{"application/json"}, MaxResponseBytes: 1 << 20, MaxDuration: time.Second,
			CheckpointModes: []string{model.IntegrationSyncCheckpointTimestamp}},
		Consumer: integration.SyncResultConsumerFunc(func(_ context.Context, request integration.SyncConsumptionRequest) (integration.SyncConsumptionResult, error) {
			consumerCalled <- request
			return integration.NewSyncConsumptionResult(true, "", 1, 0, "ORG-SYNC-1")
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	protector, err := security.NewCredentialSecretProtectorWithKey("sync-consumer-e2e-master-key")
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := json.Marshal(map[string]string{"token": "sync-consumer-token"})
	envelope, err := protector.Seal(secret)
	if err != nil {
		t.Fatal(err)
	}
	system := model.ExternalSystem{Basic: model.Basic{Id: nextSyncTestID(), State: true}, SystemCode: "sync_consumer_e2e", Name: "Sync Consumer E2E",
		SystemType: model.ExternalSystemTypeHR, BaseURL: "https://sync-consumer.test", OwnerIdentifier: "ops", OwnerName: "Ops",
		Status: model.ExternalSystemStatusEnabled, Revision: 1}
	credential := model.Credential{Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id,
		CredentialCode: "sync_consumer_token", Name: "Sync Consumer Token", CredentialType: model.CredentialTypeBearerToken,
		Status: model.CredentialStatusActive, SecretStorageRef: envelope.StorageRef, SecretCiphertext: envelope.Ciphertext,
		SecretNonce: envelope.Nonce, SecretFingerprint: envelope.Fingerprint, Version: 1, Revision: 1}
	contract, _ := json.Marshal(integration.InterfaceInputContract{Version: 1, Parameters: []integration.InputParameterDefinition{
		{Code: "updated_from", Location: "query", DataType: "string", Required: true, MaxLength: 64},
		{Code: "updated_to", Location: "query", DataType: "string", Required: true, MaxLength: 64},
	}})
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: nextSyncTestID(), State: true}, ExternalSystemID: system.Id,
		InterfaceCode: "employees", Name: "Employees", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET,
		RelativePath: "/employees", CredentialID: &credential.Id, TimeoutSeconds: 5, ResponseLimit: 1 << 20, InputContract: contract,
		IdempotencyMode: model.InterfaceIdempotencyModeSafeMethod, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	var databaseNow time.Time
	if err := db.Raw("SELECT CURRENT_TIMESTAMP AT TIME ZONE 'UTC'").Scan(&databaseNow).Error; err != nil {
		t.Fatal(err)
	}
	checkpoint, due := databaseNow.UTC().Add(-30*time.Minute), databaseNow.UTC().Add(-time.Minute)
	plan, _ := json.Marshal(integration.SyncExecutionInputPlan{Version: 1, StaticInput: integration.ExecutionInputValues{},
		WindowStartBinding: &integration.SyncWindowBinding{Location: "query", Code: "updated_from", Format: "rfc3339"},
		WindowEndBinding:   &integration.SyncWindowBinding{Location: "query", Code: "updated_to", Format: "rfc3339"}})
	task := model.IntegrationSyncTask{Basic: model.Basic{Id: nextSyncTestID(), State: true}, TaskCode: "sync_consumer_e2e", TaskName: "Sync Consumer E2E",
		Version: 1, Status: model.IntegrationSyncTaskStatusEnabled, ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id,
		ConsumerCode: "test_sync_consumer", ConsumerVersion: 1, ScheduleType: model.IntegrationSyncScheduleCron, CronExpression: "* * * * *",
		Timezone: "UTC", NextScheduledAt: &due, CheckpointMode: model.IntegrationSyncCheckpointTimestamp,
		InitialCheckpointAt: &checkpoint, CheckpointAt: &checkpoint, WindowSliceSeconds: 3600, InputPlan: datatypes.JSON(plan), Revision: 1}
	for _, value := range []any{&system, &credential, &definition, &task} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}

	primary := &database.PrimaryDB{DB: db}
	executions := impl.NewIntegrationExecutionRepositoryImpl(primary)
	systems := impl.NewExternalSystemRepositoryImpl(primary)
	interfaces := impl.NewInterfaceDefinitionRepositoryImpl(primary)
	credentials := impl.NewCredentialRepositoryImpl(primary)
	batches := impl.NewIntegrationSyncBatchRepositoryImpl(primary)
	tasks := impl.NewIntegrationSyncTaskRepositoryImpl(primary)
	snowflake, _ := utils.NewSnowflake(4)
	executionService := NewIntegrationExecutionService(executions, impl.NewIntegrationLogRepositoryImpl(primary), systems, interfaces,
		impl.NewRetryPolicyRepositoryImpl(primary), snowflake, &integrationExecutionAuditWriter{})
	coordinator := NewIntegrationSyncCoordinator(tasks, batches, executions, systems, interfaces, executionService,
		integration.NewPersistedSyncBusinessResultProvider(), snowflake)
	syncRunner, err := integration.NewIntegrationSyncRunner(coordinator, integration.SyncRunnerConfig{Enabled: true, RunnerID: "sync-consumer-runner",
		PollInterval: time.Second, ScheduleBatchSize: 2, CoordinateBatchSize: 2, ShutdownTimeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	provider := integration.NewCredentialProvider(credentials, interfaces, protector)
	guard, _ := integration.NewInMemoryConcurrencyGuard(2, 2, 2)
	engine, err := integration.NewIntegrationExecutionEngine(executions, systems, interfaces, credentials, batches, provider, transport, guard, registry,
		snowflake, integration.ExecutionEngineOptions{WorkerID: "sync-consumer-worker", LeaseDuration: integration.IntegrationDefaultLeaseDuration, BatchSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := integration.NewIntegrationWorkerRunner(engine, integration.WorkerRunnerConfig{Enabled: true, WorkerID: "sync-consumer-worker",
		PollInterval: time.Second, ClaimBatchSize: 2, InstanceConcurrency: 2, LeaseRecoveryInterval: 10 * time.Second,
		ShutdownTimeout: 3 * time.Second, LeaseDuration: integration.IntegrationDefaultLeaseDuration})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := worker.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := syncRunner.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syncRunner.Stop(context.Background()); _ = worker.Stop(context.Background()) })
	waitForPostgreSQLSyncBatchStatus(t, db, model.IntegrationSyncBatchStatusSucceeded, 12*time.Second)
	if err := syncRunner.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case request := <-consumerCalled:
		if request.ExecutionNo() == "" || request.SyncBatchNo() == "" || request.TaskCode() != task.TaskCode || request.TaskVersion() != task.Version ||
			request.SliceNo() != 1 || !strings.Contains(string(request.Body()), "10001") {
			t.Fatalf("consumer request=%s", request.String())
		}
	default:
		t.Fatal("consumer was not called")
	}
	var execution model.IntegrationExecution
	if err := db.Where("sync_batch_id IS NOT NULL").First(&execution).Error; err != nil {
		t.Fatal(err)
	}
	if execution.Status != model.IntegrationExecutionStatusSucceeded || execution.SyncBusinessStatus != model.IntegrationSyncBusinessStatusSucceeded ||
		execution.SyncBusinessSuccessCount != 1 || execution.SyncBusinessReference != "ORG-SYNC-1" || httpCalls.Load() != 1 {
		t.Fatalf("execution=%+v http_calls=%d", execution, httpCalls.Load())
	}
	var refreshedTask model.IntegrationSyncTask
	if err := db.First(&refreshedTask, task.Id).Error; err != nil || refreshedTask.CheckpointAt == nil || !refreshedTask.CheckpointAt.After(checkpoint) {
		t.Fatalf("checkpoint=%+v err=%v", refreshedTask.CheckpointAt, err)
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
	if err := db.AutoMigrate(&model.ExternalSystem{}, &model.Credential{}, &model.RetryPolicy{}, &model.InterfaceDefinition{}, &model.IntegrationSyncTask{}, &model.IntegrationSyncBatch{}, &model.IntegrationExecution{}, &model.IntegrationLog{}); err != nil {
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
