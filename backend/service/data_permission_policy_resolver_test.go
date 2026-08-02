package service

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"backend/dto/response"
	"backend/internal/database"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const policyResolverResourceCode = "service:tms.transport_order"

type policyResolverFixtures struct {
	resource  model.DataResource
	operation model.DataResourceOperation
	dimension model.DataDimensionDefinition
	ownership model.DataOwnershipField
	policy    model.DataPolicy
	rule      model.DataPolicyRule
	grant     model.DataGrant
}

type policyResolverTestProvider struct {
	calls         int
	dimensionCode string
	subject       datapermission.SubjectContext
	values        datapermission.DimensionValues
	err           error
}

func (provider *policyResolverTestProvider) ResolveDimensionValues(
	_ *gin.Context,
	subject datapermission.SubjectContext,
	dimensionCode string,
) (datapermission.DimensionValues, error) {
	provider.calls++
	provider.dimensionCode = dimensionCode
	provider.subject = subject
	return provider.values, provider.err
}

func TestDataPermissionPolicyResolverResolvesSinglePolicy(t *testing.T) {
	resolver, _, provider, _ := newPolicyResolverTestSubject(t)
	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve policy: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered {
		t.Fatalf("decision = %s, want filtered", result.Decision())
	}
	groups := result.ConditionGroups()
	if len(groups) != 1 || len(groups[0].Conditions()) != 1 {
		t.Fatalf("unexpected condition groups: %+v", groups)
	}
	condition := groups[0].Conditions()[0]
	if condition.OwnershipCode() != "owner_org" ||
		condition.DimensionId() != 201 ||
		condition.Operator() != datapermission.DataScopeOperatorIn ||
		condition.ValueType() != datapermission.DataScopeValueTypeBigint ||
		!reflect.DeepEqual(condition.BigintValues(), []int64{11, 12}) {
		t.Fatalf("unexpected condition: %+v values=%v", condition, condition.BigintValues())
	}
	if provider.calls != 1 || provider.dimensionCode != datapermission.DimensionCodeManagementOrg ||
		provider.subject.EmployeeId() != policyResolverSubject(t).EmployeeId() {
		t.Fatalf("unexpected Provider call: calls=%d dimension=%s employee=%d", provider.calls, provider.dimensionCode, provider.subject.EmployeeId())
	}
}

func TestDataPermissionPolicyResolverResolvesUserGrant(t *testing.T) {
	resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
	mustUpdatePolicyResolverField(
		t,
		db,
		&model.DataGrant{},
		fixtures.grant.Id,
		"subject_type",
		model.DataGrantSubjectTypeUser,
	)
	mustUpdatePolicyResolverField(t, db, &model.DataGrant{}, fixtures.grant.Id, "subject_id", 1001)

	subject := policyResolverSubject(t)
	var err error
	input, err := datapermission.NewResolverInput(
		subject,
		policyResolverResourceCode,
		model.DataPermissionOperationQuery,
	)
	if err != nil {
		t.Fatalf("create ResolverInput: %v", err)
	}
	result, err := resolver.Resolve(nil, input)
	if err != nil {
		t.Fatalf("resolve user Grant: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered {
		t.Fatalf("decision = %s, want filtered", result.Decision())
	}
	if provider.calls != 1 {
		t.Fatalf("Provider calls = %d, want 1", provider.calls)
	}
}

func TestDataPermissionPolicyResolverMergesUserAndRoleGrants(t *testing.T) {
	resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
	userGrant := fixtures.grant
	userGrant.Id = 402
	userGrant.SubjectType = model.DataGrantSubjectTypeUser
	userGrant.SubjectId = policyResolverSubject(t).UserId()
	testutil.MustCreate(t, db, &userGrant)

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve user and role Grants: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered ||
		len(result.ConditionGroups()) != 1 {
		t.Fatalf("unexpected merged result: decision=%s groups=%d", result.Decision(), len(result.ConditionGroups()))
	}
	if provider.calls != 2 {
		t.Fatalf("Provider calls = %d, want 2 effective Grants", provider.calls)
	}
}

func TestDataPermissionPolicyResolverReturnsNoneWithoutEffectiveGrant(t *testing.T) {
	tests := []struct {
		name   string
		change func(*testing.T, *gorm.DB, policyResolverFixtures)
	}{
		{
			name: "disabled grant",
			change: func(t *testing.T, db *gorm.DB, fixtures policyResolverFixtures) {
				mustUpdatePolicyResolverField(t, db, &model.DataGrant{}, fixtures.grant.Id, "state", false)
			},
		},
		{
			name: "future grant",
			change: func(t *testing.T, db *gorm.DB, fixtures policyResolverFixtures) {
				future := policyResolverDate(t, "2026-08-03")
				mustUpdatePolicyResolverField(t, db, &model.DataGrant{}, fixtures.grant.Id, "valid_from", future)
			},
		},
		{
			name: "historical grant",
			change: func(t *testing.T, db *gorm.DB, fixtures policyResolverFixtures) {
				historical := policyResolverDate(t, "2026-08-01")
				mustUpdatePolicyResolverField(t, db, &model.DataGrant{}, fixtures.grant.Id, "valid_to", historical)
			},
		},
		{
			name: "grant for another resource",
			change: func(t *testing.T, db *gorm.DB, fixtures policyResolverFixtures) {
				otherResource := fixtures.resource
				otherResource.Id = 103
				otherResource.ResourceCode = "service:tms.other_order"
				testutil.MustCreate(t, db, &otherResource)
				mustUpdatePolicyResolverField(
					t,
					db,
					&model.DataGrant{},
					fixtures.grant.Id,
					"resource_id",
					otherResource.Id,
				)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
			tt.change(t, db, fixtures)
			result, err := resolver.Resolve(nil, policyResolverInput(t))
			if err != nil {
				t.Fatalf("resolve policy: %v", err)
			}
			if result.Decision() != datapermission.DataScopeDecisionNone {
				t.Fatalf("decision = %s, want none", result.Decision())
			}
			if provider.calls != 0 {
				t.Fatalf("Provider must not run without a Grant, calls=%d", provider.calls)
			}
		})
	}
}

func TestDataPermissionPolicyResolverRejectsInvalidPolicyAndOwnership(t *testing.T) {
	t.Run("policy missing", func(t *testing.T) {
		resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
		if err := db.Delete(&fixtures.policy).Error; err != nil {
			t.Fatalf("delete policy: %v", err)
		}
		result, err := resolver.Resolve(nil, policyResolverInput(t))
		assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverPolicyInvalid)
		assertPolicyResolverNoAccess(t, result)
		if provider.calls != 0 {
			t.Fatalf("Provider called before Policy validation: %d", provider.calls)
		}
	})

	t.Run("ownership missing", func(t *testing.T) {
		resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
		if err := db.Delete(&fixtures.ownership).Error; err != nil {
			t.Fatalf("delete ownership: %v", err)
		}
		result, err := resolver.Resolve(nil, policyResolverInput(t))
		assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverOwnershipMissing)
		assertPolicyResolverNoAccess(t, result)
		if provider.calls != 0 {
			t.Fatalf("Provider called without Ownership: %d", provider.calls)
		}
	})

	t.Run("dimension mismatch", func(t *testing.T) {
		resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
		otherDimension := policyResolverDimension(203, datapermission.DimensionCodeLegalEntity)
		testutil.MustCreate(t, db, &otherDimension)
		mustUpdatePolicyResolverField(t, db, &model.DataOwnershipField{}, fixtures.ownership.Id, "dimension_id", otherDimension.Id)
		result, err := resolver.Resolve(nil, policyResolverInput(t))
		assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverConfigConflict)
		assertPolicyResolverNoAccess(t, result)
		if provider.calls != 0 {
			t.Fatalf("Provider called after Dimension mismatch: %d", provider.calls)
		}
	})
}

func TestDataPermissionPolicyResolverFailsClosedOnProviderError(t *testing.T) {
	resolver, _, provider, _ := newPolicyResolverTestSubject(t)
	provider.err = errors.New("organization provider unavailable")

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverDimensionFailed)
	assertPolicyResolverNoAccess(t, result)
	if provider.calls != 1 {
		t.Fatalf("Provider calls = %d, want 1", provider.calls)
	}
}

func TestDataPermissionPolicyResolverNormalizesEmptyProviderValuesToNone(t *testing.T) {
	resolver, _, provider, _ := newPolicyResolverTestSubject(t)
	empty, err := datapermission.NewDimensionValues(
		datapermission.DimensionCodeManagementOrg,
		datapermission.DataScopeValueTypeBigint,
		nil,
	)
	if err != nil {
		t.Fatalf("create empty DimensionValues: %v", err)
	}
	provider.values = empty

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve empty Provider values: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionNone {
		t.Fatalf("decision = %s, want none", result.Decision())
	}
}

func TestDataPermissionPolicyResolverRejectsMultipleRules(t *testing.T) {
	t.Run("multiple rules", func(t *testing.T) {
		resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
		secondRule := fixtures.rule
		secondRule.Id = 303
		secondRule.Sequence = 2
		testutil.MustCreate(t, db, &secondRule)

		result, err := resolver.Resolve(nil, policyResolverInput(t))
		assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverConfigConflict)
		assertPolicyResolverNoAccess(t, result)
		if provider.calls != 0 {
			t.Fatalf("Provider called for multiple Rules: %d", provider.calls)
		}
	})
}

func TestDataPermissionPolicyResolverHandlesResourceActivation(t *testing.T) {
	t.Run("permission disabled is not applicable", func(t *testing.T) {
		resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
		mustUpdatePolicyResolverField(t, db, &model.DataResource{}, fixtures.resource.Id, "permission_enabled", false)
		result, err := resolver.Resolve(nil, policyResolverInput(t))
		if err != nil {
			t.Fatalf("resolve disabled permission: %v", err)
		}
		if result.Decision() != datapermission.DataScopeDecisionNotApplicable {
			t.Fatalf("decision = %s, want not_applicable", result.Decision())
		}
		if provider.calls != 0 {
			t.Fatalf("Provider called for disabled resource: %d", provider.calls)
		}
	})

	t.Run("inactive resource fails closed", func(t *testing.T) {
		resolver, db, _, fixtures := newPolicyResolverTestSubject(t)
		mustUpdatePolicyResolverField(t, db, &model.DataResource{}, fixtures.resource.Id, "state", false)
		result, err := resolver.Resolve(nil, policyResolverInput(t))
		assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverResourceMissing)
		assertPolicyResolverNoAccess(t, result)
	})
}

func newPolicyResolverTestSubject(
	t *testing.T,
) (*DataPermissionPolicyResolver, *gorm.DB, *policyResolverTestProvider, policyResolverFixtures) {
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
	)
	fixtures := createPolicyResolverFixtures(t, db)
	values, err := datapermission.NewDimensionValues(
		datapermission.DimensionCodeManagementOrg,
		datapermission.DataScopeValueTypeBigint,
		[]any{int64(12), int64(11), int64(12)},
	)
	if err != nil {
		t.Fatalf("create Provider values: %v", err)
	}
	provider := &policyResolverTestProvider{values: values}
	primaryDB := &database.PrimaryDB{DB: db}
	resolver := NewDataPermissionPolicyResolver(
		impl.NewDataResourceRepositoryImpl(primaryDB),
		impl.NewDataResourceOperationRepositoryImpl(primaryDB),
		impl.NewDataGrantRepositoryImpl(primaryDB),
		impl.NewDataPolicyRepositoryImpl(primaryDB),
		impl.NewDataPolicyRuleRepositoryImpl(primaryDB),
		impl.NewDataOwnershipFieldRepositoryImpl(primaryDB),
		impl.NewDataDimensionDefinitionRepositoryImpl(primaryDB),
		provider,
	)
	return resolver, db, provider, fixtures
}

func createPolicyResolverFixtures(t *testing.T, db *gorm.DB) policyResolverFixtures {
	t.Helper()
	serviceCode := "tms.transport_order"
	adapterFieldCode := "owner_org_id"
	fixtures := policyResolverFixtures{
		resource: model.DataResource{
			Basic:             model.Basic{Id: 101, State: true},
			ResourceCode:      policyResolverResourceCode,
			Name:              "运输订单",
			ResourceType:      model.DataResourceTypeBusinessService,
			ServiceCode:       &serviceCode,
			AdapterCode:       "registered_filter",
			PermissionEnabled: true,
		},
		operation: model.DataResourceOperation{
			Basic:      model.Basic{Id: 102, State: true},
			ResourceId: 101,
			Operation:  model.DataPermissionOperationQuery,
		},
		dimension: policyResolverDimension(201, datapermission.DimensionCodeManagementOrg),
		ownership: model.DataOwnershipField{
			Basic:            model.Basic{Id: 202, State: true},
			ResourceId:       101,
			OwnershipCode:    "owner_org",
			DimensionId:      201,
			BindingType:      model.DataOwnershipBindingTypeRegisteredField,
			AdapterFieldCode: &adapterFieldCode,
			ValueType:        model.DataDimensionValueTypeBigint,
		},
		policy: model.DataPolicy{
			Basic:      model.Basic{Id: 301, State: true},
			Code:       "management_org_scope",
			Name:       "管理组织范围",
			PolicyType: model.DataPolicyTypeRuleSet,
		},
		rule: model.DataPolicyRule{
			Basic:         model.Basic{Id: 302, State: true},
			PolicyId:      301,
			Sequence:      1,
			DimensionId:   201,
			OwnershipCode: "owner_org",
			ScopeSource:   model.DataPolicyScopeSourceEffectiveOrgUnits,
			Relation:      model.DataPolicyRelationExact,
			Operator:      model.DataPolicyOperatorIn,
		},
		grant: model.DataGrant{
			Basic:       model.Basic{Id: 401, State: true},
			SubjectType: model.DataGrantSubjectTypeRole,
			SubjectId:   3,
			ResourceId:  101,
			Operation:   model.DataPermissionOperationQuery,
			PolicyId:    301,
		},
	}
	testutil.MustCreate(t, db, &fixtures.resource)
	testutil.MustCreate(t, db, &fixtures.operation)
	testutil.MustCreate(t, db, &fixtures.dimension)
	testutil.MustCreate(t, db, &fixtures.ownership)
	testutil.MustCreate(t, db, &fixtures.policy)
	testutil.MustCreate(t, db, &fixtures.rule)
	testutil.MustCreate(t, db, &fixtures.grant)
	return fixtures
}

func policyResolverDimension(id int, code string) model.DataDimensionDefinition {
	return model.DataDimensionDefinition{
		Basic:        model.Basic{Id: id, State: true},
		Code:         code,
		Name:         code,
		Category:     model.DataDimensionCategoryOrganization,
		ValueType:    model.DataDimensionValueTypeBigint,
		ProviderCode: organizationDimensionProviderCode,
	}
}

func policyResolverInput(t *testing.T) datapermission.ResolverInput {
	t.Helper()
	input, err := datapermission.NewResolverInput(
		policyResolverSubject(t),
		policyResolverResourceCode,
		model.DataPermissionOperationQuery,
	)
	if err != nil {
		t.Fatalf("create ResolverInput: %v", err)
	}
	return input
}

func policyResolverSubject(t *testing.T) datapermission.SubjectContext {
	t.Helper()
	employeeId := 501
	subject, err := datapermission.NewSubjectContext(1001, []int{7, 3}, &employeeId, "2026-08-02")
	if err != nil {
		t.Fatalf("create SubjectContext: %v", err)
	}
	return subject
}

func policyResolverDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	return date
}

func mustUpdatePolicyResolverField(
	t *testing.T,
	db *gorm.DB,
	value any,
	id int,
	field string,
	fieldValue any,
) {
	t.Helper()
	if err := db.Model(value).Where("id = ?", id).Update(field, fieldValue).Error; err != nil {
		t.Fatalf("update %s: %v", field, err)
	}
}

func assertPolicyResolverError(t *testing.T, err error, code int) {
	t.Helper()
	var adminError *response.AdminError
	if !errors.As(err, &adminError) {
		t.Fatalf("expected AdminError, got %T: %v", err, err)
	}
	if adminError.ErrorCode != code {
		t.Fatalf("error code = %d, want %d", adminError.ErrorCode, code)
	}
}

func assertPolicyResolverNoAccess(t *testing.T, result datapermission.DataScopeResult) {
	t.Helper()
	if result.Decision() == datapermission.DataScopeDecisionAll {
		t.Fatal("resolver failure expanded access to all")
	}
	if result.Validate() == nil {
		t.Fatalf("resolver failure returned usable result: %s", result.Decision())
	}
}
