package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/model"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

func TestConvertColumnsToSysTableFieldsHidesSensitiveFields(t *testing.T) {
	fields := convertColumnsToSysTableFields("sys_user", []model.TableColumnMate{
		{
			ColumnName:             "api_key",
			OrdinalPosition:        1,
			IsNullable:             "NO",
			DataType:               "varchar",
			CharacterMaximumLength: sql.NullInt64{Int64: 64, Valid: true},
		},
		{
			ColumnName:             "name",
			OrdinalPosition:        2,
			IsNullable:             "YES",
			DataType:               "varchar",
			CharacterMaximumLength: sql.NullInt64{Int64: 64, Valid: true},
		},
	})

	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	secret := fields[0]
	if secret.IsListShow || secret.IsInsertShow || secret.IsUpdateShow || secret.IsQuickSearch || secret.IsAdvancedSearch {
		t.Fatalf("expected sensitive field to be hidden from low-code UI/search, got %+v", secret)
	}
	ordinary := fields[1]
	if !ordinary.IsListShow {
		t.Fatalf("expected ordinary field to remain visible in list, got %+v", ordinary)
	}
}

func TestNewBaseTableFieldsHideManagedListWriteAndSearch(t *testing.T) {
	fields := newBaseTableFields(10)
	managed := map[string]bool{
		"id":              true,
		"gmt_create":      true,
		"gmt_create_user": true,
		"gmt_modify":      true,
		"gmt_modify_user": true,
		"gmt_delete":      true,
		"gmt_delete_user": true,
	}

	for _, field := range fields {
		if !managed[field.FieldCode] {
			continue
		}
		if field.IsListShow || field.IsInsertShow || field.IsUpdateShow || field.IsQuickSearch || field.IsAdvancedSearch {
			t.Fatalf("managed base field should not be exposed in list/write/search metadata: %+v", field)
		}
	}
}

func TestConvertColumnsToSysTableFieldsHidesManagedListWriteAndSearch(t *testing.T) {
	fields := convertColumnsToSysTableFields("sys_user", []model.TableColumnMate{
		{
			ColumnName:       "create_user",
			OrdinalPosition:  1,
			IsNullable:       "YES",
			DataType:         "bigint",
			NumericPrecision: sql.NullInt64{Int64: 20, Valid: true},
		},
	})

	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	field := fields[0]
	if field.IsListShow || field.IsInsertShow || field.IsUpdateShow || field.IsQuickSearch || field.IsAdvancedSearch {
		t.Fatalf("managed physical field should not be exposed in list/write/search metadata: %+v", field)
	}
}

func TestFieldVisibilityPatchRepairsManagedAndSensitiveFields(t *testing.T) {
	cases := []string{"gmt_modify_user", "password"}
	for _, fieldCode := range cases {
		patch, changed := fieldVisibilityPatch(model.SysTableField{
			FieldCode:        fieldCode,
			IsListShow:       true,
			IsInsertShow:     true,
			IsUpdateShow:     true,
			IsQuickSearch:    true,
			IsAdvancedSearch: true,
		})
		if !changed {
			t.Fatalf("expected %s visibility to be repaired", fieldCode)
		}
		if patch.IsListShow || patch.IsInsertShow || patch.IsUpdateShow || patch.IsQuickSearch || patch.IsAdvancedSearch {
			t.Fatalf("expected %s patch to hide list/write/search metadata, got %+v", fieldCode, patch)
		}
	}
}

func TestFieldVisibilityPatchPreservesBusinessFields(t *testing.T) {
	patch, changed := fieldVisibilityPatch(model.SysTableField{
		FieldCode:        "customer_name",
		IsListShow:       true,
		IsInsertShow:     true,
		IsUpdateShow:     true,
		IsQuickSearch:    true,
		IsAdvancedSearch: true,
	})
	if changed || patch.FieldCode != "" {
		t.Fatalf("business field visibility should be preserved, changed=%v patch=%+v", changed, patch)
	}
}

func TestConvertColumnsToSysTableFieldsAppliesSystemDictionaries(t *testing.T) {
	fields := convertColumnsToSysTableFields("sys_table_field", []model.TableColumnMate{
		{
			ColumnName:      "field_type",
			OrdinalPosition: 1,
			IsNullable:      "NO",
			DataType:        "tinyint",
			NumericPrecision: sql.NullInt64{
				Int64: 3,
				Valid: true,
			},
		},
		{
			ColumnName:      "input_type",
			OrdinalPosition: 2,
			IsNullable:      "NO",
			DataType:        "tinyint",
			NumericPrecision: sql.NullInt64{
				Int64: 3,
				Valid: true,
			},
		},
		{
			ColumnName:      "is_list_show",
			OrdinalPosition: 3,
			IsNullable:      "NO",
			DataType:        "tinyint",
			ColumnType:      "tinyint(1)",
		},
		{
			ColumnName:      "enabled",
			OrdinalPosition: 4,
			IsNullable:      "NO",
			DataType:        "tinyint",
			NumericPrecision: sql.NullInt64{
				Int64: 3,
				Valid: true,
			},
		},
		{
			ColumnName:             "method",
			OrdinalPosition:        5,
			IsNullable:             "YES",
			DataType:               "varchar",
			CharacterMaximumLength: sql.NullInt64{Int64: 16, Valid: true},
		},
	})

	assertFieldDict(t, fields[0], "sys_table_field_type")
	assertFieldDict(t, fields[1], "sys_table_field_input_type")
	assertFieldDict(t, fields[2], "whether")
	assertFieldDict(t, fields[3], "whether")
	assertFieldDict(t, fields[4], "http_method")

	buttonFields := convertColumnsToSysTableFields("sys_menu_button", []model.TableColumnMate{
		{
			ColumnName:             "event_action",
			OrdinalPosition:        1,
			IsNullable:             "YES",
			DataType:               "varchar",
			CharacterMaximumLength: sql.NullInt64{Int64: 256, Valid: true},
		},
	})
	assertFieldDict(t, buttonFields[0], "sys_menu_button_event_action")

	tableFields := convertColumnsToSysTableFields("sys_table", []model.TableColumnMate{
		{
			ColumnName:             "master_detail_mode",
			OrdinalPosition:        1,
			IsNullable:             "NO",
			DataType:               "varchar",
			CharacterMaximumLength: sql.NullInt64{Int64: 16, Valid: true},
		},
		{
			ColumnName:             "form_open_mode",
			OrdinalPosition:        2,
			IsNullable:             "NO",
			DataType:               "varchar",
			CharacterMaximumLength: sql.NullInt64{Int64: 16, Valid: true},
		},
		{
			ColumnName:             "detail_open_mode",
			OrdinalPosition:        3,
			IsNullable:             "NO",
			DataType:               "varchar",
			CharacterMaximumLength: sql.NullInt64{Int64: 16, Valid: true},
		},
	})
	assertFieldDict(t, tableFields[0], "sys_master_detail_mode")
	assertFieldDict(t, tableFields[1], "sys_form_open_mode")
	assertFieldDict(t, tableFields[2], "sys_detail_open_mode")
}

func TestConvertColumnsToSysTableFieldsRecognizesPostgresTypes(t *testing.T) {
	fields := convertColumnsToSysTableFields("demo_pg", []model.TableColumnMate{
		{ColumnName: "id", OrdinalPosition: 1, IsNullable: "NO", DataType: "bigint"},
		{ColumnName: "count", OrdinalPosition: 2, IsNullable: "YES", DataType: "integer"},
		{ColumnName: "flag", OrdinalPosition: 3, IsNullable: "YES", DataType: "boolean"},
		{ColumnName: "status", OrdinalPosition: 4, IsNullable: "YES", DataType: "smallint", NumericPrecision: sql.NullInt64{Int64: 16, Valid: true}},
		{ColumnName: "name", OrdinalPosition: 5, IsNullable: "YES", DataType: "character varying", CharacterMaximumLength: sql.NullInt64{Int64: 128, Valid: true}},
		{ColumnName: "amount", OrdinalPosition: 6, IsNullable: "YES", DataType: "numeric", NumericPrecision: sql.NullInt64{Int64: 12, Valid: true}, NumericScale: sql.NullInt64{Int64: 2, Valid: true}},
		{ColumnName: "payload", OrdinalPosition: 7, IsNullable: "YES", DataType: "jsonb"},
		{ColumnName: "created_at", OrdinalPosition: 8, IsNullable: "YES", DataType: "timestamp without time zone"},
		{ColumnName: "clock", OrdinalPosition: 9, IsNullable: "YES", DataType: "time without time zone"},
	})
	if len(fields) != 9 {
		t.Fatalf("expected 9 fields, got %d", len(fields))
	}
	cases := []struct {
		index     int
		fieldType enum.SysTableFieldType
		inputType enum.SysTableFieldInputType
	}{
		{0, enum.BigIntFieldType, enum.InputNumberInputType},
		{1, enum.IntFieldType, enum.InputNumberInputType},
		{2, enum.BooleanFieldType, enum.SelectInputType},
		{3, enum.SmallIntFieldType, enum.InputNumberInputType},
		{4, enum.VarcharFieldType, enum.InputType},
		{5, enum.DecimalFieldType, enum.InputNumberInputType},
		{6, enum.JsonFieldType, enum.JsonInputType},
		{7, enum.DatetimeFieldType, enum.DatetimePickerInputType},
		{8, enum.TimeFieldType, enum.TimePickerInputType},
	}
	for _, item := range cases {
		field := fields[item.index]
		if field.FieldType != item.fieldType || field.InputType != item.inputType {
			t.Fatalf("unexpected type for %s: fieldType=%v inputType=%v", field.FieldCode, field.FieldType, field.InputType)
		}
	}
	if fields[4].FieldLength != 128 {
		t.Fatalf("expected varchar length 128, got %d", fields[4].FieldLength)
	}
	if fields[5].NumericPrecision != 12 || fields[5].NumericScale != 2 {
		t.Fatalf("expected numeric(12,2), got precision=%d scale=%d", fields[5].NumericPrecision, fields[5].NumericScale)
	}
}

func TestValidateTableFieldLinkageConfigAllowsValidRelation(t *testing.T) {
	currentTable := model.SysTable{
		Basic:     model.Basic{Id: 1},
		TableCode: "orders",
		TableFields: []model.SysTableField{
			{FieldCode: "customer_id"},
			{FieldCode: "tenant_id"},
		},
	}
	relatedTable := model.SysTable{
		Basic:     model.Basic{Id: 2},
		TableCode: "customers",
		TableFields: []model.SysTableField{
			{FieldCode: "id"},
			{FieldCode: "name"},
			{FieldCode: "tenant_id"},
		},
	}
	raw := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"customers","labelKey":"name","valueKey":"id","filterMapping":{"tenant_id":"tenant_id"}}}`

	err := validateTableFieldLinkageConfig(raw, currentTable, "customer_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return relatedTable, nil
	})
	if err != nil {
		t.Fatalf("expected valid relation linkage config: %v", err)
	}
}

func TestNormalizeTableFieldLinkageConfigPreservesTableCodeAndExtras(t *testing.T) {
	currentTable := model.SysTable{
		Basic:     model.Basic{Id: 1},
		TableCode: "orders",
		TableFields: []model.SysTableField{
			{FieldCode: "customer_id"},
			{FieldCode: "tenant_id"},
		},
	}
	relatedTable := model.SysTable{
		Basic:     model.Basic{Id: 2},
		TableCode: "customers",
		TableFields: []model.SysTableField{
			{FieldCode: "id"},
			{FieldCode: "name"},
			{FieldCode: "tenant_id"},
		},
	}
	raw := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"customers","labelKey":"name","valueKey":"id","searchPageSize":80,"targetMenuId":123,"filterMapping":{" tenant_id ":" tenant_id "}},"memo":"keep"}`

	normalized, err := normalizeTableFieldLinkageConfig(raw, currentTable, "customer_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return relatedTable, nil
	})
	if err != nil {
		t.Fatalf("expected valid relation linkage config: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(normalized), &payload); err != nil {
		t.Fatalf("normalized linkage config is invalid JSON: %v", err)
	}
	linkage, ok := payload["linkage"].(map[string]interface{})
	if !ok {
		t.Fatalf("normalized linkage config missing linkage object: %s", normalized)
	}
	if got := linkage["tableCode"]; got != "customers" {
		t.Fatalf("expected normalized tableCode customers, got %#v in %s", got, normalized)
	}
	if got := linkage["searchPageSize"]; got != float64(80) {
		t.Fatalf("expected searchPageSize extra field to be preserved, got %#v", got)
	}
	if got := linkage["targetMenuId"]; got != float64(123) {
		t.Fatalf("expected targetMenuId extra field to be preserved, got %#v", got)
	}
	if got := payload["memo"]; got != "keep" {
		t.Fatalf("expected outer extra field to be preserved, got %#v", got)
	}
}

func TestValidateTableFieldLinkageConfigAllowsCurrentSelfCascaderField(t *testing.T) {
	currentTable := model.SysTable{
		Basic:     model.Basic{Id: 1},
		TableCode: "companies",
		TableFields: []model.SysTableField{
			{FieldCode: "id"},
			{FieldCode: "company_name"},
		},
	}
	relatedTable := currentTable
	raw := `{"linkage":{"enabled":true,"mode":"cascader","tableCode":"companies","labelKey":"company_name","valueKey":"id","parentKey":"parent_id"}}`

	err := validateTableFieldLinkageConfig(raw, currentTable, "parent_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return relatedTable, nil
	})
	if err != nil {
		t.Fatalf("expected current self cascader field to be accepted: %v", err)
	}
}

func TestValidateTableFieldLinkageConfigRejectsInvalidJSON(t *testing.T) {
	err := validateTableFieldLinkageConfig(`{"linkage":`, model.SysTable{}, "customer_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return model.SysTable{}, nil
	})
	if err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
}

func TestValidateTableFieldLinkageConfigRejectsInvalidMode(t *testing.T) {
	raw := `{"linkage":{"enabled":true,"mode":"popup","tableCode":"customers"}}`

	err := validateTableFieldLinkageConfig(raw, model.SysTable{}, "customer_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return model.SysTable{Basic: model.Basic{Id: 2}}, nil
	})
	if err == nil {
		t.Fatal("expected invalid mode to fail")
	}
}

func TestValidateTableFieldLinkageConfigRejectsMissingRelatedField(t *testing.T) {
	currentTable := model.SysTable{
		Basic:       model.Basic{Id: 1},
		TableCode:   "orders",
		TableFields: []model.SysTableField{{FieldCode: "tenant_id"}},
	}
	relatedTable := model.SysTable{
		Basic:       model.Basic{Id: 2},
		TableCode:   "customers",
		TableFields: []model.SysTableField{{FieldCode: "id"}},
	}
	raw := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"customers","labelKey":"name","valueKey":"id"}}`

	err := validateTableFieldLinkageConfig(raw, currentTable, "customer_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return relatedTable, nil
	})
	if err == nil {
		t.Fatal("expected missing labelKey field to fail")
	}
}

func TestValidateTableFieldLinkageConfigRejectsMissingFilterSourceField(t *testing.T) {
	currentTable := model.SysTable{
		Basic:       model.Basic{Id: 1},
		TableCode:   "orders",
		TableFields: []model.SysTableField{{FieldCode: "tenant_id"}},
	}
	relatedTable := model.SysTable{
		Basic:       model.Basic{Id: 2},
		TableCode:   "customers",
		TableFields: []model.SysTableField{{FieldCode: "id"}, {FieldCode: "tenant_id"}},
	}
	raw := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"customers","valueKey":"id","filterMapping":{"tenant_id":"project_id"}}}`

	err := validateTableFieldLinkageConfig(raw, currentTable, "customer_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return relatedTable, nil
	})
	if err == nil {
		t.Fatal("expected missing filter source field to fail")
	}
}

func TestValidateTableFieldLinkageConfigRejectsCascaderParentSameAsValue(t *testing.T) {
	currentTable := model.SysTable{
		Basic:       model.Basic{Id: 1},
		TableCode:   "companies",
		TableFields: []model.SysTableField{{FieldCode: "parent_id"}},
	}
	relatedTable := model.SysTable{
		Basic:     model.Basic{Id: 1},
		TableCode: "companies",
		TableFields: []model.SysTableField{
			{FieldCode: "id"},
			{FieldCode: "company_name"},
			{FieldCode: "parent_id"},
		},
	}
	raw := `{"linkage":{"enabled":true,"mode":"cascader","tableCode":"companies","labelKey":"company_name","valueKey":"id","parentKey":"id"}}`

	err := validateTableFieldLinkageConfig(raw, currentTable, "parent_id", func(cfg tableFieldLinkageConfig) (model.SysTable, error) {
		return relatedTable, nil
	})
	if err == nil {
		t.Fatal("expected cascader parentKey equal valueKey to fail")
	}
}

func TestValidateTableFieldLinkageConfigRequiresUniqueValueField(t *testing.T) {
	currentTable := model.SysTable{Basic: model.Basic{Id: 1}, TableCode: "orders"}
	target := model.SysTable{
		Basic: model.Basic{Id: 2}, TableCode: "customers",
		TableFields: []model.SysTableField{{FieldCode: "external_code"}, {FieldCode: "name"}},
	}
	raw := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"customers","labelKey":"name","valueKey":"external_code"}}`
	if err := validateTableFieldLinkageConfig(raw, currentTable, "customer_code", func(tableFieldLinkageConfig) (model.SysTable, error) {
		return target, nil
	}); err == nil {
		t.Fatal("expected non-unique relation value field to be rejected")
	}
}

func TestValidateTableFieldLinkageConfigRejectsSensitiveRelationFields(t *testing.T) {
	currentTable := model.SysTable{
		Basic: model.Basic{Id: 1}, TableCode: "orders",
		TableFields: []model.SysTableField{{FieldCode: "customer_id"}, {FieldCode: "tenant_id"}},
	}
	target := model.SysTable{
		Basic: model.Basic{Id: 2}, TableCode: "customers",
		TableFields: []model.SysTableField{
			{FieldCode: "id", IsPrimaryKey: true},
			{FieldCode: "name"},
			{FieldCode: "password"},
		},
	}
	raw := `{"linkage":{"enabled":true,"mode":"relation","tableCode":"customers","labelKey":"name","valueKey":"id","filterMapping":{"password":"tenant_id"}}}`
	if err := validateTableFieldLinkageConfig(raw, currentTable, "customer_id", func(tableFieldLinkageConfig) (model.SysTable, error) {
		return target, nil
	}); err == nil {
		t.Fatal("expected sensitive relation filter field to be rejected")
	}
}

func TestNormalizeDBIdentifierRejectsUnsafeValues(t *testing.T) {
	valid, err := normalizeDBIdentifier("字段编码", " name_01 ")
	if err != nil {
		t.Fatalf("expected safe identifier: %v", err)
	}
	if valid != "name_01" {
		t.Fatalf("expected trimmed identifier, got %q", valid)
	}

	for _, value := range []string{
		"",
		"1name",
		"name;DROP_TABLE",
		"name`x",
		"name x",
		"字段",
		strings.Repeat("a", maxDBIdentifierLength+1),
	} {
		if _, err := normalizeDBIdentifier("字段编码", value); err == nil {
			t.Fatalf("expected unsafe identifier %q to fail", value)
		}
	}
}

func TestValidateTableIndexFieldsRequiresMatchingFieldMetadata(t *testing.T) {
	table := model.SysTable{
		Basic:     model.Basic{Id: 1},
		TableCode: "orders",
		TableFields: []model.SysTableField{
			{Basic: model.Basic{Id: 11}, FieldCode: "name"},
			{Basic: model.Basic{Id: 12}, FieldCode: "tenant_id"},
		},
	}

	normalized, codes, err := validateTableIndexFields(table, []request.TableIndexFieldReq{
		{TableId: 1, FieldId: 11, FieldCode: " name "},
		{TableId: 1, FieldId: 12, FieldCode: "tenant_id"},
	})
	if err != nil {
		t.Fatalf("expected valid index fields: %v", err)
	}
	if len(normalized) != 2 || normalized[0].FieldCode != "name" || codes[1] != "tenant_id" {
		t.Fatalf("unexpected normalized index fields: normalized=%+v codes=%+v", normalized, codes)
	}

	if _, _, err := validateTableIndexFields(table, []request.TableIndexFieldReq{
		{TableId: 1, FieldId: 11, FieldCode: "tenant_id"},
	}); err == nil {
		t.Fatal("expected mismatched field id/code to fail")
	}
	if _, _, err := validateTableIndexFields(table, []request.TableIndexFieldReq{
		{TableId: 2, FieldId: 11, FieldCode: "name"},
	}); err == nil {
		t.Fatal("expected wrong table id to fail")
	}
	if _, _, err := validateTableIndexFields(table, []request.TableIndexFieldReq{
		{TableId: 1, FieldId: 11, FieldCode: "name"},
		{TableId: 1, FieldId: 11, FieldCode: "name"},
	}); err == nil {
		t.Fatal("expected duplicate index field to fail")
	}
}

func assertFieldDict(t *testing.T, field model.SysTableField, dictCode string) {
	t.Helper()
	if field.DictCode == nil || *field.DictCode != dictCode {
		t.Fatalf("expected %s dict %s, got %+v", field.FieldCode, dictCode, field.DictCode)
	}
	if field.InputType != enum.SelectInputType {
		t.Fatalf("expected %s to use select input, got %d", field.FieldCode, field.InputType)
	}
}
