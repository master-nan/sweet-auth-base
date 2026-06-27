package service

import (
	"backend/enum"
	"backend/model"
	"strings"
	"testing"
)

func TestBuildColumnSQLTypeUsesPostgresLengthRules(t *testing.T) {
	intSQL := buildColumnSQLTypeFromField(model.SysTableField{
		FieldType:   enum.IntFieldType,
		FieldLength: 11,
		IsNull:      false,
	})
	if strings.Contains(intSQL, "(") || !strings.HasPrefix(intSQL, "integer") {
		t.Fatalf("integer column SQL should not include length, got %q", intSQL)
	}

	varcharSQL := buildColumnSQLTypeFromField(model.SysTableField{
		FieldType:   enum.VarcharFieldType,
		FieldLength: 64,
		IsNull:      true,
	})
	if !strings.HasPrefix(varcharSQL, "varchar(64)") {
		t.Fatalf("varchar column SQL should keep length, got %q", varcharSQL)
	}
}
