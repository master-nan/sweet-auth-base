package service

import (
	"backend/enum"
	myerrors "backend/internal/errors"
	platformmetadata "backend/internal/metadata"
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
		return myerrors.NewValidationError("字段存储类型不合法")
	}
	if !validUIComponent(field.InputType) {
		return myerrors.NewValidationError("字段组件类型不合法")
	}
	if sequence <= 0 || sequence > 255 {
		return myerrors.NewValidationError("字段顺序必须在1到255之间")
	}
	if field.FieldLength < 0 || field.FieldLength > maxMetadataFieldLength ||
		field.FieldDecimalLength < 0 || field.FieldDecimalLength > field.FieldLength {
		return myerrors.NewValidationError("字段长度或小数位数不合法")
	}
	if err := validateNumericMetadata(field); err != nil {
		return err
	}
	if logicalType, ok := enum.NormalizeSysTableFieldLogicalType(string(field.LogicalType)); !ok {
		return myerrors.NewValidationError("字段逻辑类型不合法")
	} else {
		field.LogicalType = logicalType
	}
	if format, ok := enum.NormalizeSysTableFieldDisplayFormat(string(field.DisplayFormat)); !ok {
		return myerrors.NewValidationError("字段展示格式不合法")
	} else {
		field.DisplayFormat = format
	}
	if field.ListWidth != nil && *field.ListWidth <= 0 {
		return myerrors.NewValidationError("列表默认宽度必须为正整数")
	}
	if err := validateLogicalAndDisplayCompatibility(field); err != nil {
		return err
	}
	if len(strings.TrimSpace(stringValue(field.Tag))) > maxMetadataTagLength {
		return myerrors.NewValidationError(fmt.Sprintf("字段标签长度不能超过%d", maxMetadataTagLength))
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
			return myerrors.NewValidationError("普通字段不允许配置表达式")
		}
	case enum.VirtualField, enum.CalculatedField:
		if !isStructuredRelationExpression(expression) {
			return myerrors.NewValidationError("虚拟或计算字段仅允许受控rel:table.field表达式")
		}
	default:
		return myerrors.NewValidationError("字段类别不合法")
	}

	if security.IsSensitiveFieldName(field.FieldCode) || security.IsManagedMetadataField(field.FieldCode) {
		if field.IsListShow || field.IsInsertShow || field.IsUpdateShow ||
			field.IsQuickSearch || field.IsAdvancedSearch {
			return myerrors.NewValidationError("受保护字段不能用于列表、写入或查询元数据")
		}
	}
	if field.IsQuickSearch || field.IsAdvancedSearch {
		if category != enum.NormalField || field.IsPrimaryKey || expression != "" ||
			field.InputType == enum.FilePickerInputType || field.InputType == enum.RichTextInputType {
			return myerrors.NewValidationError("该字段不能注册为查询字段")
		}
	}
	if err := validateMetadataDefaultValue(*field, stringValue(field.DefaultValue)); err != nil {
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
	return value >= enum.BigIntFieldType && value <= enum.DecimalFieldType
}

func validUIComponent(value enum.SysTableFieldInputType) bool {
	return value >= enum.InputType && value <= enum.RichTextInputType
}

func validateMetadataDefaultValue(field model.SysTableField, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch platformmetadata.CanonicalStorageType(field.FieldType) {
	case enum.BigIntFieldType, enum.IntFieldType:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return myerrors.NewValidationError("整数默认值不合法")
		}
	case enum.SmallIntFieldType:
		parsed, err := strconv.ParseInt(value, 10, 16)
		if err != nil || parsed < platformmetadata.SmallIntMin || parsed > platformmetadata.SmallIntMax {
			return myerrors.NewValidationError("SmallInt默认值必须在-32768到32767之间")
		}
	case enum.DecimalFieldType:
		precision, scale := field.NumericPrecision, field.NumericScale
		if precision == 0 {
			precision, scale = field.FieldLength, field.FieldDecimalLength
		}
		if _, err := platformmetadata.NormalizeDecimal(value, precision, scale); err != nil {
			return myerrors.NewValidationError("数值默认值不合法")
		}
	case enum.BooleanFieldType:
		if value != "true" && value != "false" && value != "0" && value != "1" {
			return myerrors.NewValidationError("布尔默认值不合法")
		}
	}
	return nil
}

func validateNumericMetadata(field *model.SysTableField) error {
	canonical := platformmetadata.CanonicalStorageType(field.FieldType)
	if canonical != enum.DecimalFieldType {
		if field.NumericPrecision != 0 || field.NumericScale != 0 {
			return myerrors.NewValidationError("只有Decimal字段允许配置precision和scale")
		}
		return nil
	}
	if field.NumericPrecision == 0 {
		field.NumericPrecision = field.FieldLength
		if field.NumericPrecision == 0 && field.FieldType == enum.FloatFieldType {
			field.NumericPrecision = platformmetadata.DefaultNumericPrecision
			field.NumericScale = platformmetadata.LegacyNumericScale
		}
	}
	if field.NumericScale == 0 && field.FieldDecimalLength > 0 {
		field.NumericScale = field.FieldDecimalLength
	}
	if field.NumericPrecision <= 0 || field.NumericPrecision > platformmetadata.MaxNumericPrecision ||
		field.NumericScale < 0 || field.NumericScale > field.NumericPrecision {
		return myerrors.NewValidationError("Decimal precision必须在1到1000之间且scale不能大于precision")
	}
	return nil
}

func validateLogicalAndDisplayCompatibility(field *model.SysTableField) error {
	canonical := platformmetadata.CanonicalStorageType(field.FieldType)
	logical := field.LogicalType
	if logical != "" {
		valid := false
		switch canonical {
		case enum.BigIntFieldType, enum.IntFieldType, enum.SmallIntFieldType:
			valid = logical == enum.LogicalTypeInteger || logical == enum.LogicalTypeEnum || logical == enum.LogicalTypeRelation
		case enum.DecimalFieldType:
			valid = logical == enum.LogicalTypeDecimal || logical == enum.LogicalTypeMoney || logical == enum.LogicalTypePercent
		case enum.BooleanFieldType:
			valid = logical == enum.LogicalTypeBoolean
		case enum.DateFieldType:
			valid = logical == enum.LogicalTypeDate
		case enum.DatetimeFieldType:
			valid = logical == enum.LogicalTypeDateTime
		default:
			valid = logical == enum.LogicalTypePlain || logical == enum.LogicalTypeEnum || logical == enum.LogicalTypeRelation
		}
		if !valid {
			return myerrors.NewValidationError("字段逻辑类型与存储类型不兼容")
		}
	}
	if field.DisplayFormat == enum.DisplayFormatDictionary && field.DictCode == nil {
		return myerrors.NewValidationError("dictionary展示格式需要字典配置")
	}
	if field.DisplayFormat == enum.DisplayFormatRelation && field.LinkageConfig == nil {
		return myerrors.NewValidationError("relation展示格式需要关系配置")
	}
	return nil
}
