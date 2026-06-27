package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClassifyAccessAuditLowCodeOperations(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     map[string]interface{}
		action   string
		resource string
		id       string
		menuId   int
	}{
		{
			name:     "publish table",
			method:   "POST",
			path:     "/sweet_admin/admin/table/publish/demo_table",
			action:   "table_publish",
			resource: "demo_table",
		},
		{
			name:     "create lowcode row",
			method:   "POST",
			path:     "/sweet_admin/admin/generalization/create",
			body:     map[string]interface{}{"table_code": "demo_table", "menu_id": float64(1001)},
			action:   "lowcode_create",
			resource: "demo_table",
			menuId:   1001,
		},
		{
			name:     "update lowcode row",
			method:   "PUT",
			path:     "/sweet_admin/admin/generalization/update",
			body:     map[string]interface{}{"table_code": "demo_table", "id": float64(42), "menu_id": float64(1001)},
			action:   "lowcode_update",
			resource: "demo_table",
			id:       "42",
			menuId:   1001,
		},
		{
			name:     "delete lowcode row",
			method:   "DELETE",
			path:     "/sweet_admin/admin/generalization/delete",
			body:     map[string]interface{}{"table_code": "demo_table", "id": float64(43), "menu_id": float64(1001)},
			action:   "lowcode_delete",
			resource: "demo_table",
			id:       "43",
			menuId:   1001,
		},
		{
			name:   "table query is not a table create",
			method: "POST",
			path:   "/sweet_admin/admin/table/query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := classifyAccessAudit(tt.method, tt.path, tt.body)
			if meta.Action != tt.action {
				t.Fatalf("action = %q, want %q", meta.Action, tt.action)
			}
			if meta.ResourceCode != tt.resource {
				t.Fatalf("resource_code = %q, want %q", meta.ResourceCode, tt.resource)
			}
			if meta.ResourceId != tt.id {
				t.Fatalf("resource_id = %q, want %q", meta.ResourceId, tt.id)
			}
			if meta.MenuId != tt.menuId {
				t.Fatalf("menu_id = %d, want %d", meta.MenuId, tt.menuId)
			}
		})
	}
}

func TestClassifyAccessAuditConfigureOperations(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		action       string
		resourceType string
		resourceId   string
	}{
		{
			name:         "update configure",
			method:       "PUT",
			path:         "/sweet_admin/admin/configure/1",
			action:       "configure_update",
			resourceType: "configure",
			resourceId:   "1",
		},
		{
			name:         "test email",
			method:       "POST",
			path:         "/sweet_admin/admin/configure/test-email",
			action:       "configure_test_email",
			resourceType: "configure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := classifyAccessAudit(tt.method, tt.path, nil)
			if meta.Action != tt.action {
				t.Fatalf("action = %q, want %q", meta.Action, tt.action)
			}
			if meta.ResourceType != tt.resourceType {
				t.Fatalf("resource_type = %q, want %q", meta.ResourceType, tt.resourceType)
			}
			if meta.ResourceId != tt.resourceId {
				t.Fatalf("resource_id = %q, want %q", meta.ResourceId, tt.resourceId)
			}
		})
	}
}

func TestSanitizeLogPayloadRedactsSensitiveValues(t *testing.T) {
	payload := `{"user_name":"admin","password":"admin123","data":{"access_token":"token-value","name":"demo"}}`
	masked := sanitizeLogPayload(payload)

	for _, leaked := range []string{"admin123", "token-value"} {
		if strings.Contains(masked, leaked) {
			t.Fatalf("payload leaked sensitive value %q: %s", leaked, masked)
		}
	}
	for _, retained := range []string{"admin", "demo"} {
		if !strings.Contains(masked, retained) {
			t.Fatalf("payload did not retain non-sensitive value %q: %s", retained, masked)
		}
	}
}

func TestSanitizeLogPayloadRedactsQueryToken(t *testing.T) {
	payload := `{"token":["signed-file-access-token"],"mode":["preview"],"uuid":["file-uuid"]}`
	masked := sanitizeLogPayload(payload)

	if strings.Contains(masked, "signed-file-access-token") {
		t.Fatalf("query payload leaked signed file token: %s", masked)
	}
	for _, retained := range []string{"preview", "file-uuid"} {
		if !strings.Contains(masked, retained) {
			t.Fatalf("query payload did not retain non-sensitive value %q: %s", retained, masked)
		}
	}
}

func TestSanitizeLogPayloadRedactsSignedURLTokenInResponse(t *testing.T) {
	payload := `{"success":true,"data":{"url":"/sweet_admin/files/access/preview/file-uuid?token=signed-file-access-token"}}`
	masked := sanitizeLogPayload(payload)

	if strings.Contains(masked, "signed-file-access-token") {
		t.Fatalf("response payload leaked signed url token: %s", masked)
	}
	for _, retained := range []string{"/sweet_admin/files/access/preview/file-uuid"} {
		if !strings.Contains(masked, retained) {
			t.Fatalf("response payload did not retain non-sensitive value %q: %s", retained, masked)
		}
	}
	if !strings.Contains(masked, "token=***") {
		t.Fatalf("response payload did not mask token query value: %s", masked)
	}
}

func TestSanitizeLogPayloadRedactsTemporaryPasswordResponse(t *testing.T) {
	payload := `{"success":true,"data":{"temporary_password":"Temp-Password-2026","must_change_password":true}}`
	masked := sanitizeLogPayload(payload)

	if strings.Contains(masked, "Temp-Password-2026") {
		t.Fatalf("payload leaked temporary password: %s", masked)
	}
	if !strings.Contains(masked, "must_change_password") {
		t.Fatalf("payload did not retain non-sensitive response metadata: %s", masked)
	}
}

func TestSanitizeAccessLogPayloadRedactsOneTimeCodeOnlyOnAuthPaths(t *testing.T) {
	payload := `{"mobile":"13800138000","code":"123456","table_code":"sys_user"}`
	masked := sanitizeAccessLogPayload("/sweet_admin/api/sms_code_login", payload)

	if strings.Contains(masked, "123456") {
		t.Fatalf("sms login payload leaked one-time code: %s", masked)
	}
	if !strings.Contains(masked, "13800138000") || !strings.Contains(masked, "sys_user") {
		t.Fatalf("payload did not retain non-sensitive fields: %s", masked)
	}

	ordinary := sanitizeAccessLogPayload("/sweet_admin/admin/menu/button", payload)
	if !strings.Contains(ordinary, "123456") {
		t.Fatalf("ordinary business code should not be redacted on non-auth paths: %s", ordinary)
	}
}

func TestSanitizeAccessLogPayloadRedactsCaptchaResponse(t *testing.T) {
	payload := `{"success":true,"data":{"captcha_id":"captcha-id","image":"base64-image","other":"ok"}}`
	masked := sanitizeAccessLogPayload("/sweet_admin/admin/captcha", payload)

	for _, leaked := range []string{"captcha-id", "base64-image"} {
		if strings.Contains(masked, leaked) {
			t.Fatalf("captcha response leaked %q: %s", leaked, masked)
		}
	}
	if !strings.Contains(masked, "ok") {
		t.Fatalf("captcha response did not retain non-sensitive metadata: %s", masked)
	}
}

func TestSanitizeAccessLogPayloadRedactsCaptchaFieldsEverywhere(t *testing.T) {
	payload := `{"captcha":"1234","captcha_id":"captcha-id","name":"admin"}`
	masked := sanitizeAccessLogPayload("/sweet_admin/admin/login", payload)

	for _, leaked := range []string{"1234", "captcha-id"} {
		if strings.Contains(masked, leaked) {
			t.Fatalf("payload leaked captcha field %q: %s", leaked, masked)
		}
	}
	if !strings.Contains(masked, "admin") {
		t.Fatalf("payload did not retain ordinary field: %s", masked)
	}
}

func TestSanitizeAccessLogURLPathMasksSmsMobile(t *testing.T) {
	tests := map[string]string{
		"/sweet_admin/api/send_sms/13800138000/LOGIN_CODE":      "/sweet_admin/api/send_sms/***/LOGIN_CODE",
		"/sweet_admin/api/check_sms_status/biz-1/13800138000":   "/sweet_admin/api/check_sms_status/biz-1/***",
		"/sweet_admin/admin/generalization/query/code/sys_user": "/sweet_admin/admin/generalization/query/code/sys_user",
		"/sweet_admin/api/sms_code_login":                       "/sweet_admin/api/sms_code_login",
	}
	for input, want := range tests {
		if got := sanitizeAccessLogURLPath(input); got != want {
			t.Fatalf("sanitizeAccessLogURLPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSanitizeLogPayloadRedactsApplicationSecretResponse(t *testing.T) {
	payload := `{"success":true,"data":{"app_key":"client-key","app_secret":"One-Time-App-Secret","name":"client"}}`
	masked := sanitizeLogPayload(payload)

	if strings.Contains(masked, "One-Time-App-Secret") {
		t.Fatalf("payload leaked application secret: %s", masked)
	}
	if !strings.Contains(masked, "client-key") || !strings.Contains(masked, "client") {
		t.Fatalf("payload did not retain non-sensitive application metadata: %s", masked)
	}
}

func TestSanitizeLogPayloadTruncatesLargePayload(t *testing.T) {
	payload := `{"password":"admin123","content":"` + strings.Repeat("x", maxAuditPayloadLength) + `"}`
	masked := sanitizeLogPayload(payload)

	if len(masked) <= maxAuditPayloadLength {
		t.Fatalf("expected payload to be truncated, got length %d", len(masked))
	}
	if !strings.Contains(masked, "[truncated]") {
		t.Fatalf("expected truncation marker, got %s", masked[len(masked)-64:])
	}
	if strings.Contains(masked, "admin123") {
		t.Fatalf("payload leaked sensitive password: %s", masked[:128])
	}
}

func TestSanitizeLogPayloadRedactsIncompleteJSONText(t *testing.T) {
	payload := `{"access_token":"token-value","content":"` + strings.Repeat("x", 128)
	masked := sanitizeLogPayload(payload)

	if strings.Contains(masked, "token-value") {
		t.Fatalf("incomplete JSON payload leaked token: %s", masked)
	}
	if !strings.Contains(masked, `"access_token":"***"`) {
		t.Fatalf("incomplete JSON payload did not mask token key: %s", masked)
	}
}

func TestSanitizeLogPayloadRedactsIncompleteCaptchaText(t *testing.T) {
	payload := `{"captcha_id":"captcha-id","captcha":"1234","content":"` + strings.Repeat("x", 128)
	masked := sanitizeLogPayload(payload)

	for _, leaked := range []string{"captcha-id", "1234"} {
		if strings.Contains(masked, leaked) {
			t.Fatalf("incomplete JSON payload leaked captcha value %q: %s", leaked, masked)
		}
	}
}

func TestShouldCaptureRequestBodySkipsLargeJSON(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/sweet_admin/admin/smoke", strings.NewReader("{}"))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.ContentLength = maxAuditPayloadLength + 1

	if shouldCaptureRequestBody(ctx) {
		t.Fatal("expected large JSON request body to be skipped")
	}
}
