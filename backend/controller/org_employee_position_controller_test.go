package controller

import (
	"backend/dto/response"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/model"
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

const orgEmployeePositionQueryRole = "organization_employee_position_query"

func TestOrgControllerEmployeeAndPositionRoutesUseSeededPermissions(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	legal, unit, position, employee := orgControllerEmployeePositionFixtures()
	user := model.SysUser{
		Basic:        model.Basic{Id: 50, State: true},
		UserName:     "bound_account",
		Password:     "password-must-not-leak",
		AccessTokens: "token-must-not-leak",
	}
	employee.UserId = &user.Id
	employee.Mobile = "13800138000"
	employee.Email = "employee@example.com"
	employee.SourceVersion = "source-version-secret"
	testutil.MustCreate(t, db, &legal)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &position)
	testutil.MustCreate(t, db, &user)
	testutil.MustCreate(t, db, &employee)

	testutil.AssertPermissions(
		t,
		enforcer,
		testutil.PermissionCase{
			Name:    "employee query",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/employee/query",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "employee options",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/employee/options",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "employee detail",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/employee/:id",
			Method:  http.MethodGet,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "position query",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/position/query",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "position options",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/position/options",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "position detail",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/position/:id",
			Method:  http.MethodGet,
			Allowed: true,
		},
	)

	for _, item := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{
			name:   "employee query",
			method: http.MethodPost,
			target: "/admin/org/employee/query",
			body:   `{"page":1,"num":10,"quick_query":{"keyword":"EMP-001"}}`,
		},
		{
			name:   "employee options",
			method: http.MethodPost,
			target: "/admin/org/employee/options",
			body:   `{"page":1,"num":10,"keyword":"EMP-001"}`,
		},
		{
			name:   "employee detail",
			method: http.MethodGet,
			target: "/admin/org/employee/" + strconv.Itoa(employee.Id),
		},
		{
			name:   "position query",
			method: http.MethodPost,
			target: "/admin/org/position/query",
			body:   `{"page":1,"num":10,"quick_query":{"keyword":"POS-001"}}`,
		},
		{
			name:   "position options",
			method: http.MethodPost,
			target: "/admin/org/position/options",
			body:   `{"page":1,"num":10,"keyword":"POS-001"}`,
		},
		{
			name:   "position detail",
			method: http.MethodGet,
			target: "/admin/org/position/" + strconv.Itoa(position.Id),
		},
	} {
		t.Run(item.name, func(t *testing.T) {
			recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
				Method: item.method,
				Target: item.target,
				Body:   bytes.NewBufferString(item.body),
				Header: http.Header{"Content-Type": []string{"application/json"}},
			})
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var payload response.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !payload.Success || payload.Code != http.StatusOK {
				t.Fatalf("unexpected response: %#v", payload)
			}
			for _, forbidden := range []string{
				`"source_id"`,
				`"source_version"`,
				`"sync_status"`,
				user.Password,
				user.AccessTokens,
				employee.Mobile,
				employee.Email,
			} {
				if forbidden != "" && strings.Contains(recorder.Body.String(), forbidden) {
					t.Fatalf("response leaked %q: %s", forbidden, recorder.Body.String())
				}
			}
		})
	}
}

func TestOrgControllerEmployeeAndPositionQueryDetailPermissionBoundary(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgEmployeePositionQueryRole)
	legal, unit, position, employee := orgControllerEmployeePositionFixtures()
	testutil.MustCreate(t, db, &legal)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &position)
	testutil.MustCreate(t, db, &employee)
	for _, policy := range [][]string{
		{orgEmployeePositionQueryRole, "/admin/org/employee/query", http.MethodPost},
		{orgEmployeePositionQueryRole, "/admin/org/employee/options", http.MethodPost},
		{orgEmployeePositionQueryRole, "/admin/org/position/query", http.MethodPost},
		{orgEmployeePositionQueryRole, "/admin/org/position/options", http.MethodPost},
	} {
		if _, err := enforcer.AddPolicy(policy[0], policy[1], policy[2]); err != nil {
			t.Fatalf("add query-only policy %v: %v", policy, err)
		}
	}

	for _, target := range []string{
		"/admin/org/employee/query",
		"/admin/org/employee/options",
		"/admin/org/position/query",
		"/admin/org/position/options",
	} {
		recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
			Method: http.MethodPost,
			Target: target,
			Body:   bytes.NewBufferString(`{"page":1,"num":10}`),
			Header: http.Header{"Content-Type": []string{"application/json"}},
		})
		if recorder.Code != http.StatusOK {
			t.Fatalf("query permission target=%s status=%d body=%s", target, recorder.Code, recorder.Body.String())
		}
	}
	for _, target := range []string{
		"/admin/org/employee/" + strconv.Itoa(employee.Id),
		"/admin/org/position/" + strconv.Itoa(position.Id),
	} {
		recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
			Method: http.MethodGet,
			Target: target,
		})
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("detail permission target=%s status=%d body=%s", target, recorder.Code, recorder.Body.String())
		}
	}
}

func TestOrgControllerEmployeeAndPositionQueryRejectsMissingPermission(t *testing.T) {
	router, _, _ := newOrgControllerTestRouter(t, orgLegalEntityDeniedRole)
	for _, target := range []string{
		"/admin/org/employee/query",
		"/admin/org/employee/options",
		"/admin/org/position/query",
		"/admin/org/position/options",
	} {
		recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
			Method: http.MethodPost,
			Target: target,
			Body:   bytes.NewBufferString(`{"page":1,"num":10}`),
			Header: http.Header{"Content-Type": []string{"application/json"}},
		})
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("target=%s status=%d body=%s", target, recorder.Code, recorder.Body.String())
		}
		var payload response.AdminError
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode permission error: %v", err)
		}
		if payload.Success || payload.ErrorCode != 30006 {
			t.Fatalf("target=%s permission error=%+v", target, payload)
		}
	}
}

func TestOrgControllerEmployeeAndPositionStableErrors(t *testing.T) {
	router, _, _ := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	for _, item := range []struct {
		target    string
		status    int
		errorCode int
	}{
		{
			target:    "/admin/org/employee/not-an-id",
			status:    http.StatusBadRequest,
			errorCode: apperrors.ErrorCodeParamInvalid,
		},
		{
			target:    "/admin/org/employee/999",
			status:    http.StatusNotFound,
			errorCode: apperrors.ErrorCodeOrgEmployeeNotFound,
		},
		{
			target:    "/admin/org/position/not-an-id",
			status:    http.StatusBadRequest,
			errorCode: apperrors.ErrorCodeParamInvalid,
		},
		{
			target:    "/admin/org/position/999",
			status:    http.StatusNotFound,
			errorCode: apperrors.ErrorCodeOrgPositionNotFound,
		},
	} {
		recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
			Method: http.MethodGet,
			Target: item.target,
		})
		if recorder.Code != item.status {
			t.Fatalf("target=%s status=%d body=%s", item.target, recorder.Code, recorder.Body.String())
		}
		var payload response.AdminError
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode stable error: %v", err)
		}
		if payload.Success || payload.ErrorCode != item.errorCode {
			t.Fatalf("target=%s error=%+v", item.target, payload)
		}
	}
}

func orgControllerEmployeePositionFixtures() (
	model.OrgLegalEntity,
	model.OrgUnit,
	model.OrgPosition,
	model.OrgEmployee,
) {
	legal := orgControllerLegalEntityFixture(10, "LE-001", "法人一")
	unit := model.OrgUnit{
		Basic:                model.Basic{Id: 20, State: true},
		SourceSystemCode:     "authority",
		SourceId:             "source-unit-20",
		Code:                 "OU-001",
		Name:                 "组织一",
		UnitType:             "department",
		PrimaryLegalEntityId: &legal.Id,
		Status:               "enabled",
		SyncStatus:           "synced",
	}
	position := model.OrgPosition{
		Basic:            model.Basic{Id: 30, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-position-30",
		Code:             "POS-001",
		Name:             "岗位一",
		OrgUnitId:        unit.Id,
		PositionType:     "professional",
		Status:           "enabled",
		SyncStatus:       "synced",
	}
	employee := model.OrgEmployee{
		Basic:                model.Basic{Id: 40, State: true},
		SourceSystemCode:     "authority",
		SourceId:             "source-employee-40",
		EmployeeNo:           "EMP-001",
		Name:                 "人员一",
		EmploymentStatus:     "active",
		PrimaryLegalEntityId: &legal.Id,
		SyncStatus:           "synced",
	}
	return legal, unit, position, employee
}
