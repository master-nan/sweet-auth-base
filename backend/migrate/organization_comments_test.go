package main

import (
	"backend/model"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestOrganizationDatabaseCommentSpecsCoverEveryModelColumn(t *testing.T) {
	db := migrateTestDB(t)
	expectedTables := map[string]struct{}{
		"org_legal_entity":   {},
		"org_unit":           {},
		"org_structure":      {},
		"org_structure_node": {},
		"org_position":       {},
		"org_employee":       {},
		"org_assignment":     {},
		"org_sync_batch":     {},
		"org_sync_record":    {},
	}

	if len(organizationDatabaseCommentSpecs) != len(expectedTables) {
		t.Fatalf(
			"organization comment specs=%d, want %d",
			len(organizationDatabaseCommentSpecs),
			len(expectedTables),
		)
	}
	for _, spec := range organizationDatabaseCommentSpecs {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(spec.model); err != nil {
			t.Fatalf("parse organization model: %v", err)
		}
		if _, exists := expectedTables[statement.Schema.Table]; !exists {
			t.Fatalf("unexpected or duplicate organization comment table %s", statement.Schema.Table)
		}
		delete(expectedTables, statement.Schema.Table)
		if strings.TrimSpace(spec.tableComment) == "" {
			t.Fatalf("organization table %s has no table comment", statement.Schema.Table)
		}
		comments, err := completeOrganizationColumnComments(
			statement.Schema.DBNames,
			spec.columnComments,
		)
		if err != nil {
			t.Fatalf("validate organization table %s comments: %v", statement.Schema.Table, err)
		}
		if len(comments) != len(statement.Schema.DBNames) {
			t.Fatalf(
				"organization table %s comments=%d, columns=%d",
				statement.Schema.Table,
				len(comments),
				len(statement.Schema.DBNames),
			)
		}
	}
	if len(expectedTables) != 0 {
		t.Fatalf("organization comment specs missing tables: %v", sortedStringSet(expectedTables))
	}
}

func TestOrganizationDatabaseCommentsAreIdempotentNoopOutsidePostgreSQL(t *testing.T) {
	db := migrateTestDB(t)
	if err := db.AutoMigrate(organizationCommentModels()...); err != nil {
		t.Fatalf("migrate organization models: %v", err)
	}
	before := sqliteOrganizationSchemaSnapshot(t, db)

	if err := applyOrganizationDatabaseComments(db); err != nil {
		t.Fatalf("apply organization comments first time: %v", err)
	}
	if err := applyOrganizationDatabaseComments(db); err != nil {
		t.Fatalf("apply organization comments second time: %v", err)
	}

	after := sqliteOrganizationSchemaSnapshot(t, db)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("organization comment migration changed SQLite schema\nbefore=%v\nafter=%v", before, after)
	}
}

func TestOrganizationDatabaseCommentHelpersRejectMissingAndUnknownColumns(t *testing.T) {
	if _, err := completeOrganizationColumnComments([]string{"id", "unknown"}, nil); err == nil {
		t.Fatal("expected missing column comment to fail")
	}
	if _, err := completeOrganizationColumnComments(
		[]string{"id"},
		map[string]string{"not_a_model_column": "invalid"},
	); err == nil {
		t.Fatal("expected unknown column comment to fail")
	}
	if got := quotePostgresLiteral("组织源's value"); got != "'组织源''s value'" {
		t.Fatalf("quoted PostgreSQL literal=%q", got)
	}
}

func TestOrganizationDatabaseCommentsPersistOnPostgreSQLWithoutSchemaChanges(t *testing.T) {
	dsn := os.Getenv("SWEET_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set SWEET_TEST_POSTGRES_DSN to verify PostgreSQL organization comments")
	}

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}

	schemaName := fmt.Sprintf("org_comment_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create PostgreSQL test schema: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error
	})

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   schemaName + ".",
			SingularTable: true,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
		NowFunc:                                  model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL organization schema: %v", err)
	}
	if err := db.AutoMigrate(organizationCommentModels()...); err != nil {
		t.Fatalf("migrate PostgreSQL organization models: %v", err)
	}
	before := postgresOrganizationSchemaSnapshot(t, db, schemaName)

	if err := applyOrganizationDatabaseComments(db); err != nil {
		t.Fatalf("apply PostgreSQL organization comments first time: %v", err)
	}
	assertPostgresOrganizationComments(t, db, schemaName)
	if err := applyOrganizationDatabaseComments(db); err != nil {
		t.Fatalf("apply PostgreSQL organization comments second time: %v", err)
	}
	assertPostgresOrganizationComments(t, db, schemaName)

	after := postgresOrganizationSchemaSnapshot(t, db, schemaName)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("organization comments changed PostgreSQL structure\nbefore=%v\nafter=%v", before, after)
	}
}

func organizationCommentModels() []any {
	return []any{
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgStructure{},
		&model.OrgStructureNode{},
		&model.OrgPosition{},
		&model.OrgEmployee{},
		&model.OrgAssignment{},
		&model.OrgSyncBatch{},
		&model.OrgSyncRecord{},
	}
}

type sqliteSchemaRow struct {
	Type string
	Name string
	SQL  string
}

func sqliteOrganizationSchemaSnapshot(t *testing.T, db *gorm.DB) []sqliteSchemaRow {
	t.Helper()
	var rows []sqliteSchemaRow
	if err := db.Raw(`
SELECT type, name, COALESCE(sql, '') AS sql
FROM sqlite_master
WHERE name LIKE 'org_%'
ORDER BY type, name
`).Scan(&rows).Error; err != nil {
		t.Fatalf("snapshot SQLite organization schema: %v", err)
	}
	return rows
}

type postgresSchemaRow struct {
	Kind       string
	ObjectName string
	Definition string
}

func postgresOrganizationSchemaSnapshot(
	t *testing.T,
	db *gorm.DB,
	schemaName string,
) []postgresSchemaRow {
	t.Helper()
	tableNames := organizationCommentTableNames(t, db)
	var rows []postgresSchemaRow
	if err := db.Raw(`
SELECT
    'column' AS kind,
    table_name || '.' || column_name AS object_name,
    concat_ws('|',
        ordinal_position::text,
        data_type,
        udt_name,
        is_nullable,
        COALESCE(column_default, ''),
        COALESCE(character_maximum_length::text, ''),
        COALESCE(numeric_precision::text, ''),
        COALESCE(numeric_scale::text, '')
    ) AS definition
FROM information_schema.columns
WHERE table_schema = ? AND table_name IN ?
UNION ALL
SELECT
    'index' AS kind,
    tablename || '.' || indexname AS object_name,
    indexdef AS definition
FROM pg_indexes
WHERE schemaname = ? AND tablename IN ?
UNION ALL
SELECT
    'constraint' AS kind,
    c.relname || '.' || con.conname AS object_name,
    pg_get_constraintdef(con.oid, true) AS definition
FROM pg_constraint con
JOIN pg_class c ON c.oid = con.conrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = ? AND c.relname IN ?
ORDER BY kind, object_name
`, schemaName, tableNames, schemaName, tableNames, schemaName, tableNames).Scan(&rows).Error; err != nil {
		t.Fatalf("snapshot PostgreSQL organization schema: %v", err)
	}
	return rows
}

func assertPostgresOrganizationComments(t *testing.T, db *gorm.DB, schemaName string) {
	t.Helper()
	for _, spec := range organizationDatabaseCommentSpecs {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(spec.model); err != nil {
			t.Fatalf("parse PostgreSQL organization model: %v", err)
		}
		tableName := strings.TrimPrefix(statement.Schema.Table, schemaName+".")

		var tableComment string
		if err := db.Raw(`
SELECT COALESCE(obj_description(c.oid, 'pg_class'), '')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = ? AND c.relname = ?
`, schemaName, tableName).Scan(&tableComment).Error; err != nil {
			t.Fatalf("query PostgreSQL table comment %s: %v", tableName, err)
		}
		if tableComment != spec.tableComment {
			t.Fatalf(
				"PostgreSQL table %s comment=%q, want %q",
				tableName,
				tableComment,
				spec.tableComment,
			)
		}

		comments, err := completeOrganizationColumnComments(
			statement.Schema.DBNames,
			spec.columnComments,
		)
		if err != nil {
			t.Fatalf("validate PostgreSQL column comments for %s: %v", tableName, err)
		}
		for columnName, expected := range comments {
			var actual string
			if err := db.Raw(`
SELECT COALESCE(col_description(c.oid, a.attnum), '')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = c.oid
WHERE n.nspname = ? AND c.relname = ? AND a.attname = ?
`, schemaName, tableName, columnName).Scan(&actual).Error; err != nil {
				t.Fatalf("query PostgreSQL column comment %s.%s: %v", tableName, columnName, err)
			}
			if actual != expected {
				t.Fatalf(
					"PostgreSQL column %s.%s comment=%q, want %q",
					tableName,
					columnName,
					actual,
					expected,
				)
			}
		}
	}
}

func organizationCommentTableNames(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	names := make([]string, 0, len(organizationDatabaseCommentSpecs))
	for _, spec := range organizationDatabaseCommentSpecs {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(spec.model); err != nil {
			t.Fatalf("parse organization model table name: %v", err)
		}
		parts := strings.Split(statement.Schema.Table, ".")
		names = append(names, parts[len(parts)-1])
	}
	sort.Strings(names)
	return names
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
