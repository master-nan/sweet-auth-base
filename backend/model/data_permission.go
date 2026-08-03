package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	DataDimensionCategoryOrganization = "organization"
	DataDimensionCategoryEmployee     = "employee"
	DataDimensionCategoryUser         = "user"
	DataDimensionCategoryBusiness     = "business"
	DataDimensionCategorySystem       = "system"

	DataDimensionValueTypeBigint = "bigint"
	DataDimensionValueTypeString = "string"

	DataResourceTypeLowCodeTable    = "low_code_table"
	DataResourceTypeBusinessService = "business_service"
	DataResourceTypeReport          = "report"

	DataPermissionOperationQuery  = "query"
	DataPermissionOperationDetail = "detail"
	DataPermissionOperationCreate = "create"
	DataPermissionOperationUpdate = "update"
	DataPermissionOperationDelete = "delete"
	DataPermissionOperationExport = "export"
	DataPermissionOperationRun    = "run"

	DataOwnershipBindingTypeMetadataField   = "metadata_field"
	DataOwnershipBindingTypeRegisteredField = "registered_field"

	DataPolicyTypeAll     = "all"
	DataPolicyTypeNone    = "none"
	DataPolicyTypeRuleSet = "rule_set"

	DataPolicyScopeSourceCurrentUser            = "current_user"
	DataPolicyScopeSourceCurrentEmployee        = "current_employee"
	DataPolicyScopeSourceEffectiveLegalEntities = "effective_legal_entities"
	DataPolicyScopeSourceEffectiveOrgUnits      = "effective_org_units"
	DataPolicyScopeSourceSpecifiedValues        = "specified_values"
	DataPolicyScopeSourceProviderSubjectScope   = "provider_subject_scope"

	DataPolicyRelationExact              = "exact"
	DataPolicyRelationSelfAndDescendants = "self_and_descendants"

	DataPolicyOperatorEqual = "eq"
	DataPolicyOperatorIn    = "in"

	DataGrantSubjectTypeRole = "role"
	DataGrantSubjectTypeUser = "user"
)

// DataDimensionDefinition 声明稳定的数据权限 Dimension 及其 Provider。
type DataDimensionDefinition struct {
	Basic

	Code         string  `gorm:"size:64;not null;uniqueIndex:uni_data_dimension_definition_code" json:"code"`
	Name         string  `gorm:"size:128;not null;index:idx_data_dimension_definition_name" json:"name"`
	Category     string  `gorm:"size:32;not null;index:idx_data_dimension_definition_category;check:chk_data_dimension_category,category IN ('organization','employee','user','business','system')" json:"category"`
	ValueType    string  `gorm:"size:32;not null;index:idx_data_dimension_definition_value_type;check:chk_data_dimension_value_type,value_type IN ('bigint','string')" json:"value_type"`
	ProviderCode string  `gorm:"size:64;not null;index:idx_data_dimension_definition_provider" json:"provider_code"`
	SelectorType *string `gorm:"size:64" json:"selector_type"`
	Description  string  `gorm:"size:256" json:"description"`
}

func (DataDimensionDefinition) TableName() string {
	return "sys_data_dimension_definition"
}

// DataResource 标识受保护的表、Service 或报表资源。
type DataResource struct {
	Basic

	ResourceCode       string  `gorm:"size:128;not null;uniqueIndex:uni_data_resource_code" json:"resource_code"`
	Name               string  `gorm:"size:128;not null;index:idx_data_resource_name" json:"name"`
	ResourceType       string  `gorm:"size:32;not null;index:idx_data_resource_type;check:chk_data_resource_type,resource_type IN ('low_code_table','business_service','report')" json:"resource_type"`
	TableId            *int    `gorm:"type:bigint;index:idx_data_resource_table" json:"table_id"`
	ServiceCode        *string `gorm:"size:128;index:idx_data_resource_service" json:"service_code"`
	ReportDefinitionId *int    `gorm:"type:bigint;index:idx_data_resource_report_definition" json:"report_definition_id"`
	AdapterCode        string  `gorm:"size:64;not null;index:idx_data_resource_adapter" json:"adapter_code"`
	PermissionEnabled  bool    `gorm:"not null;default:false;index:idx_data_resource_permission_enabled" json:"permission_enabled"`
	Description        string  `gorm:"size:256" json:"description"`
}

func (DataResource) TableName() string {
	return "sys_data_resource"
}

// DataResourceOperation 声明 Resource 支持的一项受保护 Operation。
type DataResourceOperation struct {
	Basic

	ResourceId        int    `gorm:"type:bigint;not null;uniqueIndex:uni_data_resource_operation,priority:1,where:gmt_delete IS NULL;index:idx_data_resource_operation_resource" json:"resource_id"`
	Operation         string `gorm:"size:32;not null;uniqueIndex:uni_data_resource_operation,priority:2,where:gmt_delete IS NULL;index:idx_data_resource_operation_operation;check:chk_data_resource_operation,operation IN ('query','detail','create','update','delete','export','run')" json:"operation"`
	PermissionEnabled bool   `gorm:"not null;default:false;index:idx_data_resource_operation_enabled" json:"permission_enabled"`
	Description       string `gorm:"size:256" json:"description"`
}

func (DataResourceOperation) TableName() string {
	return "sys_data_resource_operation"
}

// DataOwnershipField 将资源归属标识绑定到受控 Dimension。
type DataOwnershipField struct {
	Basic

	ResourceId       int     `gorm:"type:bigint;not null;uniqueIndex:uni_data_ownership_field,priority:1,where:gmt_delete IS NULL;index:idx_data_ownership_field_resource" json:"resource_id"`
	OwnershipCode    string  `gorm:"size:64;not null;uniqueIndex:uni_data_ownership_field,priority:2,where:gmt_delete IS NULL" json:"ownership_code"`
	DimensionId      int     `gorm:"type:bigint;not null;index:idx_data_ownership_field_dimension" json:"dimension_id"`
	BindingType      string  `gorm:"size:32;not null;index:idx_data_ownership_field_binding_type;check:chk_data_ownership_binding_type,binding_type IN ('metadata_field','registered_field')" json:"binding_type"`
	TableFieldId     *int    `gorm:"type:bigint;index:idx_data_ownership_field_table_field" json:"table_field_id"`
	AdapterFieldCode *string `gorm:"size:128;index:idx_data_ownership_field_adapter_field" json:"adapter_field_code"`
	ValueType        string  `gorm:"size:32;not null;index:idx_data_ownership_field_value_type;check:chk_data_ownership_value_type,value_type IN ('bigint','string')" json:"value_type"`
	Description      string  `gorm:"size:256" json:"description"`
}

func (DataOwnershipField) TableName() string {
	return "sys_data_ownership_field"
}

// DataPolicy 定义不直接绑定 Subject 或 Resource 的可复用数据范围策略。
type DataPolicy struct {
	Basic

	Code        string `gorm:"size:128;not null;uniqueIndex:uni_data_policy_code" json:"code"`
	Name        string `gorm:"size:128;not null;index:idx_data_policy_name" json:"name"`
	PolicyType  string `gorm:"size:32;not null;index:idx_data_policy_type;check:chk_data_policy_type,policy_type IN ('all','none','rule_set')" json:"policy_type"`
	Description string `gorm:"size:512" json:"description"`
}

func (DataPolicy) TableName() string {
	return "sys_data_policy"
}

// DataPolicyRule 是规则集 Policy 中的一条结构化 Rule。
type DataPolicyRule struct {
	Basic

	PolicyId        int            `gorm:"type:bigint;not null;uniqueIndex:uni_data_policy_rule_sequence,priority:1,where:gmt_delete IS NULL;index:idx_data_policy_rule_policy" json:"policy_id"`
	Sequence        int            `gorm:"not null;uniqueIndex:uni_data_policy_rule_sequence,priority:2,where:gmt_delete IS NULL" json:"sequence"`
	DimensionId     int            `gorm:"type:bigint;not null;index:idx_data_policy_rule_dimension" json:"dimension_id"`
	OwnershipCode   string         `gorm:"size:64;not null;index:idx_data_policy_rule_ownership" json:"ownership_code"`
	ScopeSource     string         `gorm:"size:64;not null;index:idx_data_policy_rule_scope_source;check:chk_data_policy_rule_scope_source,scope_source IN ('current_user','current_employee','effective_legal_entities','effective_org_units','specified_values','provider_subject_scope')" json:"scope_source"`
	Relation        string         `gorm:"size:32;not null;check:chk_data_policy_rule_relation,relation IN ('exact','self_and_descendants')" json:"relation"`
	Operator        string         `gorm:"size:16;not null;check:chk_data_policy_rule_operator,operator IN ('eq','in')" json:"operator"`
	SpecifiedValues datatypes.JSON `gorm:"type:jsonb" json:"specified_values"`
	StructureCode   *string        `gorm:"size:64" json:"structure_code"`
	Description     string         `gorm:"size:256" json:"description"`
}

func (DataPolicyRule) TableName() string {
	return "sys_data_policy_rule"
}

// DataGrant 将可复用 Policy 授予角色或用户的指定 Resource Operation。
type DataGrant struct {
	Basic

	SubjectType string     `gorm:"size:16;not null;uniqueIndex:uni_data_grant,priority:1,where:gmt_delete IS NULL;index:idx_data_grant_subject,priority:1;check:chk_data_grant_subject_type,subject_type IN ('role','user')" json:"subject_type"`
	SubjectId   int        `gorm:"type:bigint;not null;uniqueIndex:uni_data_grant,priority:2,where:gmt_delete IS NULL;index:idx_data_grant_subject,priority:2" json:"subject_id"`
	ResourceId  int        `gorm:"type:bigint;not null;uniqueIndex:uni_data_grant,priority:3,where:gmt_delete IS NULL;index:idx_data_grant_runtime,priority:1;index:idx_data_grant_policy,priority:1" json:"resource_id"`
	Operation   string     `gorm:"size:32;not null;uniqueIndex:uni_data_grant,priority:4,where:gmt_delete IS NULL;index:idx_data_grant_runtime,priority:2;check:chk_data_grant_operation,operation IN ('query','detail','create','update','delete','export','run')" json:"operation"`
	PolicyId    int        `gorm:"type:bigint;not null;uniqueIndex:uni_data_grant,priority:5,where:gmt_delete IS NULL;index:idx_data_grant_policy,priority:2" json:"policy_id"`
	ValidFrom   *time.Time `gorm:"type:date;index:idx_data_grant_valid_from" json:"valid_from"`
	ValidTo     *time.Time `gorm:"type:date;index:idx_data_grant_valid_to" json:"valid_to"`
	Description string     `gorm:"size:256" json:"description"`
}

func (DataGrant) TableName() string {
	return "sys_data_grant"
}
