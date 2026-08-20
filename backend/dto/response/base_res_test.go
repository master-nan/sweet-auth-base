package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewResponseUsesStableSuccessContract(t *testing.T) {
	resp := NewResponse().
		SetData(map[string]string{"name": "baseline"}).
		SetTotal(1)

	if !resp.Success {
		t.Fatal("expected success response")
	}
	if resp.Code != http.StatusOK || resp.Message != "操作成功" || resp.Total != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	data, ok := resp.Data.(map[string]string)
	if !ok || data["name"] != "baseline" {
		t.Fatalf("unexpected response data: %#v", resp.Data)
	}
}

func TestAdminErrorUsesHTTPResponseFieldsOnly(t *testing.T) {
	adminErr := &AdminError{
		StatusCode:   http.StatusInternalServerError,
		ErrorCode:    10000,
		ErrorMessage: "系统异常",
		Success:      false,
	}

	payload, err := json.Marshal(adminErr)
	if err != nil {
		t.Fatalf("marshal admin error: %v", err)
	}
	var fields map[string]any
	if err = json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("decode admin error: %v", err)
	}
	if len(fields) != 4 || fields["error_message"] != "系统异常" || fields["success"] != false {
		t.Fatalf("unexpected HTTP error fields: %#v", fields)
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
