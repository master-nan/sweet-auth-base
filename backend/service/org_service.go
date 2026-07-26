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

const orgAsOfDateLayout = time.DateOnly

// OrgService is the public read boundary for Organization Master Data. Other
// modules call this service instead of reading Organization repositories.
type OrgService struct {
	legalEntityRepo repository.OrgLegalEntityRepository
}

func NewOrgService(legalEntityRepo repository.OrgLegalEntityRepository) *OrgService {
	return &OrgService{legalEntityRepo: legalEntityRepo}
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

func normalizeLegalEntityReadScope(
	req request.OrgLegalEntityReadScopeReq,
) (repository.OrgLegalEntityReadScope, error) {
	onlyEffective := true
	if req.OnlyEffective != nil {
		onlyEffective = *req.OnlyEffective
	}

	asOf := model.Now()
	if raw := strings.TrimSpace(req.AsOfDate); raw != "" {
		parsed, err := time.ParseInLocation(orgAsOfDateLayout, raw, model.AppLocation())
		if err != nil {
			return repository.OrgLegalEntityReadScope{}, myerrors.WrapParameterError(
				err,
				"as_of_date格式必须为YYYY-MM-DD",
			)
		}
		asOf = parsed
	}

	scope := repository.OrgLegalEntityReadScope{
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
	if asOf.IsZero() {
		asOf = model.Now()
	}
	if entity.ValidFrom != nil && entity.ValidFrom.After(asOf) {
		return false
	}
	if entity.ValidTo != nil && entity.ValidTo.Before(asOf) {
		return false
	}
	return true
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
