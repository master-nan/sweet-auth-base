package response

import "backend/enum"

type SysTableFieldListRes struct {
	BasicRes
	TableId            int                         `json:"table_id"`
	FieldName          string                      `json:"field_name"`
	FieldCode          string                      `json:"field_code"`
	FieldType          enum.SysTableFieldType      `json:"field_type"`
	FieldLength        int                         `json:"field_length"`
	FieldDecimalLength int                         `json:"field_decimal_length"`
	InputType          enum.SysTableFieldInputType `json:"input_type"`
	FormSpan           uint8                       `json:"form_span"`
	DetailSpan         uint8                       `json:"detail_span"`
	DefaultValue       *string                     `json:"default_value,omitempty"`
	DictCode           *string                     `json:"dict_code"`
	IsPrimaryKey       bool                        `json:"is_primary_key"`
	IsIndex            bool                        `json:"is_index"`
	IsQuickSearch      bool                        `json:"is_quick_search"`
	IsAdvancedSearch   bool                        `json:"is_advanced_search"`
	IsSort             bool                        `json:"is_sort"`
	IsNull             bool                        `json:"is_null"`
	IsListShow         bool                        `json:"is_list_show"`
	IsInsertShow       bool                        `json:"is_insert_show"`
	IsUpdateShow       bool                        `json:"is_update_show"`
	Sequence           uint8                       `json:"sequence"`
	OriginalFieldId    int                         `json:"original_field_id"`
	Binding            string                      `json:"binding"`
	FieldCategory      enum.SysTableFieldCategory  `json:"field_category"`
	Expression         *string                     `json:"expression"`
	Tag                *string                     `json:"tag"`
	LinkageConfig      *string                     `json:"linkage_config"`
}

type SysTableFieldDetailRes struct {
	SysTableFieldListRes
}

type SysTableRelationRes struct {
	BasicRes
	TableId        int                       `json:"table_id"`
	RelatedTableId int                       `json:"related_table_id"`
	ReferenceKey   string                    `json:"reference_key"`
	ForeignKey     string                    `json:"foreign_key"`
	OnDelete       string                    `json:"on_delete"`
	OnUpdate       string                    `json:"on_update"`
	RelationType   enum.SysTableRelationType `json:"relation_type"`
	ManyTableCode  string                    `json:"many_table_code"`
}

type SysTableIndexRes struct {
	BasicRes
	TableId     int                    `json:"table_id"`
	IndexName   string                 `json:"index_name"`
	IsUnique    bool                   `json:"is_unique"`
	IndexFields []SysTableFieldListRes `json:"index_fields"`
}

// SysTableListRes 不携带 SQL 和字段、关系、索引等大型详情配置。
type SysTableListRes struct {
	BasicRes
	TableName        string                   `json:"table_name"`
	TableCode        string                   `json:"table_code"`
	TableType        enum.SysTableType        `json:"table_type"`
	MasterDetailMode enum.SysMasterDetailMode `json:"master_detail_mode"`
	FormOpenMode     enum.SysFormOpenMode     `json:"form_open_mode"`
	DetailOpenMode   enum.SysDetailOpenMode   `json:"detail_open_mode"`
	ParentId         int                      `json:"parent_id"`
}

type SysTableDetailRes struct {
	SysTableListRes
	SQL            string                 `json:"sql"`
	TableFields    []SysTableFieldListRes `json:"table_fields"`
	TableRelations []SysTableRelationRes  `json:"table_relations"`
	TableIndexes   []SysTableIndexRes     `json:"table_indexes"`
}
