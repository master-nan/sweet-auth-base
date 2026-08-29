package main

import (
	"testing"
	"time"
)

func TestCanonicalTimeAndIDContractPostgreSQL(t *testing.T) {
	db := openMigrationLedgerPostgreSQL(t)
	if err := db.Exec(`
CREATE TABLE canonical_contract_probe (
  id bigserial PRIMARY KEY,
  happened_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  label text NOT NULL
);
INSERT INTO canonical_contract_probe (id, happened_at, label)
VALUES (42, '2026-08-28 10:30:00', 'existing');
`).Error; err != nil {
		t.Fatalf("create legacy contract fixture: %v", err)
	}

	if err := migrateCanonicalTimeAndIDContract(db); err != nil {
		t.Fatalf("migrate canonical contract: %v", err)
	}
	if err := migrateCanonicalTimeAndIDContract(db); err != nil {
		t.Fatalf("repeat canonical contract migration: %v", err)
	}

	var dataType string
	var columnDefault *string
	if err := db.Raw(`
SELECT data_type, column_default
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = 'canonical_contract_probe'
  AND column_name = 'happened_at'
`).Row().Scan(&dataType, &columnDefault); err != nil {
		t.Fatalf("inspect timestamp column: %v", err)
	}
	if dataType != "timestamp with time zone" || columnDefault == nil {
		t.Fatalf("timestamp contract type=%q default=%v", dataType, columnDefault)
	}

	var happenedAt time.Time
	if err := db.Raw(`SELECT happened_at FROM canonical_contract_probe WHERE id = 42`).Scan(&happenedAt).Error; err != nil {
		t.Fatalf("read converted timestamp: %v", err)
	}
	if want := time.Date(2026, 8, 28, 2, 30, 0, 0, time.UTC); !happenedAt.Equal(want) {
		t.Fatalf("converted instant = %s, want %s", happenedAt, want)
	}

	var idDefault *string
	var sequenceName *string
	if err := db.Raw(`
SELECT column_default,
       pg_get_serial_sequence(format('%I.%I', table_schema, table_name), column_name)
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = 'canonical_contract_probe'
  AND column_name = 'id'
`).Row().Scan(&idDefault, &sequenceName); err != nil {
		t.Fatalf("inspect ID column: %v", err)
	}
	if idDefault != nil || sequenceName != nil {
		t.Fatalf("generated ID remains: default=%v sequence=%v", idDefault, sequenceName)
	}
	if err := db.Exec(`INSERT INTO canonical_contract_probe (label) VALUES ('missing-id')`).Error; err == nil {
		t.Fatal("database accepted a row without an application-generated ID")
	}
	if err := db.Exec(`INSERT INTO canonical_contract_probe (id, label) VALUES (492717468770816, 'snowflake')`).Error; err != nil {
		t.Fatalf("database rejected explicit snowflake ID: %v", err)
	}
}
