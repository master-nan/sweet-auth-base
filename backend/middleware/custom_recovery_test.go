package middleware

import (
	"backend/dto/response"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestCustomRecoveryReturnsStableInternalError(t *testing.T) {
	restoreLogger := zap.ReplaceGlobals(zap.NewNop())
	t.Cleanup(restoreLogger)

	engine := gin.New()
	engine.Use(CustomRecovery())
	engine.GET("/panic", func(*gin.Context) { panic("token=do-not-expose") })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	var payload response.AdminError
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if recorder.Code != http.StatusInternalServerError || payload.ErrorCode != 10000 || payload.ErrorMessage != "系统异常" {
		t.Fatalf("unexpected response: %#v", payload)
	}
	if strings.Contains(recorder.Body.String(), "do-not-expose") {
		t.Fatalf("panic detail leaked: %s", recorder.Body.String())
	}
}
