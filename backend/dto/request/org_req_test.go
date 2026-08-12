package request_test

import (
	"backend/dto/request"
	"backend/internal/utils"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestOrganizationSelectorOptionsRequestsShareJSONProtocol(t *testing.T) {
	payload := []byte(`{
		"page": 2,
		"num": 25,
		"keyword": "中心",
		"selected_ids": [11, 12],
		"only_effective": false,
		"include_history": true
	}`)
	assertProtocol := func(
		name string,
		target interface{},
		common func() request.OrgSelectorOptionsReq,
	) {
		t.Helper()
		if err := json.Unmarshal(payload, target); err != nil {
			t.Fatalf("%s unmarshal shared selector protocol: %v", name, err)
		}
		value := common()
		if value.Page != 2 ||
			value.Num != 25 ||
			value.Keyword != "中心" ||
			len(value.SelectedIds) != 2 ||
			value.OnlyEffective == nil ||
			*value.OnlyEffective ||
			!value.IncludeHistory {
			t.Fatalf("%s shared selector request=%+v", name, value)
		}
	}

	legal := request.OrgLegalEntityOptionsReq{}
	assertProtocol("legal_entity", &legal, func() request.OrgSelectorOptionsReq {
		return legal.OrgSelectorOptionsReq
	})
	unit := request.OrgUnitOptionsReq{}
	assertProtocol("org_unit", &unit, func() request.OrgSelectorOptionsReq {
		return unit.OrgSelectorOptionsReq
	})
	employee := request.OrgEmployeeOptionsReq{}
	assertProtocol("employee", &employee, func() request.OrgSelectorOptionsReq {
		return employee.OrgSelectorOptionsReq
	})
	position := request.OrgPositionOptionsReq{}
	assertProtocol("position", &position, func() request.OrgSelectorOptionsReq {
		return position.OrgSelectorOptionsReq
	})
}

func TestOrganizationQueryDTOValidation(t *testing.T) {
	validate := validator.New()
	if _, err := utils.InitializeValidator(validate); err != nil {
		t.Fatalf("initialize validator: %v", err)
	}

	if err := validate.Struct(request.OrgEmployeeQueryReq{EmploymentStatus: "active"}); err != nil {
		t.Fatalf("expected valid employment status: %v", err)
	}
	if err := validate.Struct(request.OrgEmployeeQueryReq{EmploymentStatus: "unknown"}); err == nil {
		t.Fatal("expected invalid employment status to fail")
	}
	for _, structureType := range []string{"management", "legal"} {
		if err := validate.Struct(request.OrgStructureQueryReq{StructureType: structureType}); err != nil {
			t.Fatalf("expected valid structure type %s: %v", structureType, err)
		}
	}
	if err := validate.Struct(request.OrgStructureQueryReq{StructureType: "matrix"}); err == nil {
		t.Fatal("expected unknown structure type to fail")
	}
	if err := validate.Struct(request.OrgEmployeeQueryReq{BoundStatus: "bound"}); err != nil {
		t.Fatalf("expected valid bound_status: %v", err)
	}
	if err := validate.Struct(request.OrgEmployeeQueryReq{BoundStatus: "linked"}); err == nil {
		t.Fatal("expected invalid bound_status to fail")
	}
	if err := validate.Struct(request.OrgEmployeeOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{SelectedIds: []int{1, 2}},
	}); err != nil {
		t.Fatalf("expected positive employee selected_ids: %v", err)
	}
	if err := validate.Struct(request.OrgPositionOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{SelectedIds: []int{0}},
	}); err == nil {
		t.Fatal("expected non-positive position selected_ids to fail")
	}
	zero := 0
	if err := validate.Struct(request.OrgAssignmentQueryReq{EmployeeId: &zero}); err == nil {
		t.Fatal("expected non-positive employee_id to fail")
	}
	employeeID := 1
	if err := validate.Struct(request.OrgAssignmentQueryReq{
		EmployeeId: &employeeID,
		TimeScope:  request.OrgAssignmentScopeTimeline,
		AsOfDate:   "2026-07-26",
	}); err != nil {
		t.Fatalf("expected valid assignment query to pass: %v", err)
	}
	if err := validate.Struct(request.OrgAssignmentQueryReq{
		EmployeeId: &employeeID,
		TimeScope:  "primary",
	}); err == nil {
		t.Fatal("expected invalid assignment time_scope to fail")
	}
	if err := validate.Struct(request.OrgAssignmentQueryReq{}); err == nil {
		t.Fatal("expected assignment query without employee_id to fail")
	}
	if err := validate.Struct(request.OrgSyncRecordQueryReq{Action: "rewrite"}); err == nil {
		t.Fatal("expected unsupported sync action to fail")
	}
	if err := validate.Struct(request.OrgLegalEntityTreeReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{AsOfDate: "2026-07-26"},
	}); err != nil {
		t.Fatalf("expected date-only as_of_date to pass: %v", err)
	}
	if err := validate.Struct(request.OrgLegalEntityTreeReq{
		OrgLegalEntityReadScopeReq: request.OrgLegalEntityReadScopeReq{AsOfDate: "2026/07/26"},
	}); err == nil {
		t.Fatal("expected invalid as_of_date to fail")
	}
	if err := validate.Struct(request.OrgLegalEntityOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{SelectedIds: []int{1, 2}},
	}); err != nil {
		t.Fatalf("expected positive selected_ids to pass: %v", err)
	}
	if err := validate.Struct(request.OrgLegalEntityOptionsReq{
		OrgSelectorOptionsReq: request.OrgSelectorOptionsReq{SelectedIds: []int{0}},
	}); err == nil {
		t.Fatal("expected non-positive selected_ids to fail")
	}
	if err := validate.Struct(request.OrgStructureOrgTreeReq{StructureId: 1}); err != nil {
		t.Fatalf("expected positive structure_id to pass: %v", err)
	}
	if err := validate.Struct(request.OrgStructureOrgTreeReq{}); err == nil {
		t.Fatal("expected missing structure_id to fail")
	}
	if err := validate.Struct(request.OrgUnitOptionsReq{
		StructureId: &zero,
	}); err == nil {
		t.Fatal("expected non-positive option structure_id to fail")
	}
}

func TestOrganizationQueryDTOsExcludeRestrictedFields(t *testing.T) {
	assertStructFieldsAbsent(t, reflect.TypeOf(request.OrgEmployeeQueryReq{}),
		"SourceId", "SourceVersion", "Mobile", "Email", "SyncStatus",
	)
	assertStructFieldsAbsent(t, reflect.TypeOf(request.OrgPositionQueryReq{}),
		"SourceId", "SourceVersion", "SyncStatus",
	)
	assertStructFieldsAbsent(t, reflect.TypeOf(request.OrgAssignmentQueryReq{}),
		"SourceSystemCode", "SourceId", "SourceVersion", "SyncStatus",
	)
	assertStructFieldsAbsent(t, reflect.TypeOf(request.OrgStructureNodeQueryReq{}),
		"SourceId", "SourceParentId", "Path", "Level", "SyncStatus",
	)
	assertStructFieldsAbsent(t, reflect.TypeOf(request.OrgSyncRecordQueryReq{}),
		"SourceId", "DependencyKey", "ErrorMessage",
	)
}

func assertStructFieldsAbsent(t *testing.T, typ reflect.Type, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if _, exists := typ.FieldByName(field); exists {
			t.Fatalf("%s unexpectedly exposes field %s", typ.Name(), field)
		}
	}
}
