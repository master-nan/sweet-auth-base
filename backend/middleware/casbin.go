package middleware

import (
	"backend/internal/errors"
	"backend/model"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const authorizationDiagnosticsEnv = "SWEET_AUTHZ_DIAGNOSTICS"

type CasbinOptions struct {
	EnforcePolicyCoverage bool
}

type authorizationEnforcer interface {
	Enforce(rvals ...interface{}) (bool, error)
	GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error)
	GetPolicy() ([][]string, error)
}

func CasbinHandler(enforcer authorizationEnforcer, options ...CasbinOptions) gin.HandlerFunc {
	opts := CasbinOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	return func(c *gin.Context) {
		if enforcer == nil {
			c.Next()
			return
		}

		objs := resolveRequestObjects(c)
		act := strings.ToUpper(c.Request.Method)

		if allowAuthenticatedCommonRoute(objs, act) {
			user, ok := authenticatedUser(c)
			if !ok {
				_ = c.Error(errors.ErrUserNotLogin)
				c.Abort()
				return
			}
			logAuthorizationDecision(c, enforcer, user, nil, objs, "", "", act, true, "authenticated_common_route", 0, nil)
			c.Next()
			return
		}

		if !hasPolicyForRequest(enforcer, objs, act) {
			if opts.EnforcePolicyCoverage && !allowMissingCasbinPolicy(objs, act) {
				user, _ := authenticatedUser(c)
				logAuthorizationDecision(c, enforcer, user, casbinSubjects(user), objs, "", "", act, false, "policy_coverage_missing", authorizationErrorCode(errors.ErrPermissionDenied), nil)
				_ = c.Error(errors.ErrPermissionDenied)
				c.Abort()
				return
			}
			user, _ := authenticatedUser(c)
			logAuthorizationDecision(c, enforcer, user, casbinSubjects(user), objs, "", "", act, true, "policy_not_configured_allowed", 0, nil)
			c.Next()
			return
		}

		user, ok := authenticatedUser(c)
		if !ok {
			_ = c.Error(errors.ErrUserNotLogin)
			c.Abort()
			return
		}

		subjects := casbinSubjects(user)
		for _, obj := range objs {
			for _, subject := range subjects {
				allowed, err := enforcer.Enforce(subject, obj, act)
				if err != nil {
					logAuthorizationDecision(c, enforcer, user, subjects, objs, subject, obj, act, false, "enforce_error", authorizationErrorCode(errors.ErrInternalServer), err)
					_ = c.Error(errors.ErrInternalServer)
					c.Abort()
					return
				}
				if allowed {
					logAuthorizationDecision(c, enforcer, user, subjects, objs, subject, obj, act, true, "policy_allowed", 0, nil)
					c.Next()
					return
				}
			}
		}

		logAuthorizationDecision(c, enforcer, user, subjects, objs, "", "", act, false, "policy_denied", authorizationErrorCode(errors.ErrPermissionDenied), nil)
		_ = c.Error(errors.ErrPermissionDenied)
		c.Abort()
	}
}

func casbinSubjects(user model.SysUser) []string {
	seen := make(map[string]bool)
	subjects := make([]string, 0, len(user.Roles)+1)
	for _, role := range user.Roles {
		name := strings.TrimSpace(role.Name)
		if name == "" || seen[name] {
			continue
		}
		subjects = append(subjects, name)
		seen[name] = true
	}
	userName := strings.TrimSpace(user.UserName)
	if userName != "" && !seen[userName] {
		subjects = append(subjects, userName)
	}
	return subjects
}

func authenticatedUser(c *gin.Context) (model.SysUser, bool) {
	userVal, ok := c.Get("user")
	if !ok {
		return model.SysUser{}, false
	}
	user, ok := userVal.(model.SysUser)
	return user, ok
}

func allowAuthenticatedCommonRoute(objs []string, act string) bool {
	act = strings.ToUpper(strings.TrimSpace(act))
	for _, obj := range objs {
		if allowAuthenticatedIdentityRoute(obj, act) || (obj == "/admin/user/password" && act == "POST") || allowControllerScopedPermissionRoute(obj, act) {
			return true
		}
	}
	return false
}

func allowAuthenticatedIdentityRoute(obj, act string) bool {
	switch obj {
	case "/admin/logout":
		return act == "POST"
	case "/admin/user/me", "/admin/menu/my", "/admin/runtime/dict/:code", "/admin/runtime/table/:code",
		"/admin/runtime/query-scopes/:scope", "/admin/runtime/query-schemes/available",
		"/admin/query-schemes/:id", "/admin/runtime/notifications/unread-count",
		"/admin/runtime/notifications/recent", "/admin/runtime/notifications/:id":
		return act == "GET"
	case "/admin/runtime/query-schemes/:id/resolve", "/admin/query-schemes/query",
		"/admin/query-schemes/personal", "/admin/query-schemes/:id/copy-to-personal",
		"/admin/runtime/notifications/query", "/admin/runtime/notifications/:id/read",
		"/admin/runtime/notifications/read-all":
		return act == "POST"
	case "/admin/query-schemes/personal/:id", "/admin/query-schemes/personal/:id/default":
		return act == "PUT" || act == "DELETE"
	default:
		return false
	}
}

// allowControllerScopedPermissionRoute 只允许已登录用户进入由控制器做二次权限判断的通用接口。
// 低代码接口路径是复用的，Casbin 只能看到 /admin/generalization/query/code/:code，
// 看不到当前访问的是哪张发布表，所以表归属、菜单授权、按钮动作和数据权限必须在控制器中继续校验。
func allowControllerScopedPermissionRoute(obj, act string) bool {
	switch obj {
	case "/admin/report/query":
		return act == "POST"
	case "/admin/report/:id":
		return act == "GET"
	case "/admin/report/:id/run", "/admin/report/:id/export":
		return act == "POST"
	case "/admin/generalization/query/code/:code":
		return act == "POST"
	case "/admin/runtime/relation-fields/:fieldId/options":
		return act == "POST"
	case "/admin/generalization/detail/code/:code/:id":
		return act == "GET"
	case "/admin/generalization/create":
		return act == "POST"
	case "/admin/generalization/update":
		return act == "PUT"
	case "/admin/generalization/delete":
		return act == "DELETE"
	default:
		return false
	}
}

func allowMissingCasbinPolicy(objs []string, act string) bool {
	act = strings.ToUpper(strings.TrimSpace(act))
	for _, obj := range objs {
		if allowAuthenticatedIdentityRoute(obj, act) {
			return true
		}
		switch obj {
		case "/admin/generalization/detail/code/:code/:id":
			if act == "GET" {
				return true
			}
		}
	}
	return false
}

func hasPolicyForRequest(enforcer authorizationEnforcer, objs []string, act string) bool {
	for _, obj := range objs {
		policies, err := enforcer.GetFilteredPolicy(1, obj, act)
		if err != nil {
			continue
		}
		if len(policies) > 0 {
			return true
		}
	}
	return false
}

func logAuthorizationDecision(
	c *gin.Context,
	enforcer authorizationEnforcer,
	user model.SysUser,
	subjects []string,
	objects []string,
	selectedSubject string,
	selectedObject string,
	action string,
	allowed bool,
	stage string,
	errorCode int,
	enforceErr error,
) {
	if !authorizationDiagnosticsEnabled() {
		return
	}

	policyCount := -1
	policyReadStatus := "error"
	if policies, err := enforcer.GetPolicy(); err == nil {
		policyCount = len(policies)
		policyReadStatus = "ok"
	}

	roleIDs := make([]int, 0, len(user.Roles))
	roleNames := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.Id)
		roleNames = append(roleNames, role.Name)
	}

	tokenSubject, _ := c.Get("token_subject")
	tokenSubjectValue, _ := tokenSubject.(string)
	instanceID := strings.TrimSpace(os.Getenv("INSTANCE_ID"))
	if instanceID == "" {
		instanceID, _ = os.Hostname()
	}
	buildCommit := strings.TrimSpace(os.Getenv("BUILD_COMMIT"))
	if buildCommit == "" {
		buildCommit = "unknown"
	}

	fields := []zap.Field{
		zap.String("request_id", RequestID(c)),
		zap.String("trace_id", TraceID(c)),
		zap.Time("timestamp", time.Now().UTC()),
		zap.Int("process_id", os.Getpid()),
		zap.String("instance_id", instanceID),
		zap.String("build_commit", buildCommit),
		zap.Int("user_id", user.Id),
		zap.String("token_subject", tokenSubjectValue),
		zap.Ints("role_ids", roleIDs),
		zap.Strings("role_names", roleNames),
		zap.Strings("casbin_subjects", subjects),
		zap.String("http_method", c.Request.Method),
		zap.String("request_url_path", c.Request.URL.Path),
		zap.String("gin_full_path", c.FullPath()),
		zap.Strings("casbin_objects", objects),
		zap.String("casbin_subject", selectedSubject),
		zap.String("casbin_object", selectedObject),
		zap.String("casbin_action", action),
		zap.Bool("enforce_result", allowed),
		zap.String("decision_stage", stage),
		zap.String("policy_version", "not_available"),
		zap.Int("policy_count", policyCount),
		zap.String("policy_read_status", policyReadStatus),
		zap.String("permission_cache_key", ""),
		zap.String("permission_cache_status", "not_configured"),
		zap.String("permission_cache_result", "not_applicable"),
		zap.Int("error_code", errorCode),
	}
	if enforceErr != nil {
		fields = append(fields, zap.Error(enforceErr))
	}
	zap.L().Info("authorization decision", fields...)
}

func authorizationDiagnosticsEnabled() bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(authorizationDiagnosticsEnv)))
	return err == nil && enabled
}

func authorizationErrorCode(err error) int {
	clientErr, _ := toClientError(err)
	if clientErr == nil {
		return 0
	}
	return clientErr.ErrorCode
}

func resolveRequestObjects(c *gin.Context) []string {
	obj := c.FullPath()
	if obj == "" {
		obj = c.Request.URL.Path
	}
	objs := []string{obj}

	parts := strings.Split(obj, "/")
	if len(parts) > 2 && (parts[2] == "admin" || parts[2] == "api") {
		trimmed := "/" + strings.Join(parts[2:], "/")
		if trimmed != obj {
			objs = append(objs, trimmed)
		}
	}

	return objs
}
