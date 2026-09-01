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
	db := s.db.WithContext(ctx)
	err := db.
		Where("table_id = ?", id).
		Find(&indexes).Error
	if err != nil || len(indexes) == 0 {
		return indexes, err
	}

	indexPositions := make(map[int]int, len(indexes))
	indexIDs := make([]int, 0, len(indexes))
	for position := range indexes {
		indexPositions[indexes[position].Id] = position
		indexIDs = append(indexIDs, indexes[position].Id)
	}
	var links []model.SysTableIndexField
	if err := db.Where("index_id IN ?", indexIDs).
		Order("index_id ASC, sequence ASC, field_id ASC").
		Find(&links).Error; err != nil {
		return nil, err
	}
	fieldIDs := make([]int, 0, len(links))
	seenFields := make(map[int]struct{}, len(links))
	for _, link := range links {
		if _, exists := seenFields[link.FieldId]; exists {
			continue
		}
		seenFields[link.FieldId] = struct{}{}
		fieldIDs = append(fieldIDs, link.FieldId)
	}
	var fields []model.SysTableField
	if len(fieldIDs) > 0 {
		if err := db.Where("id IN ?", fieldIDs).Find(&fields).Error; err != nil {
			return nil, err
		}
	}
	fieldsByID := make(map[int]model.SysTableField, len(fields))
	for _, field := range fields {
		fieldsByID[field.Id] = field
	}
	for _, link := range links {
		position, indexExists := indexPositions[link.IndexId]
		field, fieldExists := fieldsByID[link.FieldId]
		if !indexExists || !fieldExists {
			continue
		}
		indexes[position].IndexFields = append(indexes[position].IndexFields, field)
	}
	return indexes, nil
}
