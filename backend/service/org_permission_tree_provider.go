package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type orgPermissionTreeIndex struct {
	parentByOrgUnit   map[int]int
	childrenByOrgUnit map[int][]int
}

func (s *OrgService) GetOrgAncestors(
	ctx context.Context,
	structureCode string,
	orgUnitId int,
	asOfDate string,
	includeSelf bool,
) (response.OrgAncestorsRes, error) {
	index, asOf, normalizedCode, err := s.loadPermissionTreeIndex(
		ctx,
		structureCode,
		[]int{orgUnitId},
		asOfDate,
	)
	if err != nil {
		return response.OrgAncestorsRes{}, err
	}

	items := make([]response.OrgRelationItemRes, 0)
	if includeSelf {
		items = append(items, response.OrgRelationItemRes{
			OrgUnitId: orgUnitId,
			Distance:  0,
		})
	}
	current := orgUnitId
	distance := 0
	for {
		parent, exists := index.parentByOrgUnit[current]
		if !exists {
			break
		}
		distance++
		items = append(items, response.OrgRelationItemRes{
			OrgUnitId: parent,
			Distance:  distance,
		})
		current = parent
	}
	sortOrgRelationItems(items)
	return response.OrgAncestorsRes{
		StructureCode: normalizedCode,
		OrgUnitId:     orgUnitId,
		AsOfDate:      formatOrganizationProviderDate(asOf),
		Items:         items,
	}, nil
}

func (s *OrgService) GetOrgDescendants(
	ctx context.Context,
	structureCode string,
	orgUnitId int,
	asOfDate string,
	includeSelf bool,
) (response.OrgDescendantsRes, error) {
	index, asOf, normalizedCode, err := s.loadPermissionTreeIndex(
		ctx,
		structureCode,
		[]int{orgUnitId},
		asOfDate,
	)
	if err != nil {
		return response.OrgDescendantsRes{}, err
	}

	items := make([]response.OrgRelationItemRes, 0)
	if includeSelf {
		items = append(items, response.OrgRelationItemRes{
			OrgUnitId: orgUnitId,
			Distance:  0,
		})
	}
	type queueItem struct {
		orgUnitId int
		distance  int
	}
	queue := make([]queueItem, 0, len(index.childrenByOrgUnit[orgUnitId]))
	for _, childId := range index.childrenByOrgUnit[orgUnitId] {
		queue = append(queue, queueItem{orgUnitId: childId, distance: 1})
	}
	for head := 0; head < len(queue); head++ {
		item := queue[head]
		items = append(items, response.OrgRelationItemRes{
			OrgUnitId: item.orgUnitId,
			Distance:  item.distance,
		})
		if len(items) > orgStructureTreeMaxNodeCount {
			return response.OrgDescendantsRes{}, myerrors.ErrOrgTreeTooLarge
		}
		for _, childId := range index.childrenByOrgUnit[item.orgUnitId] {
			queue = append(queue, queueItem{
				orgUnitId: childId,
				distance:  item.distance + 1,
			})
		}
	}
	sortOrgRelationItems(items)
	return response.OrgDescendantsRes{
		StructureCode: normalizedCode,
		OrgUnitId:     orgUnitId,
		AsOfDate:      formatOrganizationProviderDate(asOf),
		Items:         items,
	}, nil
}

func (s *OrgService) IsOrgDescendant(
	ctx context.Context,
	structureCode string,
	ancestorOrgUnitId int,
	descendantOrgUnitId int,
	asOfDate string,
	includeSelf bool,
) (response.OrgDescendantCheckRes, error) {
	index, asOf, normalizedCode, err := s.loadPermissionTreeIndex(
		ctx,
		structureCode,
		[]int{ancestorOrgUnitId, descendantOrgUnitId},
		asOfDate,
	)
	if err != nil {
		return response.OrgDescendantCheckRes{}, err
	}
	result := response.OrgDescendantCheckRes{
		StructureCode:       normalizedCode,
		AncestorOrgUnitId:   ancestorOrgUnitId,
		DescendantOrgUnitId: descendantOrgUnitId,
		AsOfDate:            formatOrganizationProviderDate(asOf),
	}
	if ancestorOrgUnitId == descendantOrgUnitId {
		if includeSelf {
			distance := 0
			result.IsDescendant = true
			result.Distance = &distance
		}
		return result, nil
	}

	current := descendantOrgUnitId
	distance := 0
	for {
		parent, exists := index.parentByOrgUnit[current]
		if !exists {
			return result, nil
		}
		distance++
		if parent == ancestorOrgUnitId {
			result.IsDescendant = true
			result.Distance = &distance
			return result, nil
		}
		current = parent
	}
}

func (s *OrgService) loadPermissionTreeIndex(
	ctx context.Context,
	structureCode string,
	requiredOrgUnitIds []int,
	asOfDate string,
) (*orgPermissionTreeIndex, time.Time, string, error) {
	normalizedCode := strings.TrimSpace(structureCode)
	if normalizedCode == "" {
		return nil, time.Time{}, "", myerrors.NewParameterError(
			"structure_code不能为空",
		)
	}
	asOf, err := normalizeRequiredOrganizationDate(asOfDate)
	if err != nil {
		return nil, time.Time{}, "", err
	}
	for _, orgUnitId := range requiredOrgUnitIds {
		if orgUnitId <= 0 {
			return nil, time.Time{}, "", myerrors.NewParameterError(
				"org_unit_id必须大于0",
			)
		}
	}

	structure, err := s.structureRepo.FindByCode(ctx, normalizedCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, time.Time{}, "", myerrors.ErrOrgStructureNotFound
		}
		return nil, time.Time{}, "", myerrors.WrapDatabaseError(err)
	}
	if !isOrgStructureEffective(structure, asOf) {
		return nil, time.Time{}, "", myerrors.ErrOrgStructureInactive
	}
	for _, orgUnitId := range uniquePositiveOrganizationIds(requiredOrgUnitIds) {
		if err = s.validatePermissionOrgUnit(ctx, orgUnitId, asOf); err != nil {
			return nil, time.Time{}, "", err
		}
	}

	tree, err := s.getStructureOrgTreeForRead(
		ctx,
		request.OrgStructureOrgTreeReq{StructureId: structure.Id},
		repository.OrgReadScope{AsOf: asOf},
	)
	if err != nil {
		return nil, time.Time{}, "", err
	}
	index, err := buildPermissionTreeIndex(tree)
	if err != nil {
		return nil, time.Time{}, "", err
	}
	for _, orgUnitId := range uniquePositiveOrganizationIds(requiredOrgUnitIds) {
		if _, exists := index.parentByOrgUnit[orgUnitId]; exists {
			continue
		}
		if _, exists := index.childrenByOrgUnit[orgUnitId]; !exists {
			return nil, time.Time{}, "", myerrors.ErrOrgStructureMembershipNotFound
		}
	}
	return index, asOf, normalizedCode, nil
}

func (s *OrgService) validatePermissionOrgUnit(
	ctx context.Context,
	orgUnitId int,
	asOf time.Time,
) error {
	if orgUnitId <= 0 {
		return myerrors.NewParameterError("org_unit_id必须大于0")
	}
	unit, err := s.orgUnitRepo.FindByIdForRead(ctx, orgUnitId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrOrgUnitNotFound
		}
		return myerrors.WrapDatabaseError(err)
	}
	if !isOrgUnitEffective(unit, asOf) {
		return myerrors.ErrOrgUnitInactive
	}
	return nil
}

func buildPermissionTreeIndex(
	trees []response.OrgStructureOrgTreeNodeRes,
) (*orgPermissionTreeIndex, error) {
	index := &orgPermissionTreeIndex{
		parentByOrgUnit:   make(map[int]int),
		childrenByOrgUnit: make(map[int][]int),
	}
	seen := make(map[int]struct{})
	var walk func([]response.OrgStructureOrgTreeNodeRes, *int) error
	walk = func(nodes []response.OrgStructureOrgTreeNodeRes, parentOrgUnitId *int) error {
		for _, node := range nodes {
			if node.Orphan {
				return myerrors.ErrOrgStructureNodeMissing
			}
			if _, exists := seen[node.OrgUnitId]; exists {
				return myerrors.ErrOrgStructureMembershipAmbiguous
			}
			seen[node.OrgUnitId] = struct{}{}
			if _, exists := index.childrenByOrgUnit[node.OrgUnitId]; !exists {
				index.childrenByOrgUnit[node.OrgUnitId] = []int{}
			}
			if parentOrgUnitId != nil {
				index.parentByOrgUnit[node.OrgUnitId] = *parentOrgUnitId
				index.childrenByOrgUnit[*parentOrgUnitId] = append(
					index.childrenByOrgUnit[*parentOrgUnitId],
					node.OrgUnitId,
				)
			}
			current := node.OrgUnitId
			if err := walk(node.Children, &current); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(trees, nil); err != nil {
		return nil, err
	}
	for parentId := range index.childrenByOrgUnit {
		sort.Ints(index.childrenByOrgUnit[parentId])
	}
	return index, nil
}

func sortOrgRelationItems(items []response.OrgRelationItemRes) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Distance != items[j].Distance {
			return items[i].Distance < items[j].Distance
		}
		return items[i].OrgUnitId < items[j].OrgUnitId
	})
}

func uniquePositiveOrganizationIds(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func formatOrganizationProviderDate(value time.Time) string {
	return value.In(model.AppLocation()).Format(orgAsOfDateLayout)
}
