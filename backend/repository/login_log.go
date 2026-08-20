/**
 * @Author: Nan
 * @Date: 2024/6/3 下午4:30
 */

package repository

import (
	"backend/model"
	"context"
)

type LoginLogRepository interface {
	CreateLoginLogContext(context.Context, *model.LoginLog) error
}
