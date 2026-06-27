/**
 * @Author: Nan
 * @Date: 2023/3/13 22:26
 */

package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type CorsOptions struct {
	AllowedOrigins   []string
	AllowCredentials bool
}

func CorsHandler(options ...CorsOptions) gin.HandlerFunc {
	opts := CorsOptions{AllowedOrigins: []string{"*"}}
	if len(options) > 0 {
		opts = options[0]
	}
	allowedOrigins := normalizeAllowedOrigins(opts.AllowedOrigins)
	return func(c *gin.Context) {
		zap.L().Info("CorsHandler start")
		method := c.Request.Method
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		allowOrigin, wildcard := resolveAllowedOrigin(origin, allowedOrigins)
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			if !wildcard {
				c.Header("Vary", "Origin")
			}
		}
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		if requestedHeaders := strings.TrimSpace(c.GetHeader("Access-Control-Request-Headers")); requestedHeaders != "" {
			c.Header("Access-Control-Allow-Headers", requestedHeaders)
		} else {
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		}
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		if opts.AllowCredentials && !wildcard && allowOrigin != "" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 处理请求
		c.Next()
		zap.L().Info("CorsHandler end")
	}
}

func normalizeAllowedOrigins(origins []string) map[string]bool {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = true
		}
	}
	if len(allowed) == 0 {
		allowed["*"] = true
	}
	return allowed
}

func resolveAllowedOrigin(origin string, allowed map[string]bool) (string, bool) {
	if allowed["*"] {
		return "*", true
	}
	if origin != "" && allowed[origin] {
		return origin, false
	}
	return "", false
}
