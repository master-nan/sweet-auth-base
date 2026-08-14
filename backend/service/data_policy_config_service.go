package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	dataPolicyAuditType         = "data_policy"
	dataPolicyRuleAuditType     = "data_policy_rule"
	dataPolicyCreateAction      = "create"
	dataPolicyUpdateAction      = "update"
	dataPolicyDisableAction     = "disable"
	dataPolicyRuleAddAction     = "add_rule"
	dataPolicyRuleReplaceAction = "replace_rules"
	dataPolicyRuleDisableAction = "disable_rule"

	maxDataPolicyRules           = 8
	maxDataPolicySpecifiedValues = 5000
)

var dataPolicyCodePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

var dataPolicyScopeSourceSet = map[string]struct{}{
	model.DataPolicyScopeSourceEffectiveLegalEntities: {},
	model.DataPolicyScopeSourceEffectiveOrgUnits:      {},
	model.DataPolicyScopeSourceCurrentEmployee:        {},
	model.DataPolicyScopeSourceSpecifiedValues:        {},
}

var dataPolicyRelationSet = map[string]struct{}{
	model.DataPolicyRelationExact:              {},
	model.DataPolicyRelationSelfAndDescendants: {},
}

var dataPolicyOperatorSet = map[string]struct{}{
	model.DataPolicyOperatorEqual: {},
	model.DataPolicyOperatorIn:    {},
}

// DataPolicyConfigService 负责可复用 Policy 和结构化 Rule 配置。
// Policy 不保存 resource_id，因此资源级 Ownership 匹配仍由 Grant 预检负责。
type DataPolicyConfigService struct {
	policyRepo    repository.DataPolicyRepository
	ruleRepo      repository.DataPolicyRuleRepository
	dimensionRepo repository.DataDimensionDefinitionRepository
	ownershipRepo repository.DataOwnershipFieldRepository
	sf            *utils.Snowflake
	auditWriter   TransactionalAuditWriter
}

func NewDataPolicyConfigService(
	policyRepo repository.DataPolicyRepository,
	ruleRepo repository.DataPolicyRuleRepository,
	dimensionRepo repository.DataDimensionDefinitionRepository,
	ownershipRepo repository.DataOwnershipFieldRepository,
	sf *utils.Snowflake,
	auditWriter TransactionalAuditWriter,
) *DataPolicyConfigService {
	return &DataPolicyConfigService{
		policyRepo:    policyRepo,
		ruleRepo:      ruleRepo,
		dimensionRepo: dimensionRepo,
		ownershipRepo: ownershipRepo,
		sf:            sf,
		auditWriter:   auditWriter,
	}
}

func (s *DataPolicyConfigService) CreatePolicy(
	ctx context.Context,
	req request.DataPolicyCreateReq,
) (response.DataPolicyDetailRes, error) {
	if ctx == nil {
		return response.DataPolicyDetailRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	policy := model.DataPolicy{
		Basic: model.Basic{
			State: req.State == nil || *req.State,
		},
		Code:        strings.TrimSpace(req.PolicyCode),
		Name:        strings.TrimSpace(req.Name),
		PolicyType:  model.DataPolicyTypeRuleSet,
		Description: strings.TrimSpace(req.Description),
	}
	if err := validateDataPolicy(policy); err != nil {
		return response.DataPolicyDetailRes{}, err
	}
	if len(req.Rules) > maxDataPolicyRules {
		return response.DataPolicyDetailRes{}, myerrors.ErrDataPolicyRuleCountInvalid
	}

	createdRules := make([]model.DataPolicyRule, 0, len(req.Rules))
	err := RunInTransaction(ctx, s.policyRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		if _, err := s.policyRepo.FindByFieldWithDB(tx, "code", policy.Code); err == nil {
			return myerrors.ErrDataPolicyCodeDuplicate
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.WrapDatabaseError(err)
		}

		id, err := s.generateId()
		if err != nil {
			return err
		}
		policy.Id = id
		if err = s.policyRepo.Create(tx, &policy); err != nil {
			if isDataPermissionConfigDuplicate(err) {
				return myerrors.ErrDataPolicyCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		if !policy.State {
			if _, err = s.policyRepo.UpdateFields(tx, policy.Id, map[string]any{"state": false}); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}

		createdRules, err = s.createRules(tx, policy, req.Rules)
		if err != nil {
			return err
		}
		return s.recordPolicyAudit(
			ctx,
			tx,
			dataPolicyCreateAction,
			policy,
			map[string]TransactionalAuditChange{
				"policy_code": {OldValue: nil, NewValue: policy.Code},
				"policy_type": {OldValue: nil, NewValue: policy.PolicyType},
				"rule_count":  {OldValue: nil, NewValue: len(createdRules)},
				"state":       {OldValue: nil, NewValue: policy.State},
			},
		)
	})
	if err != nil {
		return response.DataPolicyDetailRes{}, err
	}

	result := response.NewDataPolicyDetailRes(policy)
	result.Rules = dataPolicyRuleDetailResponses(createdRules)
	return result, nil
}

func (s *DataPolicyConfigService) GetPolicy(
	ctx context.Context,
	policyId int,
) (response.DataPolicyDetailRes, error) {
	if policyId <= 0 {
		return response.DataPolicyDetailRes{}, myerrors.NewParameterError("policy_id必须大于0")
	}
	policy, err := s.policyRepo.WithContext(ctx).FindById(policyId)
	if err != nil {
		return response.DataPolicyDetailRes{}, mapDataPolicyReadError(err)
	}
	rules, err := s.ruleRepo.ListByPolicyForConfigDB(s.ruleRepo.DBWithContext(ctx), policy.Id)
	if err != nil {
		return response.DataPolicyDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	result := response.NewDataPolicyDetailRes(policy)
	result.Rules = dataPolicyRuleDetailResponses(rules)
	return result, nil
}

func (s *DataPolicyConfigService) PagePolicies(
	ctx context.Context,
	req request.DataPolicyQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataPolicyListRes], error) {
	var result response.ListResult[response.DataPolicyListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	basic := req.ToBasic()
	rows, err := s.policyRepo.GetDataPolicyList(ctx, &basic, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.DataPolicyListRes, 0, len(rows.Data))
	for _, policy := range rows.Data {
		result.Data = append(result.Data, response.NewDataPolicyListRes(policy))
	}
	return result, nil
}

func (s *DataPolicyConfigService) UpdatePolicy(
	ctx context.Context,
	req request.DataPolicyUpdateReq,
) (response.DataPolicyDetailRes, error) {
	if ctx == nil {
		return response.DataPolicyDetailRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if req.Id <= 0 {
		return response.DataPolicyDetailRes{}, myerrors.NewParameterError("policy_id必须大于0")
	}

	err := RunInTransaction(ctx, s.policyRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.findPolicyForUpdate(tx, req.Id)
		if err != nil {
			return err
		}
		if req.PolicyCode != nil && strings.TrimSpace(*req.PolicyCode) != current.Code {
			return myerrors.ErrDataPolicyFieldImmutable
		}

		fields := make(map[string]any)
		changes := make(map[string]TransactionalAuditChange)
		action := dataPolicyUpdateAction
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return myerrors.ErrDataPolicyNameRequired
			}
			if name != current.Name {
				fields["name"] = name
				changes["name"] = TransactionalAuditChange{OldValue: current.Name, NewValue: name}
			}
		}
		if req.Description != nil {
			description := strings.TrimSpace(*req.Description)
			if description != current.Description {
				fields["description"] = description
				changes["description"] = TransactionalAuditChange{
					OldValue: current.Description,
					NewValue: description,
				}
			}
		}
		if req.State != nil && *req.State != current.State {
			if *req.State {
				return myerrors.ErrDataPermissionPreflightFailed
			}
			fields["state"] = *req.State
			changes["state"] = TransactionalAuditChange{OldValue: current.State, NewValue: *req.State}
			if !*req.State {
				action = dataPolicyDisableAction
			}
		}
		if len(fields) == 0 {
			return nil
		}
		changed, err := s.policyRepo.UpdateFields(tx, current.Id, fields)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !changed {
			return myerrors.ErrDataPolicyNotFound
		}
		return s.recordPolicyAudit(ctx, tx, action, current, changes)
	})
	if err != nil {
		return response.DataPolicyDetailRes{}, err
	}
	return s.GetPolicy(ctx, req.Id)
}

func (s *DataPolicyConfigService) AddPolicyRule(
	ctx context.Context,
	req request.DataPolicyRuleCreateReq,
) (response.DataPolicyRuleDetailRes, error) {
	if ctx == nil {
		return response.DataPolicyRuleDetailRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if req.PolicyId <= 0 {
		return response.DataPolicyRuleDetailRes{}, myerrors.NewParameterError("policy_id必须大于0")
	}

	var created model.DataPolicyRule
	err := RunInTransaction(ctx, s.ruleRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		policy, err := s.findRuleSetPolicy(tx, req.PolicyId)
		if err != nil {
			return err
		}
		if _, err = s.ruleRepo.FindByStableKeyForConfigDB(
			tx,
			policy.Id,
			req.Sequence,
		); err == nil {
			return myerrors.ErrDataPolicyRuleDuplicate
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.WrapDatabaseError(err)
		}
		created, err = s.newValidatedRule(tx, policy.Id, req.DataPolicyRuleCreateItemReq)
		if err != nil {
			return err
		}
		if err = s.createRule(tx, &created); err != nil {
			return err
		}
		return s.recordRuleAudit(
			ctx,
			tx,
			dataPolicyRuleAddAction,
			created,
			map[string]TransactionalAuditChange{
				"policy_id":      {OldValue: nil, NewValue: created.PolicyId},
				"sequence":       {OldValue: nil, NewValue: created.Sequence},
				"ownership_code": {OldValue: nil, NewValue: created.OwnershipCode},
				"dimension_id":   {OldValue: nil, NewValue: created.DimensionId},
			},
		)
	})
	if err != nil {
		return response.DataPolicyRuleDetailRes{}, err
	}
	return s.policyRuleDetail(ctx, created)
}

func (s *DataPolicyConfigService) ReplacePolicyRules(
	ctx context.Context,
	req request.DataPolicyRuleBatchReq,
) ([]response.DataPolicyRuleListRes, error) {
	if ctx == nil {
		return nil, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if req.PolicyId <= 0 {
		return nil, myerrors.NewParameterError("policy_id必须大于0")
	}
	if len(req.Items) == 0 || len(req.Items) > maxDataPolicyRules {
		return nil, myerrors.ErrDataPolicyRuleCountInvalid
	}

	created := make([]model.DataPolicyRule, 0, len(req.Items))
	err := RunInTransaction(ctx, s.ruleRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		policy, err := s.findRuleSetPolicy(tx, req.PolicyId)
		if err != nil {
			return err
		}
		seenSequences := make(map[int]struct{}, len(req.Items))
		for _, item := range req.Items {
			if _, exists := seenSequences[item.Sequence]; exists {
				return myerrors.ErrDataPolicyRuleDuplicate
			}
			seenSequences[item.Sequence] = struct{}{}
			rule, err := s.newValidatedRule(tx, policy.Id, item)
			if err != nil {
				return err
			}
			created = append(created, rule)
		}

		existing, err := s.ruleRepo.ListByPolicyForConfigDB(tx, policy.Id)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		for _, rule := range existing {
			if err = s.ruleRepo.DeleteById(tx, rule.Id); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
		for i := range created {
			if err = s.createRule(tx, &created[i]); err != nil {
				return err
			}
		}
		return s.recordPolicyAudit(
			ctx,
			tx,
			dataPolicyRuleReplaceAction,
			policy,
			map[string]TransactionalAuditChange{
				"rule_count": {OldValue: len(existing), NewValue: len(created)},
			},
		)
	})
	if err != nil {
		return nil, err
	}
	return dataPolicyRuleListResponses(created), nil
}

func (s *DataPolicyConfigService) PagePolicyRules(
	ctx context.Context,
	req request.DataPolicyRuleQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataPolicyRuleListRes], error) {
	var result response.ListResult[response.DataPolicyRuleListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	basic := req.ToBasic()
	rows, err := s.ruleRepo.GetDataPolicyRuleList(ctx, &basic, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = dataPolicyRuleListResponses(rows.Data)
	return result, nil
}

func (s *DataPolicyConfigService) DisablePolicyRule(ctx context.Context, ruleId int) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if ruleId <= 0 {
		return myerrors.NewParameterError("rule_id必须大于0")
	}
	return RunInTransaction(ctx, s.ruleRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		rule, err := s.ruleRepo.FindByIdWithDB(tx, ruleId)
		if err != nil {
			return mapDataPolicyRuleReadError(err)
		}
		if !rule.State {
			return nil
		}
		changed, err := s.ruleRepo.UpdateFields(tx, rule.Id, map[string]any{"state": false})
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !changed {
			return myerrors.ErrDataPolicyRuleNotFound
		}
		return s.recordRuleAudit(
			ctx,
			tx,
			dataPolicyRuleDisableAction,
			rule,
			map[string]TransactionalAuditChange{
				"state": {OldValue: true, NewValue: false},
			},
		)
	})
}

func (s *DataPolicyConfigService) createRules(
	tx *gorm.DB,
	policy model.DataPolicy,
	items []request.DataPolicyRuleCreateItemReq,
) ([]model.DataPolicyRule, error) {
	if len(items) == 0 {
		return []model.DataPolicyRule{}, nil
	}
	seenSequences := make(map[int]struct{}, len(items))
	rules := make([]model.DataPolicyRule, 0, len(items))
	for _, item := range items {
		if _, exists := seenSequences[item.Sequence]; exists {
			return nil, myerrors.ErrDataPolicyRuleDuplicate
		}
		seenSequences[item.Sequence] = struct{}{}
		rule, err := s.newValidatedRule(tx, policy.Id, item)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	for i := range rules {
		if err := s.createRule(tx, &rules[i]); err != nil {
			return nil, err
		}
	}
	return rules, nil
}

func (s *DataPolicyConfigService) createRule(tx *gorm.DB, rule *model.DataPolicyRule) error {
	id, err := s.generateId()
	if err != nil {
		return err
	}
	rule.Id = id
	if err = s.ruleRepo.Create(tx, rule); err != nil {
		if isDataPermissionConfigDuplicate(err) {
			return myerrors.ErrDataPolicyRuleDuplicate
		}
		return myerrors.WrapDatabaseError(err)
	}
	if !rule.State {
		if _, err = s.ruleRepo.UpdateFields(tx, rule.Id, map[string]any{"state": false}); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
	}
	return nil
}

func (s *DataPolicyConfigService) newValidatedRule(
	tx *gorm.DB,
	policyId int,
	item request.DataPolicyRuleCreateItemReq,
) (model.DataPolicyRule, error) {
	rule := model.DataPolicyRule{
		Basic: model.Basic{
			State: item.State == nil || *item.State,
		},
		PolicyId:      policyId,
		Sequence:      item.Sequence,
		DimensionId:   item.DimensionId,
		OwnershipCode: strings.TrimSpace(item.OwnershipCode),
		ScopeSource:   strings.TrimSpace(item.ScopeSource),
		Relation:      strings.TrimSpace(item.Relation),
		Operator:      strings.TrimSpace(item.Operator),
		StructureCode: normalizeOptionalPolicyString(item.StructureCode),
		Description:   strings.TrimSpace(item.Description),
	}
	if rule.Sequence <= 0 || rule.DimensionId <= 0 || !dataPolicyCodePattern.MatchString(rule.OwnershipCode) {
		return model.DataPolicyRule{}, myerrors.ErrDataPolicyRuleOwnershipNotFound
	}
	if _, ok := dataPolicyScopeSourceSet[rule.ScopeSource]; !ok {
		return model.DataPolicyRule{}, myerrors.ErrDataPolicyRuleScopeSourceInvalid
	}
	if _, ok := dataPolicyRelationSet[rule.Relation]; !ok {
		return model.DataPolicyRule{}, myerrors.ErrDataPolicyRuleRelationInvalid
	}
	if _, ok := dataPolicyOperatorSet[rule.Operator]; !ok {
		return model.DataPolicyRule{}, myerrors.ErrDataPolicyRuleOperatorInvalid
	}

	dimension, err := s.findActivePolicyDimension(tx, rule.DimensionId)
	if err != nil {
		return model.DataPolicyRule{}, err
	}
	if err = s.validateRuleOwnershipIdentity(tx, rule); err != nil {
		return model.DataPolicyRule{}, err
	}
	if err = validateRuleScopeCompatibility(rule, dimension); err != nil {
		return model.DataPolicyRule{}, err
	}
	rule.SpecifiedValues, err = normalizePolicySpecifiedValues(
		item.SpecifiedValues,
		rule.ScopeSource,
		rule.Operator,
		dimension.ValueType,
	)
	if err != nil {
		return model.DataPolicyRule{}, err
	}
	return rule, nil
}

func (s *DataPolicyConfigService) validateRuleOwnershipIdentity(
	tx *gorm.DB,
	rule model.DataPolicyRule,
) error {
	codeCount, err := s.ownershipRepo.CountByIdentityForConfig(
		tx,
		rule.OwnershipCode,
		nil,
		true,
	)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if codeCount == 0 {
		return myerrors.ErrDataPolicyRuleOwnershipNotFound
	}
	exactCount, err := s.ownershipRepo.CountByIdentityForConfig(
		tx,
		rule.OwnershipCode,
		&rule.DimensionId,
		true,
	)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if exactCount == 0 {
		return myerrors.ErrDataPolicyRuleDimensionMismatch
	}
	return nil
}

func validateRuleScopeCompatibility(
	rule model.DataPolicyRule,
	dimension model.DataDimensionDefinition,
) error {
	if rule.Relation == model.DataPolicyRelationSelfAndDescendants {
		if dimension.Code != "management_org" || rule.StructureCode == nil {
			return myerrors.ErrDataPolicyRuleRelationInvalid
		}
	} else if rule.StructureCode != nil {
		return myerrors.ErrDataPolicyRuleRelationInvalid
	}

	switch dimension.Code {
	case "legal_entity":
		if dimension.ProviderCode != "organization" ||
			(rule.ScopeSource != model.DataPolicyScopeSourceEffectiveLegalEntities &&
				rule.ScopeSource != model.DataPolicyScopeSourceSpecifiedValues) {
			return myerrors.ErrDataPolicyRuleScopeSourceInvalid
		}
	case "management_org":
		if dimension.ProviderCode != "organization" ||
			(rule.ScopeSource != model.DataPolicyScopeSourceEffectiveOrgUnits &&
				rule.ScopeSource != model.DataPolicyScopeSourceSpecifiedValues) {
			return myerrors.ErrDataPolicyRuleScopeSourceInvalid
		}
	case "employee":
		if dimension.ProviderCode != "organization" ||
			(rule.ScopeSource != model.DataPolicyScopeSourceCurrentEmployee &&
				rule.ScopeSource != model.DataPolicyScopeSourceSpecifiedValues) {
			return myerrors.ErrDataPolicyRuleScopeSourceInvalid
		}
	default:
		if rule.ScopeSource != model.DataPolicyScopeSourceSpecifiedValues {
			return myerrors.ErrDataPolicyRuleScopeSourceInvalid
		}
	}

	switch rule.ScopeSource {
	case model.DataPolicyScopeSourceCurrentEmployee:
		if rule.Operator != model.DataPolicyOperatorEqual {
			return myerrors.ErrDataPolicyRuleOperatorInvalid
		}
	case model.DataPolicyScopeSourceEffectiveLegalEntities,
		model.DataPolicyScopeSourceEffectiveOrgUnits:
		if rule.Operator != model.DataPolicyOperatorIn {
			return myerrors.ErrDataPolicyRuleOperatorInvalid
		}
	}
	return nil
}

func normalizePolicySpecifiedValues(
	raw json.RawMessage,
	scopeSource string,
	operator string,
	valueType string,
) (datatypes.JSON, error) {
	trimmed := bytes.TrimSpace(raw)
	if scopeSource != model.DataPolicyScopeSourceSpecifiedValues {
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			return nil, nil
		}
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}
	if err := ensurePolicyJSONEOF(decoder); err != nil {
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}
	values, ok := decoded.([]any)
	if !ok || len(values) > maxDataPolicySpecifiedValues {
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}
	if operator == model.DataPolicyOperatorEqual && len(values) != 1 {
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}

	normalizedValues, err := normalizePolicyValues(values, valueType)
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(normalizedValues)
	if err != nil {
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}
	return datatypes.JSON(normalized), nil
}

func normalizePolicyValues(values []any, valueType string) (any, error) {
	switch valueType {
	case model.DataDimensionValueTypeBigint:
		unique := make(map[int64]struct{}, len(values))
		for _, value := range values {
			number, ok := value.(json.Number)
			if !ok {
				return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
			}
			parsed, err := strconv.ParseInt(number.String(), 10, 64)
			if err != nil || parsed <= 0 {
				return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
			}
			unique[parsed] = struct{}{}
		}
		normalized := make([]int64, 0, len(unique))
		for value := range unique {
			normalized = append(normalized, value)
		}
		sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
		return normalized, nil
	case model.DataDimensionValueTypeString:
		unique := make(map[string]struct{}, len(values))
		for _, value := range values {
			text, ok := value.(string)
			if !ok {
				return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
			}
			text = strings.TrimSpace(text)
			if text == "" {
				return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
			}
			unique[text] = struct{}{}
		}
		normalized := make([]string, 0, len(unique))
		for value := range unique {
			normalized = append(normalized, value)
		}
		sort.Strings(normalized)
		return normalized, nil
	default:
		return nil, myerrors.ErrDataPolicyRuleSpecifiedValuesInvalid
	}
}

func ensurePolicyJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values")
	}
	return err
}

func validateDataPolicy(policy model.DataPolicy) error {
	if !dataPolicyCodePattern.MatchString(policy.Code) {
		return myerrors.ErrDataPolicyCodeInvalid
	}
	if policy.Name == "" {
		return myerrors.ErrDataPolicyNameRequired
	}
	return nil
}

func (s *DataPolicyConfigService) findPolicyForUpdate(
	tx *gorm.DB,
	policyId int,
) (model.DataPolicy, error) {
	policy, err := s.policyRepo.FindByIdWithDB(tx, policyId)
	if err != nil {
		return model.DataPolicy{}, mapDataPolicyReadError(err)
	}
	return policy, nil
}

func (s *DataPolicyConfigService) findRuleSetPolicy(
	tx *gorm.DB,
	policyId int,
) (model.DataPolicy, error) {
	policy, err := s.findPolicyForUpdate(tx, policyId)
	if err != nil {
		return model.DataPolicy{}, err
	}
	if policy.PolicyType != model.DataPolicyTypeRuleSet {
		return model.DataPolicy{}, myerrors.ErrDataPolicyStateInvalid
	}
	return policy, nil
}

func (s *DataPolicyConfigService) findActivePolicyDimension(
	tx *gorm.DB,
	dimensionId int,
) (model.DataDimensionDefinition, error) {
	dimension, err := s.dimensionRepo.FindByIdWithDB(tx, dimensionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.DataDimensionDefinition{}, myerrors.ErrDataDimensionNotFound
	}
	if err != nil {
		return model.DataDimensionDefinition{}, myerrors.WrapDatabaseError(err)
	}
	if !dimension.State {
		return model.DataDimensionDefinition{}, myerrors.ErrDataDimensionNotFound
	}
	return dimension, nil
}

func (s *DataPolicyConfigService) policyRuleDetail(
	ctx context.Context,
	rule model.DataPolicyRule,
) (response.DataPolicyRuleDetailRes, error) {
	policy, err := s.policyRepo.WithContext(ctx).FindById(rule.PolicyId)
	if err != nil {
		return response.DataPolicyRuleDetailRes{}, mapDataPolicyReadError(err)
	}
	dimension, err := s.dimensionRepo.WithContext(ctx).FindById(rule.DimensionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.DataPolicyRuleDetailRes{}, myerrors.ErrDataDimensionNotFound
	}
	if err != nil {
		return response.DataPolicyRuleDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	result := response.NewDataPolicyRuleDetailRes(rule)
	result.Policy = &response.DataPermissionReferenceSummaryRes{
		Id:   policy.Id,
		Code: policy.Code,
		Name: policy.Name,
	}
	result.Dimension = &response.DataPermissionReferenceSummaryRes{
		Id:   dimension.Id,
		Code: dimension.Code,
		Name: dimension.Name,
	}
	return result, nil
}

func (s *DataPolicyConfigService) generateId() (int, error) {
	if s.sf == nil {
		return 0, myerrors.WrapSystemError(errors.New("data policy id generator is required"))
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, myerrors.WrapSystemError(err)
	}
	return int(id), nil
}

func (s *DataPolicyConfigService) recordPolicyAudit(
	ctx context.Context,
	tx *gorm.DB,
	action string,
	policy model.DataPolicy,
	changes map[string]TransactionalAuditChange,
) error {
	if s.auditWriter == nil {
		return myerrors.WrapSystemError(ErrTransactionalAuditRepositoryRequired)
	}
	if err := s.auditWriter.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: dataPolicyAuditType,
		ResourceCode: policy.Code,
		ResourceId:   strconv.Itoa(policy.Id),
		Changes:      changes,
	}); err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func (s *DataPolicyConfigService) recordRuleAudit(
	ctx context.Context,
	tx *gorm.DB,
	action string,
	rule model.DataPolicyRule,
	changes map[string]TransactionalAuditChange,
) error {
	if s.auditWriter == nil {
		return myerrors.WrapSystemError(ErrTransactionalAuditRepositoryRequired)
	}
	if err := s.auditWriter.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: dataPolicyRuleAuditType,
		ResourceCode: strconv.Itoa(rule.PolicyId) + ":" + strconv.Itoa(rule.Sequence),
		ResourceId:   strconv.Itoa(rule.Id),
		Changes:      changes,
	}); err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func mapDataPolicyReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataPolicyNotFound
	}
	return myerrors.WrapDatabaseError(err)
}

func mapDataPolicyRuleReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataPolicyRuleNotFound
	}
	return myerrors.WrapDatabaseError(err)
}

func dataPolicyRuleListResponses(values []model.DataPolicyRule) []response.DataPolicyRuleListRes {
	result := make([]response.DataPolicyRuleListRes, 0, len(values))
	for _, value := range values {
		result = append(result, response.NewDataPolicyRuleListRes(value))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Sequence == result[j].Sequence {
			return result[i].Id < result[j].Id
		}
		return result[i].Sequence < result[j].Sequence
	})
	return result
}

func dataPolicyRuleDetailResponses(values []model.DataPolicyRule) []response.DataPolicyRuleDetailRes {
	result := make([]response.DataPolicyRuleDetailRes, 0, len(values))
	for _, value := range values {
		result = append(result, response.NewDataPolicyRuleDetailRes(value))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Sequence == result[j].Sequence {
			return result[i].Id < result[j].Id
		}
		return result[i].Sequence < result[j].Sequence
	})
	return result
}

func normalizeOptionalPolicyString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}
