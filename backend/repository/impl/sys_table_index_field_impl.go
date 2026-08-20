/**
 * @Author: Nan
 * @Date: 2024/7/22 上午10:19
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"gorm.io/gorm"
)

type SysTableIndexFieldRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysTableIndexField]
}

func NewSysTableIndexFieldRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysTableIndexFieldRepositoryImpl {
	return &SysTableIndexFieldRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysTableIndexField{}),
	}
}

func (s *SysTableIndexFieldRepositoryImpl) UpdateSequence(db *gorm.DB, indexID, fieldID int, sequence uint8) error {
	return db.Model(&model.SysTableIndexField{}).
		Where("index_id = ? AND field_id = ?", indexID, fieldID).
		Update("sequence", sequence).Error
}
