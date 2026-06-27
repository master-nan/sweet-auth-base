/**
 * @Author: Nan
 * @Date: 2024/5/17 上午11:39
 */

package request

import (
	"backend/enum"
)

type TableCreateReq struct {
	TableName        string                   `json:"table_name" binding:"required"`
	TableCode        string                   `json:"table_code" binding:"required"`
	TableType        enum.SysTableType        `json:"table_type" binding:"required"`
	MasterDetailMode enum.SysMasterDetailMode `json:"master_detail_mode"`
	FormOpenMode     enum.SysFormOpenMode     `json:"form_open_mode"`
	DetailOpenMode   enum.SysDetailOpenMode   `json:"detail_open_mode"`
	ParentId         int                      `json:"parent_id"`
	SQL              string                   `json:"sql"`
}

type TableUpdateReq struct {
	Id               int                      `json:"id" binding:"required"`
	TableName        string                   `json:"table_name" binding:"required"`
	TableCode        string                   `json:"table_code"`
	TableType        enum.SysTableType        `json:"table_type"`
	MasterDetailMode enum.SysMasterDetailMode `json:"master_detail_mode"`
	FormOpenMode     enum.SysFormOpenMode     `json:"form_open_mode"`
	DetailOpenMode   enum.SysDetailOpenMode   `json:"detail_open_mode"`
	ParentId         int                      `json:"parent_id"`
	SQL              string                   `json:"sql"`
}

type TablePublishReq struct {
	ParentId int `json:"parent_id"`
}

type TableFieldCreateReq struct {
	TableId            int                         `json:"table_id" binding:"required"`
	FieldName          string                      `json:"field_name" binding:"required"` // 列名
	FieldCode          string                      `json:"field_code" binding:"required"` // 数据库中字段名
	FieldType          enum.SysTableFieldType      `json:"type" binding:"required"`       // 字段类型
	FieldLength        int                         `json:"field_length"`                  // 字段长度
	FieldDecimalLength int                         `json:"field_decimal_length"`          // 小数位数
	InputType          enum.SysTableFieldInputType `json:"input_type" binding:"required"` // 输入类型
	FormSpan           uint8                       `json:"form_span"`                     // 表单占位列数，0为自动
	DetailSpan         uint8                       `json:"detail_span"`                   // 详情占位列数，0为自动
	DefaultValue       string                      `json:"default_value"`                 // 默认值
	DictCode           string                      `json:"dict_code"`                     // 所用字典
	IsPrimaryKey       bool                        `json:"is_primary_key"`                // 是否主键
	IsIndex            bool                        `json:"is_index"`                      // 是否索引
	IsQuickSearch      bool                        `json:"is_quick_search"`               // 是否快捷搜索
	IsAdvancedSearch   bool                        `json:"is_advanced_search"`            // 是否高级搜索
	IsSort             bool                        `json:"is_sort"`                       // 是否可排序
	IsNull             bool                        `json:"is_null"`                       // 是否可空
	IsListShow         bool                        `json:"is_list_show"`                  // 是否列表显示
	IsInsertShow       bool                        `json:"is_insert_show"`                // 是否新增显示
	IsUpdateShow       bool                        `json:"is_update_show"`                // 是否更新显示
	Sequence           int                         `json:"sequence" binding:"required"`   // 排序
	Binding            string                      `json:"binding"`                       // 校验规则
	OriginalFieldId    int                         `json:"original_field_id"`             // 原字段Id
	FieldCategory      enum.SysTableFieldCategory  `json:"field_category"`                // 字段类别
	Expression         string                      `json:"expression"`                    // 计算字段表达式
	LinkageConfig      string                      `json:"linkage_config"`                // 联动配置

}

type TableFieldUpdateReq struct {
	Id                 int                         `json:"id" binding:"required"`
	TableId            int                         `json:"table_id" binding:"required"`
	FieldName          string                      `json:"field_name" binding:"required"` // 列名
	FieldCode          string                      `json:"field_code" binding:"required"` // 数据库中字段名
	FieldType          enum.SysTableFieldType      `json:"type" binding:"required"`       // 字段类型
	FieldLength        int                         `json:"field_length"`                  // 字段长度
	FieldDecimalLength int                         `json:"field_decimal_length"`          // 小数位数
	InputType          enum.SysTableFieldInputType `json:"input_type" binding:"required"` // 输入类型
	FormSpan           uint8                       `json:"form_span"`                     // 表单占位列数，0为自动
	DetailSpan         uint8                       `json:"detail_span"`                   // 详情占位列数，0为自动
	DefaultValue       string                      `json:"default_value"`                 // 默认值
	DictCode           string                      `json:"dict_code"`                     // 所用字典
	IsPrimaryKey       bool                        `json:"is_primary_key"`                // 是否主键
	IsIndex            bool                        `json:"is_index"`                      // 是否索引
	IsQuickSearch      bool                        `json:"is_quick_search"`               // 是否快捷搜索
	IsAdvancedSearch   bool                        `json:"is_advanced_search"`            // 是否高级搜索
	IsSort             bool                        `json:"is_sort"`                       // 是否可排序
	IsNull             bool                        `json:"is_null"`                       // 是否可空
	IsListShow         bool                        `json:"is_list_show"`                  // 是否列表显示
	IsInsertShow       bool                        `json:"is_insert_show"`                // 是否新增显示
	IsUpdateShow       bool                        `json:"is_update_show"`                // 是否更新显示
	Sequence           int                         `json:"sequence"`                      // 排序
	Binding            string                      `json:"binding"`                       // 校验规则
	OriginalFieldId    int                         `json:"original_field_id"`             // 原字段Id
	FieldCategory      enum.SysTableFieldCategory  `json:"field_category"`                // 字段类别
	Expression         string                      `json:"expression"`                    // 计算字段表达式
	LinkageConfig      string                      `json:"linkage_config"`                // 联动配置
}

type TableRelationCreateReq struct {
	TableId        int                       `json:"table_id" binding:"required"`
	RelatedTableId int                       `json:"related_table_id" binding:"required"` // 关联的表的Id
	ReferenceKey   string                    `json:"reference_key" binding:"required"`    // 主表对应字段
	ForeignKey     string                    `json:"foreign_key" binding:"required"`      // 关联表 字段
	RelationType   enum.SysTableRelationType `json:"relation_type" binding:"required"`
	ManyTableCode  string                    `gorm:"size:128;comment:多对多关系中间表" json:"manyTableCode"` // 多对多关系使用到的中间表
}

type TableRelationUpdateReq struct {
	Id             int                       `json:"id" binding:"required"`
	TableId        int                       `json:"table_id" binding:"required"`
	RelatedTableId int                       `json:"related_table_id" binding:"required"` // 关联的表的Id
	ReferenceKey   string                    `json:"reference_key" binding:"required"`    // 主表对应字段
	ForeignKey     string                    `json:"foreign_key" binding:"required"`      // 关联表 字段
	RelationType   enum.SysTableRelationType `json:"relation_type" binding:"required"`
	ManyTableCode  string                    `gorm:"size:128;comment:多对多关系中间表" json:"manyTableCode"` // 多对多关系使用到的中间表
}

type TableIndexFieldReq struct {
	TableId   int    `json:"table_id" binding:"required"`
	FieldId   int    `json:"field_id" binding:"required"`
	FieldCode string `json:"field_code"  binding:"required"`
}

type TableIndexCreateReq struct {
	TableId     int                  `json:"table_id" binding:"required"`
	IndexName   string               `json:"index_name" binding:"required"`
	IsUnique    bool                 `json:"is_unique"`
	IndexFields []TableIndexFieldReq `json:"index_fields" binding:"required,min=1"`
}

type TableIndexUpdateReq struct {
	Id          int                  `json:"id" binding:"required"`
	TableId     int                  `json:"table_id" binding:"required"`
	IndexName   string               `json:"index_name" binding:"required"`
	IsUnique    bool                 `json:"is_unique"`
	IndexFields []TableIndexFieldReq `json:"index_fields" binding:"required"`
}
