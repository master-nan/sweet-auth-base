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

// OpenSQLite 为不依赖数据库类型的 Repository 和 Service 测试创建隔离的内存数据库。
// PostgreSQL 特有行为需要真实 PostgreSQL 集成测试，不能从此辅助方法推断。
func OpenSQLite(t testing.TB, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open(isolatedSQLiteDSN(t, "standard")),
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

	registerSQLiteCleanup(t, db, 1)

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("migrate test fixtures: %v", err)
		}
	}
	return db
}

// OpenSQLiteWithConfig creates an isolated SQLite database for tests that need
// DryRun, multiple connections, or another explicit GORM configuration.
func OpenSQLiteWithConfig(t testing.TB, config *gorm.Config, models ...interface{}) *gorm.DB {
	t.Helper()
	if config == nil {
		config = &gorm.Config{}
	}
	db, err := gorm.Open(sqlite.Open(isolatedSQLiteDSN(t, "custom")), config)
	if err != nil {
		t.Fatalf("open custom test sqlite database: %v", err)
	}
	registerSQLiteCleanup(t, db, 0)
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("migrate custom test fixtures: %v", err)
		}
	}
	return db
}

func isolatedSQLiteDSN(t testing.TB, purpose string) string {
	t.Helper()
	databaseName := fmt.Sprintf(
		"sweet_test_%s_%s_%d",
		normalizeDatabaseName(t.Name()),
		normalizeDatabaseName(purpose),
		sqliteDatabaseSequence.Add(1),
	)
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", databaseName)
}

func registerSQLiteCleanup(t testing.TB, db *gorm.DB, maxOpenConnections int) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open test sqlite handle: %v", err)
	}
	if maxOpenConnections > 0 {
		sqlDB.SetMaxOpenConns(maxOpenConnections)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
}

// MustCreate 写入显式测试夹具，发生错误时终止当前测试。
// 夹具工厂保留在拥有对应 Model 的包内。
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
