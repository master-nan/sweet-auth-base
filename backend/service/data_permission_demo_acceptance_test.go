package service

import (
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"

	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/database"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestDataPermissionDemoAcceptanceDescendantPolicyResolves(t *testing.T) {
	resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
	structureCode := "DP-ACCEPTANCE-MGMT"
	if err := db.Model(&model.DataPolicyRule{}).
		Where("id = ?", fixtures.rule.Id).
		Updates(map[string]any{
			"relation":       model.DataPolicyRelationSelfAndDescendants,
			"structure_code": structureCode,
		}).Error; err != nil {
		t.Fatalf("configure acceptance descendant Rule: %v", err)
	}

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve descendant policy: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered {
		t.Fatalf("decision = %s, want filtered", result.Decision())
	}
	if provider.calls != 1 ||
		provider.relation != model.DataPolicyRelationSelfAndDescendants ||
		provider.structureCode != structureCode {
		t.Fatalf("unexpected Provider request: calls=%d relation=%s structure=%s", provider.calls, provider.relation, provider.structureCode)
	}
	groups := result.ConditionGroups()
	if len(groups) != 1 || len(groups[0].Conditions()) != 1 {
		t.Fatalf("unexpected descendant conditions: %+v", groups)
	}
	if got := groups[0].Conditions()[0].BigintValues(); len(got) != 2 || got[0] != 11 || got[1] != 12 {
		t.Fatalf("descendant values = %v, want [11 12]", got)
	}
}

func TestDataPermissionDemoAcceptanceEndToEnd(t *testing.T) {
	service, table := newDataPermissionDemoAcceptanceService(t)
	tests := []struct {
		name         string
		userId       int
		wantOrders   []string
		allowedIds   []int
		forbiddenIds []int
	}{
		{
			name: "east manager includes descendants", userId: 1001,
			wantOrders: []string{"ORD001", "ORD002"},
			allowedIds: []int{910001, 910002}, forbiddenIds: []int{910003},
		},
		{
			name: "south manager remains isolated", userId: 1002,
			wantOrders: []string{"ORD003"},
			allowedIds: []int{910003}, forbiddenIds: []int{910001, 910002},
		},
		{
			name: "ungranted user receives none", userId: 1003,
			wantOrders: []string{},
			allowedIds: []int{}, forbiddenIds: []int{910001, 910002, 910003},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := lowCodeRuntimeGinContext(tt.userId)
			result, err := service.QueryWithDataPermission(
				ctx,
				&request.Basic{
					Page:  1,
					Num:   10,
					Order: request.Order{Field: "id", IsAsc: true},
				},
				table,
				model.DataPermissionOperationQuery,
			)
			if err != nil {
				t.Fatalf("query acceptance rows: %v", err)
			}
			if got := demoAcceptanceOrderNumbers(result.Data); !reflect.DeepEqual(got, tt.wantOrders) {
				t.Fatalf("rows = %v, want %v", got, tt.wantOrders)
			}
			if result.Total != len(tt.wantOrders) {
				t.Fatalf("total = %d, want %d", result.Total, len(tt.wantOrders))
			}

			for _, id := range tt.allowedIds {
				detail, detailErr := service.GetByIdWithDataPermission(
					ctx, table, id,
					model.DataPermissionOperationDetail,
				)
				if detailErr != nil || detail["order_no"] == "" {
					t.Fatalf("allowed detail %d: detail=%v err=%v", id, detail, detailErr)
				}
			}
			for _, id := range tt.forbiddenIds {
				_, detailErr := service.GetByIdWithDataPermission(
					ctx, table, id,
					model.DataPermissionOperationDetail,
				)
				if !errors.Is(detailErr, myerrors.ErrDataNotFound) {
					t.Fatalf("forbidden detail %d leaked existence: %v", id, detailErr)
				}
			}
		})
	}
}

func newDataPermissionDemoAcceptanceService(
	t *testing.T,
) (*GeneralizationService, model.SysTable) {
	t.Helper()
	db := testutil.OpenSQLite(
		t,
		&model.SysTable{},
		&model.SysTableField{},
		&model.DataDimensionDefinition{},
		&model.DataResource{},
		&model.DataResourceOperation{},
		&model.DataOwnershipField{},
		&model.DataPolicy{},
		&model.DataPolicyRule{},
		&model.DataGrant{},
	)
	if err := db.Exec(`CREATE TABLE demo_transport_order (
		id INTEGER PRIMARY KEY,
		order_no TEXT NOT NULL,
		owner_org_id INTEGER NOT NULL,
		amount NUMERIC NOT NULL,
		gmt_delete DATETIME NULL
	)`).Error; err != nil {
		t.Fatalf("create demo transport order: %v", err)
	}
	if err := db.Exec(`INSERT INTO demo_transport_order
		(id, order_no, owner_org_id, amount) VALUES
		(910001, 'ORD001', 101, 1000),
		(910002, 'ORD002', 102, 2000),
		(910003, 'ORD003', 103, 3000)`).Error; err != nil {
		t.Fatalf("seed demo transport orders: %v", err)
	}

	table := demoAcceptanceTable()
	tableRecord := table
	tableRecord.TableFields = nil
	testutil.MustCreate(t, db, &tableRecord)
	for index := range table.TableFields {
		field := table.TableFields[index]
		testutil.MustCreate(t, db, &field)
	}
	demoAcceptancePermissionConfig(t, db, table)

	primaryDB := &database.PrimaryDB{DB: db}
	dimensionRepo := impl.NewDataDimensionDefinitionRepositoryImpl(primaryDB)
	resourceRepo := impl.NewDataResourceRepositoryImpl(primaryDB)
	operationRepo := impl.NewDataResourceOperationRepositoryImpl(primaryDB)
	ownershipRepo := impl.NewDataOwnershipFieldRepositoryImpl(primaryDB)
	policyRepo := impl.NewDataPolicyRepositoryImpl(primaryDB)
	ruleRepo := impl.NewDataPolicyRuleRepositoryImpl(primaryDB)
	grantRepo := impl.NewDataGrantRepositoryImpl(primaryDB)

	dimensionRuntime := newDimensionProviderRuntime(
		dimensionRepo.FindByCode,
		demoAcceptanceOrganizationScope,
		demoAcceptanceOrganizationDescendants,
	)
	resolver := NewDataPermissionPolicyResolver(
		resourceRepo,
		operationRepo,
		grantRepo,
		policyRepo,
		ruleRepo,
		ownershipRepo,
		dimensionRepo,
		dimensionRuntime,
	)
	subjectBuilder := newSubjectContextBuilder(
		func(userId int) (model.SysUser, error) {
			if userId < 1001 || userId > 1003 {
				return model.SysUser{}, gorm.ErrRecordNotFound
			}
			return model.SysUser{Basic: model.Basic{Id: userId, State: true}}, nil
		},
		func(userId int) ([]model.SysRole, error) {
			roleId := 700
			if userId == 1003 {
				roleId = 701
			}
			return []model.SysRole{{Basic: model.Basic{Id: roleId, State: true}}}, nil
		},
		func(_ *gin.Context, userId int) (response.OrgEmployeeContextRes, error) {
			employeeId := userId - 500
			return response.NewOrgEmployeeContextRes(userId, &employeeId), nil
		},
		func() time.Time {
			return time.Date(2026, time.August, 3, 9, 0, 0, 0, model.AppLocation())
		},
	)
	metadataAdapter, err := datapermission.NewMetadataFieldAdapter(
		impl.NewDataPermissionMetadataReaderImpl(primaryDB),
	)
	if err != nil {
		t.Fatalf("create Metadata Adapter: %v", err)
	}
	runtime := newLowCodeDataPermissionRuntime(
		resourceRepo.ListByTableId,
		ownershipRepo.ListByResource,
		subjectBuilder.Build,
		resolver.Resolve,
		metadataAdapter.Apply,
	)
	generalizationRepo := impl.NewGeneralizationRepositoryImpl(primaryDB)
	return NewGeneralizationServiceWithDataPermission(generalizationRepo, nil, runtime), table
}

func demoAcceptanceTable() model.SysTable {
	const tableId = 9200
	return model.SysTable{
		Basic:     model.Basic{Id: tableId, State: true},
		TableName: "数据权限验收运输订单",
		TableCode: "demo_transport_order",
		TableFields: []model.SysTableField{
			{Basic: model.Basic{Id: 9201, State: true}, TableId: tableId, FieldName: "主键ID", FieldCode: "id", FieldType: enum.BigIntFieldType, IsPrimaryKey: true, IsListShow: true},
			{Basic: model.Basic{Id: 9202, State: true}, TableId: tableId, FieldName: "运单号", FieldCode: "order_no", FieldType: enum.VarcharFieldType, IsListShow: true},
			{Basic: model.Basic{Id: 9203, State: true}, TableId: tableId, FieldName: "所属管理组织", FieldCode: "owner_org_id", FieldType: enum.BigIntFieldType, FieldCategory: enum.NormalField, IsAdvancedSearch: true, IsListShow: true},
			{Basic: model.Basic{Id: 9204, State: true}, TableId: tableId, FieldName: "金额", FieldCode: "amount", FieldType: enum.FloatFieldType, IsListShow: true},
			{Basic: model.Basic{Id: 9205, State: true}, TableId: tableId, FieldName: "删除时间", FieldCode: "gmt_delete", FieldType: enum.DatetimeFieldType},
		},
	}
}

func demoAcceptancePermissionConfig(t *testing.T, db *gorm.DB, table model.SysTable) {
	t.Helper()
	structureCode := "DP-ACCEPTANCE-MGMT"
	resource := model.DataResource{
		Basic: model.Basic{Id: 9300, State: true}, ResourceCode: "transport_order",
		Name: "运输订单", ResourceType: model.DataResourceTypeLowCodeTable,
		TableId: &table.Id, AdapterCode: "metadata_filter", PermissionEnabled: true,
	}
	dimension := model.DataDimensionDefinition{
		Basic: model.Basic{Id: 9301, State: true}, Code: datapermission.DimensionCodeManagementOrg,
		Name: "管理组织", Category: model.DataDimensionCategoryOrganization,
		ValueType: model.DataDimensionValueTypeBigint, ProviderCode: organizationDimensionProviderCode,
	}
	fieldId := 9203
	ownership := model.DataOwnershipField{
		Basic: model.Basic{Id: 9302, State: true}, ResourceId: resource.Id,
		OwnershipCode: "owner_org", DimensionId: dimension.Id,
		BindingType:  model.DataOwnershipBindingTypeMetadataField,
		TableFieldId: &fieldId, ValueType: model.DataDimensionValueTypeBigint,
	}
	policy := model.DataPolicy{
		Basic: model.Basic{Id: 9303, State: true}, Code: "own_org_and_descendants",
		Name: "本组织及下级组织", PolicyType: model.DataPolicyTypeRuleSet,
	}
	rule := model.DataPolicyRule{
		Basic: model.Basic{Id: 9304, State: true}, PolicyId: policy.Id, Sequence: 1,
		DimensionId: dimension.Id, OwnershipCode: ownership.OwnershipCode,
		ScopeSource: model.DataPolicyScopeSourceEffectiveOrgUnits,
		Relation:    model.DataPolicyRelationSelfAndDescendants,
		Operator:    model.DataPolicyOperatorIn, StructureCode: &structureCode,
	}
	testutil.MustCreate(t, db, &resource)
	testutil.MustCreate(t, db, &dimension)
	testutil.MustCreate(t, db, &ownership)
	testutil.MustCreate(t, db, &policy)
	testutil.MustCreate(t, db, &rule)
	for index, operation := range []string{model.DataPermissionOperationQuery, model.DataPermissionOperationDetail} {
		testutil.MustCreate(t, db, &model.DataResourceOperation{
			Basic: model.Basic{Id: 9310 + index, State: true}, ResourceId: resource.Id,
			Operation: operation, PermissionEnabled: true,
		})
		testutil.MustCreate(t, db, &model.DataGrant{
			Basic:       model.Basic{Id: 9320 + index, State: true},
			SubjectType: model.DataGrantSubjectTypeRole, SubjectId: 700,
			ResourceId: resource.Id, Operation: operation, PolicyId: policy.Id,
		})
	}
}

func demoAcceptanceOrganizationScope(
	_ *gin.Context,
	employeeId int,
	asOfDate string,
) (response.OrgEffectiveOrganizationScopeRes, error) {
	rootByEmployee := map[int]int{501: 101, 502: 103, 503: 101}
	rootId, exists := rootByEmployee[employeeId]
	if !exists {
		return response.OrgEffectiveOrganizationScopeRes{}, myerrors.ErrOrgEmployeeNotFound
	}
	return response.OrgEffectiveOrganizationScopeRes{
		EmployeeId: employeeId, AsOfDate: asOfDate,
		ScopeStatus: response.OrgEffectiveScopeResolved, AssignmentCount: 1,
		LegalEntityIds: []int{201}, OrgUnitIds: []int{rootId},
	}, nil
}

func demoAcceptanceOrganizationDescendants(
	_ *gin.Context,
	structureCode string,
	rootId int,
	asOfDate string,
	includeSelf bool,
) (response.OrgDescendantsRes, error) {
	if structureCode != "DP-ACCEPTANCE-MGMT" || !includeSelf {
		return response.OrgDescendantsRes{}, myerrors.ErrOrgStructureNotFound
	}
	itemsByRoot := map[int][]response.OrgRelationItemRes{
		101: {
			{OrgUnitId: 101, Distance: 0},
			{OrgUnitId: 102, Distance: 1},
		},
		103: {{OrgUnitId: 103, Distance: 0}},
	}
	items, exists := itemsByRoot[rootId]
	if !exists {
		return response.OrgDescendantsRes{}, myerrors.ErrOrgUnitNotFound
	}
	return response.OrgDescendantsRes{
		StructureCode: structureCode, OrgUnitId: rootId,
		AsOfDate: asOfDate, Items: items,
	}, nil
}

func demoAcceptanceOrderNumbers(rows []map[string]interface{}) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		value, _ := row["order_no"].(string)
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
