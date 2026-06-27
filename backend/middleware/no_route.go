/**
 * @Author: Nan
 * @Date: 2024/10/27 22:09
 */

package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		zap.L().Info("NoRouteHandler start")
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"errorCode": http.StatusNotFound,
			"message":   "请求的资源不存在",
		})
		zap.L().Info("NoRouteHandler end")
	}
}
