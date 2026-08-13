/**
 * @Author: Nan
 * @Date: 2024/10/24 15:09
 */

package middleware

import (
	"backend/internal/cache"
	"backend/internal/errors"
	"backend/internal/token"
	"backend/service"
	"crypto/subtle"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

func AuthHMACHandler(hmacTokenGenerator token.Generator,
	applicationCache *cache.ApplicationCache,
	applicationService *service.ApplicationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// hmac 认证
		xAppToken := c.GetHeader("X-APP-TOKEN")
		zap.L().Info("AuthHAMCHandler start")
		if xAppToken == "" {
			_ = c.Error(errors.ErrAppUnauthorized)
			c.Abort()
			return
		}
		application, err := applicationCache.Get(xAppToken)
		if err != nil {
			_ = c.Error(errors.ErrAppExpired)
			c.Abort()
			return
		}
		current, err := applicationService.GetApplicationForAuthentication(c.Request.Context(), application.Id)
		if err != nil {
			_ = c.Error(errors.WrapSystemError(err))
			c.Abort()
			return
		}
		if current.Id == 0 || !current.State || current.AppKey != application.AppKey || subtle.ConstantTimeCompare([]byte(current.AppSecret), []byte(application.AppSecret)) != 1 {
			_ = c.Error(errors.ErrAppUnauthorized)
			c.Abort()
			return
		}
		conf := token.Config{
			Issuer:                 strconv.Itoa(current.Id),
			SecretKey:              current.AppSecret,
			AccessTokenExpiration:  current.Expiration,
			RefreshTokenExpiration: current.Expiration,
		}
		_, err = hmacTokenGenerator.ParseToken(xAppToken, conf)
		if err != nil {
			_ = c.Error(errors.ErrAppTokenInvalid)
			c.Abort()
			return
		}
		c.Set("application", current)
		c.Next()
		zap.L().Info("AuthHAMCHandler end")

	}
}
