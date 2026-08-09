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
		&model.ExternalSystem{}, &model.Credential{}, &model.InterfaceDefinition{},
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
