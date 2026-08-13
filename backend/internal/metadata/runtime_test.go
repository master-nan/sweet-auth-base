package metadata

import (
	"backend/enum"
	"backend/model"
	"testing"
)

func TestProjectTableSeparatesRuntimeConcernsAndFiltersUnsafeFields(t *testing.T) {
	relationExpression := "rel:org_unit.name"
	unsafeExpression := "(select password from sys_user)"
	table := model.SysTable{
		Basic:            model.Basic{Id: 10, State: true},
		TableCode:        "org_employee",
		TableName:        "员工",
		TableType:        enum.System,
		MasterDetailMode: enum.MasterDetailSummary,
		FormOpenMode:     enum.FormOpenDialog,
		DetailOpenMode:   enum.DetailOpenPage,
		TableFields: []model.SysTableField{
			metadataTestField(14, "display_name", 4, enum.VarcharFieldType),
			metadataTestField(11, "password_hash", 1, enum.VarcharFieldType),
			metadataTestField(13, "gmt_create", 2, enum.DatetimeFieldType),
			{
				Basic:         model.Basic{Id: 15, State: true},
				TableId:       10,
				FieldName:     "组织名称",
				FieldCode:     "org_name",
				FieldType:     enum.VarcharFieldType,
				InputType:     enum.InputType,
				Sequence:      3,
				FieldCategory: enum.VirtualField,
				Expression:    &relationExpression,
				IsListShow:    true,
			},
			{
				Basic:         model.Basic{Id: 16, State: true},
				TableId:       10,
				FieldName:     "危险表达式",
				FieldCode:     "unsafe_value",
				FieldType:     enum.VarcharFieldType,
				InputType:     enum.InputType,
				Sequence:      5,
				FieldCategory: enum.CalculatedField,
				Expression:    &unsafeExpression,
			},
			{
				Basic:     model.Basic{Id: 17, State: false},
				TableId:   10,
				FieldName: "停用字段",
				FieldCode: "disabled_field",
				FieldType: enum.VarcharFieldType,
				InputType: enum.InputType,
				Sequence:  6,
			},
		},
	}

	projected := ProjectTable(table)
	if projected.Code != table.TableCode || projected.Name != table.TableName || projected.TableType != table.TableType {
		t.Fatalf("unexpected table projection: %+v", projected)
	}
	if projected.MasterDetailMode != table.MasterDetailMode || projected.FormOpenMode != table.FormOpenMode || projected.DetailOpenMode != table.DetailOpenMode {
		t.Fatalf("runtime UI metadata was not preserved: %+v", projected)
	}
	if len(projected.Fields) != 3 {
		t.Fatalf("runtime fields = %d, want 3: %+v", len(projected.Fields), projected.Fields)
	}
	if projected.Fields[0].Code != "gmt_create" || !projected.Fields[0].SystemManaged {
		t.Fatalf("managed field projection = %+v", projected.Fields[0])
	}
	if projected.Fields[0].ListVisible || projected.Fields[0].QuickQuery || projected.Fields[0].AdvancedQuery {
		t.Fatalf("managed field exposed runtime capabilities: %+v", projected.Fields[0])
	}
	if projected.Fields[1].Code != "org_name" || projected.Fields[1].RelationExpression != relationExpression {
		t.Fatalf("structured relation projection = %+v", projected.Fields[1])
	}
	if projected.Fields[2].Code != "display_name" || projected.Fields[2].LogicalType != LogicalFieldTypeString {
		t.Fatalf("business field projection = %+v", projected.Fields[2])
	}

	queryFields := projected.QueryFields()
	if len(queryFields) != 2 || queryFields[0].Code != "org_name" || queryFields[1].Code != "display_name" {
		t.Fatalf("query fields = %+v", queryFields)
	}

	legacy := projected.QueryModel()
	if len(legacy.TableFields) != 3 || legacy.TableFields[0].IsListShow {
		t.Fatalf("legacy compatibility projection leaked capabilities: %+v", legacy.TableFields)
	}
	if legacy.MasterDetailMode != table.MasterDetailMode || legacy.FormOpenMode != table.FormOpenMode || legacy.DetailOpenMode != table.DetailOpenMode {
		t.Fatalf("legacy runtime UI metadata was not preserved: %+v", legacy)
	}
}

func TestProjectFieldMapsStorageLogicalAndUITypeSeparately(t *testing.T) {
	field := metadataTestField(21, "amount", 1, enum.FloatFieldType)
	field.InputType = enum.InputNumberInputType
	field.IsQuickSearch = true
	field.IsAdvancedSearch = true

	projected, ok := ProjectField(field)
	if !ok {
		t.Fatal("expected field projection")
	}
	if projected.StorageType != enum.FloatFieldType || projected.LogicalType != LogicalFieldTypeDecimal || projected.UIComponent != enum.InputNumberInputType {
		t.Fatalf("field type boundaries collapsed: %+v", projected)
	}
}

func metadataTestField(id int, code string, sequence uint8, fieldType enum.SysTableFieldType) model.SysTableField {
	return model.SysTableField{
		Basic:            model.Basic{Id: id, State: true},
		TableId:          10,
		FieldName:        code,
		FieldCode:        code,
		FieldType:        fieldType,
		InputType:        enum.InputType,
		Sequence:         sequence,
		FieldCategory:    enum.NormalField,
		IsListShow:       true,
		IsQuickSearch:    true,
		IsAdvancedSearch: true,
	}
}
