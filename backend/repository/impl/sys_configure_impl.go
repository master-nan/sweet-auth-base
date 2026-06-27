/**
 * @Author: Nan
 * @Date: 2024/6/3 下午2:52
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"gorm.io/gorm"
)

type SysConfigureRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysConfigure]
}

func NewSysConfigureRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysConfigureRepositoryImpl {
	return &SysConfigureRepositoryImpl{PrimaryDB.DB, NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysConfigure{})}
}

func (c *SysConfigureRepositoryImpl) GetSysConfigure() (model.SysConfigure, error) {
	var data model.SysConfigure
	err := c.db.First(&data).Error
	return data, err
}
