package impl

import "testing"

func TestQuoteSQLIdentifierEscapesDoubleQuotes(t *testing.T) {
	got := quoteSQLIdentifier(`idx"name`)
	if got != `"idx""name"` {
		t.Fatalf("unexpected quoted identifier: %s", got)
	}
}

func TestQuoteSQLIdentifierListSkipsEmptyParts(t *testing.T) {
	got := quoteSQLIdentifierList(" name, ,tenant_id ")
	if got != `"name","tenant_id"` {
		t.Fatalf("unexpected quoted identifier list: %s", got)
	}
}
