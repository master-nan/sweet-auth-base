/**
 * @Author: Nan
 * @Date: 2025/2/7 21:55
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

type SmsLogRepository interface {
	BasicRepository[model.SmsLog]
	GetSmsLogList(*request.Basic) (response.ListResult[model.SmsLog], error)
}
