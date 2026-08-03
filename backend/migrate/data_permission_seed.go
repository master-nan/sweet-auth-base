package main

import (
	"backend/internal/utils"
	"backend/model"
	"fmt"

	"gorm.io/gorm"
)

type dataPermissionDimensionSeed struct {
	code         string
	name         string
	category     string
	valueType    string
	providerCode string
	selectorType string
	description  string
}

// seedDataPermissionFoundation 仅初始化已评审的平台字典和 Dimension 定义。
// Resource、Policy 和 Grant 由后续配置完成。
func seedDataPermissionFoundation(db *gorm.DB, sf *utils.Snowflake) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range dataPermissionDictionarySeeds() {
			if err := seedSystemDict(tx, sf, seed); err != nil {
				return fmt.Errorf("seed data permission dictionary %s: %w", seed.code, err)
			}
		}
		for _, seed := range dataPermissionDimensionSeeds() {
			if err := seedDataPermissionDimension(tx, sf, seed); err != nil {
				return fmt.Errorf("seed data permission dimension %s: %w", seed.code, err)
			}
		}
		return nil
	})
}

func dataPermissionDictionarySeeds() []systemDictSeed {
	return []systemDictSeed{
		{
			name: "数据权限维度类别",
			code: "data_permission_dimension_category",
			items: []systemDictItemSeed{
				{name: "组织", code: "data_permission_dimension_category_organization", value: model.DataDimensionCategoryOrganization},
				{name: "人员", code: "data_permission_dimension_category_employee", value: model.DataDimensionCategoryEmployee},
				{name: "用户", code: "data_permission_dimension_category_user", value: model.DataDimensionCategoryUser},
				{name: "业务", code: "data_permission_dimension_category_business", value: model.DataDimensionCategoryBusiness},
				{name: "系统", code: "data_permission_dimension_category_system", value: model.DataDimensionCategorySystem},
			},
		},
		{
			name: "数据权限维度值类型",
			code: "data_permission_dimension_value_type",
			items: []systemDictItemSeed{
				{name: "大整数", code: "data_permission_dimension_value_type_bigint", value: model.DataDimensionValueTypeBigint},
				{name: "字符串", code: "data_permission_dimension_value_type_string", value: model.DataDimensionValueTypeString},
			},
		},
		{
			name: "数据资源操作",
			code: "data_permission_operation",
			items: []systemDictItemSeed{
				{name: "查询", code: "data_permission_operation_query", value: model.DataPermissionOperationQuery},
				{name: "详情", code: "data_permission_operation_detail", value: model.DataPermissionOperationDetail},
				{name: "新增", code: "data_permission_operation_create", value: model.DataPermissionOperationCreate},
				{name: "更新", code: "data_permission_operation_update", value: model.DataPermissionOperationUpdate},
				{name: "删除", code: "data_permission_operation_delete", value: model.DataPermissionOperationDelete},
				{name: "导出", code: "data_permission_operation_export", value: model.DataPermissionOperationExport},
				{name: "运行", code: "data_permission_operation_run", value: model.DataPermissionOperationRun},
			},
		},
		{
			name: "数据归属绑定类型",
			code: "data_permission_ownership_binding_type",
			items: []systemDictItemSeed{
				{name: "元数据字段", code: "data_permission_ownership_binding_type_metadata_field", value: model.DataOwnershipBindingTypeMetadataField},
				{name: "注册字段", code: "data_permission_ownership_binding_type_registered_field", value: model.DataOwnershipBindingTypeRegisteredField},
			},
		},
		{
			name: "数据权限范围来源",
			code: "data_permission_scope_source",
			items: []systemDictItemSeed{
				{name: "当前用户", code: "data_permission_scope_source_current_user", value: model.DataPolicyScopeSourceCurrentUser},
				{name: "当前人员", code: "data_permission_scope_source_current_employee", value: model.DataPolicyScopeSourceCurrentEmployee},
				{name: "有效法人集合", code: "data_permission_scope_source_effective_legal_entities", value: model.DataPolicyScopeSourceEffectiveLegalEntities},
				{name: "有效组织集合", code: "data_permission_scope_source_effective_org_units", value: model.DataPolicyScopeSourceEffectiveOrgUnits},
				{name: "指定值", code: "data_permission_scope_source_specified_values", value: model.DataPolicyScopeSourceSpecifiedValues},
				{name: "Provider主体范围", code: "data_permission_scope_source_provider_subject_scope", value: model.DataPolicyScopeSourceProviderSubjectScope},
			},
		},
		{
			name: "数据权限关系",
			code: "data_permission_relation",
			items: []systemDictItemSeed{
				{name: "精确匹配", code: "data_permission_relation_exact", value: model.DataPolicyRelationExact},
				{name: "本级及下级", code: "data_permission_relation_self_and_descendants", value: model.DataPolicyRelationSelfAndDescendants},
			},
		},
		{
			name: "数据权限操作符",
			code: "data_permission_operator",
			items: []systemDictItemSeed{
				{name: "等于", code: "data_permission_operator_eq", value: model.DataPolicyOperatorEqual},
				{name: "包含于", code: "data_permission_operator_in", value: model.DataPolicyOperatorIn},
			},
		},
	}
}

func dataPermissionDimensionSeeds() []dataPermissionDimensionSeed {
	return []dataPermissionDimensionSeed{
		{
			code:         "legal_entity",
			name:         "法人主体",
			category:     model.DataDimensionCategoryOrganization,
			valueType:    model.DataDimensionValueTypeBigint,
			providerCode: "organization",
			selectorType: "legal_entity",
			description:  "法人主体维度",
		},
		{
			code:         "management_org",
			name:         "管理组织",
			category:     model.DataDimensionCategoryOrganization,
			valueType:    model.DataDimensionValueTypeBigint,
			providerCode: "organization",
			selectorType: "org_unit",
			description:  "管理组织维度",
		},
		{
			code:         "employee",
			name:         "企业人员",
			category:     model.DataDimensionCategoryEmployee,
			valueType:    model.DataDimensionValueTypeBigint,
			providerCode: "organization",
			selectorType: "employee",
			description:  "企业人员维度",
		},
	}
}

func seedDataPermissionDimension(
	db *gorm.DB,
	sf *utils.Snowflake,
	seed dataPermissionDimensionSeed,
) error {
	var existing model.DataDimensionDefinition
	err := db.Unscoped().Where("code = ?", seed.code).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	id, err := newMigrationID(sf)
	if err != nil {
		return err
	}
	selectorType := seed.selectorType
	return db.Create(&model.DataDimensionDefinition{
		Basic:        model.Basic{Id: id, State: true},
		Code:         seed.code,
		Name:         seed.name,
		Category:     seed.category,
		ValueType:    seed.valueType,
		ProviderCode: seed.providerCode,
		SelectorType: &selectorType,
		Description:  seed.description,
	}).Error
}
