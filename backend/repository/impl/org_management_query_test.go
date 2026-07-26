package impl

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"
	"testing"
)

func TestOrgStructureRepositoryReadQueryFiltersThroughStructureNodes(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.OrgStructure{},
		&model.OrgUnit{},
		&model.OrgStructureNode{},
	)
	primaryDB := &database.PrimaryDB{DB: db}
	legalEntityId := 101
	matching := repositoryStructureFixture(1, "MGMT-A", "架构甲", "enabled")
	other := repositoryStructureFixture(2, "MGMT-B", "架构乙", "enabled")
	disabled := repositoryStructureFixture(3, "MGMT-C", "停用架构", "disabled")
	testutil.MustCreate(t, db, &[]model.OrgStructure{matching, other, disabled})
	unit := repositoryUnitFixture(10, "OU-A", "组织甲", "enabled", &legalEntityId)
	otherUnit := repositoryUnitFixture(11, "OU-B", "组织乙", "enabled", nil)
	testutil.MustCreate(t, db, &[]model.OrgUnit{unit, otherUnit})
	testutil.MustCreate(t, db, &[]model.OrgStructureNode{
		repositoryNodeFixture(20, matching.Id, unit.Id, "enabled"),
		repositoryNodeFixture(21, other.Id, otherUnit.Id, "enabled"),
	})

	req := &request.OrgStructureQueryReq{
		Basic: request.Basic{
			Page: 1,
			Num:  1,
			Order: request.Order{
				Field: "code",
				IsAsc: true,
			},
		},
		LegalEntityId: &legalEntityId,
	}
	result, err := NewOrgStructureRepositoryImpl(primaryDB).QueryForRead(
		nil,
		req,
		managementRepositoryTable("org_structure"),
		repository.OrgReadScope{AsOf: model.Now()},
	)
	if err != nil {
		t.Fatalf("query structures for legal entity: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 1 || result.Data[0].Id != matching.Id {
		t.Fatalf("unexpected structure repository result: %+v", result)
	}
	if req.Filters != nil {
		t.Fatalf("repository mutated caller filters: %+v", req.Filters)
	}
}

func TestOrgUnitRepositoryReadQueryFiltersByStructureAndScope(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.OrgStructure{},
		&model.OrgUnit{},
		&model.OrgStructureNode{},
	)
	primaryDB := &database.PrimaryDB{DB: db}
	structure := repositoryStructureFixture(1, "MGMT", "行政架构", "enabled")
	testutil.MustCreate(t, db, &structure)
	matching := repositoryUnitFixture(10, "OU-A", "组织甲", "enabled", nil)
	other := repositoryUnitFixture(11, "OU-B", "组织乙", "enabled", nil)
	disabled := repositoryUnitFixture(12, "OU-C", "停用组织", "disabled", nil)
	testutil.MustCreate(t, db, &[]model.OrgUnit{matching, other, disabled})
	testutil.MustCreate(t, db, &[]model.OrgStructureNode{
		repositoryNodeFixture(20, structure.Id, matching.Id, "enabled"),
		repositoryNodeFixture(21, structure.Id, disabled.Id, "enabled"),
	})

	result, err := NewOrgUnitRepositoryImpl(primaryDB).QueryForRead(
		nil,
		&request.OrgUnitQueryReq{},
		managementRepositoryTable("org_unit"),
		repository.OrgReadScope{AsOf: model.Now()},
		&structure.Id,
	)
	if err != nil {
		t.Fatalf("query units by structure: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 1 || result.Data[0].Id != matching.Id {
		t.Fatalf("unexpected structure-filtered units: %+v", result)
	}

	withDisabled, err := NewOrgUnitRepositoryImpl(primaryDB).QueryForRead(
		nil,
		&request.OrgUnitQueryReq{},
		managementRepositoryTable("org_unit"),
		repository.OrgReadScope{AsOf: model.Now(), IncludeDisabled: true},
		&structure.Id,
	)
	if err != nil || withDisabled.Total != 2 {
		t.Fatalf("include disabled units=%+v err=%v", withDisabled, err)
	}
}

func TestOrgStructureNodeRepositoryReadListUsesOneBoundedQuery(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.OrgStructureNode{})
	repo := NewOrgStructureNodeRepositoryImpl(&database.PrimaryDB{DB: db})
	nodes := []model.OrgStructureNode{
		repositoryNodeFixture(1, 100, 200, "enabled"),
		repositoryNodeFixture(2, 100, 201, "disabled"),
		repositoryNodeFixture(3, 101, 202, "enabled"),
	}
	testutil.MustCreate(t, db, &nodes)

	result, err := repo.ListByStructureForRead(
		nil,
		100,
		repository.OrgReadScope{AsOf: model.Now()},
		2,
	)
	if err != nil {
		t.Fatalf("list structure nodes: %v", err)
	}
	if len(result) != 1 || result[0].Id != nodes[0].Id {
		t.Fatalf("unexpected effective structure nodes: %+v", result)
	}

	result, err = repo.ListByStructureForRead(
		nil,
		100,
		repository.OrgReadScope{AsOf: model.Now(), IncludeDisabled: true},
		1,
	)
	if err != nil || len(result) != 1 {
		t.Fatalf("bounded structure node query=%+v err=%v", result, err)
	}
}

func managementRepositoryTable(tableCode string) model.SysTable {
	fields := []model.SysTableField{
		{
			FieldCode:        "id",
			FieldType:        enum.BigIntFieldType,
			IsListShow:       true,
			IsAdvancedSearch: true,
			IsSort:           true,
		},
		{
			FieldCode:        "code",
			FieldType:        enum.VarcharFieldType,
			IsListShow:       true,
			IsQuickSearch:    true,
			IsAdvancedSearch: true,
			IsSort:           true,
		},
		{
			FieldCode:        "name",
			FieldType:        enum.VarcharFieldType,
			IsListShow:       true,
			IsQuickSearch:    true,
			IsAdvancedSearch: true,
			IsSort:           true,
		},
		{
			FieldCode:        "status",
			FieldType:        enum.VarcharFieldType,
			IsListShow:       true,
			IsAdvancedSearch: true,
			IsSort:           true,
		},
	}
	return model.SysTable{
		Basic:       model.Basic{Id: 1, State: true},
		TableCode:   tableCode,
		TableFields: fields,
	}
}

func repositoryStructureFixture(id int, code, name, status string) model.OrgStructure {
	return model.OrgStructure{
		Basic:            model.Basic{Id: id, State: true},
		Code:             code,
		Name:             name,
		StructureType:    "management",
		SourceSystemCode: "authority",
		SourceId:         "source-" + code,
		Status:           status,
		SyncStatus:       "synced",
	}
}

func repositoryUnitFixture(
	id int,
	code string,
	name string,
	status string,
	legalEntityId *int,
) model.OrgUnit {
	return model.OrgUnit{
		Basic:                model.Basic{Id: id, State: true},
		SourceSystemCode:     "authority",
		SourceId:             "source-" + code,
		Code:                 code,
		Name:                 name,
		UnitType:             "department",
		PrimaryLegalEntityId: legalEntityId,
		Status:               status,
		SyncStatus:           "synced",
	}
}

func repositoryNodeFixture(
	id int,
	structureId int,
	orgUnitId int,
	status string,
) model.OrgStructureNode {
	return model.OrgStructureNode{
		Basic:            model.Basic{Id: id, State: true},
		StructureId:      structureId,
		OrgUnitId:        orgUnitId,
		SourceSystemCode: "authority",
		SourceId:         "node-" + string(rune(id+64)),
		Path:             "/node/",
		Level:            1,
		Sort:             id,
		Status:           status,
		SyncStatus:       "synced",
	}
}
