package middleware

import (
	"backend/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

func TestCasbinModelAllowsDirectRolePolicy(t *testing.T) {
	enforcer, err := casbin.NewEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("audit_manager", "/admin/log/access/query", "POST"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	allowed, err := enforcer.Enforce("audit_manager", "/admin/log/access/query", "POST")
	if err != nil {
		t.Fatalf("enforce audit_manager: %v", err)
	}
	if !allowed {
		t.Fatal("expected direct role policy to allow audit_manager")
	}

	allowed, err = enforcer.Enforce("ordinary_user", "/admin/log/access/query", "POST")
	if err != nil {
		t.Fatalf("enforce ordinary_user: %v", err)
	}
	if allowed {
		t.Fatal("expected ordinary_user to be denied without policy")
	}
}

func TestCasbinHandlerStrictModeDeniesRouteWithoutPolicy(t *testing.T) {
	enforcer := newTestEnforcer(t)
	called := false
	router := newCasbinTestRouter(enforcer, CasbinOptions{EnforcePolicyCoverage: true}, &called)

	req := httptest.NewRequest(http.MethodGet, "/admin/uncovered", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if called {
		t.Fatal("expected uncovered route to be blocked in strict mode")
	}
}

func TestCasbinHandlerCompatibilityModeAllowsRouteWithoutPolicy(t *testing.T) {
	enforcer := newTestEnforcer(t)
	called := false
	router := newCasbinTestRouter(enforcer, CasbinOptions{}, &called)

	req := httptest.NewRequest(http.MethodGet, "/admin/uncovered", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("expected uncovered route to pass in compatibility mode")
	}
}

func TestCasbinHandlerAllowsAuthenticatedCommonRouteWithoutPolicy(t *testing.T) {
	for _, route := range []string{
		"/admin/menu/my",
		"/admin/runtime/dict/:code",
		"/admin/runtime/table/:code",
		"/admin/runtime/query-scopes/:scope",
		"/admin/runtime/query-schemes/available",
	} {
		t.Run(route, func(t *testing.T) {
			enforcer := newTestEnforcer(t)
			called := false
			router := gin.New()
			router.Use(func(ctx *gin.Context) {
				ctx.Set("user", model.SysUser{UserName: "tom", Roles: []model.SysRole{{Name: "audit_viewer"}}})
				ctx.Next()
			})
			router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
			router.GET(route, func(ctx *gin.Context) {
				called = true
				ctx.Status(http.StatusNoContent)
			})

			requestPath := strings.Replace(strings.Replace(route, ":code", "example", 1), ":scope", "system.user.list", 1)
			req := httptest.NewRequest(http.MethodGet, requestPath, nil)
			router.ServeHTTP(httptest.NewRecorder(), req)
			if !called {
				t.Fatalf("expected authenticated common route %s to pass without policy", route)
			}
		})
	}
}

func TestCasbinHandlerQuerySchemePersonalIsControllerScopedButSharedIsNot(t *testing.T) {
	enforcer := newTestEnforcer(t)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{UserName: "tom", Roles: []model.SysRole{{Name: "viewer"}}})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	personalCalled := false
	sharedCalled := false
	router.POST("/admin/query-schemes/personal", func(ctx *gin.Context) { personalCalled = true })
	router.POST("/admin/query-schemes/shared", func(ctx *gin.Context) { sharedCalled = true })
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/admin/query-schemes/personal", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/admin/query-schemes/shared", nil))
	if !personalCalled || sharedCalled {
		t.Fatalf("personal_called=%v shared_called=%v", personalCalled, sharedCalled)
	}
}

func TestCasbinHandlerRuntimeReadStillRequiresAuthenticatedUser(t *testing.T) {
	enforcer := newTestEnforcer(t)
	called := false
	router := gin.New()
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.GET("/admin/runtime/dict/:code", func(ctx *gin.Context) { called = true })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/admin/runtime/dict/example", nil))
	if called {
		t.Fatal("runtime dictionary route must reject unauthenticated users")
	}
}

func TestCasbinHandlerAllowsNonAdminRolePolicy(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("audit_viewer", "/admin/log/access/query", "POST"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	called := false
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{UserName: "tom", Roles: []model.SysRole{{Name: "audit_viewer"}}})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.POST("/admin/log/access/query", func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/log/access/query", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("expected non-admin role policy to pass")
	}
}

func TestCasbinHandlerDeniesAPIWithoutGrantedButtonPolicy(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("example_operator", "/admin/example/:id", "PUT"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	called := false
	router := newRoleProtectedRouter(t, enforcer, "example_viewer", &called)

	req := httptest.NewRequest(http.MethodPut, "/admin/example/7", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if called {
		t.Fatal("expected API call without granted button policy to be denied")
	}
}

func TestCasbinHandlerAllowsAPIWithGrantedButtonPolicy(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("example_operator", "/admin/example/:id", "PUT"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	called := false
	router := newRoleProtectedRouter(t, enforcer, "example_operator", &called)

	req := httptest.NewRequest(http.MethodPut, "/admin/example/7", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("expected API call with granted button policy to pass")
	}
}

func TestCasbinHandlerAllowsSelfPasswordChangeWithExistingPolicy(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("password_admin", "/admin/user/password", "POST"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	called := false
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{UserName: "ordinary_user", Roles: []model.SysRole{{Name: "ordinary_user"}}})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.POST("/admin/user/password", func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/user/password", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("expected authenticated user to change own password without menu permission")
	}
}

func TestCasbinHandlerAllowsControllerScopedLowCodeRouteForAuthenticatedUser(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("lowcode_admin", "/admin/generalization/query/code/:code", "POST"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	called := false
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{UserName: "ordinary_user", Roles: []model.SysRole{{Name: "ordinary_user"}}})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.POST("/admin/generalization/query/code/:code", func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/generalization/query/code/demo_file_page", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("expected low-code route to pass middleware and leave table permission to controller")
	}
}

func TestCasbinHandlerDeniesControllerScopedLowCodeRouteWithoutLogin(t *testing.T) {
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("lowcode_admin", "/admin/generalization/query/code/:code", "POST"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	called := false
	router := gin.New()
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.POST("/admin/generalization/query/code/:code", func(ctx *gin.Context) {
		called = true
		ctx.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/generalization/query/code/demo_file_page", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if called {
		t.Fatal("expected low-code route to require login before controller permission checks")
	}
}

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	enforcer, err := casbin.NewEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	return enforcer
}

func newCasbinTestRouter(enforcer *casbin.Enforcer, options CasbinOptions, called *bool) *gin.Engine {
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{UserName: "tester", Roles: []model.SysRole{{Name: "tester"}}})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, options))
	router.GET("/admin/uncovered", func(ctx *gin.Context) {
		*called = true
		ctx.Status(http.StatusNoContent)
	})
	return router
}

func newRoleProtectedRouter(t *testing.T, enforcer *casbin.Enforcer, roleName string, called *bool) *gin.Engine {
	t.Helper()
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("user", model.SysUser{UserName: "tester", Roles: []model.SysRole{{Name: roleName}}})
		ctx.Next()
	})
	router.Use(CasbinHandler(enforcer, CasbinOptions{EnforcePolicyCoverage: true}))
	router.PUT("/admin/example/:id", func(ctx *gin.Context) {
		*called = true
		ctx.Status(http.StatusNoContent)
	})
	return router
}
