package service

import (
	"backend/internal/database"
	apperrors "backend/internal/errors"
	"backend/internal/integration"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"backend/repository/impl"
	"context"
	"errors"
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

func TestIntegrationExecutionPostgreSQLRetryCancelAndClaimRace(t *testing.T) {
	db := openIntegrationExecutionRetryPostgreSQL(t)
	now := model.Now().UTC()
	retrySnapshot := datatypes.JSON([]byte("{\"version\":1,\"policy_code\":\"cancel_claim\",\"policy_version\":1,\"max_attempts\":3,\"initial_delay_ms\":1000,\"max_delay_ms\":1000,\"backoff_type\":\"fixed\",\"backoff_multiplier\":\"1\",\"jitter_type\":\"none\",\"jitter_ratio\":\"0\",\"retry_window_ms\":60000,\"retryable_error_categories\":[\"network\",\"remote\",\"timeout\"],\"retryable_http_statuses\":[429,502,503,504],\"respect_retry_after\":true,\"idempotency_mode\":\"safe_method\",\"remote_idempotency_header\":\"\"}"))
	inputSnapshot := datatypes.JSON([]byte("{\"version\":1,\"path_params\":{},\"query_params\":{},\"headers\":{}}"))
	startedAt := now.Add(-10 * time.Second)
	execution := model.IntegrationExecution{
		Basic: model.Basic{Id: 8101, State: true}, ExecutionNo: "INT-PG-CANCEL-CLAIM",
		ExternalSystemID: 1, ExternalSystemCode: "cancel_claim", ExternalSystemName: "Cancel Claim",
		InterfaceDefinitionID: 2, InterfaceCode: "cancel_claim_api", InterfaceName: "Cancel Claim API", InterfaceVersion: 1,
		TriggerSource: model.IntegrationTriggerSourceManual, Status: model.IntegrationExecutionStatusRetryWaiting,
		IdempotencyScope: "cancel-claim", IdempotencyKey: "cancel-claim",
		InputHash:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		InputSnapshot: inputSnapshot, InputSnapshotVersion: model.IntegrationExecutionInputSnapshotVersion, InputSnapshotSize: len(inputSnapshot),
		RetryPolicySnapshot: retrySnapshot, RetryPolicySnapshotVersion: integration.RetryPolicySnapshotVersion,
		RemoteIdempotencyMode: model.InterfaceIdempotencyModeSafeMethod,
		CurrentAttempt:        1, StartedAt: &startedAt, LastAttemptAt: &startedAt, NextRunAt: &now, Revision: 1,
	}
	firstAttempt := model.IntegrationLog{
		Basic: model.Basic{Id: 8111, State: true}, ExecutionID: execution.Id, AttemptNo: 1,
		Status: model.IntegrationLogStatusFailed, StartedAt: startedAt, EndedAt: &now,
		ResultCertainty: model.IntegrationResultCertaintyConfirmed,
	}
	if err := db.Create(&execution).Error; err != nil {
		t.Fatalf("create retry execution: %v", err)
	}
	if err := db.Create(&firstAttempt).Error; err != nil {
		t.Fatalf("create first attempt: %v", err)
	}
	if err := db.Exec("UPDATE integration_execution SET next_run_at = CURRENT_TIMESTAMP AT TIME ZONE 'UTC' WHERE id = ?", execution.Id).Error; err != nil {
		t.Fatalf("schedule retry with database time: %v", err)
	}

	primary := &database.PrimaryDB{DB: db}
	executionRepository := impl.NewIntegrationExecutionRepositoryImpl(primary)
	sf, err := utils.NewSnowflake(7)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	svc := NewIntegrationExecutionService(
		executionRepository,
		impl.NewIntegrationLogRepositoryImpl(primary),
		impl.NewExternalSystemRepositoryImpl(primary),
		impl.NewInterfaceDefinitionRepositoryImpl(primary),
		impl.NewRetryPolicyRepositoryImpl(primary),
		sf,
		&integrationExecutionAuditWriter{},
	)
	start := make(chan struct{})
	claimResult := make(chan []repository.ClaimedIntegrationExecution, 1)
	claimErrors := make(chan error, 1)
	cancelErrors := make(chan error, 1)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		<-start
		claimed, err := executionRepository.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
			WorkerID: "cancel-claim-worker", StartedAt: now,
			LeaseExpiresAt: now.Add(integration.IntegrationDefaultLeaseDuration), AttemptIDs: []int{8121},
		})
		if err != nil {
			claimErrors <- err
			return
		}
		claimResult <- claimed
	}()
	go func() {
		defer group.Done()
		<-start
		_, err := svc.CancelExecution(context.Background(), execution.Id, execution.Revision)
		cancelErrors <- err
	}()
	close(start)
	group.Wait()
	close(claimResult)
	close(claimErrors)
	close(cancelErrors)
	for err := range claimErrors {
		t.Fatalf("claim retry: %v", err)
	}
	claimed := <-claimResult
	cancelErr := <-cancelErrors
	var stored model.IntegrationExecution
	if err := db.First(&stored, execution.Id).Error; err != nil {
		t.Fatalf("load race result: %v", err)
	}
	var attempts int64
	if err := db.Model(&model.IntegrationLog{}).Where("execution_id = ?", execution.Id).Count(&attempts).Error; err != nil {
		t.Fatalf("count race attempts: %v", err)
	}
	switch {
	case cancelErr == nil:
		if len(claimed) != 0 || stored.Status != model.IntegrationExecutionStatusCancelled || attempts != 1 {
			t.Fatalf("cancel won but claim also progressed: claimed=%d execution=%+v attempts=%d", len(claimed), stored, attempts)
		}
	case errors.Is(cancelErr, apperrors.ErrIntegrationRetryCancelConflict):
		if len(claimed) != 1 || stored.Status != model.IntegrationExecutionStatusRunning || attempts != 2 {
			t.Fatalf("claim won but cancel result inconsistent: claimed=%d execution=%+v attempts=%d err=%v", len(claimed), stored, attempts, cancelErr)
		}
	default:
		t.Fatalf("unexpected cancel/claim race error: %v", cancelErr)
	}
}

func TestIntegrationExecutionPostgreSQLRetryCancelAndClaimOrderedOutcomes(t *testing.T) {
	db := openIntegrationExecutionRetryPostgreSQL(t)
	now := model.Now().UTC()
	primary := &database.PrimaryDB{DB: db}
	executionRepository := impl.NewIntegrationExecutionRepositoryImpl(primary)
	sf, err := utils.NewSnowflake(8)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	svc := NewIntegrationExecutionService(
		executionRepository,
		impl.NewIntegrationLogRepositoryImpl(primary),
		impl.NewExternalSystemRepositoryImpl(primary),
		impl.NewInterfaceDefinitionRepositoryImpl(primary),
		impl.NewRetryPolicyRepositoryImpl(primary),
		sf,
		&integrationExecutionAuditWriter{},
	)
	createFixture := func(id int, executionNo string) model.IntegrationExecution {
		t.Helper()
		retrySnapshot := datatypes.JSON([]byte("{\"version\":1,\"policy_code\":\"cancel_claim\",\"policy_version\":1,\"max_attempts\":3,\"initial_delay_ms\":1000,\"max_delay_ms\":1000,\"backoff_type\":\"fixed\",\"backoff_multiplier\":\"1\",\"jitter_type\":\"none\",\"jitter_ratio\":\"0\",\"retry_window_ms\":60000,\"retryable_error_categories\":[\"network\",\"remote\",\"timeout\"],\"retryable_http_statuses\":[429,502,503,504],\"respect_retry_after\":true,\"idempotency_mode\":\"safe_method\",\"remote_idempotency_header\":\"\"}"))
		inputSnapshot := datatypes.JSON([]byte("{\"version\":1,\"path_params\":{},\"query_params\":{},\"headers\":{}}"))
		startedAt := now.Add(-10 * time.Second)
		execution := model.IntegrationExecution{
			Basic: model.Basic{Id: id, State: true}, ExecutionNo: executionNo,
			ExternalSystemID: 1, ExternalSystemCode: "cancel_claim", ExternalSystemName: "Cancel Claim",
			InterfaceDefinitionID: 2, InterfaceCode: "cancel_claim_api", InterfaceName: "Cancel Claim API", InterfaceVersion: 1,
			TriggerSource: model.IntegrationTriggerSourceManual, Status: model.IntegrationExecutionStatusRetryWaiting,
			IdempotencyScope: executionNo, IdempotencyKey: executionNo,
			InputHash:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			InputSnapshot: inputSnapshot, InputSnapshotVersion: model.IntegrationExecutionInputSnapshotVersion, InputSnapshotSize: len(inputSnapshot),
			RetryPolicySnapshot: retrySnapshot, RetryPolicySnapshotVersion: integration.RetryPolicySnapshotVersion,
			RemoteIdempotencyMode: model.InterfaceIdempotencyModeSafeMethod,
			CurrentAttempt:        1, StartedAt: &startedAt, LastAttemptAt: &startedAt, NextRunAt: &now, Revision: 1,
		}
		attempt := model.IntegrationLog{
			Basic: model.Basic{Id: id + 1, State: true}, ExecutionID: execution.Id, AttemptNo: 1,
			Status: model.IntegrationLogStatusFailed, StartedAt: startedAt, EndedAt: &now,
			ResultCertainty: model.IntegrationResultCertaintyConfirmed,
		}
		if err := db.Create(&execution).Error; err != nil {
			t.Fatalf("create execution fixture: %v", err)
		}
		if err := db.Create(&attempt).Error; err != nil {
			t.Fatalf("create attempt fixture: %v", err)
		}
		if err := db.Exec("UPDATE integration_execution SET next_run_at = CURRENT_TIMESTAMP AT TIME ZONE 'UTC' WHERE id = ?", execution.Id).Error; err != nil {
			t.Fatalf("schedule retry fixture: %v", err)
		}
		return execution
	}

	t.Run("cancel commits first", func(t *testing.T) {
		execution := createFixture(8201, "INT-PG-CANCEL-FIRST")
		if _, err := svc.CancelExecution(context.Background(), execution.Id, execution.Revision); err != nil {
			t.Fatalf("cancel retry: %v", err)
		}
		claimed, err := executionRepository.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
			WorkerID: "cancel-first-worker", StartedAt: now,
			LeaseExpiresAt: now.Add(integration.IntegrationDefaultLeaseDuration), AttemptIDs: []int{8221},
		})
		if err != nil || len(claimed) != 0 {
			t.Fatalf("cancelled execution was claimed: claimed=%+v err=%v", claimed, err)
		}
		var attemptCount int64
		_ = db.Model(&model.IntegrationLog{}).Where("execution_id = ?", execution.Id).Count(&attemptCount).Error
		if attemptCount != 1 {
			t.Fatalf("cancel-first attempt count=%d", attemptCount)
		}
	})

	t.Run("claim commits first", func(t *testing.T) {
		execution := createFixture(8301, "INT-PG-CLAIM-FIRST")
		claimed, err := executionRepository.ClaimReadyExecutions(context.Background(), repository.IntegrationExecutionClaimRequest{
			WorkerID: "claim-first-worker", StartedAt: now,
			LeaseExpiresAt: now.Add(integration.IntegrationDefaultLeaseDuration), AttemptIDs: []int{8321},
		})
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim retry: claimed=%+v err=%v", claimed, err)
		}
		if _, err := svc.CancelExecution(context.Background(), execution.Id, execution.Revision); !errors.Is(err, apperrors.ErrIntegrationRetryCancelConflict) {
			t.Fatalf("claim-first cancel error=%v", err)
		}
		var stored model.IntegrationExecution
		var attemptCount int64
		_ = db.First(&stored, execution.Id).Error
		_ = db.Model(&model.IntegrationLog{}).Where("execution_id = ?", execution.Id).Count(&attemptCount).Error
		if stored.Status != model.IntegrationExecutionStatusRunning || attemptCount != 2 {
			t.Fatalf("claim-first result execution=%+v attempts=%d", stored, attemptCount)
		}
	})
}

func openIntegrationExecutionRetryPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := openPostgresTestDB(t, postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("integration_retry_cancel_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf("CREATE SCHEMA %q", schemaName)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schemaName)).Error })
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	query.Set("TimeZone", "Asia/Shanghai")
	parsed.RawQuery = query.Encode()
	db, err := openPostgresTestDB(t, postgres.Open(parsed.String()), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := db.AutoMigrate(&model.IntegrationExecution{}, &model.IntegrationLog{}); err != nil {
		t.Fatalf("migrate runtime tables: %v", err)
	}
	return db
}
