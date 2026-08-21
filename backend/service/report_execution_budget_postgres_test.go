package service

import (
	myerrors "backend/internal/errors"
	testutil "backend/internal/test"
	"context"
	"errors"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresReportStatementTimeoutIsTransactionLocalAndConnectionReusable(t *testing.T) {
	dsn := testutil.PostgreSQLDSN(t)
	db, err := openPostgresTestDB(t, postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open PostgreSQL pool: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	var originalTimeout string
	var originalBackendPID int
	if err := db.Raw("SHOW statement_timeout").Scan(&originalTimeout).Error; err != nil {
		t.Fatalf("read original statement_timeout: %v", err)
	}
	if err := db.Raw("SELECT pg_backend_pid()").Scan(&originalBackendPID).Error; err != nil {
		t.Fatalf("read original backend pid: %v", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := applyPostgresReportStatementTimeout(tx, 100); err != nil {
			return err
		}
		var localTimeout string
		if err := tx.Raw("SHOW statement_timeout").Scan(&localTimeout).Error; err != nil {
			return err
		}
		if localTimeout != "100ms" {
			t.Fatalf("transaction statement_timeout = %q, want 100ms", localTimeout)
		}
		return tx.Exec("SELECT pg_sleep(1)").Error
	})
	if err == nil {
		t.Fatal("expected PostgreSQL statement timeout")
	}
	normalized := normalizeReportExecutionError(err)
	if !errors.Is(normalized, err) || myerrors.KindOf(normalized) != myerrors.KindTimeout {
		t.Fatalf("timeout error = %v kind=%s", normalized, myerrors.KindOf(normalized))
	}

	var reusableValue int
	var reusableBackendPID int
	var restoredTimeout string
	if err := db.WithContext(context.Background()).Raw("SELECT 1").Scan(&reusableValue).Error; err != nil || reusableValue != 1 {
		t.Fatalf("connection is not reusable after timeout: value=%d err=%v", reusableValue, err)
	}
	if err := db.Raw("SELECT pg_backend_pid()").Scan(&reusableBackendPID).Error; err != nil {
		t.Fatalf("read reusable backend pid: %v", err)
	}
	if reusableBackendPID != originalBackendPID {
		t.Fatalf("backend connection changed after timeout: before=%d after=%d", originalBackendPID, reusableBackendPID)
	}
	if err := db.Raw("SHOW statement_timeout").Scan(&restoredTimeout).Error; err != nil {
		t.Fatalf("read restored statement_timeout: %v", err)
	}
	if restoredTimeout != originalTimeout {
		t.Fatalf("statement_timeout leaked outside transaction: before=%q after=%q", originalTimeout, restoredTimeout)
	}
}
