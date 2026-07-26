/**
 * @Author: Nan
 * @Date: 2024/5/25 下午4:22
 */

package middleware

import (
	"backend/dto/response"
	error2 "backend/internal/errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ResponseHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			requestErr := c.Errors[0].Err
			clientErr, classified := error2.ToClientError(requestErr)
			category := error2.CategoryOf(requestErr)
			if !classified ||
				category == response.ErrorCategoryDatabase ||
				category == response.ErrorCategorySystem {
				zap.L().Error(
					"request failed",
					zap.Error(requestErr),
					zap.String("request_id", RequestID(c)),
					zap.String("trace_id", TraceID(c)),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("error_category", string(category)),
				)
			}
			c.JSON(clientErr.StatusCode, clientErr)
			return
		}
		if resp, exists := c.Get("response"); exists {
			c.JSON(http.StatusOK, resp)
		}
	}
}
