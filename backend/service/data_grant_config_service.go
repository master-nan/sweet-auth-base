package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	dataGrantAuditType      = "data_grant"
	dataGrantCreateAction   = "create"
	dataGrantDisableAction  = "disable"
	dataGrantRestoreAction  = "restore"
	dataGrantRemoveAction   = "remove"
	maxDataGrantBatchCreate = 100
)

var dataGrantSubjectTypeSet = map[string]struct{}{
	model.DataGrantSubjectTypeRole: {},
	model.DataGrantSubjectTypeUser: {},
}

// DataGrantConfigService binds an existing role or user to an existing
// resource operation and policy. It does not resolve scope, call providers or
// create query filters.
type DataGrantConfigService struct {
	grantRepo              repository.DataGrantRepository
	resourceRepo           repository.DataResourceRepository
	operationRepo          repository.DataResourceOperationRepository
	ownershipRepo          repository.DataOwnershipFieldRepository
	dimensionRepo          repository.DataDimensionDefinitionRepository
	policyRepo             repository.DataPolicyRepository
	ruleRepo               repository.DataPolicyRuleRepository
	registeredFieldChecker datapermission.OwnershipFieldOperationValidator
	sf                     *utils.Snowflake
	auditWriter            TransactionalAuditWriter
}

func NewDataGrantConfigService(
	grantRepo repository.DataGrantRepository,
	resourceRepo repository.DataResourceRepository,
	operationRepo repository.DataResourceOperationRepository,
	ownershipRepo repository.DataOwnershipFieldRepository,
	dimensionRepo repository.DataDimensionDefinitionRepository,
	policyRepo repository.DataPolicyRepository,
	ruleRepo repository.DataPolicyRuleRepository,
	registeredFieldChecker datapermission.OwnershipFieldOperationValidator,
	sf *utils.Snowflake,
	auditWriter TransactionalAuditWriter,
) *DataGrantConfigService {
	return &DataGrantConfigService{
		grantRepo:              grantRepo,
		resourceRepo:           resourceRepo,
		operationRepo:          operationRepo,
		ownershipRepo:          ownershipRepo,
		dimensionRepo:          dimensionRepo,
		policyRepo:             policyRepo,
		ruleRepo:               ruleRepo,
		registeredFieldChecker: registeredFieldChecker,
		sf:                     sf,
		auditWriter:            auditWriter,
	}
}

func (s *DataGrantConfigService) CreateGrant(
	ctx *gin.Context,
	req request.DataGrantCreateReq,
) (response.DataGrantDetailRes, error) {
	created, err := s.createGrants(ctx, []request.DataGrantCreateReq{req})
	if err != nil {
		return response.DataGrantDetailRes{}, err
	}
	return s.grantDetail(ctx, created[0])
}

func (s *DataGrantConfigService) CreateGrants(
	ctx *gin.Context,
	req request.DataGrantBatchCreateReq,
) ([]response.DataGrantListRes, error) {
	created, err := s.createGrants(ctx, req.Items)
	if err != nil {
		return nil, err
	}
	return dataGrantListResponses(created), nil
}

func (s *DataGrantConfigService) GetGrant(
	ctx *gin.Context,
	grantId int,
) (response.DataGrantDetailRes, error) {
	if grantId <= 0 {
		return response.DataGrantDetailRes{}, myerrors.NewParameterError("grant_id必须大于0")
	}
	grant, err := s.grantRepo.FindByIdForConfig(ctx, grantId)
	if err != nil {
		return response.DataGrantDetailRes{}, mapDataGrantReadError(err)
	}
	return s.grantDetail(ctx, grant)
}

func (s *DataGrantConfigService) PageGrants(
	ctx *gin.Context,
	req request.DataGrantQueryReq,
	table model.SysTable,
) (response.ListResult[response.DataGrantListRes], error) {
	var result response.ListResult[response.DataGrantListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	rows, err := s.grantRepo.Query(ctx, &req, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = dataGrantListResponses(rows.Data)
	return result, nil
}

func (s *DataGrantConfigService) SetGrantState(
	ctx *gin.Context,
	req request.DataGrantStateReq,
) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if req.Id <= 0 || req.State == nil {
		return myerrors.NewParameterError("grant_id和state不能为空")
	}
	return RunInTransaction(ctx, s.grantRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		grant, err := s.findGrantForUpdate(tx, req.Id)
		if err != nil {
			return err
		}
		if grant.State == *req.State {
			return nil
		}
		if *req.State {
			if err = s.validateGrantContext(tx, grant); err != nil {
				return err
			}
			if err = validateDataGrantValidity(grant.ValidFrom, grant.ValidTo); err != nil {
				return err
			}
		}
		changed, err := s.grantRepo.UpdateFieldsForConfig(
			tx,
			grant.Id,
			map[string]any{"state": *req.State},
		)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !changed {
			return myerrors.ErrDataGrantNotFound
		}
		action := dataGrantDisableAction
		if *req.State {
			action = dataGrantRestoreAction
		}
		return s.recordGrantAudit(
			ctx,
			tx,
			action,
			grant,
			map[string]TransactionalAuditChange{
				"state": {OldValue: grant.State, NewValue: *req.State},
			},
		)
	})
}

func (s *DataGrantConfigService) DisableGrant(ctx *gin.Context, grantId int) error {
	state := false
	return s.SetGrantState(ctx, request.DataGrantStateReq{Id: grantId, State: &state})
}

func (s *DataGrantConfigService) RestoreGrant(ctx *gin.Context, grantId int) error {
	state := true
	return s.SetGrantState(ctx, request.DataGrantStateReq{Id: grantId, State: &state})
}

// RemoveGrant only performs the platform soft delete. The service does not
// expose a physical-delete path, so historical authorization identity remains
// auditable and database references are never cascaded.
func (s *DataGrantConfigService) RemoveGrant(ctx *gin.Context, grantId int) error {
	if ctx == nil {
		return myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if grantId <= 0 {
		return myerrors.NewParameterError("grant_id必须大于0")
	}
	return RunInTransaction(ctx, s.grantRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		grant, err := s.findGrantForUpdate(tx, grantId)
		if err != nil {
			return err
		}
		if err = s.grantRepo.DeleteById(tx, grant.Id); err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		return s.recordGrantAudit(
			ctx,
			tx,
			dataGrantRemoveAction,
			grant,
			map[string]TransactionalAuditChange{
				"deleted": {OldValue: false, NewValue: true},
			},
		)
	})
}

func (s *DataGrantConfigService) createGrants(
	ctx *gin.Context,
	items []request.DataGrantCreateReq,
) ([]model.DataGrant, error) {
	if ctx == nil {
		return nil, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}
	if len(items) == 0 || len(items) > maxDataGrantBatchCreate {
		return nil, myerrors.ErrDataGrantCountInvalid
	}

	normalized := make([]model.DataGrant, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		grant, err := newDataGrantFromRequest(item)
		if err != nil {
			return nil, err
		}
		key := dataGrantStableKey(grant)
		if _, exists := seen[key]; exists {
			return nil, myerrors.ErrDataGrantDuplicate
		}
		seen[key] = struct{}{}
		normalized = append(normalized, grant)
	}

	created := make([]model.DataGrant, 0, len(normalized))
	err := RunInTransaction(ctx, s.grantRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		for i := range normalized {
			grant := normalized[i]
			if err := s.validateGrantContext(tx, grant); err != nil {
				return err
			}
			if err := s.ensureGrantAbsent(tx, grant); err != nil {
				return err
			}
			id, err := s.generateGrantId()
			if err != nil {
				return err
			}
			grant.Id = id
			if err = s.grantRepo.Create(tx, &grant); err != nil {
				if isDataPermissionConfigDuplicate(err) {
					return myerrors.ErrDataGrantDuplicate
				}
				return myerrors.WrapDatabaseError(err)
			}
			if !grant.State {
				if _, err = s.grantRepo.UpdateFieldsForConfig(
					tx,
					grant.Id,
					map[string]any{"state": false},
				); err != nil {
					return myerrors.WrapDatabaseError(err)
				}
			}
			if err = s.recordGrantAudit(
				ctx,
				tx,
				dataGrantCreateAction,
				grant,
				map[string]TransactionalAuditChange{
					"subject_type": {OldValue: nil, NewValue: grant.SubjectType},
					"subject_id":   {OldValue: nil, NewValue: grant.SubjectId},
					"resource_id":  {OldValue: nil, NewValue: grant.ResourceId},
					"operation":    {OldValue: nil, NewValue: grant.Operation},
					"policy_id":    {OldValue: nil, NewValue: grant.PolicyId},
					"state":        {OldValue: nil, NewValue: grant.State},
				},
			); err != nil {
				return err
			}
			created = append(created, grant)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func newDataGrantFromRequest(req request.DataGrantCreateReq) (model.DataGrant, error) {
	grant := model.DataGrant{
		Basic: model.Basic{
			State: req.State == nil || *req.State,
		},
		SubjectType: strings.ToLower(strings.TrimSpace(req.SubjectType)),
		SubjectId:   req.SubjectId,
		ResourceId:  req.ResourceId,
		Operation:   strings.ToLower(strings.TrimSpace(req.Operation)),
		PolicyId:    req.PolicyId,
		ValidFrom:   normalizeDataGrantDate(req.ValidFrom),
		ValidTo:     normalizeDataGrantDate(req.ValidTo),
		Description: strings.TrimSpace(req.Description),
	}
	if _, ok := dataGrantSubjectTypeSet[grant.SubjectType]; !ok {
		return model.DataGrant{}, myerrors.ErrDataGrantSubjectTypeInvalid
	}
	if grant.SubjectId <= 0 || grant.ResourceId <= 0 || grant.PolicyId <= 0 {
		return model.DataGrant{}, myerrors.NewParameterError(
			"subject_id、resource_id和policy_id必须大于0",
		)
	}
	if _, ok := dataResourceOperationSet[grant.Operation]; !ok {
		return model.DataGrant{}, myerrors.ErrDataResourceOperationInvalid
	}
	if err := validateDataGrantValidity(grant.ValidFrom, grant.ValidTo); err != nil {
		return model.DataGrant{}, err
	}
	return grant, nil
}

func (s *DataGrantConfigService) validateGrantContext(
	tx *gorm.DB,
	grant model.DataGrant,
) error {
	exists, err := s.grantSubjectExists(tx, grant.SubjectType, grant.SubjectId)
	if err != nil {
		return err
	}
	if !exists {
		return myerrors.ErrDataGrantSubjectNotFound
	}

	resource, err := s.resourceRepo.FindByIdForConfigDB(tx, grant.ResourceId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataResourceNotFound
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if !resource.State {
		return myerrors.ErrDataResourceStateInvalid
	}

	operation, err := s.operationRepo.FindByStableKeyForConfigDB(
		tx,
		resource.Id,
		grant.Operation,
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataResourceOperationNotFound
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if !operation.State {
		return myerrors.ErrDataResourceOperationInvalid
	}

	policy, err := s.policyRepo.FindByIdForConfigDB(tx, grant.PolicyId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataPolicyNotFound
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if !policy.State {
		return myerrors.ErrDataGrantPolicyInvalid
	}
	return s.validatePolicyForGrant(tx, resource, grant.Operation, policy)
}

func (s *DataGrantConfigService) validatePolicyForGrant(
	tx *gorm.DB,
	resource model.DataResource,
	operation string,
	policy model.DataPolicy,
) error {
	rules, err := s.ruleRepo.ListByPolicyForConfigDB(tx, policy.Id)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
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
			return myerrors.ErrDataGrantPolicyRuleInvalid
		}
		return nil
	case model.DataPolicyTypeRuleSet:
		if len(activeRules) == 0 || len(activeRules) > maxDataPolicyRules {
			return myerrors.ErrDataGrantPolicyRuleInvalid
		}
	default:
		return myerrors.ErrDataGrantPolicyInvalid
	}

	for _, rule := range activeRules {
		if err = s.validateRuleForGrant(tx, resource, operation, rule); err != nil {
			return err
		}
	}
	return nil
}

func (s *DataGrantConfigService) validateRuleForGrant(
	tx *gorm.DB,
	resource model.DataResource,
	operation string,
	rule model.DataPolicyRule,
) error {
	if rule.Sequence <= 0 ||
		!dataPolicyCodePattern.MatchString(rule.OwnershipCode) {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}
	if _, ok := dataPolicyScopeSourceSet[rule.ScopeSource]; !ok {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}
	if _, ok := dataPolicyRelationSet[rule.Relation]; !ok {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}
	if _, ok := dataPolicyOperatorSet[rule.Operator]; !ok {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}

	dimension, err := s.dimensionRepo.FindByIdForConfigDB(tx, rule.DimensionId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if !dimension.State {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}
	if err = validateRuleScopeCompatibility(rule, dimension); err != nil {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}
	if _, err = normalizePolicySpecifiedValues(
		json.RawMessage(rule.SpecifiedValues),
		rule.ScopeSource,
		rule.Operator,
		dimension.ValueType,
	); err != nil {
		return myerrors.ErrDataGrantPolicyRuleInvalid
	}

	ownership, err := s.ownershipRepo.FindByStableKeyForConfigDB(
		tx,
		resource.Id,
		rule.OwnershipCode,
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataGrantOwnershipMismatch
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if !ownership.State ||
		ownership.DimensionId != rule.DimensionId ||
		ownership.ValueType != dimension.ValueType {
		return myerrors.ErrDataGrantOwnershipMismatch
	}
	switch ownership.BindingType {
	case model.DataOwnershipBindingTypeMetadataField:
		if ownership.TableFieldId == nil {
			return myerrors.ErrDataGrantOwnershipMismatch
		}
	case model.DataOwnershipBindingTypeRegisteredField:
		if ownership.AdapterFieldCode == nil || s.registeredFieldChecker == nil {
			return myerrors.ErrDataGrantOwnershipMismatch
		}
		if err = s.registeredFieldChecker.ValidateOperation(
			datapermission.OwnershipFieldOperationValidation{
				ResourceCode:  resource.ResourceCode,
				OwnershipCode: ownership.OwnershipCode,
				Operation:     operation,
			},
		); err != nil {
			return myerrors.ErrDataGrantOwnershipMismatch
		}
	default:
		return myerrors.ErrDataGrantOwnershipMismatch
	}
	return nil
}

func (s *DataGrantConfigService) grantSubjectExists(
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
		exists, err = s.grantRepo.RoleExistsForConfig(tx, subjectId)
	case model.DataGrantSubjectTypeUser:
		exists, err = s.grantRepo.UserExistsForConfig(tx, subjectId)
	default:
		return false, myerrors.ErrDataGrantSubjectTypeInvalid
	}
	if err != nil {
		return false, myerrors.WrapDatabaseError(err)
	}
	return exists, nil
}

func (s *DataGrantConfigService) ensureGrantAbsent(
	tx *gorm.DB,
	grant model.DataGrant,
) error {
	existing, err := s.grantRepo.FindByStableKeyForConfigDB(
		tx,
		grant.SubjectType,
		grant.SubjectId,
		grant.ResourceId,
		grant.Operation,
		grant.PolicyId,
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	if existing.State {
		return myerrors.ErrDataGrantDuplicate
	}
	return myerrors.ErrDataGrantExists
}

func (s *DataGrantConfigService) findGrantForUpdate(
	tx *gorm.DB,
	grantId int,
) (model.DataGrant, error) {
	grant, err := s.grantRepo.FindByIdForConfigDB(tx, grantId)
	if err != nil {
		return model.DataGrant{}, mapDataGrantReadError(err)
	}
	return grant, nil
}

func (s *DataGrantConfigService) grantDetail(
	ctx *gin.Context,
	grant model.DataGrant,
) (response.DataGrantDetailRes, error) {
	resource, err := s.resourceRepo.FindByIdForConfig(ctx, grant.ResourceId)
	if err != nil {
		return response.DataGrantDetailRes{}, mapDataResourceReadError(err)
	}
	policy, err := s.policyRepo.FindByIdForConfig(ctx, grant.PolicyId)
	if err != nil {
		return response.DataGrantDetailRes{}, mapDataPolicyReadError(err)
	}
	result := response.NewDataGrantDetailRes(grant)
	result.Resource = &response.DataPermissionReferenceSummaryRes{
		Id:   resource.Id,
		Code: resource.ResourceCode,
		Name: resource.Name,
	}
	result.Policy = &response.DataPermissionReferenceSummaryRes{
		Id:   policy.Id,
		Code: policy.Code,
		Name: policy.Name,
	}
	return result, nil
}

func (s *DataGrantConfigService) generateGrantId() (int, error) {
	if s.sf == nil {
		return 0, myerrors.WrapSystemError(errors.New("data grant id generator is required"))
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return 0, myerrors.WrapSystemError(err)
	}
	return int(id), nil
}

func (s *DataGrantConfigService) recordGrantAudit(
	ctx *gin.Context,
	tx *gorm.DB,
	action string,
	grant model.DataGrant,
	changes map[string]TransactionalAuditChange,
) error {
	if s.auditWriter == nil {
		return myerrors.WrapSystemError(ErrTransactionalAuditRepositoryRequired)
	}
	if err := s.auditWriter.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: dataGrantAuditType,
		ResourceCode: dataGrantStableKey(grant),
		ResourceId:   strconv.Itoa(grant.Id),
		Changes:      changes,
	}); err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func mapDataGrantReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrDataGrantNotFound
	}
	return myerrors.WrapDatabaseError(err)
}

func dataGrantListResponses(values []model.DataGrant) []response.DataGrantListRes {
	result := make([]response.DataGrantListRes, 0, len(values))
	for _, value := range values {
		result = append(result, response.NewDataGrantListRes(value))
	}
	return result
}

func dataGrantStableKey(grant model.DataGrant) string {
	return strings.Join([]string{
		grant.SubjectType,
		strconv.Itoa(grant.SubjectId),
		strconv.Itoa(grant.ResourceId),
		grant.Operation,
		strconv.Itoa(grant.PolicyId),
	}, ":")
}

func normalizeDataGrantDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := time.Date(
		value.Year(),
		value.Month(),
		value.Day(),
		0,
		0,
		0,
		0,
		value.Location(),
	)
	return &normalized
}

func validateDataGrantValidity(validFrom, validTo *time.Time) error {
	if validFrom != nil && validTo != nil && validFrom.After(*validTo) {
		return myerrors.ErrDataGrantValidityInvalid
	}
	return nil
}
