package service

import (
	"backend/dto/response"
	"backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"reflect"
	"testing"
	"time"
)

func TestOrgPermissionProviderGetEmployeeByUser(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	boundUser := model.SysUser{
		Basic:    model.Basic{Id: 1001, State: true},
		UserName: "permission-bound-user",
	}
	unboundUser := model.SysUser{
		Basic:    model.Basic{Id: 1002, State: true},
		UserName: "permission-unbound-user",
	}
	testutil.MustCreate(t, db, &[]model.SysUser{boundUser, unboundUser})

	employee := orgServiceEmployeeFixture(2001, "EMP-PERM-001", "权限员工", "active")
	employee.UserId = &boundUser.Id
	testutil.MustCreate(t, db, &employee)

	bound, err := orgService.GetEmployeeByUser(nil, boundUser.Id)
	if err != nil {
		t.Fatalf("get bound employee context: %v", err)
	}
	if bound.UserId != boundUser.Id ||
		bound.EmployeeId == nil ||
		*bound.EmployeeId != employee.Id ||
		bound.BindingStatus != response.OrgEmployeeBindingBound {
		t.Fatalf("unexpected bound employee context: %+v", bound)
	}

	unbound, err := orgService.GetEmployeeByUser(nil, unboundUser.Id)
	if err != nil {
		t.Fatalf("get unbound employee context: %v", err)
	}
	if unbound.UserId != unboundUser.Id ||
		unbound.EmployeeId != nil ||
		unbound.BindingStatus != response.OrgEmployeeBindingUnbound {
		t.Fatalf("unexpected unbound employee context: %+v", unbound)
	}

	_, err = orgService.GetEmployeeByUser(nil, 99999)
	assertOrgServiceAdminError(
		t,
		err,
		errors.CategoryBusiness,
		errors.ErrorCodeOrgUserNotFound,
	)
}

func TestOrgPermissionProviderGetEffectiveAssignmentsSingle(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	asOf := orgServiceDate(t, "2026-07-27")
	legal, _, unit, _, position, _ := seedEmployeePositionOwnership(t, db)
	employee := orgServiceEmployeeFixture(2051, "EMP-PERM-SINGLE", "单任职员工", "active")
	testutil.MustCreate(t, db, &employee)
	assignment := orgPermissionAssignmentFixture(
		2052,
		employee.Id,
		legal.Id,
		unit.Id,
		&position.Id,
		asOf.AddDate(0, -1, 0),
		nil,
	)
	assignment.IsPrimary = false
	testutil.MustCreate(t, db, &assignment)

	result, err := orgService.GetEffectiveAssignments(nil, employee.Id, "2026-07-27")
	if err != nil {
		t.Fatalf("get single effective assignment: %v", err)
	}
	if len(result) != 1 ||
		result[0].AssignmentId != assignment.Id ||
		result[0].EmployeeId != employee.Id ||
		result[0].LegalEntityId != legal.Id ||
		result[0].OrgUnitId != unit.Id {
		t.Fatalf("unexpected single effective assignment: %+v", result)
	}
}

func TestOrgPermissionProviderGetEffectiveAssignmentsFiltersAndSorts(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	asOf := orgServiceDate(t, "2026-07-27")
	legalA, legalB, unitA, unitB, positionA, positionB :=
		seedEmployeePositionOwnership(t, db)
	employee := orgServiceEmployeeFixture(2101, "EMP-PERM-002", "多任职员工", "active")
	testutil.MustCreate(t, db, &employee)

	currentLater := orgPermissionAssignmentFixture(
		2202,
		employee.Id,
		legalB.Id,
		unitB.Id,
		&positionB.Id,
		asOf.AddDate(0, -1, 0),
		nil,
	)
	currentLater.IsPrimary = true
	currentFirst := orgPermissionAssignmentFixture(
		2201,
		employee.Id,
		legalA.Id,
		unitA.Id,
		&positionA.Id,
		asOf,
		&asOf,
	)
	currentFirst.IsPrimary = false
	currentSameScope := orgPermissionAssignmentFixture(
		2203,
		employee.Id,
		legalA.Id,
		unitA.Id,
		nil,
		asOf.AddDate(0, -2, 0),
		nil,
	)
	historicalEnd := asOf.AddDate(0, 0, -1)
	historical := orgPermissionAssignmentFixture(
		2204,
		employee.Id,
		legalA.Id,
		unitA.Id,
		&positionA.Id,
		asOf.AddDate(-1, 0, 0),
		&historicalEnd,
	)
	future := orgPermissionAssignmentFixture(
		2205,
		employee.Id,
		legalB.Id,
		unitB.Id,
		&positionB.Id,
		asOf.AddDate(0, 0, 1),
		nil,
	)
	disabled := orgPermissionAssignmentFixture(
		2206,
		employee.Id,
		legalB.Id,
		unitB.Id,
		&positionB.Id,
		asOf.AddDate(0, -1, 0),
		nil,
	)
	disabled.Status = "disabled"
	sourceDeleted := orgPermissionAssignmentFixture(
		2207,
		employee.Id,
		legalB.Id,
		unitB.Id,
		&positionB.Id,
		asOf.AddDate(0, -1, 0),
		nil,
	)
	sourceDeleted.SourceDeleted = true
	testutil.MustCreate(t, db, &[]model.OrgAssignment{
		currentLater,
		currentFirst,
		currentSameScope,
		historical,
		future,
		disabled,
		sourceDeleted,
	})

	result, err := orgService.GetEffectiveAssignments(nil, employee.Id, "2026-07-27")
	if err != nil {
		t.Fatalf("get effective assignments: %v", err)
	}
	wantIds := []int{currentFirst.Id, currentSameScope.Id, currentLater.Id}
	if len(result) != len(wantIds) {
		t.Fatalf("effective assignments=%+v, want ids=%v", result, wantIds)
	}
	for index, wantId := range wantIds {
		if result[index].AssignmentId != wantId {
			t.Fatalf(
				"effective assignment[%d].id=%d, want %d: %+v",
				index,
				result[index].AssignmentId,
				wantId,
				result,
			)
		}
	}
	if result[0].ValidFrom == nil || !result[0].ValidFrom.Equal(asOf) ||
		result[0].ValidTo == nil || !result[0].ValidTo.Equal(asOf) {
		t.Fatalf("inclusive assignment boundary lost: %+v", result[0])
	}
}

func TestOrgPermissionProviderEffectiveOrganizationScopeDeduplicatesAndSorts(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	asOf := orgServiceDate(t, "2026-07-27")
	legalA, legalB, unitA, unitB, positionA, positionB :=
		seedEmployeePositionOwnership(t, db)
	employee := orgServiceEmployeeFixture(2301, "EMP-PERM-003", "范围员工", "active")
	emptyEmployee := orgServiceEmployeeFixture(2302, "EMP-PERM-004", "空范围员工", "active")
	testutil.MustCreate(t, db, &[]model.OrgEmployee{employee, emptyEmployee})

	first := orgPermissionAssignmentFixture(
		2303,
		employee.Id,
		legalB.Id,
		unitB.Id,
		&positionB.Id,
		asOf.AddDate(0, -2, 0),
		nil,
	)
	first.IsPrimary = false
	duplicateScope := orgPermissionAssignmentFixture(
		2304,
		employee.Id,
		legalB.Id,
		unitB.Id,
		nil,
		asOf.AddDate(0, -1, 0),
		nil,
	)
	duplicateScope.IsPrimary = false
	second := orgPermissionAssignmentFixture(
		2305,
		employee.Id,
		legalA.Id,
		unitA.Id,
		&positionA.Id,
		asOf,
		nil,
	)
	second.IsPrimary = false
	testutil.MustCreate(t, db, &[]model.OrgAssignment{first, duplicateScope, second})

	scope, err := orgService.GetEmployeeEffectiveOrganizationScope(
		nil,
		employee.Id,
		"2026-07-27",
	)
	if err != nil {
		t.Fatalf("get effective organization scope: %v", err)
	}
	if scope.ScopeStatus != response.OrgEffectiveScopeResolved ||
		scope.AssignmentCount != 3 ||
		scope.AsOfDate != "2026-07-27" {
		t.Fatalf("unexpected resolved scope: %+v", scope)
	}
	if !reflect.DeepEqual(scope.LegalEntityIds, []int{legalA.Id, legalB.Id}) {
		t.Fatalf("legal entity ids=%v", scope.LegalEntityIds)
	}
	if !reflect.DeepEqual(scope.OrgUnitIds, []int{unitA.Id, unitB.Id}) {
		t.Fatalf("org unit ids=%v", scope.OrgUnitIds)
	}

	empty, err := orgService.GetEmployeeEffectiveOrganizationScope(
		nil,
		emptyEmployee.Id,
		"2026-07-27",
	)
	if err != nil {
		t.Fatalf("get empty organization scope: %v", err)
	}
	if empty.ScopeStatus != response.OrgEffectiveScopeEmpty ||
		empty.AssignmentCount != 0 ||
		len(empty.LegalEntityIds) != 0 ||
		len(empty.OrgUnitIds) != 0 ||
		empty.LegalEntityIds == nil ||
		empty.OrgUnitIds == nil {
		t.Fatalf("unexpected empty scope: %+v", empty)
	}
}

func TestOrgPermissionProviderRejectsInactiveEmployeeAndReferences(t *testing.T) {
	t.Run("inactive employee", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		employee := orgServiceEmployeeFixture(2401, "EMP-PERM-005", "停用员工", "terminated")
		testutil.MustCreate(t, db, &employee)

		_, err := orgService.GetEffectiveAssignments(nil, employee.Id, "2026-07-27")
		assertOrgServiceAdminError(
			t,
			err,
			errors.CategoryBusiness,
			errors.ErrorCodeOrgEmployeeInactive,
		)
	})

	t.Run("inactive legal entity", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		asOf := orgServiceDate(t, "2026-07-27")
		legal, _, unit, _, position, _ := seedEmployeePositionOwnership(t, db)
		employee := orgServiceEmployeeFixture(2402, "EMP-PERM-006", "法人异常员工", "active")
		legal.Status = "disabled"
		if err := db.Model(&model.OrgLegalEntity{}).
			Where("id = ?", legal.Id).
			Update("status", legal.Status).Error; err != nil {
			t.Fatalf("disable legal entity: %v", err)
		}
		testutil.MustCreate(t, db, &employee)
		assignment := orgPermissionAssignmentFixture(
			2403,
			employee.Id,
			legal.Id,
			unit.Id,
			&position.Id,
			asOf,
			nil,
		)
		testutil.MustCreate(t, db, &assignment)

		_, err := orgService.GetEffectiveAssignments(nil, employee.Id, "2026-07-27")
		assertOrgServiceAdminError(
			t,
			err,
			errors.CategoryBusiness,
			errors.ErrorCodeOrgLegalEntityInactive,
		)
	})

	t.Run("inactive org unit", func(t *testing.T) {
		orgService, db := newOrgServiceTestSubject(t)
		asOf := orgServiceDate(t, "2026-07-27")
		legal, _, unit, _, position, _ := seedEmployeePositionOwnership(t, db)
		employee := orgServiceEmployeeFixture(2404, "EMP-PERM-007", "组织异常员工", "active")
		if err := db.Model(&model.OrgUnit{}).
			Where("id = ?", unit.Id).
			Update("status", "disabled").Error; err != nil {
			t.Fatalf("disable org unit: %v", err)
		}
		testutil.MustCreate(t, db, &employee)
		assignment := orgPermissionAssignmentFixture(
			2405,
			employee.Id,
			legal.Id,
			unit.Id,
			&position.Id,
			asOf,
			nil,
		)
		testutil.MustCreate(t, db, &assignment)

		_, err := orgService.GetEffectiveAssignments(nil, employee.Id, "2026-07-27")
		assertOrgServiceAdminError(
			t,
			err,
			errors.CategoryBusiness,
			errors.ErrorCodeOrgUnitInactive,
		)
	})
}

func orgPermissionAssignmentFixture(
	id int,
	employeeId int,
	legalEntityId int,
	orgUnitId int,
	positionId *int,
	validFrom time.Time,
	validTo *time.Time,
) model.OrgAssignment {
	assignment := orgAssignmentFixture(
		id,
		employeeId,
		legalEntityId,
		orgUnitId,
		positionId,
		validFrom,
		validTo,
		"enabled",
	)
	assignment.AssignmentType = "part_time"
	return assignment
}
