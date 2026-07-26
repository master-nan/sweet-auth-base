package controller

import (
	"backend/dto/response"
	testutil "backend/internal/test"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestOrgControllerFourSelectorOptionsUseSharedEnvelopeAndQueryPermission(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	legal, unit, position, employee := orgControllerEmployeePositionFixtures()
	testutil.MustCreate(t, db, &legal)
	testutil.MustCreate(t, db, &unit)
	testutil.MustCreate(t, db, &position)
	testutil.MustCreate(t, db, &employee)

	cases := []struct {
		name          string
		target        string
		keyword       string
		expectedValue int
		expectedLabel string
	}{
		{
			name:          "legal entity",
			target:        "/admin/org/legal-entity/options",
			keyword:       legal.Code,
			expectedValue: legal.Id,
			expectedLabel: legal.Code + " - " + legal.Name,
		},
		{
			name:          "organization unit",
			target:        "/admin/org/unit/options",
			keyword:       unit.Code,
			expectedValue: unit.Id,
			expectedLabel: unit.Code + " - " + unit.Name,
		},
		{
			name:          "employee",
			target:        "/admin/org/employee/options",
			keyword:       employee.EmployeeNo,
			expectedValue: employee.Id,
			expectedLabel: employee.EmployeeNo + " - " + employee.Name,
		},
		{
			name:          "position",
			target:        "/admin/org/position/options",
			keyword:       position.Code,
			expectedValue: position.Id,
			expectedLabel: position.Code + " - " + position.Name,
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			testutil.AssertPermissions(
				t,
				enforcer,
				testutil.PermissionCase{
					Name:    item.name + " query permission",
					Subject: orgLegalEntityReaderRole,
					Path:    item.target,
					Method:  http.MethodPost,
					Allowed: true,
				},
			)
			requestBody, err := json.Marshal(map[string]interface{}{
				"page":           1,
				"num":            10,
				"keyword":        item.keyword,
				"only_effective": true,
			})
			if err != nil {
				t.Fatalf("marshal selector request: %v", err)
			}
			recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
				Method: http.MethodPost,
				Target: item.target,
				Body:   bytes.NewReader(requestBody),
				Header: http.Header{"Content-Type": []string{"application/json"}},
			})
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d, want 200: %s", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				Success bool                            `json:"success"`
				Data    []response.OrgSelectorOptionRes `json:"data"`
				Total   int                             `json:"total"`
			}
			if err = json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode selector response: %v", err)
			}
			if !payload.Success || payload.Total != 1 || len(payload.Data) != 1 {
				t.Fatalf("unexpected selector envelope: %+v", payload)
			}
			option := payload.Data[0]
			if option.Value != item.expectedValue ||
				option.Label != item.expectedLabel ||
				option.Disabled {
				t.Fatalf("unexpected selector option: %+v", option)
			}
		})
	}
}

func TestOrgControllerFourSelectorOptionsRejectRoleWithoutQueryPermission(t *testing.T) {
	router, _, enforcer := newOrgControllerTestRouter(t, orgLegalEntityDeniedRole)
	for _, target := range []string{
		"/admin/org/legal-entity/options",
		"/admin/org/unit/options",
		"/admin/org/employee/options",
		"/admin/org/position/options",
	} {
		testutil.AssertPermissions(
			t,
			enforcer,
			testutil.PermissionCase{
				Name:    target + " denied",
				Subject: orgLegalEntityDeniedRole,
				Path:    target,
				Method:  http.MethodPost,
				Allowed: false,
			},
		)
		recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
			Method: http.MethodPost,
			Target: target,
			Body:   bytes.NewBufferString(`{"page":1,"num":10}`),
			Header: http.Header{"Content-Type": []string{"application/json"}},
		})
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("target=%s status=%d, want 403: %s", target, recorder.Code, recorder.Body.String())
		}
		var payload response.AdminError
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode permission response: %v", err)
		}
		if payload.Success || payload.ErrorCode != 30006 {
			t.Fatalf("target=%s permission response=%+v", target, payload)
		}
	}
}
