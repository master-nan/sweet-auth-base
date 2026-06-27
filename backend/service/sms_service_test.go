package service

import "testing"

func TestRedactSmsTemplateParamsForLogMasksVerificationCode(t *testing.T) {
	params := map[string]interface{}{
		"code": "123456",
		"name": "Nan",
	}

	redacted := redactSmsTemplateParamsForLog(params)
	if redacted["code"] != "***" {
		t.Fatalf("expected sms verification code to be redacted, got %#v", redacted["code"])
	}
	if redacted["name"] != "Nan" {
		t.Fatalf("expected non-code params to be preserved, got %#v", redacted["name"])
	}
	if params["code"] != "123456" {
		t.Fatalf("redaction must not mutate original params, got %#v", params["code"])
	}
}
