/**
 * @Author: Nan
 * @Date: 2025/2/7 21:54
 */

package repository

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
)

type SmsTemplateRepository interface {
	BasicRepository[model.SmsTemplate]
	GetSmsTemplateList(*request.Basic, model.SysTable) (response.ListResult[model.SmsTemplate], error)
}
