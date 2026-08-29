package database

import (
	"backend/internal/utils"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const snowflakeIDCallbackName = "sweet:assign_snowflake_id"

// RegisterSnowflakeIDs 为所有单列 id 主键补齐雪花 ID。
// Service 可以继续显式生成 ID；此处只处理遗漏的零值，防止写入路径退回数据库自增。
func RegisterSnowflakeIDs(db *gorm.DB, sf *utils.Snowflake) error {
	if db == nil || sf == nil {
		return fmt.Errorf("database and snowflake generator are required")
	}
	return db.Callback().Create().Before("gorm:before_create").Register(
		snowflakeIDCallbackName,
		func(tx *gorm.DB) {
			if err := assignSnowflakeIDs(tx, sf); err != nil {
				tx.AddError(err)
			}
		},
	)
}

func assignSnowflakeIDs(tx *gorm.DB, sf *utils.Snowflake) error {
	if tx == nil || tx.Statement == nil || tx.Statement.Schema == nil {
		return nil
	}
	primaryFields := tx.Statement.Schema.PrimaryFields
	if len(primaryFields) != 1 {
		return nil
	}
	primaryField := primaryFields[0]
	if primaryField == nil || !strings.EqualFold(primaryField.DBName, "id") || !integerField(primaryField) {
		return nil
	}
	return assignSnowflakeValue(tx, sf, primaryField, tx.Statement.ReflectValue)
}

func assignSnowflakeValue(tx *gorm.DB, sf *utils.Snowflake, field *schema.Field, value reflect.Value) error {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		for index := 0; index < value.Len(); index++ {
			if err := assignSnowflakeValue(tx, sf, field, value.Index(index)); err != nil {
				return err
			}
		}
		return nil
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	_, zero := field.ValueOf(tx.Statement.Context, value)
	if !zero {
		return nil
	}
	id, err := sf.GenerateUniqueID()
	if err != nil {
		return err
	}
	converted, err := snowflakeFieldValue(field.FieldType, id)
	if err != nil {
		return err
	}
	return field.Set(tx.Statement.Context, value, converted)
}

func integerField(field *schema.Field) bool {
	if field == nil {
		return false
	}
	switch field.FieldType.Kind() {
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		return true
	default:
		return false
	}
}

func snowflakeFieldValue(fieldType reflect.Type, id int64) (interface{}, error) {
	value := reflect.New(fieldType).Elem()
	switch fieldType.Kind() {
	case reflect.Int, reflect.Int64:
		if value.OverflowInt(id) {
			return nil, fmt.Errorf("snowflake ID exceeds %s", fieldType)
		}
		value.SetInt(id)
	case reflect.Uint, reflect.Uint64:
		if id < 0 || value.OverflowUint(uint64(id)) {
			return nil, fmt.Errorf("snowflake ID exceeds %s", fieldType)
		}
		value.SetUint(uint64(id))
	default:
		return nil, fmt.Errorf("unsupported snowflake ID type %s", fieldType)
	}
	return value.Interface(), nil
}
