/**
 * @Author: Nan
 * @Date: 2024/5/29 上午11:44
 */

package util

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/security"
	"backend/model"
	"backend/repository"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type invalidQueryValue struct{}

func parseValue(value interface{}, valueType enum.SysTableFieldType) interface{} {
	if value == nil {
		return nil
	}
	switch valueType {
	case enum.BigIntFieldType, enum.IntFieldType, enum.TinyintFieldType:
		if v, ok := parseQueryInt(value); ok {
			return v
		}
		return invalidQueryValue{}
	case enum.FloatFieldType:
		if v, ok := parseQueryFloat(value); ok {
			return v
		}
		return invalidQueryValue{}
	case enum.VarcharFieldType:
		if v, ok := value.(string); ok {
			return v
		}
		return fmt.Sprintf("%v", value)
	case enum.BooleanFieldType:
		if v, ok := parseQueryBool(value); ok {
			return v
		}
		return invalidQueryValue{}
	case enum.TextFieldType:
		if v, ok := value.(string); ok {
			return v
		}
		return fmt.Sprintf("%v", value)
	case enum.DateFieldType:
		if t, ok := parseQueryDate(value); ok {
			return t
		}
		return invalidQueryValue{}
	case enum.DatetimeFieldType:
		if t, ok := parseQueryDateTime(value); ok {
			return t
		}
		return invalidQueryValue{}
	case enum.TimeFieldType:
		if t, ok := parseQueryTime(value); ok {
			return t
		}
		return invalidQueryValue{}
	default:
		return value
	}
}

func parseQueryInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		if int64(int(v)) != v {
			return 0, false
		}
		return int(v), true
	case int32:
		return int(v), true
	case uint:
		if uint64(v) > maxQueryInt() {
			return 0, false
		}
		return int(v), true
	case uint64:
		if v > maxQueryInt() {
			return 0, false
		}
		return int(v), true
	case uint32:
		return int(v), true
	case float64:
		if !isFiniteQueryFloat(v) || math.Trunc(v) != v {
			return 0, false
		}
		return int(v), true
	case float32:
		f := float64(v)
		if !isFiniteQueryFloat(f) || math.Trunc(f) != f {
			return 0, false
		}
		return int(f), true
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return 0, false
		}
		i, err := strconv.Atoi(raw)
		return i, err == nil
	}
	return 0, false
}

func parseQueryFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, isFiniteQueryFloat(v)
	case float32:
		f := float64(v)
		return f, isFiniteQueryFloat(f)
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(raw, 64)
		return f, err == nil && isFiniteQueryFloat(f)
	}
	return 0, false
}

func parseQueryBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case int:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case int64:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case int32:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case float64:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case float32:
		if v == 0 || v == 1 {
			return v == 1, true
		}
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	}
	return false, false
}

func parseQueryDate(value interface{}) (time.Time, bool) {
	if t, ok := value.(time.Time); ok {
		return t, true
	}
	raw, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(raw), model.AppLocation())
	return t, err == nil
}

func parseQueryDateTime(value interface{}) (time.Time, bool) {
	if t, ok := value.(time.Time); ok {
		return t, true
	}
	raw, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.DateTime, "2006-01-02T15:04", "2006-01-02T15:04:05", time.RFC3339, time.RFC3339Nano} {
		if t, err := time.ParseInLocation(layout, raw, model.AppLocation()); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func parseQueryTime(value interface{}) (string, bool) {
	if t, ok := value.(time.Time); ok {
		return t.Format(time.TimeOnly), true
	}
	raw, ok := value.(string)
	if !ok {
		return "", false
	}
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.TimeOnly, "15:04"} {
		if t, err := time.ParseInLocation(layout, raw, model.AppLocation()); err == nil {
			return t.Format(time.TimeOnly), true
		}
	}
	return "", false
}

func isInvalidQueryValue(value interface{}) bool {
	_, ok := value.(invalidQueryValue)
	return ok
}

func isFiniteQueryFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func maxQueryInt() uint64 {
	return uint64(^uint(0) >> 1)
}

func splitListText(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\r'
	})
}

func parseListValue(value interface{}, valueType enum.SysTableFieldType) interface{} {
	if value == nil {
		return nil
	}
	var rawValues []interface{}
	switch v := value.(type) {
	case []interface{}:
		rawValues = v
	case []string:
		rawValues = make([]interface{}, 0, len(v))
		for _, item := range v {
			rawValues = append(rawValues, item)
		}
	case []int:
		rawValues = make([]interface{}, 0, len(v))
		for _, item := range v {
			rawValues = append(rawValues, item)
		}
	case []int64:
		rawValues = make([]interface{}, 0, len(v))
		for _, item := range v {
			rawValues = append(rawValues, item)
		}
	case []float64:
		rawValues = make([]interface{}, 0, len(v))
		for _, item := range v {
			rawValues = append(rawValues, item)
		}
	case string:
		parts := splitListText(v)
		rawValues = make([]interface{}, 0, len(parts))
		for _, item := range parts {
			rawValues = append(rawValues, strings.TrimSpace(item))
		}
	default:
		rawValues = []interface{}{v}
	}

	values := make([]interface{}, 0, len(rawValues))
	for _, item := range rawValues {
		if item == nil {
			continue
		}
		if text, ok := item.(string); ok {
			item = strings.TrimSpace(text)
			if item == "" {
				continue
			}
		}
		parsed := parseValue(item, valueType)
		if isInvalidQueryValue(parsed) {
			return invalidQueryValue{}
		}
		values = append(values, parsed)
	}
	return values
}

func parseRangeValue(value interface{}, valueType enum.SysTableFieldType) (interface{}, interface{}, bool) {
	values := parseListValue(value, valueType)
	if isInvalidQueryValue(values) {
		return nil, nil, false
	}
	if isEmptyListValue(values) {
		return nil, nil, false
	}
	items, ok := values.([]interface{})
	if !ok || len(items) != 2 {
		return nil, nil, false
	}
	return items[0], items[1], true
}

func isEmptyListValue(value interface{}) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		return rv.Len() == 0
	default:
		return false
	}
}

func applyLikeRule(query *gorm.DB, fieldExpr string, value interface{}, valueType enum.SysTableFieldType, negate bool) *gorm.DB {
	values := parseListValue(value, valueType)
	if isInvalidQueryValue(values) {
		return query.Where("1 = 0")
	}
	if isEmptyListValue(values) {
		if negate {
			return query
		}
		return query.Where("1 = 0")
	}
	items, ok := values.([]interface{})
	if !ok {
		return query.Where("1 = 0")
	}
	operator := "LIKE"
	joiner := " OR "
	if negate {
		operator = "NOT LIKE"
		joiner = " AND "
	}
	conditions := make([]string, 0, len(items))
	args := make([]interface{}, 0, len(items))
	textFieldExpr := fmt.Sprintf("CAST(%s AS TEXT)", fieldExpr)
	for _, item := range items {
		conditions = append(conditions, fmt.Sprintf("%s %s ?", textFieldExpr, operator))
		args = append(args, fmt.Sprintf("%%%v%%", item))
	}
	condition := strings.Join(conditions, joiner)
	if len(conditions) > 1 {
		condition = "(" + condition + ")"
	}
	return query.Where(condition, args...)
}

func buildKeywordQuery(db *gorm.DB, keyword string, table model.SysTable) *gorm.DB {
	var conditions []string
	var values []interface{}

	// 对关键字进行转义处理
	keyword = strings.ReplaceAll(keyword, "'", "''")
	keyword = strings.ReplaceAll(keyword, "\"", "\"\"")

	// 遍历所有字段，找出支持快速搜索的字段
	for _, field := range table.TableFields {
		if security.IsSensitiveFieldName(field.FieldCode) {
			continue
		}
		if field.IsQuickSearch {
			fieldExpr := qualifyField(field.FieldCode, table.TableCode)
			// 使用参数化查询，避免SQL注入
			conditions = append(conditions, fmt.Sprintf("CAST(%s AS TEXT) LIKE ?", fieldExpr))
			values = append(values, fmt.Sprintf("%%%s%%", keyword))
		}
	}

	// 如果有可搜索字段，构建 OR 条件
	if len(conditions) > 0 {
		return db.Where(strings.Join(conditions, " OR "), values...)
	}

	return db
}

func ExecuteQuery(db *gorm.DB, basic *request.Basic, table model.SysTable) *gorm.DB {
	if basic == nil {
		basic = &request.Basic{}
	}
	if basic.QuickQuery == nil {
		basic.QuickQuery = &request.QuickQuery{}
	}
	query := buildQuery(db, basic, table)
	if basic.QuickQuery != nil && basic.QuickQuery.Keyword != "" {
		query = buildKeywordQuery(query, basic.QuickQuery.Keyword, table)
	}

	// 应用排序和分页
	query = finalizeQuery(query, basic, table)

	return query
}

func applyRule(query *gorm.DB, rule request.QueryRule, value interface{}, table model.SysTable) *gorm.DB {
	if rule.Field == "" {
		return query
	}
	tableField, ok := findField(table, rule.Field)
	if !ok || security.IsSensitiveFieldName(rule.Field) {
		return query.Where("1 = 0")
	}
	fieldType := tableField.FieldType
	fieldExpr := qualifyField(rule.Field, table.TableCode)
	switch rule.ExpressionType {
	case enum.Gt:
		value = parseValue(value, fieldType)
		if value == nil {
			return query
		}
		if isInvalidQueryValue(value) {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s > ?", fieldExpr), value)
	case enum.Lt:
		value = parseValue(value, fieldType)
		if value == nil {
			return query
		}
		if isInvalidQueryValue(value) {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s < ?", fieldExpr), value)
	case enum.Gte:
		value = parseValue(value, fieldType)
		if value == nil {
			return query
		}
		if isInvalidQueryValue(value) {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s >= ?", fieldExpr), value)
	case enum.Lte:
		value = parseValue(value, fieldType)
		if value == nil {
			return query
		}
		if isInvalidQueryValue(value) {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s <= ?", fieldExpr), value)
	case enum.Eq:
		value = parseValue(value, fieldType)
		if value == nil {
			return query
		}
		if isInvalidQueryValue(value) {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s = ?", fieldExpr), value)
	case enum.Ne:
		value = parseValue(value, fieldType)
		if value == nil {
			return query
		}
		if isInvalidQueryValue(value) {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s != ?", fieldExpr), value)
	case enum.Like:
		if value == nil {
			return query
		}
		return applyLikeRule(query, fieldExpr, value, fieldType, false)
	case enum.NotLike:
		if value == nil {
			return query
		}
		return applyLikeRule(query, fieldExpr, value, fieldType, true)
	case enum.In:
		values := parseListValue(value, fieldType)
		if isInvalidQueryValue(values) {
			return query.Where("1 = 0")
		}
		if isEmptyListValue(values) {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s IN ?", fieldExpr), values)
	case enum.NotIn:
		values := parseListValue(value, fieldType)
		if isInvalidQueryValue(values) {
			return query.Where("1 = 0")
		}
		if isEmptyListValue(values) {
			return query
		}
		return query.Where(fmt.Sprintf("%s NOT IN ?", fieldExpr), values)
	case enum.IsNull:
		return query.Where(fmt.Sprintf("%s IS NULL", fieldExpr))
	case enum.IsNotNull:
		return query.Where(fmt.Sprintf("%s IS NOT NULL", fieldExpr))
	case enum.Between:
		start, end, ok := parseRangeValue(value, fieldType)
		if !ok {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s BETWEEN ? AND ?", fieldExpr), start, end)
	case enum.NotBetween:
		start, end, ok := parseRangeValue(value, fieldType)
		if !ok {
			return query.Where("1 = 0")
		}
		return query.Where(fmt.Sprintf("%s NOT BETWEEN ? AND ?", fieldExpr), start, end)
	default:
		return query
	}
}

func combineSubQuery(query, subQuery *gorm.DB, logic enum.ExpressionLogic) *gorm.DB {
	switch logic {
	case enum.And:
		return query.Where(subQuery)
	case enum.Or:
		return query.Or(subQuery)
	default:
		return query
	}
}

func buildQuery(db *gorm.DB, basic *request.Basic, table model.SysTable) *gorm.DB {
	if basic == nil {
		return db
	}
	query := db
	// 构建查询条件
	for _, exprGroup := range basic.Expressions {
		// 使用子查询处理每个表达式组
		subQuery := db.Session(&gorm.Session{NewDB: true})

		// 处理规则
		for j, rule := range exprGroup.Rules {
			if j == 0 {
				// 第一个规则直接应用
				subQuery = applyRule(subQuery, rule, rule.Value, table)
			} else {
				// 后续规则根据逻辑组合
				ruleQuery := applyRule(db.Session(&gorm.Session{NewDB: true}), rule, rule.Value, table)
				subQuery = combineSubQuery(subQuery, ruleQuery, exprGroup.Logic)
			}
		}
		// 处理嵌套表达式
		for _, nestedExpr := range exprGroup.Nested {
			nestedQuery := buildQuery(db.Session(&gorm.Session{NewDB: true}), &request.Basic{Expressions: []request.ExpressionGroup{nestedExpr}}, table)
			subQuery = combineSubQuery(subQuery, nestedQuery, exprGroup.Logic)
		}
		// 组合到主查询
		//if i == 0 {
		//	query = subQuery
		//} else {
		//query = combineSubQuery(query, subQuery, exprGroup.Logic)
		//}
		query = combineSubQuery(query, subQuery, enum.And)
	}
	if len(basic.Filters) > 0 {
		query = applyFilters(query, basic.Filters, table)
	}
	return query
}

func applyFilters(query *gorm.DB, filters map[string]any, table model.SysTable) *gorm.DB {
	for field, raw := range filters {
		if raw == nil {
			continue
		}
		tableField, ok := findField(table, field)
		if !ok || security.IsSensitiveFieldName(field) {
			return query.Where("1 = 0")
		}
		qualified := qualifyField(field, table.TableCode)
		switch val := raw.(type) {
		case []interface{}:
			values := parseListValue(val, tableField.FieldType)
			if isInvalidQueryValue(values) {
				return query.Where("1 = 0")
			}
			if isEmptyListValue(values) {
				continue
			}
			query = query.Where(qualified+" IN ?", values)
		case []string:
			values := parseListValue(val, tableField.FieldType)
			if isInvalidQueryValue(values) {
				return query.Where("1 = 0")
			}
			if isEmptyListValue(values) {
				continue
			}
			query = query.Where(qualified+" IN ?", values)
		case []int:
			values := parseListValue(val, tableField.FieldType)
			if isInvalidQueryValue(values) {
				return query.Where("1 = 0")
			}
			if isEmptyListValue(values) {
				continue
			}
			query = query.Where(qualified+" IN ?", values)
		case []int64:
			values := parseListValue(val, tableField.FieldType)
			if isInvalidQueryValue(values) {
				return query.Where("1 = 0")
			}
			if isEmptyListValue(values) {
				continue
			}
			query = query.Where(qualified+" IN ?", values)
		case []float64:
			values := parseListValue(val, tableField.FieldType)
			if isInvalidQueryValue(values) {
				return query.Where("1 = 0")
			}
			if isEmptyListValue(values) {
				continue
			}
			query = query.Where(qualified+" IN ?", values)
		default:
			value := parseValue(val, tableField.FieldType)
			if isInvalidQueryValue(value) {
				return query.Where("1 = 0")
			}
			query = query.Where(qualified+" = ?", value)
		}
	}
	return query
}

func finalizeQuery(query *gorm.DB, basic *request.Basic, table model.SysTable) *gorm.DB {
	// 添加排序
	if basic.Order.Field != "" && hasField(table, basic.Order.Field) && !security.IsSensitiveFieldName(basic.Order.Field) {
		order := qualifyField(basic.Order.Field, table.TableCode)
		if !basic.Order.IsAsc {
			order += " desc"
		}
		query = query.Order(order)
	}

	// 设置 Page 和 Num 的默认值
	if basic.Page <= 0 {
		basic.Page = 1 // 默认页码为 1
	}
	if basic.Num <= 0 {
		basic.Num = 10 // 默认每页数量为 10
	}
	// 设置 Num 的上限
	const maxNum = 5000
	if basic.Num > maxNum {
		basic.Num = maxNum
	}
	// 计算偏移量，确保从第一页开始
	offset := (basic.Page - 1) * basic.Num
	query = query.Limit(basic.Num).Offset(offset)
	return query
}

func ApplyDataScope(query *gorm.DB, scope *request.DataScope, table model.SysTable) *gorm.DB {
	if scope == nil || scope.AllowAll {
		return query
	}
	if scope.DenyAll {
		return query.Where("1 = 0")
	}
	for _, condition := range scope.Conditions {
		tableField, ok := findField(table, condition.Field)
		if !ok || security.IsSensitiveFieldName(condition.Field) {
			return query.Where("1 = 0")
		}
		fieldExpr := qualifyField(condition.Field, table.TableCode)
		values := parseListValue(condition.Values, tableField.FieldType)
		if isInvalidQueryValue(values) || isEmptyListValue(values) {
			return query.Where("1 = 0")
		}
		switch normalizeDataScopeMatchType(condition.MatchType) {
		case "eq":
			items, ok := values.([]interface{})
			if !ok || len(items) != 1 {
				return query.Where("1 = 0")
			}
			query = query.Where(fmt.Sprintf("%s = ?", fieldExpr), items[0])
		default:
			query = query.Where(fmt.Sprintf("%s IN ?", fieldExpr), values)
		}
	}
	return query
}

func DataScopeValueAllowed(table model.SysTable, condition request.DataScopeCondition, raw interface{}) bool {
	tableField, ok := findField(table, condition.Field)
	if !ok || security.IsSensitiveFieldName(condition.Field) {
		return false
	}
	parsed := parseValue(raw, tableField.FieldType)
	if isInvalidQueryValue(parsed) || parsed == nil {
		return false
	}
	values := parseListValue(condition.Values, tableField.FieldType)
	if isInvalidQueryValue(values) || isEmptyListValue(values) {
		return false
	}
	items, ok := values.([]interface{})
	if !ok {
		return false
	}
	switch normalizeDataScopeMatchType(condition.MatchType) {
	case "eq":
		return len(items) == 1 && reflect.DeepEqual(parsed, items[0])
	default:
		for _, item := range items {
			if reflect.DeepEqual(parsed, item) {
				return true
			}
		}
		return false
	}
}

func normalizeDataScopeMatchType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "eq" {
		return "eq"
	}
	return "in"
}

// DynamicQuery 动态生成结构体并进行查询
func DynamicQuery(db *gorm.DB, basic *request.Basic, table model.SysTable) (repository.GeneralizationListResult, error) {
	var result repository.GeneralizationListResult
	resultFields := listResultFields(table.TableFields)
	// 创建动态结构体
	modelType := CreateDynamicStruct(resultFields)

	// 构建查询
	query := ExecuteQuery(db.Table(table.TableCode), basic, table)

	// 软删除过滤：排除已删除数据
	if !basic.IncludeDeleted && hasDeleteField(table) {
		query = query.Where(qualifyField("gmt_delete", table.TableCode) + " IS NULL")
	}

	query = ApplyDataScope(query, basic.DataScope, table)

	joinAliasMap := make(map[string]string)
	if len(table.TableRelations) > 0 {
		joinAliasMap = buildRelationJoins(&query, db, table)
	}

	selectParts := make([]string, 0, len(resultFields))
	for _, field := range resultFields {
		fieldCode := field.FieldCode
		if field.FieldCategory == enum.CalculatedField || field.FieldCategory == enum.VirtualField {
			if field.Expression != nil && strings.TrimSpace(*field.Expression) != "" {
				if relTable, relField, ok := parseRelationExpression(*field.Expression); ok {
					if alias, exists := joinAliasMap[relTable]; exists {
						selectParts = append(selectParts, fmt.Sprintf("%s.%s AS %s", QuoteIdentifier(alias), QuoteIdentifier(relField), QuoteIdentifier(fieldCode)))
						continue
					}
				}
				selectParts = append(selectParts, fmt.Sprintf("%s AS %s", *field.Expression, QuoteIdentifier(fieldCode)))
			}
			continue
		}
		selectParts = append(selectParts, fmt.Sprintf("%s AS %s", qualifyField(fieldCode, table.TableCode), QuoteIdentifier(fieldCode)))
	}
	if len(selectParts) > 0 {
		query = query.Select(strings.Join(selectParts, ","))
	}

	// 查询结果
	results := reflect.New(reflect.SliceOf(modelType)).Elem()
	err := query.Find(results.Addr().Interface()).Error
	if err != nil {
		return result, err
	}
	// 总数查询
	var total int64
	err = query.Limit(-1).Offset(-1).Count(&total).Error
	if err != nil {
		return result, err
	}
	// 转换结果为更通用的格式
	records := make([]map[string]interface{}, results.Len())
	for i := 0; i < results.Len(); i++ {
		record := make(map[string]interface{})
		val := results.Index(i)
		for _, field := range resultFields {
			fieldValue := val.FieldByName(toCamelCaseGo(field.FieldCode))
			if fieldValue.IsValid() {
				v := fieldValue.Interface()
				// 时间类型统一格式化为 "2006-01-02 15:04:05"，与 CustomTime 序列化保持一致
				if t, ok := v.(time.Time); ok {
					if t.IsZero() {
						record[field.FieldCode] = ""
					} else {
						record[field.FieldCode] = t.Format(time.DateTime)
					}
				} else {
					record[field.FieldCode] = v
				}
			}
		}
		records[i] = record
	}
	result.Data = records
	result.Total = int(total)
	return result, nil
}

func hasField(table model.SysTable, fieldCode string) bool {
	_, ok := findField(table, fieldCode)
	return ok
}

func findField(table model.SysTable, fieldCode string) (model.SysTableField, bool) {
	for _, field := range table.TableFields {
		if field.FieldCode == fieldCode {
			return field, true
		}
	}
	return model.SysTableField{}, false
}

func listResultFields(fields []model.SysTableField) []model.SysTableField {
	result := make([]model.SysTableField, 0, len(fields))
	for _, field := range fields {
		if field.IsListShow && !security.IsSensitiveFieldName(field.FieldCode) {
			result = append(result, field)
		}
	}
	result = prependPrimaryKeyField(result, fields)
	if len(result) > 0 {
		return result
	}
	for _, field := range fields {
		if field.IsPrimaryKey || field.FieldCode == "id" {
			return []model.SysTableField{field}
		}
	}
	return []model.SysTableField{}
}

func prependPrimaryKeyField(result []model.SysTableField, fields []model.SysTableField) []model.SysTableField {
	for _, field := range result {
		if field.IsPrimaryKey || field.FieldCode == "id" {
			return result
		}
	}
	for _, field := range fields {
		if field.IsPrimaryKey || field.FieldCode == "id" {
			if security.IsSensitiveFieldName(field.FieldCode) {
				return result
			}
			withID := make([]model.SysTableField, 0, len(result)+1)
			withID = append(withID, field)
			withID = append(withID, result...)
			return withID
		}
	}
	return result
}

func parseRelationExpression(expression string) (tableCode string, fieldCode string, ok bool) {
	value := strings.TrimSpace(expression)
	if !strings.HasPrefix(value, "rel:") {
		return "", "", false
	}
	value = strings.TrimPrefix(value, "rel:")
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func buildRelationJoins(query **gorm.DB, db *gorm.DB, table model.SysTable) map[string]string {
	aliasMap := make(map[string]string)
	if *query == nil {
		return aliasMap
	}
	if len(table.TableRelations) == 0 {
		return aliasMap
	}
	relatedIds := make([]int, 0, len(table.TableRelations))
	for _, rel := range table.TableRelations {
		if rel.RelationType == enum.ManyToMany || rel.RelationType == enum.OneToMany {
			continue
		}
		relatedIds = append(relatedIds, rel.RelatedTableId)
	}
	if len(relatedIds) == 0 {
		return aliasMap
	}

	codeMap, err := fetchTableCodeMap(db, relatedIds)
	if err != nil {
		return aliasMap
	}
	aliasIndex := 1
	for _, rel := range table.TableRelations {
		if rel.RelationType == enum.ManyToMany || rel.RelationType == enum.OneToMany {
			continue
		}
		relatedCode := codeMap[rel.RelatedTableId]
		if relatedCode == "" {
			continue
		}
		alias, exists := aliasMap[relatedCode]
		if !exists {
			alias = fmt.Sprintf("r%d", aliasIndex)
			aliasIndex++
			aliasMap[relatedCode] = alias
			joinExpr := fmt.Sprintf("LEFT JOIN %s AS %s ON %s = %s.%s", QuoteIdentifier(relatedCode), QuoteIdentifier(alias), qualifyField(rel.ReferenceKey, table.TableCode), QuoteIdentifier(alias), QuoteIdentifier(rel.ForeignKey))
			*query = (*query).Joins(joinExpr)
		}
	}
	return aliasMap
}

func fetchTableCodeMap(db *gorm.DB, tableIds []int) (map[int]string, error) {
	result := make(map[int]string)
	if len(tableIds) == 0 {
		return result, nil
	}
	var rows []struct {
		Id        int    `gorm:"column:id"`
		TableCode string `gorm:"column:table_code"`
	}
	err := db.Table("sys_table").Select("id, table_code").Where("id IN ?", tableIds).Find(&rows).Error
	if err != nil {
		return result, err
	}
	for _, row := range rows {
		result[row.Id] = row.TableCode
	}
	return result, nil
}

func qualifyField(field string, baseTable string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		for i := range parts {
			parts[i] = QuoteIdentifier(parts[i])
		}
		return strings.Join(parts, ".")
	}
	if baseTable == "" {
		return QuoteIdentifier(field)
	}
	return fmt.Sprintf("%s.%s", QuoteIdentifier(baseTable), QuoteIdentifier(field))
}

func QuoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(identifier), `"`, `""`) + `"`
}

// hasDeleteField 检查表是否包含软删除字段
func hasDeleteField(table model.SysTable) bool {
	for _, f := range table.TableFields {
		if f.FieldCode == "gmt_delete" {
			return true
		}
	}
	return false
}

// CreateDynamicStruct 根据表元数据创建动态结构体
func CreateDynamicStruct(fields []model.SysTableField) reflect.Type {
	var fieldsList []reflect.StructField
	for _, field := range fields {
		fieldType := GetFieldType(field.FieldType)
		fieldTag := BuildTag(field)
		if field.Tag != nil {
			fieldTag = *field.Tag
		}
		fieldsList = append(fieldsList, reflect.StructField{
			Name: toCamelCaseGo(field.FieldCode),
			Type: fieldType,
			Tag:  reflect.StructTag(fieldTag),
		})
	}
	return reflect.StructOf(fieldsList)
}

// GetFieldType 获取对应类型
func GetFieldType(fieldType enum.SysTableFieldType) reflect.Type {
	switch fieldType {
	case enum.BigIntFieldType, enum.IntFieldType:
		return reflect.TypeOf(0) // 或 reflect.TypeOf(int64(0)) 根据需要选择
	case enum.FloatFieldType:
		return reflect.TypeOf(0.0) // 使用 float64 是 Go 中最常用的浮点类型
	case enum.VarcharFieldType, enum.TextFieldType:
		return reflect.TypeOf("") // 字符串类型
	case enum.BooleanFieldType:
		return reflect.TypeOf(false)
	case enum.DateFieldType, enum.DatetimeFieldType:
		return reflect.TypeOf(time.Time{})
	case enum.TimeFieldType:
		return reflect.TypeOf("")
	case enum.TinyintFieldType:
		return reflect.TypeOf(int8(0)) // tinyint 类型
	case enum.JsonFieldType:
		return reflect.TypeOf(map[string]interface{}{}) // JSON 类型
	default:
		return reflect.TypeOf(nil) // 未知类型返回 nil 类型，可能需要处理错误
	}
}

// BuildTag 构建结构体tag
func BuildTag(field model.SysTableField) string {
	gormParts := []string{
		fmt.Sprintf(`column:%s`, field.FieldCode),
		fmt.Sprintf(`type:%s`, getSQLType(field.FieldType, field.FieldLength, field.FieldDecimalLength)),
	}
	if field.FieldLength > 0 {
		gormParts = append(gormParts, fmt.Sprintf(`size:%d`, field.FieldLength))
	}
	if field.DefaultValue != nil && *field.DefaultValue != "" {
		str := getDefaultValue(*field.DefaultValue, field.FieldType)
		gormParts = append(gormParts, str)
	}
	if field.IsPrimaryKey {
		gormParts = append(gormParts, `primaryKey:true`)
	}
	if !field.IsNull {
		gormParts = append(gormParts, `notNull:true`)
	}
	//if field.IsIndex {
	//	gormParts = append(gormParts, `index:true`)
	//}
	gormParts = append(gormParts, fmt.Sprintf(`comment:'%s'`, field.FieldName))

	// JSON 标签
	jsonPart := fmt.Sprintf(`json:"%s"`, toCamelCaseJson(field.FieldCode))

	// Binding 标签，如果字段定义了 Binding 规则，使用该规则
	bindingPart := ""
	if field.Binding != "" {
		bindingPart = fmt.Sprintf(`binding:"%s"`, field.Binding)
	}
	// 组合 GORM, JSON 和 Binding 标签
	fullTag := fmt.Sprintf(`gorm:"%s" %s %s`, strings.Join(gormParts, ";"), jsonPart, bindingPart)
	return fullTag
}

func getDefaultValue(defaultValue string, fieldType enum.SysTableFieldType) string {
	switch fieldType {
	case enum.BigIntFieldType, enum.TinyintFieldType, enum.IntFieldType:
		d, _ := strconv.Atoi(defaultValue)
		return fmt.Sprintf(`default:%d`, d)
	case enum.FloatFieldType:
		f, _ := strconv.ParseFloat(defaultValue, 64)
		return fmt.Sprintf(`default:%f`, f)
	case enum.BooleanFieldType:
		return fmt.Sprintf(`default:%v`, defaultValue)
	case enum.VarcharFieldType, enum.TextFieldType:
		return fmt.Sprintf(`default:%s`, defaultValue)
	default:
		return fmt.Sprintf(`default:%v`, defaultValue)
	}
}

// getSQLType 返回类型和长度
func getSQLType(fieldType enum.SysTableFieldType, length int, decimal int) string {
	switch fieldType {
	case enum.BigIntFieldType:
		return "bigint"
	case enum.IntFieldType:
		return "integer"
	case enum.FloatFieldType:
		if length > 0 && decimal > 0 {
			return fmt.Sprintf("numeric(%d,%d)", length, decimal)
		}
		return "numeric"
	case enum.VarcharFieldType:
		if length <= 0 {
			length = 255
		}
		return fmt.Sprintf("varchar(%d)", length)
	case enum.TextFieldType:
		return "text"
	case enum.BooleanFieldType:
		return "boolean"
	case enum.DateFieldType:
		return "date"
	case enum.DatetimeFieldType:
		return "timestamp"
	case enum.TimeFieldType:
		return "time"
	case enum.TinyintFieldType:
		return "smallint"
	case enum.JsonFieldType:
		return "jsonb"
	default:
		return "text"
	}
}

func GetTableName(db *gorm.DB, tableCode string) string {
	tableName := db.NamingStrategy.TableName(tableCode)
	return tableName
}

func toCamelCaseGo(input string) string {
	parts := strings.Split(input, "_")
	c := cases.Title(language.English) // 使用英语规则进行标题转换
	for i, part := range parts {
		parts[i] = c.String(part)
	}
	return strings.Join(parts, "")
}

func toCamelCaseJson(input string) string {
	parts := strings.Split(input, "_")
	c := cases.Title(language.English) // 使用英语规则进行标题转换
	for i, part := range parts {
		if i == 0 {
			// 第一个单词首字母小写
			parts[i] = strings.ToLower(part)
		} else {
			// 其余单词首字母大写
			parts[i] = c.String(part)
		}
	}
	return strings.Join(parts, "")
}
