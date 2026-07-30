/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:27
 */

package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type SysTableFieldRepository interface {
	BasicRepository[model.SysTableField]
	GetTableFieldsByTableId(int) ([]model.SysTableField, error)
	FindByIdForConfigDB(*gorm.DB, int) (model.SysTableField, error)
}
