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
	LegacyNumericScale      = 18
)

// CanonicalStorageType keeps legacy persisted values readable while exposing
// the storage semantics used by new runtime consumers.
func CanonicalStorageType(fieldType enum.SysTableFieldType) enum.SysTableFieldType {
	switch fieldType {
	case enum.TinyintFieldType:
		return enum.SmallIntFieldType
	case enum.FloatFieldType:
		return enum.DecimalFieldType
	default:
		return fieldType
	}
}

func StorageTypesCompatible(left, right enum.SysTableFieldType) bool {
	left = CanonicalStorageType(left)
	right = CanonicalStorageType(right)
	if left == right {
		return true
	}
	return isIntegerStorage(left) && isIntegerStorage(right)
}

func isIntegerStorage(fieldType enum.SysTableFieldType) bool {
	switch fieldType {
	case enum.BigIntFieldType, enum.IntFieldType, enum.SmallIntFieldType:
		return true
	default:
		return false
	}
}

// NormalizeDecimal validates an exact base-10 value without converting it to
// float64. Exponent syntax is deliberately excluded from the V1 contract.
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

// NormalizeDecimalValue validates the protocol representation before field
// metadata is available. Field precision/scale is enforced by application
// validation before persistence.
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
