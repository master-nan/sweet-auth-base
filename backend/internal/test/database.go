package testutil

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"backend/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var sqliteDatabaseSequence atomic.Uint64

// OpenSQLite creates an isolated in-memory database for database-agnostic
// repository and service tests. PostgreSQL-specific behavior requires a real
// PostgreSQL integration test and must not be inferred from this helper.
func OpenSQLite(t testing.TB, models ...interface{}) *gorm.DB {
	t.Helper()

	databaseName := fmt.Sprintf(
		"sweet_test_%s_%d",
		normalizeDatabaseName(t.Name()),
		sqliteDatabaseSequence.Add(1),
	)
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", databaseName)),
		&gorm.Config{
			NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
			NowFunc:                                  model.Now,
		},
	)
	if err != nil {
		t.Fatalf("open test sqlite database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open test sqlite handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("migrate test fixtures: %v", err)
		}
	}
	return db
}

// MustCreate writes an explicit test fixture and fails the current test on
// error. Fixture factories remain local to the package that owns the model.
func MustCreate(t testing.TB, db *gorm.DB, value interface{}) {
	t.Helper()
	if db == nil {
		t.Fatal("test database is required")
	}
	if value == nil {
		t.Fatal("test fixture is required")
	}
	if err := db.Create(value).Error; err != nil {
		t.Fatalf("create test fixture: %v", err)
	}
}

// WithRollback runs a test section in a transaction and always rolls it back.
// It is for fixture isolation, not for testing production transaction helpers.
func WithRollback(t testing.TB, db *gorm.DB, run func(tx *gorm.DB)) {
	t.Helper()
	if db == nil {
		t.Fatal("test database is required")
	}
	if run == nil {
		t.Fatal("rollback callback is required")
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test rollback transaction: %v", tx.Error)
	}
	rolledBack := false
	defer func() {
		if !rolledBack {
			_ = tx.Rollback().Error
		}
	}()

	run(tx)
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback test transaction: %v", err)
	}
	rolledBack = true
}

func normalizeDatabaseName(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(value) {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		default:
			builder.WriteByte('_')
		}
	}
	name := strings.Trim(builder.String(), "_")
	if name == "" {
		return "database"
	}
	return name
}
