/**
 * @Author: Nan
 * @Date: 2024/11/13 18:00
 */

package middleware

import (
	"backend/dto/response"
	"backend/internal/errors"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CustomRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				zap.L().Error(
					"Recovered from panic",
					zap.Any("errors", r),
					zap.String("request_id", RequestID(c)),
					zap.String("trace_id", TraceID(c)),
					zap.String("path", c.FullPath()),
					zap.String("method", c.Request.Method),
					zap.String("stack", string(debug.Stack())),
				)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(errors.ErrInternalServer.(*response.AdminError).StatusCode, errors.ErrInternalServer)
					return
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}
