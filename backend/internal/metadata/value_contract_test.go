package metadata

import (
	"backend/enum"
	"testing"
)

func TestStorageDescriptorsUseCanonicalFieldTypeIDs(t *testing.T) {
	canonical := []enum.SysTableFieldType{
		enum.BigIntFieldType, enum.DecimalFieldType, enum.VarcharFieldType,
		enum.TextFieldType, enum.BooleanFieldType, enum.DateFieldType,
		enum.DatetimeFieldType, enum.TimeFieldType, enum.SmallIntFieldType,
		enum.JsonFieldType, enum.IntFieldType,
	}
	for index, fieldType := range canonical {
		if fieldType != enum.SysTableFieldType(index+1) {
			t.Fatalf("canonical field type at index %d = %d", index, fieldType)
		}
		if _, ok := DescribeStorage(fieldType); !ok {
			t.Fatalf("canonical storage type %d must be accepted", fieldType)
		}
	}
	for _, removed := range []enum.SysTableFieldType{12, 13} {
		if _, ok := DescribeStorage(removed); ok {
			t.Fatalf("historical storage type %d must not be accepted", removed)
		}
	}
	decimal, ok := DescribeStorage(enum.DecimalFieldType)
	if !ok || decimal.SQLType != "numeric" || !decimal.Ordered {
		t.Fatalf("unexpected Decimal descriptor: %+v", decimal)
	}
}

func TestNormalizeDecimalPreservesExactTextAndBounds(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		precision int
		scale     int
		want      string
	}{
		{name: "money", value: "999999999999999999.99", precision: 20, scale: 2, want: "999999999999999999.99"},
		{name: "weight", value: "000123456789.123400", precision: 18, scale: 6, want: "123456789.123400"},
		{name: "negative", value: "-0.1250", precision: 8, scale: 4, want: "-0.1250"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeDecimal(test.value, test.precision, test.scale)
			if err != nil || got != test.want {
				t.Fatalf("NormalizeDecimal()=%q, %v; want %q", got, err, test.want)
			}
		})
	}
	for _, invalid := range []any{"1e10", 1.25, "123456789.12", "1.234"} {
		if _, err := NormalizeDecimal(invalid, 10, 2); err == nil {
			t.Fatalf("expected decimal value %#v to be rejected", invalid)
		}
	}
}

func TestSmallIntBounds(t *testing.T) {
	if SmallIntMin != -32768 || SmallIntMax != 32767 {
		t.Fatalf("unexpected SmallInt range %d..%d", SmallIntMin, SmallIntMax)
	}
}
