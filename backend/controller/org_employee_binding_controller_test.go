package controller

import (
	"backend/dto/response"
	testutil "backend/internal/test"
	"backend/model"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestOrgControllerEmployeeUserBindingUsesPermissionsAndSafeResponse(t *testing.T) {
	router, db, enforcer := newOrgControllerTestRouter(t, orgLegalEntityReaderRole)
	user := model.SysUser{
		Basic:        model.Basic{Id: 501, State: true},
		UserName:     "binding_account",
		Password:     "password-must-not-leak",
		AccessTokens: "token-must-not-leak",
	}
	employee := model.OrgEmployee{
		Basic:            model.Basic{Id: 601, State: true},
		SourceSystemCode: "authority",
		SourceId:         "source-binding-employee",
		EmployeeNo:       "EMP-BINDING",
		Name:             "绑定测试人员",
		EmploymentStatus: "active",
		SyncStatus:       "synced",
	}
	testutil.MustCreate(t, db, &user)
	testutil.MustCreate(t, db, &employee)

	testutil.AssertPermissions(
		t,
		enforcer,
		testutil.PermissionCase{
			Name:    "bind allowed",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/employee/:id/bind-user",
			Method:  http.MethodPost,
			Allowed: true,
		},
		testutil.PermissionCase{
			Name:    "unbind allowed",
			Subject: orgLegalEntityReaderRole,
			Path:    "/admin/org/employee/:id/unbind-user",
			Method:  http.MethodPost,
			Allowed: true,
		},
	)

	bindRecorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodPost,
		Target: "/admin/org/employee/601/bind-user",
		Body:   bytes.NewBufferString(`{"user_id":501}`),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	})
	assertEmployeeBindingControllerSuccess(t, bindRecorder.Code, bindRecorder.Body.Bytes())
	body := bindRecorder.Body.String()
	for _, required := range []string{
		`"employee_id":601`,
		`"user_id":501`,
		`"binding_status":"bound"`,
		`"user_name":"binding_account"`,
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("bind response missing %s: %s", required, body)
		}
	}
	for _, forbidden := range []string{
		`"password"`,
		`"access_tokens"`,
		`"token"`,
		`"roles"`,
		`"permissions"`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("bind response leaked %s: %s", forbidden, body)
		}
	}

	unbindRecorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
		Method: http.MethodPost,
		Target: "/admin/org/employee/601/unbind-user",
	})
	assertEmployeeBindingControllerSuccess(t, unbindRecorder.Code, unbindRecorder.Body.Bytes())
	if !strings.Contains(unbindRecorder.Body.String(), `"binding_status":"unbound"`) {
		t.Fatalf("unexpected unbind response: %s", unbindRecorder.Body.String())
	}

	var stored model.OrgEmployee
	if err := db.First(&stored, employee.Id).Error; err != nil {
		t.Fatalf("reload employee: %v", err)
	}
	if stored.UserId != nil {
		t.Fatalf("stored user_id = %v, want nil", stored.UserId)
	}
}

func TestOrgControllerEmployeeUserBindingRejectsMissingButtonPermissions(t *testing.T) {
	router, _, enforcer := newOrgControllerTestRouter(t, orgLegalEntityDeniedRole)
	testutil.AssertPermissions(
		t,
		enforcer,
		testutil.PermissionCase{
			Name:    "bind denied",
			Subject: orgLegalEntityDeniedRole,
			Path:    "/admin/org/employee/:id/bind-user",
			Method:  http.MethodPost,
			Allowed: false,
		},
		testutil.PermissionCase{
			Name:    "unbind denied",
			Subject: orgLegalEntityDeniedRole,
			Path:    "/admin/org/employee/:id/unbind-user",
			Method:  http.MethodPost,
			Allowed: false,
		},
	)

	for _, item := range []struct {
		name   string
		target string
		body   string
	}{
		{name: "bind", target: "/admin/org/employee/601/bind-user", body: `{"user_id":501}`},
		{name: "unbind", target: "/admin/org/employee/601/unbind-user"},
	} {
		t.Run(item.name, func(t *testing.T) {
			recorder := testutil.PerformRequest(t, router, testutil.HTTPRequest{
				Method: http.MethodPost,
				Target: item.target,
				Body:   bytes.NewBufferString(item.body),
				Header: http.Header{"Content-Type": []string{"application/json"}},
			})
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403: %s", recorder.Code, recorder.Body.String())
			}
			var payload response.AdminError
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode permission response: %v", err)
			}
			if payload.Success || payload.ErrorCode != 30006 {
				t.Fatalf("unexpected permission response: %+v", payload)
			}
		})
	}
}

func assertEmployeeBindingControllerSuccess(t *testing.T, status int, body []byte) {
	t.Helper()
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", status, body)
	}
	var payload response.Response
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode success response: %v", err)
	}
	if !payload.Success || payload.Code != http.StatusOK {
		t.Fatalf("unexpected success response: %+v", payload)
	}
}
