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

const orgAssignmentQueryRole = "organization_assignment_query"

func TestOrgControllerAssignmentQueryDetailAndSummary(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	legal, unit, position, employee := orgControllerEmployeePositionFixtures()
	assignment := orgControllerAssignmentFixture(
		900,
		employee.Id,
		legal.Id,
		unit.Id,
		position.Id,
	)
	assignment.SourceVersion = "source-version-must-not-leak"
	assignment.SyncStatus = "failed"
	testutil.MustCreate(t, db, &legal)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &position)
	testutil.MustCreate(t, db, &employee)
	testutil.MustCreate(t, db, &assignment)

	testutil.AssertPermissions(
		t,
		enforcer,
		testutil.PermissionCase{
			Name:    "assignment query",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/assignment/query",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "assignment detail",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/assignment/:id",
			Method:  http.MethodGet,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "assignment summary",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/employee/:id/assignments/summary",
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
			name:   "query",
			method: http.MethodPost,
			target: "/admin/org/assignment/query",
			body: `{"page":1,"num":10,"employee_id":` +
				strconv.Itoa(employee.Id) + `,"time_scope":"current"}`,
		},
		{
			name:   "detail",
			method: http.MethodGet,
			target: "/admin/org/assignment/" + strconv.Itoa(assignment.Id),
		},
		{
			name:   "summary",
			method: http.MethodGet,
			target: "/admin/org/employee/" + strconv.Itoa(employee.Id) +
				"/assignments/summary",
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
				`"source_deleted"`,
				`"sync_status"`,
				assignment.SourceVersion,
			} {
				if strings.Contains(recorder.Body.String(), forbidden) {
					t.Fatalf("response leaked %q: %s", forbidden, recorder.Body.String())
				}
			}
		})
	}
}

func TestOrgControllerAssignmentQueryAndDetailPermissionBoundary(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgAssignmentQueryRole)
	legal, unit, position, employee := orgControllerEmployeePositionFixtures()
	assignment := orgControllerAssignmentFixture(
		901,
		employee.Id,
		legal.Id,
		unit.Id,
		position.Id,
	)
	testutil.MustCreate(t, db, &legal)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &position)
	testutil.MustCreate(t, db, &employee)
	testutil.MustCreate(t, db, &assignment)
	for _, policy := range [][]string{
		{orgAssignmentQueryRole, "/admin/org/assignment/query", http.MethodPost},
		{
			orgAssignmentQueryRole,
			"/admin/org/employee/:id/assignments/summary",
			http.MethodGet,
		},
	} {
		if _, err := enforcer.AddPolicy(policy[0], policy[1], policy[2]); err != nil {
			t.Fatalf("add assignment query policy %v: %v", policy, err)
		}
	}

	queryRecorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodPost,
		Target: "/admin/org/assignment/query",
		Body: bytes.NewBufferString(
			`{"page":1,"num":10,"employee_id":` + strconv.Itoa(employee.Id) + `}`,
		),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	})
	if queryRecorder.Code != http.StatusOK {
		t.Fatalf(
			"query status=%d body=%s",
			queryRecorder.Code,
			queryRecorder.Body.String(),
		)
	}
	summaryRecorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodGet,
		Target: "/admin/org/employee/" + strconv.Itoa(employee.Id) +
			"/assignments/summary",
	})
	if summaryRecorder.Code != http.StatusOK {
		t.Fatalf(
			"summary status=%d body=%s",
			summaryRecorder.Code,
			summaryRecorder.Body.String(),
		)
	}
	detailRecorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodGet,
		Target: "/admin/org/assignment/" + strconv.Itoa(assignment.Id),
	})
	if detailRecorder.Code != http.StatusForbidden {
		t.Fatalf(
			"detail status=%d body=%s",
			detailRecorder.Code,
			detailRecorder.Body.String(),
		)
	}
}

func TestOrgControllerAssignmentStableErrors(t *testing.T) {
	router, _, _ := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	for _, item := range []struct {
		target    string
		status    int
		errorCode int
	}{
		{
			target:    "/admin/org/assignment/not-an-id",
			status:    http.StatusBadRequest,
			errorCode: apperrors.ErrorCodeParamInvalid,
		},
		{
			target:    "/admin/org/assignment/999",
			status:    http.StatusNotFound,
			errorCode: apperrors.ErrorCodeOrgAssignmentNotFound,
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

func orgControllerAssignmentFixture(
	id int,
	employeeId int,
	legalEntityId int,
	orgUnitId int,
	positionId int,
) model.OrgAssignment {
	return model.OrgAssignment{
		Basic:            model.Basic{Id: id, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-assignment-" + strconv.Itoa(id),
		EmployeeId:       employeeId,
		LegalEntityId:    legalEntityId,
		OrgUnitId:        orgUnitId,
		PositionId:       &positionId,
		AssignmentType:   "part_time",
		Status:           "enabled",
		SyncStatus:       "synced",
	}
}
