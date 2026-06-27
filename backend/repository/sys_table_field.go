/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:27
 */

package repository

import (
	"backend/model"
)

type SysTableFieldRepository interface {
	BasicRepository[model.SysTableField]
	GetTableFieldsByTableId(int) ([]model.SysTableField, error)
}
