package middleware

import (
	"backend/dto/response"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestNoRouteUsesStableHTTPErrorContract(t *testing.T) {
	restoreLogger := zap.ReplaceGlobals(zap.NewNop())
	t.Cleanup(restoreLogger)

	engine := gin.New()
	engine.NoRoute(NoRouteHandler())
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	var payload response.AdminError
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Success || payload.StatusCode != http.StatusNotFound || payload.ErrorCode != http.StatusNotFound {
		t.Fatalf("unexpected response: %#v", payload)
	}
}
