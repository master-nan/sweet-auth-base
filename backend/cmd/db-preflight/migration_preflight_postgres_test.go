package main

import (
	migrationstate "backend/internal/migration"
	testutil "backend/internal/test"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPreflightPostgreSQLMigrationLedger(t *testing.T) {
	db := openPreflightPostgreSQL(t)
	ctx := context.Background()

	missing := newReport("test", true)
	checkMigrationLedger(ctx, missing, db, true)
	if len(missing.Problems) != 1 || !strings.Contains(missing.Problems[0], "missing") {
		t.Fatalf("missing ledger problems = %#v", missing.Problems)
	}

	if err := migrationstate.EnsureLedger(ctx, db); err != nil {
		t.Fatalf("ensure migration ledger: %v", err)
	}
	incomplete := newReport("test", true)
	checkMigrationLedger(ctx, incomplete, db, true)
	if len(incomplete.Problems) != 1 || !strings.Contains(incomplete.Problems[0], "incomplete") {
		t.Fatalf("incomplete ledger problems = %#v", incomplete.Problems)
	}
	definitions := migrationstate.Catalog()
	for _, definition := range definitions {
		if _, err := db.ExecContext(ctx, `
INSERT INTO schema_migration (version, "key", checksum)
VALUES ($1, $2, $3)
`, definition.Version, definition.Key, definition.Checksum); err != nil {
			t.Fatalf("insert ledger version %d: %v", definition.Version, err)
		}
	}

	complete := newReport("test", true)
	checkMigrationLedger(ctx, complete, db, true)
	if len(complete.Problems) != 0 || len(complete.Warnings) != 0 {
		t.Fatalf("complete ledger report problems=%#v warnings=%#v", complete.Problems, complete.Warnings)
	}
	expected := int64(len(definitions))
	if complete.Metrics["schema_migration.applied"] != expected || complete.Metrics["schema_migration.expected"] != expected {
		t.Fatalf("complete ledger metrics = %#v", complete.Metrics)
	}

	if _, err := db.ExecContext(ctx, `UPDATE schema_migration SET checksum = $1 WHERE version = 1`, strings.Repeat("0", 64)); err != nil {
		t.Fatalf("corrupt ledger checksum: %v", err)
	}
	corrupt := newReport("test", true)
	checkMigrationLedger(ctx, corrupt, db, true)
	if len(corrupt.Problems) != 1 || !strings.Contains(corrupt.Problems[0], "checksum") {
		t.Fatalf("corrupt ledger problems = %#v", corrupt.Problems)
	}
}

func TestPreflightPostgreSQLRejectsKnownDefaultApplicationSecret(t *testing.T) {
	db := openPreflightPostgreSQL(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
CREATE TABLE application (
    id bigint PRIMARY KEY,
    app_key text NOT NULL,
    app_secret text NOT NULL,
    state boolean NOT NULL,
    gmt_delete timestamp NULL
)
`); err != nil {
		t.Fatalf("create application table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO application (id, app_key, app_secret, state)
VALUES (1, 'sweet-admin', $1, TRUE)
`, "sweet-admin-secret"); err != nil {
		t.Fatalf("insert default application: %v", err)
	}

	t.Setenv("APP_REQUIRE_SECURE_CONFIG", "")
	report := newReport("production", true)
	checkDefaultApplicationSecret(ctx, report, db)
	if len(report.Problems) != 1 {
		t.Fatalf("default application problems = %#v", report.Problems)
	}
	serialized, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("serialize preflight report: %v", err)
	}
	if strings.Contains(string(serialized), "sweet-admin-secret") {
		t.Fatal("preflight problem exposed the known application secret")
	}

	if _, err := db.ExecContext(ctx, `UPDATE application SET app_secret = $1 WHERE id = 1`, "rotated-production-secret-value"); err != nil {
		t.Fatalf("rotate default application secret: %v", err)
	}
	rotated := newReport("production", true)
	checkDefaultApplicationSecret(ctx, rotated, db)
	if len(rotated.Problems) != 0 {
		t.Fatalf("rotated application problems = %#v", rotated.Problems)
	}
}

func openPreflightPostgreSQL(t *testing.T) *sql.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	if err := admin.Ping(); err != nil {
		t.Fatalf("ping PostgreSQL admin connection: %v", err)
	}
	schemaName := fmt.Sprintf("preflight_ledger_%d", time.Now().UnixNano())
	if _, err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA %q`, schemaName)); err != nil {
		t.Fatalf("create PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %q CASCADE`, schemaName))
		_ = admin.Close()
	})

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL test DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
