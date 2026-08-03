package main

import (
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

type postgresCheckConstraint struct {
	model      any
	name       string
	expression string
}

type postgresForeignKeyConstraint struct {
	model           any
	name            string
	columns         []string
	referenceModel  any
	referenceFields []string
}

// migrateDataPermissionSchema 负责已评审的 Data Permission V1 schema。
// 跨记录 Policy 校验仍由 Service 负责。
func migrateDataPermissionSchema(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(dataPermissionDomainModels()...); err != nil {
			return fmt.Errorf("auto migrate data permission schema: %w", err)
		}
		if tx.Dialector.Name() != "postgres" {
			return nil
		}
		if err := createDataPermissionPostgresChecks(tx); err != nil {
			return err
		}
		if err := createDataPermissionPostgresForeignKeys(tx); err != nil {
			return err
		}
		return nil
	})
}

func dataPermissionDomainModels() []any {
	return []any{
		&model.DataDimensionDefinition{},
		&model.DataResource{},
		&model.DataResourceOperation{},
		&model.DataOwnershipField{},
		&model.DataPolicy{},
		&model.DataPolicyRule{},
		&model.DataGrant{},
	}
}

func createDataPermissionPostgresChecks(db *gorm.DB) error {
	constraints := []postgresCheckConstraint{
		{
			model: &model.DataResource{},
			name:  "chk_data_resource_target",
			expression: `(
				(resource_type = 'low_code_table' AND table_id IS NOT NULL AND service_code IS NULL AND report_definition_id IS NULL)
				OR (resource_type = 'business_service' AND table_id IS NULL AND service_code IS NOT NULL AND btrim(service_code) <> '' AND report_definition_id IS NULL)
				OR (resource_type = 'report' AND table_id IS NULL AND service_code IS NULL AND report_definition_id IS NOT NULL)
			)`,
		},
		{
			model: &model.DataOwnershipField{},
			name:  "chk_data_ownership_binding_target",
			expression: `(
				(binding_type = 'metadata_field' AND table_field_id IS NOT NULL AND adapter_field_code IS NULL)
				OR (binding_type = 'registered_field' AND table_field_id IS NULL AND adapter_field_code IS NOT NULL AND btrim(adapter_field_code) <> '')
			)`,
		},
		{
			model:      &model.DataPolicyRule{},
			name:       "chk_data_policy_rule_sequence",
			expression: "sequence > 0",
		},
		{
			model: &model.DataPolicyRule{},
			name:  "chk_data_policy_rule_specified_values",
			expression: `(
				(scope_source = 'specified_values' AND specified_values IS NOT NULL AND jsonb_typeof(specified_values) = 'array')
				OR (scope_source <> 'specified_values' AND specified_values IS NULL)
			)`,
		},
		{
			model: &model.DataPolicyRule{},
			name:  "chk_data_policy_rule_structure",
			expression: `(
				(relation = 'self_and_descendants' AND structure_code IS NOT NULL AND btrim(structure_code) <> '')
				OR (relation = 'exact' AND structure_code IS NULL)
			)`,
		},
		{
			model: &model.DataGrant{},
			name:  "chk_data_grant_valid_range",
			expression: `(
				valid_from IS NULL
				OR valid_to IS NULL
				OR valid_from <= valid_to
			)`,
		},
	}

	for _, constraint := range constraints {
		if err := createPostgresCheckConstraint(db, constraint); err != nil {
			return err
		}
	}
	return nil
}

func createDataPermissionPostgresForeignKeys(db *gorm.DB) error {
	constraints := []postgresForeignKeyConstraint{
		{
			model:           &model.DataResource{},
			name:            "fk_data_resource_table",
			columns:         []string{"table_id"},
			referenceModel:  &model.SysTable{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataResource{},
			name:            "fk_data_resource_report_definition",
			columns:         []string{"report_definition_id"},
			referenceModel:  &model.ReportDefinition{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataResourceOperation{},
			name:            "fk_data_resource_operation_resource",
			columns:         []string{"resource_id"},
			referenceModel:  &model.DataResource{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataOwnershipField{},
			name:            "fk_data_ownership_field_resource",
			columns:         []string{"resource_id"},
			referenceModel:  &model.DataResource{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataOwnershipField{},
			name:            "fk_data_ownership_field_dimension",
			columns:         []string{"dimension_id"},
			referenceModel:  &model.DataDimensionDefinition{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataOwnershipField{},
			name:            "fk_data_ownership_field_table_field",
			columns:         []string{"table_field_id"},
			referenceModel:  &model.SysTableField{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataPolicyRule{},
			name:            "fk_data_policy_rule_policy",
			columns:         []string{"policy_id"},
			referenceModel:  &model.DataPolicy{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataPolicyRule{},
			name:            "fk_data_policy_rule_dimension",
			columns:         []string{"dimension_id"},
			referenceModel:  &model.DataDimensionDefinition{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataGrant{},
			name:            "fk_data_grant_resource",
			columns:         []string{"resource_id"},
			referenceModel:  &model.DataResource{},
			referenceFields: []string{"id"},
		},
		{
			model:           &model.DataGrant{},
			name:            "fk_data_grant_policy",
			columns:         []string{"policy_id"},
			referenceModel:  &model.DataPolicy{},
			referenceFields: []string{"id"},
		},
	}

	for _, constraint := range constraints {
		if err := createPostgresForeignKeyConstraint(db, constraint); err != nil {
			return err
		}
	}
	return nil
}

func createPostgresCheckConstraint(
	db *gorm.DB,
	constraint postgresCheckConstraint,
) error {
	tableName, quotedTableName, err := postgresModelTableNames(db, constraint.model)
	if err != nil {
		return fmt.Errorf("resolve PostgreSQL check table: %w", err)
	}
	exists, err := postgresConstraintExists(db, tableName, constraint.name)
	if err != nil {
		return fmt.Errorf("inspect PostgreSQL check %s: %w", constraint.name, err)
	}
	if exists {
		return nil
	}
	quotedConstraintName, err := quotePostgresIdentifier(constraint.name)
	if err != nil {
		return fmt.Errorf("quote PostgreSQL check %s: %w", constraint.name, err)
	}
	sql := fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)",
		quotedTableName,
		quotedConstraintName,
		constraint.expression,
	)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("create PostgreSQL check %s: %w", constraint.name, err)
	}
	return nil
}

func createPostgresForeignKeyConstraint(
	db *gorm.DB,
	constraint postgresForeignKeyConstraint,
) error {
	tableName, quotedTableName, err := postgresModelTableNames(db, constraint.model)
	if err != nil {
		return fmt.Errorf("resolve PostgreSQL foreign key table: %w", err)
	}
	_, quotedReferenceTableName, err := postgresModelTableNames(db, constraint.referenceModel)
	if err != nil {
		return fmt.Errorf("resolve PostgreSQL foreign key reference: %w", err)
	}
	exists, err := postgresConstraintExists(db, tableName, constraint.name)
	if err != nil {
		return fmt.Errorf("inspect PostgreSQL foreign key %s: %w", constraint.name, err)
	}
	if exists {
		return nil
	}
	quotedConstraintName, err := quotePostgresIdentifier(constraint.name)
	if err != nil {
		return fmt.Errorf("quote PostgreSQL foreign key %s: %w", constraint.name, err)
	}
	quotedColumns, err := quotePostgresIdentifiers(constraint.columns)
	if err != nil {
		return fmt.Errorf("quote PostgreSQL foreign key columns %s: %w", constraint.name, err)
	}
	quotedReferenceFields, err := quotePostgresIdentifiers(constraint.referenceFields)
	if err != nil {
		return fmt.Errorf("quote PostgreSQL foreign key reference columns %s: %w", constraint.name, err)
	}
	sql := fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s) ON UPDATE RESTRICT ON DELETE RESTRICT",
		quotedTableName,
		quotedConstraintName,
		quotedColumns,
		quotedReferenceTableName,
		quotedReferenceFields,
	)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("create PostgreSQL foreign key %s: %w", constraint.name, err)
	}
	return nil
}

func postgresModelTableNames(db *gorm.DB, modelValue any) (string, string, error) {
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(modelValue); err != nil {
		return "", "", err
	}
	quotedTableName, err := quotePostgresQualifiedIdentifier(statement.Schema.Table)
	if err != nil {
		return "", "", err
	}
	return statement.Schema.Table, quotedTableName, nil
}

func postgresConstraintExists(db *gorm.DB, tableName string, constraintName string) (bool, error) {
	var exists bool
	err := db.Raw(`
SELECT EXISTS (
	SELECT 1
	FROM pg_constraint
	WHERE conrelid = to_regclass(?)
	  AND conname = ?
)`,
		tableName,
		constraintName,
	).Scan(&exists).Error
	return exists, err
}

func quotePostgresIdentifiers(values []string) (string, error) {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		identifier, err := quotePostgresIdentifier(value)
		if err != nil {
			return "", err
		}
		quoted = append(quoted, identifier)
	}
	return joinQuotedIdentifiers(quoted), nil
}

func joinQuotedIdentifiers(values []string) string {
	result := ""
	for index, value := range values {
		if index > 0 {
			result += ", "
		}
		result += value
	}
	return result
}
