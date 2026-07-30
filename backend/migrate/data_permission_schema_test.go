package main

import (
	"backend/model"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestDataPermissionDomainMigrationIsIdempotentAndComplete(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema: %v", err)
	}
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema twice: %v", err)
	}

	expectedColumns := map[any][]string{
		&model.DataDimensionDefinition{}: {"code", "category", "value_type", "provider_code"},
		&model.DataResource{}: {
			"resource_code",
			"resource_type",
			"table_id",
			"service_code",
			"report_definition_id",
			"permission_enabled",
		},
		&model.DataResourceOperation{}: {"resource_id", "operation", "permission_enabled"},
		&model.DataOwnershipField{}: {
			"resource_id",
			"ownership_code",
			"dimension_id",
			"binding_type",
			"table_field_id",
			"adapter_field_code",
			"value_type",
		},
		&model.DataPolicy{}: {"code", "policy_type"},
		&model.DataPolicyRule{}: {
			"policy_id",
			"sequence",
			"dimension_id",
			"ownership_code",
			"scope_source",
			"relation",
			"operator",
			"specified_values",
			"structure_code",
		},
		&model.DataGrant{}: {
			"subject_type",
			"subject_id",
			"resource_id",
			"operation",
			"policy_id",
			"valid_from",
			"valid_to",
		},
	}
	for modelValue, columns := range expectedColumns {
		if !db.Migrator().HasTable(modelValue) {
			t.Errorf("missing migrated table for %T", modelValue)
		}
		for _, column := range columns {
			if !db.Migrator().HasColumn(modelValue, column) {
				t.Errorf("missing migrated column %T.%s", modelValue, column)
			}
		}
	}

	resource := model.DataResource{
		ResourceCode:      "migration-default",
		Name:              "Migration Default",
		ResourceType:      model.DataResourceTypeBusinessService,
		ServiceCode:       stringPointer("migration-default"),
		AdapterCode:       "registered",
		PermissionEnabled: false,
	}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatalf("create data resource with default permission setting: %v", err)
	}
	var persisted model.DataResource
	if err := db.Where("resource_code = ?", resource.ResourceCode).First(&persisted).Error; err != nil {
		t.Fatalf("reload data resource: %v", err)
	}
	if persisted.PermissionEnabled {
		t.Fatal("data resource permission_enabled must default to false")
	}
}

func TestDataPermissionDomainMigrationEnforcesPortableConstraints(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema: %v", err)
	}

	dimension := model.DataDimensionDefinition{
		Basic:        model.Basic{Id: 101},
		Code:         "legal_entity",
		Name:         "Legal Entity",
		Category:     model.DataDimensionCategoryOrganization,
		ValueType:    model.DataDimensionValueTypeBigint,
		ProviderCode: "organization",
	}
	if err := db.Create(&dimension).Error; err != nil {
		t.Fatalf("create dimension: %v", err)
	}
	policy := model.DataPolicy{
		Basic:      model.Basic{Id: 201},
		Code:       "same-sequence",
		Name:       "Same Sequence",
		PolicyType: model.DataPolicyTypeRuleSet,
	}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("create policy: %v", err)
	}
	firstRule := validDataPolicyRule(policy.Id, dimension.Id, 1)
	if err := db.Create(&firstRule).Error; err != nil {
		t.Fatalf("create first policy rule: %v", err)
	}
	secondRule := validDataPolicyRule(policy.Id, dimension.Id, 1)
	if err := db.Create(&secondRule).Error; err == nil {
		t.Fatal("expected duplicate policy_id and sequence to be rejected")
	}

	invalidOwnership := model.DataOwnershipField{
		ResourceId:       1,
		OwnershipCode:    "legal_entity_id",
		DimensionId:      dimension.Id,
		BindingType:      "report_source",
		AdapterFieldCode: stringPointer("legal_entity_id"),
		ValueType:        model.DataDimensionValueTypeBigint,
	}
	if err := db.Create(&invalidOwnership).Error; err == nil {
		t.Fatal("expected report_source ownership binding to be rejected")
	}
}

func TestDataPermissionDomainMigrationPostgreSQLConstraints(t *testing.T) {
	dsn := os.Getenv("SWEET_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set SWEET_TEST_POSTGRES_DSN to verify PostgreSQL data permission constraints")
	}

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	schemaName := fmt.Sprintf("data_permission_migration_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create PostgreSQL test schema: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error
	})

	isolatedDSN := postgresDSNWithSearchPath(t, dsn, schemaName)
	db, err := gorm.Open(postgres.Open(isolatedDSN), &gorm.Config{
		NamingStrategy:                           schema.NamingStrategy{SingularTable: true},
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
		NowFunc:                                  model.Now,
	})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL data permission schema: %v", err)
	}
	createDataPermissionPostgresPrerequisites(t, db, schemaName)

	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate PostgreSQL data permission schema: %v", err)
	}
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate PostgreSQL data permission schema twice: %v", err)
	}

	assertPostgresDataPermissionConstraints(t, db, schemaName)
	assertPostgresDataPermissionConstraintBehavior(t, db)
	assertPostgresDataPermissionIntegrity(t, db)
}

func createDataPermissionPostgresPrerequisites(t *testing.T, db *gorm.DB, schemaName string) {
	t.Helper()
	for _, sql := range []string{
		fmt.Sprintf(`CREATE TABLE "%s"."sys_table" ("id" bigint PRIMARY KEY)`, schemaName),
		fmt.Sprintf(`CREATE TABLE "%s"."sys_table_field" ("id" bigint PRIMARY KEY)`, schemaName),
		fmt.Sprintf(`CREATE TABLE "%s"."report_definition" ("id" bigint PRIMARY KEY)`, schemaName),
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create PostgreSQL prerequisite: %v", err)
		}
	}
}

func postgresDSNWithSearchPath(t *testing.T, dsn string, schemaName string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL test DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func assertPostgresDataPermissionConstraints(t *testing.T, db *gorm.DB, schemaName string) {
	t.Helper()
	for _, tableName := range []string{
		"sys_data_dimension_definition",
		"sys_data_resource",
		"sys_data_resource_operation",
		"sys_data_ownership_field",
		"sys_data_policy",
		"sys_data_policy_rule",
		"sys_data_grant",
	} {
		var exists bool
		if err := db.Raw(`
SELECT EXISTS (
	SELECT 1 FROM information_schema.tables
	WHERE table_schema = ? AND table_name = ?
)`,
			schemaName,
			tableName,
		).Scan(&exists).Error; err != nil {
			t.Fatalf("inspect PostgreSQL table %s: %v", tableName, err)
		}
		if !exists {
			t.Errorf("missing PostgreSQL table %s", tableName)
		}
	}

	for _, constraintName := range []string{
		"chk_data_dimension_category",
		"chk_data_dimension_value_type",
		"chk_data_resource_type",
		"chk_data_resource_target",
		"chk_data_resource_operation",
		"chk_data_ownership_binding_type",
		"chk_data_ownership_value_type",
		"chk_data_ownership_binding_target",
		"chk_data_policy_type",
		"chk_data_policy_rule_scope_source",
		"chk_data_policy_rule_relation",
		"chk_data_policy_rule_operator",
		"chk_data_policy_rule_sequence",
		"chk_data_policy_rule_specified_values",
		"chk_data_policy_rule_structure",
		"chk_data_grant_subject_type",
		"chk_data_grant_operation",
		"chk_data_grant_valid_range",
		"fk_data_resource_table",
		"fk_data_resource_report_definition",
		"fk_data_resource_operation_resource",
		"fk_data_ownership_field_resource",
		"fk_data_ownership_field_dimension",
		"fk_data_ownership_field_table_field",
		"fk_data_policy_rule_policy",
		"fk_data_policy_rule_dimension",
		"fk_data_grant_resource",
		"fk_data_grant_policy",
	} {
		var count int64
		if err := db.Raw(`
SELECT COUNT(*)
FROM pg_constraint constraint_definition
JOIN pg_namespace namespace_definition
  ON namespace_definition.oid = constraint_definition.connamespace
WHERE namespace_definition.nspname = ?
  AND constraint_definition.conname = ?
`,
			schemaName,
			constraintName,
		).Scan(&count).Error; err != nil {
			t.Fatalf("inspect PostgreSQL constraint %s: %v", constraintName, err)
		}
		if count != 1 {
			t.Errorf("PostgreSQL constraint %s count=%d, want 1", constraintName, count)
		}
	}

	var indexDefinition string
	if err := db.Raw(
		`SELECT indexdef FROM pg_indexes WHERE schemaname = ? AND indexname = ?`,
		schemaName,
		"uni_data_policy_rule_sequence",
	).Scan(&indexDefinition).Error; err != nil {
		t.Fatalf("inspect PostgreSQL policy rule unique index: %v", err)
	}
	if !strings.Contains(strings.ToUpper(indexDefinition), "UNIQUE") ||
		!strings.Contains(strings.ToUpper(indexDefinition), " WHERE ") {
		t.Fatalf("unexpected PostgreSQL policy rule unique index: %s", indexDefinition)
	}

	assertPostgresColumnDefinition(
		t,
		db,
		schemaName,
		"sys_data_policy_rule",
		"dimension_id",
		"bigint",
		"NO",
		"",
	)
	assertPostgresColumnDefinition(
		t,
		db,
		schemaName,
		"sys_data_policy_rule",
		"specified_values",
		"jsonb",
		"YES",
		"",
	)
	assertPostgresColumnDefinition(
		t,
		db,
		schemaName,
		"sys_data_resource",
		"permission_enabled",
		"boolean",
		"NO",
		"false",
	)
}

func assertPostgresColumnDefinition(
	t *testing.T,
	db *gorm.DB,
	schemaName string,
	tableName string,
	columnName string,
	dataType string,
	nullable string,
	defaultContains string,
) {
	t.Helper()
	var definition struct {
		DataType      string
		IsNullable    string
		ColumnDefault string
	}
	if err := db.Raw(`
SELECT
	data_type,
	is_nullable,
	COALESCE(column_default, '') AS column_default
FROM information_schema.columns
WHERE table_schema = ?
  AND table_name = ?
  AND column_name = ?
`,
		schemaName,
		tableName,
		columnName,
	).Scan(&definition).Error; err != nil {
		t.Fatalf("inspect PostgreSQL column %s.%s: %v", tableName, columnName, err)
	}
	if definition.DataType != dataType || definition.IsNullable != nullable {
		t.Fatalf(
			"PostgreSQL column %s.%s type=%s nullable=%s, want type=%s nullable=%s",
			tableName,
			columnName,
			definition.DataType,
			definition.IsNullable,
			dataType,
			nullable,
		)
	}
	if defaultContains != "" &&
		!strings.Contains(strings.ToLower(definition.ColumnDefault), defaultContains) {
		t.Fatalf(
			"PostgreSQL column %s.%s default=%q, want containing %q",
			tableName,
			columnName,
			definition.ColumnDefault,
			defaultContains,
		)
	}
}

func assertPostgresDataPermissionConstraintBehavior(t *testing.T, db *gorm.DB) {
	t.Helper()
	dimension := model.DataDimensionDefinition{
		Code:         "legal_entity",
		Name:         "Legal Entity",
		Category:     model.DataDimensionCategoryOrganization,
		ValueType:    model.DataDimensionValueTypeBigint,
		ProviderCode: "organization",
	}
	if err := db.Create(&dimension).Error; err != nil {
		t.Fatalf("create PostgreSQL dimension: %v", err)
	}
	policy := model.DataPolicy{
		Code:       "postgres-policy",
		Name:       "PostgreSQL Policy",
		PolicyType: model.DataPolicyTypeRuleSet,
	}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("create PostgreSQL policy: %v", err)
	}
	resource := model.DataResource{
		ResourceCode: "registered-resource",
		Name:         "Registered Resource",
		ResourceType: model.DataResourceTypeBusinessService,
		ServiceCode:  stringPointer("registered-resource"),
		AdapterCode:  "registered",
	}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatalf("create PostgreSQL resource: %v", err)
	}
	if resource.PermissionEnabled {
		t.Fatal("PostgreSQL permission_enabled must default to false")
	}

	assertPostgresWriteRejected(t, db, &model.DataResource{
		ResourceCode: "invalid-target",
		Name:         "Invalid Target",
		ResourceType: model.DataResourceTypeBusinessService,
		TableId:      intPointer(1),
		ServiceCode:  stringPointer("invalid-target"),
		AdapterCode:  "registered",
	})
	assertPostgresWriteRejected(t, db, &model.DataOwnershipField{
		ResourceId:       resource.Id,
		OwnershipCode:    "legal_entity_id",
		DimensionId:      dimension.Id,
		BindingType:      "report_source",
		AdapterFieldCode: stringPointer("legal_entity_id"),
		ValueType:        model.DataDimensionValueTypeBigint,
	})

	firstRule := validDataPolicyRule(policy.Id, dimension.Id, 1)
	if err := db.Create(&firstRule).Error; err != nil {
		t.Fatalf("create PostgreSQL policy rule: %v", err)
	}
	duplicateRule := validDataPolicyRule(policy.Id, dimension.Id, 1)
	assertPostgresWriteRejected(t, db, &duplicateRule)

	invalidSequence := validDataPolicyRule(policy.Id, dimension.Id, 0)
	assertPostgresWriteRejected(t, db, &invalidSequence)
	invalidStructure := validDataPolicyRule(policy.Id, dimension.Id, 2)
	invalidStructure.Relation = model.DataPolicyRelationSelfAndDescendants
	assertPostgresWriteRejected(t, db, &invalidStructure)
	invalidValues := validDataPolicyRule(policy.Id, dimension.Id, 3)
	invalidValues.SpecifiedValues = []byte(`[]`)
	assertPostgresWriteRejected(t, db, &invalidValues)

	validFrom := time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)
	validTo := validFrom.AddDate(0, 0, -1)
	assertPostgresWriteRejected(t, db, &model.DataGrant{
		SubjectType: model.DataGrantSubjectTypeRole,
		SubjectId:   1,
		ResourceId:  resource.Id,
		Operation:   model.DataPermissionOperationQuery,
		PolicyId:    policy.Id,
		ValidFrom:   &validFrom,
		ValidTo:     &validTo,
	})
	assertPostgresWriteRejected(t, db, &model.DataGrant{
		SubjectType: model.DataGrantSubjectTypeRole,
		SubjectId:   1,
		ResourceId:  resource.Id + 999,
		Operation:   model.DataPermissionOperationQuery,
		PolicyId:    policy.Id,
	})
}

func assertPostgresWriteRejected(t *testing.T, db *gorm.DB, value any) {
	t.Helper()
	if err := db.Create(value).Error; err == nil {
		t.Fatalf("expected PostgreSQL to reject %T", value)
	}
}

func validDataPolicyRule(policyID int, dimensionID int, sequence int) model.DataPolicyRule {
	return model.DataPolicyRule{
		PolicyId:      policyID,
		Sequence:      sequence,
		DimensionId:   dimensionID,
		OwnershipCode: "legal_entity_id",
		ScopeSource:   model.DataPolicyScopeSourceEffectiveLegalEntities,
		Relation:      model.DataPolicyRelationExact,
		Operator:      model.DataPolicyOperatorIn,
	}
}

func stringPointer(value string) *string {
	return &value
}

func intPointer(value int) *int {
	return &value
}
