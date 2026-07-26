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

func TestOrgControllerManagementRoutesUseQueryAndDetailPermissions(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	structure, unit, node := orgControllerManagementFixtures()
	testutil.MustCreate(t, db, &structure)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &node)

	for _, permission := range []testutil.PermissionCase{
		{
			Name:    "structure query",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/structure/query",
			Method:  http.MethodPost,
			Allowed: true,
		},
		{
			Name:    "structure detail",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/structure/:id",
			Method:  http.MethodGet,
			Allowed: true,
		},
		{
			Name:    "unit query",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/unit/query",
			Method:  http.MethodPost,
			Allowed: true,
		},
		{
			Name:    "unit tree",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/unit/tree",
			Method:  http.MethodPost,
			Allowed: true,
		},
		{
			Name:    "unit detail",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/unit/:id",
			Method:  http.MethodGet,
			Allowed: true,
		},
	} {
		testutil.AssertPermissions(t, enforcer, permission)
	}

	for _, item := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{
			name:   "structure query",
			method: http.MethodPost,
			target: "/admin/org/structure/query",
			body:   `{"page":1,"num":10}`,
		},
		{
			name:   "structure options",
			method: http.MethodPost,
			target: "/admin/org/structure/options",
			body:   `{"page":1,"num":10,"keyword":"行政"}`,
		},
		{
			name:   "structure detail",
			method: http.MethodGet,
			target: "/admin/org/structure/" + strconv.Itoa(structure.Id),
		},
		{
			name:   "unit query",
			method: http.MethodPost,
			target: "/admin/org/unit/query",
			body:   `{"page":1,"num":10}`,
		},
		{
			name:   "unit options",
			method: http.MethodPost,
			target: "/admin/org/unit/options",
			body:   `{"page":1,"num":10,"structure_id":10}`,
		},
		{
			name:   "unit tree",
			method: http.MethodPost,
			target: "/admin/org/unit/tree",
			body:   `{"structure_id":10}`,
		},
		{
			name:   "unit detail",
			method: http.MethodGet,
			target: "/admin/org/unit/" + strconv.Itoa(unit.Id),
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
				t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
			}
			var payload response.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !payload.Success || payload.Code != http.StatusOK {
				t.Fatalf("unexpected unified response: %#v", payload)
			}
			for _, forbidden := range []string{
				`"source_id"`,
				`"source_version"`,
				`"sync_status"`,
				`"last_error"`,
				`"path"`,
			} {
				if strings.Contains(recorder.Body.String(), forbidden) {
					t.Fatalf("response leaked %s: %s", forbidden, recorder.Body.String())
				}
			}
		})
	}
}

func TestOrgControllerManagementQueryAndDetailPermissionBoundary(t *testing.T) {
	router, db, _ := newOrgControllerTestRouter(t, orgManagementQueryRole)
	structure, unit, node := orgControllerManagementFixtures()
	testutil.MustCreate(t, db, &structure)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &node)

	query := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodPost,
		Target: "/admin/org/structure/query",
		Body:   bytes.NewBufferString(`{"page":1,"num":10}`),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	})
	if query.Code != http.StatusOK {
		t.Fatalf("query status = %d, want 200: %s", query.Code, query.Body.String())
	}

	detail := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodGet,
		Target: "/admin/org/structure/" + strconv.Itoa(structure.Id),
	})
	if detail.Code != http.StatusForbidden {
		t.Fatalf("detail status = %d, want 403: %s", detail.Code, detail.Body.String())
	}
}

func TestOrgControllerManagementErrorsUseStableFormat(t *testing.T) {
	router, _, _ := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	for _, item := range []struct {
		name      string
		method    string
		target    string
		body      string
		status    int
		errorCode int
	}{
		{
			name:      "missing structure",
			method:    http.MethodGet,
			target:    "/admin/org/structure/999",
			status:    http.StatusNotFound,
			errorCode: apperrors.ErrorCodeOrgStructureNotFound,
		},
		{
			name:      "invalid unit id",
			method:    http.MethodGet,
			target:    "/admin/org/unit/not-an-id",
			status:    http.StatusBadRequest,
			errorCode: apperrors.ErrorCodeParamInvalid,
		},
		{
			name:      "tree requires structure id",
			method:    http.MethodPost,
			target:    "/admin/org/unit/tree",
			body:      `{}`,
			status:    http.StatusBadRequest,
			errorCode: apperrors.ErrorCodeParamInvalid,
		},
	} {
		t.Run(item.name, func(t *testing.T) {
			recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
				Method: item.method,
				Target: item.target,
				Body:   bytes.NewBufferString(item.body),
				Header: http.Header{"Content-Type": []string{"application/json"}},
			})
			if recorder.Code != item.status {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, item.status, recorder.Body.String())
			}
			var payload response.AdminError
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode stable error: %v", err)
			}
			if payload.Success || payload.ErrorCode != item.errorCode {
				t.Fatalf("unexpected error response: %#v", payload)
			}
		})
	}
}

func orgControllerManagementFixtures() (
	model.OrgStructure,
	model.OrgUnit,
	model.OrgStructureNode,
) {
	structure := model.OrgStructure{
		Basic:            model.Basic{Id: 10, State: true},
		Code:             "MGMT",
		Name:             "行政架构",
		StructureType:    "management",
		SourceSystemCode: "authority",
		SourceId:         "structure-10",
		Status:           "enabled",
		SyncStatus:       "synced",
	}
	unit := model.OrgUnit{
		Basic:            model.Basic{Id: 20, State: true},
		SourceSystemCode: "authority",
		SourceId:         "unit-20",
		SourceCode:       "SRC-OU-20",
		Code:             "OU-20",
		Name:             "运营中心",
		UnitType:         "center",
		Status:           "enabled",
		SyncStatus:       "synced",
	}
	node := model.OrgStructureNode{
		Basic:            model.Basic{Id: 30, State: true},
		StructureId:      structure.Id,
		OrgUnitId:        unit.Id,
		SourceSystemCode: "authority",
		SourceId:         "node-30",
		Path:             "/30/",
		Level:            1,
		Status:           "enabled",
		SyncStatus:       "synced",
	}
	return structure, unit, node
}
