/**
 * @Author: Nan
 * @Date: 2024/7/20 下午3:34
 */

package impl

import (
	"backend/internal/database"
	"backend/model"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SysTableFieldRepositoryImpl struct {
	db *gorm.DB
	*BasicRepositoryImpl[model.SysTableField]
}

func NewSysTableFieldRepositoryImpl(PrimaryDB *database.PrimaryDB) *SysTableFieldRepositoryImpl {
	return &SysTableFieldRepositoryImpl{
		PrimaryDB.DB,
		NewBasicRepositoryImpl(PrimaryDB.DB, &model.SysTableField{}),
	}
}

func (s *SysTableFieldRepositoryImpl) Create(tx *gorm.DB, entity interface{}) error {
	switch fields := entity.(type) {
	case *model.SysTableField:
		return tx.Model(&model.SysTableField{}).Create(sysTableFieldCreateMap(tx, fields)).Error
	case *[]model.SysTableField:
		rows := make([]map[string]interface{}, 0, len(*fields))
		for i := range *fields {
			rows = append(rows, sysTableFieldCreateMap(tx, &(*fields)[i]))
		}
		return tx.Model(&model.SysTableField{}).Create(rows).Error
	case []model.SysTableField:
		rows := make([]map[string]interface{}, 0, len(fields))
		for i := range fields {
			rows = append(rows, sysTableFieldCreateMap(tx, &fields[i]))
		}
		return tx.Model(&model.SysTableField{}).Create(rows).Error
	default:
		return tx.Model(&model.SysTableField{}).Create(entity).Error
	}
}

func (s *SysTableFieldRepositoryImpl) GetTableFieldsByTableId(id int) ([]model.SysTableField, error) {
	var items []model.SysTableField
	err := s.db.Where("table_id = ?", id).Order("sequence").Find(&items).Error
	return items, err
}

func sysTableFieldCreateMap(tx *gorm.DB, field *model.SysTableField) map[string]interface{} {
	now := time.Now()
	gmtCreate := time.Time(field.GmtCreate)
	if gmtCreate.IsZero() {
		gmtCreate = now
	}
	gmtModify := time.Time(field.GmtModify)
	if gmtModify.IsZero() {
		gmtModify = now
	}
	row := map[string]interface{}{
		"id":                   field.Id,
		"gmt_create":           gmtCreate,
		"gmt_modify":           gmtModify,
		"state":                true,
		"table_id":             field.TableId,
		"field_name":           field.FieldName,
		"field_code":           field.FieldCode,
		"field_type":           field.FieldType,
		"field_length":         field.FieldLength,
		"field_decimal_length": field.FieldDecimalLength,
		"input_type":           field.InputType,
		"form_span":            field.FormSpan,
		"detail_span":          field.DetailSpan,
		"default_value":        field.DefaultValue,
		"dict_code":            field.DictCode,
		"is_primary_key":       field.IsPrimaryKey,
		"is_index":             field.IsIndex,
		"is_quick_search":      field.IsQuickSearch,
		"is_advanced_search":   field.IsAdvancedSearch,
		"is_sort":              field.IsSort,
		"is_null":              field.IsNull,
		"is_list_show":         field.IsListShow,
		"is_insert_show":       field.IsInsertShow,
		"is_update_show":       field.IsUpdateShow,
		"sequence":             field.Sequence,
		"original_field_id":    field.OriginalFieldId,
		"binding":              field.Binding,
		"field_category":       field.FieldCategory,
		"expression":           field.Expression,
		"tag":                  field.Tag,
		"linkage_config":       field.LinkageConfig,
	}
	if field.State {
		row["state"] = field.State
	}
	if field.CreateUser != nil {
		row["create_user"] = field.CreateUser
	}
	if field.ModifyUser != nil {
		row["modify_user"] = field.ModifyUser
	}
	ctx, ok := tx.Statement.Context.(*gin.Context)
	if ok {
		user := ctx.MustGet("user").(model.SysUser)
		row["create_user"] = user.Id
	}
	return row
}
