package service

import (
	"backend/dto/request"
	"backend/enum"
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
	"time"

	"gorm.io/gorm"
)

func TestDataGrantConfigServiceCreateRoleAndUserGrant(t *testing.T) {
	auditWriter := &testTransactionalAuditWriter{}
	service, db := newDataGrantConfigTestSubject(t, auditWriter)
	fixtures := createDataGrantConfigFixtures(t, db)

	roleResult, err := service.CreateGrant(
		dataResourceConfigContext(),
		dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures),
	)
	if err != nil {
		t.Fatalf("create role grant: %v", err)
	}
	if roleResult.SubjectType != model.DataGrantSubjectTypeRole ||
		roleResult.SubjectId != fixtures.role.Id ||
		roleResult.Operation != model.DataPermissionOperationQuery ||
		roleResult.Resource == nil ||
		roleResult.Resource.Code != fixtures.resource.ResourceCode ||
		roleResult.Policy == nil ||
		roleResult.Policy.Code != fixtures.policy.Code {
		t.Fatalf("unexpected role grant response: %+v", roleResult)
	}

	userResult, err := service.CreateGrant(
		dataResourceConfigContext(),
		dataGrantCreateRequest(model.DataGrantSubjectTypeUser, fixtures.user.Id, fixtures),
	)
	if err != nil {
		t.Fatalf("create user grant: %v", err)
	}
	if userResult.SubjectType != model.DataGrantSubjectTypeUser ||
		userResult.SubjectId != fixtures.user.Id {
		t.Fatalf("unexpected user grant response: %+v", userResult)
	}

	detail, err := service.GetGrant(dataResourceConfigContext(), userResult.Id)
	if err != nil || detail.Resource == nil || detail.Policy == nil {
		t.Fatalf("get grant = %+v, err=%v", detail, err)
	}
	page, err := service.PageGrants(
		dataResourceConfigContext(),
		request.DataGrantQueryReq{
			DataPermissionConfigQueryReq: request.DataPermissionConfigQueryReq{
				Page: 1,
				Num:  10,
			},
			ResourceId: &fixtures.resource.Id,
		},
	)
	if err != nil || page.Total != 2 || len(page.Data) != 2 {
		t.Fatalf("page grants = %+v, err=%v", page, err)
	}

	payload, err := json.Marshal(roleResult)
	if err != nil {
		t.Fatalf("marshal grant response: %v", err)
	}
	for _, forbidden := range []string{
		`"description":`,
		`"password":`,
		`"roles":`,
		`"source_id":`,
		`"organization_id":`,
		`"provider_result":`,
	} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("grant response leaked %q: %s", forbidden, payload)
		}
	}
	if len(auditWriter.snapshot()) != 2 {
		t.Fatalf("audit count = %d, want 2", len(auditWriter.snapshot()))
	}
}

func TestDataGrantConfigServiceSubjectValidation(t *testing.T) {
	t.Run("rejects unsupported subject type", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		req := dataGrantCreateRequest("position", fixtures.role.Id, fixtures)
		_, err := service.CreateGrant(dataResourceConfigContext(), req)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantSubjectTypeInvalid)
	})

	t.Run("rejects missing role", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		_, err := service.CreateGrant(
			dataResourceConfigContext(),
			dataGrantCreateRequest(model.DataGrantSubjectTypeRole, 999, fixtures),
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantSubjectNotFound)
	})

	t.Run("rejects missing user", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		_, err := service.CreateGrant(
			dataResourceConfigContext(),
			dataGrantCreateRequest(model.DataGrantSubjectTypeUser, 999, fixtures),
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantSubjectNotFound)
	})
}

func TestDataGrantConfigServiceResourceAndPolicyValidation(t *testing.T) {
	tests := []struct {
		name      string
		edit      func(*request.DataGrantCreateReq, *gorm.DB, dataGrantConfigFixtures)
		errorCode int
	}{
		{
			name: "resource not found",
			edit: func(req *request.DataGrantCreateReq, _ *gorm.DB, _ dataGrantConfigFixtures) {
				req.ResourceId = 999
			},
			errorCode: apperrors.ErrorCodeDataResourceNotFound,
		},
		{
			name: "resource disabled",
			edit: func(_ *request.DataGrantCreateReq, db *gorm.DB, fixtures dataGrantConfigFixtures) {
				mustSetState(t, db, &model.DataResource{}, fixtures.resource.Id, false)
			},
			errorCode: apperrors.ErrorCodeDataResourceStateInvalid,
		},
		{
			name: "resource operation missing",
			edit: func(req *request.DataGrantCreateReq, _ *gorm.DB, _ dataGrantConfigFixtures) {
				req.Operation = model.DataPermissionOperationDetail
			},
			errorCode: apperrors.ErrorCodeDataResourceOperationNotFound,
		},
		{
			name: "policy not found",
			edit: func(req *request.DataGrantCreateReq, _ *gorm.DB, _ dataGrantConfigFixtures) {
				req.PolicyId = 999
			},
			errorCode: apperrors.ErrorCodeDataPolicyNotFound,
		},
		{
			name: "policy disabled",
			edit: func(_ *request.DataGrantCreateReq, db *gorm.DB, fixtures dataGrantConfigFixtures) {
				mustSetState(t, db, &model.DataPolicy{}, fixtures.policy.Id, false)
			},
			errorCode: apperrors.ErrorCodeDataGrantPolicyInvalid,
		},
		{
			name: "policy rule disabled",
			edit: func(_ *request.DataGrantCreateReq, db *gorm.DB, fixtures dataGrantConfigFixtures) {
				mustSetState(t, db, &model.DataPolicyRule{}, fixtures.rule.Id, false)
			},
			errorCode: apperrors.ErrorCodeDataGrantPolicyRuleInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
			fixtures := createDataGrantConfigFixtures(t, db)
			req := dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures)
			test.edit(&req, db, fixtures)
			_, err := service.CreateGrant(dataResourceConfigContext(), req)
			assertDataResourceConfigError(t, err, test.errorCode)
			assertDataGrantCount(t, db, 0)
		})
	}
}

func TestDataGrantConfigServiceOwnershipCompatibility(t *testing.T) {
	t.Run("requires exact ownership code", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		if err := db.Delete(&fixtures.ownership).Error; err != nil {
			t.Fatalf("delete ownership fixture: %v", err)
		}
		_, err := service.CreateGrant(
			dataResourceConfigContext(),
			dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures),
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantOwnershipMismatch)
	})

	t.Run("requires matching dimension", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		otherDimension := dataPolicyDimensionFixture(220, "legal_entity")
		testutil.MustCreate(t, db, &otherDimension)
		if err := db.Model(&model.DataOwnershipField{}).
			Where("id = ?", fixtures.ownership.Id).
			Update("dimension_id", otherDimension.Id).Error; err != nil {
			t.Fatalf("change ownership dimension: %v", err)
		}
		_, err := service.CreateGrant(
			dataResourceConfigContext(),
			dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures),
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantOwnershipMismatch)
	})

	t.Run("registered ownership must support operation", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		detailOperation := model.DataResourceOperation{
			Basic:             model.Basic{Id: 103, State: true},
			ResourceId:        fixtures.resource.Id,
			Operation:         model.DataPermissionOperationDetail,
			PermissionEnabled: false,
		}
		testutil.MustCreate(t, db, &detailOperation)
		req := dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures)
		req.Operation = model.DataPermissionOperationDetail
		_, err := service.CreateGrant(dataResourceConfigContext(), req)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantOwnershipMismatch)
	})
}

func TestDataGrantConfigServiceDuplicateStateAndSoftDelete(t *testing.T) {
	auditWriter := &testTransactionalAuditWriter{}
	service, db := newDataGrantConfigTestSubject(t, auditWriter)
	fixtures := createDataGrantConfigFixtures(t, db)
	req := dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures)

	created, err := service.CreateGrant(dataResourceConfigContext(), req)
	if err != nil {
		t.Fatalf("create grant: %v", err)
	}
	_, err = service.CreateGrant(dataResourceConfigContext(), req)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantDuplicate)

	if err = service.DisableGrant(dataResourceConfigContext(), created.Id); err != nil {
		t.Fatalf("disable grant: %v", err)
	}
	assertDataGrantState(t, db, created.Id, false)
	_, err = service.CreateGrant(dataResourceConfigContext(), req)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantExists)

	if err = service.RestoreGrant(dataResourceConfigContext(), created.Id); err != nil {
		t.Fatalf("restore grant: %v", err)
	}
	assertDataGrantState(t, db, created.Id, true)

	if err = service.RemoveGrant(dataResourceConfigContext(), created.Id); err != nil {
		t.Fatalf("soft delete grant: %v", err)
	}
	var activeCount int64
	if err = db.Model(&model.DataGrant{}).Where("id = ?", created.Id).Count(&activeCount).Error; err != nil {
		t.Fatalf("count active grant: %v", err)
	}
	var unscopedCount int64
	if err = db.Unscoped().Model(&model.DataGrant{}).
		Where("id = ?", created.Id).
		Count(&unscopedCount).Error; err != nil {
		t.Fatalf("count soft-deleted grant: %v", err)
	}
	if activeCount != 0 || unscopedCount != 1 {
		t.Fatalf("grant counts active=%d unscoped=%d, want 0/1", activeCount, unscopedCount)
	}
	if len(auditWriter.snapshot()) != 4 {
		t.Fatalf("audit count = %d, want 4", len(auditWriter.snapshot()))
	}
}

func TestDataGrantConfigServiceTransactionRollback(t *testing.T) {
	t.Run("batch failure rolls back earlier grants", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		valid := dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures)
		invalid := dataGrantCreateRequest(model.DataGrantSubjectTypeUser, 999, fixtures)
		_, err := service.CreateGrants(
			dataResourceConfigContext(),
			request.DataGrantBatchCreateReq{Items: []request.DataGrantCreateReq{valid, invalid}},
		)
		assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantSubjectNotFound)
		assertDataGrantCount(t, db, 0)
	})

	t.Run("audit failure rolls back grant", func(t *testing.T) {
		service, db := newDataGrantConfigTestSubject(
			t,
			&testTransactionalAuditWriter{err: errors.New("audit failed")},
		)
		fixtures := createDataGrantConfigFixtures(t, db)
		_, err := service.CreateGrant(
			dataResourceConfigContext(),
			dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures),
		)
		if err == nil {
			t.Fatal("expected audit failure")
		}
		assertDataGrantCount(t, db, 0)
	})

	t.Run("audit failure rolls back state change", func(t *testing.T) {
		auditWriter := &testTransactionalAuditWriter{}
		service, db := newDataGrantConfigTestSubject(t, auditWriter)
		fixtures := createDataGrantConfigFixtures(t, db)
		created, err := service.CreateGrant(
			dataResourceConfigContext(),
			dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures),
		)
		if err != nil {
			t.Fatalf("create grant: %v", err)
		}
		auditWriter.err = errors.New("audit failed")
		err = service.DisableGrant(dataResourceConfigContext(), created.Id)
		if err == nil {
			t.Fatal("expected audit failure")
		}
		assertDataGrantState(t, db, created.Id, true)
	})
}

func TestDataGrantConfigServiceValidity(t *testing.T) {
	service, db := newDataGrantConfigTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataGrantConfigFixtures(t, db)
	validFrom := time.Date(2026, time.August, 2, 15, 30, 0, 0, model.AppLocation())
	validTo := time.Date(2026, time.August, 1, 8, 0, 0, 0, model.AppLocation())
	req := dataGrantCreateRequest(model.DataGrantSubjectTypeRole, fixtures.role.Id, fixtures)
	req.ValidFrom = &validFrom
	req.ValidTo = &validTo
	_, err := service.CreateGrant(dataResourceConfigContext(), req)
	assertDataResourceConfigError(t, err, apperrors.ErrorCodeDataGrantValidityInvalid)
	assertDataGrantCount(t, db, 0)
}

type dataGrantConfigFixtures struct {
	resource  model.DataResource
	operation model.DataResourceOperation
	dimension model.DataDimensionDefinition
	ownership model.DataOwnershipField
	policy    model.DataPolicy
	rule      model.DataPolicyRule
	role      model.SysRole
	user      model.SysUser
}

func newDataGrantConfigTestSubject(
	t *testing.T,
	auditWriter TransactionalAuditWriter,
) (*DataGrantConfigService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(
		t,
		&model.DataDimensionDefinition{},
		&model.DataResource{},
		&model.DataResourceOperation{},
		&model.DataOwnershipField{},
		&model.DataPolicy{},
		&model.DataPolicyRule{},
		&model.DataGrant{},
		&model.SysRole{},
		&model.SysUser{},
	)
	primaryDB := &database.PrimaryDB{DB: db}
	registry, err := datapermission.NewOwnershipFieldRegistry(
		datapermission.OwnershipFieldRegistration{
			ResourceCode:        "tms.transport_order",
			OwnershipCode:       "owner_org",
			AdapterFieldCode:    "owner_org_id",
			ValueType:           model.DataDimensionValueTypeBigint,
			SupportedDimensions: []string{"management_org"},
			SupportedOperations: []string{model.DataPermissionOperationQuery},
		},
	)
	if err != nil {
		t.Fatalf("create ownership registry: %v", err)
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return NewDataGrantConfigService(
		impl.NewDataGrantRepositoryImpl(primaryDB),
		impl.NewDataResourceRepositoryImpl(primaryDB),
		impl.NewDataResourceOperationRepositoryImpl(primaryDB),
		impl.NewDataOwnershipFieldRepositoryImpl(primaryDB),
		impl.NewDataDimensionDefinitionRepositoryImpl(primaryDB),
		impl.NewDataPolicyRepositoryImpl(primaryDB),
		impl.NewDataPolicyRuleRepositoryImpl(primaryDB),
		registry,
		sf,
		auditWriter,
	), db
}

func createDataGrantConfigFixtures(
	t *testing.T,
	db *gorm.DB,
) dataGrantConfigFixtures {
	t.Helper()
	resource := model.DataResource{
		Basic:             model.Basic{Id: 101, State: true},
		ResourceCode:      "tms.transport_order",
		Name:              "运输订单",
		ResourceType:      model.DataResourceTypeBusinessService,
		ServiceCode:       dataPolicyStringPointer("tms.transport_order"),
		AdapterCode:       "registered_filter",
		PermissionEnabled: false,
	}
	operation := model.DataResourceOperation{
		Basic:             model.Basic{Id: 102, State: true},
		ResourceId:        resource.Id,
		Operation:         model.DataPermissionOperationQuery,
		PermissionEnabled: false,
	}
	dimension := dataPolicyDimensionFixture(201, "management_org")
	adapterFieldCode := "owner_org_id"
	ownership := model.DataOwnershipField{
		Basic:            model.Basic{Id: 202, State: true},
		ResourceId:       resource.Id,
		OwnershipCode:    "owner_org",
		DimensionId:      dimension.Id,
		BindingType:      model.DataOwnershipBindingTypeRegisteredField,
		AdapterFieldCode: &adapterFieldCode,
		ValueType:        model.DataDimensionValueTypeBigint,
	}
	policy := dataPolicyFixture(301, "management_org_scope")
	rule := dataPolicyRuleFixture(302, policy.Id, dimension.Id)
	role := model.SysRole{
		Basic: model.Basic{Id: 401, State: true},
		Name:  "运输管理员",
	}
	user := model.SysUser{
		Basic:    model.Basic{Id: 402, State: true},
		UserName: "grant_user",
	}
	testutil.MustCreate(t, db, &resource)
	testutil.MustCreate(t, db, &operation)
	testutil.MustCreate(t, db, &dimension)
	testutil.MustCreate(t, db, &ownership)
	testutil.MustCreate(t, db, &policy)
	testutil.MustCreate(t, db, &rule)
	testutil.MustCreate(t, db, &role)
	testutil.MustCreate(t, db, &user)
	return dataGrantConfigFixtures{
		resource:  resource,
		operation: operation,
		dimension: dimension,
		ownership: ownership,
		policy:    policy,
		rule:      rule,
		role:      role,
		user:      user,
	}
}

func dataGrantCreateRequest(
	subjectType string,
	subjectId int,
	fixtures dataGrantConfigFixtures,
) request.DataGrantCreateReq {
	return request.DataGrantCreateReq{
		SubjectType: subjectType,
		SubjectId:   subjectId,
		ResourceId:  fixtures.resource.Id,
		Operation:   fixtures.operation.Operation,
		PolicyId:    fixtures.policy.Id,
		Description: "配置端内部说明",
	}
}

func dataGrantConfigTable() model.SysTable {
	return model.SysTable{
		TableCode: "sys_data_grant",
		TableFields: []model.SysTableField{
			{
				FieldCode:        "subject_type",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsQuickSearch:    true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "subject_id",
				FieldType:        enum.BigIntFieldType,
				IsListShow:       true,
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
			{
				FieldCode:        "operation",
				FieldType:        enum.VarcharFieldType,
				IsListShow:       true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
			{
				FieldCode:        "policy_id",
				FieldType:        enum.BigIntFieldType,
				IsListShow:       true,
				IsAdvancedSearch: true,
				IsSort:           true,
			},
		},
	}
}

func assertDataGrantCount(t *testing.T, db *gorm.DB, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.DataGrant{}).Count(&count).Error; err != nil {
		t.Fatalf("count grants: %v", err)
	}
	if count != expected {
		t.Fatalf("grant count = %d, want %d", count, expected)
	}
}

func assertDataGrantState(t *testing.T, db *gorm.DB, id int, expected bool) {
	t.Helper()
	var grant model.DataGrant
	if err := db.First(&grant, id).Error; err != nil {
		t.Fatalf("reload grant: %v", err)
	}
	if grant.State != expected {
		t.Fatalf("grant state = %t, want %t", grant.State, expected)
	}
}
