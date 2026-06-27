package response

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
