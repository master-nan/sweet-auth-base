package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	orgAsOfDateLayout            = time.DateOnly
	orgStructureTreeMaxNodeCount = 5000
	orgAssignmentSummaryMaxCount = 5000
	orgEmployeeBound             = "bound"
	orgEmployeeUnbound           = "unbound"
	orgEmployeeBindUserAction    = "bind_user"
	orgEmployeeUnbindUserAction  = "unbind_user"
)

// OrgService is the public read boundary for Organization Master Data. Other
// modules call this service instead of reading Organization repositories.
type OrgService struct {
	legalEntityRepo   repository.OrgLegalEntityRepository
	orgUnitRepo       repository.OrgUnitRepository
	structureRepo     repository.OrgStructureRepository
	structureNodeRepo repository.OrgStructureNodeRepository
	employeeRepo      repository.OrgEmployeeRepository
	positionRepo      repository.OrgPositionRepository
	assignmentRepo    repository.OrgAssignmentRepository
	syncBatchRepo     repository.OrgSyncBatchRepository
	syncRecordRepo    repository.OrgSyncRecordRepository
	auditWriter       TransactionalAuditWriter
}

func NewOrgService(
	legalEntityRepo repository.OrgLegalEntityRepository,
	orgUnitRepo repository.OrgUnitRepository,
	structureRepo repository.OrgStructureRepository,
	structureNodeRepo repository.OrgStructureNodeRepository,
	employeeRepo repository.OrgEmployeeRepository,
	positionRepo repository.OrgPositionRepository,
	assignmentRepo repository.OrgAssignmentRepository,
	syncBatchRepo repository.OrgSyncBatchRepository,
	syncRecordRepo repository.OrgSyncRecordRepository,
	auditWriter TransactionalAuditWriter,
) *OrgService {
	return &OrgService{
		legalEntityRepo:   legalEntityRepo,
		orgUnitRepo:       orgUnitRepo,
		structureRepo:     structureRepo,
		structureNodeRepo: structureNodeRepo,
		employeeRepo:      employeeRepo,
		positionRepo:      positionRepo,
		assignmentRepo:    assignmentRepo,
		syncBatchRepo:     syncBatchRepo,
		syncRecordRepo:    syncRecordRepo,
		auditWriter:       auditWriter,
	}
}

func (s *OrgService) QueryLegalEntities(
	ctx *gin.Context,
	req request.OrgLegalEntityQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgLegalEntityListRes], error) {
	var result response.ListResult[response.OrgLegalEntityListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeLegalEntityReadScope(req.OrgLegalEntityReadScopeReq)
	if err != nil {
		return result, err
	}
	table.TableCode = "org_legal_entity"
	rows, err := s.legalEntityRepo.Query(ctx, &req, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.OrgLegalEntityListRes, 0, len(rows.Data))
	for _, entity := range rows.Data {
		result.Data = append(result.Data, response.NewOrgLegalEntityListRes(entity))
	}
	return result, nil
}

func (s *OrgService) GetLegalEntityDetail(
	ctx *gin.Context,
	legalEntityId int,
	req request.OrgLegalEntityDetailReq,
) (response.OrgLegalEntityDetailRes, error) {
	if legalEntityId <= 0 {
		return response.OrgLegalEntityDetailRes{}, myerrors.NewParameterError("legal_entity_id必须大于0")
	}
	scope, err := normalizeLegalEntityReadScope(req.OrgLegalEntityReadScopeReq)
	if err != nil {
		return response.OrgLegalEntityDetailRes{}, err
	}
	entity, err := s.legalEntityRepo.FindByIdForRead(ctx, legalEntityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgLegalEntityDetailRes{}, myerrors.ErrOrgLegalEntityNotFound
		}
		return response.OrgLegalEntityDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	if !legalEntityVisible(entity, scope) {
		return response.OrgLegalEntityDetailRes{}, myerrors.ErrOrgLegalEntityNotFound
	}
	return response.NewOrgLegalEntityDetailRes(entity), nil
}

func (s *OrgService) GetLegalEntityTree(
	ctx *gin.Context,
	req request.OrgLegalEntityTreeReq,
) ([]response.OrgLegalEntityTreeNodeRes, error) {
	if req.RootId != nil && *req.RootId <= 0 {
		return nil, myerrors.NewParameterError("root_id必须大于0")
	}
	scope, err := normalizeLegalEntityReadScope(req.OrgLegalEntityReadScopeReq)
	if err != nil {
		return nil, err
	}
	entities, err := s.legalEntityRepo.ListForTree(ctx, scope)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	return buildLegalEntityTree(entities, scope, req.RootId)
}

func (s *OrgService) QueryLegalEntityOptions(
	ctx *gin.Context,
	req request.OrgLegalEntityOptionsReq,
	table model.SysTable,
) (response.OrgSelectorOptionsRes, error) {
	var result response.OrgSelectorOptionsRes
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeLegalEntityReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	selectedIds, err := normalizeLegalEntitySelectedIds(req.SelectedIds)
	if err != nil {
		return result, err
	}

	queryReq := request.OrgLegalEntityQueryReq{
		Basic: request.Basic{
			Page: req.Page,
			Num:  req.Num,
			QuickQuery: &request.QuickQuery{
				Keyword: strings.TrimSpace(req.Keyword),
			},
		},
		OrgLegalEntityReadScopeReq: req.OrgReadScopeReq,
	}
	table.TableCode = "org_legal_entity"
	rows, err := s.legalEntityRepo.Query(ctx, &queryReq, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}

	result.Total = rows.Total
	result.Items = make([]response.OrgSelectorOptionRes, 0, len(rows.Data)+len(selectedIds))
	seen := make(map[int]struct{}, len(rows.Data)+len(selectedIds))
	for _, entity := range rows.Data {
		result.Items = append(
			result.Items,
			response.NewOrgLegalEntityOptionRes(entity, !isLegalEntityEffective(entity, scope.AsOf)),
		)
		seen[entity.Id] = struct{}{}
	}

	selected, err := s.legalEntityRepo.FindByIdsForDisplay(ctx, selectedIds)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	selectedById := make(map[int]model.OrgLegalEntity, len(selected))
	for _, entity := range selected {
		selectedById[entity.Id] = entity
	}
	for _, id := range selectedIds {
		if _, exists := seen[id]; exists {
			continue
		}
		entity, exists := selectedById[id]
		if !exists {
			continue
		}
		result.Items = append(
			result.Items,
			response.NewOrgLegalEntityOptionRes(entity, !isLegalEntityEffective(entity, scope.AsOf)),
		)
		seen[id] = struct{}{}
	}
	return result, nil
}

func (s *OrgService) QueryStructures(
	ctx *gin.Context,
	req request.OrgStructureQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgStructureListRes], error) {
	var result response.ListResult[response.OrgStructureListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	table.TableCode = "org_structure"
	rows, err := s.structureRepo.QueryForRead(ctx, &req, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.OrgStructureListRes, 0, len(rows.Data))
	for _, structure := range rows.Data {
		result.Data = append(result.Data, response.NewOrgStructureListRes(structure))
	}
	return result, nil
}

func (s *OrgService) GetStructureDetail(
	ctx *gin.Context,
	structureId int,
	req request.OrgStructureDetailReq,
) (response.OrgStructureDetailRes, error) {
	if structureId <= 0 {
		return response.OrgStructureDetailRes{}, myerrors.NewParameterError("structure_id必须大于0")
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return response.OrgStructureDetailRes{}, err
	}
	structure, err := s.structureRepo.FindByIdForRead(ctx, structureId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgStructureDetailRes{}, myerrors.ErrOrgStructureNotFound
		}
		return response.OrgStructureDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	if !orgStructureVisible(structure, scope) {
		return response.OrgStructureDetailRes{}, myerrors.ErrOrgStructureInactive
	}
	return response.NewOrgStructureDetailRes(structure), nil
}

func (s *OrgService) QueryStructureOptions(
	ctx *gin.Context,
	req request.OrgStructureOptionsReq,
	table model.SysTable,
) (response.ListResult[response.OrgSelectorOptionRes], error) {
	var result response.ListResult[response.OrgSelectorOptionRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	selectedIds, err := normalizeOrganizationSelectedIds(req.SelectedIds)
	if err != nil {
		return result, err
	}

	queryReq := request.OrgStructureQueryReq{
		Basic: request.Basic{
			Page:       req.Page,
			Num:        req.Num,
			QuickQuery: &request.QuickQuery{Keyword: strings.TrimSpace(req.Keyword)},
		},
		OrgReadScopeReq: req.OrgReadScopeReq,
		LegalEntityId:   req.LegalEntityId,
	}
	table.TableCode = "org_structure"
	rows, err := s.structureRepo.QueryForRead(ctx, &queryReq, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.OrgSelectorOptionRes, 0, len(rows.Data)+len(selectedIds))
	seen := make(map[int]struct{}, len(rows.Data)+len(selectedIds))
	for _, structure := range rows.Data {
		result.Data = append(
			result.Data,
			response.NewOrgStructureOptionRes(
				structure,
				!isOrgStructureEffective(structure, scope.AsOf),
			),
		)
		seen[structure.Id] = struct{}{}
	}

	selected, err := s.structureRepo.FindByIdsForDisplay(ctx, selectedIds)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	selectedById := make(map[int]model.OrgStructure, len(selected))
	for _, structure := range selected {
		selectedById[structure.Id] = structure
	}
	for _, id := range selectedIds {
		if _, exists := seen[id]; exists {
			continue
		}
		structure, exists := selectedById[id]
		if !exists {
			continue
		}
		result.Data = append(
			result.Data,
			response.NewOrgStructureOptionRes(
				structure,
				!isOrgStructureEffective(structure, scope.AsOf),
			),
		)
		seen[id] = struct{}{}
	}
	return result, nil
}

func (s *OrgService) QueryOrgUnits(
	ctx *gin.Context,
	req request.OrgUnitQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgUnitListRes], error) {
	var result response.ListResult[response.OrgUnitListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	if err := normalizeOrgUnitLegalEntityFilter(&req); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	table.TableCode = "org_unit"
	rows, err := s.orgUnitRepo.QueryForRead(ctx, &req, table, scope, nil)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.OrgUnitListRes, 0, len(rows.Data))
	for _, unit := range rows.Data {
		result.Data = append(result.Data, response.NewOrgUnitListRes(unit))
	}
	return result, nil
}

func (s *OrgService) GetOrgUnitDetail(
	ctx *gin.Context,
	orgUnitId int,
	req request.OrgUnitDetailReq,
) (response.OrgUnitDetailRes, error) {
	if orgUnitId <= 0 {
		return response.OrgUnitDetailRes{}, myerrors.NewParameterError("org_unit_id必须大于0")
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return response.OrgUnitDetailRes{}, err
	}
	unit, err := s.orgUnitRepo.FindByIdForRead(ctx, orgUnitId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgUnitDetailRes{}, myerrors.ErrOrgUnitNotFound
		}
		return response.OrgUnitDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	if !orgUnitVisible(unit, scope) {
		return response.OrgUnitDetailRes{}, myerrors.ErrOrgUnitNotFound
	}

	result := response.NewOrgUnitDetailRes(unit)
	if unit.PrimaryLegalEntityId != nil {
		legalEntity, legalErr := s.legalEntityRepo.FindByIdForRead(ctx, *unit.PrimaryLegalEntityId)
		switch {
		case legalErr == nil:
			summary := response.NewOrgReferenceSummaryRes(
				legalEntity.Id,
				legalEntity.Code,
				legalEntity.Name,
			)
			result.PrimaryLegalEntity = &summary
		case errors.Is(legalErr, gorm.ErrRecordNotFound):
			// Preserve the stable foreign-key ID even if a historical display
			// record is no longer available.
		default:
			return response.OrgUnitDetailRes{}, myerrors.WrapDatabaseError(legalErr)
		}
	}
	return result, nil
}

func (s *OrgService) QueryOrgUnitOptions(
	ctx *gin.Context,
	req request.OrgUnitOptionsReq,
	table model.SysTable,
) (response.OrgSelectorOptionsRes, error) {
	var result response.OrgSelectorOptionsRes
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	selectedIds, err := normalizeOrganizationSelectedIds(req.SelectedIds)
	if err != nil {
		return result, err
	}
	if req.StructureId != nil {
		structure, findErr := s.structureRepo.FindByIdForRead(ctx, *req.StructureId)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return result, myerrors.ErrOrgStructureNotFound
			}
			return result, myerrors.WrapDatabaseError(findErr)
		}
		if !orgStructureVisible(structure, scope) {
			return result, myerrors.ErrOrgStructureInactive
		}
	}

	queryReq := request.OrgUnitQueryReq{
		Basic: request.Basic{
			Page:       req.Page,
			Num:        req.Num,
			QuickQuery: &request.QuickQuery{Keyword: strings.TrimSpace(req.Keyword)},
		},
		OrgReadScopeReq: req.OrgReadScopeReq,
		LegalEntityId:   req.LegalEntityId,
	}
	if err = normalizeOrgUnitLegalEntityFilter(&queryReq); err != nil {
		return result, err
	}
	table.TableCode = "org_unit"
	rows, err := s.orgUnitRepo.QueryForRead(ctx, &queryReq, table, scope, req.StructureId)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Items = make([]response.OrgSelectorOptionRes, 0, len(rows.Data)+len(selectedIds))
	seen := make(map[int]struct{}, len(rows.Data)+len(selectedIds))
	for _, unit := range rows.Data {
		result.Items = append(
			result.Items,
			response.NewOrgUnitOptionRes(unit, !isOrgUnitEffective(unit, scope.AsOf)),
		)
		seen[unit.Id] = struct{}{}
	}

	selected, err := s.orgUnitRepo.FindByIdsForDisplay(ctx, selectedIds)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	selectedById := make(map[int]model.OrgUnit, len(selected))
	for _, unit := range selected {
		selectedById[unit.Id] = unit
	}
	for _, id := range selectedIds {
		if _, exists := seen[id]; exists {
			continue
		}
		unit, exists := selectedById[id]
		if !exists {
			continue
		}
		result.Items = append(
			result.Items,
			response.NewOrgUnitOptionRes(unit, !isOrgUnitEffective(unit, scope.AsOf)),
		)
		seen[id] = struct{}{}
	}
	return result, nil
}

func (s *OrgService) QueryEmployees(
	ctx *gin.Context,
	req request.OrgEmployeeQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgEmployeeListRes], error) {
	var result response.ListResult[response.OrgEmployeeListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	if err := normalizeEmployeeBoundStatus(&req); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	table.TableCode = "org_employee"
	rows, err := s.employeeRepo.QueryForRead(ctx, &req, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}

	result.Total = rows.Total
	result.Data = make([]response.OrgEmployeeListRes, 0, len(rows.Data))
	for _, employee := range rows.Data {
		result.Data = append(result.Data, response.NewOrgEmployeeListRes(employee))
	}
	if err = s.attachEmployeeAccountSummaries(ctx, result.Data); err != nil {
		return response.ListResult[response.OrgEmployeeListRes]{}, err
	}
	return result, nil
}

func (s *OrgService) GetEmployeeDetail(
	ctx *gin.Context,
	employeeId int,
	req request.OrgEmployeeDetailReq,
) (response.OrgEmployeeDetailRes, error) {
	if employeeId <= 0 {
		return response.OrgEmployeeDetailRes{}, myerrors.NewParameterError("employee_id必须大于0")
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return response.OrgEmployeeDetailRes{}, err
	}
	employee, err := s.employeeRepo.FindByIdForRead(ctx, employeeId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgEmployeeDetailRes{}, myerrors.ErrOrgEmployeeNotFound
		}
		return response.OrgEmployeeDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	if !orgEmployeeVisible(employee, scope) {
		return response.OrgEmployeeDetailRes{}, myerrors.ErrOrgEmployeeNotFound
	}

	result := response.NewOrgEmployeeDetailRes(employee)
	list := []response.OrgEmployeeListRes{result.OrgEmployeeListRes}
	if err = s.attachEmployeeAccountSummaries(ctx, list); err != nil {
		return response.OrgEmployeeDetailRes{}, err
	}
	result.OrgEmployeeListRes = list[0]
	return result, nil
}

func (s *OrgService) QueryEmployeeOptions(
	ctx *gin.Context,
	req request.OrgEmployeeOptionsReq,
	table model.SysTable,
) (response.OrgSelectorOptionsRes, error) {
	var result response.OrgSelectorOptionsRes
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	selectedIds, err := normalizeOrganizationSelectedIds(req.SelectedIds)
	if err != nil {
		return result, err
	}

	queryReq := request.OrgEmployeeQueryReq{
		Basic: request.Basic{
			Page:       req.Page,
			Num:        req.Num,
			QuickQuery: &request.QuickQuery{Keyword: strings.TrimSpace(req.Keyword)},
		},
		OrgReadScopeReq: req.OrgReadScopeReq,
		LegalEntityId:   req.LegalEntityId,
		OrgUnitId:       req.OrgUnitId,
		PositionId:      req.PositionId,
	}
	table.TableCode = "org_employee"
	rows, err := s.employeeRepo.QueryForRead(ctx, &queryReq, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}

	result.Total = rows.Total
	result.Items = make([]response.OrgSelectorOptionRes, 0, len(rows.Data)+len(selectedIds))
	seen := make(map[int]struct{}, len(rows.Data)+len(selectedIds))
	for _, employee := range rows.Data {
		result.Items = append(
			result.Items,
			response.NewOrgEmployeeOptionRes(employee, !isOrgEmployeeEffective(employee, scope.AsOf)),
		)
		seen[employee.Id] = struct{}{}
	}

	selected, err := s.employeeRepo.FindByIdsForDisplay(ctx, selectedIds)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	selectedById := make(map[int]model.OrgEmployee, len(selected))
	for _, employee := range selected {
		selectedById[employee.Id] = employee
	}
	for _, id := range selectedIds {
		if _, exists := seen[id]; exists {
			continue
		}
		employee, exists := selectedById[id]
		if !exists {
			continue
		}
		result.Items = append(
			result.Items,
			response.NewOrgEmployeeOptionRes(employee, !isOrgEmployeeEffective(employee, scope.AsOf)),
		)
		seen[id] = struct{}{}
	}
	return result, nil
}

func (s *OrgService) BindEmployeeUser(
	ctx *gin.Context,
	req request.OrgEmployeeBindUserReq,
) (response.OrgEmployeeUserBindingRes, error) {
	if req.EmployeeId <= 0 {
		return response.OrgEmployeeUserBindingRes{}, myerrors.NewParameterError("employee_id必须大于0")
	}
	if req.UserId <= 0 {
		return response.OrgEmployeeUserBindingRes{}, myerrors.NewParameterError("user_id必须大于0")
	}
	if ctx == nil {
		return response.OrgEmployeeUserBindingRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}

	var account repository.OrgBoundUserSummary
	err := RunInTransaction(ctx, s.employeeRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		employee, err := s.employeeRepo.FindByIdForBinding(tx, req.EmployeeId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.ErrOrgEmployeeNotFound
			}
			return myerrors.WrapDatabaseError(err)
		}
		if employee.UserId != nil {
			return myerrors.ErrOrgEmployeeAlreadyBound
		}

		account, err = s.employeeRepo.FindUserForBinding(tx, req.UserId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.ErrOrgUserNotFound
			}
			return myerrors.WrapDatabaseError(err)
		}
		if _, err = s.employeeRepo.FindByBoundUserIdForBinding(tx, req.UserId); err == nil {
			return myerrors.ErrOrgUserAlreadyBound
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.WrapDatabaseError(err)
		}

		if err = s.employeeRepo.UpdatePlatformFields(
			tx,
			req.EmployeeId,
			map[string]any{"user_id": req.UserId},
		); err != nil {
			if isEmployeeUserBindingUniqueViolation(err) {
				return myerrors.ErrOrgUserAlreadyBound
			}
			return myerrors.WrapDatabaseError(err)
		}
		if err = s.recordEmployeeUserBindingAudit(
			ctx,
			tx,
			orgEmployeeBindUserAction,
			req.EmployeeId,
			nil,
			&req.UserId,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return response.OrgEmployeeUserBindingRes{}, err
	}

	accountRes := response.NewOrgBoundUserSummaryRes(account.UserId, account.UserName)
	return response.NewOrgEmployeeUserBindingRes(req.EmployeeId, &accountRes), nil
}

func (s *OrgService) UnbindEmployeeUser(
	ctx *gin.Context,
	req request.OrgEmployeeUnbindUserReq,
) (response.OrgEmployeeUserBindingRes, error) {
	if req.EmployeeId <= 0 {
		return response.OrgEmployeeUserBindingRes{}, myerrors.NewParameterError("employee_id必须大于0")
	}
	if ctx == nil {
		return response.OrgEmployeeUserBindingRes{}, myerrors.WrapSystemError(ErrTransactionContextRequired)
	}

	err := RunInTransaction(ctx, s.employeeRepo.DBWithContext(ctx), func(tx *gorm.DB) error {
		employee, err := s.employeeRepo.FindByIdForBinding(tx, req.EmployeeId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return myerrors.ErrOrgEmployeeNotFound
			}
			return myerrors.WrapDatabaseError(err)
		}
		oldUserId := cloneOptionalInt(employee.UserId)
		if employee.UserId != nil {
			if err = s.employeeRepo.UpdatePlatformFields(
				tx,
				req.EmployeeId,
				map[string]any{"user_id": nil},
			); err != nil {
				return myerrors.WrapDatabaseError(err)
			}
		}
		if err = s.recordEmployeeUserBindingAudit(
			ctx,
			tx,
			orgEmployeeUnbindUserAction,
			req.EmployeeId,
			oldUserId,
			nil,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return response.OrgEmployeeUserBindingRes{}, err
	}
	return response.NewOrgEmployeeUserBindingRes(req.EmployeeId, nil), nil
}

func (s *OrgService) QueryPositions(
	ctx *gin.Context,
	req request.OrgPositionQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgPositionListRes], error) {
	var result response.ListResult[response.OrgPositionListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	table.TableCode = "org_position"
	rows, err := s.positionRepo.QueryForRead(ctx, &req, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.OrgPositionListRes, 0, len(rows.Data))
	for _, position := range rows.Data {
		result.Data = append(result.Data, response.NewOrgPositionListRes(position))
	}
	return result, nil
}

func (s *OrgService) GetPositionDetail(
	ctx *gin.Context,
	positionId int,
	req request.OrgPositionDetailReq,
) (response.OrgPositionDetailRes, error) {
	if positionId <= 0 {
		return response.OrgPositionDetailRes{}, myerrors.NewParameterError("position_id必须大于0")
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return response.OrgPositionDetailRes{}, err
	}
	position, err := s.positionRepo.FindByIdForRead(ctx, positionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgPositionDetailRes{}, myerrors.ErrOrgPositionNotFound
		}
		return response.OrgPositionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	if !orgPositionVisible(position, scope) {
		return response.OrgPositionDetailRes{}, myerrors.ErrOrgPositionNotFound
	}

	result := response.NewOrgPositionDetailRes(position)
	unit, unitErr := s.orgUnitRepo.FindByIdForRead(ctx, position.OrgUnitId)
	switch {
	case unitErr == nil:
		unitSummary := response.NewOrgReferenceSummaryRes(unit.Id, unit.Code, unit.Name)
		result.OrgUnit = &unitSummary
		if unit.PrimaryLegalEntityId != nil {
			legalEntity, legalErr := s.legalEntityRepo.FindByIdForRead(
				ctx,
				*unit.PrimaryLegalEntityId,
			)
			switch {
			case legalErr == nil:
				legalSummary := response.NewOrgReferenceSummaryRes(
					legalEntity.Id,
					legalEntity.Code,
					legalEntity.Name,
				)
				result.LegalEntity = &legalSummary
			case errors.Is(legalErr, gorm.ErrRecordNotFound):
			default:
				return response.OrgPositionDetailRes{}, myerrors.WrapDatabaseError(legalErr)
			}
		}
	case errors.Is(unitErr, gorm.ErrRecordNotFound):
	default:
		return response.OrgPositionDetailRes{}, myerrors.WrapDatabaseError(unitErr)
	}
	return result, nil
}

func (s *OrgService) QueryPositionOptions(
	ctx *gin.Context,
	req request.OrgPositionOptionsReq,
	table model.SysTable,
) (response.OrgSelectorOptionsRes, error) {
	var result response.OrgSelectorOptionsRes
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return result, err
	}
	selectedIds, err := normalizeOrganizationSelectedIds(req.SelectedIds)
	if err != nil {
		return result, err
	}

	queryReq := request.OrgPositionQueryReq{
		Basic: request.Basic{
			Page:       req.Page,
			Num:        req.Num,
			QuickQuery: &request.QuickQuery{Keyword: strings.TrimSpace(req.Keyword)},
		},
		OrgReadScopeReq: req.OrgReadScopeReq,
		LegalEntityId:   req.LegalEntityId,
		OrgUnitId:       req.OrgUnitId,
	}
	table.TableCode = "org_position"
	rows, err := s.positionRepo.QueryForRead(ctx, &queryReq, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}

	result.Total = rows.Total
	result.Items = make([]response.OrgSelectorOptionRes, 0, len(rows.Data)+len(selectedIds))
	seen := make(map[int]struct{}, len(rows.Data)+len(selectedIds))
	for _, position := range rows.Data {
		result.Items = append(
			result.Items,
			response.NewOrgPositionOptionRes(position, !isOrgPositionEffective(position, scope.AsOf)),
		)
		seen[position.Id] = struct{}{}
	}

	selected, err := s.positionRepo.FindByIdsForDisplay(ctx, selectedIds)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	selectedById := make(map[int]model.OrgPosition, len(selected))
	for _, position := range selected {
		selectedById[position.Id] = position
	}
	for _, id := range selectedIds {
		if _, exists := seen[id]; exists {
			continue
		}
		position, exists := selectedById[id]
		if !exists {
			continue
		}
		result.Items = append(
			result.Items,
			response.NewOrgPositionOptionRes(position, !isOrgPositionEffective(position, scope.AsOf)),
		)
		seen[id] = struct{}{}
	}
	return result, nil
}

func (s *OrgService) QueryAssignments(
	ctx *gin.Context,
	req request.OrgAssignmentQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgAssignmentListRes], error) {
	var result response.ListResult[response.OrgAssignmentListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	if req.EmployeeId == nil || *req.EmployeeId <= 0 {
		return result, myerrors.NewParameterError("employee_id必须大于0")
	}
	if err := s.ensureEmployeeExists(ctx, *req.EmployeeId); err != nil {
		return result, err
	}
	scope, err := normalizeAssignmentReadScope(req.TimeScope, req.AsOfDate)
	if err != nil {
		return result, err
	}
	req.TimeScope = scope.TimeScope
	table.TableCode = "org_assignment"
	rows, err := s.assignmentRepo.QueryForRead(ctx, &req, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}

	result.Total = rows.Total
	result.Data = make([]response.OrgAssignmentListRes, 0, len(rows.Data))
	for _, assignment := range rows.Data {
		result.Data = append(
			result.Data,
			response.NewOrgAssignmentListRes(
				assignment,
				classifyAssignmentTimeScope(assignment, scope.AsOf),
			),
		)
	}
	if err = s.attachAssignmentReferences(ctx, result.Data); err != nil {
		return response.ListResult[response.OrgAssignmentListRes]{}, err
	}
	return result, nil
}

func (s *OrgService) GetAssignmentDetail(
	ctx *gin.Context,
	assignmentId int,
	_ request.OrgAssignmentDetailReq,
) (response.OrgAssignmentDetailRes, error) {
	if assignmentId <= 0 {
		return response.OrgAssignmentDetailRes{}, myerrors.NewParameterError(
			"assignment_id必须大于0",
		)
	}
	assignment, err := s.assignmentRepo.FindByIdForRead(ctx, assignmentId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgAssignmentDetailRes{}, myerrors.ErrOrgAssignmentNotFound
		}
		return response.OrgAssignmentDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	asOf := model.Now()
	list := []response.OrgAssignmentListRes{
		response.NewOrgAssignmentListRes(
			assignment,
			classifyAssignmentTimeScope(assignment, asOf),
		),
	}
	if err = s.attachAssignmentReferences(ctx, list); err != nil {
		return response.OrgAssignmentDetailRes{}, err
	}
	return response.OrgAssignmentDetailRes{OrgAssignmentListRes: list[0]}, nil
}

func (s *OrgService) GetEmployeeCurrentAssignmentSummary(
	ctx *gin.Context,
	employeeId int,
	req request.OrgEmployeeCurrentAssignmentSummaryReq,
	table model.SysTable,
) (response.OrgEmployeeCurrentAssignmentSummaryRes, error) {
	if employeeId <= 0 {
		return response.OrgEmployeeCurrentAssignmentSummaryRes{},
			myerrors.NewParameterError("employee_id必须大于0")
	}
	if err := s.ensureEmployeeExists(ctx, employeeId); err != nil {
		return response.OrgEmployeeCurrentAssignmentSummaryRes{}, err
	}
	scope, err := normalizeAssignmentReadScope(
		request.OrgAssignmentScopeCurrent,
		req.AsOfDate,
	)
	if err != nil {
		return response.OrgEmployeeCurrentAssignmentSummaryRes{}, err
	}
	queryReq := request.OrgAssignmentQueryReq{
		Basic: request.Basic{
			Page: 1,
			Num:  orgAssignmentSummaryMaxCount,
		},
		EmployeeId: &employeeId,
		TimeScope:  request.OrgAssignmentScopeCurrent,
		AsOfDate:   req.AsOfDate,
	}
	table.TableCode = "org_assignment"
	rows, err := s.assignmentRepo.QueryForRead(ctx, &queryReq, table, scope)
	if err != nil {
		return response.OrgEmployeeCurrentAssignmentSummaryRes{},
			myerrors.WrapDatabaseError(err)
	}
	if rows.Total > len(rows.Data) {
		return response.OrgEmployeeCurrentAssignmentSummaryRes{},
			myerrors.ErrOrgAssignmentResultTooLarge
	}

	assignments := make([]response.OrgAssignmentListRes, 0, len(rows.Data))
	for _, assignment := range rows.Data {
		assignments = append(
			assignments,
			response.NewOrgAssignmentListRes(
				assignment,
				request.OrgAssignmentScopeCurrent,
			),
		)
	}
	if err = s.attachAssignmentReferences(ctx, assignments); err != nil {
		return response.OrgEmployeeCurrentAssignmentSummaryRes{}, err
	}
	legalEntities, orgUnits, positions := collectAssignmentReferenceSummaries(assignments)
	return response.NewOrgEmployeeCurrentAssignmentSummaryRes(
		employeeId,
		scope.AsOf.In(model.AppLocation()).Format(orgAsOfDateLayout),
		len(assignments),
		legalEntities,
		orgUnits,
		positions,
	), nil
}

func (s *OrgService) QuerySyncBatches(
	ctx *gin.Context,
	req request.OrgSyncBatchQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgSyncBatchListRes], error) {
	var result response.ListResult[response.OrgSyncBatchListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	table.TableCode = "org_sync_batch"
	rows, err := s.syncBatchRepo.Query(ctx, &req, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.OrgSyncBatchListRes, 0, len(rows.Data))
	for _, batch := range rows.Data {
		result.Data = append(result.Data, response.NewOrgSyncBatchListRes(batch))
	}
	return result, nil
}

func (s *OrgService) GetSyncBatchDetail(
	ctx *gin.Context,
	batchId int,
) (response.OrgSyncBatchDetailRes, error) {
	if batchId <= 0 {
		return response.OrgSyncBatchDetailRes{},
			myerrors.NewParameterError("batch_id必须大于0")
	}
	batch, err := s.syncBatchRepo.FindByIdForRead(ctx, batchId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgSyncBatchDetailRes{}, myerrors.ErrOrgSyncBatchNotFound
		}
		return response.OrgSyncBatchDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewOrgSyncBatchDetailRes(batch), nil
}

func (s *OrgService) GetSyncBatchError(
	ctx *gin.Context,
	batchId int,
) (response.OrgSyncBatchErrorRes, error) {
	if batchId <= 0 {
		return response.OrgSyncBatchErrorRes{},
			myerrors.NewParameterError("batch_id必须大于0")
	}
	batch, err := s.syncBatchRepo.FindByIdForRead(ctx, batchId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgSyncBatchErrorRes{}, myerrors.ErrOrgSyncBatchNotFound
		}
		return response.OrgSyncBatchErrorRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewOrgSyncBatchErrorRes(batch), nil
}

func (s *OrgService) QuerySyncRecords(
	ctx *gin.Context,
	req request.OrgSyncRecordQueryReq,
	table model.SysTable,
) (response.ListResult[response.OrgSyncRecordListRes], error) {
	var result response.ListResult[response.OrgSyncRecordListRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	table.TableCode = "org_sync_record"
	rows, err := s.syncRecordRepo.Query(ctx, &req, table)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}
	result.Total = rows.Total
	result.Data = make([]response.OrgSyncRecordListRes, 0, len(rows.Data))
	for _, record := range rows.Data {
		result.Data = append(result.Data, response.NewOrgSyncRecordListRes(record))
	}
	return result, nil
}

func (s *OrgService) GetSyncRecordDetail(
	ctx *gin.Context,
	recordId int,
) (response.OrgSyncRecordDetailRes, error) {
	if recordId <= 0 {
		return response.OrgSyncRecordDetailRes{},
			myerrors.NewParameterError("record_id必须大于0")
	}
	record, err := s.syncRecordRepo.FindByIdForRead(ctx, recordId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgSyncRecordDetailRes{}, myerrors.ErrOrgSyncRecordNotFound
		}
		return response.OrgSyncRecordDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewOrgSyncRecordDetailRes(record), nil
}

func (s *OrgService) GetSyncRecordError(
	ctx *gin.Context,
	recordId int,
) (response.OrgSyncRecordErrorRes, error) {
	if recordId <= 0 {
		return response.OrgSyncRecordErrorRes{},
			myerrors.NewParameterError("record_id必须大于0")
	}
	record, err := s.syncRecordRepo.FindByIdForRead(ctx, recordId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.OrgSyncRecordErrorRes{}, myerrors.ErrOrgSyncRecordNotFound
		}
		return response.OrgSyncRecordErrorRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewOrgSyncRecordErrorRes(record), nil
}

func (s *OrgService) GetStructureOrgTree(
	ctx *gin.Context,
	req request.OrgStructureOrgTreeReq,
) ([]response.OrgStructureOrgTreeNodeRes, error) {
	if req.StructureId <= 0 {
		return nil, myerrors.NewParameterError("structure_id必须大于0")
	}
	if req.RootNodeId != nil && req.RootOrgUnitId != nil {
		return nil, myerrors.NewParameterError(
			"root_node_id与root_org_unit_id不能同时指定",
		)
	}
	scope, err := normalizeOrganizationReadScope(req.OrgReadScopeReq)
	if err != nil {
		return nil, err
	}
	return s.getStructureOrgTreeForRead(ctx, req, scope)
}

func (s *OrgService) getStructureOrgTreeForRead(
	ctx *gin.Context,
	req request.OrgStructureOrgTreeReq,
	scope repository.OrgReadScope,
) ([]response.OrgStructureOrgTreeNodeRes, error) {
	structure, err := s.structureRepo.FindByIdForRead(ctx, req.StructureId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, myerrors.ErrOrgStructureNotFound
		}
		return nil, myerrors.WrapDatabaseError(err)
	}
	if !orgStructureVisible(structure, scope) {
		return nil, myerrors.ErrOrgStructureInactive
	}

	nodes, err := s.structureNodeRepo.ListByStructureForRead(
		ctx,
		req.StructureId,
		scope,
		orgStructureTreeMaxNodeCount+1,
	)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	if len(nodes) > orgStructureTreeMaxNodeCount {
		return nil, myerrors.ErrOrgTreeTooLarge
	}

	unitIds := uniqueStructureNodeUnitIds(nodes)
	units, err := s.orgUnitRepo.FindByIdsForDisplay(ctx, unitIds)
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	unitsById := make(map[int]model.OrgUnit, len(units))
	for _, unit := range units {
		unitsById[unit.Id] = unit
	}

	visibleNodes := make([]model.OrgStructureNode, 0, len(nodes))
	for _, node := range nodes {
		unit, exists := unitsById[node.OrgUnitId]
		if !exists {
			zap.L().Warn(
				"organization structure node references a missing unit",
				zap.Int("structure_id", req.StructureId),
				zap.Int("structure_node_id", node.Id),
				zap.Int("org_unit_id", node.OrgUnitId),
			)
			return nil, myerrors.ErrOrgUnitNotFound
		}
		if orgUnitVisible(unit, scope) {
			visibleNodes = append(visibleNodes, node)
		}
	}
	return buildStructureOrgTree(
		structure,
		visibleNodes,
		unitsById,
		scope,
		req.RootNodeId,
		req.RootOrgUnitId,
		strings.TrimSpace(req.Keyword),
	)
}

func normalizeLegalEntityReadScope(
	req request.OrgLegalEntityReadScopeReq,
) (repository.OrgLegalEntityReadScope, error) {
	return normalizeOrganizationReadScope(req)
}

func normalizeAssignmentReadScope(
	rawScope string,
	rawAsOfDate string,
) (repository.OrgAssignmentReadScope, error) {
	timeScope := strings.TrimSpace(rawScope)
	if timeScope == "" {
		timeScope = request.OrgAssignmentScopeCurrent
	}
	switch timeScope {
	case request.OrgAssignmentScopeCurrent,
		request.OrgAssignmentScopeHistory,
		request.OrgAssignmentScopeFuture,
		request.OrgAssignmentScopeTimeline:
	default:
		return repository.OrgAssignmentReadScope{},
			myerrors.NewParameterError("time_scope取值不合法")
	}

	asOf := model.Now()
	if raw := strings.TrimSpace(rawAsOfDate); raw != "" {
		parsed, err := time.ParseInLocation(orgAsOfDateLayout, raw, model.AppLocation())
		if err != nil {
			return repository.OrgAssignmentReadScope{}, myerrors.WrapParameterError(
				err,
				"as_of_date格式必须为YYYY-MM-DD",
			)
		}
		asOf = parsed
	}
	return repository.OrgAssignmentReadScope{AsOf: asOf, TimeScope: timeScope}, nil
}

func normalizeOrganizationReadScope(
	req request.OrgReadScopeReq,
) (repository.OrgReadScope, error) {
	onlyEffective := true
	if req.OnlyEffective != nil {
		onlyEffective = *req.OnlyEffective
	}

	asOf := model.Now()
	if raw := strings.TrimSpace(req.AsOfDate); raw != "" {
		parsed, err := time.ParseInLocation(orgAsOfDateLayout, raw, model.AppLocation())
		if err != nil {
			return repository.OrgReadScope{}, myerrors.WrapParameterError(
				err,
				"as_of_date格式必须为YYYY-MM-DD",
			)
		}
		asOf = parsed
	}

	scope := repository.OrgReadScope{
		AsOf:            asOf,
		IncludeDisabled: req.IncludeDisabled,
		IncludeHistory:  req.IncludeHistory,
	}
	if !onlyEffective {
		scope.IncludeDisabled = true
		scope.IncludeHistory = true
	}
	return scope, nil
}

func classifyAssignmentTimeScope(
	assignment model.OrgAssignment,
	asOf time.Time,
) string {
	if asOf.IsZero() {
		asOf = model.Now()
	}
	if strings.TrimSpace(assignment.Status) == "enabled" && !assignment.SourceDeleted {
		if assignment.ValidFrom != nil && assignment.ValidFrom.After(asOf) {
			return request.OrgAssignmentScopeFuture
		}
		if organizationDateEffective(assignment.ValidFrom, assignment.ValidTo, asOf) {
			return request.OrgAssignmentScopeCurrent
		}
	}
	return request.OrgAssignmentScopeHistory
}

func (s *OrgService) ensureEmployeeExists(ctx *gin.Context, employeeId int) error {
	_, err := s.employeeRepo.FindByIdForRead(ctx, employeeId)
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrOrgEmployeeNotFound
	}
	return myerrors.WrapDatabaseError(err)
}

func (s *OrgService) attachAssignmentReferences(
	ctx *gin.Context,
	assignments []response.OrgAssignmentListRes,
) error {
	legalEntityIds := make([]int, 0, len(assignments))
	orgUnitIds := make([]int, 0, len(assignments))
	positionIds := make([]int, 0, len(assignments))
	legalSeen := make(map[int]struct{}, len(assignments))
	unitSeen := make(map[int]struct{}, len(assignments))
	positionSeen := make(map[int]struct{}, len(assignments))
	for _, assignment := range assignments {
		if _, ok := legalSeen[assignment.LegalEntityId]; !ok {
			legalSeen[assignment.LegalEntityId] = struct{}{}
			legalEntityIds = append(legalEntityIds, assignment.LegalEntityId)
		}
		if _, ok := unitSeen[assignment.OrgUnitId]; !ok {
			unitSeen[assignment.OrgUnitId] = struct{}{}
			orgUnitIds = append(orgUnitIds, assignment.OrgUnitId)
		}
		if assignment.PositionId != nil {
			if _, ok := positionSeen[*assignment.PositionId]; !ok {
				positionSeen[*assignment.PositionId] = struct{}{}
				positionIds = append(positionIds, *assignment.PositionId)
			}
		}
	}

	legalEntities, err := s.legalEntityRepo.FindByIdsForDisplay(ctx, legalEntityIds)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	orgUnits, err := s.orgUnitRepo.FindByIdsForDisplay(ctx, orgUnitIds)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	positions, err := s.positionRepo.FindByIdsForDisplay(ctx, positionIds)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}

	legalById := make(map[int]response.OrgReferenceSummaryRes, len(legalEntities))
	for _, entity := range legalEntities {
		legalById[entity.Id] = response.NewOrgReferenceSummaryRes(
			entity.Id,
			entity.Code,
			entity.Name,
		)
	}
	unitById := make(map[int]response.OrgReferenceSummaryRes, len(orgUnits))
	for _, unit := range orgUnits {
		unitById[unit.Id] = response.NewOrgReferenceSummaryRes(unit.Id, unit.Code, unit.Name)
	}
	positionById := make(map[int]response.OrgReferenceSummaryRes, len(positions))
	for _, position := range positions {
		positionById[position.Id] = response.NewOrgReferenceSummaryRes(
			position.Id,
			position.Code,
			position.Name,
		)
	}

	for index := range assignments {
		legalEntity := referenceSummaryPointer(legalById, assignments[index].LegalEntityId)
		orgUnit := referenceSummaryPointer(unitById, assignments[index].OrgUnitId)
		var position *response.OrgReferenceSummaryRes
		if assignments[index].PositionId != nil {
			position = referenceSummaryPointer(positionById, *assignments[index].PositionId)
		}
		assignments[index].SetReferences(legalEntity, orgUnit, position)
	}
	return nil
}

func referenceSummaryPointer(
	summaries map[int]response.OrgReferenceSummaryRes,
	id int,
) *response.OrgReferenceSummaryRes {
	summary, exists := summaries[id]
	if !exists {
		return nil
	}
	return &summary
}

func collectAssignmentReferenceSummaries(
	assignments []response.OrgAssignmentListRes,
) (
	[]response.OrgReferenceSummaryRes,
	[]response.OrgReferenceSummaryRes,
	[]response.OrgReferenceSummaryRes,
) {
	legalEntities := make(map[int]response.OrgReferenceSummaryRes, len(assignments))
	orgUnits := make(map[int]response.OrgReferenceSummaryRes, len(assignments))
	positions := make(map[int]response.OrgReferenceSummaryRes, len(assignments))
	for _, assignment := range assignments {
		if assignment.LegalEntity != nil {
			legalEntities[assignment.LegalEntity.Id] = *assignment.LegalEntity
		}
		if assignment.OrgUnit != nil {
			orgUnits[assignment.OrgUnit.Id] = *assignment.OrgUnit
		}
		if assignment.Position != nil {
			positions[assignment.Position.Id] = *assignment.Position
		}
	}
	return sortedReferenceSummaries(legalEntities),
		sortedReferenceSummaries(orgUnits),
		sortedReferenceSummaries(positions)
}

func sortedReferenceSummaries(
	summaries map[int]response.OrgReferenceSummaryRes,
) []response.OrgReferenceSummaryRes {
	result := make([]response.OrgReferenceSummaryRes, 0, len(summaries))
	for _, summary := range summaries {
		result = append(result, summary)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Code != result[right].Code {
			return result[left].Code < result[right].Code
		}
		if result[left].Name != result[right].Name {
			return result[left].Name < result[right].Name
		}
		return result[left].Id < result[right].Id
	})
	return result
}

func legalEntityVisible(entity model.OrgLegalEntity, scope repository.OrgLegalEntityReadScope) bool {
	if !scope.IncludeDisabled && strings.TrimSpace(entity.Status) != "enabled" {
		return false
	}
	if scope.IncludeHistory {
		return true
	}
	if entity.SourceDeleted {
		return false
	}
	return legalEntityDateEffective(entity, scope.AsOf)
}

func isLegalEntityEffective(entity model.OrgLegalEntity, asOf time.Time) bool {
	return strings.TrimSpace(entity.Status) == "enabled" &&
		!entity.SourceDeleted &&
		legalEntityDateEffective(entity, asOf)
}

func legalEntityDateEffective(entity model.OrgLegalEntity, asOf time.Time) bool {
	return organizationDateEffective(entity.ValidFrom, entity.ValidTo, asOf)
}

func orgStructureVisible(structure model.OrgStructure, scope repository.OrgReadScope) bool {
	if !scope.IncludeDisabled && strings.TrimSpace(structure.Status) != "enabled" {
		return false
	}
	if scope.IncludeHistory {
		return true
	}
	return organizationDateEffective(structure.ValidFrom, structure.ValidTo, scope.AsOf)
}

func isOrgStructureEffective(structure model.OrgStructure, asOf time.Time) bool {
	return strings.TrimSpace(structure.Status) == "enabled" &&
		organizationDateEffective(structure.ValidFrom, structure.ValidTo, asOf)
}

func orgUnitVisible(unit model.OrgUnit, scope repository.OrgReadScope) bool {
	if !scope.IncludeDisabled && strings.TrimSpace(unit.Status) != "enabled" {
		return false
	}
	if scope.IncludeHistory {
		return true
	}
	return !unit.SourceDeleted &&
		organizationDateEffective(unit.ValidFrom, unit.ValidTo, scope.AsOf)
}

func isOrgUnitEffective(unit model.OrgUnit, asOf time.Time) bool {
	return strings.TrimSpace(unit.Status) == "enabled" &&
		!unit.SourceDeleted &&
		organizationDateEffective(unit.ValidFrom, unit.ValidTo, asOf)
}

func isOrgStructureNodeEffective(node model.OrgStructureNode, asOf time.Time) bool {
	return strings.TrimSpace(node.Status) == "enabled" &&
		!node.SourceDeleted &&
		organizationDateEffective(node.ValidFrom, node.ValidTo, asOf)
}

func orgEmployeeVisible(employee model.OrgEmployee, scope repository.OrgReadScope) bool {
	if !scope.IncludeDisabled && !isCurrentEmploymentStatus(employee.EmploymentStatus) {
		return false
	}
	if scope.IncludeHistory {
		return true
	}
	return !employee.SourceDeleted &&
		organizationDateEffective(employee.ValidFrom, employee.ValidTo, scope.AsOf)
}

func isOrgEmployeeEffective(employee model.OrgEmployee, asOf time.Time) bool {
	return isCurrentEmploymentStatus(employee.EmploymentStatus) &&
		!employee.SourceDeleted &&
		organizationDateEffective(employee.ValidFrom, employee.ValidTo, asOf)
}

func isCurrentEmploymentStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "active", "probation":
		return true
	default:
		return false
	}
}

func orgPositionVisible(position model.OrgPosition, scope repository.OrgReadScope) bool {
	if !scope.IncludeDisabled && strings.TrimSpace(position.Status) != "enabled" {
		return false
	}
	if scope.IncludeHistory {
		return true
	}
	return !position.SourceDeleted &&
		organizationDateEffective(position.ValidFrom, position.ValidTo, scope.AsOf)
}

func isOrgPositionEffective(position model.OrgPosition, asOf time.Time) bool {
	return strings.TrimSpace(position.Status) == "enabled" &&
		!position.SourceDeleted &&
		organizationDateEffective(position.ValidFrom, position.ValidTo, asOf)
}

func organizationDateEffective(validFrom, validTo *time.Time, asOf time.Time) bool {
	if asOf.IsZero() {
		asOf = model.Now()
	}
	if validFrom != nil && validFrom.After(asOf) {
		return false
	}
	if validTo != nil && validTo.Before(asOf) {
		return false
	}
	return true
}

func normalizeOrgUnitLegalEntityFilter(req *request.OrgUnitQueryReq) error {
	if req == nil {
		return nil
	}
	if req.LegalEntityId != nil && req.PrimaryLegalEntityId != nil &&
		*req.LegalEntityId != *req.PrimaryLegalEntityId {
		return myerrors.NewParameterError(
			"legal_entity_id与primary_legal_entity_id不能指定不同值",
		)
	}
	if req.LegalEntityId != nil {
		req.PrimaryLegalEntityId = req.LegalEntityId
	}
	return nil
}

func normalizeEmployeeBoundStatus(req *request.OrgEmployeeQueryReq) error {
	if req == nil {
		return nil
	}
	switch req.BoundStatus {
	case "", "all", orgEmployeeBound, orgEmployeeUnbound:
		return nil
	default:
		return myerrors.NewParameterError("bound_status取值不合法")
	}
}

func (s *OrgService) attachEmployeeAccountSummaries(
	ctx *gin.Context,
	employees []response.OrgEmployeeListRes,
) error {
	userIds := make([]int, 0, len(employees))
	seen := make(map[int]struct{}, len(employees))
	for _, employee := range employees {
		if employee.BoundUserId == nil {
			continue
		}
		if _, exists := seen[*employee.BoundUserId]; exists {
			continue
		}
		seen[*employee.BoundUserId] = struct{}{}
		userIds = append(userIds, *employee.BoundUserId)
	}
	if len(userIds) == 0 {
		return nil
	}
	users, err := s.employeeRepo.FindBoundUserSummaries(ctx, userIds)
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	usersById := make(map[int]repository.OrgBoundUserSummary, len(users))
	for _, user := range users {
		usersById[user.UserId] = user
	}
	for index := range employees {
		if employees[index].BoundUserId == nil {
			continue
		}
		user, exists := usersById[*employees[index].BoundUserId]
		if !exists {
			continue
		}
		employees[index].SetBoundAccount(
			response.NewOrgBoundUserSummaryRes(user.UserId, user.UserName),
		)
	}
	return nil
}

func uniqueStructureNodeUnitIds(nodes []model.OrgStructureNode) []int {
	result := make([]int, 0, len(nodes))
	seen := make(map[int]struct{}, len(nodes))
	for _, node := range nodes {
		if _, exists := seen[node.OrgUnitId]; exists {
			continue
		}
		seen[node.OrgUnitId] = struct{}{}
		result = append(result, node.OrgUnitId)
	}
	return result
}

func buildStructureOrgTree(
	structure model.OrgStructure,
	nodes []model.OrgStructureNode,
	unitsById map[int]model.OrgUnit,
	scope repository.OrgReadScope,
	rootNodeId *int,
	rootOrgUnitId *int,
	keyword string,
) ([]response.OrgStructureOrgTreeNodeRes, error) {
	nodesById := make(map[int]model.OrgStructureNode, len(nodes))
	for _, node := range nodes {
		nodesById[node.Id] = node
	}
	if structureNodeTreeHasCycle(nodesById) {
		return nil, myerrors.ErrOrgStructureCycle
	}

	resolvedRootId := 0
	if rootNodeId != nil {
		if _, exists := nodesById[*rootNodeId]; !exists {
			return nil, myerrors.ErrOrgStructureNodeMissing
		}
		resolvedRootId = *rootNodeId
	}
	if rootOrgUnitId != nil {
		matches := make([]int, 0, 1)
		for _, node := range nodes {
			if node.OrgUnitId == *rootOrgUnitId {
				matches = append(matches, node.Id)
			}
		}
		switch len(matches) {
		case 0:
			return nil, myerrors.ErrOrgStructureNodeMissing
		case 1:
			resolvedRootId = matches[0]
		default:
			return nil, myerrors.ErrOrgTreeRootAmbiguous
		}
	}

	allChildren := make(map[int][]int, len(nodes))
	for _, node := range nodes {
		if node.ParentNodeId != nil {
			allChildren[*node.ParentNodeId] = append(allChildren[*node.ParentNodeId], node.Id)
		}
	}

	selected := make(map[int]struct{}, len(nodes))
	if resolvedRootId != 0 {
		stack := []int{resolvedRootId}
		for len(stack) > 0 {
			index := len(stack) - 1
			id := stack[index]
			stack = stack[:index]
			if _, exists := selected[id]; exists {
				continue
			}
			selected[id] = struct{}{}
			stack = append(stack, allChildren[id]...)
		}
	} else {
		for id := range nodesById {
			selected[id] = struct{}{}
		}
	}

	if keyword != "" {
		normalizedKeyword := strings.ToLower(keyword)
		matchedWithAncestors := make(map[int]struct{})
		for id := range selected {
			unit := unitsById[nodesById[id].OrgUnitId]
			if !structureOrgTreeKeywordMatch(unit, normalizedKeyword) {
				continue
			}
			currentId := id
			for {
				if _, allowed := selected[currentId]; !allowed {
					break
				}
				matchedWithAncestors[currentId] = struct{}{}
				parentId := nodesById[currentId].ParentNodeId
				if parentId == nil {
					break
				}
				currentId = *parentId
			}
		}
		selected = matchedWithAncestors
	}

	children := make(map[int][]int, len(selected))
	roots := make([]int, 0)
	orphanIds := make([]int, 0)
	for id := range selected {
		node := nodesById[id]
		if node.ParentNodeId == nil {
			roots = append(roots, id)
			continue
		}
		if _, parentSelected := selected[*node.ParentNodeId]; parentSelected {
			children[*node.ParentNodeId] = append(children[*node.ParentNodeId], id)
			continue
		}
		roots = append(roots, id)
		if id != resolvedRootId {
			if _, parentVisible := nodesById[*node.ParentNodeId]; !parentVisible {
				orphanIds = append(orphanIds, id)
			}
		}
	}

	sortStructureNodeIds(roots, nodesById, unitsById)
	for parentId := range children {
		sortStructureNodeIds(children[parentId], nodesById, unitsById)
	}
	orphanSet := make(map[int]struct{}, len(orphanIds))
	if len(orphanIds) > 0 {
		sort.Ints(orphanIds)
		zap.L().Warn(
			"management organization tree contains orphan nodes",
			zap.Int("structure_id", structure.Id),
			zap.Ints("structure_node_ids", orphanIds),
		)
		for _, id := range orphanIds {
			orphanSet[id] = struct{}{}
		}
	}

	structureEffective := isOrgStructureEffective(structure, scope.AsOf)
	var buildNode func(int) response.OrgStructureOrgTreeNodeRes
	buildNode = func(id int) response.OrgStructureOrgTreeNodeRes {
		sourceNode := nodesById[id]
		unit := unitsById[sourceNode.OrgUnitId]
		node := response.NewOrgStructureOrgTreeNodeRes(
			sourceNode,
			unit,
			!structureEffective ||
				!isOrgStructureNodeEffective(sourceNode, scope.AsOf) ||
				!isOrgUnitEffective(unit, scope.AsOf),
		)
		_, node.Orphan = orphanSet[id]
		if childIds := children[id]; len(childIds) > 0 {
			node.Children = make([]response.OrgStructureOrgTreeNodeRes, 0, len(childIds))
			for _, childId := range childIds {
				node.Children = append(node.Children, buildNode(childId))
			}
		}
		return node
	}

	result := make([]response.OrgStructureOrgTreeNodeRes, 0, len(roots))
	for _, id := range roots {
		result = append(result, buildNode(id))
	}
	return result, nil
}

func structureNodeTreeHasCycle(nodes map[int]model.OrgStructureNode) bool {
	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[int]int, len(nodes))
	var visit func(int) bool
	visit = func(id int) bool {
		switch state[id] {
		case visiting:
			return true
		case visited:
			return false
		}
		state[id] = visiting
		node := nodes[id]
		if node.ParentNodeId != nil {
			if _, exists := nodes[*node.ParentNodeId]; exists && visit(*node.ParentNodeId) {
				return true
			}
		}
		state[id] = visited
		return false
	}
	for id := range nodes {
		if state[id] == unvisited && visit(id) {
			return true
		}
	}
	return false
}

func structureOrgTreeKeywordMatch(unit model.OrgUnit, keyword string) bool {
	return strings.Contains(strings.ToLower(unit.Code), keyword) ||
		strings.Contains(strings.ToLower(unit.Name), keyword)
}

func sortStructureNodeIds(
	ids []int,
	nodes map[int]model.OrgStructureNode,
	units map[int]model.OrgUnit,
) {
	sort.SliceStable(ids, func(left, right int) bool {
		leftNode := nodes[ids[left]]
		rightNode := nodes[ids[right]]
		leftUnit := units[leftNode.OrgUnitId]
		rightUnit := units[rightNode.OrgUnitId]
		switch {
		case leftNode.Sort != rightNode.Sort:
			return leftNode.Sort < rightNode.Sort
		case leftUnit.Code != rightUnit.Code:
			return leftUnit.Code < rightUnit.Code
		default:
			return leftNode.Id < rightNode.Id
		}
	})
}

func buildLegalEntityTree(
	entities []model.OrgLegalEntity,
	scope repository.OrgLegalEntityReadScope,
	rootId *int,
) ([]response.OrgLegalEntityTreeNodeRes, error) {
	byId := make(map[int]model.OrgLegalEntity, len(entities))
	for _, entity := range entities {
		byId[entity.Id] = entity
	}
	if rootId != nil {
		if _, exists := byId[*rootId]; !exists {
			return nil, myerrors.ErrOrgLegalEntityNotFound
		}
	}
	if legalEntityTreeHasCycle(byId) {
		return nil, myerrors.ErrOrgLegalEntityCycle
	}

	children := make(map[int][]int)
	roots := make([]int, 0)
	orphanIds := make([]int, 0)
	for id, entity := range byId {
		if entity.ParentId == nil {
			roots = append(roots, id)
			continue
		}
		if _, parentExists := byId[*entity.ParentId]; !parentExists {
			roots = append(roots, id)
			orphanIds = append(orphanIds, id)
			continue
		}
		children[*entity.ParentId] = append(children[*entity.ParentId], id)
	}

	sortLegalEntityIds(roots, byId)
	for parentId := range children {
		sortLegalEntityIds(children[parentId], byId)
	}
	if len(orphanIds) > 0 {
		sort.Ints(orphanIds)
		zap.L().Warn(
			"legal entity tree contains orphan nodes",
			zap.Ints("legal_entity_ids", orphanIds),
		)
	}

	orphanSet := make(map[int]struct{}, len(orphanIds))
	for _, id := range orphanIds {
		orphanSet[id] = struct{}{}
	}
	buildNode := func(id int, recurse func(int) response.OrgLegalEntityTreeNodeRes) response.OrgLegalEntityTreeNodeRes {
		entity := byId[id]
		node := response.NewOrgLegalEntityTreeNodeRes(
			entity,
			!isLegalEntityEffective(entity, scope.AsOf),
		)
		_, node.Orphan = orphanSet[id]
		childIds := children[id]
		if len(childIds) > 0 {
			node.Children = make([]response.OrgLegalEntityTreeNodeRes, 0, len(childIds))
			for _, childId := range childIds {
				node.Children = append(node.Children, recurse(childId))
			}
		}
		return node
	}
	var recurse func(int) response.OrgLegalEntityTreeNodeRes
	recurse = func(id int) response.OrgLegalEntityTreeNodeRes {
		return buildNode(id, recurse)
	}

	if rootId != nil {
		return []response.OrgLegalEntityTreeNodeRes{recurse(*rootId)}, nil
	}
	result := make([]response.OrgLegalEntityTreeNodeRes, 0, len(roots))
	for _, id := range roots {
		result = append(result, recurse(id))
	}
	return result, nil
}

func legalEntityTreeHasCycle(entities map[int]model.OrgLegalEntity) bool {
	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[int]int, len(entities))
	var visit func(int) bool
	visit = func(id int) bool {
		switch state[id] {
		case visiting:
			return true
		case visited:
			return false
		}
		state[id] = visiting
		entity := entities[id]
		if entity.ParentId != nil {
			if _, exists := entities[*entity.ParentId]; exists && visit(*entity.ParentId) {
				return true
			}
		}
		state[id] = visited
		return false
	}
	for id := range entities {
		if state[id] == unvisited && visit(id) {
			return true
		}
	}
	return false
}

func sortLegalEntityIds(ids []int, entities map[int]model.OrgLegalEntity) {
	sort.SliceStable(ids, func(left, right int) bool {
		a := entities[ids[left]]
		b := entities[ids[right]]
		switch {
		case a.DisplayOrder != nil && b.DisplayOrder == nil:
			return true
		case a.DisplayOrder == nil && b.DisplayOrder != nil:
			return false
		case a.DisplayOrder != nil && b.DisplayOrder != nil && *a.DisplayOrder != *b.DisplayOrder:
			return *a.DisplayOrder < *b.DisplayOrder
		case a.Code != b.Code:
			return a.Code < b.Code
		default:
			return a.Id < b.Id
		}
	})
}

func normalizeLegalEntitySelectedIds(ids []int) ([]int, error) {
	return normalizeOrganizationSelectedIds(ids)
}

func normalizeOrganizationSelectedIds(ids []int) ([]int, error) {
	if len(ids) > 100 {
		return nil, myerrors.NewParameterError("selected_ids最多支持100项")
	}
	result := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, myerrors.NewParameterError("selected_ids必须为正整数")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func (s *OrgService) recordEmployeeUserBindingAudit(
	ctx *gin.Context,
	tx *gorm.DB,
	action string,
	employeeId int,
	oldUserId *int,
	newUserId *int,
) error {
	if s.auditWriter == nil {
		return myerrors.WrapSystemError(ErrTransactionalAuditRepositoryRequired)
	}
	err := s.auditWriter.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: "org_employee",
		ResourceCode: "org_employee",
		ResourceId:   strconv.Itoa(employeeId),
		Changes: map[string]TransactionalAuditChange{
			"user_id": {
				OldValue: optionalAuditInt(oldUserId),
				NewValue: optionalAuditInt(newUserId),
			},
		},
	})
	if err != nil {
		return myerrors.WrapDatabaseError(err)
	}
	return nil
}

func cloneOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func optionalAuditInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func isEmployeeUserBindingUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) &&
		pgError.Code == "23505" &&
		pgError.ConstraintName == "uni_org_employee_user"
}
