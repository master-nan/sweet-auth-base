/**
 * @Author: Nan
 * @Date: 2024/5/25 下午4:22
 */

package middleware

import (
	"backend/dto/response"
	error2 "backend/internal/errors"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

func ResponseHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		zap.L().Info("ResponseHandler start")
		c.Next()
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				var err *response.AdminError
				switch {
				case errors.As(e.Err, &err):
					// 处理自定义API错误
					err.Success = false
					c.JSON(err.StatusCode, err)
				default:
					// 处理未知错误
					c.JSON(error2.ErrInternalServer.(*response.AdminError).StatusCode, error2.NewBadRequestError(e.Err.Error()))
				}
				return
			}
		} else {
			if resp, exists := c.Get("response"); exists {
				c.JSON(http.StatusOK, resp)
			}
		}
		zap.L().Info("ResponseHandler end")
	}
}
