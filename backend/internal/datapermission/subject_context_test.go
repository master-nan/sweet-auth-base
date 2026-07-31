package datapermission_test

import (
	"encoding/json"
	stderrors "errors"
	"reflect"
	"testing"

	"backend/dto/response"
	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
)

func TestSubjectContextCreation(t *testing.T) {
	employeeId := 301
	context, err := datapermission.NewSubjectContext(
		101,
		[]int{7},
		&employeeId,
		"2026-07-31",
	)
	if err != nil {
		t.Fatalf("NewSubjectContext() error = %v", err)
	}

	if context.UserId() != 101 {
		t.Fatalf("UserId() = %d, want 101", context.UserId())
	}
	if !reflect.DeepEqual(context.RoleIds(), []int{7}) {
		t.Fatalf("RoleIds() = %v, want [7]", context.RoleIds())
	}
	if context.EmployeeId() != employeeId {
		t.Fatalf("EmployeeId() = %d, want %d", context.EmployeeId(), employeeId)
	}
	if context.AsOfDate() != "2026-07-31" {
		t.Fatalf("AsOfDate() = %q, want 2026-07-31", context.AsOfDate())
	}
	if err = context.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSubjectContextRejectsUnboundEmployee(t *testing.T) {
	_, err := datapermission.NewSubjectContext(101, []int{7}, nil, "2026-07-31")
	assertSubjectContextError(
		t,
		err,
		myerrors.ErrorCodeDataPermissionEmployeeUnbound,
	)
}

func TestSubjectContextNormalizesMultipleRolesWithoutSharingSlices(t *testing.T) {
	employeeId := 301
	roleIds := []int{9, 3, 9, 5}
	context, err := datapermission.NewSubjectContext(
		101,
		roleIds,
		&employeeId,
		"2026-07-31",
	)
	if err != nil {
		t.Fatalf("NewSubjectContext() error = %v", err)
	}

	roleIds[0] = 100
	got := context.RoleIds()
	if !reflect.DeepEqual(got, []int{3, 5, 9}) {
		t.Fatalf("RoleIds() = %v, want [3 5 9]", got)
	}

	got[0] = 200
	if !reflect.DeepEqual(context.RoleIds(), []int{3, 5, 9}) {
		t.Fatalf("RoleIds() leaked mutable state: %v", context.RoleIds())
	}
}

func TestSubjectContextDateBoundary(t *testing.T) {
	employeeId := 301
	if _, err := datapermission.NewSubjectContext(
		101,
		[]int{7},
		&employeeId,
		"2024-02-29",
	); err != nil {
		t.Fatalf("leap date should be valid: %v", err)
	}

	invalidDates := []string{
		"",
		"2023-02-29",
		"2026-07-31T00:00:00+08:00",
	}
	for _, asOfDate := range invalidDates {
		t.Run(asOfDate, func(t *testing.T) {
			_, err := datapermission.NewSubjectContext(
				101,
				[]int{7},
				&employeeId,
				asOfDate,
			)
			assertSubjectContextError(
				t,
				err,
				myerrors.ErrorCodeDataPermissionSubjectContextInvalid,
			)
		})
	}
}

func TestSubjectContextSerializationUsesStrictWhitelist(t *testing.T) {
	employeeId := 301
	context, err := datapermission.NewSubjectContext(
		101,
		[]int{9, 3},
		&employeeId,
		"2026-07-31",
	)
	if err != nil {
		t.Fatalf("NewSubjectContext() error = %v", err)
	}

	payload, err := json.Marshal(context)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var object map[string]any
	if err = json.Unmarshal(payload, &object); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	expectedKeys := []string{"as_of_date", "employee_id", "role_ids", "user_id"}
	actualKeys := make([]string, 0, len(object))
	for key := range object {
		actualKeys = append(actualKeys, key)
	}
	if !sameStringSet(actualKeys, expectedKeys) {
		t.Fatalf("serialized keys = %v, want %v", actualKeys, expectedKeys)
	}
	for _, forbidden := range []string{
		"token",
		"password",
		"legal_entity_ids",
		"org_unit_ids",
		"data_scope_result",
		"sql",
	} {
		if _, exists := object[forbidden]; exists {
			t.Fatalf("serialized context leaked forbidden field %q", forbidden)
		}
	}
}

func TestSubjectContextValidationErrorsAreStable(t *testing.T) {
	employeeId := 301
	tests := []struct {
		name      string
		userId    int
		roleIds   []int
		employee  *int
		asOfDate  string
		errorCode int
	}{
		{
			name:      "user not found",
			userId:    0,
			roleIds:   []int{7},
			employee:  &employeeId,
			asOfDate:  "2026-07-31",
			errorCode: myerrors.ErrorCodeDataPermissionSubjectUserNotFound,
		},
		{
			name:      "role context missing",
			userId:    101,
			roleIds:   nil,
			employee:  &employeeId,
			asOfDate:  "2026-07-31",
			errorCode: myerrors.ErrorCodeDataPermissionRoleContextMissing,
		},
		{
			name:      "role id invalid",
			userId:    101,
			roleIds:   []int{0, 7},
			employee:  &employeeId,
			asOfDate:  "2026-07-31",
			errorCode: myerrors.ErrorCodeDataPermissionSubjectContextInvalid,
		},
		{
			name:      "employee id invalid",
			userId:    101,
			roleIds:   []int{7},
			employee:  intPointer(0),
			asOfDate:  "2026-07-31",
			errorCode: myerrors.ErrorCodeDataPermissionSubjectContextInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := datapermission.NewSubjectContext(
				tt.userId,
				tt.roleIds,
				tt.employee,
				tt.asOfDate,
			)
			assertSubjectContextError(t, err, tt.errorCode)
		})
	}
}

func TestZeroSubjectContextCannotBeSerialized(t *testing.T) {
	_, err := json.Marshal(datapermission.SubjectContext{})
	assertSubjectContextError(
		t,
		err,
		myerrors.ErrorCodeDataPermissionSubjectUserNotFound,
	)
}

func assertSubjectContextError(t *testing.T, err error, errorCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d", errorCode)
	}
	var adminError *response.AdminError
	if !stderrors.As(err, &adminError) {
		t.Fatalf("error = %T, want *response.AdminError", err)
	}
	if adminError.ErrorCode != errorCode {
		t.Fatalf("error code = %d, want %d", adminError.ErrorCode, errorCode)
	}
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, item := range left {
		counts[item]++
	}
	for _, item := range right {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func intPointer(value int) *int {
	return &value
}
