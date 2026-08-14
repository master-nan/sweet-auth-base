package service

import (
	"backend/dto/response"
	"backend/enum"
	error2 "backend/internal/errors"
	"backend/model"
	"encoding/json"
	"errors"
	"testing"

	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
)

func TestRedactSmsTemplateParamsForLogMasksVerificationCode(t *testing.T) {
	params := map[string]interface{}{
		"code": "123456",
		"name": "Nan",
	}

	redacted := redactSmsTemplateParamsForLog(params)
	if redacted["code"] != "***" {
		t.Fatalf("expected sms verification code to be redacted, got %#v", redacted["code"])
	}
	if redacted["name"] != "***" {
		t.Fatalf("expected every template value to be redacted, got %#v", redacted["name"])
	}
	if params["code"] != "123456" {
		t.Fatalf("redaction must not mutate original params, got %#v", params["code"])
	}
}

func TestSmsStatusFromProviderReturnsSafeStatus(t *testing.T) {
	tests := map[string]struct {
		providerStatus int64
		expected       enum.SmsStatus
	}{
		"sending": {providerStatus: 1, expected: enum.SmsStatusSending},
		"failed":  {providerStatus: 2, expected: enum.SmsStatusFailed},
		"success": {providerStatus: 3, expected: enum.SmsStatusSuccess},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := &dysmsapi20170525.QuerySendDetailsResponseBody{
				Code: tea.String("OK"),
				SmsSendDetailDTOs: &dysmsapi20170525.QuerySendDetailsResponseBodySmsSendDetailDTOs{
					SmsSendDetailDTO: []*dysmsapi20170525.QuerySendDetailsResponseBodySmsSendDetailDTOsSmsSendDetailDTO{{
						SendStatus: tea.Int64(tc.providerStatus), Content: tea.String("sensitive body"), PhoneNum: tea.String("synthetic-mobile-a"),
					}},
				},
			}
			status, err := smsStatusFromProvider(result)
			if err != nil {
				t.Fatalf("unexpected status error: %v", err)
			}
			if status != tc.expected {
				t.Fatalf("expected status %d, got %d", tc.expected, status)
			}
			payload, err := json.Marshal(response.SmsStatusRes{Status: status})
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(payload, &fields); err != nil {
				t.Fatal(err)
			}
			if len(fields) != 1 || fields["status"] != tc.expected.String() {
				t.Fatalf("unexpected safe payload fields: %#v", fields)
			}
		})
	}
}

func TestSmsStatusLogOwnershipRequiresApplicationAndMobile(t *testing.T) {
	log := model.SmsLog{ApplicationId: 7, Mobile: "synthetic-mobile-a"}
	if !smsStatusLogOwnedBy(log, 7, "synthetic-mobile-a") {
		t.Fatal("expected matching application and mobile to own the status record")
	}
	if smsStatusLogOwnedBy(log, 8, "synthetic-mobile-a") || smsStatusLogOwnedBy(log, 7, "synthetic-mobile-b") {
		t.Fatal("status ownership must reject another application or mobile")
	}
}

func TestSmsStatusFromProviderRejectsMissingDetails(t *testing.T) {
	for name, result := range map[string]*dysmsapi20170525.QuerySendDetailsResponseBody{
		"nil":              nil,
		"empty":            {Code: tea.String("OK"), SmsSendDetailDTOs: &dysmsapi20170525.QuerySendDetailsResponseBodySmsSendDetailDTOs{}},
		"provider failure": {Code: tea.String("ERROR")},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := smsStatusFromProvider(result); !errors.Is(err, error2.ErrSmsStatusQueryFailed) {
				t.Fatalf("expected stable query error, got %v", err)
			}
		})
	}
}
