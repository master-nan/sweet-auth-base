package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"
)

func TestOrgServiceStructureQueriesRespectMetadataScopeAndLegalEntity(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	legalA := orgServiceLegalEntity(1, "LE-A", "法人甲", "甲", "enabled", nil, nil)
	legalB := orgServiceLegalEntity(2, "LE-B", "法人乙", "乙", "enabled", nil, nil)
	testutil.MustCreate(t, db, &[]model.OrgLegalEntity{legalA, legalB})

	current := managementStructureFixture(10, "MGMT-A", "行政管理架构", "enabled")
	other := managementStructureFixture(11, "MGMT-B", "经营管理架构", "enabled")
	disabled := managementStructureFixture(12, "MGMT-D", "停用架构", "disabled")
	expired := managementStructureFixture(13, "MGMT-H", "历史架构", "enabled")
	expiredAt := model.Now().AddDate(0, 0, -1)
	expired.ValidTo = &expiredAt
	legalStructure := managementStructureFixture(14, "LEGAL-A", "法人架构", "enabled")
	legalStructure.StructureType = model.OrgStructureTypeLegal
	testutil.MustCreate(t, db, &[]model.OrgStructure{current, other, disabled, expired, legalStructure})

	unitA := managementUnitFixture(20, "OU-A", "甲组织", "department", "enabled", &legalA.Id)
	unitB := managementUnitFixture(21, "OU-B", "乙组织", "department", "enabled", &legalB.Id)
	testutil.MustCreate(t, db, &[]model.OrgUnit{unitA, unitB})
	nodes := []model.OrgStructureNode{
		managementNodeFixture(30, current.Id, unitA.Id, nil, "enabled"),
		managementNodeFixture(31, other.Id, unitB.Id, nil, "enabled"),
		managementNodeFixture(32, legalStructure.Id, unitA.Id, nil, "enabled"),
	}
	testutil.MustCreate(t, db, &nodes)

	result, err := orgService.QueryStructures(nil, request.OrgStructureQueryReq{
		Basic: request.Basic{
			Page: 1,
			Num:  10,
			QuickQuery: &request.QuickQuery{
				Keyword: "行政",
			},
		},
		LegalEntityId: &legalA.Id,
		StructureType: "management",
	}, managementStructureTable())
	if err != nil {
		t.Fatalf("query structures: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 1 || result.Data[0].Id != current.Id {
		t.Fatalf("unexpected structure query result: %+v", result)
	}
	legalStructures, err := orgService.QueryStructures(nil, request.OrgStructureQueryReq{StructureType: "legal"}, managementStructureTable())
	if err != nil || legalStructures.Total != 1 || legalStructures.Data[0].Id != legalStructure.Id {
		t.Fatalf("legal structure query=%+v err=%v", legalStructures, err)
	}
	if detail, err := orgService.GetStructureDetail(nil, legalStructure.Id, request.OrgStructureDetailReq{}); err != nil || detail.Id != legalStructure.Id {
		t.Fatalf("legal structure detail=%+v err=%v", detail, err)
	}
	legalTree, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{StructureId: legalStructure.Id})
	if err != nil || len(legalTree) != 1 || legalTree[0].PrimaryLegalEntityId == nil || *legalTree[0].PrimaryLegalEntityId != legalA.Id {
		t.Fatalf("legal structure tree=%+v err=%v", legalTree, err)
	}

	advanced, err := orgService.QueryStructures(nil, request.OrgStructureQueryReq{
		Basic: request.Basic{
			Expressions: []request.ExpressionGroup{{
				Logic: enum.And,
				Rules: []request.QueryRule{{
					Field:          "code",
					ExpressionType: enum.Like,
					Value:          "MGMT-B",
				}},
			}},
		},
	}, managementStructureTable())
	if err != nil || advanced.Total != 1 || advanced.Data[0].Id != other.Id {
		t.Fatalf("advanced structure query result=%+v err=%v", advanced, err)
	}

	withDisabled, err := orgService.QueryStructures(nil, request.OrgStructureQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeDisabled: true},
	}, managementStructureTable())
	if err != nil || withDisabled.Total != 3 {
		t.Fatalf("include_disabled result=%+v err=%v", withDisabled, err)
	}

	withHistory, err := orgService.QueryStructures(nil, request.OrgStructureQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeHistory: true},
	}, managementStructureTable())
	if err != nil || withHistory.Total != 3 {
		t.Fatalf("include_history result=%+v err=%v", withHistory, err)
	}

	detail, err := orgService.GetStructureDetail(nil, current.Id, request.OrgStructureDetailReq{})
	if err != nil || detail.Id != current.Id {
		t.Fatalf("get structure detail=%+v err=%v", detail, err)
	}
	assertOrganizationResponseDoesNotLeakSourceFields(t, detail)

	_, err = orgService.GetStructureDetail(nil, 999, request.OrgStructureDetailReq{})
	assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgStructureNotFound)

	options, err := orgService.QueryStructureOptions(nil, request.OrgStructureOptionsReq{
		Page:          1,
		Num:           10,
		Keyword:       "行政",
		LegalEntityId: &legalA.Id,
		SelectedIds:   []int{disabled.Id},
	}, managementStructureTable())
	if err != nil {
		t.Fatalf("query structure options: %v", err)
	}
	if options.Total != 1 || len(options.Data) != 2 ||
		options.Data[0].Value != current.Id ||
		options.Data[1].Value != disabled.Id ||
		!options.Data[1].Disabled {
		t.Fatalf("unexpected structure options: %+v", options)
	}
}

func TestOrgServiceOrgUnitQueriesAndOptionsUseBusinessIDs(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	legal := orgServiceLegalEntity(1, "LE-A", "法人甲", "甲", "enabled", nil, nil)
	testutil.MustCreate(t, db, &legal)
	structure := managementStructureFixture(10, "MGMT", "行政架构", "enabled")
	testutil.MustCreate(t, db, &structure)

	current := managementUnitFixture(20, "OU-001", "运营中心", "center", "enabled", &legal.Id)
	other := managementUnitFixture(21, "OU-002", "财务中心", "center", "enabled", &legal.Id)
	disabled := managementUnitFixture(22, "OU-003", "历史中心", "center", "disabled", &legal.Id)
	expired := managementUnitFixture(23, "OU-004", "过期中心", "center", "enabled", &legal.Id)
	expiredAt := model.Now().AddDate(0, 0, -1)
	expired.ValidTo = &expiredAt
	deleted := managementUnitFixture(24, "OU-005", "源删除中心", "center", "enabled", &legal.Id)
	deleted.SourceDeleted = true
	testutil.MustCreate(t, db, &[]model.OrgUnit{current, other, disabled, expired, deleted})
	testutil.MustCreate(t, db, &model.OrgStructureNode{
		Basic:            model.Basic{Id: 30, State: true},
		StructureId:      structure.Id,
		OrgUnitId:        current.Id,
		SourceSystemCode: "authority",
		SourceId:         "node-30",
		Path:             "/30/",
		Level:            1,
		Status:           "enabled",
		SyncStatus:       "synced",
	})

	result, err := orgService.QueryOrgUnits(nil, request.OrgUnitQueryReq{
		Basic: request.Basic{
			Page:       1,
			Num:        10,
			QuickQuery: &request.QuickQuery{Keyword: "运营"},
		},
		LegalEntityId: &legal.Id,
		UnitType:      "center",
	}, managementUnitTable())
	if err != nil {
		t.Fatalf("query org units: %v", err)
	}
	if result.Total != 1 || result.Data[0].Id != current.Id {
		t.Fatalf("unexpected org unit query: %+v", result)
	}

	advanced, err := orgService.QueryOrgUnits(nil, request.OrgUnitQueryReq{
		Basic: request.Basic{
			Expressions: []request.ExpressionGroup{{
				Logic: enum.And,
				Rules: []request.QueryRule{{
					Field:          "code",
					ExpressionType: enum.Eq,
					Value:          other.Code,
				}},
			}},
		},
	}, managementUnitTable())
	if err != nil || advanced.Total != 1 || advanced.Data[0].Id != other.Id {
		t.Fatalf("advanced unit query result=%+v err=%v", advanced, err)
	}

	withDisabled, err := orgService.QueryOrgUnits(nil, request.OrgUnitQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeDisabled: true},
	}, managementUnitTable())
	if err != nil || withDisabled.Total != 3 {
		t.Fatalf("include_disabled units=%+v err=%v", withDisabled, err)
	}
	withHistory, err := orgService.QueryOrgUnits(nil, request.OrgUnitQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeHistory: true},
	}, managementUnitTable())
	if err != nil || withHistory.Total != 4 {
		t.Fatalf("include_history units=%+v err=%v", withHistory, err)
	}

	detail, err := orgService.GetOrgUnitDetail(nil, current.Id, request.OrgUnitDetailReq{})
	if err != nil || detail.Id != current.Id ||
		detail.PrimaryLegalEntity == nil ||
		detail.PrimaryLegalEntity.Id != legal.Id {
		t.Fatalf("get org unit detail=%+v err=%v", detail, err)
	}
	assertOrganizationResponseDoesNotLeakSourceFields(t, detail)

	options, err := orgService.QueryOrgUnitOptions(nil, request.OrgUnitOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
			Page:        1,
			Num:         10,
			Keyword:     "中心",
			SelectedIds: []int{disabled.Id},
		},
		StructureId:   &structure.Id,
		LegalEntityId: &legal.Id,
	}, managementUnitTable())
	if err != nil {
		t.Fatalf("query org unit options: %v", err)
	}
	if options.Total != 1 || len(options.Items) != 2 ||
		options.Items[0].Value != current.Id ||
		options.Items[1].Value != disabled.Id ||
		!options.Items[1].Disabled {
		t.Fatalf("unexpected org unit options: %+v", options)
	}
	for _, option := range options.Items {
		if option.Value == 0 || option.Value == structure.Id {
			t.Fatalf("option value is not an org_unit_id: %+v", option)
		}
	}
}

func TestOrgServiceManagementQueriesHonorAsOfDateBoundaries(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	validFrom := orgServiceDate(t, "2026-01-01")
	validTo := orgServiceDate(t, "2026-01-31")
	structure := managementStructureFixture(10, "MGMT", "行政架构", "enabled")
	structure.ValidFrom = &validFrom
	structure.ValidTo = &validTo
	unit := managementUnitFixture(20, "OU-001", "运营中心", "center", "enabled", nil)
	unit.ValidFrom = &validFrom
	unit.ValidTo = &validTo
	testutil.MustCreate(t, db, &structure)
	testutil.MustCreate(t, db, &unit)

	for _, asOf := range []string{"2026-01-01", "2026-01-31"} {
		structures, err := orgService.QueryStructures(nil, request.OrgStructureQueryReq{
			OrgReadScopeReq: request.OrgReadScopeReq{AsOfDate: asOf},
		}, managementStructureTable())
		if err != nil || structures.Total != 1 {
			t.Fatalf("structures as_of_date %s=%+v err=%v", asOf, structures, err)
		}
		units, err := orgService.QueryOrgUnits(nil, request.OrgUnitQueryReq{
			OrgReadScopeReq: request.OrgReadScopeReq{AsOfDate: asOf},
		}, managementUnitTable())
		if err != nil || units.Total != 1 {
			t.Fatalf("units as_of_date %s=%+v err=%v", asOf, units, err)
		}
	}

	structures, err := orgService.QueryStructures(nil, request.OrgStructureQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{AsOfDate: "2026-02-01"},
	}, managementStructureTable())
	if err != nil || structures.Total != 0 {
		t.Fatalf("structures after validity=%+v err=%v", structures, err)
	}
	units, err := orgService.QueryOrgUnits(nil, request.OrgUnitQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{AsOfDate: "2026-02-01"},
	}, managementUnitTable())
	if err != nil || units.Total != 0 {
		t.Fatalf("units after validity=%+v err=%v", units, err)
	}
}

func TestOrgServiceStructureTreePreservesNodeIdentityAndSearchContext(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	structure := managementStructureFixture(10, "MGMT", "行政架构", "enabled")
	testutil.MustCreate(t, db, &structure)
	root := managementUnitFixture(20, "OU-ROOT", "总部", "business_unit", "enabled", nil)
	child := managementUnitFixture(21, "OU-CHILD", "运营中心", "center", "enabled", nil)
	orphan := managementUnitFixture(22, "OU-ORPHAN", "孤立部门", "department", "enabled", nil)
	disabledUnit := managementUnitFixture(23, "OU-DISABLED", "停用部门", "department", "disabled", nil)
	testutil.MustCreate(t, db, &[]model.OrgUnit{root, child, orphan, disabledUnit})

	missingParent := 999
	nodes := []model.OrgStructureNode{
		managementNodeFixture(30, structure.Id, root.Id, nil, "enabled"),
		managementNodeFixture(31, structure.Id, child.Id, intPointer(30), "enabled"),
		managementNodeFixture(32, structure.Id, child.Id, intPointer(30), "disabled"),
		managementNodeFixture(33, structure.Id, orphan.Id, &missingParent, "enabled"),
		managementNodeFixture(34, structure.Id, disabledUnit.Id, intPointer(30), "enabled"),
	}
	testutil.MustCreate(t, db, &nodes)

	tree, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
		StructureId: structure.Id,
	})
	if err != nil {
		t.Fatalf("get structure tree: %v", err)
	}
	if countStructureTreeNodes(tree) != 3 {
		t.Fatalf("default tree nodes = %d, want 3: %+v", countStructureTreeNodes(tree), tree)
	}
	assertTreeIdentityContract(t, tree)
	if !findStructureTreeNode(tree, nodes[3].Id).Orphan {
		t.Fatalf("orphan node was not diagnosed: %+v", tree)
	}

	withDisabled, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
		StructureId:     structure.Id,
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeDisabled: true},
	})
	if err != nil {
		t.Fatalf("get tree with disabled nodes: %v", err)
	}
	if countStructureTreeNodes(withDisabled) != 5 {
		t.Fatalf("duplicate org_unit occurrence was lost: %+v", withDisabled)
	}
	firstOccurrence := findStructureTreeNode(withDisabled, nodes[1].Id)
	secondOccurrence := findStructureTreeNode(withDisabled, nodes[2].Id)
	if firstOccurrence.OrgUnitId != child.Id ||
		secondOccurrence.OrgUnitId != child.Id ||
		firstOccurrence.StructureNodeId == secondOccurrence.StructureNodeId {
		t.Fatalf("node identity was collapsed: first=%+v second=%+v", firstOccurrence, secondOccurrence)
	}

	subtree, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
		StructureId: structure.Id,
		RootNodeId:  &nodes[0].Id,
	})
	if err != nil || len(subtree) != 1 || subtree[0].StructureNodeId != nodes[0].Id {
		t.Fatalf("root_node_id subtree=%+v err=%v", subtree, err)
	}

	rootOrgUnitId := root.Id
	byOrgUnit, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
		StructureId:   structure.Id,
		RootOrgUnitId: &rootOrgUnitId,
	})
	if err != nil || len(byOrgUnit) != 1 || byOrgUnit[0].OrgUnitId != root.Id {
		t.Fatalf("root_org_unit_id tree=%+v err=%v", byOrgUnit, err)
	}

	rootOrgUnitId = child.Id
	_, err = orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
		StructureId:     structure.Id,
		RootOrgUnitId:   &rootOrgUnitId,
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeDisabled: true},
	})
	assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgTreeRootAmbiguous)

	searched, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
		StructureId: structure.Id,
		Keyword:     "运营",
	})
	if err != nil {
		t.Fatalf("search structure tree: %v", err)
	}
	if len(searched) != 1 ||
		searched[0].StructureNodeId != nodes[0].Id ||
		len(searched[0].Children) != 1 ||
		searched[0].Children[0].StructureNodeId != nodes[1].Id {
		t.Fatalf("keyword result did not retain ancestors: %+v", searched)
	}
}

func TestOrgServiceStructureTreeRejectsCyclesInactiveStructuresAndOversize(t *testing.T) {
	t.Run("self and multi-node cycles", func(t *testing.T) {
		for _, multi := range []bool{false, true} {
			orgService, db := newOrgServiceTestSubject(t)
			structure := managementStructureFixture(10, "MGMT", "行政架构", "enabled")
			testutil.MustCreate(t, db, &structure)
			firstUnit := managementUnitFixture(20, "OU-1", "组织一", "department", "enabled", nil)
			secondUnit := managementUnitFixture(21, "OU-2", "组织二", "department", "enabled", nil)
			testutil.MustCreate(t, db, &[]model.OrgUnit{firstUnit, secondUnit})
			first := managementNodeFixture(30, structure.Id, firstUnit.Id, nil, "enabled")
			testutil.MustCreate(t, db, &first)
			if multi {
				second := managementNodeFixture(31, structure.Id, secondUnit.Id, &first.Id, "enabled")
				testutil.MustCreate(t, db, &second)
				if err := db.Model(&model.OrgStructureNode{}).
					Where("id = ?", first.Id).
					UpdateColumn("parent_node_id", second.Id).Error; err != nil {
					t.Fatalf("create multi-node cycle: %v", err)
				}
			} else if err := db.Model(&model.OrgStructureNode{}).
				Where("id = ?", first.Id).
				UpdateColumn("parent_node_id", first.Id).Error; err != nil {
				t.Fatalf("create self cycle: %v", err)
			}

			_, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
				StructureId: structure.Id,
			})
			assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgStructureCycle)
		}
	})

	t.Run("inactive structure", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure := managementStructureFixture(10, "MGMT", "停用架构", "disabled")
		testutil.MustCreate(t, db, &structure)
		_, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
			StructureId: structure.Id,
		})
		assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgStructureInactive)
	})

	t.Run("missing structure and root node", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		_, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
			StructureId: 999,
		})
		assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgStructureNotFound)

		structure := managementStructureFixture(10, "MGMT", "行政架构", "enabled")
		testutil.MustCreate(t, db, &structure)
		rootNodeId := 999
		_, err = orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
			StructureId: structure.Id,
			RootNodeId:  &rootNodeId,
		})
		assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgStructureNodeMissing)
	})

	t.Run("maximum nodes", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure := managementStructureFixture(10, "MGMT", "行政架构", "enabled")
		testutil.MustCreate(t, db, &structure)
		orgService.structureNodeRepo = oversizedStructureNodeRepository{
			OrgStructureNodeRepository: orgService.structureNodeRepo,
		}
		_, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
			StructureId: structure.Id,
		})
		assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgTreeTooLarge)
	})

	t.Run("keyword narrows a structure larger than the response limit", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure := managementStructureFixture(10, "MGMT", "大型行政架构", "enabled")
		testutil.MustCreate(t, db, &structure)

		nodes := make([]model.OrgStructureNode, orgStructureTreeMaxNodeCount+1)
		units := make([]model.OrgUnit, orgStructureTreeMaxNodeCount+1)
		for index := range nodes {
			id := index + 1
			name := "普通部门"
			if index == len(nodes)-1 {
				name = "目标运营中心"
			}
			units[index] = managementUnitFixture(id, "OU-"+strconv.Itoa(id), name, "department", "enabled", nil)
			nodes[index] = managementNodeFixture(id, structure.Id, id, nil, "enabled")
		}

		nodeRepo := &keywordSearchStructureNodeRepository{
			OrgStructureNodeRepository: orgService.structureNodeRepo,
			nodes:                      nodes,
		}
		orgService.structureNodeRepo = nodeRepo
		orgService.orgUnitRepo = displayOrgUnitRepository{
			OrgUnitRepository: orgService.orgUnitRepo,
			units:             units,
		}

		tree, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
			StructureId: structure.Id,
			Keyword:     "目标运营",
		})
		if err != nil {
			t.Fatalf("search large structure tree: %v", err)
		}
		if len(tree) != 1 || tree[0].Name != "目标运营中心" {
			t.Fatalf("unexpected narrowed structure tree: %+v", tree)
		}
		if nodeRepo.limit != orgStructureTreeMaxScanCount+1 {
			t.Fatalf("large structure scan limit = %d, want %d", nodeRepo.limit, orgStructureTreeMaxScanCount+1)
		}
	})
}

func TestOrgServiceStructureTreeUsesThreeReadQueries(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	structure := managementStructureFixture(10, "MGMT", "行政架构", "enabled")
	unit := managementUnitFixture(20, "OU-ROOT", "总部", "business_unit", "enabled", nil)
	node := managementNodeFixture(30, structure.Id, unit.Id, nil, "enabled")
	testutil.MustCreate(t, db, &structure)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &node)

	var queryCount atomic.Int32
	callbackName := "test:count-org-tree-queries"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(*gorm.DB) {
		queryCount.Add(1)
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})

	tree, err := orgService.GetStructureOrgTree(nil, request.OrgStructureOrgTreeReq{
		StructureId: structure.Id,
	})
	if err != nil || countStructureTreeNodes(tree) != 1 {
		t.Fatalf("get structure tree=%+v err=%v", tree, err)
	}
	if got := queryCount.Load(); got != 3 {
		t.Fatalf("structure tree query count = %d, want 3", got)
	}
}

type oversizedStructureNodeRepository struct {
	repository.OrgStructureNodeRepository
}

func (oversizedStructureNodeRepository) ListByStructureForRead(
	context.Context,
	int,
	repository.OrgReadScope,
	int,
) ([]model.OrgStructureNode, error) {
	return make([]model.OrgStructureNode, orgStructureTreeMaxNodeCount+1), nil
}

type keywordSearchStructureNodeRepository struct {
	repository.OrgStructureNodeRepository
	nodes []model.OrgStructureNode
	limit int
}

func (r *keywordSearchStructureNodeRepository) ListByStructureForRead(
	_ context.Context,
	_ int,
	_ repository.OrgReadScope,
	limit int,
) ([]model.OrgStructureNode, error) {
	r.limit = limit
	return r.nodes, nil
}

type displayOrgUnitRepository struct {
	repository.OrgUnitRepository
	units []model.OrgUnit
}

func (r displayOrgUnitRepository) FindByIdsForDisplay(
	context.Context,
	[]int,
) ([]model.OrgUnit, error) {
	return r.units, nil
}

func managementStructureFixture(id int, code, name, status string) model.OrgStructure {
	return model.OrgStructure{
		Basic:            model.Basic{Id: id, State: true},
		Code:             code,
		Name:             name,
		StructureType:    "management",
		SourceSystemCode: "authority",
		SourceId:         "source-" + code,
		Status:           status,
		IsDefault:        code == "MGMT",
		SyncStatus:       "synced",
	}
}

func managementUnitFixture(
	id int,
	code string,
	name string,
	unitType string,
	status string,
	legalEntityId *int,
) model.OrgUnit {
	return model.OrgUnit{
		Basic:                model.Basic{Id: id, State: true},
		SourceSystemCode:     "authority",
		SourceId:             "source-" + code,
		SourceCode:           "source-code-" + code,
		Code:                 code,
		Name:                 name,
		UnitType:             unitType,
		PrimaryLegalEntityId: legalEntityId,
		Status:               status,
		SyncStatus:           "synced",
	}
}

func managementNodeFixture(
	id int,
	structureId int,
	orgUnitId int,
	parentNodeId *int,
	status string,
) model.OrgStructureNode {
	return model.OrgStructureNode{
		Basic:            model.Basic{Id: id, State: true},
		StructureId:      structureId,
		OrgUnitId:        orgUnitId,
		ParentNodeId:     parentNodeId,
		SourceSystemCode: "authority",
		SourceId:         "source-node-" + strconv.Itoa(id),
		Path:             "/node/",
		Level:            1,
		Sort:             id,
		Status:           status,
		SyncStatus:       "synced",
	}
}

func managementStructureTable() model.SysTable {
	return managementQueryTable("org_structure", []string{"code", "name"}, map[string]enum.SysTableFieldType{
		"id":             enum.BigIntFieldType,
		"code":           enum.VarcharFieldType,
		"name":           enum.VarcharFieldType,
		"structure_type": enum.VarcharFieldType,
		"status":         enum.VarcharFieldType,
		"is_default":     enum.BooleanFieldType,
		"valid_from":     enum.DatetimeFieldType,
		"valid_to":       enum.DatetimeFieldType,
	})
}

func managementUnitTable() model.SysTable {
	return managementQueryTable("org_unit", []string{"code", "name"}, map[string]enum.SysTableFieldType{
		"id":                      enum.BigIntFieldType,
		"code":                    enum.VarcharFieldType,
		"name":                    enum.VarcharFieldType,
		"unit_type":               enum.VarcharFieldType,
		"primary_legal_entity_id": enum.BigIntFieldType,
		"status":                  enum.VarcharFieldType,
		"valid_from":              enum.DatetimeFieldType,
		"valid_to":                enum.DatetimeFieldType,
	})
}

func managementQueryTable(
	tableCode string,
	quickFields []string,
	fields map[string]enum.SysTableFieldType,
) model.SysTable {
	quick := make(map[string]struct{}, len(quickFields))
	for _, field := range quickFields {
		quick[field] = struct{}{}
	}
	tableFields := make([]model.SysTableField, 0, len(fields))
	for code, fieldType := range fields {
		_, isQuick := quick[code]
		tableFields = append(tableFields, model.SysTableField{
			FieldCode:        code,
			FieldType:        fieldType,
			IsListShow:       true,
			IsQuickSearch:    isQuick,
			IsAdvancedSearch: true,
			IsSort:           true,
		})
	}
	return model.SysTable{
		Basic:       model.Basic{Id: 1, State: true},
		TableCode:   tableCode,
		TableFields: tableFields,
	}
}

func assertOrganizationResponseDoesNotLeakSourceFields(t *testing.T, value any) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal organization response: %v", err)
	}
	for _, forbidden := range []string{
		"source_id",
		"source_version",
		"sync_status",
		"last_error",
		"path",
	} {
		if strings.Contains(string(encoded), `"`+forbidden+`"`) {
			t.Fatalf("organization response leaked %s: %s", forbidden, encoded)
		}
	}
}

func countStructureTreeNodes(nodes []response.OrgStructureOrgTreeNodeRes) int {
	total := 0
	for _, node := range nodes {
		total += 1 + countStructureTreeNodes(node.Children)
	}
	return total
}

func findStructureTreeNode(
	nodes []response.OrgStructureOrgTreeNodeRes,
	nodeId int,
) response.OrgStructureOrgTreeNodeRes {
	for _, node := range nodes {
		if node.StructureNodeId == nodeId {
			return node
		}
		if found := findStructureTreeNode(node.Children, nodeId); found.StructureNodeId != 0 {
			return found
		}
	}
	return response.OrgStructureOrgTreeNodeRes{}
}

func assertTreeIdentityContract(t *testing.T, nodes []response.OrgStructureOrgTreeNodeRes) {
	t.Helper()
	for _, node := range nodes {
		if node.StructureNodeId == 0 || node.OrgUnitId == 0 {
			t.Fatalf("tree identity is incomplete: %+v", node)
		}
		assertTreeIdentityContract(t, node.Children)
	}
}

func intPointer(value int) *int {
	return &value
}
