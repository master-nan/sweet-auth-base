package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type policyResolverResourceLookup func(*gin.Context, string) (model.DataResource, error)
type policyResolverOperationLookup func(*gin.Context, int, string) (model.DataResourceOperation, error)
type policyResolverGrantLookup func(
	*gin.Context,
	int,
	[]int,
	int,
	string,
	time.Time,
) ([]model.DataGrant, error)
type policyResolverPolicyLookup func(*gin.Context, int) (model.DataPolicy, error)
type policyResolverRuleLookup func(*gin.Context, int) ([]model.DataPolicyRule, error)
type policyResolverOwnershipLookup func(*gin.Context, int, string) (model.DataOwnershipField, error)
type policyResolverDimensionLookup func(*gin.Context, int) (model.DataDimensionDefinition, error)
type policyResolverDimensionValuesLookup func(
	*gin.Context,
	datapermission.SubjectContext,
	string,
) (datapermission.DimensionValues, error)

// DataPermissionPolicyResolver resolves all effective Grants for one resource
// and operation. Each Grant still resolves exactly one Policy and one Rule.
type DataPermissionPolicyResolver struct {
	findResource     policyResolverResourceLookup
	findOperation    policyResolverOperationLookup
	findGrants       policyResolverGrantLookup
	findPolicy       policyResolverPolicyLookup
	findRules        policyResolverRuleLookup
	findOwnership    policyResolverOwnershipLookup
	findDimension    policyResolverDimensionLookup
	resolveDimension policyResolverDimensionValuesLookup
}

var _ datapermission.Resolver = (*DataPermissionPolicyResolver)(nil)

func NewDataPermissionPolicyResolver(
	resourceRepo repository.DataResourceRepository,
	operationRepo repository.DataResourceOperationRepository,
	grantRepo repository.DataGrantRepository,
	policyRepo repository.DataPolicyRepository,
	ruleRepo repository.DataPolicyRuleRepository,
	ownershipRepo repository.DataOwnershipFieldRepository,
	dimensionRepo repository.DataDimensionDefinitionRepository,
	dimensionProvider DimensionProvider,
) *DataPermissionPolicyResolver {
	return newDataPermissionPolicyResolver(
		resourceRepo.FindByCode,
		operationRepo.FindByStableKey,
		grantRepo.ListEffectiveBySubjects,
		policyRepo.FindByIdForConfig,
		ruleRepo.ListByPolicy,
		ownershipRepo.FindByStableKey,
		dimensionRepo.FindByIdForConfig,
		dimensionProvider.ResolveDimensionValues,
	)
}

func newDataPermissionPolicyResolver(
	findResource policyResolverResourceLookup,
	findOperation policyResolverOperationLookup,
	findGrants policyResolverGrantLookup,
	findPolicy policyResolverPolicyLookup,
	findRules policyResolverRuleLookup,
	findOwnership policyResolverOwnershipLookup,
	findDimension policyResolverDimensionLookup,
	resolveDimension policyResolverDimensionValuesLookup,
) *DataPermissionPolicyResolver {
	return &DataPermissionPolicyResolver{
		findResource:     findResource,
		findOperation:    findOperation,
		findGrants:       findGrants,
		findPolicy:       findPolicy,
		findRules:        findRules,
		findOwnership:    findOwnership,
		findDimension:    findDimension,
		resolveDimension: resolveDimension,
	}
}

func (resolver *DataPermissionPolicyResolver) Resolve(
	ctx *gin.Context,
	input datapermission.ResolverInput,
) (datapermission.DataScopeResult, error) {
	if resolver == nil {
		return datapermission.ResolverFunc(nil).Resolve(ctx, input)
	}
	return datapermission.ResolverFunc(resolver.resolveGrantPolicies).Resolve(ctx, input)
}

func (resolver *DataPermissionPolicyResolver) resolveGrantPolicies(
	ctx *gin.Context,
	input datapermission.ResolverInput,
) (datapermission.DataScopeResult, error) {
	if !resolver.hasRequiredLookups() {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverFailed
	}

	resource, err := resolver.findResource(ctx, input.ResourceCode())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverResourceMissing
		}
		return datapermission.DataScopeResult{}, err
	}
	if resource.Id <= 0 || resource.ResourceCode != input.ResourceCode() || !resource.State {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverResourceMissing
	}

	operation, err := resolver.findOperation(ctx, resource.Id, input.Operation())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverOperationMissing
		}
		return datapermission.DataScopeResult{}, err
	}
	if operation.Id <= 0 || operation.ResourceId != resource.Id ||
		operation.Operation != input.Operation() || !operation.State {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverOperationMissing
	}
	if !resource.PermissionEnabled {
		return datapermission.NewNotApplicableResult(input.ResourceCode(), input.Operation())
	}

	subject := input.SubjectContext()
	asOf, err := time.Parse(time.DateOnly, subject.AsOfDate())
	if err != nil {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionSubjectContextInvalid
	}
	grants, err := resolver.findGrants(
		ctx,
		subject.UserId(),
		subject.RoleIds(),
		resource.Id,
		input.Operation(),
		asOf,
	)
	if err != nil {
		return datapermission.DataScopeResult{}, err
	}
	if len(grants) == 0 {
		return datapermission.NewNoneResult(input.ResourceCode(), input.Operation())
	}
	result, err := datapermission.NewNoneResult(input.ResourceCode(), input.Operation())
	if err != nil {
		return datapermission.DataScopeResult{}, err
	}
	for _, grant := range grants {
		if !validResolverGrant(grant, subject, resource.Id, input.Operation(), asOf) {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
		}
		grantResult, resolveErr := resolver.resolveGrant(ctx, input, resource, grant)
		if resolveErr != nil {
			return datapermission.DataScopeResult{}, resolveErr
		}
		result, err = mergeGrantScopeResults(result, grantResult)
		if err != nil {
			return datapermission.DataScopeResult{}, err
		}
	}
	return result, nil
}

func (resolver *DataPermissionPolicyResolver) resolveGrant(
	ctx *gin.Context,
	input datapermission.ResolverInput,
	resource model.DataResource,
	grant model.DataGrant,
) (datapermission.DataScopeResult, error) {
	policy, err := resolver.findPolicy(ctx, grant.PolicyId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverPolicyInvalid
		}
		return datapermission.DataScopeResult{}, err
	}
	if policy.Id != grant.PolicyId || !policy.State {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverPolicyInvalid
	}

	rules, err := resolver.findRules(ctx, policy.Id)
	if err != nil {
		return datapermission.DataScopeResult{}, err
	}
	activeRules := activePolicyResolverRules(rules)
	switch policy.PolicyType {
	case model.DataPolicyTypeAll:
		if len(activeRules) != 0 {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
		}
		return datapermission.NewAllResult(input.ResourceCode(), input.Operation())
	case model.DataPolicyTypeNone:
		if len(activeRules) != 0 {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
		}
		return datapermission.NewNoneResult(input.ResourceCode(), input.Operation())
	case model.DataPolicyTypeRuleSet:
		if len(activeRules) == 0 {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverPolicyInvalid
		}
		if len(activeRules) != 1 {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
		}
	default:
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverPolicyInvalid
	}
	if activeRules[0].PolicyId != policy.Id {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}

	return resolver.resolveRule(ctx, input, resource, activeRules[0])
}

func mergeGrantScopeResults(
	left datapermission.DataScopeResult,
	right datapermission.DataScopeResult,
) (datapermission.DataScopeResult, error) {
	if left.Decision() == datapermission.DataScopeDecisionNotApplicable ||
		right.Decision() == datapermission.DataScopeDecisionNotApplicable {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}
	result, err := datapermission.OrDataScopeResults(left, right)
	if err != nil {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}
	return result, nil
}

func (resolver *DataPermissionPolicyResolver) resolveRule(
	ctx *gin.Context,
	input datapermission.ResolverInput,
	resource model.DataResource,
	rule model.DataPolicyRule,
) (datapermission.DataScopeResult, error) {
	if rule.PolicyId <= 0 || rule.DimensionId <= 0 || strings.TrimSpace(rule.OwnershipCode) == "" ||
		rule.Relation != model.DataPolicyRelationExact {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}

	ownership, err := resolver.findOwnership(ctx, resource.Id, rule.OwnershipCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverOwnershipMissing
		}
		return datapermission.DataScopeResult{}, err
	}
	if ownership.Id <= 0 || !ownership.State || ownership.ResourceId != resource.Id ||
		ownership.OwnershipCode != rule.OwnershipCode {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverOwnershipMissing
	}
	if ownership.DimensionId != rule.DimensionId {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}

	dimension, err := resolver.findDimension(ctx, rule.DimensionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverDimensionFailed
		}
		return datapermission.DataScopeResult{}, err
	}
	valueType, ok := resolverValueType(dimension.ValueType)
	if dimension.Id != rule.DimensionId || !dimension.State || !ok || ownership.ValueType != dimension.ValueType {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}
	if !validResolverRuleSource(rule, dimension.Code) {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}

	values, err := resolver.resolveRuleValues(ctx, input.SubjectContext(), rule, dimension, valueType)
	if err != nil {
		return datapermission.DataScopeResult{}, err
	}
	if len(values) == 0 {
		return datapermission.NewNoneResult(input.ResourceCode(), input.Operation())
	}
	condition, err := datapermission.NewDataScopeCondition(datapermission.DataScopeConditionInput{
		OwnershipCode: ownership.OwnershipCode,
		DimensionId:   dimension.Id,
		Operator:      datapermission.DataScopeOperator(rule.Operator),
		ValueType:     valueType,
		Values:        values,
	})
	if err != nil {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}
	group, err := datapermission.NewDataScopeConditionGroup([]datapermission.DataScopeCondition{condition})
	if err != nil {
		return datapermission.DataScopeResult{}, myerrors.ErrDataPermissionResolverConfigConflict
	}
	return datapermission.NewFilteredResult(
		input.ResourceCode(),
		input.Operation(),
		[]datapermission.DataScopeConditionGroup{group},
	)
}

func (resolver *DataPermissionPolicyResolver) resolveRuleValues(
	ctx *gin.Context,
	subject datapermission.SubjectContext,
	rule model.DataPolicyRule,
	dimension model.DataDimensionDefinition,
	valueType datapermission.DataScopeValueType,
) ([]any, error) {
	if rule.ScopeSource == model.DataPolicyScopeSourceSpecifiedValues {
		values, err := decodeResolverSpecifiedValues(rule.SpecifiedValues)
		if err != nil {
			return nil, myerrors.ErrDataPermissionResolverConfigConflict
		}
		return values, nil
	}

	values, err := resolver.resolveDimension(ctx, subject, dimension.Code)
	if err != nil {
		return nil, myerrors.ErrDataPermissionResolverDimensionFailed
	}
	if err = values.Validate(); err != nil {
		return nil, myerrors.ErrDataPermissionResolverDimensionFailed
	}
	if values.DimensionCode() != dimension.Code || values.ValueType() != valueType {
		return nil, myerrors.ErrDataPermissionResolverConfigConflict
	}
	return values.Values(), nil
}

func (resolver *DataPermissionPolicyResolver) hasRequiredLookups() bool {
	return resolver.findResource != nil && resolver.findOperation != nil &&
		resolver.findGrants != nil && resolver.findPolicy != nil && resolver.findRules != nil &&
		resolver.findOwnership != nil && resolver.findDimension != nil &&
		resolver.resolveDimension != nil
}

func validResolverGrant(
	grant model.DataGrant,
	subject datapermission.SubjectContext,
	resourceId int,
	operation string,
	asOf time.Time,
) bool {
	if grant.Id <= 0 || !grant.State || grant.ResourceId != resourceId ||
		grant.Operation != operation || grant.PolicyId <= 0 {
		return false
	}
	if grant.ValidFrom != nil && grant.ValidFrom.After(asOf) ||
		grant.ValidTo != nil && grant.ValidTo.Before(asOf) {
		return false
	}
	if grant.SubjectType == model.DataGrantSubjectTypeUser {
		return grant.SubjectId == subject.UserId()
	}
	if grant.SubjectType != model.DataGrantSubjectTypeRole {
		return false
	}
	for _, roleId := range subject.RoleIds() {
		if grant.SubjectId == roleId {
			return true
		}
	}
	return false
}

func activePolicyResolverRules(rules []model.DataPolicyRule) []model.DataPolicyRule {
	active := make([]model.DataPolicyRule, 0, len(rules))
	for _, rule := range rules {
		if rule.State {
			active = append(active, rule)
		}
	}
	return active
}

func resolverValueType(valueType string) (datapermission.DataScopeValueType, bool) {
	switch valueType {
	case model.DataDimensionValueTypeBigint:
		return datapermission.DataScopeValueTypeBigint, true
	case model.DataDimensionValueTypeString:
		return datapermission.DataScopeValueTypeString, true
	default:
		return "", false
	}
}

func validResolverRuleSource(rule model.DataPolicyRule, dimensionCode string) bool {
	if rule.Operator != model.DataPolicyOperatorEqual && rule.Operator != model.DataPolicyOperatorIn {
		return false
	}
	if rule.ScopeSource == model.DataPolicyScopeSourceSpecifiedValues {
		return true
	}
	switch rule.ScopeSource {
	case model.DataPolicyScopeSourceEffectiveLegalEntities:
		return dimensionCode == datapermission.DimensionCodeLegalEntity &&
			rule.Operator == model.DataPolicyOperatorIn
	case model.DataPolicyScopeSourceEffectiveOrgUnits:
		return dimensionCode == datapermission.DimensionCodeManagementOrg &&
			rule.Operator == model.DataPolicyOperatorIn
	case model.DataPolicyScopeSourceCurrentEmployee:
		return dimensionCode == datapermission.DimensionCodeEmployee &&
			rule.Operator == model.DataPolicyOperatorEqual
	default:
		return false
	}
}

func decodeResolverSpecifiedValues(raw []byte) ([]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, errors.New("specified values are empty")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var values []any
	if err := decoder.Decode(&values); err != nil {
		return nil, err
	}
	if err := ensurePolicyJSONEOF(decoder); err != nil {
		return nil, err
	}
	return values, nil
}
