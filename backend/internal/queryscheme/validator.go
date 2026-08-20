package queryscheme

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/metadata"
	"backend/internal/querycapability"
	"backend/model"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Validator struct {
	metadata metadata.RuntimeReader
}

func NewValidator(reader metadata.RuntimeReader) *Validator {
	return &Validator{metadata: reader}
}

func Normalize(payload QuerySchemePayloadV1) QuerySchemePayloadV1 {
	normalized := payload
	normalized.QuickQuery.Keyword = strings.TrimSpace(normalized.QuickQuery.Keyword)
	normalized.Order.Field = strings.TrimSpace(normalized.Order.Field)
	if normalized.Expressions == nil {
		normalized.Expressions = []request.ExpressionGroup{}
	}
	if normalized.Bindings == nil {
		normalized.Bindings = []Binding{}
	}
	normalized.Expressions = normalizeGroups(normalized.Expressions)
	return normalized
}

func normalizeGroups(groups []request.ExpressionGroup) []request.ExpressionGroup {
	result := append([]request.ExpressionGroup(nil), groups...)
	for groupIndex := range result {
		group := &result[groupIndex]
		if group.Rules == nil {
			group.Rules = []request.QueryRule{}
		} else {
			group.Rules = append([]request.QueryRule(nil), group.Rules...)
		}
		for ruleIndex := range group.Rules {
			group.Rules[ruleIndex].Field = strings.TrimSpace(group.Rules[ruleIndex].Field)
		}
		if group.Nested == nil {
			group.Nested = []request.ExpressionGroup{}
		} else {
			group.Nested = normalizeGroups(group.Nested)
		}
	}
	return result
}

func ValidateSchema(payload QuerySchemePayloadV1) ValidationResult {
	issues := make([]ValidationIssue, 0)
	raw, err := json.Marshal(payload)
	if err != nil {
		return invalidIssue("payload cannot be encoded")
	}
	if len(raw) > MaxPayloadBytes {
		return invalidIssue("query payload exceeds 32 KiB")
	}
	if len(payload.Expressions) > MaxTopLevelGroups {
		return invalidIssue("too many top-level expression groups")
	}
	if utf8.RuneCountInString(payload.QuickQuery.Keyword) > MaxKeywordLength {
		return invalidIssue("quick keyword is too long")
	}
	ruleCount := 0
	seenBindings := make(map[string]struct{}, len(payload.Bindings))
	validPointers := make(map[string]request.QueryRule)
	for index, group := range payload.Expressions {
		validateGroupSchema(group, 1, "/expressions/"+strconv.Itoa(index), &ruleCount, validPointers, &issues)
	}
	if ruleCount > MaxRules {
		issues = append(issues, ValidationIssue{Code: IssuePayloadInvalid, Message: "query payload contains more than 50 rules"})
	}
	for _, binding := range payload.Bindings {
		if _, exists := seenBindings[binding.Pointer]; exists {
			issues = append(issues, ValidationIssue{Code: IssueBindingUnavailable, Path: binding.Pointer, Message: "binding target is duplicated"})
			continue
		}
		seenBindings[binding.Pointer] = struct{}{}
		if err := validateBindingParams(binding); err != nil {
			issues = append(issues, ValidationIssue{Code: IssueBindingUnavailable, Path: binding.Pointer, Message: "binding definition is invalid"})
			continue
		}
		basePointer, _, pointerOK := bindingTarget(binding.Pointer)
		if !pointerOK {
			issues = append(issues, ValidationIssue{Code: IssueBindingUnavailable, Path: binding.Pointer, Message: "binding target is invalid"})
			continue
		}
		if _, exists := validPointers[basePointer]; !exists {
			issues = append(issues, ValidationIssue{Code: IssueBindingUnavailable, Path: binding.Pointer, Message: "binding target is invalid"})
		}
	}
	if len(issues) > 0 {
		return ValidationResult{Status: ValidationInvalid, Issues: issues}
	}
	return ValidationResult{Status: ValidationValid, Issues: []ValidationIssue{}}
}

func validateGroupSchema(
	group request.ExpressionGroup,
	depth int,
	path string,
	ruleCount *int,
	pointers map[string]request.QueryRule,
	issues *[]ValidationIssue,
) {
	if depth > MaxSchemaDepth {
		*issues = append(*issues, ValidationIssue{Code: IssuePayloadInvalid, Path: path, Message: "query nesting exceeds schema limit"})
		return
	}
	if group.Logic != enum.And && group.Logic != enum.Or {
		*issues = append(*issues, ValidationIssue{Code: IssuePayloadInvalid, Path: path, Message: "expression logic is invalid"})
	}
	for index, rule := range group.Rules {
		*ruleCount++
		rulePath := path + "/rules/" + strconv.Itoa(index)
		if strings.TrimSpace(rule.Field) == "" || !querycapability.SupportsExecution(rule.ExpressionType) {
			*issues = append(*issues, ValidationIssue{Code: IssuePayloadInvalid, FieldCode: rule.Field, Path: rulePath, Message: "query rule is invalid"})
		}
		if multi, ok := valueSlice(rule.Value); ok && len(multi) > MaxMultiValues {
			*issues = append(*issues, ValidationIssue{Code: IssuePayloadInvalid, FieldCode: rule.Field, Path: rulePath, Message: "query rule contains too many values"})
		}
		pointers[rulePath+"/value"] = rule
	}
	for index, nested := range group.Nested {
		validateGroupSchema(nested, depth+1, path+"/nested/"+strconv.Itoa(index), ruleCount, pointers, issues)
	}
}

func (validator *Validator) ValidateMetadata(
	ctx context.Context,
	config ScopeConfig,
	payload QuerySchemePayloadV1,
) (ValidationResult, error) {
	if validator == nil || validator.metadata == nil {
		return ValidationResult{}, fmt.Errorf("query scheme metadata reader is unavailable")
	}
	table, err := validator.metadata.GetTable(ctx, config.TableCode)
	if err != nil {
		return ValidationResult{}, err
	}
	fields := make(map[string]metadata.FieldMetadata, len(table.Fields))
	for _, field := range table.Fields {
		fields[field.Code] = field
	}
	issues := make([]ValidationIssue, 0)
	rules := make(map[string]request.QueryRule)
	for index, group := range payload.Expressions {
		collectRules(group, "/expressions/"+strconv.Itoa(index), rules)
	}
	for path, rule := range rules {
		field, exists := fields[rule.Field]
		if !exists || field.SystemManaged {
			issues = append(issues, ValidationIssue{Code: IssueFieldUnavailable, FieldCode: rule.Field, Path: path, Message: "查询字段已不可用"})
			continue
		}
		if !field.AdvancedQuery {
			issues = append(issues, ValidationIssue{Code: IssueFieldNotQueryable, FieldCode: rule.Field, Path: path, Message: "字段不允许高级查询"})
			continue
		}
		if rule.Type != 0 && rule.Type != field.StorageType {
			issues = append(issues, ValidationIssue{Code: IssueValueInvalid, FieldCode: rule.Field, Path: path, Message: "字段类型已发生变化"})
			continue
		}
		if !querycapability.SupportsMetadata(field, rule.ExpressionType) {
			issues = append(issues, ValidationIssue{Code: IssueOperatorIncompatible, FieldCode: rule.Field, Path: path, Message: "操作符与字段类型不兼容"})
			continue
		}
		if !validRuleValue(rule, field.StorageType) {
			issues = append(issues, ValidationIssue{Code: IssueValueInvalid, FieldCode: rule.Field, Path: path, Message: "查询值不合法"})
		}
		if field.LinkageConfig != nil && !json.Valid([]byte(*field.LinkageConfig)) {
			issues = append(issues, ValidationIssue{Code: IssueFieldUnavailable, FieldCode: rule.Field, Path: path, Message: "字段关联配置已失效"})
		}
	}
	if payload.Order.Field != "" {
		field, exists := fields[payload.Order.Field]
		if (!exists || !field.Sortable) && !config.AllowsSort(payload.Order.Field) {
			issues = append(issues, ValidationIssue{Code: IssueSortUnavailable, FieldCode: payload.Order.Field, Path: "/order/field", Message: "排序字段已不可用"})
		}
	}
	for _, binding := range payload.Bindings {
		basePointer, elementIndex, pointerOK := bindingTarget(binding.Pointer)
		if !pointerOK {
			issues = append(issues, ValidationIssue{Code: IssueBindingUnavailable, Path: binding.Pointer, Message: "动态条件已不可用"})
			continue
		}
		rule, exists := rules[strings.TrimSuffix(basePointer, "/value")]
		field, fieldExists := fields[rule.Field]
		if !exists || !fieldExists || !config.AllowsBinding(binding.Kind) || !bindingMatchesRule(binding, elementIndex, rule, field) {
			issues = append(issues, ValidationIssue{Code: IssueBindingUnavailable, FieldCode: rule.Field, Path: binding.Pointer, Message: "动态条件已不可用"})
		}
	}
	if len(issues) > 0 {
		return ValidationResult{Status: ValidationDegraded, Issues: issues}, nil
	}
	return ValidationResult{Status: ValidationValid, Issues: []ValidationIssue{}}, nil
}

func collectRules(group request.ExpressionGroup, path string, result map[string]request.QueryRule) {
	for index, rule := range group.Rules {
		result[path+"/rules/"+strconv.Itoa(index)] = rule
	}
	for index, nested := range group.Nested {
		collectRules(nested, path+"/nested/"+strconv.Itoa(index), result)
	}
}

func bindingMatchesRule(binding Binding, elementIndex *int, rule request.QueryRule, field metadata.FieldMetadata) bool {
	dateBinding := binding.Kind == BindingToday || binding.Kind == BindingStartOfWeek || binding.Kind == BindingEndOfWeek ||
		binding.Kind == BindingStartOfMonth || binding.Kind == BindingEndOfMonth
	if dateBinding {
		if field.StorageType != enum.DateFieldType && field.StorageType != enum.DatetimeFieldType {
			return false
		}
		if rule.ExpressionType == enum.Between || rule.ExpressionType == enum.NotBetween {
			return elementIndex != nil && (*elementIndex == 0 || *elementIndex == 1)
		}
		return elementIndex == nil
	}
	if binding.Kind == BindingCurrentUser || binding.Kind == BindingCurrentEmployee {
		switch field.StorageType {
		case enum.BigIntFieldType, enum.IntFieldType, enum.SmallIntFieldType:
			if rule.ExpressionType == enum.Eq || rule.ExpressionType == enum.Ne {
				return elementIndex == nil
			}
			if rule.ExpressionType == enum.In || rule.ExpressionType == enum.NotIn {
				values, ok := valueSlice(rule.Value)
				return ok && elementIndex != nil && *elementIndex >= 0 && *elementIndex < len(values)
			}
		}
	}
	return false
}

func bindingTarget(pointer string) (string, *int, bool) {
	if !strings.HasPrefix(pointer, "/expressions/") {
		return "", nil, false
	}
	if strings.HasSuffix(pointer, "/value") {
		return pointer, nil, true
	}
	lastSlash := strings.LastIndex(pointer, "/")
	if lastSlash < 0 || !strings.HasSuffix(pointer[:lastSlash], "/value") {
		return "", nil, false
	}
	index, err := strconv.Atoi(pointer[lastSlash+1:])
	if err != nil || index < 0 {
		return "", nil, false
	}
	return pointer[:lastSlash], &index, true
}

func validRuleValue(rule request.QueryRule, fieldType enum.SysTableFieldType) bool {
	if rule.ExpressionType == enum.IsNull || rule.ExpressionType == enum.IsNotNull {
		return true
	}
	if rule.ExpressionType == enum.In || rule.ExpressionType == enum.NotIn || rule.ExpressionType == enum.Between || rule.ExpressionType == enum.NotBetween {
		values, ok := valueSlice(rule.Value)
		if !ok || len(values) == 0 || len(values) > MaxMultiValues {
			return false
		}
		if (rule.ExpressionType == enum.Between || rule.ExpressionType == enum.NotBetween) && len(values) != 2 {
			return false
		}
		for _, value := range values {
			if !scalarCompatible(value, fieldType) {
				return false
			}
		}
		return true
	}
	return scalarCompatible(rule.Value, fieldType)
}

func scalarCompatible(value any, fieldType enum.SysTableFieldType) bool {
	if value == nil {
		return false
	}
	switch fieldType {
	case enum.BigIntFieldType, enum.IntFieldType:
		switch typed := value.(type) {
		case int, int32, int64, uint, uint32, uint64:
			return true
		case float64:
			return !math.IsNaN(typed) && !math.IsInf(typed, 0) && math.Trunc(typed) == typed
		case json.Number:
			_, err := typed.Int64()
			return err == nil
		case string:
			_, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			return err == nil
		}
	case enum.SmallIntFieldType:
		text := fmt.Sprintf("%v", value)
		parsed, err := strconv.ParseInt(strings.TrimSpace(text), 10, 16)
		return err == nil && parsed >= metadata.SmallIntMin && parsed <= metadata.SmallIntMax
	case enum.DecimalFieldType:
		_, err := metadata.NormalizeDecimalValue(value)
		return err == nil
	case enum.BooleanFieldType:
		if _, ok := value.(bool); ok {
			return true
		}
		if text, ok := value.(string); ok {
			_, err := strconv.ParseBool(strings.TrimSpace(text))
			return err == nil
		}
	case enum.DateFieldType:
		return validTimeString(value, time.DateOnly)
	case enum.DatetimeFieldType:
		return validTimeString(value, time.DateTime) || validRFC3339(value)
	case enum.TimeFieldType:
		return validTimeString(value, time.TimeOnly)
	case enum.VarcharFieldType, enum.TextFieldType:
		_, ok := value.(string)
		return ok
	default:
		return value != nil
	}
	return false
}

func validTimeString(value any, layout string) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	_, err := time.ParseInLocation(layout, strings.TrimSpace(text), model.AppLocation())
	return err == nil
}

func validRFC3339(value any) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	_, err := time.Parse(time.RFC3339, strings.TrimSpace(text))
	return err == nil
}

func valueSlice(value any) ([]any, bool) {
	switch values := value.(type) {
	case []any:
		return values, true
	case []string:
		result := make([]any, len(values))
		for index := range values {
			result[index] = values[index]
		}
		return result, true
	case []int:
		result := make([]any, len(values))
		for index := range values {
			result[index] = values[index]
		}
		return result, true
	default:
		return nil, false
	}
}

func invalidIssue(message string) ValidationResult {
	return ValidationResult{Status: ValidationInvalid, Issues: []ValidationIssue{{Code: IssuePayloadInvalid, Message: message}}}
}
