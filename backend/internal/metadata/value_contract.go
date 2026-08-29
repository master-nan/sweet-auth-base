package metadata

import (
	"backend/enum"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

const (
	SmallIntMin             = -32768
	SmallIntMax             = 32767
	MaxNumericPrecision     = 1000
	DefaultNumericPrecision = 38
)

type StorageDescriptor struct {
	SQLType       string
	LogicalType   enum.SysTableFieldLogicalType
	DisplayFormat enum.SysTableFieldDisplayFormat
	Integer       bool
	Ordered       bool
	TextSearch    bool
	AcceptsLength bool
}

func DescribeStorage(fieldType enum.SysTableFieldType) (StorageDescriptor, bool) {
	switch fieldType {
	case enum.BigIntFieldType:
		return StorageDescriptor{SQLType: "bigint", LogicalType: enum.LogicalTypeInteger, DisplayFormat: enum.DisplayFormatInteger, Integer: true, Ordered: true}, true
	case enum.IntFieldType:
		return StorageDescriptor{SQLType: "integer", LogicalType: enum.LogicalTypeInteger, DisplayFormat: enum.DisplayFormatInteger, Integer: true, Ordered: true}, true
	case enum.SmallIntFieldType:
		return StorageDescriptor{SQLType: "smallint", LogicalType: enum.LogicalTypeInteger, DisplayFormat: enum.DisplayFormatInteger, Integer: true, Ordered: true}, true
	case enum.DecimalFieldType:
		return StorageDescriptor{SQLType: "numeric", LogicalType: enum.LogicalTypeDecimal, DisplayFormat: enum.DisplayFormatDecimal, Ordered: true, AcceptsLength: true}, true
	case enum.VarcharFieldType:
		return StorageDescriptor{SQLType: "varchar", LogicalType: enum.LogicalTypePlain, DisplayFormat: enum.DisplayFormatPlain, TextSearch: true, AcceptsLength: true}, true
	case enum.TextFieldType:
		return StorageDescriptor{SQLType: "text", LogicalType: enum.LogicalTypePlain, DisplayFormat: enum.DisplayFormatPlain, TextSearch: true}, true
	case enum.BooleanFieldType:
		return StorageDescriptor{SQLType: "boolean", LogicalType: enum.LogicalTypeBoolean, DisplayFormat: enum.DisplayFormatPlain}, true
	case enum.DateFieldType:
		return StorageDescriptor{SQLType: "date", LogicalType: enum.LogicalTypeDate, DisplayFormat: enum.DisplayFormatDate, Ordered: true}, true
	case enum.DatetimeFieldType:
		return StorageDescriptor{SQLType: "timestamptz", LogicalType: enum.LogicalTypeDateTime, DisplayFormat: enum.DisplayFormatDateTime, Ordered: true}, true
	case enum.TimeFieldType:
		return StorageDescriptor{SQLType: "time", LogicalType: enum.LogicalTypePlain, DisplayFormat: enum.DisplayFormatPlain, Ordered: true}, true
	case enum.JsonFieldType:
		return StorageDescriptor{SQLType: "jsonb", LogicalType: enum.LogicalTypePlain, DisplayFormat: enum.DisplayFormatPlain}, true
	default:
		return StorageDescriptor{}, false
	}
}

func StorageTypesCompatible(left, right enum.SysTableFieldType) bool {
	if left == right {
		_, valid := DescribeStorage(left)
		return valid
	}
	leftDescriptor, leftValid := DescribeStorage(left)
	rightDescriptor, rightValid := DescribeStorage(right)
	return leftValid && rightValid && leftDescriptor.Integer && rightDescriptor.Integer
}

// NormalizeDecimal 校验十进制精确值且不经过float64；协议不接受指数形式。
func NormalizeDecimal(value any, precision, scale int) (string, error) {
	if precision <= 0 || precision > MaxNumericPrecision || scale < 0 || scale > precision {
		return "", fmt.Errorf("invalid numeric precision or scale")
	}
	normalized, integerDigits, fractionDigits, err := normalizeDecimalSyntax(value)
	if err != nil {
		return "", err
	}
	if integerDigits > precision-scale || fractionDigits > scale {
		return "", fmt.Errorf("decimal value exceeds precision or scale")
	}
	return normalized, nil
}

// NormalizeDecimalValue 在尚未取得字段Metadata时校验协议表示；
// 写入前再由Application Service按具体字段校验precision和scale。
func NormalizeDecimalValue(value any) (string, error) {
	normalized, integerDigits, fractionDigits, err := normalizeDecimalSyntax(value)
	if err != nil {
		return "", err
	}
	if integerDigits+fractionDigits > MaxNumericPrecision {
		return "", fmt.Errorf("decimal value is too large")
	}
	return normalized, nil
}

func CompareDecimal(left, right string) (int, error) {
	leftValue, ok := new(big.Rat).SetString(left)
	if !ok {
		return 0, fmt.Errorf("invalid left decimal")
	}
	rightValue, ok := new(big.Rat).SetString(right)
	if !ok {
		return 0, fmt.Errorf("invalid right decimal")
	}
	return leftValue.Cmp(rightValue), nil
}

func normalizeDecimalSyntax(value any) (string, int, int, error) {
	var raw string
	switch typed := value.(type) {
	case string:
		raw = typed
	case json.Number:
		raw = typed.String()
	case int:
		raw = strconv.Itoa(typed)
	case int8:
		raw = strconv.FormatInt(int64(typed), 10)
	case int16:
		raw = strconv.FormatInt(int64(typed), 10)
	case int32:
		raw = strconv.FormatInt(int64(typed), 10)
	case int64:
		raw = strconv.FormatInt(typed, 10)
	case uint:
		raw = strconv.FormatUint(uint64(typed), 10)
	case uint8:
		raw = strconv.FormatUint(uint64(typed), 10)
	case uint16:
		raw = strconv.FormatUint(uint64(typed), 10)
	case uint32:
		raw = strconv.FormatUint(uint64(typed), 10)
	case uint64:
		raw = strconv.FormatUint(typed, 10)
	default:
		return "", 0, 0, fmt.Errorf("decimal value must be a string or integer")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, 0, fmt.Errorf("decimal value is empty")
	}
	sign := ""
	if raw[0] == '-' || raw[0] == '+' {
		if raw[0] == '-' {
			sign = "-"
		}
		raw = raw[1:]
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" {
		return "", 0, 0, fmt.Errorf("invalid decimal value")
	}
	integerPart := parts[0]
	fractionPart := ""
	if len(parts) == 2 {
		fractionPart = parts[1]
		if fractionPart == "" {
			return "", 0, 0, fmt.Errorf("invalid decimal value")
		}
	}
	if !decimalDigits(integerPart) || !decimalDigits(fractionPart) {
		return "", 0, 0, fmt.Errorf("invalid decimal value")
	}
	integerDigits := len(strings.TrimLeft(integerPart, "0"))
	if integerDigits == 0 {
		integerDigits = 1
	}
	integerPart = strings.TrimLeft(integerPart, "0")
	if integerPart == "" {
		integerPart = "0"
	}
	if integerPart == "0" && strings.Trim(fractionPart, "0") == "" {
		sign = ""
	}
	if fractionPart == "" {
		return sign + integerPart, integerDigits, 0, nil
	}
	return sign + integerPart + "." + fractionPart, integerDigits, len(fractionPart), nil
}

func decimalDigits(value string) bool {
	for index := range value {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}
