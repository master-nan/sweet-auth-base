package impl

import (
	"context"
	"errors"

	"backend/internal/database"
	"backend/internal/datapermission"
	"backend/model"

	"gorm.io/gorm"
)

// DataPermissionMetadataReaderImpl returns only the metadata columns required
// by the runtime adapter. It deliberately uses unscoped reads so the adapter
// can distinguish inactive or soft-deleted fields without exposing DB details.
type DataPermissionMetadataReaderImpl struct {
	db *gorm.DB
}

var _ datapermission.MetadataFieldReader = (*DataPermissionMetadataReaderImpl)(nil)

func NewDataPermissionMetadataReaderImpl(
	primaryDB *database.PrimaryDB,
) *DataPermissionMetadataReaderImpl {
	return &DataPermissionMetadataReaderImpl{db: primaryDB.DB}
}

func (reader *DataPermissionMetadataReaderImpl) FindMetadataTable(
	ctx context.Context,
	tableId int,
) (datapermission.MetadataTableRecord, error) {
	var table model.SysTable
	err := reader.db.WithContext(ctx).Unscoped().Select(
		"id", "state", "gmt_delete",
	).Where("id = ?", tableId).First(&table).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return datapermission.MetadataTableRecord{}, datapermission.ErrMetadataTableRecordNotFound
	}
	if err != nil {
		return datapermission.MetadataTableRecord{}, err
	}
	return datapermission.MetadataTableRecord{
		Id:      table.Id,
		State:   table.State,
		Deleted: table.GmtDelete.Valid,
	}, nil
}

func (reader *DataPermissionMetadataReaderImpl) FindMetadataField(
	ctx context.Context,
	fieldId int,
) (datapermission.MetadataFieldRecord, error) {
	var field model.SysTableField
	err := reader.db.WithContext(ctx).Unscoped().Select(
		"id", "table_id", "state", "gmt_delete", "field_code", "field_type", "input_type",
		"field_category", "expression", "is_primary_key", "is_advanced_search",
	).Where("id = ?", fieldId).First(&field).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return datapermission.MetadataFieldRecord{}, datapermission.ErrMetadataFieldRecordNotFound
	}
	if err != nil {
		return datapermission.MetadataFieldRecord{}, err
	}
	expression := ""
	if field.Expression != nil {
		expression = *field.Expression
	}
	return datapermission.MetadataFieldRecord{
		Id:               field.Id,
		TableId:          field.TableId,
		State:            field.State,
		Deleted:          field.GmtDelete.Valid,
		FieldCode:        field.FieldCode,
		FieldType:        field.FieldType,
		InputType:        field.InputType,
		FieldCategory:    field.FieldCategory,
		Expression:       expression,
		IsPrimaryKey:     field.IsPrimaryKey,
		IsAdvancedSearch: field.IsAdvancedSearch,
	}, nil
}
