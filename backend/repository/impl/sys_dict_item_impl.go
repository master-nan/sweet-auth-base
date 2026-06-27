/**
 * @Author: Nan
 * @Date: 2024/7/20 下午2:52
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"gorm.io/gorm"
)

type SysDictItemRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysDictItem]
}

func NewSysDictItemRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysDictItemRepositoryImpl {
	return &SysDictItemRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysDictItem{}),
	}
}

func (i *SysDictItemRepositoryImpl) GetSysDictItemsByDictId(id int) ([]model.SysDictItem, error) {
	var items []model.SysDictItem
	err := i.db.Where("dict_id = ?", id).Find(&items).Error
	return items, err
}
