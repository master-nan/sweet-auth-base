/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:27
 */

package repository

import (
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type SysTableFieldRepository interface {
	BasicRepository[model.SysTableField]
	GetTableFieldsByTableId(context.Context, int) ([]model.SysTableField, error)
	FindMetadataSecurityField(*gorm.DB, int) (model.SysTableField, error)
}
