package middleware

import (
	"backend/internal/audit"
	error2 "backend/internal/errors"
	"backend/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const bearerLength = len("Bearer ")

// AuthHandler 是统一认证链的HTTP Adapter，只把校验后的可信身份写入请求上下文。
func AuthHandler(authService *service.AuthApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if len(authorization) < bearerLength {
			_ = c.Error(error2.ErrUserNotLogin)
			c.Abort()
			return
		}
		access, err := authService.AuthenticateAccessToken(c.Request.Context(), authorization[bearerLength:])
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		if access.MustChangePassword && !allowsRequiredPasswordChange(c.Request.Method, c.Request.URL.Path) {
			_ = c.Error(error2.ErrPasswordChangeRequired)
			c.Abort()
			return
		}
		c.Set("user", access.User)
		c.Set("id", access.User.Id)
		c.Set("token_subject", strconv.Itoa(access.User.Id))
		c.Set("auth_session_id", access.SessionID)
		c.Set("auth_token_id", access.TokenID)
		c.Set("token_expires_at", access.ExpiresAt)
		InjectAuditSubject(c, audit.NewAuditSubject(access.User.Id, access.User.UserName))
		c.Next()
	}
}

func allowsRequiredPasswordChange(method, path string) bool {
	if method != http.MethodPost {
		return false
	}
	path = strings.TrimSuffix(path, "/")
	return strings.HasSuffix(path, "/admin/user/password") || strings.HasSuffix(path, "/api/user/password")
}
