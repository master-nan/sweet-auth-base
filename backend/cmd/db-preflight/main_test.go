package main

import (
	"backend/config"
	"reflect"
	"testing"
)

func TestMissingItemsSorted(t *testing.T) {
	got := missingItems([]string{"sys_user", "sys_menu", "sys_table"}, map[string]bool{
		"sys_user": true,
	})
	want := []string{"sys_menu", "sys_table"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missingItems() = %#v, want %#v", got, want)
	}
}

func TestRequiredPrimaryTablesUsesCompleteManagedCatalog(t *testing.T) {
	tables := requiredPrimaryTables()
	if len(tables) != 56 {
		t.Fatalf("required primary tables = %d, want 56", len(tables))
	}
	required := map[string]bool{
		"org_assignment":                   false,
		"report_definition_version":        false,
		"integration_interface_definition": false,
		"query_scheme_role":                false,
		"sys_menu_button_template":         false,
		"notification":                     false,
		"notification_recipient":           false,
		"sys_user_session":                 false,
	}
	for _, table := range tables {
		if _, exists := required[table]; exists {
			required[table] = true
		}
	}
	for table, found := range required {
		if !found {
			t.Errorf("required primary tables missing %s", table)
		}
	}
}

func TestAddMigratedIssue(t *testing.T) {
	warnReport := newReport("production", false)
	warnReport.addMigratedIssue(false, "schema", "missing table")
	if len(warnReport.Warnings) != 1 || len(warnReport.Problems) != 0 {
		t.Fatalf("expected migrated issue to be warning, got warnings=%#v problems=%#v", warnReport.Warnings, warnReport.Problems)
	}

	strictReport := newReport("production", true)
	strictReport.addMigratedIssue(true, "schema", "missing table")
	if len(strictReport.Problems) != 1 || len(strictReport.Warnings) != 0 {
		t.Fatalf("expected migrated issue to be problem, got warnings=%#v problems=%#v", strictReport.Warnings, strictReport.Problems)
	}
}

func TestSafeIdentifier(t *testing.T) {
	for _, value := range []string{"sys_user", "access_log_2026", "A1"} {
		if !safeIdentifier(value) {
			t.Fatalf("expected %q to be a safe identifier", value)
		}
	}
	for _, value := range []string{"", "sys-user", "sys_user;drop"} {
		if safeIdentifier(value) {
			t.Fatalf("expected %q to be unsafe", value)
		}
	}
}

func TestNewRedactorHidesConfiguredSecrets(t *testing.T) {
	redact := newRedactor(&config.Server{
		DBS: config.DBS{
			Primary: config.DB{Password: "primary-secret"},
		},
		Redis:   config.Redis{Password: "redis-secret"},
		Session: config.Session{Secret: "session-secret"},
		Conf:    config.Conf{Salt: "salt-secret"},
	})

	got := redact("primary-secret redis-secret visible")
	want := "[redacted] [redacted] visible"
	if got != want {
		t.Fatalf("redact() = %q, want %q", got, want)
	}
}

func TestParseBoolEnv(t *testing.T) {
	if !parseBoolEnv(" yes ") {
		t.Fatal("expected yes to parse true")
	}
	if parseBoolEnv("false") {
		t.Fatal("expected false to parse false")
	}
}

func TestSecurePreflightMode(t *testing.T) {
	t.Setenv("APP_REQUIRE_SECURE_CONFIG", "")
	for _, environment := range []string{"pro", "prod", "production"} {
		if !securePreflightMode(environment) {
			t.Fatalf("expected %q to enable secure preflight", environment)
		}
	}
	if securePreflightMode("dev") {
		t.Fatal("expected dev without secure override to remain non-secure")
	}
	t.Setenv("APP_REQUIRE_SECURE_CONFIG", "true")
	if !securePreflightMode("dev") {
		t.Fatal("expected secure override to enable secure preflight")
	}
}

func TestMetadataFieldTypeCompatible(t *testing.T) {
	tests := []struct {
		name       string
		fieldType  int
		dataType   string
		columnType string
		want       bool
	}{
		{name: "bigint", fieldType: 1, dataType: "bigint", want: true},
		{name: "boolean", fieldType: 5, dataType: "boolean", want: true},
		{name: "time", fieldType: 8, dataType: "time", want: true},
		{name: "integer", fieldType: 11, dataType: "integer", want: true},
		{name: "varchar", fieldType: 3, dataType: "character varying", want: true},
		{name: "datetime", fieldType: 7, dataType: "timestamp with time zone", want: true},
		{name: "datetime without timezone", fieldType: 7, dataType: "timestamp without time zone", want: false},
		{name: "jsonb", fieldType: 10, dataType: "jsonb", want: true},
		{name: "varchar mismatch", fieldType: 3, dataType: "integer", want: false},
		{name: "datetime mismatch", fieldType: 7, dataType: "date", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metadataFieldTypeCompatible(tt.fieldType, tt.dataType, tt.columnType)
			if got != tt.want {
				t.Fatalf("metadataFieldTypeCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeMetadataPhysicalColumns(t *testing.T) {
	result := analyzeMetadataPhysicalColumns([]metadataPhysicalColumn{
		{
			TableCode:     "demo",
			FieldCode:     "name",
			FieldType:     3,
			FieldLength:   64,
			FieldCategory: "normal_field",
			ColumnExists:  true,
			DataType:      "character varying",
			ColumnType:    "varchar",
			CharMaxLength: 32,
		},
		{
			TableCode:     "demo",
			FieldCode:     "status",
			FieldType:     9,
			FieldCategory: "normal_field",
			ColumnExists:  true,
			DataType:      "text",
			ColumnType:    "text",
		},
		{
			TableCode:     "demo",
			FieldCode:     "missing_col",
			FieldType:     11,
			FieldCategory: "normal_field",
			ColumnExists:  false,
		},
		{
			TableCode:     "demo",
			FieldCode:     "calc",
			FieldType:     11,
			FieldCategory: "calculated_field",
			ColumnExists:  false,
		},
		{
			TableCode:     "demo",
			FieldCode:     "enabled",
			FieldType:     5,
			FieldCategory: "normal_field",
			ColumnExists:  true,
			DataType:      "boolean",
			ColumnType:    "bool",
			IsNull:        false,
			IsNullable:    true,
		},
	})

	if result.MissingColumns != 1 {
		t.Fatalf("MissingColumns = %d, want 1", result.MissingColumns)
	}
	if result.TypeMismatches != 1 {
		t.Fatalf("TypeMismatches = %d, want 1", result.TypeMismatches)
	}
	if result.NullableMismatches != 1 {
		t.Fatalf("NullableMismatches = %d, want 1", result.NullableMismatches)
	}
	if result.LengthMismatches != 1 {
		t.Fatalf("LengthMismatches = %d, want 1", result.LengthMismatches)
	}
	if result.VirtualFieldsSkipped != 1 {
		t.Fatalf("VirtualFieldsSkipped = %d, want 1", result.VirtualFieldsSkipped)
	}
}
