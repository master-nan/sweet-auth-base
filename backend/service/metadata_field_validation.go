package service

import (
	"backend/enum"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	"backend/model"
	"fmt"
	"strconv"
	"strings"
)

const (
	maxMetadataFieldLength = 65535
	maxMetadataTagLength   = 128
)

// validateMetadataFieldDefinition is the single server-side validation
// boundary shared by field create and update operations.
func validateMetadataFieldDefinition(field *model.SysTableField, sequence int) error {
	if field == nil {
		return myerrors.ErrParamInvalid
	}
	if !validStorageType(field.FieldType) {
		return myerrors.NewBadRequestError("字段存储类型不合法")
	}
	if !validUIComponent(field.InputType) {
		return myerrors.NewBadRequestError("字段组件类型不合法")
	}
	if sequence <= 0 || sequence > 255 {
		return myerrors.NewBadRequestError("字段顺序必须在1到255之间")
	}
	if field.FieldLength < 0 || field.FieldLength > maxMetadataFieldLength ||
		field.FieldDecimalLength < 0 || field.FieldDecimalLength > field.FieldLength {
		return myerrors.NewBadRequestError("字段长度或小数位数不合法")
	}
	if len(strings.TrimSpace(stringValue(field.Tag))) > maxMetadataTagLength {
		return myerrors.NewBadRequestError(fmt.Sprintf("字段标签长度不能超过%d", maxMetadataTagLength))
	}

	category := field.FieldCategory
	if category == "" {
		category = enum.NormalField
		field.FieldCategory = category
	}
	expression := strings.TrimSpace(stringValue(field.Expression))
	switch category {
	case enum.NormalField:
		if expression != "" {
			return myerrors.NewBadRequestError("普通字段不允许配置表达式")
		}
	case enum.VirtualField, enum.CalculatedField:
		if !isStructuredRelationExpression(expression) {
			return myerrors.NewBadRequestError("虚拟或计算字段仅允许受控rel:table.field表达式")
		}
	default:
		return myerrors.NewBadRequestError("字段类别不合法")
	}

	if security.IsSensitiveFieldName(field.FieldCode) || security.IsManagedMetadataField(field.FieldCode) {
		if field.IsListShow || field.IsInsertShow || field.IsUpdateShow ||
			field.IsQuickSearch || field.IsAdvancedSearch {
			return myerrors.NewBadRequestError("受保护字段不能用于列表、写入或查询元数据")
		}
	}
	if field.IsQuickSearch || field.IsAdvancedSearch {
		if category != enum.NormalField || field.IsPrimaryKey || expression != "" ||
			field.InputType == enum.FilePickerInputType || field.InputType == enum.RichTextInputType {
			return myerrors.NewBadRequestError("该字段不能注册为查询字段")
		}
	}
	if err := validateMetadataDefaultValue(field.FieldType, stringValue(field.DefaultValue)); err != nil {
		return err
	}
	return nil
}

func isStructuredRelationExpression(expression string) bool {
	const prefix = "rel:"
	if !strings.HasPrefix(expression, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(expression, prefix), ".")
	return len(parts) == 2 && validMetadataIdentifier(parts[0]) && validMetadataIdentifier(parts[1])
}

func validMetadataIdentifier(value string) bool {
	normalized, err := normalizeDBIdentifier("元数据标识", value)
	return err == nil && normalized == value
}

func validStorageType(value enum.SysTableFieldType) bool {
	return value >= enum.BigIntFieldType && value <= enum.IntFieldType
}

func validUIComponent(value enum.SysTableFieldInputType) bool {
	return value >= enum.InputType && value <= enum.RichTextInputType
}

func validateMetadataDefaultValue(fieldType enum.SysTableFieldType, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch fieldType {
	case enum.BigIntFieldType, enum.IntFieldType, enum.TinyintFieldType:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return myerrors.NewBadRequestError("整数默认值不合法")
		}
	case enum.FloatFieldType:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return myerrors.NewBadRequestError("数值默认值不合法")
		}
	case enum.BooleanFieldType:
		if value != "true" && value != "false" && value != "0" && value != "1" {
			return myerrors.NewBadRequestError("布尔默认值不合法")
		}
	}
	return nil
}
