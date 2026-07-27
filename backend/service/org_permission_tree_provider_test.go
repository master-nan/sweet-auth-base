package service

import (
	"backend/dto/response"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestOrgPermissionTreeProviderAncestorsDescendantsAndIncludeSelf(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	structure, units, nodes := seedPermissionTree(t, db, 3000, "PERM-MGMT")

	ancestors, err := orgService.GetOrgAncestors(
		nil,
		structure.Code,
		units[3].Id,
		"2026-07-27",
		false,
	)
	if err != nil {
		t.Fatalf("get ancestors: %v", err)
	}
	assertPermissionRelationItems(t, ancestors.Items, []response.OrgRelationItemRes{
		{OrgUnitId: units[2].Id, Distance: 1},
		{OrgUnitId: units[1].Id, Distance: 2},
		{OrgUnitId: units[0].Id, Distance: 3},
	})
	if ancestors.StructureCode != structure.Code ||
		ancestors.OrgUnitId != units[3].Id ||
		ancestors.AsOfDate != "2026-07-27" {
		t.Fatalf("unexpected ancestors envelope: %+v", ancestors)
	}

	withSelf, err := orgService.GetOrgAncestors(
		nil,
		structure.Code,
		units[3].Id,
		"2026-07-27",
		true,
	)
	if err != nil {
		t.Fatalf("get ancestors with self: %v", err)
	}
	assertPermissionRelationItems(t, withSelf.Items, []response.OrgRelationItemRes{
		{OrgUnitId: units[3].Id, Distance: 0},
		{OrgUnitId: units[2].Id, Distance: 1},
		{OrgUnitId: units[1].Id, Distance: 2},
		{OrgUnitId: units[0].Id, Distance: 3},
	})

	descendants, err := orgService.GetOrgDescendants(
		nil,
		structure.Code,
		units[0].Id,
		"2026-07-27",
		false,
	)
	if err != nil {
		t.Fatalf("get descendants: %v", err)
	}
	assertPermissionRelationItems(t, descendants.Items, []response.OrgRelationItemRes{
		{OrgUnitId: units[1].Id, Distance: 1},
		{OrgUnitId: units[2].Id, Distance: 2},
		{OrgUnitId: units[3].Id, Distance: 3},
	})

	descendantsWithSelf, err := orgService.GetOrgDescendants(
		nil,
		structure.Code,
		units[0].Id,
		"2026-07-27",
		true,
	)
	if err != nil {
		t.Fatalf("get descendants with self: %v", err)
	}
	assertPermissionRelationItems(t, descendantsWithSelf.Items, []response.OrgRelationItemRes{
		{OrgUnitId: units[0].Id, Distance: 0},
		{OrgUnitId: units[1].Id, Distance: 1},
		{OrgUnitId: units[2].Id, Distance: 2},
		{OrgUnitId: units[3].Id, Distance: 3},
	})

	check, err := orgService.IsOrgDescendant(
		nil,
		structure.Code,
		units[0].Id,
		units[3].Id,
		"2026-07-27",
		false,
	)
	if err != nil || !check.IsDescendant || check.Distance == nil || *check.Distance != 3 {
		t.Fatalf("descendant check=%+v err=%v", check, err)
	}

	sameWithoutSelf, err := orgService.IsOrgDescendant(
		nil,
		structure.Code,
		units[1].Id,
		units[1].Id,
		"2026-07-27",
		false,
	)
	if err != nil || sameWithoutSelf.IsDescendant || sameWithoutSelf.Distance != nil {
		t.Fatalf("same node without self=%+v err=%v", sameWithoutSelf, err)
	}
	sameWithSelf, err := orgService.IsOrgDescendant(
		nil,
		structure.Code,
		units[1].Id,
		units[1].Id,
		"2026-07-27",
		true,
	)
	if err != nil ||
		!sameWithSelf.IsDescendant ||
		sameWithSelf.Distance == nil ||
		*sameWithSelf.Distance != 0 {
		t.Fatalf("same node with self=%+v err=%v", sameWithSelf, err)
	}

	serialized, err := json.Marshal(struct {
		Ancestors   response.OrgAncestorsRes   `json:"ancestors"`
		Descendants response.OrgDescendantsRes `json:"descendants"`
	}{Ancestors: ancestors, Descendants: descendants})
	if err != nil {
		t.Fatalf("marshal relation responses: %v", err)
	}
	for _, forbidden := range []string{"structure_node_id", `"path"`} {
		if strings.Contains(string(serialized), forbidden) {
			t.Fatalf("permission relation response leaked %q: %s", forbidden, serialized)
		}
	}
	if nodes[0].Id == 0 {
		t.Fatal("invalid tree fixture")
	}
}

func TestOrgPermissionTreeProviderIsolatesStructuresAndReturnsNoRelation(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	first := managementStructureFixture(3100, "PERM-MGMT-A", "权限架构甲", "enabled")
	second := managementStructureFixture(3101, "PERM-MGMT-B", "权限架构乙", "enabled")
	left := managementUnitFixture(3110, "PERM-LEFT", "左组织", "department", "enabled", nil)
	right := managementUnitFixture(3111, "PERM-RIGHT", "右组织", "department", "enabled", nil)
	root := managementUnitFixture(3112, "PERM-ROOT", "根组织", "business_unit", "enabled", nil)
	testutil.MustCreate(t, db, &[]model.OrgStructure{first, second})
	testutil.MustCreate(t, db, &[]model.OrgUnit{left, right, root})

	firstNodes := []model.OrgStructureNode{
		managementNodeFixture(3120, first.Id, root.Id, nil, "enabled"),
		managementNodeFixture(3121, first.Id, left.Id, intPointer(3120), "enabled"),
		managementNodeFixture(3122, first.Id, right.Id, intPointer(3120), "enabled"),
	}
	secondNodes := []model.OrgStructureNode{
		managementNodeFixture(3130, second.Id, right.Id, nil, "enabled"),
		managementNodeFixture(3131, second.Id, left.Id, intPointer(3130), "enabled"),
	}
	testutil.MustCreate(t, db, &firstNodes)
	testutil.MustCreate(t, db, &secondNodes)

	noRelation, err := orgService.IsOrgDescendant(
		nil,
		first.Code,
		left.Id,
		right.Id,
		"2026-07-27",
		false,
	)
	if err != nil || noRelation.IsDescendant || noRelation.Distance != nil {
		t.Fatalf("sibling relation=%+v err=%v", noRelation, err)
	}

	isolatedRelation, err := orgService.IsOrgDescendant(
		nil,
		second.Code,
		right.Id,
		left.Id,
		"2026-07-27",
		false,
	)
	if err != nil ||
		!isolatedRelation.IsDescendant ||
		isolatedRelation.Distance == nil ||
		*isolatedRelation.Distance != 1 {
		t.Fatalf("isolated structure relation=%+v err=%v", isolatedRelation, err)
	}
}

func TestOrgPermissionTreeProviderRejectsMissingStructureAndOrganizations(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	structure, units, _ := seedPermissionTree(t, db, 3200, "PERM-MISSING")
	outside := managementUnitFixture(
		3290,
		"PERM-OUTSIDE",
		"架构外组织",
		"department",
		"enabled",
		nil,
	)
	testutil.MustCreate(t, db, &outside)

	_, err := orgService.GetOrgAncestors(
		nil,
		"",
		units[0].Id,
		"2026-07-27",
		false,
	)
	assertOrgServiceAdminError(
		t,
		err,
		response.ErrorCategoryParameter,
		apperrors.ErrorCodeParamInvalid,
	)

	_, err = orgService.GetOrgAncestors(
		nil,
		structure.Code,
		units[0].Id,
		"",
		false,
	)
	assertOrgServiceAdminError(
		t,
		err,
		response.ErrorCategoryParameter,
		apperrors.ErrorCodeParamInvalid,
	)

	_, err = orgService.GetOrgAncestors(
		nil,
		"NOT-EXISTS",
		units[0].Id,
		"2026-07-27",
		false,
	)
	assertOrgServiceAdminError(
		t,
		err,
		response.ErrorCategoryBusiness,
		apperrors.ErrorCodeOrgStructureNotFound,
	)

	_, err = orgService.GetOrgDescendants(
		nil,
		structure.Code,
		99999,
		"2026-07-27",
		false,
	)
	assertOrgServiceAdminError(
		t,
		err,
		response.ErrorCategoryBusiness,
		apperrors.ErrorCodeOrgUnitNotFound,
	)

	_, err = orgService.GetOrgAncestors(
		nil,
		structure.Code,
		outside.Id,
		"2026-07-27",
		false,
	)
	assertOrgServiceAdminError(
		t,
		err,
		response.ErrorCategoryBusiness,
		apperrors.ErrorCodeOrgStructureMembershipNotFound,
	)
}

func TestOrgPermissionTreeProviderRejectsCyclesAndOrphans(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure, units, nodes := seedPermissionTree(t, db, 3300, "PERM-CYCLE")
		if err := db.Model(&model.OrgStructureNode{}).
			Where("id = ?", nodes[0].Id).
			UpdateColumn("parent_node_id", nodes[3].Id).Error; err != nil {
			t.Fatalf("create cycle: %v", err)
		}

		_, err := orgService.GetOrgDescendants(
			nil,
			structure.Code,
			units[0].Id,
			"2026-07-27",
			false,
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgStructureCycle,
		)
	})

	t.Run("orphan", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure, units, nodes := seedPermissionTree(t, db, 3400, "PERM-ORPHAN")
		if err := db.Model(&model.OrgStructureNode{}).
			Where("id = ?", nodes[2].Id).
			UpdateColumn("parent_node_id", 99999).Error; err != nil {
			t.Fatalf("create orphan: %v", err)
		}

		_, err := orgService.GetOrgAncestors(
			nil,
			structure.Code,
			units[3].Id,
			"2026-07-27",
			false,
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgStructureNodeMissing,
		)
	})
}

func TestOrgPermissionTreeProviderRejectsOversizeAndInvalidDates(t *testing.T) {
	t.Run("oversize", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure := managementStructureFixture(3500, "PERM-LARGE", "超大架构", "enabled")
		unit := managementUnitFixture(
			3510,
			"PERM-LARGE-ROOT",
			"超大架构根组织",
			"business_unit",
			"enabled",
			nil,
		)
		testutil.MustCreate(t, db, &structure)
		testutil.MustCreate(t, db, &unit)
		orgService.structureNodeRepo = oversizedStructureNodeRepository{
			OrgStructureNodeRepository: orgService.structureNodeRepo,
		}

		_, err := orgService.GetOrgDescendants(
			nil,
			structure.Code,
			unit.Id,
			"2026-07-27",
			true,
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgTreeTooLarge,
		)
	})

	t.Run("inactive structure at date", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure, units, _ := seedPermissionTree(t, db, 3600, "PERM-FUTURE-STRUCTURE")
		future := orgServiceDate(t, "2026-07-28")
		if err := db.Model(&model.OrgStructure{}).
			Where("id = ?", structure.Id).
			Update("valid_from", future).Error; err != nil {
			t.Fatalf("set structure validity: %v", err)
		}

		_, err := orgService.GetOrgAncestors(
			nil,
			structure.Code,
			units[0].Id,
			"2026-07-27",
			false,
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgStructureInactive,
		)
	})

	t.Run("inactive unit at date", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure, units, _ := seedPermissionTree(t, db, 3700, "PERM-FUTURE-UNIT")
		future := orgServiceDate(t, "2026-07-28")
		if err := db.Model(&model.OrgUnit{}).
			Where("id = ?", units[1].Id).
			Update("valid_from", future).Error; err != nil {
			t.Fatalf("set unit validity: %v", err)
		}

		_, err := orgService.GetOrgDescendants(
			nil,
			structure.Code,
			units[1].Id,
			"2026-07-27",
			false,
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgUnitInactive,
		)
	})

	t.Run("inactive node at date", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		structure, units, nodes := seedPermissionTree(t, db, 3800, "PERM-FUTURE-NODE")
		future := orgServiceDate(t, "2026-07-28")
		if err := db.Model(&model.OrgStructureNode{}).
			Where("id = ?", nodes[2].Id).
			Update("valid_from", future).Error; err != nil {
			t.Fatalf("set node validity: %v", err)
		}

		_, err := orgService.GetOrgAncestors(
			nil,
			structure.Code,
			units[2].Id,
			"2026-07-27",
			false,
		)
		assertOrgServiceAdminError(
			t,
			err,
			response.ErrorCategoryBusiness,
			apperrors.ErrorCodeOrgStructureNodeMissing,
		)
	})
}

func seedPermissionTree(
	t *testing.T,
	db *gorm.DB,
	baseId int,
	structureCode string,
) (model.OrgStructure, []model.OrgUnit, []model.OrgStructureNode) {
	t.Helper()
	structure := managementStructureFixture(
		baseId,
		structureCode,
		"权限消费架构",
		"enabled",
	)
	units := []model.OrgUnit{
		managementUnitFixture(baseId+10, structureCode+"-ROOT", "总部", "business_unit", "enabled", nil),
		managementUnitFixture(baseId+11, structureCode+"-DIV", "事业部", "business_unit", "enabled", nil),
		managementUnitFixture(baseId+12, structureCode+"-DEPT", "部门", "department", "enabled", nil),
		managementUnitFixture(baseId+13, structureCode+"-TEAM", "团队", "team", "enabled", nil),
	}
	nodes := []model.OrgStructureNode{
		managementNodeFixture(baseId+20, structure.Id, units[0].Id, nil, "enabled"),
		managementNodeFixture(baseId+21, structure.Id, units[1].Id, intPointer(baseId+20), "enabled"),
		managementNodeFixture(baseId+22, structure.Id, units[2].Id, intPointer(baseId+21), "enabled"),
		managementNodeFixture(baseId+23, structure.Id, units[3].Id, intPointer(baseId+22), "enabled"),
	}
	testutil.MustCreate(t, db, &structure)
	testutil.MustCreate(t, db, &units)
	testutil.MustCreate(t, db, &nodes)
	return structure, units, nodes
}

func assertPermissionRelationItems(
	t *testing.T,
	got []response.OrgRelationItemRes,
	want []response.OrgRelationItemRes,
) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("relation items=%+v, want %+v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf(
				"relation item[%d]=%+v, want %+v; all=%+v",
				index,
				got[index],
				want[index],
				got,
			)
		}
	}
}
