package datapermission

import (
	"encoding/json"
	"regexp"
	"strings"

	myerrors "backend/internal/errors"
)

const (
	DimensionCodeLegalEntity   = "legal_entity"
	DimensionCodeManagementOrg = "management_org"
	DimensionCodeEmployee      = "employee"
)

var dimensionCodePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

// DimensionValues contains only trusted facts returned by a Dimension
// Provider. Permission decisions and executable filtering expressions are
// intentionally absent.
type DimensionValues struct {
	dimensionCode string
	valueType     DataScopeValueType
	bigintValues  []int64
	stringValues  []string
}

func NewDimensionValues(
	dimensionCode string,
	valueType DataScopeValueType,
	values []any,
) (DimensionValues, error) {
	result := DimensionValues{
		dimensionCode: strings.TrimSpace(dimensionCode),
		valueType: DataScopeValueType(
			strings.ToLower(strings.TrimSpace(string(valueType))),
		),
	}
	if !dimensionCodePattern.MatchString(result.dimensionCode) {
		return DimensionValues{}, myerrors.ErrDataPermissionDimensionUnsupported
	}
	if len(values) > DataScopeMaxValuesPerCondition {
		return DimensionValues{}, myerrors.ErrDataScopeValueCountExceeded
	}

	var err error
	switch result.valueType {
	case DataScopeValueTypeBigint:
		result.bigintValues, err = normalizeBigintValues(values)
	case DataScopeValueTypeString:
		result.stringValues, err = normalizeStringValues(values)
	default:
		return DimensionValues{}, myerrors.ErrDataPermissionDimensionTypeMismatch
	}
	if err != nil {
		return DimensionValues{}, myerrors.ErrDataPermissionDimensionTypeMismatch
	}
	return result, nil
}

func (values DimensionValues) Validate() error {
	_, err := NewDimensionValues(values.dimensionCode, values.valueType, values.Values())
	return err
}

func (values DimensionValues) DimensionCode() string {
	return values.dimensionCode
}

func (values DimensionValues) ValueType() DataScopeValueType {
	return values.valueType
}

func (values DimensionValues) Values() []any {
	if values.valueType == DataScopeValueTypeBigint {
		result := make([]any, len(values.bigintValues))
		for index, value := range values.bigintValues {
			result[index] = value
		}
		return result
	}
	result := make([]any, len(values.stringValues))
	for index, value := range values.stringValues {
		result[index] = value
	}
	return result
}

func (values DimensionValues) BigintValues() []int64 {
	return append([]int64(nil), values.bigintValues...)
}

func (values DimensionValues) StringValues() []string {
	return append([]string(nil), values.stringValues...)
}

func (values DimensionValues) MarshalJSON() ([]byte, error) {
	if err := values.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		DimensionCode string             `json:"dimension_code"`
		ValueType     DataScopeValueType `json:"value_type"`
		Values        []any              `json:"values"`
	}{
		DimensionCode: values.dimensionCode,
		ValueType:     values.valueType,
		Values:        values.Values(),
	})
}
