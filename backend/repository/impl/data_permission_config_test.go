package impl

import (
	"sync/atomic"
	"testing"

	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestDataPermissionConfigRepositoriesQueryAndStableKeys(t *testing.T) {
	db := openDataPermissionConfigTestDB(t)
	fixtures := createDataPermissionConfigFixtures(t, db)
	primaryDB := &database.PrimaryDB{DB: db}

	dimensionRepo := NewDataDimensionDefinitionRepositoryImpl(primaryDB)
	resourceRepo := NewDataResourceRepositoryImpl(primaryDB)
	operationRepo := NewDataResourceOperationRepositoryImpl(primaryDB)
	ownershipRepo := NewDataOwnershipFieldRepositoryImpl(primaryDB)
	policyRepo := NewDataPolicyRepositoryImpl(primaryDB)
	ruleRepo := NewDataPolicyRuleRepositoryImpl(primaryDB)
	grantRepo := NewDataGrantRepositoryImpl(primaryDB)

	t.Run("dimension", func(t *testing.T) {
		query := request.DataDimensionDefinitionQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{
				Page: 1,
				Num:  1,
				Expressions: []request.ExpressionGroup{{
					Logic: enum.And,
					Rules: []request.QueryRule{{
						Field:          "code",
						ExpressionType: enum.Eq,
						Value:          fixtures.dimension.Code,
						Type:           enum.VarcharFieldType,
					}},
				}},
				QuickQuery: &request.QuickQuery{
					Keyword: "组织",
				},
			},
			Category: model.DataDimensionCategoryOrganization,
		}
		basic := query.ToBasic()
		result, err := dimensionRepo.GetDataDimensionDefinitionList(nil, &basic, dataPermissionConfigTestTable(
			"sys_data_dimension_definition",
			map[string]enum.SysTableFieldType{
				"code":     enum.VarcharFieldType,
				"name":     enum.VarcharFieldType,
				"category": enum.VarcharFieldType,
			},
			"code", "name",
		))
		if err != nil {
			t.Fatalf("query dimensions: %v", err)
		}
		assertConfigPage(t, result.Total, len(result.Data), 1, 1)
		if result.Data[0].Code != fixtures.dimension.Code ||
			result.Data[0].Description != "" {
			t.Fatalf("unexpected dimension query result: %+v", result.Data[0])
		}
		stable, err := dimensionRepo.FindByField("code", fixtures.dimension.Code)
		if err != nil || stable.Id != fixtures.dimension.Id {
			t.Fatalf("find dimension stable key: value=%+v err=%v", stable, err)
		}
		byID, err := dimensionRepo.FindById(fixtures.dimension.Id)
		if err != nil || byID.Code != fixtures.dimension.Code {
			t.Fatalf("find dimension by ID: value=%+v err=%v", byID, err)
		}
		batch, err := dimensionRepo.FindListByFieldIn("id", []int{fixtures.dimension.Id, fixtures.employeeDimension.Id})
		if err != nil || len(batch) != 2 {
			t.Fatalf("batch dimensions: values=%+v err=%v", batch, err)
		}
	})

	t.Run("resource", func(t *testing.T) {
		disabled := false
		query := request.DataResourceQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{Page: 1, Num: 1},
			ResourceType:                 model.DataResourceTypeLowCodeTable,
			PermissionEnabled:            &disabled,
		}
		basic := query.ToBasic()
		result, err := resourceRepo.GetDataResourceList(nil, &basic, dataPermissionConfigTestTable(
			"sys_data_resource",
			map[string]enum.SysTableFieldType{
				"resource_code":      enum.VarcharFieldType,
				"name":               enum.VarcharFieldType,
				"resource_type":      enum.VarcharFieldType,
				"permission_enabled": enum.BooleanFieldType,
			},
			"resource_code", "name",
		))
		if err != nil {
			t.Fatalf("query resources: %v", err)
		}
		assertConfigPage(t, result.Total, len(result.Data), 1, 1)
		if result.Data[0].ResourceCode != fixtures.resource.ResourceCode ||
			result.Data[0].Description != "" {
			t.Fatalf("unexpected resource query result: %+v", result.Data[0])
		}
		stable, err := resourceRepo.FindByField("resource_code", fixtures.resource.ResourceCode)
		if err != nil || stable.Id != fixtures.resource.Id {
			t.Fatalf("find resource stable key: value=%+v err=%v", stable, err)
		}
		assertResourceReadMethods(t, resourceRepo, fixtures)
	})

	t.Run("resource_operation", func(t *testing.T) {
		query := request.DataResourceOperationQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{Page: 1, Num: 1},
			ResourceId:                   &fixtures.resource.Id,
			Operation:                    model.DataPermissionOperationQuery,
		}
		basic := query.ToBasic()
		result, err := operationRepo.GetDataResourceOperationList(nil, &basic, dataPermissionConfigTestTable(
			"sys_data_resource_operation",
			map[string]enum.SysTableFieldType{
				"resource_id": enum.BigIntFieldType,
				"operation":   enum.VarcharFieldType,
			},
			"operation",
		))
		if err != nil {
			t.Fatalf("query operations: %v", err)
		}
		assertConfigPage(t, result.Total, len(result.Data), 1, 1)
		stable, err := operationRepo.FindByStableKey(nil, fixtures.resource.Id, model.DataPermissionOperationQuery)
		if err != nil || stable.Id != fixtures.operation.Id {
			t.Fatalf("find operation stable key: value=%+v err=%v", stable, err)
		}
		byID, err := operationRepo.FindById(fixtures.operation.Id)
		if err != nil || byID.Operation != fixtures.operation.Operation {
			t.Fatalf("find operation by ID: value=%+v err=%v", byID, err)
		}
		batch, err := operationRepo.FindListByFieldIn("id", []int{fixtures.operation.Id, fixtures.detailOperation.Id})
		if err != nil || len(batch) != 2 {
			t.Fatalf("batch operations: values=%+v err=%v", batch, err)
		}
	})

	t.Run("ownership", func(t *testing.T) {
		query := request.DataOwnershipFieldQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{Page: 1, Num: 1},
			ResourceId:                   &fixtures.resource.Id,
			DimensionId:                  &fixtures.dimension.Id,
			BindingType:                  model.DataOwnershipBindingTypeMetadataField,
		}
		basic := query.ToBasic()
		result, err := ownershipRepo.GetDataOwnershipFieldList(nil, &basic, dataPermissionConfigTestTable(
			"sys_data_ownership_field",
			map[string]enum.SysTableFieldType{
				"resource_id":    enum.BigIntFieldType,
				"dimension_id":   enum.BigIntFieldType,
				"binding_type":   enum.VarcharFieldType,
				"ownership_code": enum.VarcharFieldType,
			},
			"ownership_code",
		))
		if err != nil {
			t.Fatalf("query ownership fields: %v", err)
		}
		assertConfigPage(t, result.Total, len(result.Data), 1, 1)
		stable, err := ownershipRepo.FindByStableKey(nil, fixtures.resource.Id, fixtures.ownership.OwnershipCode)
		if err != nil || stable.Id != fixtures.ownership.Id {
			t.Fatalf("find ownership stable key: value=%+v err=%v", stable, err)
		}
		byID, err := ownershipRepo.FindById(fixtures.ownership.Id)
		if err != nil || byID.OwnershipCode != fixtures.ownership.OwnershipCode {
			t.Fatalf("find ownership by ID: value=%+v err=%v", byID, err)
		}
		batch, err := ownershipRepo.FindListByFieldIn("id", []int{fixtures.ownership.Id, fixtures.secondOwnership.Id})
		if err != nil || len(batch) != 2 {
			t.Fatalf("batch ownership fields: values=%+v err=%v", batch, err)
		}
	})

	t.Run("policy", func(t *testing.T) {
		query := request.DataPolicyQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{Page: 1, Num: 1},
			PolicyType:                   model.DataPolicyTypeRuleSet,
		}
		basic := query.ToBasic()
		result, err := policyRepo.GetDataPolicyList(nil, &basic, dataPermissionConfigTestTable(
			"sys_data_policy",
			map[string]enum.SysTableFieldType{
				"code":        enum.VarcharFieldType,
				"name":        enum.VarcharFieldType,
				"policy_type": enum.VarcharFieldType,
			},
			"code", "name",
		))
		if err != nil {
			t.Fatalf("query policies: %v", err)
		}
		assertConfigPage(t, result.Total, len(result.Data), 1, 1)
		stable, err := policyRepo.FindByField("code", fixtures.policy.Code)
		if err != nil || stable.Id != fixtures.policy.Id {
			t.Fatalf("find policy stable key: value=%+v err=%v", stable, err)
		}
		byID, err := policyRepo.FindById(fixtures.policy.Id)
		if err != nil || byID.Code != fixtures.policy.Code {
			t.Fatalf("find policy by ID: value=%+v err=%v", byID, err)
		}
		batch, err := policyRepo.FindListByFieldIn("id", []int{fixtures.policy.Id, fixtures.allPolicy.Id})
		if err != nil || len(batch) != 2 {
			t.Fatalf("batch policies: values=%+v err=%v", batch, err)
		}
	})

	t.Run("policy_rule", func(t *testing.T) {
		query := request.DataPolicyRuleQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{Page: 1, Num: 1},
			PolicyId:                     &fixtures.policy.Id,
			DimensionId:                  &fixtures.dimension.Id,
			ScopeSource:                  model.DataPolicyScopeSourceEffectiveOrgUnits,
		}
		basic := query.ToBasic()
		result, err := ruleRepo.GetDataPolicyRuleList(nil, &basic, dataPermissionConfigTestTable(
			"sys_data_policy_rule",
			map[string]enum.SysTableFieldType{
				"policy_id":    enum.BigIntFieldType,
				"dimension_id": enum.BigIntFieldType,
				"scope_source": enum.VarcharFieldType,
				"sequence":     enum.IntFieldType,
			},
		))
		if err != nil {
			t.Fatalf("query policy rules: %v", err)
		}
		assertConfigPage(t, result.Total, len(result.Data), 1, 1)
		stable, err := ruleRepo.FindByStableKey(nil, fixtures.policy.Id, fixtures.rule.Sequence)
		if err != nil || stable.Id != fixtures.rule.Id {
			t.Fatalf("find rule stable key: value=%+v err=%v", stable, err)
		}
		byID, err := ruleRepo.FindById(fixtures.rule.Id)
		if err != nil || byID.Sequence != fixtures.rule.Sequence {
			t.Fatalf("find rule by ID: value=%+v err=%v", byID, err)
		}
		batch, err := ruleRepo.FindListByFieldIn("id", []int{fixtures.rule.Id, fixtures.secondRule.Id})
		if err != nil || len(batch) != 2 {
			t.Fatalf("batch policy rules: values=%+v err=%v", batch, err)
		}
	})

	t.Run("grant", func(t *testing.T) {
		query := request.DataGrantQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{Page: 1, Num: 1},
			SubjectType:                  model.DataGrantSubjectTypeRole,
			SubjectId:                    &fixtures.grant.SubjectId,
			ResourceId:                   &fixtures.resource.Id,
			Operation:                    model.DataPermissionOperationQuery,
			PolicyId:                     &fixtures.policy.Id,
		}
		basic := query.ToBasic()
		result, err := grantRepo.GetDataGrantList(nil, &basic, dataPermissionConfigTestTable(
			"sys_data_grant",
			map[string]enum.SysTableFieldType{
				"subject_type": enum.VarcharFieldType,
				"subject_id":   enum.BigIntFieldType,
				"resource_id":  enum.BigIntFieldType,
				"operation":    enum.VarcharFieldType,
				"policy_id":    enum.BigIntFieldType,
			},
		))
		if err != nil {
			t.Fatalf("query grants: %v", err)
		}
		assertConfigPage(t, result.Total, len(result.Data), 1, 1)
		stable, err := grantRepo.FindByStableKey(
			nil,
			fixtures.grant.SubjectType,
			fixtures.grant.SubjectId,
			fixtures.grant.ResourceId,
			fixtures.grant.Operation,
			fixtures.grant.PolicyId,
		)
		if err != nil || stable.Id != fixtures.grant.Id {
			t.Fatalf("find grant stable key: value=%+v err=%v", stable, err)
		}
		byID, err := grantRepo.FindById(fixtures.grant.Id)
		if err != nil || byID.SubjectId != fixtures.grant.SubjectId {
			t.Fatalf("find grant by ID: value=%+v err=%v", byID, err)
		}
		batch, err := grantRepo.FindListByFieldIn("id", []int{fixtures.grant.Id, fixtures.secondGrant.Id})
		if err != nil || len(batch) != 2 {
			t.Fatalf("batch grants: values=%+v err=%v", batch, err)
		}
	})
}

func TestDataPermissionConfigRepositoryUsesControlledMetadataAndNoPreload(t *testing.T) {
	db := openDataPermissionConfigTestDB(t)
	fixtures := createDataPermissionConfigFixtures(t, db)
	repo := NewDataResourceRepositoryImpl(&database.PrimaryDB{DB: db})

	var queryCount atomic.Int32
	callbackName := "test:data-permission-config-query-count"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(*gorm.DB) {
		queryCount.Add(1)
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})

	table := dataPermissionConfigTestTable(
		"client_supplied_table_name",
		map[string]enum.SysTableFieldType{
			"resource_code": enum.VarcharFieldType,
			"name":          enum.VarcharFieldType,
			"description":   enum.VarcharFieldType,
		},
		"resource_code", "name", "description",
	)
	query := request.DataResourceQueryReq{
		DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{
			Page: 1,
			Num:  10,
			QuickQuery: &request.QuickQuery{
				Keyword: fixtures.resource.Description,
			},
		},
	}
	basic := query.ToBasic()
	result, err := repo.GetDataResourceList(nil, &basic, table)
	if err != nil {
		t.Fatalf("query controlled metadata: %v", err)
	}
	if result.Total != 0 || len(result.Data) != 0 {
		t.Fatalf("restricted description was accepted for quick query: %+v", result)
	}
	if got := queryCount.Load(); got != 2 {
		t.Fatalf("query issued %d statements, want count + data only (no preload)", got)
	}
}

func TestDataPermissionConfigRepositoryPropagatesDatabaseError(t *testing.T) {
	db := openDataPermissionConfigTestDB(t)
	repo := NewDataPolicyRepositoryImpl(&database.PrimaryDB{DB: db})
	if err := db.Migrator().DropTable(&model.DataPolicy{}); err != nil {
		t.Fatalf("drop policy table: %v", err)
	}
	if _, err := repo.FindByField("code", "missing"); err == nil {
		t.Fatal("database error was swallowed")
	}
}

type dataPermissionConfigFixtures struct {
	dimension         model.DataDimensionDefinition
	employeeDimension model.DataDimensionDefinition
	resource          model.DataResource
	serviceResource   model.DataResource
	operation         model.DataResourceOperation
	detailOperation   model.DataResourceOperation
	ownership         model.DataOwnershipField
	secondOwnership   model.DataOwnershipField
	policy            model.DataPolicy
	allPolicy         model.DataPolicy
	rule              model.DataPolicyRule
	secondRule        model.DataPolicyRule
	grant             model.DataGrant
	secondGrant       model.DataGrant
}

func createDataPermissionConfigFixtures(t *testing.T, db *gorm.DB) dataPermissionConfigFixtures {
	t.Helper()
	tableId := 501
	serviceCode := "shipment"
	tableFieldId := 601
	adapterFieldCode := "owner_employee_id"
	fixtures := dataPermissionConfigFixtures{
		dimension: model.DataDimensionDefinition{
			Basic:        model.Basic{Id: 1, State: true},
			Code:         "org_unit",
			Name:         "组织维度",
			Category:     model.DataDimensionCategoryOrganization,
			ValueType:    model.DataDimensionValueTypeBigint,
			ProviderCode: "organization",
			Description:  "internal-dimension-description",
		},
		employeeDimension: model.DataDimensionDefinition{
			Basic:        model.Basic{Id: 2, State: true},
			Code:         "employee",
			Name:         "人员维度",
			Category:     model.DataDimensionCategoryEmployee,
			ValueType:    model.DataDimensionValueTypeBigint,
			ProviderCode: "organization",
		},
		resource: model.DataResource{
			Basic:             model.Basic{Id: 11, State: true},
			ResourceCode:      "shipment_order",
			Name:              "运输单",
			ResourceType:      model.DataResourceTypeLowCodeTable,
			TableId:           &tableId,
			AdapterCode:       "metadata",
			PermissionEnabled: false,
			Description:       "internal-resource-description",
		},
		serviceResource: model.DataResource{
			Basic:             model.Basic{Id: 12, State: true},
			ResourceCode:      "shipment_service",
			Name:              "运输服务",
			ResourceType:      model.DataResourceTypeBusinessService,
			ServiceCode:       &serviceCode,
			AdapterCode:       "registered_service",
			PermissionEnabled: true,
		},
		operation: model.DataResourceOperation{
			Basic:             model.Basic{Id: 21, State: true},
			ResourceId:        11,
			Operation:         model.DataPermissionOperationQuery,
			PermissionEnabled: true,
		},
		detailOperation: model.DataResourceOperation{
			Basic:             model.Basic{Id: 22, State: true},
			ResourceId:        11,
			Operation:         model.DataPermissionOperationDetail,
			PermissionEnabled: true,
		},
		ownership: model.DataOwnershipField{
			Basic:         model.Basic{Id: 31, State: true},
			ResourceId:    11,
			OwnershipCode: "org_unit",
			DimensionId:   1,
			BindingType:   model.DataOwnershipBindingTypeMetadataField,
			TableFieldId:  &tableFieldId,
			ValueType:     model.DataDimensionValueTypeBigint,
		},
		secondOwnership: model.DataOwnershipField{
			Basic:            model.Basic{Id: 32, State: true},
			ResourceId:       11,
			OwnershipCode:    "employee",
			DimensionId:      2,
			BindingType:      model.DataOwnershipBindingTypeRegisteredField,
			AdapterFieldCode: &adapterFieldCode,
			ValueType:        model.DataDimensionValueTypeBigint,
		},
		policy: model.DataPolicy{
			Basic:      model.Basic{Id: 41, State: true},
			Code:       "own_org",
			Name:       "本组织",
			PolicyType: model.DataPolicyTypeRuleSet,
		},
		allPolicy: model.DataPolicy{
			Basic:      model.Basic{Id: 42, State: true},
			Code:       "all",
			Name:       "全部",
			PolicyType: model.DataPolicyTypeAll,
		},
		rule: model.DataPolicyRule{
			Basic:           model.Basic{Id: 51, State: true},
			PolicyId:        41,
			Sequence:        1,
			DimensionId:     1,
			OwnershipCode:   "org_unit",
			ScopeSource:     model.DataPolicyScopeSourceEffectiveOrgUnits,
			Relation:        model.DataPolicyRelationExact,
			Operator:        model.DataPolicyOperatorIn,
			SpecifiedValues: datatypes.JSON([]byte(`[]`)),
		},
		secondRule: model.DataPolicyRule{
			Basic:           model.Basic{Id: 52, State: true},
			PolicyId:        41,
			Sequence:        2,
			DimensionId:     2,
			OwnershipCode:   "employee",
			ScopeSource:     model.DataPolicyScopeSourceCurrentEmployee,
			Relation:        model.DataPolicyRelationExact,
			Operator:        model.DataPolicyOperatorEqual,
			SpecifiedValues: datatypes.JSON([]byte(`[]`)),
		},
		grant: model.DataGrant{
			Basic:       model.Basic{Id: 61, State: true},
			SubjectType: model.DataGrantSubjectTypeRole,
			SubjectId:   7,
			ResourceId:  11,
			Operation:   model.DataPermissionOperationQuery,
			PolicyId:    41,
		},
		secondGrant: model.DataGrant{
			Basic:       model.Basic{Id: 62, State: true},
			SubjectType: model.DataGrantSubjectTypeUser,
			SubjectId:   8,
			ResourceId:  11,
			Operation:   model.DataPermissionOperationDetail,
			PolicyId:    42,
		},
	}

	for _, value := range []any{
		&fixtures.dimension,
		&fixtures.employeeDimension,
		&fixtures.resource,
		&fixtures.serviceResource,
		&fixtures.operation,
		&fixtures.detailOperation,
		&fixtures.ownership,
		&fixtures.secondOwnership,
		&fixtures.policy,
		&fixtures.allPolicy,
		&fixtures.rule,
		&fixtures.secondRule,
		&fixtures.grant,
		&fixtures.secondGrant,
	} {
		testutil.MustCreate(t, db, value)
	}
	return fixtures
}

func openDataPermissionConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.OpenSQLite(
		t,
		&model.DataDimensionDefinition{},
		&model.DataResource{},
		&model.DataResourceOperation{},
		&model.DataOwnershipField{},
		&model.DataPolicy{},
		&model.DataPolicyRule{},
		&model.DataGrant{},
	)
}

func dataPermissionConfigTestTable(
	tableCode string,
	fieldTypes map[string]enum.SysTableFieldType,
	quickFields ...string,
) model.SysTable {
	quick := make(map[string]struct{}, len(quickFields))
	for _, field := range quickFields {
		quick[field] = struct{}{}
	}
	fields := make([]model.SysTableField, 0, len(fieldTypes))
	for code, fieldType := range fieldTypes {
		_, isQuick := quick[code]
		fields = append(fields, model.SysTableField{
			FieldCode:        code,
			FieldType:        fieldType,
			IsListShow:       true,
			IsQuickSearch:    isQuick,
			IsAdvancedSearch: true,
			IsSort:           true,
		})
	}
	return model.SysTable{TableCode: tableCode, TableFields: fields}
}

func assertConfigPage(t *testing.T, total, rows, expectedTotal, expectedRows int) {
	t.Helper()
	if total != expectedTotal || rows != expectedRows {
		t.Fatalf(
			"unexpected page total=%d rows=%d, want total=%d rows=%d",
			total, rows, expectedTotal, expectedRows,
		)
	}
}

func assertResourceReadMethods(
	t *testing.T,
	repo *DataResourceRepositoryImpl,
	fixtures dataPermissionConfigFixtures,
) {
	t.Helper()
	byID, err := repo.FindById(fixtures.resource.Id)
	if err != nil || byID.ResourceCode != fixtures.resource.ResourceCode {
		t.Fatalf("find resource by ID: value=%+v err=%v", byID, err)
	}
	batch, err := repo.FindListByFieldIn("id", []int{fixtures.resource.Id, fixtures.serviceResource.Id})
	if err != nil || len(batch) != 2 {
		t.Fatalf("batch resources: values=%+v err=%v", batch, err)
	}
}
