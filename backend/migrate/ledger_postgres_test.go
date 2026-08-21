package main

import (
	"backend/config"
	migrationstate "backend/internal/migration"
	testutil "backend/internal/test"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestMigrationLedgerPostgreSQLFreshAndRerun(t *testing.T) {
	db := openMigrationLedgerPostgreSQL(t)
	if err := migrateSchema(db); err != nil {
		t.Fatalf("fresh migrate: %v", err)
	}
	first := loadMigrationLedgerEntries(t, db)
	if err := migrationstate.ValidateLedger(first, migrationstate.Catalog(), true); err != nil {
		t.Fatalf("validate fresh ledger: %v", err)
	}
	if len(first) != 15 {
		t.Fatalf("fresh ledger has %d entries, want 15", len(first))
	}
	assertAllManagedTablesExist(t, db)
	assertAccessLogOperationalIndexes(t, db)

	if err := migrateSchema(db); err != nil {
		t.Fatalf("rerun migrate: %v", err)
	}
	second := loadMigrationLedgerEntries(t, db)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("rerun changed ledger\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func assertAccessLogOperationalIndexes(t *testing.T, db *gorm.DB) {
	t.Helper()
	var count int64
	if err := db.Raw(`
SELECT COUNT(*)
FROM pg_indexes
WHERE schemaname = current_schema()
  AND tablename = 'access_log'
  AND indexname IN (
    'idx_access_log_time',
    'idx_access_log_action_time',
    'idx_access_log_resource_time',
    'idx_access_log_success_time'
  )
`).Scan(&count).Error; err != nil {
		t.Fatalf("inspect access log indexes: %v", err)
	}
	if count != 4 {
		t.Fatalf("access log operational indexes = %d, want 4", count)
	}
}

func TestMigrationLedgerPostgreSQLAdoptCanonicalAndPartial(t *testing.T) {
	for _, test := range []struct {
		name      string
		stepCount int
	}{
		{name: "canonical", stepCount: len(migrationSteps())},
		{name: "partial", stepCount: 7},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openMigrationLedgerPostgreSQL(t)
			steps := migrationSteps()
			if err := runUntrackedMigrationSteps(db, steps[:test.stepCount]); err != nil {
				t.Fatalf("prepare %s database: %v", test.name, err)
			}
			if err := migrateSchema(db); err == nil || !strings.Contains(err.Error(), "migrate adopt") {
				t.Fatalf("ordinary migrate should require explicit adopt, got %v", err)
			}
			if err := adoptSchema(db); err != nil {
				t.Fatalf("adopt %s database: %v", test.name, err)
			}
			entries := loadMigrationLedgerEntries(t, db)
			if err := migrationstate.ValidateLedger(entries, migrationstate.Catalog(), true); err != nil {
				t.Fatalf("validate adopted ledger: %v", err)
			}
			assertAllManagedTablesExist(t, db)
		})
	}
}

func TestMigrationLedgerPostgreSQLFailedStepIsNotRecorded(t *testing.T) {
	db := openMigrationLedgerPostgreSQL(t)
	steps := []migrationStep{
		newTestMigrationStep(1, "successful_step", "test|successful-step|committed", func(tx *gorm.DB) error {
			return tx.Exec(`CREATE TABLE ledger_success_probe (id bigint PRIMARY KEY)`).Error
		}),
		newTestMigrationStep(2, "failed_step", "test|failed-step|transaction-rollback", func(tx *gorm.DB) error {
			if err := tx.Exec(`CREATE TABLE ledger_failure_probe (id bigint PRIMARY KEY)`).Error; err != nil {
				return err
			}
			return errors.New("injected migration failure")
		}),
	}
	if err := runMigrationStepsWithMode(db, steps, []string{"ledger_success_probe", "ledger_failure_probe"}, false); err == nil {
		t.Fatal("expected migration failure")
	}
	entries := loadMigrationLedgerEntries(t, db)
	if len(entries) != 1 || entries[0].Key != "successful_step" {
		t.Fatalf("ledger after failed step = %#v", entries)
	}
	var successExists bool
	var failureExists bool
	if err := db.Raw(`SELECT to_regclass('ledger_success_probe') IS NOT NULL`).Scan(&successExists).Error; err != nil {
		t.Fatalf("inspect successful-step table: %v", err)
	}
	if err := db.Raw(`SELECT to_regclass('ledger_failure_probe') IS NOT NULL`).Scan(&failureExists).Error; err != nil {
		t.Fatalf("inspect failed-step table: %v", err)
	}
	if !successExists {
		t.Fatal("successful migration transaction was not committed")
	}
	if failureExists {
		t.Fatal("failed migration transaction left its table behind")
	}
}

func TestMigrationLedgerPostgreSQLChecksumMismatchFailsClosed(t *testing.T) {
	db := openMigrationLedgerPostgreSQL(t)
	if err := migrateSchema(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec(`UPDATE schema_migration SET checksum = ? WHERE version = 1`, strings.Repeat("0", 64)).Error; err != nil {
		t.Fatalf("corrupt checksum: %v", err)
	}
	if err := migrateSchema(db); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum failure, got %v", err)
	}
}

func TestMigrationLedgerPostgreSQLConcurrentMigrateSerializes(t *testing.T) {
	db := openMigrationLedgerPostgreSQL(t)
	start := make(chan struct{})
	errorsByRunner := make(chan error, 2)
	var runners sync.WaitGroup
	for i := 0; i < 2; i++ {
		runners.Add(1)
		go func() {
			defer runners.Done()
			<-start
			errorsByRunner <- migrateSchema(db.Session(&gorm.Session{}))
		}()
	}
	close(start)
	runners.Wait()
	close(errorsByRunner)
	for err := range errorsByRunner {
		if err != nil {
			t.Fatalf("concurrent migrate: %v", err)
		}
	}
	entries := loadMigrationLedgerEntries(t, db)
	if err := migrationstate.ValidateLedger(entries, migrationstate.Catalog(), true); err != nil {
		t.Fatalf("validate concurrent ledger: %v", err)
	}
}

func TestMigrationLedgerPostgreSQLSeedUsesSharedAdvisoryLock(t *testing.T) {
	db := openMigrationLedgerPostgreSQL(t)
	if err := migrateSchema(db); err != nil {
		t.Fatalf("migrate before seed: %v", err)
	}

	lockHeld := make(chan struct{})
	releaseLock := make(chan struct{})
	lockDone := make(chan error, 1)
	go func() {
		lockDone <- migrationstate.WithAdvisoryLock(context.Background(), db, func(_ *gorm.DB) error {
			close(lockHeld)
			<-releaseLock
			return nil
		})
	}()
	<-lockHeld

	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_BOOTSTRAP_ADMIN_PASSWORD", "Migration-Ledger-Test-2026!")
	cfg := &config.Server{}
	cfg.Conf.Salt = "migration-ledger-seed-test-salt"
	sf := newMigrationTestSnowflake(t)
	seedDone := make(chan error, 1)
	go func() {
		seedDone <- seedAllData(db, cfg, sf)
	}()

	var prematureSeedErr error
	seedCompletedEarly := false
	select {
	case prematureSeedErr = <-seedDone:
		seedCompletedEarly = true
	case <-time.After(150 * time.Millisecond):
	}
	close(releaseLock)
	if err := <-lockDone; err != nil {
		t.Fatalf("release held advisory lock: %v", err)
	}
	if seedCompletedEarly {
		t.Fatalf("seed completed while shared advisory lock was held: %v", prematureSeedErr)
	}
	if err := <-seedDone; err != nil {
		t.Fatalf("seed after advisory lock release: %v", err)
	}
}

func openMigrationLedgerPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := openPostgresTestDB(t, postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	schemaName := fmt.Sprintf("migration_ledger_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA %q`, schemaName)).Error; err != nil {
		t.Fatalf("create PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		_ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE`, schemaName)).Error
	})

	db, err := openPostgresTestDB(t, postgres.Open(postgresDSNWithSearchPath(t, dsn, schemaName)), &gorm.Config{
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open isolated PostgreSQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(8)
	return db
}

func loadMigrationLedgerEntries(t *testing.T, db *gorm.DB) []migrationstate.LedgerEntry {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open PostgreSQL handle: %v", err)
	}
	entries, exists, err := migrationstate.LoadLedger(context.Background(), sqlDB)
	if err != nil {
		t.Fatalf("load migration ledger: %v", err)
	}
	if !exists {
		t.Fatal("migration ledger does not exist")
	}
	return entries
}

func assertAllManagedTablesExist(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open PostgreSQL handle: %v", err)
	}
	tables, err := migrationstate.ExistingManagedTables(context.Background(), sqlDB, migrationstate.ManagedTables())
	if err != nil {
		t.Fatalf("inspect managed tables: %v", err)
	}
	if len(tables) != len(migrationstate.ManagedTables()) {
		t.Fatalf("managed tables present = %d, want %d", len(tables), len(migrationstate.ManagedTables()))
	}
}

func newTestMigrationStep(version int64, key string, contract string, run func(*gorm.DB) error) migrationStep {
	digest := sha256.Sum256([]byte(contract))
	return migrationStep{
		version:  version,
		name:     key,
		contract: contract,
		checksum: hex.EncodeToString(digest[:]),
		run:      run,
	}
}
