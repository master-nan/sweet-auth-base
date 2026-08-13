/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:55
 */

package repository

import (
	"backend/model"
	"context"
)

type SysTableIndexRepository interface {
	BasicRepository[model.SysTableIndex]
	GetTableIndexesByTableId(context.Context, int) ([]model.SysTableIndex, error)
}
