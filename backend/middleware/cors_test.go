package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCorsHandlerWildcardDoesNotAllowCredentials(t *testing.T) {
	router := gin.New()
	router.Use(CorsHandler())
	router.OPTIONS("/smoke", func(ctx *gin.Context) {})

	req := httptest.NewRequest(http.MethodOptions, "/smoke", nil)
	req.Header.Set("Origin", "https://example.com")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow origin = %q, want wildcard", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("wildcard CORS must not allow credentials, got %q", got)
	}
}

func TestCorsHandlerAllowedOriginCanAllowCredentials(t *testing.T) {
	router := gin.New()
	router.Use(CorsHandler(CorsOptions{
		AllowedOrigins:   []string{"https://admin.example.com"},
		AllowCredentials: true,
	}))
	router.OPTIONS("/smoke", func(ctx *gin.Context) {})

	req := httptest.NewRequest(http.MethodOptions, "/smoke", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Fatalf("allow origin = %q, want configured origin", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials = %q, want true", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Fatalf("allow headers = %q, want requested headers", got)
	}
}

func TestCorsHandlerRejectsUnlistedOrigin(t *testing.T) {
	router := gin.New()
	router.Use(CorsHandler(CorsOptions{AllowedOrigins: []string{"https://admin.example.com"}}))
	router.OPTIONS("/smoke", func(ctx *gin.Context) {})

	req := httptest.NewRequest(http.MethodOptions, "/smoke", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow origin for unlisted origin: %q", got)
	}
}
