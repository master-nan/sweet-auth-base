package response

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewResponseUsesStableSuccessContract(t *testing.T) {
	resp := NewResponse().
		SetData(map[string]string{"name": "baseline"}).
		SetTotal(1).
		SetMessage("完成").
		SetCode(http.StatusCreated)

	if !resp.Success {
		t.Fatal("expected success response")
	}
	if resp.Code != http.StatusCreated || resp.Message != "完成" || resp.Total != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	data, ok := resp.Data.(map[string]string)
	if !ok || data["name"] != "baseline" {
		t.Fatalf("unexpected response data: %#v", resp.Data)
	}
}

func TestAdminErrorPreservesCauseWithoutSerializingIt(t *testing.T) {
	rootErr := errors.New("database password=do-not-expose")
	adminErr := &AdminError{
		StatusCode:   http.StatusInternalServerError,
		ErrorCode:    10000,
		ErrorMessage: "系统异常",
		Success:      false,
		Category:     ErrorCategoryDatabase,
		Cause:        rootErr,
	}

	if !errors.Is(adminErr, rootErr) {
		t.Fatal("expected AdminError to preserve its cause")
	}
	payload, err := json.Marshal(adminErr)
	if err != nil {
		t.Fatalf("marshal admin error: %v", err)
	}
	if strings.Contains(string(payload), "do-not-expose") ||
		strings.Contains(string(payload), "database") {
		t.Fatalf("internal error details leaked into JSON: %s", payload)
	}

	clientErr := adminErr.ForClient()
	if clientErr == adminErr || clientErr.Cause != nil {
		t.Fatalf("expected a detached client error without cause: %#v", clientErr)
	}
	if clientErr.ErrorCode != adminErr.ErrorCode ||
		clientErr.ErrorMessage != adminErr.ErrorMessage {
		t.Fatalf("client error changed public contract: %#v", clientErr)
	}
}

func TestBufferedResponseWriterCapsCapturedBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	writer := &BufferedResponseWriter{
		ResponseWriter: ctx.Writer,
		Body:           bytes.NewBuffer(nil),
		MaxBodyBytes:   5,
	}

	n, err := writer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len("hello world") {
		t.Fatalf("unexpected write length: %d", n)
	}
	if got := writer.Body.String(); got != "hello" {
		t.Fatalf("captured body = %q, want capped prefix", got)
	}
	if !writer.Truncated {
		t.Fatal("expected captured body to be marked truncated")
	}
	if got := recorder.Body.String(); got != "hello world" {
		t.Fatalf("underlying response body changed: %q", got)
	}
}
