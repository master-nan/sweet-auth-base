package impl

import (
	"backend/internal/database"
	"backend/internal/integration"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// TestIntegrationExecutionPostgreSQLClaimUsesRowLock 在配置真实 PostgreSQL 时验证多实例领取语义。
// 未配置 SWEET_TEST_POSTGRES_DSN 时由统一测试辅助函数明确跳过。
func TestIntegrationExecutionPostgreSQLClaimUsesRowLock(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("integration_claim_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })

	db, err := gorm.Open(postgres.Open(postgresClaimDSN(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.Default.LogMode(logger.Silent), NowFunc: model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := db.AutoMigrate(&model.IntegrationExecution{}, &model.IntegrationLog{}); err != nil {
		t.Fatalf("migrate runtime tables: %v", err)
	}
	execution := repositoryExecutionFixture(7001, "INT-PG-7001", model.IntegrationExecutionStatusCreated, "pg-claim")
	if err := db.Create(&execution).Error; err != nil {
		t.Fatalf("create execution: %v", err)
	}
	primary := &database.PrimaryDB{DB: db}
	first := NewIntegrationExecutionRepositoryImpl(primary)
	second := NewIntegrationExecutionRepositoryImpl(primary)
	databaseBefore, err := first.CurrentDatabaseTime(context.Background())
	if err != nil {
		t.Fatalf("read database time: %v", err)
	}
	applicationNow := databaseBefore.Add(8 * time.Hour)
	requests := []repository.IntegrationExecutionClaimRequest{
		{WorkerID: "postgres-worker-a", StartedAt: applicationNow, LeaseExpiresAt: applicationNow.Add(integration.IntegrationDefaultLeaseDuration), AttemptIDs: []int{7101}},
		{WorkerID: "postgres-worker-b", StartedAt: applicationNow, LeaseExpiresAt: applicationNow.Add(integration.IntegrationDefaultLeaseDuration), AttemptIDs: []int{7102}},
	}
	start := make(chan struct{})
	var group sync.WaitGroup
	results := make(chan int, 2)
	errorsChannel := make(chan error, 2)
	for index, request := range requests {
		group.Add(1)
		go func(index int, request repository.IntegrationExecutionClaimRequest) {
			defer group.Done()
			<-start
			repositoryValue := first
			if index == 1 {
				repositoryValue = second
			}
			claimed, claimErr := repositoryValue.ClaimReadyExecutions(context.Background(), request)
			if claimErr != nil {
				errorsChannel <- claimErr
				return
			}
			results <- len(claimed)
		}(index, request)
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsChannel)
	for claimErr := range errorsChannel {
		t.Fatalf("claim execution: %v", claimErr)
	}
	claimedTotal := 0
	for count := range results {
		claimedTotal += count
	}
	if claimedTotal != 1 {
		t.Fatalf("PostgreSQL concurrent claim total = %d, want 1", claimedTotal)
	}
	var attempts int64
	if err := db.Model(&model.IntegrationLog{}).Where("execution_id = ?", execution.Id).Count(&attempts).Error; err != nil || attempts != 1 {
		t.Fatalf("attempt count = %d err=%v", attempts, err)
	}
	var running model.IntegrationExecution
	if err := db.First(&running, execution.Id).Error; err != nil {
		t.Fatalf("load claimed execution: %v", err)
	}
	if running.StartedAt == nil || running.StartedAt.Sub(databaseBefore) > 5*time.Second || databaseBefore.Sub(*running.StartedAt) > 5*time.Second {
		t.Fatalf("claim did not use database time: started_at=%v database_before=%v application_now=%v", running.StartedAt, databaseBefore, applicationNow)
	}
	if running.LeaseExpiresAt == nil || running.LeaseExpiresAt.Sub(*running.StartedAt) != integration.IntegrationDefaultLeaseDuration {
		t.Fatalf("claimed lease does not include safety margin: %+v", running.LeaseExpiresAt)
	}
	if candidates, err := first.FindExpiredRunningExecutions(context.Background(), 1); err != nil || len(candidates) != 0 {
		t.Fatalf("valid maximum-duration lease was recovered early: values=%+v err=%v", candidates, err)
	}
	expiredAt := databaseBefore.Add(-time.Second)
	if err := db.Model(&model.IntegrationExecution{}).Where("id = ?", execution.Id).Update("lease_expires_at", expiredAt).Error; err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	var runningAttempt model.IntegrationLog
	if err := db.Where("execution_id = ?", execution.Id).First(&runningAttempt).Error; err != nil {
		t.Fatalf("load running attempt: %v", err)
	}
	if _, err := first.CompleteAttemptAndExecution(context.Background(), repository.IntegrationAttemptCompletion{
		ExecutionID: execution.Id, AttemptID: runningAttempt.Id, AttemptNo: runningAttempt.AttemptNo,
		WorkerID: running.LeaseOwner, ExpectedRevision: running.Revision,
		ExecutionStatus: model.IntegrationExecutionStatusSucceeded,
		CompletedAt:     databaseBefore.Add(-time.Hour), ResultCertainty: model.IntegrationResultCertaintyConfirmed,
	}); err != repository.ErrIntegrationExecutionLeaseLost {
		t.Fatalf("expired lease accepted application completion time: %v", err)
	}
	expiredCandidates, err := first.FindExpiredRunningExecutions(context.Background(), 1)
	if err != nil || len(expiredCandidates) != 1 {
		t.Fatalf("expired lease was not visible: values=%+v err=%v", expiredCandidates, err)
	}
	recovered, err := first.RecoverExpiredExecution(context.Background(), repository.ExpiredExecutionRecovery{ExecutionID: execution.Id})
	if err != nil || !recovered {
		t.Fatalf("recover expired execution = %t err=%v", recovered, err)
	}
	var recoveredExecution model.IntegrationExecution
	var recoveredAttempt model.IntegrationLog
	if err := db.First(&recoveredExecution, execution.Id).Error; err != nil {
		t.Fatalf("load recovered execution: %v", err)
	}
	if err := db.Where("execution_id = ?", execution.Id).First(&recoveredAttempt).Error; err != nil {
		t.Fatalf("load recovered attempt: %v", err)
	}
	if recoveredExecution.Status != model.IntegrationExecutionStatusFailed || recoveredAttempt.ResultCertainty != model.IntegrationResultCertaintyUnknown {
		t.Fatalf("unexpected recovery result: execution=%+v attempt=%+v", recoveredExecution, recoveredAttempt)
	}
}

func TestIntegrationExecutionPostgreSQLRetryClaimUsesDatabaseTimeAndSkipLocked(t *testing.T) {
	db := openPostgresClaimSchema(t, "integration_retry_claim")
	now := model.Now()
	execution := repositoryRetryWaitingFixture(7201, "INT-PG-RETRY-7201", now, now.Add(-10*time.Second), 1, 3)
	firstAttempt := model.IntegrationLog{
		Basic: model.Basic{Id: 7211, State: true}, ExecutionID: execution.Id, AttemptNo: 1,
		Status: model.IntegrationLogStatusFailed, StartedAt: *execution.StartedAt, EndedAt: &now,
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
	repositories := []*IntegrationExecutionRepositoryImpl{
		NewIntegrationExecutionRepositoryImpl(primary),
		NewIntegrationExecutionRepositoryImpl(primary),
	}
	requests := []repository.IntegrationExecutionClaimRequest{
		{WorkerID: "retry-worker-a", StartedAt: now, LeaseExpiresAt: now.Add(integration.IntegrationDefaultLeaseDuration), AttemptIDs: []int{7221}},
		{WorkerID: "retry-worker-b", StartedAt: now, LeaseExpiresAt: now.Add(integration.IntegrationDefaultLeaseDuration), AttemptIDs: []int{7222}},
	}
	start := make(chan struct{})
	results := make(chan []repository.ClaimedIntegrationExecution, 2)
	errorsChannel := make(chan error, 2)
	var group sync.WaitGroup
	for index := range repositories {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			claimed, err := repositories[index].ClaimReadyExecutions(context.Background(), requests[index])
			if err != nil {
				errorsChannel <- err
				return
			}
			results <- claimed
		}(index)
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsChannel)
	for err := range errorsChannel {
		t.Fatalf("claim due retry: %v", err)
	}
	claimedTotal := 0
	for values := range results {
		claimedTotal += len(values)
		if len(values) == 1 && values[0].Attempt.AttemptNo != 2 {
			t.Fatalf("retry attempt number = %d", values[0].Attempt.AttemptNo)
		}
	}
	if claimedTotal != 1 {
		var diagnostic model.IntegrationExecution
		_ = db.First(&diagnostic, execution.Id).Error
		t.Fatalf("PostgreSQL concurrent retry claim total = %d, want 1; execution=%+v", claimedTotal, diagnostic)
	}
	var stored model.IntegrationExecution
	if err := db.First(&stored, execution.Id).Error; err != nil {
		t.Fatalf("load claimed retry: %v", err)
	}
	if stored.Status != model.IntegrationExecutionStatusRunning || stored.CurrentAttempt != 2 || stored.NextRunAt != nil || stored.LeaseOwner == "" || stored.Revision != execution.Revision+1 {
		t.Fatalf("unexpected claimed retry: %+v", stored)
	}
	var attempts int64
	if err := db.Model(&model.IntegrationLog{}).Where("execution_id = ?", execution.Id).Count(&attempts).Error; err != nil || attempts != 2 {
		t.Fatalf("retry attempt count = %d err=%v", attempts, err)
	}
}

func openPostgresClaimSchema(t *testing.T, prefix string) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })
	db, err := gorm.Open(postgres.Open(postgresClaimDSN(t, dsn, schemaName)), &gorm.Config{
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

func postgresClaimDSN(t *testing.T, dsn string, schemaName string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	query.Set("TimeZone", "Asia/Shanghai")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
