package service

import (
	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"gorm.io/gorm"
)

const (
	dataPermissionPreflightEnableAction  = "enable"
	dataPermissionPreflightDisableAction = "disable"
	maxDataPermissionBatchEnable         = 100

	dataPermissionObjectResource  = "resource"
	dataPermissionObjectPolicy    = "policy"
	dataPermissionObjectRule      = "policy_rule"
	dataPermissionObjectGrant     = "grant"
	dataPermissionObjectOwnership = "ownership"
	dataPermissionObjectDimension = "dimension"
	dataPermissionObjectOperation = "resource_operation"
	dataPermissionObjectSubject   = "subject"

	diagnosticResourceNotFound          = "resource_not_found"
	diagnosticResourceInactive          = "resource_inactive"
	diagnosticOwnershipRequired         = "ownership_required"
	diagnosticOwnershipNotFound         = "ownership_not_found"
	diagnosticOwnershipInactive         = "ownership_inactive"
	diagnosticOwnershipDimension        = "ownership_dimension_mismatch"
	diagnosticOwnershipBinding          = "ownership_binding_invalid"
	diagnosticPolicyNotFound            = "policy_not_found"
	diagnosticPolicyInactive            = "policy_inactive"
	diagnosticPolicyTypeInvalid         = "policy_type_invalid"
	diagnosticPolicyRuleRequired        = "policy_rule_required"
	diagnosticPolicyRuleCount           = "policy_rule_count_invalid"
	diagnosticRuleIdentity              = "policy_rule_identity_invalid"
	diagnosticDimensionNotFound         = "dimension_not_found"
	diagnosticDimensionInactive         = "dimension_inactive"
	diagnosticScopeSourceInvalid        = "scope_source_invalid"
	diagnosticRelationInvalid           = "relation_invalid"
	diagnosticOperatorInvalid           = "operator_invalid"
	diagnosticSpecifiedValuesInvalid    = "specified_values_invalid"
	diagnosticProviderCapabilityInvalid = "provider_capability_unsupported"
	diagnosticGrantNotFound             = "grant_not_found"
	diagnosticGrantSubjectType          = "grant_subject_type_invalid"
	diagnosticGrantSubjectNotFound      = "grant_subject_not_found"
	diagnosticGrantValidity             = "grant_validity_invalid"
	diagnosticOperationNotFound         = "resource_operation_not_found"
	diagnosticOperationInactive         = "resource_operation_inactive"
)

// DataPermissionConfigPreflightService 校验配置声明并变更其启用状态。
// 它不解析主体范围，也不构造数据过滤条件。
type DataPermissionConfigPreflightService struct {
	validator   dataPermissionConfigValidator
	auditWriter TransactionalAuditWriter
}

type dataPermissionConfigValidator struct {
	grantRepo              repository.DataGrantRepository
	resourceRepo           repository.DataResourceRepository
	operationRepo          repository.DataResourceOperationRepository
	ownershipRepo          repository.DataOwnershipFieldRepository
	dimensionRepo          repository.DataDimensionDefinitionRepository
	policyRepo             repository.DataPolicyRepository
	ruleRepo               repository.DataPolicyRuleRepository
	registeredFieldChecker datapermission.OwnershipFieldOperationValidator
}

type dataPermissionValidationCollector struct {
	errors []response.DataPermissionValidationErrorRes
	seen   map[string]struct{}
}

func NewDataPermissionConfigPreflightService(
	grantRepo repository.DataGrantRepository,
	resourceRepo repository.DataResourceRepository,
	operationRepo repository.DataResourceOperationRepository,
	ownershipRepo repository.DataOwnershipFieldRepository,
	dimensionRepo repository.DataDimensionDefinitionRepository,
	policyRepo repository.DataPolicyRepository,
	ruleRepo repository.DataPolicyRuleRepository,
	registeredFieldChecker datapermission.OwnershipFieldOperationValidator,
	auditWriter TransactionalAuditWriter,
) *DataPermissionConfigPreflightService {
	return &DataPermissionConfigPreflightService{
		validator: dataPermissionConfigValidator{
			grantRepo:              grantRepo,
			resourceRepo:           resourceRepo,
			operationRepo:          operationRepo,
			ownershipRepo:          ownershipRepo,
			dimensionRepo:          dimensionRepo,
			policyRepo:             policyRepo,
			ruleRepo:               ruleRepo,
			registeredFieldChecker: registeredFieldChecker,
		},
		auditWriter: auditWriter,
	}
}

func (s *DataPermissionConfigPreflightService) PreflightResource(
	ctx context.Context,
	resourceId int,
) (response.DataPermissionValidationResultRes, error) {
	if resourceId <= 0 {
		return response.DataPermissionValidationResultRes{}, myerrors.NewParameterError(
			"resource_id必须大于0",
		)
	}
	return s.validator.validateResource(s.validator.resourceRepo.DBWithContext(ctx), resourceId)
}

func (s *DataPermissionConfigPreflightService) PreflightPolicy(
	ctx context.Context,
	policyId int,
) (response.DataPermissionValidationResultRes, error) {
	if policyId <= 0 {
		return response.DataPermissionValidationResultRes{}, myerrors.NewParameterError(
			"policy_id必须大于0",
		)
	}
	return s.validator.validatePolicy(s.validator.policyRepo.DBWithContext(ctx), policyId)
}

func (s *DataPermissionConfigPreflightService) PreflightGrant(
	ctx context.Context,
	grantId int,
) (response.DataPermissionValidationResultRes, error) {
	if grantId <= 0 {
		return response.DataPermissionValidationResultRes{}, myerrors.NewParameterError(
			"grant_id必须大于0",
		)
	}
	return s.validator.validateGrant(s.validator.grantRepo.DBWithContext(ctx), grantId)
}

func (s *DataPermissionConfigPreflightService) EnableResource(
	ctx context.Context,
	resourceId int,
) (response.DataPermissionValidationResultRes, error) {
	return s.EnableResources(ctx, []int{resourceId})
}

func (s *DataPermissionConfigPreflightService) EnableResources(
	ctx context.Context,
	resourceIds []int,
) (response.DataPermissionValidationResultRes, error) {
	if ctx == nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapSystemError(
			ErrTransactionContextRequired,
		)
	}
	resourceIds, err := normalizePreflightIds(resourceIds)
	if err != nil {
		return response.DataPermissionValidationResultRes{}, err
	}

	result := validDataPermissionValidationResult()
	err = RunInTransaction(ctx, s.validator.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		collector := newDataPermissionValidationCollector()
		resources := make([]model.DataResource, 0, len(resourceIds))
		for _, resourceId := range resourceIds {
			validation, validationErr := s.validator.validateResource(tx, resourceId)
			if validationErr != nil {
				return validationErr
			}
			collector.merge(validation)
			resource, findErr := s.validator.resourceRepo.FindByIdWithDB(tx, resourceId)
			if findErr == nil {
				resources = append(resources, resource)
			} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return myerrors.WrapDatabaseError(findErr)
			}
		}
		result = collector.result()
		if !result.Valid {
			return myerrors.ErrDataPermissionPreflightFailed
		}
		for _, resource := range resources {
			if resource.PermissionEnabled {
				continue
			}
			changed, updateErr := s.validator.resourceRepo.UpdateFields(
				tx,
				resource.Id,
				map[string]any{"permission_enabled": true},
			)
			if updateErr != nil {
				return myerrors.WrapDatabaseError(updateErr)
			}
			if !changed {
				return myerrors.ErrDataResourceNotFound
			}
			if auditErr := s.recordStateAudit(
				ctx,
				tx,
				dataResourceAuditType,
				resource.ResourceCode,
				resource.Id,
				dataPermissionPreflightEnableAction,
				"permission_enabled",
				false,
				true,
			); auditErr != nil {
				return auditErr
			}
		}
		return nil
	})
	return result, err
}

func (s *DataPermissionConfigPreflightService) DisableResource(
	ctx context.Context,
	resourceId int,
) (response.DataPermissionValidationResultRes, error) {
	if ctx == nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapSystemError(
			ErrTransactionContextRequired,
		)
	}
	if resourceId <= 0 {
		return response.DataPermissionValidationResultRes{}, myerrors.NewParameterError(
			"resource_id必须大于0",
		)
	}
	result := validDataPermissionValidationResult()
	err := RunInTransaction(ctx, s.validator.resourceRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		resource, findErr := s.validator.resourceRepo.FindByIdWithDB(tx, resourceId)
		if findErr != nil {
			return mapDataResourceReadError(findErr)
		}
		if !resource.PermissionEnabled {
			return nil
		}
		changed, updateErr := s.validator.resourceRepo.UpdateFields(
			tx,
			resource.Id,
			map[string]any{"permission_enabled": false},
		)
		if updateErr != nil {
			return myerrors.WrapDatabaseError(updateErr)
		}
		if !changed {
			return myerrors.ErrDataResourceNotFound
		}
		return s.recordStateAudit(
			ctx,
			tx,
			dataResourceAuditType,
			resource.ResourceCode,
			resource.Id,
			dataPermissionPreflightDisableAction,
			"permission_enabled",
			true,
			false,
		)
	})
	return result, err
}

func (s *DataPermissionConfigPreflightService) EnablePolicy(
	ctx context.Context,
	policyId int,
) (response.DataPermissionValidationResultRes, error) {
	return s.setPolicyState(ctx, policyId, true)
}

func (s *DataPermissionConfigPreflightService) DisablePolicy(
	ctx context.Context,
	policyId int,
) (response.DataPermissionValidationResultRes, error) {
	return s.setPolicyState(ctx, policyId, false)
}

func (s *DataPermissionConfigPreflightService) EnableGrant(
	ctx context.Context,
	grantId int,
) (response.DataPermissionValidationResultRes, error) {
	return s.setGrantState(ctx, grantId, true)
}

func (s *DataPermissionConfigPreflightService) DisableGrant(
	ctx context.Context,
	grantId int,
) (response.DataPermissionValidationResultRes, error) {
	return s.setGrantState(ctx, grantId, false)
}

func (s *DataPermissionConfigPreflightService) setPolicyState(
	ctx context.Context,
	policyId int,
	state bool,
) (response.DataPermissionValidationResultRes, error) {
	if ctx == nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapSystemError(
			ErrTransactionContextRequired,
		)
	}
	if policyId <= 0 {
		return response.DataPermissionValidationResultRes{}, myerrors.NewParameterError(
			"policy_id必须大于0",
		)
	}
	result := validDataPermissionValidationResult()
	err := RunInTransaction(ctx, s.validator.policyRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		policy, findErr := s.validator.policyRepo.FindByIdWithDB(tx, policyId)
		if findErr != nil {
			return mapDataPolicyReadError(findErr)
		}
		if state {
			result, findErr = s.validator.validatePolicy(tx, policyId)
			if findErr != nil {
				return findErr
			}
			if !result.Valid {
				return myerrors.ErrDataPermissionPreflightFailed
			}
		}
		if policy.State == state {
			return nil
		}
		changed, updateErr := s.validator.policyRepo.UpdateFields(
			tx,
			policy.Id,
			map[string]any{"state": state},
		)
		if updateErr != nil {
			return myerrors.WrapDatabaseError(updateErr)
		}
		if !changed {
			return myerrors.ErrDataPolicyNotFound
		}
		action := dataPermissionPreflightDisableAction
		if state {
			action = dataPermissionPreflightEnableAction
		}
		return s.recordStateAudit(
			ctx,
			tx,
			dataPolicyAuditType,
			policy.Code,
			policy.Id,
			action,
			"state",
			policy.State,
			state,
		)
	})
	return result, err
}

func (s *DataPermissionConfigPreflightService) setGrantState(
	ctx context.Context,
	grantId int,
	state bool,
) (response.DataPermissionValidationResultRes, error) {
	if ctx == nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapSystemError(
			ErrTransactionContextRequired,
		)
	}
	if grantId <= 0 {
		return response.DataPermissionValidationResultRes{}, myerrors.NewParameterError(
			"grant_id必须大于0",
		)
	}
	result := validDataPermissionValidationResult()
	err := RunInTransaction(ctx, s.validator.grantRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		grant, findErr := s.validator.grantRepo.FindByIdWithDB(tx, grantId)
		if findErr != nil {
			return mapDataGrantReadError(findErr)
		}
		if state {
			result, findErr = s.validator.validateGrantRecord(tx, grant, false)
			if findErr != nil {
				return findErr
			}
			if !result.Valid {
				return myerrors.ErrDataPermissionPreflightFailed
			}
		}
		if grant.State == state {
			return nil
		}
		changed, updateErr := s.validator.grantRepo.UpdateFields(
			tx,
			grant.Id,
			map[string]any{"state": state},
		)
		if updateErr != nil {
			return myerrors.WrapDatabaseError(updateErr)
		}
		if !changed {
			return myerrors.ErrDataGrantNotFound
		}
		action := dataPermissionPreflightDisableAction
		if state {
			action = dataPermissionPreflightEnableAction
		}
		return s.recordStateAudit(
			ctx,
			tx,
			dataGrantAuditType,
			dataGrantStableKey(grant),
			grant.Id,
			action,
			"state",
			grant.State,
			state,
		)
	})
	return result, err
}

func (s *DataPermissionConfigPreflightService) recordStateAudit(
	ctx context.Context,
	tx *gorm.DB,
	resourceType string,
	resourceCode string,
	resourceId int,
	action string,
	field string,
	oldValue any,
	newValue any,
) error {
	if s.auditWriter == nil {
		return myerrors.WrapSystemError(ErrTransactionalAuditRepositoryRequired)
	}
	if err := s.auditWriter.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: resourceType,
		ResourceCode: resourceCode,
		ResourceId:   strconv.Itoa(resourceId),
		Changes: map[string]TransactionalAuditChange{
			field: {OldValue: oldValue, NewValue: newValue},
		},
	}); err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func (v dataPermissionConfigValidator) validateResource(
	tx *gorm.DB,
	resourceId int,
) (response.DataPermissionValidationResultRes, error) {
	collector := newDataPermissionValidationCollector()
	resource, err := v.resourceRepo.FindByIdWithDB(tx, resourceId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		collector.add(
			diagnosticResourceNotFound,
			"数据资源不存在",
			dataPermissionObjectResource,
			resourceId,
		)
		return collector.result(), nil
	}
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}
	if !resource.State {
		collector.add(
			diagnosticResourceInactive,
			"数据资源已停用",
			dataPermissionObjectResource,
			resource.Id,
		)
	}

	ownerships, err := v.ownershipRepo.ListByResourceForConfigDB(tx, resource.Id)
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}
	activeOwnershipCount := 0
	for _, ownership := range ownerships {
		if ownership.State {
			activeOwnershipCount++
		}
	}
	if activeOwnershipCount == 0 {
		collector.add(
			diagnosticOwnershipRequired,
			"数据资源至少需要一个有效归属定义",
			dataPermissionObjectResource,
			resource.Id,
		)
	}

	grants, err := v.grantRepo.ListByResourceForConfigDB(tx, resource.Id)
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}
	for _, grant := range grants {
		if !grant.State {
			continue
		}
		validation, validationErr := v.validateGrantRecord(tx, grant, false)
		if validationErr != nil {
			return response.DataPermissionValidationResultRes{}, validationErr
		}
		collector.merge(validation)
	}
	return collector.result(), nil
}

func (v dataPermissionConfigValidator) validatePolicy(
	tx *gorm.DB,
	policyId int,
) (response.DataPermissionValidationResultRes, error) {
	collector := newDataPermissionValidationCollector()
	policy, err := v.policyRepo.FindByIdWithDB(tx, policyId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		collector.add(
			diagnosticPolicyNotFound,
			"数据权限策略不存在",
			dataPermissionObjectPolicy,
			policyId,
		)
		return collector.result(), nil
	}
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}

	rules, err := v.validatePolicyRules(tx, policy, collector)
	if err != nil {
		return response.DataPermissionValidationResultRes{}, err
	}
	grants, err := v.grantRepo.ListByPolicyForConfigDB(tx, policy.Id)
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}
	for _, grant := range grants {
		if !grant.State {
			continue
		}
		resource, findErr := v.resourceRepo.FindByIdWithDB(tx, grant.ResourceId)
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			collector.add(
				diagnosticResourceNotFound,
				"授权引用的数据资源不存在",
				dataPermissionObjectGrant,
				grant.Id,
			)
			continue
		}
		if findErr != nil {
			return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(findErr)
		}
		if !resource.State {
			collector.add(
				diagnosticResourceInactive,
				"授权引用的数据资源已停用",
				dataPermissionObjectGrant,
				grant.Id,
			)
		}
		if err = v.validateRulesForResource(tx, resource, grant.Operation, rules, collector); err != nil {
			return response.DataPermissionValidationResultRes{}, err
		}
	}
	return collector.result(), nil
}

func (v dataPermissionConfigValidator) validateGrant(
	tx *gorm.DB,
	grantId int,
) (response.DataPermissionValidationResultRes, error) {
	grant, err := v.grantRepo.FindByIdWithDB(tx, grantId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		collector := newDataPermissionValidationCollector()
		collector.add(
			diagnosticGrantNotFound,
			"数据权限授权不存在",
			dataPermissionObjectGrant,
			grantId,
		)
		return collector.result(), nil
	}
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}
	return v.validateGrantRecord(tx, grant, false)
}

func (v dataPermissionConfigValidator) validateGrantRecord(
	tx *gorm.DB,
	grant model.DataGrant,
	allowInactivePolicy bool,
) (response.DataPermissionValidationResultRes, error) {
	collector := newDataPermissionValidationCollector()
	if _, ok := dataGrantSubjectTypeSet[grant.SubjectType]; !ok {
		collector.add(
			diagnosticGrantSubjectType,
			"授权主体类型不合法",
			dataPermissionObjectGrant,
			grant.Id,
		)
	} else {
		exists, err := v.grantSubjectExists(tx, grant.SubjectType, grant.SubjectId)
		if err != nil {
			return response.DataPermissionValidationResultRes{}, err
		}
		if !exists {
			collector.add(
				diagnosticGrantSubjectNotFound,
				"授权主体不存在或已停用",
				dataPermissionObjectSubject,
				grant.SubjectId,
			)
		}
	}
	if err := validateDataGrantValidity(grant.ValidFrom, grant.ValidTo); err != nil {
		collector.add(
			diagnosticGrantValidity,
			"授权有效期不合法",
			dataPermissionObjectGrant,
			grant.Id,
		)
	}

	resource, err := v.resourceRepo.FindByIdWithDB(tx, grant.ResourceId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		collector.add(
			diagnosticResourceNotFound,
			"授权引用的数据资源不存在",
			dataPermissionObjectGrant,
			grant.Id,
		)
		return collector.result(), nil
	}
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}
	if !resource.State {
		collector.add(
			diagnosticResourceInactive,
			"授权引用的数据资源已停用",
			dataPermissionObjectResource,
			resource.Id,
		)
	}

	operation, err := v.operationRepo.FindByStableKeyForConfigDB(
		tx,
		resource.Id,
		grant.Operation,
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		collector.add(
			diagnosticOperationNotFound,
			"授权引用的资源操作不存在",
			dataPermissionObjectGrant,
			grant.Id,
		)
	} else if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	} else if !operation.State {
		collector.add(
			diagnosticOperationInactive,
			"授权引用的资源操作已停用",
			dataPermissionObjectOperation,
			operation.Id,
		)
	}

	policy, err := v.policyRepo.FindByIdWithDB(tx, grant.PolicyId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		collector.add(
			diagnosticPolicyNotFound,
			"授权引用的数据权限策略不存在",
			dataPermissionObjectGrant,
			grant.Id,
		)
		return collector.result(), nil
	}
	if err != nil {
		return response.DataPermissionValidationResultRes{}, myerrors.WrapDatabaseError(err)
	}
	if !policy.State && !allowInactivePolicy {
		collector.add(
			diagnosticPolicyInactive,
			"授权引用的数据权限策略已停用",
			dataPermissionObjectPolicy,
			policy.Id,
		)
	}
	rules, err := v.validatePolicyRules(tx, policy, collector)
	if err != nil {
		return response.DataPermissionValidationResultRes{}, err
	}
	if err = v.validateRulesForResource(tx, resource, grant.Operation, rules, collector); err != nil {
		return response.DataPermissionValidationResultRes{}, err
	}
	return collector.result(), nil
}

func (v dataPermissionConfigValidator) validatePolicyRules(
	tx *gorm.DB,
	policy model.DataPolicy,
	collector *dataPermissionValidationCollector,
) ([]model.DataPolicyRule, error) {
	rules, err := v.ruleRepo.ListByPolicyForConfigDB(tx, policy.Id)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	activeRules := make([]model.DataPolicyRule, 0, len(rules))
	for _, rule := range rules {
		if rule.State {
			activeRules = append(activeRules, rule)
		}
	}
	switch policy.PolicyType {
	case model.DataPolicyTypeAll, model.DataPolicyTypeNone:
		if len(activeRules) != 0 {
			collector.add(
				diagnosticPolicyRuleCount,
				"全部或无权限策略不能包含启用规则",
				dataPermissionObjectPolicy,
				policy.Id,
			)
		}
	case model.DataPolicyTypeRuleSet:
		if len(activeRules) == 0 {
			collector.add(
				diagnosticPolicyRuleRequired,
				"规则策略至少需要一条启用规则",
				dataPermissionObjectPolicy,
				policy.Id,
			)
		}
		if len(activeRules) > maxDataPolicyRules {
			collector.add(
				diagnosticPolicyRuleCount,
				"数据权限策略规则数量超过限制",
				dataPermissionObjectPolicy,
				policy.Id,
			)
		}
	default:
		collector.add(
			diagnosticPolicyTypeInvalid,
			"数据权限策略类型不合法",
			dataPermissionObjectPolicy,
			policy.Id,
		)
	}

	for _, rule := range activeRules {
		if err = v.validateRuleDeclaration(tx, rule, collector); err != nil {
			return nil, err
		}
	}
	return activeRules, nil
}

func (v dataPermissionConfigValidator) validateRuleDeclaration(
	tx *gorm.DB,
	rule model.DataPolicyRule,
	collector *dataPermissionValidationCollector,
) error {
	if rule.Sequence <= 0 || !dataPolicyCodePattern.MatchString(rule.OwnershipCode) {
		collector.add(
			diagnosticRuleIdentity,
			"策略规则身份字段不合法",
			dataPermissionObjectRule,
			rule.Id,
		)
	}
	if _, ok := dataPolicyScopeSourceSet[rule.ScopeSource]; !ok {
		collector.add(
			diagnosticScopeSourceInvalid,
			"策略规则范围来源不合法",
			dataPermissionObjectRule,
			rule.Id,
		)
	}
	if _, ok := dataPolicyRelationSet[rule.Relation]; !ok {
		collector.add(
			diagnosticRelationInvalid,
			"策略规则关系类型不合法",
			dataPermissionObjectRule,
			rule.Id,
		)
	}
	if _, ok := dataPolicyOperatorSet[rule.Operator]; !ok {
		collector.add(
			diagnosticOperatorInvalid,
			"策略规则操作符不合法",
			dataPermissionObjectRule,
			rule.Id,
		)
	}

	dimension, err := v.dimensionRepo.FindByIdWithDB(tx, rule.DimensionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		collector.add(
			diagnosticDimensionNotFound,
			"策略规则引用的维度不存在",
			dataPermissionObjectDimension,
			rule.DimensionId,
		)
		return nil
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if !dimension.State {
		collector.add(
			diagnosticDimensionInactive,
			"策略规则引用的维度已停用",
			dataPermissionObjectDimension,
			dimension.Id,
		)
	}
	if compatibilityErr := validateRuleScopeCompatibility(rule, dimension); compatibilityErr != nil {
		switch {
		case errors.Is(compatibilityErr, myerrors.ErrDataPolicyRuleRelationInvalid):
			collector.add(
				diagnosticRelationInvalid,
				"策略规则关系类型与维度不兼容",
				dataPermissionObjectRule,
				rule.Id,
			)
		case errors.Is(compatibilityErr, myerrors.ErrDataPolicyRuleOperatorInvalid):
			collector.add(
				diagnosticOperatorInvalid,
				"策略规则操作符与范围来源不兼容",
				dataPermissionObjectRule,
				rule.Id,
			)
		default:
			collector.add(
				diagnosticProviderCapabilityInvalid,
				"策略规则范围来源不受当前维度Provider声明支持",
				dataPermissionObjectRule,
				rule.Id,
			)
		}
	}
	if _, normalizeErr := normalizePolicySpecifiedValues(
		json.RawMessage(rule.SpecifiedValues),
		rule.ScopeSource,
		rule.Operator,
		dimension.ValueType,
	); normalizeErr != nil {
		collector.add(
			diagnosticSpecifiedValuesInvalid,
			"策略规则指定值格式或类型不合法",
			dataPermissionObjectRule,
			rule.Id,
		)
	}

	codeCount, err := v.ownershipRepo.CountByIdentityForConfig(
		tx,
		rule.OwnershipCode,
		nil,
		true,
	)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if codeCount == 0 {
		collector.add(
			diagnosticOwnershipNotFound,
			"策略规则未匹配到有效归属定义",
			dataPermissionObjectRule,
			rule.Id,
		)
		return nil
	}
	exactCount, err := v.ownershipRepo.CountByIdentityForConfig(
		tx,
		rule.OwnershipCode,
		&rule.DimensionId,
		true,
	)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if exactCount == 0 {
		collector.add(
			diagnosticOwnershipDimension,
			"策略规则归属编码与维度不匹配",
			dataPermissionObjectRule,
			rule.Id,
		)
	}
	return nil
}

func (v dataPermissionConfigValidator) validateRulesForResource(
	tx *gorm.DB,
	resource model.DataResource,
	operation string,
	rules []model.DataPolicyRule,
	collector *dataPermissionValidationCollector,
) error {
	for _, rule := range rules {
		ownership, err := v.ownershipRepo.FindByStableKeyForConfigDB(
			tx,
			resource.Id,
			rule.OwnershipCode,
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			collector.add(
				diagnosticOwnershipNotFound,
				"目标资源缺少策略规则要求的归属定义",
				dataPermissionObjectRule,
				rule.Id,
			)
			continue
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ownership.State {
			collector.add(
				diagnosticOwnershipInactive,
				"策略规则使用了已停用的归属定义",
				dataPermissionObjectOwnership,
				ownership.Id,
			)
		}
		if ownership.DimensionId != rule.DimensionId {
			collector.add(
				diagnosticOwnershipDimension,
				"目标资源归属定义与策略规则维度不一致",
				dataPermissionObjectOwnership,
				ownership.Id,
			)
			continue
		}
		dimension, err := v.dimensionRepo.FindByIdWithDB(tx, rule.DimensionId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return myerrors.WrapDatabaseError(err)
		}
		if ownership.ValueType != dimension.ValueType {
			collector.add(
				diagnosticOwnershipDimension,
				"目标资源归属值类型与维度不一致",
				dataPermissionObjectOwnership,
				ownership.Id,
			)
		}
		switch ownership.BindingType {
		case model.DataOwnershipBindingTypeMetadataField:
			if ownership.TableFieldId == nil {
				collector.add(
					diagnosticOwnershipBinding,
					"元数据归属定义缺少字段引用",
					dataPermissionObjectOwnership,
					ownership.Id,
				)
			}
		case model.DataOwnershipBindingTypeRegisteredField:
			if ownership.AdapterFieldCode == nil || v.registeredFieldChecker == nil {
				collector.add(
					diagnosticOwnershipBinding,
					"注册归属定义缺少服务端注册能力",
					dataPermissionObjectOwnership,
					ownership.Id,
				)
				continue
			}
			if err = v.registeredFieldChecker.ValidateOperation(
				datapermission.OwnershipFieldOperationValidation{
					ResourceCode:  resource.ResourceCode,
					OwnershipCode: ownership.OwnershipCode,
					Operation:     operation,
				},
			); err != nil {
				collector.add(
					diagnosticOwnershipBinding,
					"注册归属定义不支持当前资源操作",
					dataPermissionObjectOwnership,
					ownership.Id,
				)
			}
		default:
			collector.add(
				diagnosticOwnershipBinding,
				"归属定义绑定类型不合法",
				dataPermissionObjectOwnership,
				ownership.Id,
			)
		}
	}
	return nil
}

func (v dataPermissionConfigValidator) grantSubjectExists(
	tx *gorm.DB,
	subjectType string,
	subjectId int,
) (bool, error) {
	var (
		exists bool
		err    error
	)
	switch subjectType {
	case model.DataGrantSubjectTypeRole:
		exists, err = v.grantRepo.RoleExistsForConfig(tx, subjectId)
	case model.DataGrantSubjectTypeUser:
		exists, err = v.grantRepo.UserExistsForConfig(tx, subjectId)
	default:
		return false, nil
	}
	if err != nil {
		return false, myerrors.WrapDatabaseError(err)
	}
	return exists, nil
}

func newDataPermissionValidationCollector() *dataPermissionValidationCollector {
	return &dataPermissionValidationCollector{
		errors: make([]response.DataPermissionValidationErrorRes, 0),
		seen:   make(map[string]struct{}),
	}
}

func (c *dataPermissionValidationCollector) add(
	code string,
	message string,
	objectType string,
	objectId int,
) {
	key := code + ":" + objectType + ":" + strconv.Itoa(objectId)
	if _, exists := c.seen[key]; exists {
		return
	}
	c.seen[key] = struct{}{}
	c.errors = append(c.errors, response.DataPermissionValidationErrorRes{
		Code:       code,
		Message:    message,
		ObjectType: objectType,
		ObjectId:   objectId,
	})
}

func (c *dataPermissionValidationCollector) merge(
	result response.DataPermissionValidationResultRes,
) {
	for _, validationErr := range result.Errors {
		c.add(
			validationErr.Code,
			validationErr.Message,
			validationErr.ObjectType,
			validationErr.ObjectId,
		)
	}
}

func (c *dataPermissionValidationCollector) result() response.DataPermissionValidationResultRes {
	errorsCopy := append([]response.DataPermissionValidationErrorRes(nil), c.errors...)
	return response.DataPermissionValidationResultRes{
		Valid:  len(errorsCopy) == 0,
		Errors: errorsCopy,
	}
}

func validDataPermissionValidationResult() response.DataPermissionValidationResultRes {
	return response.DataPermissionValidationResultRes{
		Valid:  true,
		Errors: []response.DataPermissionValidationErrorRes{},
	}
}

func normalizePreflightIds(ids []int) ([]int, error) {
	if len(ids) == 0 || len(ids) > maxDataPermissionBatchEnable {
		return nil, myerrors.NewParameterError("批量启用数量必须在1到100之间")
	}
	unique := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, myerrors.NewParameterError("resource_id必须大于0")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, nil
}

func dataPermissionDiagnosticAsGrantError(
	result response.DataPermissionValidationResultRes,
) error {
	if result.Valid || len(result.Errors) == 0 {
		return nil
	}
	switch result.Errors[0].Code {
	case diagnosticGrantSubjectType:
		return myerrors.ErrDataGrantSubjectTypeInvalid
	case diagnosticGrantSubjectNotFound:
		return myerrors.ErrDataGrantSubjectNotFound
	case diagnosticResourceNotFound:
		return myerrors.ErrDataResourceNotFound
	case diagnosticResourceInactive:
		return myerrors.ErrDataResourceStateInvalid
	case diagnosticOperationNotFound:
		return myerrors.ErrDataResourceOperationNotFound
	case diagnosticOperationInactive:
		return myerrors.ErrDataResourceOperationInvalid
	case diagnosticPolicyNotFound:
		return myerrors.ErrDataPolicyNotFound
	case diagnosticPolicyInactive, diagnosticPolicyTypeInvalid:
		return myerrors.ErrDataGrantPolicyInvalid
	case diagnosticOwnershipNotFound,
		diagnosticOwnershipInactive,
		diagnosticOwnershipDimension,
		diagnosticOwnershipBinding:
		return myerrors.ErrDataGrantOwnershipMismatch
	case diagnosticGrantValidity:
		return myerrors.ErrDataGrantValidityInvalid
	default:
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}
}
