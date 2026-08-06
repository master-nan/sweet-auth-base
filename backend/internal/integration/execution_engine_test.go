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
	"net/http"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestIntegrationExecutionEngineClaimRunAndConverge(t *testing.T) {
	engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusOK)
	defer closeServer()

	claimed, err := engine.ClaimCreatedExecutions(context.Background())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim executions = %+v err=%v", claimed, err)
	}
	if claimed[0].Execution.Status != model.IntegrationExecutionStatusRunning || claimed[0].Attempt.AttemptNo != 1 || claimed[0].Attempt.WorkerID != "runtime-worker-1" {
		t.Fatalf("unexpected claim: %+v", claimed[0])
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
	for _, statusCode := range []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			engine, db, execution, closeServer := newExecutionEngineFixture(t, statusCode)
			defer closeServer()
			claimed, err := engine.ClaimCreatedExecutions(context.Background())
			if err != nil || len(claimed) != 1 {
				t.Fatalf("claim = %+v err=%v", claimed, err)
			}
			result, err := engine.RunExecution(context.Background(), claimed[0])
			if err != nil || !result.RetryEligible || result.Certainty != model.IntegrationResultCertaintyConfirmed {
				t.Fatalf("retry result = %+v err=%v", result, err)
			}
			var stored model.IntegrationExecution
			if err := db.First(&stored, execution.Id).Error; err != nil || stored.Status != model.IntegrationExecutionStatusRetryWaiting {
				t.Fatalf("retry waiting execution = %+v err=%v", stored, err)
			}
		})
	}
	t.Run("expired lease becomes unknown failed without HTTP retry", func(t *testing.T) {
		engine, db, execution, closeServer := newExecutionEngineFixture(t, http.StatusOK)
		defer closeServer()
		claimed, err := engine.ClaimCreatedExecutions(context.Background())
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
	claimed, err := engine.ClaimCreatedExecutions(context.Background())
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

func TestIntegrationExecutionEngineUsesSupportedCredentialTypes(t *testing.T) {
	for _, credentialType := range []string{model.CredentialTypeBasic, model.CredentialTypeAPIKey, model.CredentialTypeBearerToken} {
		t.Run(credentialType, func(t *testing.T) {
			engine, _, _, closeServer := newExecutionEngineFixture(t, http.StatusOK, credentialType)
			defer closeServer()
			claimed, err := engine.ClaimCreatedExecutions(context.Background())
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
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.Credential{}, &model.InterfaceDefinition{}, &model.IntegrationExecution{}, &model.IntegrationLog{})
	client, _, closeServer := newTestTransportClient(t, false, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(responseStatus)
		_, _ = writer.Write([]byte(`{"result":"ok"}`))
	}))
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
		TimeoutSeconds: 15, ResponseLimit: 1024, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	execution := model.IntegrationExecution{Basic: model.Basic{Id: 104, State: true}, ExecutionNo: "INT-RUNTIME-104", ExternalSystemID: system.Id,
		ExternalSystemCode: system.SystemCode, ExternalSystemName: system.Name, InterfaceDefinitionID: definition.Id, InterfaceCode: definition.InterfaceCode,
		InterfaceName: definition.Name, InterfaceVersion: definition.Version, TriggerSource: model.IntegrationTriggerSourceManual,
		Status: model.IntegrationExecutionStatusCreated, IdempotencyScope: "runtime", IdempotencyKey: "execution-104",
		InputHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Revision: 1}
	for _, value := range []any{&system, &credential, &definition, &execution} {
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
		provider, client, guard, sf, ExecutionEngineOptions{WorkerID: "runtime-worker-1", LeaseDuration: time.Minute, BatchSize: 2},
	)
	if err != nil {
		closeServer()
		t.Fatalf("new execution engine: %v", err)
	}
	return engine, db, execution, closeServer
}

var _ repository.CredentialRuntimeIdentity
