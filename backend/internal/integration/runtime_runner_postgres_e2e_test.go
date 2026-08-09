package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"backend/dto/request"
	"backend/internal/database"
	"backend/internal/integration"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"backend/service"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type runtimeAcceptanceResolver map[string][]net.IP

func (r runtimeAcceptanceResolver) LookupIP(_ context.Context, host string) ([]net.IP, error) {
	values, exists := r[host]
	if !exists {
		return nil, errors.New("host not found")
	}
	return append([]net.IP(nil), values...), nil
}

type runtimeAcceptanceRoundTripper struct {
	target *url.URL
	base   http.RoundTripper
}

func (r runtimeAcceptanceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	copyRequest := req.Clone(req.Context())
	copyURL := *req.URL
	copyURL.Scheme = r.target.Scheme
	copyURL.Host = r.target.Host
	copyRequest.URL = &copyURL
	copyRequest.Host = ""
	return r.base.RoundTrip(copyRequest)
}

type runtimeAcceptanceAuditWriter struct{}

func (runtimeAcceptanceAuditWriter) RecordTransactionalAuditContext(
	context.Context,
	*gorm.DB,
	service.TransactionalAuditRecord,
) error {
	return nil
}

func TestIntegrationWorkerRunnerPostgreSQLJSONBAndTLSEndToEnd(t *testing.T) {
	db := openRuntimeAcceptancePostgreSQL(t)
	received := make(chan struct{}, 1)
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/base/runtime/10001" {
			t.Errorf("unexpected request target: %s %s", req.Method, req.URL.String())
		}
		if req.URL.Query().Get("page") != "1" || strings.Join(req.URL.Query()["tags"], ",") != "east,south" {
			t.Errorf("unexpected query: %s", req.URL.RawQuery)
		}
		if req.Header.Get("X-Correlation-ID") != "runner-jsonb-roundtrip" {
			t.Errorf("unexpected ordinary header: %q", req.Header.Get("X-Correlation-ID"))
		}
		if req.Header.Get("Authorization") != "Bearer runner-secret-token" {
			t.Errorf("credential was not injected after snapshot validation")
		}
		var body map[string]any
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body["name"] != "张三" || body["active"] != true {
			t.Errorf("unexpected JSON body: %#v err=%v", body, err)
		}
		received <- struct{}{}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer tlsServer.Close()
	target, err := url.Parse(tlsServer.URL)
	if err != nil {
		t.Fatalf("parse TLS server URL: %v", err)
	}
	policy, err := integration.NewEndpointPolicy(false, nil, runtimeAcceptanceResolver{
		"runtime.acceptance.test": {net.ParseIP("93.184.216.34")},
	})
	if err != nil {
		t.Fatalf("create endpoint policy: %v", err)
	}
	transport, err := integration.NewHTTPTransportClient(policy, integration.TransportClientOptions{
		RoundTripper: runtimeAcceptanceRoundTripper{target: target, base: tlsServer.Client().Transport},
	})
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}

	protector, err := security.NewCredentialSecretProtectorWithKey("runtime-acceptance-master-key")
	if err != nil {
		t.Fatalf("create credential protector: %v", err)
	}
	secret, _ := json.Marshal(map[string]string{"token": "runner-secret-token"})
	envelope, err := protector.Seal(secret)
	if err != nil {
		t.Fatalf("seal credential: %v", err)
	}
	system := model.ExternalSystem{
		Basic: model.Basic{Id: 7101, State: true}, SystemCode: "runtime_acceptance", Name: "Runtime Acceptance",
		SystemType: model.ExternalSystemTypeHR, BaseURL: "https://runtime.acceptance.test/base",
		OwnerIdentifier: "runtime-owner", OwnerName: "运行验收负责人",
		Status: model.ExternalSystemStatusEnabled, Revision: 1,
	}
	credential := model.Credential{
		Basic: model.Basic{Id: 7102, State: true}, ExternalSystemID: system.Id,
		CredentialCode: "runtime_acceptance_token", Name: "Runtime Acceptance Token",
		CredentialType: model.CredentialTypeBearerToken, Status: model.CredentialStatusActive,
		SecretStorageRef: envelope.StorageRef, SecretCiphertext: envelope.Ciphertext,
		SecretNonce: envelope.Nonce, SecretFingerprint: envelope.Fingerprint,
		Version: 1, Revision: 1,
	}
	credentialID := credential.Id
	contract := datatypes.JSON([]byte(`{
		"version":1,
		"parameters":[
			{"code":"employee_id","location":"path","data_type":"string","required":true,"max_length":64,"allow_multiple":false,"sensitive":false},
			{"code":"page","location":"query","data_type":"integer","required":true,"max_length":8,"allow_multiple":false,"sensitive":false},
			{"code":"tags","location":"query","data_type":"string","required":false,"max_length":32,"allow_multiple":true,"sensitive":false},
			{"code":"X-Correlation-ID","location":"header","data_type":"string","required":true,"max_length":64,"allow_multiple":false,"sensitive":false},
			{"code":"name","location":"body","data_type":"string","required":true,"max_length":128,"allow_multiple":false,"sensitive":false},
			{"code":"active","location":"body","data_type":"boolean","required":true,"max_length":8,"allow_multiple":false,"sensitive":false}
		]
	}`))
	definition := model.InterfaceDefinition{
		Basic: model.Basic{Id: 7103, State: true}, ExternalSystemID: system.Id,
		InterfaceCode: "runtime_submit", Name: "Runtime Submit", Version: 1,
		Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodPOST,
		RelativePath: "/runtime/{employee_id}", CredentialID: &credentialID,
		TimeoutSeconds: 10, ResponseLimit: 1024 * 1024, InputContract: contract,
		Status: model.InterfaceDefinitionStatusEnabled, Revision: 1,
	}
	for _, value := range []any{&system, &credential, &definition} {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("create runtime fixture: %v", err)
		}
	}

	primary := &database.PrimaryDB{DB: db}
	executions := impl.NewIntegrationExecutionRepositoryImpl(primary)
	logs := impl.NewIntegrationLogRepositoryImpl(primary)
	systems := impl.NewExternalSystemRepositoryImpl(primary)
	interfaces := impl.NewInterfaceDefinitionRepositoryImpl(primary)
	policies := impl.NewRetryPolicyRepositoryImpl(primary)
	credentials := impl.NewCredentialRepositoryImpl(primary)
	snowflake, err := utils.NewSnowflake(9)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	applicationService := service.NewIntegrationExecutionService(
		executions, logs, systems, interfaces, policies, snowflake, runtimeAcceptanceAuditWriter{},
	)
	submitContext, cancelSubmit := context.WithCancel(context.Background())
	created, err := applicationService.CreateExecution(submitContext, request.IntegrationExecutionCreateReq{
		ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id,
		TriggerSource:    model.IntegrationTriggerSourceManual,
		IdempotencyScope: "runtime-acceptance", IdempotencyKey: "jsonb-runner-tls",
		Input: request.IntegrationExecutionInputReq{
			PathParams:  map[string]string{"employee_id": "10001"},
			QueryParams: map[string][]string{"tags": {"south", "east"}, "page": {"1"}},
			Headers:     map[string][]string{"X-Correlation-ID": {"runner-jsonb-roundtrip"}},
			JSONBody:    json.RawMessage(` { "active":true, "name":"\u5f20\u4e09" } `),
		},
	})
	cancelSubmit()
	if err != nil {
		t.Fatalf("submit execution: %v", err)
	}

	provider := integration.NewCredentialProvider(credentials, interfaces, protector)
	guard, err := integration.NewInMemoryConcurrencyGuard(4, 2, 1)
	if err != nil {
		t.Fatalf("create concurrency guard: %v", err)
	}
	engine, err := integration.NewIntegrationExecutionEngine(
		executions, systems, interfaces, credentials, provider, transport, guard, snowflake,
		integration.ExecutionEngineOptions{
			WorkerID: "runtime-acceptance-worker", LeaseDuration: integration.IntegrationDefaultLeaseDuration, BatchSize: 2,
		},
	)
	if err != nil {
		t.Fatalf("create execution engine: %v", err)
	}
	runner, err := integration.NewIntegrationWorkerRunner(engine, integration.WorkerRunnerConfig{
		Enabled: true, WorkerID: "runtime-acceptance-worker", PollInterval: time.Second,
		ClaimBatchSize: 2, InstanceConcurrency: 2, LeaseRecoveryInterval: 10 * time.Second,
		ShutdownTimeout: 3 * time.Second, LeaseDuration: integration.IntegrationDefaultLeaseDuration,
	})
	if err != nil {
		t.Fatalf("create worker runner: %v", err)
	}
	runnerContext, cancelRunner := context.WithCancel(context.Background())
	defer cancelRunner()
	if err := runner.Start(runnerContext); err != nil {
		t.Fatalf("start persistent runner: %v", err)
	}
	t.Cleanup(func() { _ = runner.Stop(context.Background()) })

	waitRuntimeAcceptance(t, 5*time.Second, func() bool {
		var execution model.IntegrationExecution
		return db.First(&execution, created.Id).Error == nil && execution.Status == model.IntegrationExecutionStatusSucceeded
	})
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("local TLS server did not receive rebuilt request")
	}
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatalf("stop persistent runner: %v", err)
	}

	var stored model.IntegrationExecution
	var attempt model.IntegrationLog
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("load completed execution: %v", err)
	}
	if err := db.Where("execution_id = ?", created.Id).First(&attempt).Error; err != nil {
		t.Fatalf("load completed attempt: %v", err)
	}
	if stored.Status != model.IntegrationExecutionStatusSucceeded || stored.CurrentAttempt != 1 ||
		attempt.Status != model.IntegrationLogStatusSucceeded || attempt.CredentialCode != credential.CredentialCode {
		t.Fatalf("unexpected runtime convergence: execution=%+v attempt=%+v", stored, attempt)
	}
}

func TestIntegrationWorkerRunnerPostgreSQLRetry503ThenSuccess(t *testing.T) {
	runRuntimeRetryEndToEnd(t, runtimeRetryScenario{
		statuses: []int{http.StatusServiceUnavailable, http.StatusOK}, maxAttempts: 3,
		expectedStatus: model.IntegrationExecutionStatusSucceeded, expectedAttempts: 2,
	})
}

func TestIntegrationWorkerRunnerPostgreSQLRetryExhaustsMaxAttempts(t *testing.T) {
	runRuntimeRetryEndToEnd(t, runtimeRetryScenario{
		statuses: []int{http.StatusServiceUnavailable, http.StatusServiceUnavailable}, maxAttempts: 2,
		expectedStatus: model.IntegrationExecutionStatusFailed, expectedAttempts: 2,
	})
}

func TestIntegrationWorkerRunnerPostgreSQLSingleAttemptNeverCreatesRetry(t *testing.T) {
	runRuntimeRetryEndToEnd(t, runtimeRetryScenario{
		statuses: []int{http.StatusServiceUnavailable}, maxAttempts: 1,
		expectedStatus: model.IntegrationExecutionStatusFailed, expectedAttempts: 1,
	})
}

func TestIntegrationWorkerRunnerPostgreSQLRetryAfter429ThenSuccess(t *testing.T) {
	runRuntimeRetryEndToEnd(t, runtimeRetryScenario{
		statuses: []int{http.StatusTooManyRequests, http.StatusOK}, maxAttempts: 2,
		retryAfter: "2", expectedRetrySource: integration.RetryAfterSourceHTTPDelta,
		expectedMinimumDelay: 1900 * time.Millisecond,
		expectedStatus:       model.IntegrationExecutionStatusSucceeded, expectedAttempts: 2,
	})
}

func TestIntegrationWorkerRunnerPostgreSQLUnknownPostWithoutRemoteKeyDoesNotRetry(t *testing.T) {
	runRuntimeRetryEndToEnd(t, runtimeRetryScenario{
		method: http.MethodPost, idempotencyMode: model.InterfaceIdempotencyModeNone,
		firstResponseDelay: 1200 * time.Millisecond, timeoutSeconds: 1, maxAttempts: 2,
		expectedStatus: model.IntegrationExecutionStatusFailed, expectedAttempts: 1,
	})
}

func TestIntegrationWorkerRunnerPostgreSQLUnknownPostReusesRemoteIdempotencyKey(t *testing.T) {
	runRuntimeRetryEndToEnd(t, runtimeRetryScenario{
		method: http.MethodPost, idempotencyMode: model.InterfaceIdempotencyModeRemoteKeyHeader,
		firstResponseDelay: 1200 * time.Millisecond, timeoutSeconds: 1, maxAttempts: 2,
		expectedStatus: model.IntegrationExecutionStatusSucceeded, expectedAttempts: 2,
		expectRemoteIdempotencyKey: true,
	})
}

type runtimeRetryScenario struct {
	statuses                   []int
	maxAttempts                int
	expectedStatus             string
	expectedAttempts           int
	method                     string
	idempotencyMode            string
	retryAfter                 string
	expectedRetrySource        string
	expectedMinimumDelay       time.Duration
	firstResponseDelay         time.Duration
	timeoutSeconds             int
	expectRemoteIdempotencyKey bool
}

func runRuntimeRetryEndToEnd(t *testing.T, scenario runtimeRetryScenario) {
	t.Helper()
	if scenario.method == "" {
		scenario.method = http.MethodGet
	}
	if scenario.idempotencyMode == "" {
		scenario.idempotencyMode = model.InterfaceIdempotencyModeSafeMethod
	}
	if scenario.timeoutSeconds == 0 {
		scenario.timeoutSeconds = 10
	}
	if scenario.expectedMinimumDelay == 0 && scenario.expectedAttempts > 1 {
		scenario.expectedMinimumDelay = 900 * time.Millisecond
	}
	db := openRuntimeAcceptancePostgreSQL(t)
	var callCount atomic.Int32
	var firstCallAt atomic.Int64
	var secondCallAt atomic.Int64
	var firstRemoteKey atomic.Value
	var secondRemoteKey atomic.Value
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		call := int(callCount.Add(1))
		if call == 1 {
			firstCallAt.Store(time.Now().UnixNano())
		} else if call == 2 {
			secondCallAt.Store(time.Now().UnixNano())
		}
		if req.Method != scenario.method || req.URL.Path != "/base/retry" {
			t.Errorf("unexpected retry target: %s %s", req.Method, req.URL.String())
		}
		if req.Header.Get("Authorization") != "Bearer retry-runner-token" {
			t.Errorf("retry credential was not injected")
		}
		if scenario.expectRemoteIdempotencyKey {
			key := req.Header.Get(integration.RemoteIdempotencyHeaderName)
			if call == 1 {
				firstRemoteKey.Store(key)
			} else if call == 2 {
				secondRemoteKey.Store(key)
			}
		}
		if call == 1 && scenario.firstResponseDelay > 0 {
			time.Sleep(scenario.firstResponseDelay)
		}
		writer.Header().Set("Content-Type", "application/json")
		if call == 1 && scenario.retryAfter != "" {
			writer.Header().Set("Retry-After", scenario.retryAfter)
		}
		status := http.StatusOK
		if len(scenario.statuses) > 0 {
			status = scenario.statuses[len(scenario.statuses)-1]
			if call <= len(scenario.statuses) {
				status = scenario.statuses[call-1]
			}
		}
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte("{\"retry\":true}"))
	}))
	defer tlsServer.Close()
	target, err := url.Parse(tlsServer.URL)
	if err != nil {
		t.Fatalf("parse retry TLS server URL: %v", err)
	}
	endpointPolicy, err := integration.NewEndpointPolicy(false, nil, runtimeAcceptanceResolver{
		"retry.runtime.acceptance.test": {net.ParseIP("93.184.216.34")},
	})
	if err != nil {
		t.Fatalf("create retry endpoint policy: %v", err)
	}
	transport, err := integration.NewHTTPTransportClient(endpointPolicy, integration.TransportClientOptions{
		RoundTripper: runtimeAcceptanceRoundTripper{target: target, base: tlsServer.Client().Transport},
	})
	if err != nil {
		t.Fatalf("create retry transport: %v", err)
	}
	protector, err := security.NewCredentialSecretProtectorWithKey("runtime-retry-master-key")
	if err != nil {
		t.Fatalf("create retry credential protector: %v", err)
	}
	secret, _ := json.Marshal(map[string]string{"token": "retry-runner-token"})
	envelope, err := protector.Seal(secret)
	if err != nil {
		t.Fatalf("seal retry credential: %v", err)
	}
	baseID := 7300 + scenario.maxAttempts*10
	system := model.ExternalSystem{
		Basic: model.Basic{Id: baseID + 1, State: true}, SystemCode: fmt.Sprintf("runtime_retry_%d", scenario.maxAttempts), Name: "Runtime Retry",
		SystemType: model.ExternalSystemTypeHR, BaseURL: "https://retry.runtime.acceptance.test/base",
		OwnerIdentifier: "retry-owner", OwnerName: "重试验收负责人", Status: model.ExternalSystemStatusEnabled, Revision: 1,
	}
	credential := model.Credential{
		Basic: model.Basic{Id: baseID + 2, State: true}, ExternalSystemID: system.Id,
		CredentialCode: fmt.Sprintf("runtime_retry_token_%d", scenario.maxAttempts), Name: "Runtime Retry Token",
		CredentialType: model.CredentialTypeBearerToken, Status: model.CredentialStatusActive,
		SecretStorageRef: envelope.StorageRef, SecretCiphertext: envelope.Ciphertext, SecretNonce: envelope.Nonce,
		SecretFingerprint: envelope.Fingerprint, Version: 1, Revision: 1,
	}
	policy := model.RetryPolicy{
		Basic: model.Basic{Id: baseID + 3, State: true}, PolicyCode: fmt.Sprintf("runtime_retry_policy_%d", scenario.maxAttempts),
		PolicyName: "Runtime Retry Policy", Version: 1, Status: model.RetryPolicyStatusEnabled,
		MaxAttempts: scenario.maxAttempts, InitialDelayMs: 1000, MaxDelayMs: 5000,
		BackoffType: model.RetryBackoffTypeFixed, BackoffMultiplier: 1,
		JitterType: model.RetryJitterTypeNone, JitterRatio: 0, RetryWindowMs: 60000,
		RetryableErrorCategories: datatypes.JSON([]byte("[\"network\",\"remote\",\"timeout\"]")),
		RetryableHTTPStatuses:    datatypes.JSON([]byte("[429,502,503,504]")), RespectRetryAfter: true, Revision: 1,
	}
	credentialID := credential.Id
	policyID := policy.Id
	definition := model.InterfaceDefinition{
		Basic: model.Basic{Id: baseID + 4, State: true}, ExternalSystemID: system.Id,
		InterfaceCode: fmt.Sprintf("runtime_retry_call_%d", scenario.maxAttempts), Name: "Runtime Retry Call", Version: 1,
		Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: scenario.method, RelativePath: "/retry",
		CredentialID: &credentialID, RetryPolicyID: &policyID, TimeoutSeconds: scenario.timeoutSeconds, ResponseLimit: 1024 * 1024,
		InputContract:   datatypes.JSON([]byte("{\"version\":1,\"parameters\":[]}")),
		IdempotencyMode: scenario.idempotencyMode,
		Status:          model.InterfaceDefinitionStatusEnabled, Revision: 1,
	}
	if scenario.idempotencyMode == integration.RemoteIdempotencyKeyHeader {
		definition.RemoteIdempotencyHeader = integration.RemoteIdempotencyHeaderName
	}
	for _, value := range []any{&system, &credential, &policy, &definition} {
		if err := db.Create(value).Error; err != nil {
			t.Fatalf("create retry runtime fixture: %v", err)
		}
	}
	if err := db.Model(&model.RetryPolicy{}).Where("id = ?", policy.Id).Update("jitter_ratio", 0).Error; err != nil {
		t.Fatalf("normalize no-jitter policy fixture: %v", err)
	}

	primary := &database.PrimaryDB{DB: db}
	executions := impl.NewIntegrationExecutionRepositoryImpl(primary)
	logs := impl.NewIntegrationLogRepositoryImpl(primary)
	systems := impl.NewExternalSystemRepositoryImpl(primary)
	interfaces := impl.NewInterfaceDefinitionRepositoryImpl(primary)
	policies := impl.NewRetryPolicyRepositoryImpl(primary)
	credentials := impl.NewCredentialRepositoryImpl(primary)
	snowflake, err := utils.NewSnowflake(3)
	if err != nil {
		t.Fatalf("create retry snowflake: %v", err)
	}
	applicationService := service.NewIntegrationExecutionService(executions, logs, systems, interfaces, policies, snowflake, runtimeAcceptanceAuditWriter{})
	submitContext, cancelSubmit := context.WithCancel(context.Background())
	created, err := applicationService.CreateExecution(submitContext, request.IntegrationExecutionCreateReq{
		ExternalSystemID: system.Id, InterfaceDefinitionID: definition.Id,
		TriggerSource:    model.IntegrationTriggerSourceManual,
		IdempotencyScope: "runtime-retry", IdempotencyKey: fmt.Sprintf("runner-retry-%d", scenario.maxAttempts),
		Input: request.IntegrationExecutionInputReq{},
	})
	cancelSubmit()
	if err != nil {
		var storedPolicy model.RetryPolicy
		var storedDefinition model.InterfaceDefinition
		_ = db.First(&storedPolicy, policy.Id).Error
		_ = db.First(&storedDefinition, definition.Id).Error
		t.Fatalf("submit retry execution: %v policy=%+v definition_retry=%v", err, storedPolicy, storedDefinition.RetryPolicyID)
	}
	provider := integration.NewCredentialProvider(credentials, interfaces, protector)
	guard, err := integration.NewInMemoryConcurrencyGuard(4, 2, 1)
	if err != nil {
		t.Fatalf("create retry concurrency guard: %v", err)
	}
	workerID := fmt.Sprintf("runtime-retry-worker-%d", scenario.maxAttempts)
	engine, err := integration.NewIntegrationExecutionEngine(
		executions, systems, interfaces, credentials, provider, transport, guard, snowflake,
		integration.ExecutionEngineOptions{WorkerID: workerID, LeaseDuration: integration.IntegrationDefaultLeaseDuration, BatchSize: 2},
	)
	if err != nil {
		t.Fatalf("create retry engine: %v", err)
	}
	runner, err := integration.NewIntegrationWorkerRunner(engine, integration.WorkerRunnerConfig{
		Enabled: true, WorkerID: workerID, PollInterval: time.Second, ClaimBatchSize: 2,
		InstanceConcurrency: 2, LeaseRecoveryInterval: 10 * time.Second,
		ShutdownTimeout: 3 * time.Second, LeaseDuration: integration.IntegrationDefaultLeaseDuration,
	})
	if err != nil {
		t.Fatalf("create retry runner: %v", err)
	}
	runnerContext, cancelRunner := context.WithCancel(context.Background())
	defer cancelRunner()
	if err := runner.Start(runnerContext); err != nil {
		t.Fatalf("start retry runner: %v", err)
	}
	t.Cleanup(func() { _ = runner.Stop(context.Background()) })
	waitRuntimeAcceptance(t, 12*time.Second, func() bool {
		var execution model.IntegrationExecution
		return db.First(&execution, created.Id).Error == nil && execution.Status == scenario.expectedStatus && execution.CurrentAttempt == scenario.expectedAttempts
	})
	if err := runner.Stop(context.Background()); err != nil {
		t.Fatalf("stop retry runner: %v", err)
	}
	if calls := int(callCount.Load()); calls != scenario.expectedAttempts {
		t.Fatalf("retry HTTP count = %d, want %d", calls, scenario.expectedAttempts)
	}
	var stored model.IntegrationExecution
	var attempts []model.IntegrationLog
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("load retry execution: %v", err)
	}
	if err := db.Where("execution_id = ?", created.Id).Order("attempt_no ASC").Find(&attempts).Error; err != nil {
		t.Fatalf("load retry attempts: %v", err)
	}
	if first := firstCallAt.Load(); scenario.expectedAttempts > 1 &&
		(first == 0 || secondCallAt.Load()-first < int64(scenario.expectedMinimumDelay)) {
		t.Fatalf("retry was not delayed by persisted next_run_at: first=%d second=%d attempts=%+v", first, secondCallAt.Load(), attempts)
	}
	if scenario.expectRemoteIdempotencyKey {
		first, _ := firstRemoteKey.Load().(string)
		second, _ := secondRemoteKey.Load().(string)
		if first == "" || first != second {
			t.Fatalf("remote idempotency key was not frozen across attempts: first=%q second=%q", first, second)
		}
	}
	if stored.Status != scenario.expectedStatus || stored.CurrentAttempt != scenario.expectedAttempts || stored.NextRunAt != nil || len(attempts) != scenario.expectedAttempts {
		t.Fatalf("unexpected retry convergence: execution=%+v attempts=%+v", stored, attempts)
	}
	for index := range attempts {
		if attempts[index].AttemptNo != index+1 {
			t.Fatalf("attempt sequence is not monotonic: %+v", attempts)
		}
	}
	if scenario.expectedAttempts > 1 && (!attempts[0].Retryable || attempts[0].RetryScheduledAt == nil || attempts[0].RetryReasonCode != integration.RetryReasonAllowed) {
		t.Fatalf("first attempt lacks retry diagnostics: %+v", attempts[0])
	}
	if scenario.expectedRetrySource != "" && attempts[0].RetryAfterSource != scenario.expectedRetrySource {
		t.Fatalf("retry source = %q, want %q", attempts[0].RetryAfterSource, scenario.expectedRetrySource)
	}
}

func openRuntimeAcceptancePostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("integration_runtime_runner_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := db.AutoMigrate(
		&model.ExternalSystem{}, &model.Credential{}, &model.RetryPolicy{}, &model.InterfaceDefinition{},
		&model.IntegrationExecution{}, &model.IntegrationLog{},
	); err != nil {
		t.Fatalf("migrate runtime acceptance schema: %v", err)
	}
	return db
}

func waitRuntimeAcceptance(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("runtime acceptance condition was not satisfied")
}
