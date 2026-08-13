package metadata

import (
	"backend/enum"
	"backend/internal/security"
	"backend/model"
	"context"
	"sort"
	"strings"
)

// LogicalFieldType is the source-independent value type exposed to runtime
// metadata consumers. It deliberately does not describe the physical SQL
// column or the frontend component used to edit the value.
type LogicalFieldType string

const (
	LogicalFieldTypeInteger  LogicalFieldType = "integer"
	LogicalFieldTypeDecimal  LogicalFieldType = "decimal"
	LogicalFieldTypeString   LogicalFieldType = "string"
	LogicalFieldTypeText     LogicalFieldType = "text"
	LogicalFieldTypeBoolean  LogicalFieldType = "boolean"
	LogicalFieldTypeDate     LogicalFieldType = "date"
	LogicalFieldTypeDateTime LogicalFieldType = "datetime"
	LogicalFieldTypeTime     LogicalFieldType = "time"
	LogicalFieldTypeJSON     LogicalFieldType = "json"
)

// TableMetadata is the stable runtime projection. Administration-only facts
// such as view SQL, edit audit fields and physical DDL details are excluded.
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

// FieldMetadata keeps storage, logical and UI concerns separate even though
// the current persistence model stores them on one row.
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
	InsertVisible      bool
	UpdateVisible      bool
	Sequence           uint8
	OriginalFieldID    int
	Binding            string
	Category           enum.SysTableFieldCategory
	RelationExpression string
	LinkageConfig      *string
	SystemManaged      bool
}

type RelationMetadata struct {
	ID             int
	TableID        int
	RelatedTableID int
	ReferenceKey   string
	ForeignKey     string
	RelationType   enum.SysTableRelationType
	ManyTableCode  string
}

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

// RuntimeReader is the stable read-only boundary used by platform runtime
// consumers. Cache lifecycle and configuration models are deliberately absent.
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
		LogicalType:        logicalFieldType(source.FieldType),
		UIComponent:        source.InputType,
		Length:             source.FieldLength,
		DecimalLength:      source.FieldDecimalLength,
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
		InsertVisible:      source.IsInsertShow && !managed,
		UpdateVisible:      source.IsUpdateShow && !managed,
		Sequence:           source.Sequence,
		OriginalFieldID:    source.OriginalFieldId,
		Binding:            source.Binding,
		Category:           source.FieldCategory,
		RelationExpression: expression,
		LinkageConfig:      cloneString(source.LinkageConfig),
		SystemManaged:      managed,
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

// QueryModel is a compatibility adapter for the existing dynamic query
// engine. Runtime callers receive TableMetadata; conversion to the legacy ORM
// description remains inside an application service.
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
	switch fieldType {
	case enum.BigIntFieldType, enum.IntFieldType, enum.TinyintFieldType:
		return LogicalFieldTypeInteger
	case enum.FloatFieldType:
		return LogicalFieldTypeDecimal
	case enum.TextFieldType:
		return LogicalFieldTypeText
	case enum.BooleanFieldType:
		return LogicalFieldTypeBoolean
	case enum.DateFieldType:
		return LogicalFieldTypeDate
	case enum.DatetimeFieldType:
		return LogicalFieldTypeDateTime
	case enum.TimeFieldType:
		return LogicalFieldTypeTime
	case enum.JsonFieldType:
		return LogicalFieldTypeJSON
	default:
		return LogicalFieldTypeString
	}
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
