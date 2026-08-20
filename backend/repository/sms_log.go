/**
 * @Author: Nan
 * @Date: 2025/2/7 21:55
 */

package repository

import (
	"backend/model"
	"context"
)

type SmsLogRepository interface {
	BasicRepository[model.SmsLog]
	CreateSmsLogContext(context.Context, *model.SmsLog) error
}
