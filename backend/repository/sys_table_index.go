/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:55
 */

package repository

import (
	"backend/model"
)

type SysTableIndexRepository interface {
	BasicRepository[model.SysTableIndex]
	GetTableIndexesByTableId(int) ([]model.SysTableIndex, error)
}
