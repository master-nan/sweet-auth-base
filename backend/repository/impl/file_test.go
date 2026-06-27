package impl

import (
	"strings"
	"testing"
)

func TestDeletedFileUniqueValueFitsColumn(t *testing.T) {
	value := deletedFileUniqueValue(strings.Repeat("a", 200), 12345)

	if len(value) > 128 {
		t.Fatalf("deleted unique value exceeds column size: %d", len(value))
	}
	if !strings.HasSuffix(value, "#deleted-12345") {
		t.Fatalf("deleted unique value missing suffix: %s", value)
	}
	if again := deletedFileUniqueValue(value, 12345); again != value {
		t.Fatalf("deleted unique value should be idempotent, got %s", again)
	}
}
