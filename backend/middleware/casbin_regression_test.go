package middleware

import (
	"backend/model"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestCasbinHandlerSameUserThousandRequestsAreStable(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("stable_role", "/admin/stable/:id", http.MethodGet); err != nil {
		t.Fatalf("add stable policy: %v", err)
	}
	router := newAuthorizationRegressionRouter(enforcer)

	for i := 0; i < 1000; i++ {
		recorder := performAuthorizationRequest(router, http.MethodGet, fmt.Sprintf("/admin/stable/%d", i), "stable_role")
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d returned %d, want %d", i, recorder.Code, http.StatusNoContent)
		}
	}
}

func TestCasbinHandlerConcurrentUsersAndDecisionsDoNotCrossTalk(t *testing.T) {
	enforcer := newTestEnforcer(t)
	policies := [][]string{
		{"role_a", "/admin/a/:id", http.MethodGet},
		{"role_b", "/admin/b/:id", http.MethodGet},
	}
	if _, err := enforcer.AddPolicies(policies); err != nil {
		t.Fatalf("add policies: %v", err)
	}
	router := newAuthorizationRegressionRouter(enforcer)

	type requestCase struct {
		path       string
		role       string
		wantStatus int
	}
	cases := []requestCase{
		{path: "/admin/a/1", role: "role_a", wantStatus: http.StatusNoContent},
		{path: "/admin/b/1", role: "role_b", wantStatus: http.StatusNoContent},
		{path: "/admin/a/1", role: "role_b", wantStatus: http.StatusForbidden},
		{path: "/admin/b/1", role: "role_a", wantStatus: http.StatusForbidden},
	}

	var failures atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 250; i++ {
		for _, item := range cases {
			item := item
			wg.Add(1)
			go func() {
				defer wg.Done()
				recorder := performAuthorizationRequest(router, http.MethodGet, item.path, item.role)
				if recorder.Code != item.wantStatus {
					failures.Add(1)
				}
			}()
		}
	}
	wg.Wait()
	if got := failures.Load(); got != 0 {
		t.Fatalf("authorization decisions crossed request boundaries %d times", got)
	}
}

func TestCasbinHandlerConcurrentUnrelatedPolicyUpdatesKeepStableDecision(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("new synced enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("stable_role", "/admin/stable/:id", http.MethodGet); err != nil {
		t.Fatalf("add stable policy: %v", err)
	}
	router := newAuthorizationRegressionRouter(enforcer)

	var failures atomic.Int64
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			if _, err := enforcer.AddPolicy("writer_role", "/admin/unrelated/:id", http.MethodPut); err != nil {
				failures.Add(1)
				return
			}
			if _, err := enforcer.RemovePolicy("writer_role", "/admin/unrelated/:id", http.MethodPut); err != nil {
				failures.Add(1)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			recorder := performAuthorizationRequest(router, http.MethodGet, "/admin/stable/1", "stable_role")
			if recorder.Code != http.StatusNoContent {
				failures.Add(1)
			}
		}
	}()
	wg.Wait()
	if got := failures.Load(); got != 0 {
		t.Fatalf("stable authorization failed during unrelated policy updates %d times", got)
	}
}

func TestCasbinHandlerMultiRoleUsesAnyAllowedRole(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("allowed_role", "/admin/multi/:id", http.MethodGet); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(responseMiddlewareForAuthorizationTest())
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{
			UserName: "multi_user",
			Roles: []model.SysRole{
				{Name: "denied_role"},
				{Name: "allowed_role"},
			},
		})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.GET("/admin/multi/:id", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	for i := 0; i < 1000; i++ {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/multi/1", nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("multi-role request %d returned %d", i, recorder.Code)
		}
	}
}

func TestCasbinHandlerPathMethodAndTrailingSlashBoundaries(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("stable_role", "/admin/stable/:id", http.MethodGet); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	router := newAuthorizationRegressionRouter(enforcer)
	router.RedirectTrailingSlash = false

	cases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "dynamic route ignores query string",
			method:     http.MethodGet,
			path:       "/admin/stable/42?view=detail",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "different method remains denied",
			method:     http.MethodPost,
			path:       "/admin/stable/42",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "trailing slash is not silently matched",
			method:     http.MethodGet,
			path:       "/admin/stable/42/",
			wantStatus: http.StatusForbidden,
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			for run := 0; run < 100; run++ {
				recorder := performAuthorizationRequest(router, item.method, item.path, "stable_role")
				if recorder.Code != item.wantStatus {
					t.Fatalf("run %d returned %d, want %d", run, recorder.Code, item.wantStatus)
				}
			}
		})
	}
}

func TestCasbinHandlerDiagnosticsCompareAllowedAndDeniedDecisions(t *testing.T) {
	t.Setenv(authorizationDiagnosticsEnv, "true")
	core, logs := observer.New(zap.InfoLevel)
	originalLogger := zap.L()
	zap.ReplaceGlobals(zap.New(core))
	t.Cleanup(func() {
		zap.ReplaceGlobals(originalLogger)
	})

	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("role_a", "/admin/a/:id", http.MethodGet); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	router := newAuthorizationRegressionRouter(enforcer)

	allowed := performAuthorizationRequest(router, http.MethodGet, "/admin/a/1", "role_a")
	denied := performAuthorizationRequest(router, http.MethodGet, "/admin/a/1", "role_b")
	if allowed.Code != http.StatusNoContent || denied.Code != http.StatusForbidden {
		t.Fatalf("unexpected decisions allowed=%d denied=%d", allowed.Code, denied.Code)
	}

	entries := logs.FilterMessage("authorization decision").All()
	if len(entries) != 2 {
		t.Fatalf("authorization diagnostic entries=%d, want 2", len(entries))
	}
	if entries[0].ContextMap()["decision_stage"] != "policy_allowed" ||
		entries[1].ContextMap()["decision_stage"] != "policy_denied" {
		t.Fatalf("unexpected diagnostic stages: %#v %#v", entries[0].ContextMap(), entries[1].ContextMap())
	}
	for _, entry := range entries {
		context := entry.ContextMap()
		for _, key := range []string{
			"request_id",
			"trace_id",
			"process_id",
			"instance_id",
			"user_id",
			"role_ids",
			"casbin_subjects",
			"http_method",
			"request_url_path",
			"gin_full_path",
			"casbin_objects",
			"casbin_action",
			"enforce_result",
			"policy_count",
			"permission_cache_status",
			"permission_cache_result",
			"error_code",
		} {
			if _, ok := context[key]; !ok {
				t.Fatalf("diagnostic entry missing %q: %#v", key, context)
			}
		}
	}
}

func newAuthorizationRegressionRouter(enforcer authorizationEnforcer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(responseMiddlewareForAuthorizationTest())
	router.Use(func(ctx *gin.Context) {
		role := ctx.GetHeader("X-Test-Role")
		EnsureLogContext(ctx)
		ctx.Set("token_subject", ctx.GetHeader("X-Test-Subject"))
		ctx.Set("user", model.SysUser{
			Basic:    model.Basic{Id: 100},
			UserName: role + "_user",
			Roles: []model.SysRole{{
				Basic: model.Basic{Id: 200},
				Name:  role,
			}},
		})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.GET("/admin/stable/:id", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})
	router.POST("/admin/stable/:id", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})
	router.GET("/admin/a/:id", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})
	router.GET("/admin/b/:id", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})
	return router
}

func responseMiddlewareForAuthorizationTest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
		if len(ctx.Errors) == 0 || ctx.Writer.Written() {
			return
		}
		ctx.Status(http.StatusForbidden)
	}
}

func performAuthorizationRequest(router *gin.Engine, method, path, role string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("X-Test-Role", role)
	request.Header.Set("X-Test-Subject", "100")
	router.ServeHTTP(recorder, request)
	return recorder
}
