package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type LedgerEntry struct {
	Version   int64
	Key       string
	Checksum  string
	AppliedAt time.Time
}

type Queryer interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Execer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

func EnsureLedger(ctx context.Context, execer Execer) error {
	_, err := execer.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migration (
    version bigint PRIMARY KEY,
    "key" varchar(128) NOT NULL UNIQUE,
    checksum char(64) NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_schema_migration_checksum CHECK (checksum ~ '^[0-9a-f]{64}$')
)
`)
	if err != nil {
		return fmt.Errorf("ensure migration ledger: %w", err)
	}
	return nil
}

func LoadLedger(ctx context.Context, queryer Queryer) ([]LedgerEntry, bool, error) {
	var exists bool
	if err := queryer.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = current_schema() AND table_name = $1
)
`, LedgerTable).Scan(&exists); err != nil {
		return nil, false, fmt.Errorf("inspect migration ledger: %w", err)
	}
	if !exists {
		return nil, false, nil
	}

	rows, err := queryer.QueryContext(ctx, `
SELECT version, "key", checksum, applied_at
FROM schema_migration
ORDER BY version
`)
	if err != nil {
		return nil, true, fmt.Errorf("load migration ledger: %w", err)
	}
	defer rows.Close()

	entries := make([]LedgerEntry, 0)
	for rows.Next() {
		var entry LedgerEntry
		if err := rows.Scan(&entry.Version, &entry.Key, &entry.Checksum, &entry.AppliedAt); err != nil {
			return nil, true, fmt.Errorf("scan migration ledger: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, true, fmt.Errorf("read migration ledger: %w", err)
	}
	return entries, true, nil
}

func ExistingManagedTables(ctx context.Context, queryer Queryer, tables []string) ([]string, error) {
	if len(tables) == 0 {
		return nil, nil
	}
	rows, err := queryer.QueryContext(ctx, `
SELECT table_name
FROM information_schema.tables
WHERE table_schema = current_schema() AND table_name = ANY($1)
ORDER BY table_name
`, pq.Array(tables))
	if err != nil {
		return nil, fmt.Errorf("inspect managed tables: %w", err)
	}
	defer rows.Close()

	existing := make([]string, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("scan managed table: %w", err)
		}
		existing = append(existing, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read managed tables: %w", err)
	}
	return existing, nil
}

func ValidateLedger(entries []LedgerEntry, definitions []Definition, requireComplete bool) error {
	if err := ValidateCatalog(definitions); err != nil {
		return err
	}
	if len(entries) > len(definitions) {
		return fmt.Errorf("migration ledger contains %d entries but catalog contains %d; unknown migration present", len(entries), len(definitions))
	}
	for i, entry := range entries {
		expected := definitions[i]
		if entry.Version != expected.Version {
			return fmt.Errorf("migration ledger order invalid at position %d: version %d, expected %d", i, entry.Version, expected.Version)
		}
		if entry.Key != expected.Key {
			return fmt.Errorf("migration ledger key mismatch at version %d: %q, expected %q; unknown migration present", entry.Version, entry.Key, expected.Key)
		}
		if entry.Checksum != expected.Checksum {
			return fmt.Errorf("migration ledger checksum mismatch at version %d (%s)", entry.Version, entry.Key)
		}
	}
	if requireComplete && len(entries) != len(definitions) {
		return fmt.Errorf("migration ledger incomplete: applied %d of %d migrations", len(entries), len(definitions))
	}
	return nil
}

func WithAdvisoryLock(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	if db.Dialector.Name() != "postgres" {
		return fn(db.WithContext(ctx))
	}
	return db.WithContext(ctx).Connection(func(conn *gorm.DB) (returnErr error) {
		pool := conn.Statement.ConnPool
		var lockKey int64
		if err := pool.QueryRowContext(ctx, `
SELECT hashtextextended(
    'sweet-auth-base:schema-migration:' || current_database() || ':' || current_schema(),
    0
)
`).Scan(&lockKey); err != nil {
			return fmt.Errorf("derive migration advisory lock: %w", err)
		}
		if _, err := pool.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
			return fmt.Errorf("acquire migration advisory lock: %w", err)
		}
		defer func() {
			unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_, unlockErr := pool.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", lockKey)
			if unlockErr == nil {
				return
			}
			unlockErr = fmt.Errorf("release migration advisory lock: %w", unlockErr)
			if returnErr == nil {
				returnErr = unlockErr
			} else {
				returnErr = errors.Join(returnErr, unlockErr)
			}
		}()
		return fn(conn.Session(&gorm.Session{NewDB: true}))
	})
}
