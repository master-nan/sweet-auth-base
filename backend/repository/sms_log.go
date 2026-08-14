/**
 * @Author: Nan
 * @Date: 2025/2/7 21:55
 */

package repository

import "backend/model"

type SmsLogRepository interface {
	BasicRepository[model.SmsLog]
}
