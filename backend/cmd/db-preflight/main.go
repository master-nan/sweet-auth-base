package main

import (
	"backend/config"
	"backend/enum"
	"backend/initialize"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

const queryTimeout = 15 * time.Second
const postgresSchema = "public"

type componentStatus struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type preflightReport struct {
	GeneratedAt     string            `json:"generated_at"`
	Environment     string            `json:"environment"`
	RequireMigrated bool              `json:"require_migrated"`
	Components      []componentStatus `json:"components"`
	Metrics         map[string]int64  `json:"metrics"`
	Warnings        []string          `json:"warnings"`
	Problems        []string          `json:"problems"`
}

type metadataPhysicalColumn struct {
	TableCode     string
	FieldCode     string
	FieldType     int
	FieldLength   int64
	IsNull        bool
	FieldCategory string
	ColumnExists  bool
	DataType      string
	ColumnType    string
	IsNullable    bool
	CharMaxLength int64
}

type metadataPhysicalCompatibility struct {
	MissingColumns       int64
	TypeMismatches       int64
	NullableMismatches   int64
	LengthMismatches     int64
	VirtualFieldsSkipped int64
	MissingExamples      []string
	TypeExamples         []string
	NullableExamples     []string
	LengthExamples       []string
}

func main() {
	requireMigrated := parseBoolEnv(os.Getenv("APP_DB_PREFLIGHT_REQUIRE_MIGRATED"))
	report := newReport(os.Getenv("APP_ENV"), requireMigrated)

	cfg, err := initialize.LoadConfig()
	if err != nil {
		report.addProblem("config", fmt.Sprintf("load config failed: %v", err))
		writeReportAndExit(report)
		return
	}
	report.Environment = firstNonEmpty(os.Getenv("APP_ENV"), "dev")
	redactor := newRedactor(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	primaryDB := checkPostgres(ctx, report, redactor, "db_primary", cfg.DBS.Primary)
	if primaryDB != nil {
		defer primaryDB.Close()
		inspectPrimaryDB(ctx, report, primaryDB, cfg, requireMigrated)
	}

	checkRedis(ctx, report, redactor, cfg.Redis)

	writeReportAndExit(report)
}

func newReport(environment string, requireMigrated bool) *preflightReport {
	return &preflightReport{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Environment:     firstNonEmpty(environment, "dev"),
		RequireMigrated: requireMigrated,
		Metrics:         map[string]int64{},
	}
}

func checkPostgres(ctx context.Context, report *preflightReport, redactor func(string) string, name string, dbCfg config.DB) *sql.DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=10", dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.Name)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		message := redactor(err.Error())
		report.addComponent(name, false, message)
		report.addProblem(name, message)
		return nil
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute)
	if err := db.PingContext(ctx); err != nil {
		message := redactor(err.Error())
		report.addComponent(name, false, message)
		report.addProblem(name, message)
		_ = db.Close()
		return nil
	}
	report.addComponent(name, true, fmt.Sprintf("connected to database %s", dbCfg.Name))
	return db
}

func checkRedis(ctx context.Context, report *preflightReport, redactor func(string) string, redisCfg config.Redis) {
	client := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password:        redisCfg.Password,
		DB:              redisCfg.DB,
		PoolSize:        2,
		MinIdleConns:    0,
		ConnMaxIdleTime: time.Duration(redisCfg.ConnMaxIdleTime) * time.Second,
	})
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		message := redactor(err.Error())
		report.addComponent("redis", false, message)
		report.addProblem("redis", message)
		return
	}
	report.addComponent("redis", true, fmt.Sprintf("connected to db %d", redisCfg.DB))
}

func inspectPrimaryDB(ctx context.Context, report *preflightReport, db *sql.DB, cfg *config.Server, requireMigrated bool) {
	tables := requiredPrimaryTables()
	existing, err := existingTables(ctx, db, postgresSchema, tables)
	if err != nil {
		report.addProblem("db_primary_schema", err.Error())
		return
	}

	missing := missingItems(tables, existing)
	if len(missing) > 0 {
		report.addMigratedIssue(requireMigrated, "db_primary_schema", fmt.Sprintf("missing core tables: %s", strings.Join(missing, ", ")))
	}

	checkDatabaseFootprint(ctx, report, db, postgresSchema)

	for _, table := range sortedKeys(seedMinimums()) {
		if !existing[table] {
			continue
		}
		count, err := countTableRows(ctx, db, table)
		if err != nil {
			report.addProblem("db_primary_rows", err.Error())
			continue
		}
		report.Metrics["rows."+table] = count
		if min := seedMinimums()[table]; count < min {
			report.addMigratedIssue(requireMigrated, "db_primary_rows", fmt.Sprintf("%s has %d rows, expected at least %d after migration", table, count, min))
		}
	}

	if existing["access_log"] {
		checkAccessLogIndexes(ctx, report, db, postgresSchema, requireMigrated)
	}
	if existing["sys_dict"] {
		checkRequiredDicts(ctx, report, db, requireMigrated)
	}
	if existing["casbin_rule"] {
		checkCasbinColumnsAndPolicies(ctx, report, db, postgresSchema, requireMigrated)
	}
	if existing["file"] {
		checkDeletedFileUniqueBackfill(ctx, report, db, requireMigrated)
	}
	checkMetadataIntegrity(ctx, report, db, postgresSchema, existing)
}

func requiredPrimaryTables() []string {
	return []string{
		"access_log",
		"application",
		"casbin_rule",
		"file",
		"file_chunk",
		"login_log",
		"sms_log",
		"sms_template",
		"sys_configure",
		"sys_dict",
		"sys_dict_item",
		"sys_menu",
		"sys_menu_button",
		"sys_role",
		"sys_role_menu",
		"sys_role_menu_button",
		"sys_data_dimension_definition",
		"sys_data_resource",
		"sys_data_resource_operation",
		"sys_data_ownership_field",
		"sys_data_policy",
		"sys_data_policy_rule",
		"sys_data_grant",
		"sys_table",
		"sys_table_field",
		"sys_table_index",
		"sys_table_index_field",
		"sys_table_relation",
		"sys_user",
		"sys_user_role",
	}
}

func seedMinimums() map[string]int64 {
	return map[string]int64{
		"application":     1,
		"casbin_rule":     1,
		"sys_configure":   1,
		"sys_dict":        1,
		"sys_dict_item":   1,
		"sys_menu":        1,
		"sys_menu_button": 1,
		"sys_role":        1,
		"sys_table":       1,
		"sys_table_field": 1,
		"sys_user":        1,
	}
}

func existingTables(ctx context.Context, db *sql.DB, schemaName string, tables []string) (map[string]bool, error) {
	result := make(map[string]bool, len(tables))
	if len(tables) == 0 {
		return result, nil
	}
	rows, err := db.QueryContext(ctx, `
SELECT table_name
FROM information_schema.tables
WHERE table_schema = $1 AND table_name = ANY($2)
`, schemaName, pq.Array(tables))
	if err != nil {
		return nil, fmt.Errorf("query primary table inventory: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		result[table] = true
	}
	return result, rows.Err()
}

func checkDatabaseFootprint(ctx context.Context, report *preflightReport, db *sql.DB, schemaName string) {
	var dataLength sql.NullInt64
	var indexLength sql.NullInt64
	err := db.QueryRowContext(ctx, `
SELECT
  COALESCE(SUM(pg_total_relation_size(format('%I.%I', schemaname, tablename)::regclass)), 0),
  COALESCE(SUM(pg_indexes_size(format('%I.%I', schemaname, tablename)::regclass)), 0)
FROM pg_tables
WHERE schemaname = $1
`, schemaName).Scan(&dataLength, &indexLength)
	if err != nil {
		report.addWarning("db_primary_footprint", fmt.Sprintf("could not inspect database footprint: %v", err))
		return
	}
	report.Metrics["db_primary.data_mb"] = bytesToMB(dataLength.Int64)
	report.Metrics["db_primary.index_mb"] = bytesToMB(indexLength.Int64)
}

func countTableRows(ctx context.Context, db *sql.DB, table string) (int64, error) {
	if !safeIdentifier(table) {
		return 0, fmt.Errorf("unsafe table identifier %q", table)
	}
	var count int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count rows for %s: %w", table, err)
	}
	return count, nil
}

func checkAccessLogIndexes(ctx context.Context, report *preflightReport, db *sql.DB, schemaName string, requireMigrated bool) {
	required := []string{
		"idx_access_log_time",
		"idx_access_log_action_time",
		"idx_access_log_resource_time",
		"idx_access_log_success_time",
	}
	existing, err := existingIndexes(ctx, db, schemaName, "access_log", required)
	if err != nil {
		report.addProblem("access_log_indexes", err.Error())
		return
	}
	missing := missingItems(required, existing)
	if len(missing) > 0 {
		report.addMigratedIssue(requireMigrated, "access_log_indexes", fmt.Sprintf("missing access_log indexes: %s", strings.Join(missing, ", ")))
	}
}

func existingIndexes(ctx context.Context, db *sql.DB, schemaName string, table string, indexes []string) (map[string]bool, error) {
	result := make(map[string]bool, len(indexes))
	if len(indexes) == 0 {
		return result, nil
	}
	rows, err := db.QueryContext(ctx, `
SELECT indexname
FROM pg_indexes
WHERE schemaname = $1 AND tablename = $2 AND indexname = ANY($3)
`, schemaName, table, pq.Array(indexes))
	if err != nil {
		return nil, fmt.Errorf("query indexes for %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var index string
		if err := rows.Scan(&index); err != nil {
			return nil, err
		}
		result[index] = true
	}
	return result, rows.Err()
}

func checkRequiredDicts(ctx context.Context, report *preflightReport, db *sql.DB, requireMigrated bool) {
	required := []string{
		"http_method",
		"sys_menu_button_position",
		"sys_master_detail_mode",
		"sys_form_open_mode",
		"sys_detail_open_mode",
		"sys_table_field_input_type",
		"sys_table_field_type",
		"sys_table_relation_type",
		"sys_table_type",
		"whether",
	}
	rows, err := db.QueryContext(ctx, `
SELECT dict_code
FROM sys_dict
WHERE dict_code = ANY($1) AND gmt_delete IS NULL
`, pq.Array(required))
	if err != nil {
		report.addProblem("system_dicts", fmt.Sprintf("query system dicts: %v", err))
		return
	}
	defer rows.Close()

	existing := map[string]bool{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			report.addProblem("system_dicts", err.Error())
			return
		}
		existing[code] = true
	}
	missing := missingItems(required, existing)
	if len(missing) > 0 {
		report.addMigratedIssue(requireMigrated, "system_dicts", fmt.Sprintf("missing system dictionaries: %s", strings.Join(missing, ", ")))
	}
}

func checkCasbinColumnsAndPolicies(ctx context.Context, report *preflightReport, db *sql.DB, schemaName string, requireMigrated bool) {
	columns, err := existingColumns(ctx, db, schemaName, "casbin_rule", []string{"ptype", "p_type", "v1", "v2"})
	if err != nil {
		report.addProblem("casbin_rule", err.Error())
		return
	}
	if !columns["ptype"] && columns["p_type"] {
		report.addMigratedIssue(requireMigrated, "casbin_rule", "casbin_rule still has legacy p_type column without normalized ptype")
		return
	}
	if !columns["ptype"] || !columns["v1"] || !columns["v2"] {
		report.addMigratedIssue(requireMigrated, "casbin_rule", "casbin_rule is missing ptype/v1/v2 columns required by strict permission checks")
		return
	}

	requiredPolicies := [][2]string{
		{"/admin/application/:id/rotate-secret", "POST"},
		{"/admin/log/access/query", "POST"},
		{"/admin/table/code/:code", "GET"},
	}
	missing := make([]string, 0)
	for _, policy := range requiredPolicies {
		var count int64
		err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM casbin_rule
WHERE ptype = 'p' AND v1 = $1 AND v2 = $2
`, policy[0], policy[1]).Scan(&count)
		if err != nil {
			report.addProblem("casbin_policies", fmt.Sprintf("query policy %s %s: %v", policy[1], policy[0], err))
			return
		}
		if count == 0 {
			missing = append(missing, policy[1]+" "+policy[0])
		}
	}
	if len(missing) > 0 {
		report.addMigratedIssue(requireMigrated, "casbin_policies", fmt.Sprintf("missing critical route policies: %s", strings.Join(missing, ", ")))
	}
}

func checkDeletedFileUniqueBackfill(ctx context.Context, report *preflightReport, db *sql.DB, requireMigrated bool) {
	var count int64
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM file
WHERE gmt_delete IS NOT NULL
  AND (
    file_md5 NOT LIKE ('%#deleted-' || id::text)
    OR file_uuid NOT LIKE ('%#deleted-' || id::text)
  )
`).Scan(&count)
	if err != nil {
		report.addProblem("file.deleted_unique_backfill", fmt.Sprintf("query soft-deleted file unique keys: %v", err))
		return
	}
	report.Metrics["file.deleted_unique_backfill_pending"] = count
	if count > 0 {
		report.addMigratedIssue(requireMigrated, "file.deleted_unique_backfill", fmt.Sprintf("%d soft-deleted files still need unique-key backfill", count))
	}
}

func existingColumns(ctx context.Context, db *sql.DB, schemaName string, table string, columns []string) (map[string]bool, error) {
	result := make(map[string]bool, len(columns))
	rows, err := db.QueryContext(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2 AND column_name = ANY($3)
`, schemaName, table, pq.Array(columns))
	if err != nil {
		return nil, fmt.Errorf("query columns for %s: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		result[column] = true
	}
	return result, rows.Err()
}

func checkMetadataIntegrity(ctx context.Context, report *preflightReport, db *sql.DB, schemaName string, existing map[string]bool) {
	if existing["sys_table"] && existing["sys_table_field"] {
		countAndWarn(ctx, report, db, "metadata.tables_without_fields", `
SELECT COUNT(*)
FROM sys_table t
LEFT JOIN sys_table_field f ON f.table_id = t.id AND f.gmt_delete IS NULL
WHERE t.gmt_delete IS NULL AND f.id IS NULL
`, "published metadata contains tables without fields")
		countAndWarn(ctx, report, db, "metadata.orphan_fields", `
SELECT COUNT(*)
FROM sys_table_field f
LEFT JOIN sys_table t ON t.id = f.table_id AND t.gmt_delete IS NULL
WHERE f.gmt_delete IS NULL AND t.id IS NULL
`, "metadata contains fields whose table no longer exists")
		countMissingPhysicalTables(ctx, report, db, schemaName)
		checkMetadataPhysicalColumns(ctx, report, db, schemaName, report.RequireMigrated)
	}
	if existing["sys_table_field"] && existing["sys_dict"] {
		countAndWarn(ctx, report, db, "metadata.fields_with_missing_dict", `
SELECT COUNT(*)
FROM sys_table_field f
LEFT JOIN sys_dict d ON d.dict_code = f.dict_code AND d.gmt_delete IS NULL
WHERE f.gmt_delete IS NULL AND f.dict_code IS NOT NULL AND f.dict_code <> '' AND d.id IS NULL
`, "metadata contains fields referencing missing dictionaries")
	}
	if existing["sys_role_menu_button"] && existing["sys_menu_button"] {
		countAndWarn(ctx, report, db, "permissions.orphan_role_buttons", `
SELECT COUNT(*)
FROM sys_role_menu_button rb
LEFT JOIN sys_menu_button b ON b.id = rb.button_id AND b.gmt_delete IS NULL
WHERE b.id IS NULL
`, "role button permissions reference missing buttons")
	}
}

func countMissingPhysicalTables(ctx context.Context, report *preflightReport, db *sql.DB, schemaName string) {
	var count int64
	err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM sys_table t
LEFT JOIN information_schema.tables it ON it.table_schema = $1 AND it.table_name = t.table_code
WHERE t.gmt_delete IS NULL AND t.table_type = 1 AND it.table_name IS NULL
`, schemaName).Scan(&count)
	if err != nil {
		report.addProblem("metadata.missing_physical_tables", fmt.Sprintf("query missing physical tables: %v", err))
		return
	}
	report.Metrics["metadata.missing_physical_tables"] = count
	if count > 0 {
		report.addWarning("metadata.missing_physical_tables", fmt.Sprintf("%d low-code metadata tables have no matching physical table", count))
	}
}

func checkMetadataPhysicalColumns(ctx context.Context, report *preflightReport, db *sql.DB, schemaName string, requireMigrated bool) {
	rows, err := db.QueryContext(ctx, `
SELECT
  t.table_code,
  f.field_code,
  f.field_type,
  COALESCE(f.field_length, 0),
  CASE WHEN f.is_null THEN 1 ELSE 0 END,
  COALESCE(f.field_category, ''),
  c.column_name,
  c.data_type,
  COALESCE(c.udt_name, c.data_type),
  c.is_nullable,
  c.character_maximum_length
FROM sys_table t
JOIN sys_table_field f ON f.table_id = t.id AND f.gmt_delete IS NULL
LEFT JOIN information_schema.columns c
  ON c.table_schema = $1 AND c.table_name = t.table_code AND c.column_name = f.field_code
WHERE t.gmt_delete IS NULL AND t.table_type = 1
ORDER BY t.table_code, f.field_code
`, schemaName)
	if err != nil {
		report.addProblem("metadata.physical_columns", fmt.Sprintf("query metadata physical columns: %v", err))
		return
	}
	defer rows.Close()

	columns := make([]metadataPhysicalColumn, 0)
	for rows.Next() {
		var column metadataPhysicalColumn
		var isNull int
		var columnName sql.NullString
		var dataType sql.NullString
		var columnType sql.NullString
		var isNullable sql.NullString
		var charMaxLength sql.NullInt64
		if err := rows.Scan(
			&column.TableCode,
			&column.FieldCode,
			&column.FieldType,
			&column.FieldLength,
			&isNull,
			&column.FieldCategory,
			&columnName,
			&dataType,
			&columnType,
			&isNullable,
			&charMaxLength,
		); err != nil {
			report.addProblem("metadata.physical_columns", fmt.Sprintf("scan metadata physical columns: %v", err))
			return
		}
		column.IsNull = isNull == 1
		column.ColumnExists = columnName.Valid
		column.DataType = strings.ToLower(strings.TrimSpace(dataType.String))
		column.ColumnType = strings.ToLower(strings.TrimSpace(columnType.String))
		column.IsNullable = strings.EqualFold(strings.TrimSpace(isNullable.String), "YES")
		if charMaxLength.Valid {
			column.CharMaxLength = charMaxLength.Int64
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		report.addProblem("metadata.physical_columns", fmt.Sprintf("read metadata physical columns: %v", err))
		return
	}

	result := analyzeMetadataPhysicalColumns(columns)
	report.Metrics["metadata.fields_missing_physical_column"] = result.MissingColumns
	report.Metrics["metadata.fields_type_mismatch"] = result.TypeMismatches
	report.Metrics["metadata.fields_nullable_mismatch"] = result.NullableMismatches
	report.Metrics["metadata.fields_length_mismatch"] = result.LengthMismatches
	report.Metrics["metadata.virtual_fields_skipped"] = result.VirtualFieldsSkipped

	if result.MissingColumns > 0 {
		report.addMigratedIssue(
			requireMigrated,
			"metadata.fields_missing_physical_column",
			fmt.Sprintf("%d metadata fields have no matching physical column; examples: %s", result.MissingColumns, strings.Join(result.MissingExamples, ", ")),
		)
	}
	if result.TypeMismatches > 0 {
		report.addMigratedIssue(
			requireMigrated,
			"metadata.fields_type_mismatch",
			fmt.Sprintf("%d metadata fields have incompatible physical column types; examples: %s", result.TypeMismatches, strings.Join(result.TypeExamples, ", ")),
		)
	}
	if result.NullableMismatches > 0 {
		report.addWarning(
			"metadata.fields_nullable_mismatch",
			fmt.Sprintf("%d metadata fields have nullable flag drift from physical columns; examples: %s", result.NullableMismatches, strings.Join(result.NullableExamples, ", ")),
		)
	}
	if result.LengthMismatches > 0 {
		report.addMigratedIssue(
			requireMigrated,
			"metadata.fields_length_mismatch",
			fmt.Sprintf("%d varchar metadata fields exceed physical column length; examples: %s", result.LengthMismatches, strings.Join(result.LengthExamples, ", ")),
		)
	}
}

func analyzeMetadataPhysicalColumns(columns []metadataPhysicalColumn) metadataPhysicalCompatibility {
	var result metadataPhysicalCompatibility
	for _, column := range columns {
		if shouldSkipPhysicalColumnCheck(column.FieldCategory) {
			result.VirtualFieldsSkipped++
			continue
		}
		field := column.TableCode + "." + column.FieldCode
		if !column.ColumnExists {
			result.MissingColumns++
			result.MissingExamples = appendExample(result.MissingExamples, field)
			continue
		}
		if !metadataFieldTypeCompatible(column.FieldType, column.DataType, column.ColumnType) {
			result.TypeMismatches++
			result.TypeExamples = appendExample(result.TypeExamples, fmt.Sprintf("%s metadata=%d physical=%s", field, column.FieldType, firstNonEmpty(column.ColumnType, column.DataType)))
		}
		if column.IsNull != column.IsNullable {
			result.NullableMismatches++
			result.NullableExamples = appendExample(result.NullableExamples, fmt.Sprintf("%s metadata_nullable=%t physical_nullable=%t", field, column.IsNull, column.IsNullable))
		}
		if column.FieldType == 3 && column.FieldLength > 0 && column.CharMaxLength > 0 && column.CharMaxLength < column.FieldLength {
			result.LengthMismatches++
			result.LengthExamples = appendExample(result.LengthExamples, fmt.Sprintf("%s metadata_length=%d physical_length=%d", field, column.FieldLength, column.CharMaxLength))
		}
	}
	return result
}

func shouldSkipPhysicalColumnCheck(fieldCategory string) bool {
	category := strings.ToLower(strings.TrimSpace(fieldCategory))
	return category == "virtual_field" || category == "calculated_field"
}

func metadataFieldTypeCompatible(fieldType int, dataType string, columnType string) bool {
	dataType = strings.ToLower(strings.TrimSpace(dataType))
	columnType = strings.ToLower(strings.TrimSpace(columnType))
	switch fieldType {
	case int(enum.BigIntFieldType):
		return dataType == "bigint"
	case int(enum.VarcharFieldType):
		return inStringSet(dataType, "character varying", "character", "varchar", "char")
	case int(enum.TextFieldType):
		return strings.Contains(dataType, "text")
	case int(enum.BooleanFieldType):
		return dataType == "tinyint" || dataType == "boolean" || strings.Contains(columnType, "tinyint(1)")
	case int(enum.DateFieldType):
		return dataType == "date"
	case int(enum.DatetimeFieldType):
		return dataType == "timestamp without time zone" || dataType == "timestamp with time zone" || dataType == "timestamp"
	case int(enum.TimeFieldType):
		return dataType == "time"
	case int(enum.JsonFieldType):
		return dataType == "json" || dataType == "jsonb"
	case int(enum.IntFieldType):
		return inStringSet(dataType, "int", "integer", "mediumint", "smallint")
	case int(enum.SmallIntFieldType):
		return dataType == "smallint"
	case int(enum.DecimalFieldType):
		return inStringSet(dataType, "numeric", "decimal")
	default:
		return false
	}
}

func inStringSet(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}

func appendExample(examples []string, value string) []string {
	if len(examples) >= 5 {
		return examples
	}
	return append(examples, value)
}

func countAndWarn(ctx context.Context, report *preflightReport, db *sql.DB, metric string, query string, warning string) {
	var count int64
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		report.addProblem(metric, fmt.Sprintf("query %s: %v", metric, err))
		return
	}
	report.Metrics[metric] = count
	if count > 0 {
		report.addWarning(metric, fmt.Sprintf("%s: %d", warning, count))
	}
}

func missingItems(required []string, existing map[string]bool) []string {
	missing := make([]string, 0)
	for _, item := range required {
		if !existing[item] {
			missing = append(missing, item)
		}
	}
	sort.Strings(missing)
	return missing
}

func sortedKeys(values map[string]int64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func safeIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func bytesToMB(value int64) int64 {
	if value <= 0 {
		return 0
	}
	return value / 1024 / 1024
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func newRedactor(cfg *config.Server) func(string) string {
	secrets := []string{}
	if cfg != nil {
		secrets = append(secrets,
			cfg.DBS.Primary.Password,
			cfg.Redis.Password,
			cfg.Session.Secret,
			cfg.Conf.Salt,
			cfg.Upload.OSS.AccessKeyID,
			cfg.Upload.OSS.AccessKeySecret,
			cfg.ALiYun.SMS.AccessKeyId,
			cfg.ALiYun.SMS.AccessKeySecret,
		)
	}
	return func(value string) string {
		for _, secret := range secrets {
			secret = strings.TrimSpace(secret)
			if secret == "" {
				continue
			}
			value = strings.ReplaceAll(value, secret, "[redacted]")
		}
		return value
	}
}

func (r *preflightReport) addComponent(name string, ok bool, message string) {
	r.Components = append(r.Components, componentStatus{Name: name, OK: ok, Message: message})
}

func (r *preflightReport) addProblem(code string, message string) {
	r.Problems = append(r.Problems, code+": "+message)
}

func (r *preflightReport) addWarning(code string, message string) {
	r.Warnings = append(r.Warnings, code+": "+message)
}

func (r *preflightReport) addMigratedIssue(requireMigrated bool, code string, message string) {
	if requireMigrated {
		r.addProblem(code, message)
		return
	}
	r.addWarning(code, message)
}

func writeReportAndExit(report *preflightReport) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
	if len(report.Problems) > 0 {
		os.Exit(1)
	}
}
