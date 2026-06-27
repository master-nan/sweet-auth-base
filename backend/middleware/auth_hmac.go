/**
 * @Author: Nan
 * @Date: 2024/10/24 15:09
 */

package middleware

import (
	"backend/internal/cache"
	"backend/internal/errors"
	"backend/internal/token"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

func AuthHMACHandler(hmacTokenGenerator token.Generator,
	applicationCache *cache.ApplicationCache) gin.HandlerFunc {
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
		conf := token.Config{
			Issuer:                 strconv.Itoa(application.Id),
			SecretKey:              application.AppSecret,
			AccessTokenExpiration:  application.Expiration,
			RefreshTokenExpiration: application.Expiration,
		}
		_, err = hmacTokenGenerator.ParseToken(xAppToken, conf)
		if err != nil {
			_ = c.Error(errors.ErrAppTokenInvalid)
			c.Abort()
			return
		}
		c.Set("application", application)
		c.Next()
		zap.L().Info("AuthHAMCHandler end")

	}
}
