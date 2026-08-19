package errors

import (
	stderrors "errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestApplicationErrorCategories(t *testing.T) {
	rootErr := stderrors.New("storage unavailable")
	tests := []struct {
		name string
		err  error
		want Category
	}{
		{name: "parameter", err: ErrParamInvalid, want: CategoryParameter},
		{name: "permission", err: ErrPermissionDenied, want: CategoryPermission},
		{name: "business", err: NewValidationError("业务校验失败"), want: CategoryBusiness},
		{name: "database", err: WrapDatabaseError(rootErr), want: CategoryDatabase},
		{name: "system", err: WrapSystemError(rootErr), want: CategorySystem},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CategoryOf(test.err); got != test.want {
				t.Fatalf("category = %q, want %q", got, test.want)
			}
		})
	}
}

func TestApplicationErrorPreservesCause(t *testing.T) {
	rootErr := stderrors.New("unique constraint")
	err := WrapApplicationError(rootErr, KindConflict, CategoryBusiness, 81001, "记录冲突")

	if !stderrors.Is(err, rootErr) {
		t.Fatal("expected wrapped error to retain root cause")
	}
	var applicationErr *ApplicationError
	if !stderrors.As(err, &applicationErr) {
		t.Fatalf("expected ApplicationError, got %T", err)
	}
	if applicationErr.Kind != KindConflict || applicationErr.Code != 81001 ||
		applicationErr.SafeMessage != "记录冲突" || applicationErr.Category != CategoryBusiness {
		t.Fatalf("unexpected application error: %#v", applicationErr)
	}
}

func TestClassifyFindsWrappedApplicationError(t *testing.T) {
	err := fmt.Errorf("service context: %w", ErrPermissionDenied)
	if !stderrors.Is(err, ErrPermissionDenied) {
		t.Fatal("expected errors.Is to match the stable application error")
	}

	applicationErr, classified := Classify(err)
	if !classified {
		t.Fatal("expected wrapped ApplicationError to remain classified")
	}
	if applicationErr.Kind != KindForbidden || applicationErr.Code != 30006 ||
		applicationErr.SafeMessage != "无权限访问" {
		t.Fatalf("unexpected application error: %#v", applicationErr)
	}
}

func TestClassifySanitizesUnknownTechnicalError(t *testing.T) {
	rootErr := stderrors.New("pq: password=do-not-expose")

	applicationErr, classified := Classify(rootErr)
	if classified {
		t.Fatal("expected raw technical error to remain unclassified")
	}
	if applicationErr.Kind != KindInternal || applicationErr.Code != ErrorCodeGeneric ||
		applicationErr.SafeMessage != "系统异常" || applicationErr.Category != CategorySystem {
		t.Fatalf("unexpected fallback error: %#v", applicationErr)
	}
	if strings.Contains(applicationErr.SafeMessage, "do-not-expose") {
		t.Fatalf("raw error leaked: %#v", applicationErr)
	}
}

func TestDependencyWrappersKeepStableMessages(t *testing.T) {
	providerErr := stderrors.New("provider token=do-not-expose endpoint=internal")
	tests := []struct {
		name    string
		err     error
		code    int
		message string
	}{
		{name: "sms send", err: WrapSmsSendFailed(providerErr), code: 50005, message: "短信发送失败"},
		{name: "sms status", err: WrapSmsStatusQueryFailed(providerErr), code: 50010, message: "短信状态查询失败"},
		{name: "dingtalk", err: WrapDingTalkRequestFailed(providerErr), code: 60004, message: "钉钉服务请求失败"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			applicationErr, classified := Classify(test.err)
			if !classified || applicationErr.Kind != KindDependencyFailed || applicationErr.Code != test.code || applicationErr.SafeMessage != test.message {
				t.Fatalf("unexpected application error: %#v", applicationErr)
			}
			if strings.Contains(applicationErr.SafeMessage, "do-not-expose") || !stderrors.Is(test.err, providerErr) {
				t.Fatalf("provider detail leaked or cause was lost: %#v", applicationErr)
			}
		})
	}
}

func TestClassifyRecognizesRawParameterError(t *testing.T) {
	_, rawErr := strconv.Atoi("not-an-id")
	if rawErr == nil {
		t.Fatal("expected strconv error")
	}

	applicationErr, classified := Classify(rawErr)
	if !classified || applicationErr.Kind != KindInvalidArgument ||
		applicationErr.Code != ErrorCodeParamInvalid || applicationErr.Category != CategoryParameter {
		t.Fatalf("unexpected parameter error: %#v", applicationErr)
	}
}

func TestQuerySchemeErrorsRemainStableAndSafe(t *testing.T) {
	tests := []struct {
		err  error
		code int
		kind Kind
	}{
		{ErrQuerySchemeScopeForbidden, ErrorCodeQuerySchemeScopeForbidden, KindForbidden},
		{ErrQuerySchemeRevisionConflict, ErrorCodeQuerySchemeRevisionConflict, KindConflict},
		{ErrQuerySchemePayloadTooLarge, ErrorCodeQuerySchemePayloadTooLarge, KindPayloadTooLarge},
		{ErrQuerySchemeMetadataDegraded, ErrorCodeQuerySchemeMetadataDegraded, KindUnprocessable},
	}
	for _, test := range tests {
		applicationErr, ok := AsApplicationError(test.err)
		if !ok || applicationErr.Code != test.code || applicationErr.Kind != test.kind {
			t.Fatalf("query scheme error mismatch: %+v", applicationErr)
		}
		if strings.Contains(strings.ToLower(applicationErr.SafeMessage), "sql") {
			t.Fatalf("technical detail leaked: %q", applicationErr.SafeMessage)
		}
	}
}
