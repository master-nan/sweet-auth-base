package errors

import (
	"backend/dto/response"
	stderrors "errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestErrorCategories(t *testing.T) {
	rootErr := stderrors.New("storage unavailable")
	tests := []struct {
		name string
		err  error
		want response.ErrorCategory
	}{
		{name: "parameter", err: ErrParamInvalid, want: response.ErrorCategoryParameter},
		{name: "permission", err: ErrPermissionDenied, want: response.ErrorCategoryPermission},
		{name: "business", err: NewBadRequestError("业务校验失败"), want: response.ErrorCategoryBusiness},
		{name: "database", err: WrapDatabaseError(rootErr), want: response.ErrorCategoryDatabase},
		{name: "system", err: WrapSystemError(rootErr), want: response.ErrorCategorySystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CategoryOf(tt.err); got != tt.want {
				t.Fatalf("expected category %q, got %q", tt.want, got)
			}
		})
	}
}

func TestWrapErrorPreservesCauseAndAdminError(t *testing.T) {
	rootErr := stderrors.New("unique constraint")
	err := WrapBusinessError(rootErr, http.StatusConflict, 81001, "记录冲突")

	if !stderrors.Is(err, rootErr) {
		t.Fatal("expected wrapped error to retain root cause")
	}
	var adminErr *response.AdminError
	if !stderrors.As(err, &adminErr) {
		t.Fatalf("expected AdminError, got %T", err)
	}
	if adminErr.StatusCode != http.StatusConflict ||
		adminErr.ErrorCode != 81001 ||
		adminErr.ErrorMessage != "记录冲突" ||
		adminErr.Category != response.ErrorCategoryBusiness {
		t.Fatalf("unexpected wrapped error: %#v", adminErr)
	}
}

func TestToClientErrorFindsWrappedAdminError(t *testing.T) {
	err := fmt.Errorf("service context: %w", ErrPermissionDenied)

	clientErr, classified := ToClientError(err)
	if !classified {
		t.Fatal("expected wrapped AdminError to remain classified")
	}
	if clientErr.StatusCode != http.StatusForbidden ||
		clientErr.ErrorCode != 30006 ||
		clientErr.ErrorMessage != "无权限访问" {
		t.Fatalf("unexpected client error: %#v", clientErr)
	}
	if clientErr.Cause != nil {
		t.Fatalf("client error must not expose cause: %#v", clientErr)
	}
}

func TestToClientErrorSanitizesUnknownError(t *testing.T) {
	rootErr := stderrors.New("pq: password=do-not-expose")

	clientErr, classified := ToClientError(rootErr)
	if classified {
		t.Fatal("expected raw system error to remain unclassified")
	}
	if clientErr.StatusCode != http.StatusInternalServerError ||
		clientErr.ErrorCode != ErrorCodeGeneric ||
		clientErr.ErrorMessage != "系统异常" ||
		clientErr.Category != response.ErrorCategorySystem {
		t.Fatalf("unexpected fallback error: %#v", clientErr)
	}
	if strings.Contains(clientErr.ErrorMessage, "do-not-expose") {
		t.Fatalf("raw error leaked to client: %#v", clientErr)
	}
}

func TestToClientErrorRecognizesRawParameterError(t *testing.T) {
	_, rawErr := strconv.Atoi("not-an-id")
	if rawErr == nil {
		t.Fatal("expected strconv error")
	}

	clientErr, classified := ToClientError(rawErr)
	if !classified {
		t.Fatal("expected number parsing error to be classified")
	}
	if clientErr.StatusCode != http.StatusBadRequest ||
		clientErr.ErrorCode != ErrorCodeParamInvalid ||
		clientErr.Category != response.ErrorCategoryParameter {
		t.Fatalf("unexpected parameter error: %#v", clientErr)
	}
}
