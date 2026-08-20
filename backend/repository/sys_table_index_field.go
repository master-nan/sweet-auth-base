/**
 * @Author: Nan
 * @Date: 2024/7/22 上午10:18
 */

package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type SysTableIndexFieldRepository interface {
	BasicRepository[model.SysTableIndexField]
	UpdateSequence(*gorm.DB, int, int, uint8) error
}
