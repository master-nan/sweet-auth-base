package impl

import (
	"backend/internal/database"
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
	now := model.Now()
	requests := []repository.IntegrationExecutionClaimRequest{
		{WorkerID: "postgres-worker-a", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{7101}},
		{WorkerID: "postgres-worker-b", StartedAt: now, LeaseExpiresAt: now.Add(time.Minute), AttemptIDs: []int{7102}},
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
			claimed, claimErr := repositoryValue.ClaimCreatedExecutions(context.Background(), request)
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
}

func postgresClaimDSN(t *testing.T, dsn string, schemaName string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
