/**
 * @Author: Nan
 * @Date: 2024/7/20 上午10:26
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"

	"github.com/gin-gonic/gin"
)

type AccessLogRepository interface {
	BasicRepository[model.AccessLog]
	GetAccessLogList(*gin.Context, *request.Basic) (response.ListResult[model.AccessLog], error)
}
