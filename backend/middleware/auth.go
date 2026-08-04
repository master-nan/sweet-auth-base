/**
 * @Author: Nan
 * @Date: 2023/3/15 11:32
 */

package middleware

import (
	"backend/config"
	"backend/dto/response"
	"backend/enum"
	"backend/internal/audit"
	"backend/internal/cache"
	error2 "backend/internal/errors"
	"backend/internal/token"
	"backend/model"
	"backend/service"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

const bearerLength = len("Bearer ")
const passwordChangedAtFutureTolerance = 5 * time.Minute

func AuthHandler(serverConfig *config.Server, tokenGenerator token.Generator, userService *service.SysUserService, tokenBlackCache *cache.TokenBlackCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		zap.L().Info("AuthHandler start")
		if len(authorization) < bearerLength {
			_ = c.Error(error2.ErrUserNotLogin)
			c.Abort()
			return
		}
		tk := authorization[bearerLength:]
		exists := tokenBlackCache.Exists(tk)
		if exists {
			_ = c.Error(error2.ErrTokenExpired)
			c.Abort()
			return
		}
		conf := token.Config{
			Issuer:                 serverConfig.Name,
			SecretKey:              serverConfig.Conf.Salt,
			AccessTokenExpiration:  7200,
			RefreshTokenExpiration: 60 * 60 * 24 * 30,
		}
		claims, err := tokenGenerator.ParseToken(tk, conf)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		if claims.Type != enum.AccessToken {
			_ = c.Error(error2.ErrTokenInvalidType)
			c.Abort()
			return
		}
		i, err := strconv.Atoi(claims.ID)
		if err != nil {
			_ = c.Error(error2.ErrTokenInvalid)
			c.Abort()
			return
		}
		user, err := userService.GetById(i)
		if err != nil {
			var e *response.AdminError
			switch {
			case errors.As(err, &e):
				_ = c.Error(err)
			default:
				_ = c.Error(error2.WrapSystemError(err))
			}
			c.Abort()
			return
		}
		if tokenIssuedBeforePasswordChange(claims.IssuedAt, user) {
			_ = c.Error(error2.ErrTokenExpired)
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Set("id", i)
		c.Set("token_subject", claims.ID)
		InjectAuditSubject(c, audit.NewAuditSubject(user.Id, user.UserName))
		c.Next()
		zap.L().Info("AuthHandler end")
	}
}

func tokenIssuedBeforePasswordChange(issuedAt time.Time, user model.SysUser) bool {
	return tokenIssuedBeforePasswordChangeAt(issuedAt, user, time.Now().UTC())
}

func tokenIssuedBeforePasswordChangeAt(issuedAt time.Time, user model.SysUser, now time.Time) bool {
	if user.PasswordChangedAt == nil || time.Time(*user.PasswordChangedAt).IsZero() {
		return false
	}
	passwordChangedAt := time.Time(*user.PasswordChangedAt)
	if passwordChangedAt.After(now.Add(passwordChangedAtFutureTolerance)) {
		zap.L().Warn("password_changed_at is in the future; skip token invalidation check", zap.Time("password_changed_at", passwordChangedAt), zap.Time("now", now))
		return false
	}
	return issuedAt.Add(time.Second).Before(passwordChangedAt)
}
