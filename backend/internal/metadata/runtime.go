package metadata

import (
	"backend/enum"
	"backend/internal/security"
	"backend/model"
	"context"
	"encoding/json"
	"sort"
	"strings"
)

// LogicalFieldType 表示Runtime Metadata向消费者暴露的逻辑值类型，
// 与底层SQL列类型及前端编辑组件保持解耦。
type LogicalFieldType = enum.SysTableFieldLogicalType

const (
	LogicalFieldTypeInteger  = enum.LogicalTypeInteger
	LogicalFieldTypeDecimal  = enum.LogicalTypeDecimal
	LogicalFieldTypeString   = enum.LogicalTypePlain
	LogicalFieldTypeText     = enum.LogicalTypePlain
	LogicalFieldTypeBoolean  = enum.LogicalTypeBoolean
	LogicalFieldTypeDate     = enum.LogicalTypeDate
	LogicalFieldTypeDateTime = enum.LogicalTypeDateTime
	LogicalFieldTypeTime     = enum.LogicalTypePlain
	LogicalFieldTypeJSON     = enum.LogicalTypePlain
)

// TableMetadata 是稳定的运行时投影，不包含View SQL、管理审计字段和物理DDL细节。
type TableMetadata struct {
	ID               int
	Code             string
	Name             string
	TableType        enum.SysTableType
	MasterDetailMode enum.SysMasterDetailMode
	FormOpenMode     enum.SysFormOpenMode
	DetailOpenMode   enum.SysDetailOpenMode
	Fields           []FieldMetadata
	Relations        []RelationMetadata
}

// FieldMetadata 将存储类型、逻辑类型和UI组件分开表达，
// 即使当前持久化模型把这些配置保存在同一行中也不混淆其职责。
type FieldMetadata struct {
	ID                 int
	TableID            int
	Code               string
	DisplayName        string
	StorageType        enum.SysTableFieldType
	LogicalType        LogicalFieldType
	UIComponent        enum.SysTableFieldInputType
	Length             int
	DecimalLength      int
	NumericPrecision   int
	NumericScale       int
	DisplayFormat      enum.SysTableFieldDisplayFormat
	ListWidth          *int
	FormSpan           uint8
	DetailSpan         uint8
	DefaultValue       *string
	DictionaryCode     *string
	PrimaryKey         bool
	Indexed            bool
	QuickQuery         bool
	AdvancedQuery      bool
	Sortable           bool
	Nullable           bool
	ListVisible        bool
	DetailVisible      bool
	InsertVisible      bool
	UpdateVisible      bool
	Sequence           uint8
	OriginalFieldID    int
	Binding            string
	Category           enum.SysTableFieldCategory
	RelationExpression string
	LinkageConfig      *string
	SystemManaged      bool
	Relation           *RelationDisplayMetadata
}

// RelationDisplayMetadata 描述关系字段的值和展示合同，供列表、详情、表单及查询共同使用。
type RelationDisplayMetadata struct {
	TargetTableCode string
	ValueField      string
	DisplayField    string
	ParentField     string
	FilterMapping   map[string]string
}

// RelationMetadata 是运行时可见的表关系投影，不暴露管理和DDL内部状态。
type RelationMetadata struct {
	ID             int
	TableID        int
	RelatedTableID int
	ReferenceKey   string
	ForeignKey     string
	RelationType   enum.SysTableRelationType
	ManyTableCode  string
}

// QueryFieldMetadata 只保留查询和列表解析需要的字段能力。
type QueryFieldMetadata struct {
	ID            int
	TableCode     string
	Code          string
	DisplayName   string
	LogicalType   LogicalFieldType
	UIComponent   enum.SysTableFieldInputType
	QuickQuery    bool
	AdvancedQuery bool
	Sortable      bool
	ListVisible   bool
	Sequence      uint8
}

// RuntimeReader 是平台运行时消费者使用的稳定只读边界，
// 不暴露缓存生命周期方法和Metadata管理模型。
type RuntimeReader interface {
	GetTable(context.Context, string) (TableMetadata, error)
	GetTableByID(context.Context, int) (TableMetadata, error)
	GetField(context.Context, int) (FieldMetadata, error)
	GetFields(context.Context, int) ([]FieldMetadata, error)
	ListTables(context.Context) ([]TableMetadata, error)
}

func ProjectTable(source model.SysTable) TableMetadata {
	result := TableMetadata{
		ID:               source.Id,
		Code:             source.TableCode,
		Name:             source.TableName,
		TableType:        source.TableType,
		MasterDetailMode: source.MasterDetailMode,
		FormOpenMode:     source.FormOpenMode,
		DetailOpenMode:   source.DetailOpenMode,
		Fields:           make([]FieldMetadata, 0, len(source.TableFields)),
		Relations:        make([]RelationMetadata, 0, len(source.TableRelations)),
	}
	for _, field := range source.TableFields {
		if !field.State || security.IsSensitiveFieldName(field.FieldCode) {
			continue
		}
		projected, ok := ProjectField(field)
		if ok {
			result.Fields = append(result.Fields, projected)
		}
	}
	for _, relation := range source.TableRelations {
		if !relation.State {
			continue
		}
		result.Relations = append(result.Relations, RelationMetadata{
			ID:             relation.Id,
			TableID:        relation.TableId,
			RelatedTableID: relation.RelatedTableId,
			ReferenceKey:   relation.ReferenceKey,
			ForeignKey:     relation.ForeignKey,
			RelationType:   relation.RelationType,
			ManyTableCode:  relation.ManyTableCode,
		})
	}
	sort.SliceStable(result.Fields, func(i, j int) bool {
		if result.Fields[i].Sequence == result.Fields[j].Sequence {
			return result.Fields[i].Code < result.Fields[j].Code
		}
		return result.Fields[i].Sequence < result.Fields[j].Sequence
	})
	return result
}

func ProjectField(source model.SysTableField) (FieldMetadata, bool) {
	if !source.State || source.Id <= 0 || source.TableId <= 0 || strings.TrimSpace(source.FieldCode) == "" ||
		security.IsSensitiveFieldName(source.FieldCode) {
		return FieldMetadata{}, false
	}
	expression := strings.TrimSpace(stringValue(source.Expression))
	if source.FieldCategory != "" && source.FieldCategory != enum.NormalField {
		if !isStructuredRelationExpression(expression) {
			return FieldMetadata{}, false
		}
	}
	managed := security.IsManagedMetadataField(source.FieldCode)
	return FieldMetadata{
		ID:                 source.Id,
		TableID:            source.TableId,
		Code:               source.FieldCode,
		DisplayName:        source.FieldName,
		StorageType:        source.FieldType,
		LogicalType:        resolveLogicalType(source),
		DisplayFormat:      resolveDisplayFormat(source),
		UIComponent:        source.InputType,
		Length:             source.FieldLength,
		DecimalLength:      source.FieldDecimalLength,
		NumericPrecision:   numericPrecision(source),
		NumericScale:       numericScale(source),
		ListWidth:          cloneInt(source.ListWidth),
		FormSpan:           source.FormSpan,
		DetailSpan:         source.DetailSpan,
		DefaultValue:       cloneString(source.DefaultValue),
		DictionaryCode:     cloneString(source.DictCode),
		PrimaryKey:         source.IsPrimaryKey,
		Indexed:            source.IsIndex,
		QuickQuery:         source.IsQuickSearch && !managed,
		AdvancedQuery:      source.IsAdvancedSearch && !managed,
		Sortable:           source.IsSort,
		Nullable:           source.IsNull,
		ListVisible:        source.IsListShow && !managed,
		DetailVisible:      !managed || source.IsListShow,
		InsertVisible:      source.IsInsertShow && !managed,
		UpdateVisible:      source.IsUpdateShow && !managed,
		Sequence:           source.Sequence,
		OriginalFieldID:    source.OriginalFieldId,
		Binding:            source.Binding,
		Category:           source.FieldCategory,
		RelationExpression: expression,
		LinkageConfig:      cloneString(source.LinkageConfig),
		SystemManaged:      managed,
		Relation:           relationDisplayMetadata(source.LinkageConfig),
	}, true
}

func (table TableMetadata) QueryFields() []QueryFieldMetadata {
	result := make([]QueryFieldMetadata, 0, len(table.Fields))
	for _, field := range table.Fields {
		if !field.QuickQuery && !field.AdvancedQuery && !field.ListVisible {
			continue
		}
		result = append(result, QueryFieldMetadata{
			ID:            field.ID,
			TableCode:     table.Code,
			Code:          field.Code,
			DisplayName:   field.DisplayName,
			LogicalType:   field.LogicalType,
			UIComponent:   field.UIComponent,
			QuickQuery:    field.QuickQuery,
			AdvancedQuery: field.AdvancedQuery,
			Sortable:      field.Sortable,
			ListVisible:   field.ListVisible,
			Sequence:      field.Sequence,
		})
	}
	return result
}

// QueryModel 是Runtime Metadata进入现有动态查询引擎的唯一技术桥接。
// 生产调用方仅限Generalization和当前Report模块；新的运行时消费者不得依赖该持久化模型投影。
func (table TableMetadata) QueryModel() model.SysTable {
	result := model.SysTable{
		Basic:            model.Basic{Id: table.ID, State: true},
		TableCode:        table.Code,
		TableName:        table.Name,
		TableType:        table.TableType,
		MasterDetailMode: table.MasterDetailMode,
		FormOpenMode:     table.FormOpenMode,
		DetailOpenMode:   table.DetailOpenMode,
		TableFields:      make([]model.SysTableField, 0, len(table.Fields)),
		TableRelations:   make([]model.SysTableRelation, 0, len(table.Relations)),
	}
	for _, field := range table.Fields {
		expression := stringPointer(field.RelationExpression)
		result.TableFields = append(result.TableFields, model.SysTableField{
			Basic:              model.Basic{Id: field.ID, State: true},
			TableId:            field.TableID,
			FieldName:          field.DisplayName,
			FieldCode:          field.Code,
			FieldType:          field.StorageType,
			FieldLength:        field.Length,
			FieldDecimalLength: field.DecimalLength,
			NumericPrecision:   field.NumericPrecision,
			NumericScale:       field.NumericScale,
			LogicalType:        field.LogicalType,
			DisplayFormat:      field.DisplayFormat,
			ListWidth:          cloneInt(field.ListWidth),
			InputType:          field.UIComponent,
			FormSpan:           field.FormSpan,
			DetailSpan:         field.DetailSpan,
			DefaultValue:       cloneString(field.DefaultValue),
			DictCode:           cloneString(field.DictionaryCode),
			IsPrimaryKey:       field.PrimaryKey,
			IsIndex:            field.Indexed,
			IsQuickSearch:      field.QuickQuery,
			IsAdvancedSearch:   field.AdvancedQuery,
			IsSort:             field.Sortable,
			IsNull:             field.Nullable,
			IsListShow:         field.ListVisible,
			IsInsertShow:       field.InsertVisible,
			IsUpdateShow:       field.UpdateVisible,
			Sequence:           field.Sequence,
			OriginalFieldId:    field.OriginalFieldID,
			Binding:            field.Binding,
			FieldCategory:      field.Category,
			Expression:         expression,
			LinkageConfig:      cloneString(field.LinkageConfig),
		})
	}
	for _, relation := range table.Relations {
		result.TableRelations = append(result.TableRelations, model.SysTableRelation{
			Basic:          model.Basic{Id: relation.ID, State: true},
			TableId:        relation.TableID,
			RelatedTableId: relation.RelatedTableID,
			ReferenceKey:   relation.ReferenceKey,
			ForeignKey:     relation.ForeignKey,
			RelationType:   relation.RelationType,
			ManyTableCode:  relation.ManyTableCode,
		})
	}
	return result
}

func logicalFieldType(fieldType enum.SysTableFieldType) LogicalFieldType {
	if descriptor, ok := DescribeStorage(fieldType); ok {
		return descriptor.LogicalType
	}
	return LogicalFieldTypeString
}

func resolveLogicalType(field model.SysTableField) LogicalFieldType {
	if normalized, ok := enum.NormalizeSysTableFieldLogicalType(string(field.LogicalType)); ok && normalized != "" {
		return normalized
	}
	if field.LinkageConfig != nil {
		return enum.LogicalTypeRelation
	}
	if field.DictCode != nil {
		return enum.LogicalTypeEnum
	}
	return logicalFieldType(field.FieldType)
}

func resolveDisplayFormat(field model.SysTableField) enum.SysTableFieldDisplayFormat {
	if normalized, ok := enum.NormalizeSysTableFieldDisplayFormat(string(field.DisplayFormat)); ok && normalized != "" {
		return normalized
	}
	switch resolveLogicalType(field) {
	case enum.LogicalTypeMoney:
		return enum.DisplayFormatMoney
	case enum.LogicalTypePercent:
		return enum.DisplayFormatPercent
	case enum.LogicalTypeEnum:
		return enum.DisplayFormatDictionary
	case enum.LogicalTypeRelation:
		return enum.DisplayFormatRelation
	}
	descriptor, _ := DescribeStorage(field.FieldType)
	return descriptor.DisplayFormat
}

func numericPrecision(field model.SysTableField) int {
	if field.NumericPrecision > 0 {
		return field.NumericPrecision
	}
	if field.FieldType == enum.DecimalFieldType {
		if field.FieldLength > 0 {
			return field.FieldLength
		}
		return DefaultNumericPrecision
	}
	return 0
}

func numericScale(field model.SysTableField) int {
	if field.NumericScale > 0 {
		return field.NumericScale
	}
	if field.FieldType == enum.DecimalFieldType {
		return field.FieldDecimalLength
	}
	return 0
}

func relationDisplayMetadata(raw *string) *RelationDisplayMetadata {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var envelope struct {
		Linkage struct {
			Enabled       bool              `json:"enabled"`
			TableCode     string            `json:"tableCode"`
			ValueKey      string            `json:"valueKey"`
			LabelKey      string            `json:"labelKey"`
			ParentKey     string            `json:"parentKey"`
			FilterMapping map[string]string `json:"filterMapping"`
		} `json:"linkage"`
	}
	if json.Unmarshal([]byte(*raw), &envelope) != nil || !envelope.Linkage.Enabled ||
		envelope.Linkage.TableCode == "" || envelope.Linkage.ValueKey == "" || envelope.Linkage.LabelKey == "" {
		return nil
	}
	return &RelationDisplayMetadata{TargetTableCode: envelope.Linkage.TableCode, ValueField: envelope.Linkage.ValueKey,
		DisplayField: envelope.Linkage.LabelKey, ParentField: envelope.Linkage.ParentKey,
		FilterMapping: envelope.Linkage.FilterMapping}
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func isStructuredRelationExpression(value string) bool {
	if !strings.HasPrefix(value, "rel:") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(value, "rel:"), ".")
	return len(parts) == 2 && validIdentifier(parts[0]) && validIdentifier(parts[1])
}

func validIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 63 {
		return false
	}
	for index := range value {
		char := value[index]
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '_' ||
			(index > 0 && char >= '0' && char <= '9') {
			continue
		}
		return false
	}
	return true
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
