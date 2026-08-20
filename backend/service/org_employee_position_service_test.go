package service

import (
	"backend/dto/request"
	"backend/enum"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestOrgServiceEmployeeQueryKeepsAssignmentFiltersOnOneRow(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	legalA, legalB, unitA, unitB, positionA, positionB := seedEmployeePositionOwnership(t, db)
	employeeA := orgServiceEmployeeFixture(1000, "EMP-001", "张三", "active")
	employeeB := orgServiceEmployeeFixture(1001, "EMP-002", "李四", "probation")
	testutil.MustCreate(t, db, &[]model.OrgEmployee{employeeA, employeeB})
	testutil.MustCreate(t, db, &[]model.OrgAssignment{
		orgServiceAssignmentFixture(1, employeeA.Id, legalA.Id, unitA.Id, positionA.Id),
		orgServiceAssignmentFixture(2, employeeA.Id, legalB.Id, unitB.Id, positionB.Id),
		orgServiceAssignmentFixture(3, employeeA.Id, legalA.Id, unitA.Id, positionA.Id),
		orgServiceAssignmentFixture(4, employeeB.Id, legalB.Id, unitB.Id, positionB.Id),
	})

	result, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		Basic: request.Basic{
			Page:       1,
			Num:        10,
			QuickQuery: &request.QuickQuery{Keyword: "EMP-001"},
		},
		LegalEntityId: &legalA.Id,
		OrgUnitId:     &unitA.Id,
		PositionId:    &positionA.Id,
	}, orgEmployeePositionServiceTable("org_employee"))
	if err != nil {
		t.Fatalf("query employees: %v", err)
	}
	if result.Total != 1 || len(result.Data) != 1 || result.Data[0].Id != employeeA.Id {
		t.Fatalf("multi-assignment pagination was not stable: %+v", result)
	}

	result, err = orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		OrgUnitId:  &unitA.Id,
		PositionId: &positionB.Id,
	}, orgEmployeePositionServiceTable("org_employee"))
	if err != nil {
		t.Fatalf("query split assignment filters: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("filters from different assignments matched unexpectedly: %+v", result)
	}

	advanced, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		Basic: request.Basic{
			Expressions: []request.ExpressionGroup{{
				Logic: enum.And,
				Rules: []request.QueryRule{{
					Field:          "name",
					ExpressionType: enum.Eq,
					Value:          "李四",
				}},
			}},
		},
	}, orgEmployeePositionServiceTable("org_employee"))
	if err != nil || advanced.Total != 1 || advanced.Data[0].Id != employeeB.Id {
		t.Fatalf("advanced employee query=%+v err=%v", advanced, err)
	}

	options, err := orgService.QueryEmployeeOptions(nil, request.OrgEmployeeOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
			Page: 1,
			Num:  10,
		},
		LegalEntityId: &legalA.Id,
		OrgUnitId:     &unitA.Id,
		PositionId:    &positionA.Id,
	}, orgEmployeePositionServiceTable("org_employee"))
	if err != nil || options.Total != 1 ||
		len(options.Items) != 1 ||
		options.Items[0].Value != employeeA.Id {
		t.Fatalf("assignment-filtered employee options=%+v err=%v", options, err)
	}
}

func TestOrgServiceEmployeeProjectsPrimaryLegalEntityName(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	legal := orgServiceLegalEntity(701, "LE-701", "华东法人", "", "enabled", nil, nil)
	testutil.MustCreate(t, db, &legal)
	employee := orgServiceEmployeeFixture(702, "EMP-702", "展示人员", "active")
	employee.PrimaryLegalEntityId = &legal.Id
	testutil.MustCreate(t, db, &employee)

	result, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{Basic: request.Basic{Page: 1, Num: 10}}, orgEmployeePositionServiceTable("org_employee"))
	if err != nil || len(result.Data) != 1 {
		t.Fatalf("query employee projection=%+v err=%v", result, err)
	}
	projection := result.Data[0].PrimaryLegalEntity
	if projection == nil || projection.Id != legal.Id || projection.Name != legal.Name {
		t.Fatalf("primary legal entity projection=%+v", projection)
	}
}

func TestOrgServiceEmployeeScopeBindingDetailAndOptions(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	user := model.SysUser{
		Basic:        model.Basic{Id: 50, State: true},
		UserName:     "employee_account",
		Password:     "password-must-not-leak",
		AccessTokens: "token-must-not-leak",
		Email:        "account@example.com",
	}
	testutil.MustCreate(t, db, &user)
	bound := orgServiceEmployeeFixture(1, "EMP-A", "王一", "active")
	bound.UserId = &user.Id
	bound.Mobile = "13800138000"
	bound.Email = "employee@example.com"
	bound.SourceVersion = "source-version-secret"
	unbound := orgServiceEmployeeFixture(2, "EMP-B", "王二", "probation")
	resigned := orgServiceEmployeeFixture(3, "EMP-C", "王三", "resigned")
	expired := orgServiceEmployeeFixture(4, "EMP-D", "王四", "active")
	expiredAt := model.Now().Add(-24 * time.Hour)
	expired.ValidTo = &expiredAt
	sourceDeleted := orgServiceEmployeeFixture(5, "EMP-E", "王五", "active")
	sourceDeleted.SourceDeleted = true
	testutil.MustCreate(t, db, &[]model.OrgEmployee{
		bound,
		unbound,
		resigned,
		expired,
		sourceDeleted,
	})

	table := orgEmployeePositionServiceTable("org_employee")
	active, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{}, table)
	if err != nil || active.Total != 2 {
		t.Fatalf("default employee scope=%+v err=%v", active, err)
	}
	boundRows, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		BoundStatus: "bound",
	}, table)
	if err != nil || boundRows.Total != 1 ||
		boundRows.Data[0].BindingStatus != "bound" ||
		boundRows.Data[0].BoundAccount == nil ||
		boundRows.Data[0].BoundAccount.UserName != user.UserName {
		t.Fatalf("bound employee result=%+v err=%v", boundRows, err)
	}
	unboundRows, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		BoundStatus: "unbound",
	}, table)
	if err != nil || unboundRows.Total != 1 ||
		unboundRows.Data[0].BindingStatus != "unbound" {
		t.Fatalf("unbound employee result=%+v err=%v", unboundRows, err)
	}
	withDisabled, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeDisabled: true},
	}, table)
	if err != nil || withDisabled.Total != 3 {
		t.Fatalf("include disabled employees=%+v err=%v", withDisabled, err)
	}
	withHistory, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeHistory: true},
	}, table)
	if err != nil || withHistory.Total != 4 {
		t.Fatalf("include history employees=%+v err=%v", withHistory, err)
	}
	contactSearch, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		Basic: request.Basic{
			QuickQuery: &request.QuickQuery{Keyword: bound.Mobile},
		},
	}, table)
	if err != nil || contactSearch.Total != 0 {
		t.Fatalf("contact fields crossed committed quick-query boundary: %+v err=%v", contactSearch, err)
	}

	detail, err := orgService.GetEmployeeDetail(nil, bound.Id, request.OrgEmployeeDetailReq{})
	if err != nil {
		t.Fatalf("get employee detail: %v", err)
	}
	if detail.MobileMasked != "138****8000" ||
		detail.EmailMasked != "e***@example.com" ||
		detail.BoundAccount == nil ||
		detail.BoundAccount.UserName != user.UserName {
		t.Fatalf("unsafe or incomplete employee detail: %+v", detail)
	}
	serialized, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal employee detail: %v", err)
	}
	for _, forbidden := range []string{
		bound.Mobile,
		bound.Email,
		bound.SourceVersion,
		user.Password,
		user.AccessTokens,
		user.Email,
	} {
		if forbidden != "" && strings.Contains(string(serialized), forbidden) {
			t.Fatalf("employee detail leaked %q: %s", forbidden, serialized)
		}
	}

	_, err = orgService.GetEmployeeDetail(nil, 0, request.OrgEmployeeDetailReq{})
	assertOrgServiceAdminError(t, err, apperrors.CategoryParameter, apperrors.ErrorCodeParamInvalid)
	_, err = orgService.GetEmployeeDetail(nil, 999, request.OrgEmployeeDetailReq{})
	assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgEmployeeNotFound)

	options, err := orgService.QueryEmployeeOptions(nil, request.OrgEmployeeOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
			Page:        1,
			Num:         10,
			Keyword:     "EMP-A",
			SelectedIds: []int{resigned.Id, expired.Id},
		},
	}, table)
	if err != nil || options.Total != 1 || len(options.Items) != 3 {
		t.Fatalf("employee options=%+v err=%v", options, err)
	}
	if options.Items[0].Value != bound.Id || options.Items[0].Label != "EMP-A - 王一" {
		t.Fatalf("employee option value/label invalid: %+v", options.Items[0])
	}
	for _, item := range options.Items[1:] {
		if !item.Disabled {
			t.Fatalf("historical employee replay must be disabled: %+v", item)
		}
	}
	tooMany := make([]int, 101)
	for index := range tooMany {
		tooMany[index] = index + 1
	}
	_, err = orgService.QueryEmployeeOptions(nil, request.OrgEmployeeOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{SelectedIds: tooMany},
	}, table)
	assertOrgServiceAdminError(t, err, apperrors.CategoryParameter, apperrors.ErrorCodeParamInvalid)
}

func TestOrgServiceEmployeeListBatchesBoundAccountLookup(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	users := []model.SysUser{
		{Basic: model.Basic{Id: 1, State: true}, UserName: "user_1"},
		{Basic: model.Basic{Id: 2, State: true}, UserName: "user_2"},
	}
	testutil.MustCreate(t, db, &users)
	employees := []model.OrgEmployee{
		orgServiceEmployeeFixture(1, "EMP-1", "人员一", "active"),
		orgServiceEmployeeFixture(2, "EMP-2", "人员二", "active"),
		organizationEmployeeWithoutUser(3, "EMP-3", "人员三"),
	}
	employees[0].UserId = &users[0].Id
	employees[1].UserId = &users[1].Id
	testutil.MustCreate(t, db, &employees)

	var queryCount atomic.Int32
	callbackName := "test:count-employee-list-queries"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(*gorm.DB) {
		queryCount.Add(1)
	}); err != nil {
		t.Fatalf("register employee query counter: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})

	result, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		Basic: request.Basic{Page: 1, Num: 10},
	}, orgEmployeePositionServiceTable("org_employee"))
	if err != nil || result.Total != 3 {
		t.Fatalf("query employee list=%+v err=%v", result, err)
	}
	if got := queryCount.Load(); got != 3 {
		t.Fatalf("employee list query count=%d, want count+rows+one account batch", got)
	}
}

func TestOrgServicePositionQueryDetailAndOptions(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	legalA, legalB, unitA, unitB, positionA, positionB := seedEmployeePositionOwnership(t, db)
	positionB.Status = "disabled"
	if err := db.Model(&model.OrgPosition{}).
		Where("id = ?", positionB.Id).
		Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable position fixture: %v", err)
	}
	expired := orgServicePositionFixture(300, "POS-H", "历史岗位", unitA.Id, "enabled")
	expiredAt := model.Now().Add(-24 * time.Hour)
	expired.ValidTo = &expiredAt
	sourceDeleted := orgServicePositionFixture(301, "POS-D", "源删除岗位", unitA.Id, "enabled")
	sourceDeleted.SourceDeleted = true
	testutil.MustCreate(t, db, &[]model.OrgPosition{expired, sourceDeleted})

	table := orgEmployeePositionServiceTable("org_position")
	result, err := orgService.QueryPositions(nil, request.OrgPositionQueryReq{
		Basic: request.Basic{
			Page:       1,
			Num:        10,
			QuickQuery: &request.QuickQuery{Keyword: "POS-A"},
		},
		LegalEntityId: &legalA.Id,
		OrgUnitId:     &unitA.Id,
	}, table)
	if err != nil || result.Total != 1 || result.Data[0].Id != positionA.Id {
		t.Fatalf("position query=%+v err=%v", result, err)
	}
	mismatch, err := orgService.QueryPositions(nil, request.OrgPositionQueryReq{
		LegalEntityId: &legalA.Id,
		OrgUnitId:     &unitB.Id,
	}, table)
	if err != nil || mismatch.Total != 0 {
		t.Fatalf("position ownership mismatch=%+v err=%v", mismatch, err)
	}
	advanced, err := orgService.QueryPositions(nil, request.OrgPositionQueryReq{
		Basic: request.Basic{
			Expressions: []request.ExpressionGroup{{
				Logic: enum.And,
				Rules: []request.QueryRule{{
					Field:          "name",
					ExpressionType: enum.Eq,
					Value:          positionA.Name,
				}},
			}},
		},
	}, table)
	if err != nil || advanced.Total != 1 {
		t.Fatalf("advanced position query=%+v err=%v", advanced, err)
	}
	withDisabled, err := orgService.QueryPositions(nil, request.OrgPositionQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeDisabled: true},
	}, table)
	if err != nil || withDisabled.Total != 2 {
		t.Fatalf("include disabled positions=%+v err=%v", withDisabled, err)
	}
	withHistory, err := orgService.QueryPositions(nil, request.OrgPositionQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeHistory: true},
	}, table)
	if err != nil || withHistory.Total != 3 {
		t.Fatalf("include history positions=%+v err=%v", withHistory, err)
	}
	disabledOnly, err := orgService.QueryPositions(nil, request.OrgPositionQueryReq{
		OrgReadScopeReq: request.OrgReadScopeReq{IncludeDisabled: true},
		Status:          "disabled",
	}, table)
	if err != nil || disabledOnly.Total != 1 || disabledOnly.Data[0].Id != positionB.Id {
		t.Fatalf("disabled position filter=%+v err=%v", disabledOnly, err)
	}

	detail, err := orgService.GetPositionDetail(nil, positionA.Id, request.OrgPositionDetailReq{})
	if err != nil ||
		detail.OrgUnit == nil ||
		detail.OrgUnit.Id != unitA.Id ||
		detail.LegalEntity == nil ||
		detail.LegalEntity.Id != legalA.Id {
		t.Fatalf("position detail=%+v err=%v", detail, err)
	}
	assertOrganizationResponseDoesNotLeakSourceFields(t, detail)
	_, err = orgService.GetPositionDetail(nil, 999, request.OrgPositionDetailReq{})
	assertOrgServiceAdminError(t, err, apperrors.CategoryBusiness, apperrors.ErrorCodeOrgPositionNotFound)

	options, err := orgService.QueryPositionOptions(nil, request.OrgPositionOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
			Page:        1,
			Num:         10,
			Keyword:     "POS-A",
			SelectedIds: []int{positionB.Id, expired.Id},
		},
		LegalEntityId: &legalA.Id,
		OrgUnitId:     &unitA.Id,
	}, table)
	if err != nil || options.Total != 1 || len(options.Items) != 3 {
		t.Fatalf("position options=%+v err=%v", options, err)
	}
	if options.Items[0].Value != positionA.Id ||
		options.Items[0].Label != positionA.Code+" - "+positionA.Name {
		t.Fatalf("position option value/label invalid: %+v", options.Items[0])
	}
	for _, item := range options.Items[1:] {
		if !item.Disabled {
			t.Fatalf("historical position replay must be disabled: %+v", item)
		}
	}
	_ = legalB
}

func TestOrgServiceEmployeeAndPositionAsOfDateIncludesBoundaries(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	asOf := time.Date(2026, time.July, 26, 0, 0, 0, 0, model.AppLocation())
	nextDay := asOf.AddDate(0, 0, 1)
	legal := orgServiceLegalEntity(10, "LE-A", "法人甲", "甲", "enabled", nil, nil)
	unit := managementUnitFixture(20, "OU-A", "组织甲", "department", "enabled", &legal.Id)
	positionAtBoundary := orgServicePositionFixture(30, "POS-A", "岗位甲", unit.Id, "enabled")
	positionAtBoundary.ValidFrom = &asOf
	positionAtBoundary.ValidTo = &asOf
	positionFuture := orgServicePositionFixture(31, "POS-B", "岗位乙", unit.Id, "enabled")
	positionFuture.ValidFrom = &nextDay
	employeeAtBoundary := orgServiceEmployeeFixture(40, "EMP-A", "人员甲", "active")
	employeeAtBoundary.ValidFrom = &asOf
	employeeAtBoundary.ValidTo = &asOf
	employeeFuture := orgServiceEmployeeFixture(41, "EMP-B", "人员乙", "active")
	employeeFuture.ValidFrom = &nextDay
	testutil.MustCreate(t, db, &legal)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &[]model.OrgPosition{positionAtBoundary, positionFuture})
	testutil.MustCreate(t, db, &[]model.OrgEmployee{employeeAtBoundary, employeeFuture})

	scope := request.OrgReadScopeReq{AsOfDate: "2026-07-26"}
	employees, err := orgService.QueryEmployees(nil, request.OrgEmployeeQueryReq{
		OrgReadScopeReq: scope,
	}, orgEmployeePositionServiceTable("org_employee"))
	if err != nil || employees.Total != 1 || employees.Data[0].Id != employeeAtBoundary.Id {
		t.Fatalf("employee as_of boundary=%+v err=%v", employees, err)
	}
	positions, err := orgService.QueryPositions(nil, request.OrgPositionQueryReq{
		OrgReadScopeReq: scope,
	}, orgEmployeePositionServiceTable("org_position"))
	if err != nil || positions.Total != 1 || positions.Data[0].Id != positionAtBoundary.Id {
		t.Fatalf("position as_of boundary=%+v err=%v", positions, err)
	}
}

func seedEmployeePositionOwnership(
	t *testing.T,
	db *gorm.DB,
) (
	model.OrgLegalEntity,
	model.OrgLegalEntity,
	model.OrgUnit,
	model.OrgUnit,
	model.OrgPosition,
	model.OrgPosition,
) {
	t.Helper()
	legalA := orgServiceLegalEntity(10, "LE-A", "法人甲", "甲", "enabled", nil, nil)
	legalB := orgServiceLegalEntity(11, "LE-B", "法人乙", "乙", "enabled", nil, nil)
	unitA := managementUnitFixture(20, "OU-A", "组织甲", "department", "enabled", &legalA.Id)
	unitB := managementUnitFixture(21, "OU-B", "组织乙", "department", "enabled", &legalB.Id)
	positionA := orgServicePositionFixture(30, "POS-A", "岗位甲", unitA.Id, "enabled")
	positionB := orgServicePositionFixture(31, "POS-B", "岗位乙", unitB.Id, "enabled")
	testutil.MustCreate(t, db, &[]model.OrgLegalEntity{legalA, legalB})
	testutil.MustCreate(t, db, &[]model.OrgUnit{unitA, unitB})
	testutil.MustCreate(t, db, &[]model.OrgPosition{positionA, positionB})
	return legalA, legalB, unitA, unitB, positionA, positionB
}

func orgServiceEmployeeFixture(id int, employeeNo, name, status string) model.OrgEmployee {
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

func organizationEmployeeWithoutUser(id int, employeeNo, name string) model.OrgEmployee {
	return orgServiceEmployeeFixture(id, employeeNo, name, "active")
}

func orgServicePositionFixture(id int, code, name string, unitId int, status string) model.OrgPosition {
	return model.OrgPosition{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-" + code,
		Code:             code,
		Name:             name,
		OrgUnitId:        unitId,
		PositionType:     "professional",
		Status:           status,
		SyncStatus:       "synced",
	}
}

func orgServiceAssignmentFixture(
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

func orgEmployeePositionServiceTable(tableCode string) model.SysTable {
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
		field("org_unit_id", enum.BigIntFieldType, false),
		field("position_type", enum.VarcharFieldType, false),
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
			field("primary_legal_entity_id", enum.BigIntFieldType, false),
			field("user_id", enum.BigIntFieldType, false),
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
