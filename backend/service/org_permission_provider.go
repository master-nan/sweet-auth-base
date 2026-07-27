package service

import (
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/model"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OrgPermissionProvider is the Organization fact boundary consumed by future
// data-permission runtime code. It does not expose policy or SQL behavior.
type OrgPermissionProvider interface {
	GetEmployeeByUser(*gin.Context, int) (response.OrgEmployeeContextRes, error)
	GetEffectiveAssignments(*gin.Context, int, string) ([]response.OrgEffectiveAssignmentRes, error)
	GetEmployeeEffectiveOrganizationScope(*gin.Context, int, string) (response.OrgEffectiveOrganizationScopeRes, error)
}

var _ OrgPermissionProvider = (*OrgService)(nil)

func (s *OrgService) GetEmployeeByUser(
	ctx *gin.Context,
	userId int,
) (response.OrgEmployeeContextRes, error) {
	if userId <= 0 {
		return response.OrgEmployeeContextRes{},
			myerrors.NewParameterError("user_id必须大于0")
	}
	users, err := s.employeeRepo.FindBoundUserSummaries(ctx, []int{userId})
	if err != nil {
		return response.OrgEmployeeContextRes{}, myerrors.WrapDatabaseError(err)
	}
	if len(users) == 0 {
		return response.OrgEmployeeContextRes{}, myerrors.ErrOrgUserNotFound
	}

	employee, err := s.employeeRepo.FindByBoundUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewOrgEmployeeContextRes(userId, nil), nil
		}
		return response.OrgEmployeeContextRes{}, myerrors.WrapDatabaseError(err)
	}
	employeeId := employee.Id
	return response.NewOrgEmployeeContextRes(userId, &employeeId), nil
}

func (s *OrgService) GetEffectiveAssignments(
	ctx *gin.Context,
	employeeId int,
	asOfDate string,
) ([]response.OrgEffectiveAssignmentRes, error) {
	if employeeId <= 0 {
		return nil, myerrors.NewParameterError("employee_id必须大于0")
	}
	asOf, err := normalizeRequiredAssignmentDate(asOfDate)
	if err != nil {
		return nil, err
	}

	employee, err := s.employeeRepo.FindByIdForRead(ctx, employeeId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, myerrors.ErrOrgEmployeeNotFound
		}
		return nil, myerrors.WrapDatabaseError(err)
	}
	if !isOrgEmployeeEffective(employee, asOf) {
		return nil, myerrors.ErrOrgEmployeeInactive
	}

	assignments, err := s.assignmentRepo.ListEffectiveByEmployee(
		ctx,
		employeeId,
		asOf,
		orgAssignmentSummaryMaxCount+1,
	)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	if len(assignments) > orgAssignmentSummaryMaxCount {
		return nil, myerrors.ErrOrgAssignmentResultTooLarge
	}
	if len(assignments) == 0 {
		return []response.OrgEffectiveAssignmentRes{}, nil
	}
	if err = s.validateEffectiveAssignmentReferences(ctx, assignments, asOf); err != nil {
		return nil, err
	}

	result := make([]response.OrgEffectiveAssignmentRes, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, response.OrgEffectiveAssignmentRes{
			AssignmentId:  assignment.Id,
			EmployeeId:    assignment.EmployeeId,
			LegalEntityId: assignment.LegalEntityId,
			OrgUnitId:     assignment.OrgUnitId,
			PositionId:    assignment.PositionId,
			ValidFrom:     assignment.ValidFrom,
			ValidTo:       assignment.ValidTo,
		})
	}
	sortEffectiveAssignments(result)
	return result, nil
}

func (s *OrgService) GetEmployeeEffectiveOrganizationScope(
	ctx *gin.Context,
	employeeId int,
	asOfDate string,
) (response.OrgEffectiveOrganizationScopeRes, error) {
	asOf, err := normalizeRequiredAssignmentDate(asOfDate)
	if err != nil {
		return response.OrgEffectiveOrganizationScopeRes{}, err
	}
	assignments, err := s.GetEffectiveAssignments(ctx, employeeId, asOfDate)
	if err != nil {
		return response.OrgEffectiveOrganizationScopeRes{}, err
	}

	result := response.OrgEffectiveOrganizationScopeRes{
		EmployeeId:      employeeId,
		AsOfDate:        asOf.In(model.AppLocation()).Format(orgAsOfDateLayout),
		ScopeStatus:     response.OrgEffectiveScopeEmpty,
		AssignmentCount: len(assignments),
		LegalEntityIds:  []int{},
		OrgUnitIds:      []int{},
	}
	if len(assignments) == 0 {
		return result, nil
	}

	legalEntityIds := make(map[int]struct{}, len(assignments))
	orgUnitIds := make(map[int]struct{}, len(assignments))
	for _, assignment := range assignments {
		legalEntityIds[assignment.LegalEntityId] = struct{}{}
		orgUnitIds[assignment.OrgUnitId] = struct{}{}
	}
	result.ScopeStatus = response.OrgEffectiveScopeResolved
	result.LegalEntityIds = sortedOrganizationIds(legalEntityIds)
	result.OrgUnitIds = sortedOrganizationIds(orgUnitIds)
	return result, nil
}

func normalizeRequiredAssignmentDate(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, myerrors.NewParameterError("as_of_date不能为空")
	}
	scope, err := normalizeAssignmentReadScope("current", raw)
	if err != nil {
		return time.Time{}, err
	}
	return scope.AsOf, nil
}

func (s *OrgService) validateEffectiveAssignmentReferences(
	ctx *gin.Context,
	assignments []model.OrgAssignment,
	asOf time.Time,
) error {
	legalIds := make(map[int]struct{}, len(assignments))
	unitIds := make(map[int]struct{}, len(assignments))
	for _, assignment := range assignments {
		legalIds[assignment.LegalEntityId] = struct{}{}
		unitIds[assignment.OrgUnitId] = struct{}{}
	}

	legalEntities, err := s.legalEntityRepo.FindByIdsForDisplay(
		ctx,
		sortedOrganizationIds(legalIds),
	)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	legalById := make(map[int]model.OrgLegalEntity, len(legalEntities))
	for _, entity := range legalEntities {
		legalById[entity.Id] = entity
	}

	units, err := s.orgUnitRepo.FindByIdsForDisplay(ctx, sortedOrganizationIds(unitIds))
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	unitById := make(map[int]model.OrgUnit, len(units))
	for _, unit := range units {
		unitById[unit.Id] = unit
	}

	for _, assignment := range assignments {
		entity, exists := legalById[assignment.LegalEntityId]
		if !exists {
			return myerrors.ErrOrgLegalEntityNotFound
		}
		if !isLegalEntityEffective(entity, asOf) {
			return myerrors.ErrOrgLegalEntityInactive
		}
		unit, exists := unitById[assignment.OrgUnitId]
		if !exists {
			return myerrors.ErrOrgUnitNotFound
		}
		if !isOrgUnitEffective(unit, asOf) {
			return myerrors.ErrOrgUnitInactive
		}
	}
	return nil
}

func sortEffectiveAssignments(assignments []response.OrgEffectiveAssignmentRes) {
	sort.Slice(assignments, func(i, j int) bool {
		left, right := assignments[i], assignments[j]
		if left.LegalEntityId != right.LegalEntityId {
			return left.LegalEntityId < right.LegalEntityId
		}
		if left.OrgUnitId != right.OrgUnitId {
			return left.OrgUnitId < right.OrgUnitId
		}
		switch {
		case left.PositionId == nil && right.PositionId != nil:
			return false
		case left.PositionId != nil && right.PositionId == nil:
			return true
		case left.PositionId != nil && right.PositionId != nil &&
			*left.PositionId != *right.PositionId:
			return *left.PositionId < *right.PositionId
		default:
			return left.AssignmentId < right.AssignmentId
		}
	})
}

func sortedOrganizationIds(values map[int]struct{}) []int {
	ids := make([]int, 0, len(values))
	for id := range values {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}
