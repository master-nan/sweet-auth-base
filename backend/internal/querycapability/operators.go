package querycapability

import (
	"backend/enum"
	"backend/internal/metadata"
	"backend/model"
)

// Supports is the backend truth for persisted Query Scheme authoring. The
// historical query builder separately uses SupportsExecution to reject
// unknown operators while preserving its existing execution contract.
func Supports(fieldType enum.SysTableFieldType, operator enum.ExpressionType, optionBacked bool) bool {
	if optionBacked || fieldType == enum.BooleanFieldType || fieldType == enum.JsonFieldType {
		return equalityOperator(operator)
	}
	switch fieldType {
	case enum.VarcharFieldType, enum.TextFieldType:
		return textOperator(operator)
	case enum.BigIntFieldType, enum.IntFieldType, enum.TinyintFieldType, enum.SmallIntFieldType,
		enum.FloatFieldType, enum.DecimalFieldType,
		enum.DateFieldType, enum.DatetimeFieldType, enum.TimeFieldType:
		return orderedOperator(operator)
	default:
		return equalityOperator(operator)
	}
}

func AllowedMetadataOperators(field metadata.FieldMetadata) []enum.ExpressionType {
	operators := make([]enum.ExpressionType, 0, int(enum.NotBetween))
	for operator := enum.Gt; operator <= enum.NotBetween; operator++ {
		if SupportsMetadata(field, operator) {
			operators = append(operators, operator)
		}
	}
	return operators
}

// SupportsExecution preserves the historical query engine contract. The
// builder accepts every known operator and lets typed value parsing fail
// closed; Scheme authoring uses the stricter SupportsMetadata policy.
func SupportsExecution(operator enum.ExpressionType) bool {
	return operator >= enum.Gt && operator <= enum.NotBetween
}

func SupportsMetadata(field metadata.FieldMetadata, operator enum.ExpressionType) bool {
	optionBacked := field.DictionaryCode != nil || field.LinkageConfig != nil || field.Relation != nil || field.RelationExpression != ""
	return Supports(field.StorageType, operator, optionBacked)
}

func SupportsTableField(field model.SysTableField, operator enum.ExpressionType) bool {
	optionBacked := field.DictCode != nil || field.LinkageConfig != nil || field.Expression != nil
	return Supports(field.FieldType, operator, optionBacked)
}

func equalityOperator(operator enum.ExpressionType) bool {
	switch operator {
	case enum.Eq, enum.Ne, enum.In, enum.NotIn, enum.IsNull, enum.IsNotNull:
		return true
	default:
		return false
	}
}

func textOperator(operator enum.ExpressionType) bool {
	return equalityOperator(operator) || operator == enum.Like || operator == enum.NotLike
}

func orderedOperator(operator enum.ExpressionType) bool {
	if equalityOperator(operator) {
		return true
	}
	switch operator {
	case enum.Gt, enum.Lt, enum.Gte, enum.Lte, enum.Between, enum.NotBetween:
		return true
	default:
		return false
	}
}
