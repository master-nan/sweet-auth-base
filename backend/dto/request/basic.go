/**
 * @Author: Nan
 * @Date: 2024/5/17 下午3:38
 */

package request

import (
	"backend/enum"
)

// Basic 请求参数参数
type Basic struct {
	Page           int               `form:"page" json:"page" example:"1"`
	Num            int               `form:"num" json:"num" example:"10"`
	Order          Order             `form:"order" json:"order"`
	TableCode      string            `form:"table_code" json:"table_code" example:"sys_dict"`
	Expressions    []ExpressionGroup `form:"expressions" json:"expressions"`
	QuickQuery     *QuickQuery       `form:"quick_query" json:"quick_query"`
	IncludeDeleted bool              `form:"include_deleted" json:"include_deleted"` // 是否查询删除数据
	Filters        map[string]any    `form:"filters" json:"filters"`                 // 额外过滤条件（联动/级联）
	MenuId         int               `form:"menu_id" json:"menu_id"`                 // 当前菜单ID（用于数据权限）
	DataScope      *DataScope        `json:"-"`                                      // 数据权限范围（由后端注入，前端不传）
}

// DataScope 数据权限范围
type DataScope struct {
	AllowAll   bool
	DenyAll    bool
	Conditions []DataScopeCondition
}

type DataScopeCondition struct {
	DimensionCode string
	Field         string
	MatchType     string
	ValueType     string
	Values        []string
}

// ExpressionGroup 参数请求组
type ExpressionGroup struct {
	Logic  enum.ExpressionLogic `form:"logic" json:"logic"`   // "and" 或 "or"
	Rules  []QueryRule          `form:"rules" json:"rules"`   // 基础查询规则
	Nested []ExpressionGroup    `form:"nested" json:"nested"` // 嵌套的表达式组
}

// QueryRule 查询规则
type QueryRule struct {
	Field          string                 `form:"field" json:"field"`                     // 字段
	ExpressionType enum.ExpressionType    `form:"expression_type" json:"expression_type"` // 比较器类型，如EQ, LT等
	Value          interface{}            `form:"value" json:"value"`                     // 值
	Type           enum.SysTableFieldType `form:"type" json:"type"`                       // 字段类型
}

// Order 排序
type Order struct {
	Field string `form:"field" json:"field"`
	IsAsc bool   `form:"is_asc" json:"is_asc"`
}

// QuickQuery 快速查询参数
type QuickQuery struct {
	Keyword string `form:"keyword" json:"keyword"`
}
