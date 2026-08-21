package service

import (
	"backend/enum"
	"backend/model"
	"strings"
	"testing"
)

func TestValidateMetadataFieldDefinitionRejectsProtectedAndExecutableMetadata(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*model.SysTableField)
	}{
		{
			name: "sensitive list field",
			mutate: func(field *model.SysTableField) {
				field.FieldCode = "access_token"
				field.IsListShow = true
			},
		},
		{
			name: "managed query field",
			mutate: func(field *model.SysTableField) {
				field.FieldCode = "gmt_delete"
				field.IsAdvancedSearch = true
			},
		},
		{
			name: "arbitrary expression",
			mutate: func(field *model.SysTableField) {
				field.FieldCategory = enum.CalculatedField
				field.Expression = metadataStringPointer("select pg_sleep(1)")
			},
		},
		{
			name: "numeric SQL fragment default",
			mutate: func(field *model.SysTableField) {
				field.FieldType = enum.IntFieldType
				field.DefaultValue = metadataStringPointer("0; drop table sys_user")
			},
		},
		{
			name: "historical smallint type",
			mutate: func(field *model.SysTableField) {
				field.FieldType = enum.SysTableFieldType(12)
			},
		},
		{
			name: "historical decimal type",
			mutate: func(field *model.SysTableField) {
				field.FieldType = enum.SysTableFieldType(13)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := validMetadataFieldDefinition()
			test.mutate(&field)
			if err := validateMetadataFieldDefinition(&field, int(field.Sequence)); err == nil {
				t.Fatal("expected metadata definition rejection")
			}
		})
	}
}

func TestValidateMetadataFieldDefinitionAllowsStructuredRelationOnly(t *testing.T) {
	field := validMetadataFieldDefinition()
	field.FieldCategory = enum.VirtualField
	field.Expression = metadataStringPointer("rel:org_unit.name")
	field.IsQuickSearch = false
	field.IsAdvancedSearch = false

	if err := validateMetadataFieldDefinition(&field, int(field.Sequence)); err != nil {
		t.Fatalf("valid structured relation rejected: %v", err)
	}
}

func TestMetadataDefinitionValidatesTableRelationAndIdentifierContracts(t *testing.T) {
	if err := validateMetadataTableType(enum.SysTableType(99)); err == nil {
		t.Fatal("unknown table type accepted")
	}
	if err := validateMetadataRelation(enum.SysTableRelationType(99), ""); err == nil {
		t.Fatal("unknown relation type accepted")
	}
	if err := validateMetadataRelation(enum.OneToMany, "unexpected_join"); err == nil {
		t.Fatal("non-many-to-many relation accepted a join table")
	}
	if _, err := normalizeDBIdentifier("表编码", strings.Repeat("a", maxDBIdentifierLength+1)); err == nil {
		t.Fatal("identifier longer than PostgreSQL limit accepted")
	}
}

func TestValidateMetadataViewSQLIsReadOnlyAndSingleStatement(t *testing.T) {
	if got, err := validateMetadataViewSQL(" SELECT id FROM org_unit "); err != nil || got != "SELECT id FROM org_unit" {
		t.Fatalf("valid view query = %q, %v", got, err)
	}
	for _, value := range []string{
		"",
		"DROP TABLE org_unit",
		"SELECT id FROM org_unit; DELETE FROM org_unit",
		"WITH removed AS (DELETE FROM org_unit RETURNING id) SELECT * FROM removed",
	} {
		if _, err := validateMetadataViewSQL(value); err == nil {
			t.Fatalf("unsafe view SQL accepted: %q", value)
		}
	}
}

func validMetadataFieldDefinition() model.SysTableField {
	return model.SysTableField{
		Basic:         model.Basic{Id: 1, State: true},
		TableId:       10,
		FieldName:     "名称",
		FieldCode:     "name",
		FieldType:     enum.VarcharFieldType,
		FieldLength:   128,
		InputType:     enum.InputType,
		Sequence:      1,
		FieldCategory: enum.NormalField,
		IsNull:        true,
		IsListShow:    true,
		IsInsertShow:  true,
		IsUpdateShow:  true,
	}
}

func metadataStringPointer(value string) *string {
	return &value
}
