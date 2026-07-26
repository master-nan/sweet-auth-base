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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	orgAsOfDateLayout            = time.DateOnly
	orgStructureTreeMaxNodeCount = 5000
)

// OrgService is the public read boundary for Organization Master Data. Other
// modules call this service instead of reading Organization repositories.
type OrgService struct {
	legalEntityRepo   repository.OrgLegalEntityRepository
	orgUnitRepo       repository.OrgUnitRepository
	structureRepo     repository.OrgStructureRepository
	structureNodeRepo repository.OrgStructureNodeRepository
}

func NewOrgService(
	legalEntityRepo repository.OrgLegalEntityRepository,
	orgUnitRepo repository.OrgUnitRepository,
	structureRepo repository.OrgStructureRepository,
	structureNodeRepo repository.OrgStructureNodeRepository,
) *OrgService {
	return &OrgService{
		legalEntityRepo:   legalEntityRepo,
		orgUnitRepo:       orgUnitRepo,
		structureRepo:     structureRepo,
		structureNodeRepo: structureNodeRepo,
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
) (response.ListResult[response.OrgSelectorOptionRes], error) {
	var result response.ListResult[response.OrgSelectorOptionRes]
	if err := utils.ValidatePagination(req.Page, req.Num); err != nil {
		return result, err
	}
	scope, err := normalizeLegalEntityReadScope(req.OrgLegalEntityReadScopeReq)
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
		OrgLegalEntityReadScopeReq: req.OrgLegalEntityReadScopeReq,
	}
	table.TableCode = "org_legal_entity"
	rows, err := s.legalEntityRepo.Query(ctx, &queryReq, table, scope)
	if err != nil {
		return result, myerrors.WrapDatabaseError(err)
	}

	result.Total = rows.Total
	result.Data = make([]response.OrgSelectorOptionRes, 0, len(rows.Data)+len(selectedIds))
	seen := make(map[int]struct{}, len(rows.Data)+len(selectedIds))
	for _, entity := range rows.Data {
		result.Data = append(
			result.Data,
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
		result.Data = append(
			result.Data,
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
	result.Data = make([]response.OrgSelectorOptionRes, 0, len(rows.Data)+len(selectedIds))
	seen := make(map[int]struct{}, len(rows.Data)+len(selectedIds))
	for _, unit := range rows.Data {
		result.Data = append(
			result.Data,
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
		result.Data = append(
			result.Data,
			response.NewOrgUnitOptionRes(unit, !isOrgUnitEffective(unit, scope.AsOf)),
		)
		seen[id] = struct{}{}
	}
	return result, nil
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
