package impl

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository"
	"testing"
	"time"
)

func TestOrgEmployeeRepositoryReadQueryUsesOneAssignmentAndStablePagination(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgPosition{},
		&model.OrgEmployee{},
		&model.OrgAssignment{},
		&model.SysUser{},
	)
	repo := NewOrgEmployeeRepositoryImpl(&database.PrimaryDB{DB: db})
	legalA := employeePositionLegalFixture(1, "LE-A")
	legalB := employeePositionLegalFixture(2, "LE-B")
	unitA := employeePositionUnitFixture(10, "OU-A", legalA.Id)
	unitB := employeePositionUnitFixture(20, "OU-B", legalB.Id)
	positionA := employeePositionPositionFixture(100, "POS-A", unitA.Id, "enabled")
	positionB := employeePositionPositionFixture(200, "POS-B", unitB.Id, "enabled")
	employeeA := employeePositionEmployeeFixture(1000, "EMP-A", "张三", "active")
	employeeB := employeePositionEmployeeFixture(1001, "EMP-B", "李四", "active")
	testutil.MustCreate(t, db, &[]model.OrgLegalEntity{legalA, legalB})
	testutil.MustCreate(t, db, &[]model.OrgUnit{unitA, unitB})
	testutil.MustCreate(t, db, &[]model.OrgPosition{positionA, positionB})
	testutil.MustCreate(t, db, &[]model.OrgEmployee{employeeA, employeeB})
	testutil.MustCreate(t, db, &[]model.OrgAssignment{
		employeePositionAssignmentFixture(1, employeeA.Id, legalA.Id, unitA.Id, positionA.Id),
		employeePositionAssignmentFixture(2, employeeA.Id, legalB.Id, unitB.Id, positionB.Id),
		employeePositionAssignmentFixture(3, employeeA.Id, legalA.Id, unitA.Id, positionA.Id),
		employeePositionAssignmentFixture(4, employeeB.Id, legalB.Id, unitB.Id, positionB.Id),
	})

	scope := repository.OrgReadScope{AsOf: model.Now()}
	result, err := repo.QueryForRead(
		nil,
		&request.OrgEmployeeQueryReq{
			Basic: request.Basic{
				Page: 1,
				Num:  10,
				Order: request.Order{
					Field: "employee_no",
					IsAsc: true,
				},
			},
			LegalEntityId: &legalA.Id,
			OrgUnitId:     &unitA.Id,
			PositionId:    &positionA.Id,
		},
		employeePositionRepositoryTable("org_employee"),
		scope,
	)
	if err != nil {
		t.Fatalf("query employees by one assignment: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 1 || result.Data[0].Id != employeeA.Id {
		t.Fatalf("duplicate assignment changed employee pagination: %+v", result)
	}

	result, err = repo.QueryForRead(
		nil,
		&request.OrgEmployeeQueryReq{
			OrgUnitId:  &unitA.Id,
			PositionId: &positionB.Id,
		},
		employeePositionRepositoryTable("org_employee"),
		scope,
	)
	if err != nil {
		t.Fatalf("query employees with split assignment filters: %v", err)
	}
	if result.Total != 0 || len(result.Data) != 0 {
		t.Fatalf("filters from different assignments were incorrectly combined: %+v", result)
	}
}

func TestOrgEmployeeRepositoryReadQuerySupportsMetadataScopeAndBinding(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.OrgEmployee{}, &model.SysUser{})
	repo := NewOrgEmployeeRepositoryImpl(&database.PrimaryDB{DB: db})
	user := model.SysUser{Basic: model.Basic{Id: 50, State: true}, UserName: "employee_user"}
	testutil.MustCreate(t, db, &user)
	bound := employeePositionEmployeeFixture(1, "EMP-001", "王一", "active")
	bound.UserId = &user.Id
	unbound := employeePositionEmployeeFixture(2, "EMP-002", "王二", "probation")
	disabled := employeePositionEmployeeFixture(3, "EMP-003", "王三", "resigned")
	expired := employeePositionEmployeeFixture(4, "EMP-004", "王四", "active")
	expiredAt := model.Now().Add(-24 * time.Hour)
	expired.ValidTo = &expiredAt
	testutil.MustCreate(t, db, &[]model.OrgEmployee{bound, unbound, disabled, expired})

	table := employeePositionRepositoryTable("org_employee")
	scope := repository.OrgReadScope{AsOf: model.Now()}
	result, err := repo.QueryForRead(nil, &request.OrgEmployeeQueryReq{
		Basic: request.Basic{
			QuickQuery: &request.QuickQuery{Keyword: "EMP-001"},
		},
		BoundStatus: "bound",
	}, table, scope)
	if err != nil || result.Total != 1 || result.Data[0].Id != bound.Id {
		t.Fatalf("bound quick query result=%+v err=%v", result, err)
	}

	result, err = repo.QueryForRead(nil, &request.OrgEmployeeQueryReq{
		Basic: request.Basic{
			Expressions: []request.ExpressionGroup{{
				Logic: enum.And,
				Rules: []request.QueryRule{{
					Field:          "name",
					ExpressionType: enum.Like,
					Value:          "王二",
				}},
			}},
		},
		BoundStatus: "unbound",
	}, table, scope)
	if err != nil || result.Total != 1 || result.Data[0].Id != unbound.Id {
		t.Fatalf("unbound advanced query result=%+v err=%v", result, err)
	}

	withDisabled, err := repo.QueryForRead(nil, &request.OrgEmployeeQueryReq{}, table, repository.OrgReadScope{
		AsOf:            model.Now(),
		IncludeDisabled: true,
	})
	if err != nil || withDisabled.Total != 3 {
		t.Fatalf("include disabled employees=%+v err=%v", withDisabled, err)
	}
	withHistory, err := repo.QueryForRead(nil, &request.OrgEmployeeQueryReq{}, table, repository.OrgReadScope{
		AsOf:           model.Now(),
		IncludeHistory: true,
	})
	if err != nil || withHistory.Total != 3 {
		t.Fatalf("include history employees=%+v err=%v", withHistory, err)
	}

	summaries, err := repo.FindBoundUserSummaries(nil, []int{user.Id})
	if err != nil || len(summaries) != 1 ||
		summaries[0].UserId != user.Id ||
		summaries[0].UserName != user.UserName {
		t.Fatalf("safe user summaries=%+v err=%v", summaries, err)
	}
}

func TestOrgPositionRepositoryReadQueryUsesRealUnitOwnership(t *testing.T) {
	db := testutil.OpenSQLite(
		t,
		&model.OrgLegalEntity{},
		&model.OrgUnit{},
		&model.OrgPosition{},
	)
	repo := NewOrgPositionRepositoryImpl(&database.PrimaryDB{DB: db})
	legalA := employeePositionLegalFixture(1, "LE-A")
	legalB := employeePositionLegalFixture(2, "LE-B")
	unitA := employeePositionUnitFixture(10, "OU-A", legalA.Id)
	unitB := employeePositionUnitFixture(20, "OU-B", legalB.Id)
	positionA := employeePositionPositionFixture(100, "POS-A", unitA.Id, "enabled")
	positionB := employeePositionPositionFixture(200, "POS-B", unitB.Id, "disabled")
	testutil.MustCreate(t, db, &[]model.OrgLegalEntity{legalA, legalB})
	testutil.MustCreate(t, db, &[]model.OrgUnit{unitA, unitB})
	testutil.MustCreate(t, db, &[]model.OrgPosition{positionA, positionB})

	result, err := repo.QueryForRead(
		nil,
		&request.OrgPositionQueryReq{
			Basic: request.Basic{
				QuickQuery: &request.QuickQuery{Keyword: "POS-A"},
			},
			LegalEntityId: &legalA.Id,
			OrgUnitId:     &unitA.Id,
		},
		employeePositionRepositoryTable("org_position"),
		repository.OrgReadScope{AsOf: model.Now()},
	)
	if err != nil || result.Total != 1 || result.Data[0].Id != positionA.Id {
		t.Fatalf("position ownership query=%+v err=%v", result, err)
	}

	result, err = repo.QueryForRead(
		nil,
		&request.OrgPositionQueryReq{
			LegalEntityId: &legalA.Id,
			OrgUnitId:     &unitB.Id,
		},
		employeePositionRepositoryTable("org_position"),
		repository.OrgReadScope{AsOf: model.Now(), IncludeDisabled: true},
	)
	if err != nil || result.Total != 0 {
		t.Fatalf("mismatched position ownership result=%+v err=%v", result, err)
	}
}

func employeePositionRepositoryTable(tableCode string) model.SysTable {
	field := func(code string, fieldType enum.SysTableFieldType, quick bool) model.SysTableField {
		return model.SysTableField{
			FieldCode:        code,
			FieldType:        fieldType,
			IsListShow:       true,
			IsQuickSearch:    quick,
			IsAdvancedSearch: true,
			IsSort:           true,
		}
	}
	fields := []model.SysTableField{
		field("id", enum.BigIntFieldType, false),
		field("code", enum.VarcharFieldType, true),
		field("name", enum.VarcharFieldType, true),
		field("status", enum.VarcharFieldType, false),
		field("valid_from", enum.DatetimeFieldType, false),
		field("valid_to", enum.DatetimeFieldType, false),
	}
	if tableCode == "org_employee" {
		fields = []model.SysTableField{
			field("id", enum.BigIntFieldType, false),
			field("employee_no", enum.VarcharFieldType, true),
			field("name", enum.VarcharFieldType, true),
			field("employment_status", enum.VarcharFieldType, false),
			field("valid_from", enum.DatetimeFieldType, false),
			field("valid_to", enum.DatetimeFieldType, false),
		}
	}
	return model.SysTable{
		Basic:       model.Basic{Id: 1, State: true},
		TableCode:   tableCode,
		TableFields: fields,
	}
}

func employeePositionLegalFixture(id int, code string) model.OrgLegalEntity {
	return model.OrgLegalEntity{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-" + code,
		Code:             code,
		Name:             code,
		EntityType:       "legal_company",
		Status:           "enabled",
		SyncStatus:       "synced",
	}
}

func employeePositionUnitFixture(id int, code string, legalEntityId int) model.OrgUnit {
	return model.OrgUnit{
		Basic:                model.Basic{Id: id, State: true},
		SourceSystemCode:     "authority",
		SourceId:             "source-" + code,
		Code:                 code,
		Name:                 code,
		UnitType:             "department",
		PrimaryLegalEntityId: &legalEntityId,
		Status:               "enabled",
		SyncStatus:           "synced",
	}
}

func employeePositionPositionFixture(id int, code string, orgUnitId int, status string) model.OrgPosition {
	return model.OrgPosition{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-" + code,
		Code:             code,
		Name:             code,
		OrgUnitId:        orgUnitId,
		PositionType:     "professional",
		Status:           status,
		SyncStatus:       "synced",
	}
}

func employeePositionEmployeeFixture(id int, employeeNo, name, status string) model.OrgEmployee {
	return model.OrgEmployee{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-" + employeeNo,
		EmployeeNo:       employeeNo,
		Name:             name,
		EmploymentStatus: status,
		SyncStatus:       "synced",
	}
}

func employeePositionAssignmentFixture(
	id int,
	employeeId int,
	legalEntityId int,
	orgUnitId int,
	positionId int,
) model.OrgAssignment {
	return model.OrgAssignment{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "assignment-" + time.Unix(int64(id), 0).UTC().Format("150405"),
		EmployeeId:       employeeId,
		LegalEntityId:    legalEntityId,
		OrgUnitId:        orgUnitId,
		PositionId:       &positionId,
		AssignmentType:   "part_time",
		Status:           "enabled",
		SyncStatus:       "synced",
	}
}
