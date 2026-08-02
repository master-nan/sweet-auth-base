package service

import (
	"errors"
	"testing"
	"time"

	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type grantMergeResolverState struct {
	resource       model.DataResource
	operation      model.DataResourceOperation
	grants         []model.DataGrant
	policies       map[int]model.DataPolicy
	rules          map[int][]model.DataPolicyRule
	ownerships     map[string]model.DataOwnershipField
	dimensions     map[int]model.DataDimensionDefinition
	providerValues map[string]datapermission.DimensionValues
	providerErrors map[string]error
	providerCalls  []string
}

func TestDataPermissionPolicyResolverMergesTwoFilteredGrants(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	state.addSpecifiedPolicy(311, 312, 411, 7, `[99]`)

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve two filtered Grants: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered {
		t.Fatalf("decision = %s, want filtered", result.Decision())
	}
	groups := result.ConditionGroups()
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	assertGrantMergeGroupValues(t, groups, []int64{11, 12}, []int64{99})
}

func TestDataPermissionPolicyResolverAllCoversFiltered(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	state.addSimplePolicy(311, 411, 7, model.DataPolicyTypeAll)

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve all OR filtered: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionAll {
		t.Fatalf("decision = %s, want all", result.Decision())
	}
}

func TestDataPermissionPolicyResolverNoneKeepsFiltered(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	state.addSimplePolicy(311, 411, 7, model.DataPolicyTypeNone)

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve none OR filtered: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered ||
		len(result.ConditionGroups()) != 1 {
		t.Fatalf("unexpected result: decision=%s groups=%d", result.Decision(), len(result.ConditionGroups()))
	}
}

func TestDataPermissionPolicyResolverMergesMultipleRolesAndDeduplicatesConditions(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	state.grants = append(state.grants, grantMergeGrant(411, 7, state.grants[0].PolicyId))

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve multi-role Grants: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered ||
		len(result.ConditionGroups()) != 1 {
		t.Fatalf("duplicate condition was not removed: decision=%s groups=%d", result.Decision(), len(result.ConditionGroups()))
	}
	if len(state.providerCalls) != 2 {
		t.Fatalf("Provider calls = %d, want 2 Grants resolved", len(state.providerCalls))
	}
}

func TestDataPermissionPolicyResolverKeepsDifferentDimensionsInSeparateGroups(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	state.addLegalEntityPolicy(t, 311, 312, 411, 7)

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve different dimensions: %v", err)
	}
	groups := result.ConditionGroups()
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	seen := make(map[string]int)
	for _, group := range groups {
		conditions := group.Conditions()
		if len(conditions) != 1 {
			t.Fatalf("different Ownerships merged into one group: %+v", conditions)
		}
		seen[conditions[0].OwnershipCode()] = conditions[0].DimensionId()
	}
	if seen["owner_org"] != 201 || seen["legal_entity"] != 211 {
		t.Fatalf("unexpected Ownership dimensions: %+v", seen)
	}
}

func TestGrantScopeMergeRejectsNotApplicable(t *testing.T) {
	resolver, _ := newGrantMergeResolver(t)
	filtered, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve filtered result: %v", err)
	}
	notApplicable, err := datapermission.NewNotApplicableResult(
		policyResolverResourceCode,
		model.DataPermissionOperationQuery,
	)
	if err != nil {
		t.Fatalf("create not_applicable result: %v", err)
	}

	result, err := mergeGrantScopeResults(notApplicable, filtered)
	assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverConfigConflict)
	assertPolicyResolverNoAccess(t, result)
}

func TestDataPermissionPolicyResolverRejectsProviderTypeConflict(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	wrongValues, err := datapermission.NewDimensionValues(
		datapermission.DimensionCodeManagementOrg,
		datapermission.DataScopeValueTypeString,
		[]any{"11"},
	)
	if err != nil {
		t.Fatalf("create mismatched Provider values: %v", err)
	}
	state.providerValues[datapermission.DimensionCodeManagementOrg] = wrongValues

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverConfigConflict)
	assertPolicyResolverNoAccess(t, result)
}

func TestDataPermissionPolicyResolverDoesNotExpandAfterAllWhenAnotherGrantFails(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	allPolicy := state.addSimplePolicy(311, 411, 7, model.DataPolicyTypeAll)
	state.grants = []model.DataGrant{
		grantMergeGrant(411, 7, allPolicy.Id),
		state.grants[0],
	}
	state.providerErrors[datapermission.DimensionCodeManagementOrg] = errors.New("provider failed")

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	assertPolicyResolverError(t, err, myerrors.ErrorCodeDataPermissionResolverDimensionFailed)
	assertPolicyResolverNoAccess(t, result)
}

func newGrantMergeResolver(
	t *testing.T,
) (*DataPermissionPolicyResolver, *grantMergeResolverState) {
	t.Helper()
	managementValues, err := datapermission.NewDimensionValues(
		datapermission.DimensionCodeManagementOrg,
		datapermission.DataScopeValueTypeBigint,
		[]any{int64(12), int64(11), int64(12)},
	)
	if err != nil {
		t.Fatalf("create management Provider values: %v", err)
	}
	state := &grantMergeResolverState{
		resource: model.DataResource{
			Basic:             model.Basic{Id: 101, State: true},
			ResourceCode:      policyResolverResourceCode,
			PermissionEnabled: true,
		},
		operation: model.DataResourceOperation{
			Basic:      model.Basic{Id: 102, State: true},
			ResourceId: 101,
			Operation:  model.DataPermissionOperationQuery,
		},
		grants: []model.DataGrant{grantMergeGrant(401, 3, 301)},
		policies: map[int]model.DataPolicy{
			301: grantMergePolicy(301, model.DataPolicyTypeRuleSet),
		},
		rules: map[int][]model.DataPolicyRule{
			301: {grantMergeRule(302, 301, 201, "owner_org", model.DataPolicyScopeSourceEffectiveOrgUnits)},
		},
		ownerships: map[string]model.DataOwnershipField{
			"owner_org": grantMergeOwnership(202, 201, "owner_org"),
		},
		dimensions: map[int]model.DataDimensionDefinition{
			201: policyResolverDimension(201, datapermission.DimensionCodeManagementOrg),
		},
		providerValues: map[string]datapermission.DimensionValues{
			datapermission.DimensionCodeManagementOrg: managementValues,
		},
		providerErrors: make(map[string]error),
	}
	resolver := newDataPermissionPolicyResolver(
		func(_ *gin.Context, code string) (model.DataResource, error) {
			if code != state.resource.ResourceCode {
				return model.DataResource{}, gorm.ErrRecordNotFound
			}
			return state.resource, nil
		},
		func(_ *gin.Context, resourceId int, operation string) (model.DataResourceOperation, error) {
			if resourceId != state.operation.ResourceId || operation != state.operation.Operation {
				return model.DataResourceOperation{}, gorm.ErrRecordNotFound
			}
			return state.operation, nil
		},
		func(_ *gin.Context, _ int, _ []int, resourceId int, operation string, _ time.Time) ([]model.DataGrant, error) {
			if resourceId != state.resource.Id || operation != state.operation.Operation {
				return nil, nil
			}
			return append([]model.DataGrant(nil), state.grants...), nil
		},
		func(_ *gin.Context, policyId int) (model.DataPolicy, error) {
			policy, ok := state.policies[policyId]
			if !ok {
				return model.DataPolicy{}, gorm.ErrRecordNotFound
			}
			return policy, nil
		},
		func(_ *gin.Context, policyId int) ([]model.DataPolicyRule, error) {
			return append([]model.DataPolicyRule(nil), state.rules[policyId]...), nil
		},
		func(_ *gin.Context, resourceId int, ownershipCode string) (model.DataOwnershipField, error) {
			ownership, ok := state.ownerships[ownershipCode]
			if !ok || ownership.ResourceId != resourceId {
				return model.DataOwnershipField{}, gorm.ErrRecordNotFound
			}
			return ownership, nil
		},
		func(_ *gin.Context, dimensionId int) (model.DataDimensionDefinition, error) {
			dimension, ok := state.dimensions[dimensionId]
			if !ok {
				return model.DataDimensionDefinition{}, gorm.ErrRecordNotFound
			}
			return dimension, nil
		},
		func(
			_ *gin.Context,
			_ datapermission.SubjectContext,
			dimensionCode string,
		) (datapermission.DimensionValues, error) {
			state.providerCalls = append(state.providerCalls, dimensionCode)
			if err := state.providerErrors[dimensionCode]; err != nil {
				return datapermission.DimensionValues{}, err
			}
			values, ok := state.providerValues[dimensionCode]
			if !ok {
				return datapermission.DimensionValues{}, errors.New("dimension values missing")
			}
			return values, nil
		},
	)
	return resolver, state
}

func (state *grantMergeResolverState) addSpecifiedPolicy(
	policyId int,
	ruleId int,
	grantId int,
	roleId int,
	values string,
) {
	state.policies[policyId] = grantMergePolicy(policyId, model.DataPolicyTypeRuleSet)
	rule := grantMergeRule(
		ruleId,
		policyId,
		201,
		"owner_org",
		model.DataPolicyScopeSourceSpecifiedValues,
	)
	rule.SpecifiedValues = datatypes.JSON([]byte(values))
	state.rules[policyId] = []model.DataPolicyRule{rule}
	state.grants = append(state.grants, grantMergeGrant(grantId, roleId, policyId))
}

func (state *grantMergeResolverState) addSimplePolicy(
	policyId int,
	grantId int,
	roleId int,
	policyType string,
) model.DataPolicy {
	policy := grantMergePolicy(policyId, policyType)
	state.policies[policyId] = policy
	state.grants = append(state.grants, grantMergeGrant(grantId, roleId, policyId))
	return policy
}

func (state *grantMergeResolverState) addLegalEntityPolicy(
	t *testing.T,
	policyId int,
	ruleId int,
	grantId int,
	roleId int,
) {
	t.Helper()
	state.dimensions[211] = policyResolverDimension(211, datapermission.DimensionCodeLegalEntity)
	state.ownerships["legal_entity"] = grantMergeOwnership(212, 211, "legal_entity")
	state.policies[policyId] = grantMergePolicy(policyId, model.DataPolicyTypeRuleSet)
	state.rules[policyId] = []model.DataPolicyRule{grantMergeRule(
		ruleId,
		policyId,
		211,
		"legal_entity",
		model.DataPolicyScopeSourceEffectiveLegalEntities,
	)}
	state.grants = append(state.grants, grantMergeGrant(grantId, roleId, policyId))
	values, err := datapermission.NewDimensionValues(
		datapermission.DimensionCodeLegalEntity,
		datapermission.DataScopeValueTypeBigint,
		[]any{int64(21)},
	)
	if err != nil {
		t.Fatalf("create legal entity Provider values: %v", err)
	}
	state.providerValues[datapermission.DimensionCodeLegalEntity] = values
}

func grantMergePolicy(id int, policyType string) model.DataPolicy {
	return model.DataPolicy{
		Basic:      model.Basic{Id: id, State: true},
		Code:       "merge_policy",
		Name:       "合并策略",
		PolicyType: policyType,
	}
}

func grantMergeGrant(id int, roleId int, policyId int) model.DataGrant {
	return model.DataGrant{
		Basic:       model.Basic{Id: id, State: true},
		SubjectType: model.DataGrantSubjectTypeRole,
		SubjectId:   roleId,
		ResourceId:  101,
		Operation:   model.DataPermissionOperationQuery,
		PolicyId:    policyId,
	}
}

func grantMergeRule(
	id int,
	policyId int,
	dimensionId int,
	ownershipCode string,
	scopeSource string,
) model.DataPolicyRule {
	return model.DataPolicyRule{
		Basic:         model.Basic{Id: id, State: true},
		PolicyId:      policyId,
		Sequence:      1,
		DimensionId:   dimensionId,
		OwnershipCode: ownershipCode,
		ScopeSource:   scopeSource,
		Relation:      model.DataPolicyRelationExact,
		Operator:      model.DataPolicyOperatorIn,
	}
}

func grantMergeOwnership(id int, dimensionId int, ownershipCode string) model.DataOwnershipField {
	return model.DataOwnershipField{
		Basic:         model.Basic{Id: id, State: true},
		ResourceId:    101,
		OwnershipCode: ownershipCode,
		DimensionId:   dimensionId,
		BindingType:   model.DataOwnershipBindingTypeRegisteredField,
		ValueType:     model.DataDimensionValueTypeBigint,
	}
}

func assertGrantMergeGroupValues(
	t *testing.T,
	groups []datapermission.DataScopeConditionGroup,
	want ...[]int64,
) {
	t.Helper()
	remaining := make([][]int64, len(want))
	copy(remaining, want)
	for _, group := range groups {
		conditions := group.Conditions()
		if len(conditions) != 1 {
			t.Fatalf("condition count = %d, want 1", len(conditions))
		}
		values := conditions[0].BigintValues()
		matched := -1
		for index, candidate := range remaining {
			if equalInt64Values(values, candidate) {
				matched = index
				break
			}
		}
		if matched < 0 {
			t.Fatalf("unexpected condition values: %v", values)
		}
		remaining = append(remaining[:matched], remaining[matched+1:]...)
	}
	if len(remaining) != 0 {
		t.Fatalf("missing condition values: %v", remaining)
	}
}

func equalInt64Values(left []int64, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
