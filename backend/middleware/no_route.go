/**
 * @Author: Nan
 * @Date: 2024/10/27 22:09
 */

package middleware

import (
	"backend/dto/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		zap.L().Info("NoRouteHandler start")
		c.AbortWithStatusJSON(http.StatusNotFound, response.AdminError{
			StatusCode:   http.StatusNotFound,
			ErrorCode:    http.StatusNotFound,
			ErrorMessage: "请求的资源不存在",
			Success:      false,
		})
		zap.L().Info("NoRouteHandler end")
	}
}
