package middleware

import (
	"backend/internal/errors"
	"backend/model"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type CasbinOptions struct {
	EnforcePolicyCoverage bool
}

func CasbinHandler(enforcer *casbin.Enforcer, options ...CasbinOptions) gin.HandlerFunc {
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
		act := c.Request.Method

		if allowAuthenticatedCommonRoute(objs, act) {
			if _, ok := authenticatedUser(c); !ok {
				_ = c.Error(errors.ErrUserNotLogin)
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if !hasPolicyForRequest(enforcer, objs, act) {
			if opts.EnforcePolicyCoverage && !allowMissingCasbinPolicy(objs, act) {
				_ = c.Error(errors.ErrPermissionDenied)
				c.Abort()
				return
			}
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
					_ = c.Error(errors.ErrInternalServer)
					c.Abort()
					return
				}
				if allowed {
					c.Next()
					return
				}
			}
		}

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
		switch obj {
		case "/admin/logout":
			if act == "POST" {
				return true
			}
		case "/admin/user/me", "/admin/menu/my":
			if act == "GET" {
				return true
			}
		case "/admin/user/password":
			if act == "POST" {
				return true
			}
		default:
			if allowControllerScopedPermissionRoute(obj, act) {
				return true
			}
		}
	}
	return false
}

// allowControllerScopedPermissionRoute 只允许已登录用户进入由控制器做二次权限判断的通用接口。
// 低代码接口路径是复用的，Casbin 只能看到 /admin/generalization/query/code/:code，
// 看不到当前访问的是哪张发布表，所以表归属、菜单授权、按钮动作和数据权限必须在控制器中继续校验。
func allowControllerScopedPermissionRoute(obj, act string) bool {
	switch obj {
	case "/admin/generalization/query/:id":
		return act == "POST"
	case "/admin/generalization/query/code/:code":
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
		switch obj {
		case "/admin/logout":
			if act == "POST" {
				return true
			}
		case "/admin/user/me", "/admin/menu/my":
			if act == "GET" {
				return true
			}
		case "/admin/generalization/detail/code/:code/:id":
			if act == "GET" {
				return true
			}
		}
	}
	return false
}

func hasPolicyForRequest(enforcer *casbin.Enforcer, objs []string, act string) bool {
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
