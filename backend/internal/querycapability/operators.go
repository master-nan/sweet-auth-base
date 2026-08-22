package querycapability

import (
	"backend/enum"
	"backend/internal/metadata"
	"backend/model"
)

// Supports 是持久化Query Scheme编辑时的后端Operator安全真值。
// 现有查询执行引擎另由SupportsExecution拒绝未知Operator，并保持其执行合同。
func Supports(fieldType enum.SysTableFieldType, operator enum.ExpressionType, optionBacked bool) bool {
	if optionBacked || fieldType == enum.BooleanFieldType || fieldType == enum.JsonFieldType {
		return equalityOperator(operator)
	}
	descriptor, ok := metadata.DescribeStorage(fieldType)
	if !ok {
		return false
	}
	if descriptor.TextSearch {
		return textOperator(operator)
	}
	if descriptor.Ordered {
		return orderedOperator(operator)
	}
	return equalityOperator(operator)
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

// SupportsExecution 保持现有查询引擎合同：已知Operator可以进入构建流程，
// 字段和值组合不合法时由类型化解析失败关闭；Scheme编辑使用更严格的SupportsMetadata策略。
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
