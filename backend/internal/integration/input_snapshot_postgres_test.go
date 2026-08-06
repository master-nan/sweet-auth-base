package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	myerrors "backend/internal/errors"
	testutil "backend/internal/test"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type inputSnapshotPostgreSQLRow struct {
	ID           int            `gorm:"column:id"`
	Snapshot     datatypes.JSON `gorm:"column:input_snapshot"`
	SemanticSize int            `gorm:"column:semantic_size"`
	InputHash    string         `gorm:"column:input_hash"`
}

func TestExecutionInputSnapshotPostgreSQLJSONBSemanticRoundTrip(t *testing.T) {
	db := openInputSnapshotPostgreSQL(t)
	contract := snapshotTestContract()
	input := ExecutionInputValues{
		PathParams:  map[string]string{"employee_id": "10001"},
		QueryParams: map[string][]string{"tags": {"south", "east"}, "page": {"1"}},
		Headers:     map[string][]string{"X-Correlation-ID": {"runtime-roundtrip"}},
		JSONBody:    json.RawMessage(` { "name":"\u5f20\u4e09", "items":[2,1], "active":true } `),
	}
	_, canonical, hash, err := BuildExecutionInputSnapshot(contract, "POST", "/api/employees/{employee_id}", 3, input)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO execution_input_snapshot_roundtrip (id, input_snapshot, semantic_size, input_hash)
		VALUES (1, ?::jsonb, ?, ?)
	`, string(canonical), len(canonical), hash).Error; err != nil {
		t.Fatalf("insert JSONB snapshot: %v", err)
	}

	row := loadInputSnapshotPostgreSQLRow(t, db)
	if bytes.Equal(row.Snapshot, canonical) || len(row.Snapshot) == len(canonical) {
		t.Fatalf("test fixture did not expose JSONB byte rewrite: stored=%q canonical=%q", row.Snapshot, canonical)
	}
	loaded, err := LoadExecutionInputSnapshot(
		contract, "POST", "/api/employees/{employee_id}", 3,
		row.Snapshot, row.SemanticSize, row.InputHash,
	)
	if err != nil {
		t.Fatalf("load JSONB semantic snapshot: %v", err)
	}
	if loaded.PathParams["employee_id"] != "10001" ||
		strings.Join(loaded.QueryParams["tags"], ",") != "east,south" ||
		loaded.Headers["x-correlation-id"][0] != "runtime-roundtrip" ||
		!bytes.Contains(loaded.JSONBody, []byte(`"name":"张三"`)) {
		t.Fatalf("unexpected normalized snapshot: %+v body=%s", loaded, loaded.JSONBody)
	}

	tamperCases := []struct {
		name string
		path string
		json string
	}{
		{name: "path", path: "{path_params,employee_id}", json: `"10002"`},
		{name: "query", path: "{query_params,page}", json: `["2"]`},
		{name: "header", path: "{headers,x-correlation-id}", json: `["runtime-tamperedx"]`},
		{name: "body", path: "{json_body,name}", json: `"李四"`},
	}
	for _, test := range tamperCases {
		t.Run(test.name+" tamper", func(t *testing.T) {
			if err := db.Exec(`
				UPDATE execution_input_snapshot_roundtrip
				SET input_snapshot = jsonb_set(?::jsonb, ?::text[], ?::jsonb, false)
				WHERE id = 1
			`, string(row.Snapshot), test.path, test.json).Error; err != nil {
				t.Fatalf("tamper JSONB: %v", err)
			}
			tampered := loadInputSnapshotPostgreSQLRow(t, db)
			_, err := LoadExecutionInputSnapshot(
				contract, "POST", "/api/employees/{employee_id}", 3,
				tampered.Snapshot, row.SemanticSize, row.InputHash,
			)
			if !errors.Is(err, myerrors.ErrIntegrationExecutionInputHashMismatch) {
				t.Fatalf("tampered snapshot error=%v", err)
			}
		})
	}

	if _, err := LoadExecutionInputSnapshot(
		contract, "POST", "/api/employees/{employee_id}", 3,
		row.Snapshot, row.SemanticSize+1, row.InputHash,
	); !errors.Is(err, myerrors.ErrIntegrationExecutionInputSizeMismatch) {
		t.Fatalf("semantic size mismatch error=%v", err)
	}
	versionTampered := bytes.Replace(row.Snapshot, []byte(`"version": 1`), []byte(`"version": 2`), 1)
	if _, err := LoadExecutionInputSnapshot(
		contract, "POST", "/api/employees/{employee_id}", 3,
		versionTampered, row.SemanticSize, row.InputHash,
	); !errors.Is(err, myerrors.ErrIntegrationExecutionInputVersionUnsupported) {
		t.Fatalf("snapshot version error=%v", err)
	}
	storageOversized := []byte(`{"version":1,"path_params":{},"query_params":{},"headers":{},"padding":"` +
		strings.Repeat("x", MaxInputSnapshotStorageBytes) + `"}`)
	if _, err := LoadExecutionInputSnapshot(nil, "GET", "/api/static", 1, storageOversized, 1, hash); !errors.Is(err, myerrors.ErrIntegrationExecutionInputStorageTooLarge) {
		t.Fatalf("storage limit error=%v", err)
	}
}

func openInputSnapshotPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testutil.PostgreSQLDSN(t)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("integration_input_snapshot_%d", time.Now().UnixNano())
	if err := admin.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() { _ = admin.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error })

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE execution_input_snapshot_roundtrip (
			id integer PRIMARY KEY,
			input_snapshot jsonb NOT NULL,
			semantic_size integer NOT NULL,
			input_hash varchar(64) NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("create roundtrip table: %v", err)
	}
	return db
}

func loadInputSnapshotPostgreSQLRow(t *testing.T, db *gorm.DB) inputSnapshotPostgreSQLRow {
	t.Helper()
	var row inputSnapshotPostgreSQLRow
	if err := db.Raw(`
		SELECT id, input_snapshot, semantic_size, input_hash
		FROM execution_input_snapshot_roundtrip WHERE id = 1
	`).Scan(&row).Error; err != nil {
		t.Fatalf("load JSONB snapshot: %v", err)
	}
	return row
}
