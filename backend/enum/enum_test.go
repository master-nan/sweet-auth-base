package enum

import "testing"

func TestSysTableFieldTypesAreCanonicalAndContinuous(t *testing.T) {
	fieldTypes := []SysTableFieldType{
		BigIntFieldType,
		DecimalFieldType,
		VarcharFieldType,
		TextFieldType,
		BooleanFieldType,
		DateFieldType,
		DatetimeFieldType,
		TimeFieldType,
		SmallIntFieldType,
		JsonFieldType,
		IntFieldType,
	}
	for index, fieldType := range fieldTypes {
		if fieldType != SysTableFieldType(index+1) {
			t.Fatalf("field type at index %d = %d, want %d", index, fieldType, index+1)
		}
	}
}
