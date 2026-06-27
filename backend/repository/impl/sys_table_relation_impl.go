/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:50
 */

package impl

import (
	"backend/internal/database"
	"backend/model"

	"gorm.io/gorm"
)

type SysTableRelationRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysTableRelation]
}

func NewSysTableRelationRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysTableRelationRepositoryImpl {
	return &SysTableRelationRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysTableRelation{}),
	}
}

func (s *SysTableRelationRepositoryImpl) GetTableRelationsByTableId(i int) ([]model.SysTableRelation, error) {
	var relations []model.SysTableRelation
	err := s.db.Where("table_id = ?", i).Find(&relations).Error
	return relations, err
}
