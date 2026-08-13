/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:55
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"context"

	"gorm.io/gorm"
)

type SysTableIndexRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysTableIndex]
}

func NewSysTableIndexRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysTableIndexRepositoryImpl {
	return &SysTableIndexRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysTableIndex{}),
	}
}

func (s *SysTableIndexRepositoryImpl) GetTableIndexesByTableId(ctx context.Context, id int) ([]model.SysTableIndex, error) {
	var indexes []model.SysTableIndex
	err := s.db.WithContext(ctx).Preload("IndexFields").Where("table_id = ?", id).Find(&indexes).Error
	return indexes, err
}
