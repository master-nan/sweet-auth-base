package service

import (
	"backend/dto/response"
	"backend/internal/database"
	"backend/internal/datapermission"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestDataPermissionConfigPreflightNormalConfiguration(t *testing.T) {
	auditWriter := &testTransactionalAuditWriter{}
	service, db := newDataPermissionPreflightTestSubject(t, auditWriter)
	fixtures := createDataGrantConfigFixtures(t, db)
	grant := createDataPermissionPreflightGrant(t, db, fixtures, true)

	resourceResult, err := service.PreflightResource(
		dataResourceConfigContext(),
		fixtures.resource.Id,
	)
	assertDataPermissionPreflightValid(t, resourceResult, err)
	policyResult, err := service.PreflightPolicy(
		dataResourceConfigContext(),
		fixtures.policy.Id,
	)
	assertDataPermissionPreflightValid(t, policyResult, err)
	grantResult, err := service.PreflightGrant(dataResourceConfigContext(), grant.Id)
	assertDataPermissionPreflightValid(t, grantResult, err)

	enableResult, err := service.EnableResource(
		dataResourceConfigContext(),
		fixtures.resource.Id,
	)
	assertDataPermissionPreflightValid(t, enableResult, err)
	assertDataPermissionResourceEnabled(t, db, fixtures.resource.Id, true)

	disableResult, err := service.DisableResource(
		dataResourceConfigContext(),
		fixtures.resource.Id,
	)
	assertDataPermissionPreflightValid(t, disableResult, err)
	assertDataPermissionResourceEnabled(t, db, fixtures.resource.Id, false)
	if len(auditWriter.snapshot()) != 2 {
		t.Fatalf("audit count = %d, want 2", len(auditWriter.snapshot()))
	}
}

func TestDataPermissionConfigPreflightMissingOwnershipBlocksResourceEnable(t *testing.T) {
	service, db := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataGrantConfigFixtures(t, db)
	createDataPermissionPreflightGrant(t, db, fixtures, true)
	if err := db.Delete(&model.DataOwnershipField{}, fixtures.ownership.Id).Error; err != nil {
		t.Fatalf("delete ownership: %v", err)
	}

	result, err := service.EnableResource(dataResourceConfigContext(), fixtures.resource.Id)
	assertDataResourceConfigError(
		t,
		err,
		apperrors.ErrorCodeDataPermissionPreflightFailed,
	)
	assertDataPermissionDiagnostic(t, result, diagnosticOwnershipRequired)
	assertDataPermissionResourceEnabled(t, db, fixtures.resource.Id, false)
}

func TestDataPermissionConfigPreflightPolicyRuleCompatibility(t *testing.T) {
	t.Run("ownership code mismatch", func(t *testing.T) {
		service, db := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		if err := db.Model(&model.DataPolicyRule{}).
			Where("id = ?", fixtures.rule.Id).
			Update("ownership_code", "missing_owner").Error; err != nil {
			t.Fatalf("update ownership code: %v", err)
		}

		result, err := service.PreflightPolicy(
			dataResourceConfigContext(),
			fixtures.policy.Id,
		)
		if err != nil {
			t.Fatalf("preflight policy: %v", err)
		}
		assertDataPermissionDiagnostic(t, result, diagnosticOwnershipNotFound)
	})

	t.Run("dimension mismatch", func(t *testing.T) {
		service, db := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
		fixtures := createDataGrantConfigFixtures(t, db)
		otherDimension := dataPolicyDimensionFixture(211, "employee")
		testutil.MustCreate(t, db, &otherDimension)
		if err := db.Model(&model.DataPolicyRule{}).
			Where("id = ?", fixtures.rule.Id).
			Update("dimension_id", otherDimension.Id).Error; err != nil {
			t.Fatalf("update dimension: %v", err)
		}

		result, err := service.PreflightPolicy(
			dataResourceConfigContext(),
			fixtures.policy.Id,
		)
		if err != nil {
			t.Fatalf("preflight policy: %v", err)
		}
		assertDataPermissionDiagnostic(t, result, diagnosticOwnershipDimension)
	})
}

func TestDataPermissionConfigPreflightPolicyRuleDeclarations(t *testing.T) {
	service, db := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataGrantConfigFixtures(t, db)
	if err := db.Exec("PRAGMA ignore_check_constraints = ON").Error; err != nil {
		t.Fatalf("disable sqlite checks: %v", err)
	}
	if err := db.Model(&model.DataPolicyRule{}).
		Where("id = ?", fixtures.rule.Id).
		Update("scope_source", "current_role").Error; err != nil {
		t.Fatalf("update scope source: %v", err)
	}
	if err := db.Exec("PRAGMA ignore_check_constraints = OFF").Error; err != nil {
		t.Fatalf("enable sqlite checks: %v", err)
	}

	result, err := service.PreflightPolicy(dataResourceConfigContext(), fixtures.policy.Id)
	if err != nil {
		t.Fatalf("preflight policy: %v", err)
	}
	assertDataPermissionDiagnostic(t, result, diagnosticScopeSourceInvalid)
	assertDataPermissionDiagnostic(t, result, diagnosticProviderCapabilityInvalid)
}

func TestDataPermissionConfigPreflightGrantRejectsInactivePolicy(t *testing.T) {
	service, db := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataGrantConfigFixtures(t, db)
	grant := createDataPermissionPreflightGrant(t, db, fixtures, false)
	if err := db.Model(&model.DataPolicy{}).
		Where("id = ?", fixtures.policy.Id).
		Update("state", false).Error; err != nil {
		t.Fatalf("disable policy: %v", err)
	}

	result, err := service.EnableGrant(dataResourceConfigContext(), grant.Id)
	assertDataResourceConfigError(
		t,
		err,
		apperrors.ErrorCodeDataPermissionPreflightFailed,
	)
	assertDataPermissionDiagnostic(t, result, diagnosticPolicyInactive)
	assertDataPermissionGrantState(t, db, grant.Id, false)
}

func TestDataPermissionConfigPreflightPolicyAndGrantState(t *testing.T) {
	service, db := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataGrantConfigFixtures(t, db)
	grant := createDataPermissionPreflightGrant(t, db, fixtures, true)

	result, err := service.DisableGrant(dataResourceConfigContext(), grant.Id)
	assertDataPermissionPreflightValid(t, result, err)
	assertDataPermissionGrantState(t, db, grant.Id, false)
	result, err = service.EnableGrant(dataResourceConfigContext(), grant.Id)
	assertDataPermissionPreflightValid(t, result, err)
	assertDataPermissionGrantState(t, db, grant.Id, true)

	result, err = service.DisablePolicy(dataResourceConfigContext(), fixtures.policy.Id)
	assertDataPermissionPreflightValid(t, result, err)
	assertDataPermissionPolicyState(t, db, fixtures.policy.Id, false)
	result, err = service.EnablePolicy(dataResourceConfigContext(), fixtures.policy.Id)
	assertDataPermissionPreflightValid(t, result, err)
	assertDataPermissionPolicyState(t, db, fixtures.policy.Id, true)
}

func TestDataPermissionConfigPreflightBatchRollback(t *testing.T) {
	service, db := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
	fixtures := createDataGrantConfigFixtures(t, db)
	second := dataResourceFixture(111, "tms.transport_order_archive")
	testutil.MustCreate(t, db, &second)

	result, err := service.EnableResources(
		dataResourceConfigContext(),
		[]int{fixtures.resource.Id, second.Id},
	)
	assertDataResourceConfigError(
		t,
		err,
		apperrors.ErrorCodeDataPermissionPreflightFailed,
	)
	assertDataPermissionDiagnostic(t, result, diagnosticOwnershipRequired)
	assertDataPermissionResourceEnabled(t, db, fixtures.resource.Id, false)
	assertDataPermissionResourceEnabled(t, db, second.Id, false)
}

func TestDataPermissionConfigPreflightAuditFailureRollsBack(t *testing.T) {
	service, db := newDataPermissionPreflightTestSubject(
		t,
		&testTransactionalAuditWriter{err: errors.New("audit failed")},
	)
	fixtures := createDataGrantConfigFixtures(t, db)

	result, err := service.EnableResource(dataResourceConfigContext(), fixtures.resource.Id)
	if err == nil {
		t.Fatal("expected audit failure")
	}
	if !result.Valid {
		t.Fatalf("configuration should be valid before audit failure: %+v", result)
	}
	assertDataPermissionResourceEnabled(t, db, fixtures.resource.Id, false)
}

func TestDataPermissionValidationResultStructure(t *testing.T) {
	service, _ := newDataPermissionPreflightTestSubject(t, &testTransactionalAuditWriter{})
	result, err := service.PreflightResource(dataResourceConfigContext(), 999)
	if err != nil {
		t.Fatalf("preflight missing resource: %v", err)
	}
	if result.Valid || len(result.Errors) != 1 {
		t.Fatalf("unexpected validation result: %+v", result)
	}
	validationErr := result.Errors[0]
	if validationErr.Code != diagnosticResourceNotFound ||
		validationErr.Message == "" ||
		validationErr.ObjectType != dataPermissionObjectResource ||
		validationErr.ObjectId != 999 {
		t.Fatalf("unexpected validation error: %+v", validationErr)
	}
}

func newDataPermissionPreflightTestSubject(
	t *testing.T,
	auditWriter TransactionalAuditWriter,
) (*DataPermissionConfigPreflightService, *gorm.DB) {
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
	return NewDataPermissionConfigPreflightService(
		impl.NewDataGrantRepositoryImpl(primaryDB),
		impl.NewDataResourceRepositoryImpl(primaryDB),
		impl.NewDataResourceOperationRepositoryImpl(primaryDB),
		impl.NewDataOwnershipFieldRepositoryImpl(primaryDB),
		impl.NewDataDimensionDefinitionRepositoryImpl(primaryDB),
		impl.NewDataPolicyRepositoryImpl(primaryDB),
		impl.NewDataPolicyRuleRepositoryImpl(primaryDB),
		registry,
		auditWriter,
	), db
}

func createDataPermissionPreflightGrant(
	t *testing.T,
	db *gorm.DB,
	fixtures dataGrantConfigFixtures,
	state bool,
) model.DataGrant {
	t.Helper()
	grant := model.DataGrant{
		Basic:       model.Basic{Id: 501, State: state},
		SubjectType: model.DataGrantSubjectTypeRole,
		SubjectId:   fixtures.role.Id,
		ResourceId:  fixtures.resource.Id,
		Operation:   fixtures.operation.Operation,
		PolicyId:    fixtures.policy.Id,
	}
	testutil.MustCreate(t, db, &grant)
	if !state {
		if err := db.Model(&model.DataGrant{}).
			Where("id = ?", grant.Id).
			Update("state", false).Error; err != nil {
			t.Fatalf("disable grant fixture: %v", err)
		}
	}
	return grant
}

func assertDataPermissionPreflightValid(
	t *testing.T,
	result response.DataPermissionValidationResultRes,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("preflight failed: %v", err)
	}
	if !result.Valid || len(result.Errors) != 0 {
		t.Fatalf("unexpected invalid preflight result: %+v", result)
	}
}

func assertDataPermissionDiagnostic(
	t *testing.T,
	result response.DataPermissionValidationResultRes,
	code string,
) {
	t.Helper()
	for _, validationErr := range result.Errors {
		if validationErr.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %+v", code, result.Errors)
}

func assertDataPermissionResourceEnabled(t *testing.T, db *gorm.DB, id int, expected bool) {
	t.Helper()
	var resource model.DataResource
	if err := db.First(&resource, id).Error; err != nil {
		t.Fatalf("reload resource: %v", err)
	}
	if resource.PermissionEnabled != expected {
		t.Fatalf("resource permission_enabled = %t, want %t", resource.PermissionEnabled, expected)
	}
}

func assertDataPermissionPolicyState(t *testing.T, db *gorm.DB, id int, expected bool) {
	t.Helper()
	var policy model.DataPolicy
	if err := db.First(&policy, id).Error; err != nil {
		t.Fatalf("reload policy: %v", err)
	}
	if policy.State != expected {
		t.Fatalf("policy state = %t, want %t", policy.State, expected)
	}
}

func assertDataPermissionGrantState(t *testing.T, db *gorm.DB, id int, expected bool) {
	t.Helper()
	var grant model.DataGrant
	if err := db.First(&grant, id).Error; err != nil {
		t.Fatalf("reload grant: %v", err)
	}
	if grant.State != expected {
		t.Fatalf("grant state = %t, want %t", grant.State, expected)
	}
}
