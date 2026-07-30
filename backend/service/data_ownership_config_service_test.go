package service

import (
	"backend/dto/request"
	"backend/internal/database"
	"backend/internal/datapermission"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"backend/enum"
	"gorm.io/gorm"
)

type testRegisteredOwnershipFieldValidator struct {
	err   error
	calls int
}

func (validator *testRegisteredOwnershipFieldValidator) ValidateBinding(
	_ datapermission.OwnershipFieldBindingValidation,
) error {
	validator.calls++
	return validator.err
}

func TestDataOwnershipConfigServiceCreateAndQuery(t *testing.T) {
	auditWriter := &testTransactionalAuditWriter{}
	service, db := newDataOwnershipConfigTestSubject(t, auditWriter, nil)
	resource, dimension, field := createMetadataOwnershipFixtures(t, db)

	result, err := service.CreateOwnership(
		dataResourceConfigContext(),
		metadataOwnershipCreateRequest(resource.Id, dimension.Id, field.Id),
	)
	if err != nil {
		t.Fatalf("create ownership: %v", err)
	}
	if result.ResourceId != resource.Id ||
		result.DimensionId != dimension.Id ||
		result.OwnershipCode != "owner_org" ||
		result.BindingTarget.ReferenceId == nil ||
		*result.BindingTarget.ReferenceId != field.Id {
		t.Fatalf("unexpected ownership response: %+v", result)
	}
	if result.Resource == nil || result.Resource.Code != resource.ResourceCode ||
		result.Dimension == nil || result.Dimension.Code != dimension.Code {
		t.Fatalf("missing ownership summaries: %+v", result)
	}

	detail, err := service.GetOwnership(dataResourceConfigContext(), result.Id)
	if err != nil {
		t.Fatalf("get ownership: %v", err)
	}
	payload, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal ownership: %v", err)
	}
	for _, forbidden := range []string{
		"table_field_id",
		"adapter_field_code",
		"gmt_delete",
		"create_user",
		"delete_user",
		"description",
	} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("ownership detail leaked %q: %s", forbidden, payload)
		}
	}

	page, err := service.PageOwnerships(
		dataResourceConfigContext(),
		request.DataOwnershipFieldQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{
				Page:       1,
				Num:        10,
				QuickQuery: &request.QuickQuery{Keyword: "owner"},
			},
			ResourceId: &resource.Id,
		},
		dataOwnershipConfigTable(),
	)
	if err != nil {
		t.Fatalf("page ownerships: %v", err)
	}
	if page.Total != 1 || len(page.Data) != 1 || page.Data[0].Id != result.Id {
		t.Fatalf("unexpected ownership page: %+v", page)
	}

	items, err := service.ListOwnershipsByResource(dataResourceConfigContext(), resource.Id)
	if err != nil {
		t.Fatalf("list ownerships by resource: %v", err)
	}
	if len(items) != 1 || items[0].OwnershipCode != "owner_org" {
		t.Fatalf("unexpected resource ownerships: %+v", items)
	}
	records := auditWriter.snapshot()
	if len(records) != 1 || records[0].Action != dataOwnershipCreateAction {
		t.Fatalf("unexpected audit records: %+v", records)
	}
}

func TestDataOwnershipConfigServiceCreateValidation(t *testing.T) {
	t.Run("resource must exist and be active", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		dimension := dataOwnershipDimensionFixture(201, model.DataDimensionValueTypeBigint)
		testutil.MustCreate(t, db, &dimension)

		req := registeredOwnershipCreateRequest(999, dimension.Id)
		_, err := service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataResourceNotFound)

		resource := dataOwnershipResourceFixture(101, model.DataResourceTypeBusinessService)
		testutil.MustCreate(t, db, &resource)
		mustSetState(t, db, &model.DataResource{}, resource.Id, false)
		req.ResourceId = resource.Id
		_, err = service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataResourceStateInvalid)
	})

	t.Run("dimension must exist and be active", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource := dataOwnershipResourceFixture(102, model.DataResourceTypeBusinessService)
		testutil.MustCreate(t, db, &resource)

		req := registeredOwnershipCreateRequest(resource.Id, 999)
		_, err := service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataDimensionNotFound)

		dimension := dataOwnershipDimensionFixture(202, model.DataDimensionValueTypeBigint)
		testutil.MustCreate(t, db, &dimension)
		mustSetState(t, db, &model.DataDimensionDefinition{}, dimension.Id, false)
		req.DimensionId = dimension.Id
		_, err = service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataDimensionNotFound)
	})

	t.Run("ownership code is unique within resource", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource := dataOwnershipResourceFixture(103, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(203, model.DataDimensionValueTypeBigint)
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &dimension)
		existing := dataOwnershipFixture(303, resource.Id, dimension.Id)
		testutil.MustCreate(t, db, &existing)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipDuplicate)
	})

	t.Run("binding types and registered codes are controlled", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource := dataOwnershipResourceFixture(104, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(204, model.DataDimensionValueTypeBigint)
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &dimension)

		req := registeredOwnershipCreateRequest(resource.Id, dimension.Id)
		req.BindingType = "report_source"
		_, err := service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipBindingInvalid)

		invalidCode := "orders.owner_org_id"
		req = registeredOwnershipCreateRequest(resource.Id, dimension.Id)
		req.BindingTarget.ReferenceCode = &invalidCode
		_, err = service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipRegisteredFieldInvalid)
	})

	t.Run("metadata field must exist belong to resource and be active", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource, dimension, field := createMetadataOwnershipFixtures(t, db)

		req := metadataOwnershipCreateRequest(resource.Id, dimension.Id, 999)
		_, err := service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipMetadataFieldNotFound)

		otherTableId := *resource.TableId + 1
		field.TableId = otherTableId
		if err = db.Model(&model.SysTableField{}).
			Where("id = ?", field.Id).
			Update("table_id", otherTableId).Error; err != nil {
			t.Fatalf("move metadata field: %v", err)
		}
		req.BindingTarget.ReferenceId = &field.Id
		_, err = service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipMetadataFieldMismatch)

		if err = db.Model(&model.SysTableField{}).
			Where("id = ?", field.Id).
			Updates(map[string]any{"table_id": *resource.TableId, "state": false}).Error; err != nil {
			t.Fatalf("disable metadata field: %v", err)
		}
		_, err = service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipMetadataFieldNotFound)
	})

	t.Run("value type must match dimension and metadata field", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource, dimension, field := createMetadataOwnershipFixtures(t, db)

		req := metadataOwnershipCreateRequest(resource.Id, dimension.Id, field.Id)
		req.ValueType = model.DataDimensionValueTypeString
		_, err := service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipValueTypeMismatch)

		if err = db.Model(&model.DataDimensionDefinition{}).
			Where("id = ?", dimension.Id).
			Update("value_type", model.DataDimensionValueTypeString).Error; err != nil {
			t.Fatalf("change dimension type: %v", err)
		}
		_, err = service.CreateOwnership(dataResourceConfigContext(), req)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipValueTypeMismatch)
	})

	t.Run("registered validation extension can reject unknown fields", func(t *testing.T) {
		validator := &testRegisteredOwnershipFieldValidator{err: errors.New("not registered")}
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, validator)
		resource := dataOwnershipResourceFixture(105, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(205, model.DataDimensionValueTypeBigint)
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &dimension)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipRegisteredFieldInvalid)
		if validator.calls != 1 {
			t.Fatalf("registered validator calls = %d, want 1", validator.calls)
		}
	})
}

func TestDataOwnershipConfigServiceUpdateAndReferenceProtection(t *testing.T) {
	t.Run("identity fields are immutable and state can change", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource := dataOwnershipResourceFixture(111, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(211, model.DataDimensionValueTypeBigint)
		ownership := dataOwnershipFixture(311, resource.Id, dimension.Id)
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &dimension)
		testutil.MustCreate(t, db, &ownership)

		otherResource := resource.Id + 1
		_, err := service.UpdateOwnership(
			dataResourceConfigContext(),
			request.DataOwnershipFieldUpdateReq{
				Id:         ownership.Id,
				ResourceId: &otherResource,
			},
		)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipFieldImmutable)

		disabled := false
		result, err := service.UpdateOwnership(
			dataResourceConfigContext(),
			request.DataOwnershipFieldUpdateReq{
				Id:    ownership.Id,
				State: &disabled,
			},
		)
		if err != nil {
			t.Fatalf("disable ownership through update: %v", err)
		}
		if result.State {
			t.Fatalf("ownership state = true, want false")
		}
		var stored model.DataOwnershipField
		if err = db.First(&stored, ownership.Id).Error; err != nil {
			t.Fatalf("reload ownership: %v", err)
		}
		if stored.State {
			t.Fatalf("stored ownership state = true, want false")
		}
	})

	t.Run("active policy reference blocks disable", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource, dimension, ownership := createRegisteredOwnershipFixtures(t, db)
		createOwnershipPolicyReference(t, db, resource.Id, dimension.Id, ownership.OwnershipCode, true)

		err := service.DisableOwnership(dataResourceConfigContext(), ownership.Id)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipReferenced)
		assertOwnershipState(t, db, ownership.Id, true)
	})

	t.Run("inactive reference allows disable but still protects delete", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		resource, dimension, ownership := createRegisteredOwnershipFixtures(t, db)
		createOwnershipPolicyReference(t, db, resource.Id, dimension.Id, ownership.OwnershipCode, false)

		if err := service.DisableOwnership(dataResourceConfigContext(), ownership.Id); err != nil {
			t.Fatalf("disable ownership with inactive reference: %v", err)
		}
		assertOwnershipState(t, db, ownership.Id, false)

		err := service.RemoveOwnership(dataResourceConfigContext(), ownership.Id)
		assertDataOwnershipConfigError(t, err, apperrors.ErrorCodeDataOwnershipReferenced)
	})

	t.Run("unreferenced ownership is soft deleted", func(t *testing.T) {
		service, db := newDataOwnershipConfigTestSubject(t, &testTransactionalAuditWriter{}, nil)
		_, _, ownership := createRegisteredOwnershipFixtures(t, db)

		if err := service.RemoveOwnership(dataResourceConfigContext(), ownership.Id); err != nil {
			t.Fatalf("remove ownership: %v", err)
		}
		var activeCount int64
		if err := db.Model(&model.DataOwnershipField{}).
			Where("id = ?", ownership.Id).
			Count(&activeCount).Error; err != nil {
			t.Fatalf("count active ownership: %v", err)
		}
		var allCount int64
		if err := db.Unscoped().
			Model(&model.DataOwnershipField{}).
			Where("id = ?", ownership.Id).
			Count(&allCount).Error; err != nil {
			t.Fatalf("count all ownership: %v", err)
		}
		if activeCount != 0 || allCount != 1 {
			t.Fatalf("soft delete counts = active:%d all:%d, want 0/1", activeCount, allCount)
		}
	})
}

func TestDataOwnershipConfigServiceTransactionRollback(t *testing.T) {
	t.Run("create rolls back when audit fails", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{err: errors.New("audit failed")}
		service, db := newDataOwnershipConfigTestSubject(t, auditWriter, nil)
		resource := dataOwnershipResourceFixture(121, model.DataResourceTypeBusinessService)
		dimension := dataOwnershipDimensionFixture(221, model.DataDimensionValueTypeBigint)
		testutil.MustCreate(t, db, &resource)
		testutil.MustCreate(t, db, &dimension)

		_, err := service.CreateOwnership(
			dataResourceConfigContext(),
			registeredOwnershipCreateRequest(resource.Id, dimension.Id),
		)
		if err == nil {
			t.Fatal("expected audit failure")
		}
		var count int64
		if err = db.Model(&model.DataOwnershipField{}).Count(&count).Error; err != nil {
			t.Fatalf("count ownerships: %v", err)
		}
		if count != 0 {
			t.Fatalf("ownership count = %d, want 0 after rollback", count)
		}
	})

	t.Run("state change rolls back when audit fails", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{err: errors.New("audit failed")}
		service, db := newDataOwnershipConfigTestSubject(t, auditWriter, nil)
		_, _, ownership := createRegisteredOwnershipFixtures(t, db)

		err := service.DisableOwnership(dataResourceConfigContext(), ownership.Id)
		if err == nil {
			t.Fatal("expected audit failure")
		}
		assertOwnershipState(t, db, ownership.Id, true)
	})
}

func newDataOwnershipConfigTestSubject(
	t *testing.T,
	auditWriter TransactionalAuditWriter,
	validator datapermission.OwnershipFieldBindingValidator,
) (*DataOwnershipConfigService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(
		t,
		&model.SysTable{},
		&model.SysTableField{},
		&model.DataDimensionDefinition{},
		&model.DataResource{},
		&model.DataOwnershipField{},
		&model.DataPolicy{},
		&model.DataPolicyRule{},
		&model.DataGrant{},
	)
	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return NewDataOwnershipConfigService(
		impl.NewDataResourceRepositoryImpl(primaryDB),
		impl.NewDataDimensionDefinitionRepositoryImpl(primaryDB),
		impl.NewDataOwnershipFieldRepositoryImpl(primaryDB),
		impl.NewSysTableFieldRepositoryImpl(primaryDB),
		validator,
		sf,
		auditWriter,
	), db
}

func createMetadataOwnershipFixtures(
	t *testing.T,
	db *gorm.DB,
) (model.DataResource, model.DataDimensionDefinition, model.SysTableField) {
	t.Helper()
	table := model.SysTable{
		Basic:     model.Basic{Id: 401, State: true},
		TableName: "运输订单",
		TableCode: "tms_order",
	}
	field := model.SysTableField{
		Basic:         model.Basic{Id: 501, State: true},
		TableId:       table.Id,
		FieldName:     "归属组织",
		FieldCode:     "owner_org_id",
		FieldType:     enum.BigIntFieldType,
		FieldCategory: enum.NormalField,
		Sequence:      1,
	}
	resource := dataOwnershipResourceFixture(101, model.DataResourceTypeLowCodeTable)
	resource.TableId = &table.Id
	resource.ServiceCode = nil
	resource.AdapterCode = "metadata_filter"
	dimension := dataOwnershipDimensionFixture(201, model.DataDimensionValueTypeBigint)
	testutil.MustCreate(t, db, &table)
	testutil.MustCreate(t, db, &field)
	testutil.MustCreate(t, db, &resource)
	testutil.MustCreate(t, db, &dimension)
	return resource, dimension, field
}

func createRegisteredOwnershipFixtures(
	t *testing.T,
	db *gorm.DB,
) (model.DataResource, model.DataDimensionDefinition, model.DataOwnershipField) {
	t.Helper()
	resource := dataOwnershipResourceFixture(101, model.DataResourceTypeBusinessService)
	dimension := dataOwnershipDimensionFixture(201, model.DataDimensionValueTypeBigint)
	ownership := dataOwnershipFixture(301, resource.Id, dimension.Id)
	testutil.MustCreate(t, db, &resource)
	testutil.MustCreate(t, db, &dimension)
	testutil.MustCreate(t, db, &ownership)
	return resource, dimension, ownership
}

func createOwnershipPolicyReference(
	t *testing.T,
	db *gorm.DB,
	resourceId int,
	dimensionId int,
	ownershipCode string,
	active bool,
) {
	t.Helper()
	policy := model.DataPolicy{
		Basic:      model.Basic{Id: 601, State: true},
		Code:       "policy_owner_org",
		Name:       "组织范围",
		PolicyType: model.DataPolicyTypeRuleSet,
	}
	rule := model.DataPolicyRule{
		Basic:         model.Basic{Id: 701, State: active},
		PolicyId:      policy.Id,
		Sequence:      1,
		DimensionId:   dimensionId,
		OwnershipCode: ownershipCode,
		ScopeSource:   model.DataPolicyScopeSourceEffectiveOrgUnits,
		Relation:      model.DataPolicyRelationExact,
		Operator:      model.DataPolicyOperatorIn,
	}
	grant := model.DataGrant{
		Basic:       model.Basic{Id: 801, State: true},
		SubjectType: model.DataGrantSubjectTypeRole,
		SubjectId:   1,
		ResourceId:  resourceId,
		Operation:   model.DataPermissionOperationQuery,
		PolicyId:    policy.Id,
	}
	testutil.MustCreate(t, db, &policy)
	testutil.MustCreate(t, db, &rule)
	if !active {
		mustSetState(t, db, &model.DataPolicyRule{}, rule.Id, false)
	}
	testutil.MustCreate(t, db, &grant)
}

func dataOwnershipResourceFixture(id int, resourceType string) model.DataResource {
	serviceCode := "tms_order"
	return model.DataResource{
		Basic:             model.Basic{Id: id, State: true},
		ResourceCode:      "tms.transport_order",
		Name:              "运输订单",
		ResourceType:      resourceType,
		ServiceCode:       &serviceCode,
		AdapterCode:       "registered_filter",
		PermissionEnabled: false,
	}
}

func dataOwnershipDimensionFixture(id int, valueType string) model.DataDimensionDefinition {
	return model.DataDimensionDefinition{
		Basic:        model.Basic{Id: id, State: true},
		Code:         "management_org",
		Name:         "管理组织",
		Category:     model.DataDimensionCategoryOrganization,
		ValueType:    valueType,
		ProviderCode: "organization",
	}
}

func dataOwnershipFixture(id, resourceId, dimensionId int) model.DataOwnershipField {
	code := "owner_org_id"
	return model.DataOwnershipField{
		Basic:            model.Basic{Id: id, State: true},
		ResourceId:       resourceId,
		OwnershipCode:    "owner_org",
		DimensionId:      dimensionId,
		BindingType:      model.DataOwnershipBindingTypeRegisteredField,
		AdapterFieldCode: &code,
		ValueType:        model.DataDimensionValueTypeBigint,
	}
}

func metadataOwnershipCreateRequest(
	resourceId int,
	dimensionId int,
	fieldId int,
) request.DataOwnershipFieldCreateReq {
	return request.DataOwnershipFieldCreateReq{
		ResourceId:    resourceId,
		OwnershipCode: "owner_org",
		DimensionId:   dimensionId,
		BindingType:   model.DataOwnershipBindingTypeMetadataField,
		BindingTarget: request.DataOwnershipBindingTargetReq{ReferenceId: &fieldId},
		ValueType:     model.DataDimensionValueTypeBigint,
	}
}

func registeredOwnershipCreateRequest(
	resourceId int,
	dimensionId int,
) request.DataOwnershipFieldCreateReq {
	fieldCode := "owner_org_id"
	return request.DataOwnershipFieldCreateReq{
		ResourceId:    resourceId,
		OwnershipCode: "owner_org",
		DimensionId:   dimensionId,
		BindingType:   model.DataOwnershipBindingTypeRegisteredField,
		BindingTarget: request.DataOwnershipBindingTargetReq{ReferenceCode: &fieldCode},
		ValueType:     model.DataDimensionValueTypeBigint,
	}
}

func dataOwnershipConfigTable() model.SysTable {
	return model.SysTable{
		TableCode: "sys_data_ownership_field",
		TableFields: []model.SysTableField{
			{
				FieldCode:        "ownership_code",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsQuickSearch:    true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "resource_id",
				FieldType:        enum.BigIntFieldType,
				IsListShow:       true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
		},
	}
}

func mustSetState(t *testing.T, db *gorm.DB, value any, id int, state bool) {
	t.Helper()
	if err := db.Model(value).Where("id = ?", id).Update("state", state).Error; err != nil {
		t.Fatalf("set state for %T: %v", value, err)
	}
}

func assertOwnershipState(t *testing.T, db *gorm.DB, ownershipId int, expected bool) {
	t.Helper()
	var ownership model.DataOwnershipField
	if err := db.First(&ownership, ownershipId).Error; err != nil {
		t.Fatalf("reload ownership: %v", err)
	}
	if ownership.State != expected {
		t.Fatalf("ownership state = %t, want %t", ownership.State, expected)
	}
}

func assertDataOwnershipConfigError(t *testing.T, err error, expectedCode int) {
	t.Helper()
	assertDataResourceConfigError(t, err, expectedCode)
}
