package request_test

import (
	"backend/dto/request"
	"backend/internal/utils"
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
)

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
	if err := validate.Struct(request.OrgEmployeeQueryReq{BoundStatus: "bound"}); err != nil {
		t.Fatalf("expected valid bound_status: %v", err)
	}
	if err := validate.Struct(request.OrgEmployeeQueryReq{BoundStatus: "linked"}); err == nil {
		t.Fatal("expected invalid bound_status to fail")
	}
	if err := validate.Struct(request.OrgEmployeeOptionsReq{SelectedIds: []int{1, 2}}); err != nil {
		t.Fatalf("expected positive employee selected_ids: %v", err)
	}
	if err := validate.Struct(request.OrgPositionOptionsReq{SelectedIds: []int{0}}); err == nil {
		t.Fatal("expected non-positive position selected_ids to fail")
	}
	zero := 0
	if err := validate.Struct(request.OrgAssignmentQueryReq{EmployeeId: &zero}); err == nil {
		t.Fatal("expected non-positive employee_id to fail")
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
	if err := validate.Struct(request.OrgLegalEntityOptionsReq{SelectedIds: []int{1, 2}}); err != nil {
		t.Fatalf("expected positive selected_ids to pass: %v", err)
	}
	if err := validate.Struct(request.OrgLegalEntityOptionsReq{SelectedIds: []int{0}}); err == nil {
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
