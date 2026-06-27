package initialize

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReadinessHandlerReportsMissingDependencies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/readyz", readinessHandler(&App{}))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for missing dependencies, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid readiness json: %v", err)
	}
	if body["status"] != "not_ready" {
		t.Fatalf("expected not_ready status, got %v", body["status"])
	}
	rendered := rec.Body.String()
	for _, secretish := range []string{"password", "sweet_admin", "postgres://", "host="} {
		if strings.Contains(rendered, secretish) {
			t.Fatalf("readiness response leaked sensitive detail %q: %s", secretish, rendered)
		}
	}
}
