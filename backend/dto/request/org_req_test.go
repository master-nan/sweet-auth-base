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
	zero := 0
	if err := validate.Struct(request.OrgAssignmentQueryReq{EmployeeId: &zero}); err == nil {
		t.Fatal("expected non-positive employee_id to fail")
	}
	if err := validate.Struct(request.OrgSyncRecordQueryReq{Action: "rewrite"}); err == nil {
		t.Fatal("expected unsupported sync action to fail")
	}
}

func TestOrganizationQueryDTOsExcludeRestrictedFields(t *testing.T) {
	assertStructFieldsAbsent(t, reflect.TypeOf(request.OrgEmployeeQueryReq{}),
		"SourceId", "SourceVersion", "Mobile", "Email", "SyncStatus",
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
