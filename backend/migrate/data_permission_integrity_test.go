package main

import (
	"backend/model"
	"fmt"
	"testing"

	"gorm.io/gorm"
)

func TestDataPermissionDomainIntegritySQLite(t *testing.T) {
	db := migrateTestDB(t)
	if err := migrateDataPermissionSchema(db); err != nil {
		t.Fatalf("migrate data permission schema: %v", err)
	}

	dimension := model.DataDimensionDefinition{
		Code:         "integrity_legal_entity",
		Name:         "Integrity Legal Entity",
		Category:     model.DataDimensionCategoryOrganization,
		ValueType:    model.DataDimensionValueTypeBigint,
		ProviderCode: "organization",
	}
	if err := db.Create(&dimension).Error; err != nil {
		t.Fatalf("create integrity dimension: %v", err)
	}
	resource := model.DataResource{
		ResourceCode: "integrity_service",
		Name:         "Integrity Service",
		ResourceType: model.DataResourceTypeBusinessService,
		ServiceCode:  stringPointer("integrity_service"),
		AdapterCode:  "registered",
	}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatalf("create integrity resource: %v", err)
	}
	policy := model.DataPolicy{
		Code:       "integrity_policy",
		Name:       "Integrity Policy",
		PolicyType: model.DataPolicyTypeRuleSet,
	}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("create integrity policy: %v", err)
	}

	t.Run("dimension_id is required", func(t *testing.T) {
		err := db.Exec(`
INSERT INTO sys_data_policy_rule (
	policy_id,
	sequence,
	dimension_id,
	ownership_code,
	scope_source,
	relation,
	operator,
	state
) VALUES (?, ?, NULL, ?, ?, ?, ?, ?)
`,
			policy.Id,
			1,
			"legal_entity_id",
			model.DataPolicyScopeSourceEffectiveLegalEntities,
			model.DataPolicyRelationExact,
			model.DataPolicyOperatorIn,
			true,
		).Error
		if err == nil {
			t.Fatal("expected NULL dimension_id to be rejected")
		}
	})

	t.Run("ownership binding types", func(t *testing.T) {
		tableFieldID := 9001
		metadataField := model.DataOwnershipField{
			ResourceId:    resource.Id,
			OwnershipCode: "metadata_legal_entity_id",
			DimensionId:   dimension.Id,
			BindingType:   model.DataOwnershipBindingTypeMetadataField,
			TableFieldId:  &tableFieldID,
			ValueType:     model.DataDimensionValueTypeBigint,
		}
		if err := db.Create(&metadataField).Error; err != nil {
			t.Fatalf("metadata_field must be accepted: %v", err)
		}
		registeredField := model.DataOwnershipField{
			ResourceId:       resource.Id,
			OwnershipCode:    "registered_legal_entity_id",
			DimensionId:      dimension.Id,
			BindingType:      model.DataOwnershipBindingTypeRegisteredField,
			AdapterFieldCode: stringPointer("legal_entity_id"),
			ValueType:        model.DataDimensionValueTypeBigint,
		}
		if err := db.Create(&registeredField).Error; err != nil {
			t.Fatalf("registered_field must be accepted: %v", err)
		}
		for index, bindingType := range []string{"report_source", "sql_expression", ""} {
			assertSQLiteWriteRejected(t, db, &model.DataOwnershipField{
				ResourceId:       resource.Id,
				OwnershipCode:    fmt.Sprintf("invalid_binding_%d", index),
				DimensionId:      dimension.Id,
				BindingType:      bindingType,
				AdapterFieldCode: stringPointer("legal_entity_id"),
				ValueType:        model.DataDimensionValueTypeBigint,
			})
		}
	})

	t.Run("permission_enabled defaults to false", func(t *testing.T) {
		err := db.Exec(`
INSERT INTO sys_data_resource (
	resource_code,
	name,
	resource_type,
	service_code,
	adapter_code,
	state
) VALUES (?, ?, ?, ?, ?, ?)
`,
			"integrity_default_permission",
			"Integrity Default Permission",
			model.DataResourceTypeBusinessService,
			"integrity_default_permission",
			"registered",
			true,
		).Error
		if err != nil {
			t.Fatalf("insert resource without permission_enabled: %v", err)
		}
		var persisted model.DataResource
		if err := db.Where(
			"resource_code = ?",
			"integrity_default_permission",
		).First(&persisted).Error; err != nil {
			t.Fatalf("reload resource default: %v", err)
		}
		if persisted.PermissionEnabled {
			t.Fatal("permission_enabled database default must be false")
		}
	})

	t.Run("policy sequence is unique", func(t *testing.T) {
		first := validDataPolicyRule(policy.Id, dimension.Id, 20)
		if err := db.Create(&first).Error; err != nil {
			t.Fatalf("create first integrity rule: %v", err)
		}
		duplicate := validDataPolicyRule(policy.Id, dimension.Id, 20)
		assertSQLiteWriteRejected(t, db, &duplicate)
	})

	t.Run("enum checks reject unknown values", func(t *testing.T) {
		for _, value := range invalidDataPermissionEnumValues(
			resource.Id,
			dimension.Id,
			policy.Id,
		) {
			assertSQLiteWriteRejected(t, db, value)
		}
	})
}

func assertPostgresDataPermissionIntegrity(t *testing.T, db *gorm.DB) {
	t.Helper()

	for _, statement := range []string{
		`INSERT INTO sys_table (id) VALUES (91001)`,
		`INSERT INTO sys_table_field (id) VALUES (91002)`,
		`INSERT INTO report_definition (id) VALUES (91003)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create PostgreSQL integrity prerequisite: %v", err)
		}
	}

	dimension := model.DataDimensionDefinition{
		Code:         "integrity_legal_entity",
		Name:         "Integrity Legal Entity",
		Category:     model.DataDimensionCategoryOrganization,
		ValueType:    model.DataDimensionValueTypeBigint,
		ProviderCode: "organization",
	}
	if err := db.Create(&dimension).Error; err != nil {
		t.Fatalf("create PostgreSQL integrity dimension: %v", err)
	}
	policy := model.DataPolicy{
		Code:       "integrity_policy",
		Name:       "Integrity Policy",
		PolicyType: model.DataPolicyTypeRuleSet,
	}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("create PostgreSQL integrity policy: %v", err)
	}
	resource := model.DataResource{
		ResourceCode: "integrity_service",
		Name:         "Integrity Service",
		ResourceType: model.DataResourceTypeBusinessService,
		ServiceCode:  stringPointer("integrity_service"),
		AdapterCode:  "registered",
	}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatalf("create PostgreSQL integrity resource: %v", err)
	}

	assertPostgresExecRejected(t, db, `
INSERT INTO sys_data_policy_rule (
	policy_id,
	sequence,
	dimension_id,
	ownership_code,
	scope_source,
	relation,
	operator,
	state
) VALUES (?, ?, NULL, ?, ?, ?, ?, ?)
`,
		policy.Id,
		1,
		"legal_entity_id",
		model.DataPolicyScopeSourceEffectiveLegalEntities,
		model.DataPolicyRelationExact,
		model.DataPolicyOperatorIn,
		true,
	)

	tableFieldID := 91002
	if err := db.Create(&model.DataOwnershipField{
		ResourceId:    resource.Id,
		OwnershipCode: "metadata_legal_entity_id",
		DimensionId:   dimension.Id,
		BindingType:   model.DataOwnershipBindingTypeMetadataField,
		TableFieldId:  &tableFieldID,
		ValueType:     model.DataDimensionValueTypeBigint,
	}).Error; err != nil {
		t.Fatalf("PostgreSQL metadata_field must be accepted: %v", err)
	}
	if err := db.Create(&model.DataOwnershipField{
		ResourceId:       resource.Id,
		OwnershipCode:    "registered_legal_entity_id",
		DimensionId:      dimension.Id,
		BindingType:      model.DataOwnershipBindingTypeRegisteredField,
		AdapterFieldCode: stringPointer("legal_entity_id"),
		ValueType:        model.DataDimensionValueTypeBigint,
	}).Error; err != nil {
		t.Fatalf("PostgreSQL registered_field must be accepted: %v", err)
	}

	assertPostgresForeignKeysRejectMissingReferences(t, db, resource, dimension, policy)
	assertPostgresEnumsRejectUnknownValues(t, db, resource, dimension, policy)
}

func assertPostgresForeignKeysRejectMissingReferences(
	t *testing.T,
	db *gorm.DB,
	resource model.DataResource,
	dimension model.DataDimensionDefinition,
	policy model.DataPolicy,
) {
	t.Helper()
	missingID := 999999

	assertPostgresWriteRejected(t, db, &model.DataResource{
		ResourceCode: "missing_table",
		Name:         "Missing Table",
		ResourceType: model.DataResourceTypeLowCodeTable,
		TableId:      &missingID,
		AdapterCode:  "metadata",
	})
	assertPostgresWriteRejected(t, db, &model.DataResource{
		ResourceCode:       "missing_report",
		Name:               "Missing Report",
		ResourceType:       model.DataResourceTypeReport,
		ReportDefinitionId: &missingID,
		AdapterCode:        "report",
	})
	assertPostgresWriteRejected(t, db, &model.DataResourceOperation{
		ResourceId: missingID,
		Operation:  model.DataPermissionOperationQuery,
	})
	assertPostgresWriteRejected(t, db, &model.DataOwnershipField{
		ResourceId:       missingID,
		OwnershipCode:    "missing_resource",
		DimensionId:      dimension.Id,
		BindingType:      model.DataOwnershipBindingTypeRegisteredField,
		AdapterFieldCode: stringPointer("legal_entity_id"),
		ValueType:        model.DataDimensionValueTypeBigint,
	})
	assertPostgresWriteRejected(t, db, &model.DataOwnershipField{
		ResourceId:       resource.Id,
		OwnershipCode:    "missing_dimension",
		DimensionId:      missingID,
		BindingType:      model.DataOwnershipBindingTypeRegisteredField,
		AdapterFieldCode: stringPointer("legal_entity_id"),
		ValueType:        model.DataDimensionValueTypeBigint,
	})
	assertPostgresWriteRejected(t, db, &model.DataOwnershipField{
		ResourceId:    resource.Id,
		OwnershipCode: "missing_table_field",
		DimensionId:   dimension.Id,
		BindingType:   model.DataOwnershipBindingTypeMetadataField,
		TableFieldId:  &missingID,
		ValueType:     model.DataDimensionValueTypeBigint,
	})
	assertPostgresWriteRejected(t, db, &model.DataPolicyRule{
		PolicyId:      missingID,
		Sequence:      40,
		DimensionId:   dimension.Id,
		OwnershipCode: "legal_entity_id",
		ScopeSource:   model.DataPolicyScopeSourceEffectiveLegalEntities,
		Relation:      model.DataPolicyRelationExact,
		Operator:      model.DataPolicyOperatorIn,
	})
	assertPostgresWriteRejected(t, db, &model.DataPolicyRule{
		PolicyId:      policy.Id,
		Sequence:      41,
		DimensionId:   missingID,
		OwnershipCode: "legal_entity_id",
		ScopeSource:   model.DataPolicyScopeSourceEffectiveLegalEntities,
		Relation:      model.DataPolicyRelationExact,
		Operator:      model.DataPolicyOperatorIn,
	})
	assertPostgresWriteRejected(t, db, &model.DataGrant{
		SubjectType: model.DataGrantSubjectTypeRole,
		SubjectId:   1,
		ResourceId:  missingID,
		Operation:   model.DataPermissionOperationQuery,
		PolicyId:    policy.Id,
	})
	assertPostgresWriteRejected(t, db, &model.DataGrant{
		SubjectType: model.DataGrantSubjectTypeRole,
		SubjectId:   1,
		ResourceId:  resource.Id,
		Operation:   model.DataPermissionOperationQuery,
		PolicyId:    missingID,
	})
}

func assertPostgresEnumsRejectUnknownValues(
	t *testing.T,
	db *gorm.DB,
	resource model.DataResource,
	dimension model.DataDimensionDefinition,
	policy model.DataPolicy,
) {
	t.Helper()
	for _, value := range invalidDataPermissionEnumValues(
		resource.Id,
		dimension.Id,
		policy.Id,
	) {
		assertPostgresWriteRejected(t, db, value)
	}
}

func invalidDataPermissionEnumValues(
	resourceID int,
	dimensionID int,
	policyID int,
) []any {
	return []any{
		&model.DataDimensionDefinition{
			Code:         "invalid_category",
			Name:         "Invalid Category",
			Category:     "department",
			ValueType:    model.DataDimensionValueTypeBigint,
			ProviderCode: "organization",
		},
		&model.DataDimensionDefinition{
			Code:         "invalid_value_type",
			Name:         "Invalid Value Type",
			Category:     model.DataDimensionCategoryOrganization,
			ValueType:    "json",
			ProviderCode: "organization",
		},
		&model.DataResource{
			ResourceCode: "invalid_resource_type",
			Name:         "Invalid Resource Type",
			ResourceType: "menu",
			ServiceCode:  stringPointer("invalid_resource_type"),
			AdapterCode:  "registered",
		},
		&model.DataResourceOperation{
			ResourceId: resourceID,
			Operation:  "approve",
		},
		&model.DataOwnershipField{
			ResourceId:       resourceID,
			OwnershipCode:    "invalid_binding_type",
			DimensionId:      dimensionID,
			BindingType:      "report_source",
			AdapterFieldCode: stringPointer("legal_entity_id"),
			ValueType:        model.DataDimensionValueTypeBigint,
		},
		&model.DataOwnershipField{
			ResourceId:       resourceID,
			OwnershipCode:    "invalid_ownership_value_type",
			DimensionId:      dimensionID,
			BindingType:      model.DataOwnershipBindingTypeRegisteredField,
			AdapterFieldCode: stringPointer("legal_entity_id"),
			ValueType:        "json",
		},
		&model.DataPolicy{
			Code:       "invalid_policy_type",
			Name:       "Invalid Policy Type",
			PolicyType: "allow",
		},
		&model.DataPolicyRule{
			PolicyId:      policyID,
			Sequence:      30,
			DimensionId:   dimensionID,
			OwnershipCode: "legal_entity_id",
			ScopeSource:   "current_role",
			Relation:      model.DataPolicyRelationExact,
			Operator:      model.DataPolicyOperatorIn,
		},
		&model.DataPolicyRule{
			PolicyId:      policyID,
			Sequence:      31,
			DimensionId:   dimensionID,
			OwnershipCode: "legal_entity_id",
			ScopeSource:   model.DataPolicyScopeSourceEffectiveLegalEntities,
			Relation:      "children",
			Operator:      model.DataPolicyOperatorIn,
		},
		&model.DataPolicyRule{
			PolicyId:      policyID,
			Sequence:      32,
			DimensionId:   dimensionID,
			OwnershipCode: "legal_entity_id",
			ScopeSource:   model.DataPolicyScopeSourceEffectiveLegalEntities,
			Relation:      model.DataPolicyRelationExact,
			Operator:      "sql",
		},
		&model.DataGrant{
			SubjectType: "position",
			SubjectId:   1,
			ResourceId:  resourceID,
			Operation:   model.DataPermissionOperationQuery,
			PolicyId:    policyID,
		},
		&model.DataGrant{
			SubjectType: model.DataGrantSubjectTypeRole,
			SubjectId:   1,
			ResourceId:  resourceID,
			Operation:   "approve",
			PolicyId:    policyID,
		},
	}
}

func assertSQLiteWriteRejected(t *testing.T, db *gorm.DB, value any) {
	t.Helper()
	if err := db.Create(value).Error; err == nil {
		t.Fatalf("expected SQLite to reject %T", value)
	}
}

func assertPostgresExecRejected(
	t *testing.T,
	db *gorm.DB,
	statement string,
	arguments ...any,
) {
	t.Helper()
	if err := db.Exec(statement, arguments...).Error; err == nil {
		t.Fatal("expected PostgreSQL statement to be rejected")
	}
}
