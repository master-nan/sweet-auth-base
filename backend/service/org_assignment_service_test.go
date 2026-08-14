package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/enum"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOrgServiceAssignmentTemporalQueriesAndTimeline(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	asOf := time.Date(2026, 7, 26, 0, 0, 0, 0, model.AppLocation())
	legalA, legalB, unitA, unitB, positionA, positionB := seedEmployeePositionOwnership(t, db)
	employee := orgServiceEmployeeFixture(700, "EMP-700", "任职测试人员", "active")
	testutil.MustCreate(t, db, &employee)

	currentOld := orgAssignmentFixture(
		701,
		employee.Id,
		legalA.Id,
		unitA.Id,
		&positionA.Id,
		asOf.AddDate(0, -2, 0),
		nil,
		"enabled",
	)
	currentOld.IsPrimary = true
	currentBoundary := orgAssignmentFixture(
		702,
		employee.Id,
		legalB.Id,
		unitB.Id,
		&positionB.Id,
		asOf,
		&asOf,
		"enabled",
	)
	history := orgAssignmentFixture(
		703,
		employee.Id,
		legalA.Id,
		unitA.Id,
		&positionA.Id,
		asOf.AddDate(-1, 0, 0),
		assignmentTimePointer(asOf.Add(-24*time.Hour)),
		"enabled",
	)
	future := orgAssignmentFixture(
		704,
		employee.Id,
		legalB.Id,
		unitB.Id,
		&positionB.Id,
		asOf.Add(24*time.Hour),
		nil,
		"enabled",
	)
	testutil.MustCreate(t, db, &[]model.OrgAssignment{
		currentOld,
		currentBoundary,
		history,
		future,
	})

	table := orgAssignmentServiceTable()
	query := func(timeScope string) response.ListResult[response.OrgAssignmentListRes] {
		t.Helper()
		result, err := orgService.QueryAssignments(nil, request.OrgAssignmentQueryReq{
			Basic:      request.Basic{Page: 1, Num: 20},
			EmployeeId: &employee.Id,
			TimeScope:  timeScope,
			AsOfDate:   "2026-07-26",
		}, table)
		if err != nil {
			t.Fatalf("query %s assignments: %v", timeScope, err)
		}
		return result
	}

	current := query(request.OrgAssignmentScopeCurrent)
	if current.Total != 2 || len(current.Data) != 2 {
		t.Fatalf("current assignments=%+v", current)
	}
	if current.Data[0].Id != currentBoundary.Id ||
		current.Data[1].Id != currentOld.Id {
		t.Fatalf("current ordering=%v,%v", current.Data[0].Id, current.Data[1].Id)
	}
	if current.Data[0].TimeScope != request.OrgAssignmentScopeCurrent ||
		current.Data[1].TimeScope != request.OrgAssignmentScopeCurrent {
		t.Fatalf("current scope classification=%+v", current.Data)
	}
	if current.Data[0].OrgUnit == nil || current.Data[0].Position == nil ||
		current.Data[1].OrgUnit == nil || current.Data[1].Position == nil {
		t.Fatalf("current assignment summaries missing: %+v", current.Data)
	}

	historical := query(request.OrgAssignmentScopeHistory)
	if historical.Total != 1 || historical.Data[0].Id != history.Id ||
		historical.Data[0].TimeScope != request.OrgAssignmentScopeHistory {
		t.Fatalf("historical assignments=%+v", historical)
	}

	upcoming := query(request.OrgAssignmentScopeFuture)
	if upcoming.Total != 1 || upcoming.Data[0].Id != future.Id ||
		upcoming.Data[0].TimeScope != request.OrgAssignmentScopeFuture {
		t.Fatalf("future assignments=%+v", upcoming)
	}

	timeline := query(request.OrgAssignmentScopeTimeline)
	wantTimeline := []int{future.Id, currentBoundary.Id, currentOld.Id, history.Id}
	if timeline.Total != len(wantTimeline) || len(timeline.Data) != len(wantTimeline) {
		t.Fatalf("timeline assignments=%+v", timeline)
	}
	for index, wantId := range wantTimeline {
		if timeline.Data[index].Id != wantId {
			t.Fatalf(
				"timeline[%d].id=%d, want %d: %+v",
				index,
				timeline.Data[index].Id,
				wantId,
				timeline.Data,
			)
		}
	}
	if timeline.Data[0].TimeScope != request.OrgAssignmentScopeFuture ||
		timeline.Data[len(timeline.Data)-1].TimeScope != request.OrgAssignmentScopeHistory {
		t.Fatalf("timeline classifications=%+v", timeline.Data)
	}

	summary, err := orgService.GetEmployeeCurrentAssignmentSummary(
		nil,
		employee.Id,
		request.OrgEmployeeCurrentAssignmentSummaryReq{AsOfDate: "2026-07-26"},
		table,
	)
	if err != nil {
		t.Fatalf("current assignment summary: %v", err)
	}
	if summary.AssignmentCount != 2 ||
		len(summary.LegalEntities) != 2 ||
		len(summary.OrgUnits) != 2 ||
		len(summary.Positions) != 2 {
		t.Fatalf("current assignment summary=%+v", summary)
	}
	if summary.AsOfDate != "2026-07-26" {
		t.Fatalf("summary as_of_date=%q", summary.AsOfDate)
	}
}

func TestOrgServiceAssignmentHistoricalReferenceReplayAndWhitelist(t *testing.T) {
	orgService, db := newOrgServiceTestSubject(t)
	asOf := time.Date(2026, 7, 26, 0, 0, 0, 0, model.AppLocation())
	legal, _, unit, _, position, _ := seedEmployeePositionOwnership(t, db)
	employee := orgServiceEmployeeFixture(710, "EMP-710", "历史任职人员", "active")
	testutil.MustCreate(t, db, &employee)

	if err := db.Model(&model.OrgLegalEntity{}).
		Where("id = ?", legal.Id).
		Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable legal entity: %v", err)
	}
	if err := db.Model(&model.OrgUnit{}).
		Where("id = ?", unit.Id).
		Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable org unit: %v", err)
	}
	if err := db.Model(&model.OrgPosition{}).
		Where("id = ?", position.Id).
		Update("status", "disabled").Error; err != nil {
		t.Fatalf("disable position: %v", err)
	}

	assignment := orgAssignmentFixture(
		711,
		employee.Id,
		legal.Id,
		unit.Id,
		&position.Id,
		asOf.AddDate(-1, 0, 0),
		assignmentTimePointer(asOf.Add(-24*time.Hour)),
		"disabled",
	)
	assignment.SourceVersion = "source-version-must-not-leak"
	assignment.SyncStatus = "failed"
	testutil.MustCreate(t, db, &assignment)

	result, err := orgService.QueryAssignments(nil, request.OrgAssignmentQueryReq{
		Basic:      request.Basic{Page: 1, Num: 10},
		EmployeeId: &employee.Id,
		TimeScope:  request.OrgAssignmentScopeHistory,
		AsOfDate:   "2026-07-26",
	}, orgAssignmentServiceTable())
	if err != nil || result.Total != 1 {
		t.Fatalf("historical replay=%+v err=%v", result, err)
	}
	row := result.Data[0]
	if row.LegalEntity == nil || row.LegalEntity.Name != legal.Name ||
		row.OrgUnit == nil || row.OrgUnit.Name != unit.Name ||
		row.Position == nil || row.Position.Name != position.Name {
		t.Fatalf("historical reference replay missing: %+v", row)
	}
	unitOptions, err := orgService.QueryOrgUnitOptions(
		nil,
		request.OrgUnitOptionsReq{
			OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
				Page:        1,
				Num:         10,
				Keyword:     "no-active-match",
				SelectedIds: []int{unit.Id},
			},
		},
		orgEmployeePositionServiceTable("org_unit"),
	)
	if err != nil || len(unitOptions.Items) != 1 ||
		unitOptions.Items[0].Value != unit.Id ||
		!unitOptions.Items[0].Disabled {
		t.Fatalf("historical org unit option replay=%+v err=%v", unitOptions, err)
	}
	positionOptions, err := orgService.QueryPositionOptions(
		nil,
		request.OrgPositionOptionsReq{
			OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{
				Page:        1,
				Num:         10,
				Keyword:     "no-active-match",
				SelectedIds: []int{position.Id},
			},
		},
		orgEmployeePositionServiceTable("org_position"),
	)
	if err != nil || len(positionOptions.Items) != 1 ||
		positionOptions.Items[0].Value != position.Id ||
		!positionOptions.Items[0].Disabled {
		t.Fatalf("historical position option replay=%+v err=%v", positionOptions, err)
	}

	detail, err := orgService.GetAssignmentDetail(
		nil,
		assignment.Id,
		request.OrgAssignmentDetailReq{},
	)
	if err != nil {
		t.Fatalf("assignment detail: %v", err)
	}
	serialized, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal assignment detail: %v", err)
	}
	for _, forbidden := range []string{
		"source_system_code",
		"source_id",
		"source_version",
		"source_deleted",
		"sync_status",
		assignment.SourceVersion,
	} {
		if strings.Contains(string(serialized), forbidden) {
			t.Fatalf("assignment response leaked %q: %s", forbidden, serialized)
		}
	}

	_, err = orgService.GetAssignmentDetail(nil, 99999, request.OrgAssignmentDetailReq{})
	assertOrgServiceAdminError(
		t,
		err,
		apperrors.CategoryBusiness,
		apperrors.ErrorCodeOrgAssignmentNotFound,
	)
}

func orgAssignmentFixture(
	id int,
	employeeId int,
	legalEntityId int,
	orgUnitId int,
	positionId *int,
	validFrom time.Time,
	validTo *time.Time,
	status string,
) model.OrgAssignment {
	return model.OrgAssignment{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "assignment-source-" + time.Unix(int64(id), 0).UTC().Format("150405"),
		EmployeeId:       employeeId,
		LegalEntityId:    legalEntityId,
		OrgUnitId:        orgUnitId,
		PositionId:       positionId,
		AssignmentType:   "part_time",
		ValidFrom:        &validFrom,
		ValidTo:          validTo,
		Status:           status,
		SyncStatus:       "synced",
	}
}

func assignmentTimePointer(value time.Time) *time.Time {
	return &value
}

func orgAssignmentServiceTable() model.SysTable {
	field := func(code string, fieldType enum.SysTableFieldType) model.SysTableField {
		return model.SysTableField{
			FieldCode:        code,
			FieldType:        fieldType,
			IsListShow:       true,
			IsAdvancedSearch: true,
			IsSort:           true,
		}
	}
	return model.SysTable{
		Basic:     model.Basic{Id: 900, State: true},
		TableCode: "org_assignment",
		TableFields: []model.SysTableField{
			field("id", enum.BigIntFieldType),
			field("employee_id", enum.BigIntFieldType),
			field("legal_entity_id", enum.BigIntFieldType),
			field("org_unit_id", enum.BigIntFieldType),
			field("position_id", enum.BigIntFieldType),
			field("assignment_type", enum.VarcharFieldType),
			field("is_primary", enum.BooleanFieldType),
			field("is_manager", enum.BooleanFieldType),
			field("status", enum.VarcharFieldType),
			field("valid_from", enum.DatetimeFieldType),
			field("valid_to", enum.DatetimeFieldType),
		},
	}
}
