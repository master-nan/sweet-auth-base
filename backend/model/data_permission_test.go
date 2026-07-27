package model_test

import (
	"strings"
	"sync"
	"testing"

	"backend/model"

	"gorm.io/gorm/schema"
)

func TestDataPermissionModelsExposeReviewedTablesAndFields(t *testing.T) {
	tests := []struct {
		table   string
		value   interface{}
		columns []string
	}{
		{
			table: "sys_data_dimension_definition",
			value: &model.DataDimensionDefinition{},
			columns: []string{
				"code", "name", "category", "value_type", "provider_code",
				"selector_type", "description",
			},
		},
		{
			table: "sys_data_resource",
			value: &model.DataResource{},
			columns: []string{
				"resource_code", "name", "resource_type", "table_id", "service_code",
				"report_definition_id", "adapter_code", "permission_enabled", "description",
			},
		},
		{
			table: "sys_data_resource_operation",
			value: &model.DataResourceOperation{},
			columns: []string{
				"resource_id", "operation", "permission_enabled", "description",
			},
		},
		{
			table: "sys_data_ownership_field",
			value: &model.DataOwnershipField{},
			columns: []string{
				"resource_id", "ownership_code", "dimension_id", "binding_type",
				"table_field_id", "adapter_field_code", "value_type", "description",
			},
		},
		{
			table: "sys_data_policy",
			value: &model.DataPolicy{},
			columns: []string{
				"code", "name", "policy_type", "description",
			},
		},
		{
			table: "sys_data_policy_rule",
			value: &model.DataPolicyRule{},
			columns: []string{
				"policy_id", "sequence", "dimension_id", "ownership_code", "scope_source",
				"relation", "operator", "specified_values", "structure_code", "description",
			},
		},
		{
			table: "sys_data_grant",
			value: &model.DataGrant{},
			columns: []string{
				"subject_type", "subject_id", "resource_id", "operation", "policy_id",
				"valid_from", "valid_to", "description",
			},
		},
	}

	basicColumns := []string{
		"id", "gmt_create", "create_user", "create_name", "gmt_modify",
		"modify_user", "modify_name", "gmt_delete", "delete_user", "delete_name", "state",
	}
	forbiddenColumns := []string{
		"menu_id", "organization_id", "structure_node_id", "sql",
		"sql_expression", "dynamic_expression", "report_source",
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			parsed := parseDataPermissionSchema(t, tt.value)
			if parsed.Table != tt.table {
				t.Fatalf("table = %q, want %q", parsed.Table, tt.table)
			}

			for _, column := range append(basicColumns, tt.columns...) {
				if parsed.LookUpField(column) == nil {
					t.Errorf("%s is missing reviewed column %q", tt.table, column)
				}
			}
			for _, column := range forbiddenColumns {
				if parsed.LookUpField(column) != nil {
					t.Errorf("%s must not expose forbidden column %q", tt.table, column)
				}
			}
			if len(parsed.Relationships.Relations) != 0 {
				t.Errorf("%s must define relation keys without automatic relationship configuration", tt.table)
			}
		})
	}
}

func TestDataPermissionRuleRequiresDimension(t *testing.T) {
	parsed := parseDataPermissionSchema(t, &model.DataPolicyRule{})
	field := parsed.LookUpField("dimension_id")
	if field == nil {
		t.Fatal("DataPolicyRule.dimension_id is required by the P0 baseline")
	}
	if !field.NotNull {
		t.Fatal("DataPolicyRule.dimension_id must be prepared as NOT NULL")
	}
}

func TestDataPermissionEnumConstraintsMatchP0Baseline(t *testing.T) {
	tests := []struct {
		name       string
		value      interface{}
		constraint string
		allowed    []string
		forbidden  []string
	}{
		{
			name:       "dimension category",
			value:      &model.DataDimensionDefinition{},
			constraint: "chk_data_dimension_category",
			allowed: []string{
				model.DataDimensionCategoryOrganization,
				model.DataDimensionCategoryEmployee,
				model.DataDimensionCategoryUser,
				model.DataDimensionCategoryBusiness,
				model.DataDimensionCategorySystem,
			},
		},
		{
			name:       "dimension value type",
			value:      &model.DataDimensionDefinition{},
			constraint: "chk_data_dimension_value_type",
			allowed: []string{
				model.DataDimensionValueTypeBigint,
				model.DataDimensionValueTypeString,
			},
		},
		{
			name:       "resource type",
			value:      &model.DataResource{},
			constraint: "chk_data_resource_type",
			allowed: []string{
				model.DataResourceTypeLowCodeTable,
				model.DataResourceTypeBusinessService,
				model.DataResourceTypeReport,
			},
		},
		{
			name:       "resource operation",
			value:      &model.DataResourceOperation{},
			constraint: "chk_data_resource_operation",
			allowed:    reviewedDataPermissionOperations(),
		},
		{
			name:       "ownership binding type",
			value:      &model.DataOwnershipField{},
			constraint: "chk_data_ownership_binding_type",
			allowed: []string{
				model.DataOwnershipBindingTypeMetadataField,
				model.DataOwnershipBindingTypeRegisteredField,
			},
			forbidden: []string{"report_source"},
		},
		{
			name:       "ownership value type",
			value:      &model.DataOwnershipField{},
			constraint: "chk_data_ownership_value_type",
			allowed: []string{
				model.DataDimensionValueTypeBigint,
				model.DataDimensionValueTypeString,
			},
		},
		{
			name:       "policy type",
			value:      &model.DataPolicy{},
			constraint: "chk_data_policy_type",
			allowed: []string{
				model.DataPolicyTypeAll,
				model.DataPolicyTypeNone,
				model.DataPolicyTypeRuleSet,
			},
		},
		{
			name:       "policy scope source",
			value:      &model.DataPolicyRule{},
			constraint: "chk_data_policy_rule_scope_source",
			allowed: []string{
				model.DataPolicyScopeSourceCurrentUser,
				model.DataPolicyScopeSourceCurrentEmployee,
				model.DataPolicyScopeSourceEffectiveLegalEntities,
				model.DataPolicyScopeSourceEffectiveOrgUnits,
				model.DataPolicyScopeSourceSpecifiedValues,
				model.DataPolicyScopeSourceProviderSubjectScope,
			},
		},
		{
			name:       "policy relation",
			value:      &model.DataPolicyRule{},
			constraint: "chk_data_policy_rule_relation",
			allowed: []string{
				model.DataPolicyRelationExact,
				model.DataPolicyRelationSelfAndDescendants,
			},
		},
		{
			name:       "policy operator",
			value:      &model.DataPolicyRule{},
			constraint: "chk_data_policy_rule_operator",
			allowed: []string{
				model.DataPolicyOperatorEqual,
				model.DataPolicyOperatorIn,
			},
		},
		{
			name:       "grant subject type",
			value:      &model.DataGrant{},
			constraint: "chk_data_grant_subject_type",
			allowed: []string{
				model.DataGrantSubjectTypeRole,
				model.DataGrantSubjectTypeUser,
			},
			forbidden: []string{"position", "assignment", "user_group"},
		},
		{
			name:       "grant operation",
			value:      &model.DataGrant{},
			constraint: "chk_data_grant_operation",
			allowed:    reviewedDataPermissionOperations(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parseDataPermissionSchema(t, tt.value)
			check, ok := parsed.ParseCheckConstraints()[tt.constraint]
			if !ok {
				t.Fatalf("missing check constraint %q", tt.constraint)
			}
			for _, value := range tt.allowed {
				if !strings.Contains(check.Constraint, "'"+value+"'") {
					t.Errorf("%s does not allow reviewed value %q", tt.constraint, value)
				}
			}
			for _, value := range tt.forbidden {
				if strings.Contains(check.Constraint, "'"+value+"'") {
					t.Errorf("%s must not allow forbidden value %q", tt.constraint, value)
				}
			}
		})
	}
}

func parseDataPermissionSchema(t *testing.T, value interface{}) *schema.Schema {
	t.Helper()

	parsed, err := schema.Parse(
		value,
		&sync.Map{},
		schema.NamingStrategy{SingularTable: true},
	)
	if err != nil {
		t.Fatalf("parse %T schema: %v", value, err)
	}
	return parsed
}

func reviewedDataPermissionOperations() []string {
	return []string{
		model.DataPermissionOperationQuery,
		model.DataPermissionOperationDetail,
		model.DataPermissionOperationCreate,
		model.DataPermissionOperationUpdate,
		model.DataPermissionOperationDelete,
		model.DataPermissionOperationExport,
		model.DataPermissionOperationRun,
	}
}
