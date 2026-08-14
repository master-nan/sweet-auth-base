package hrsync

import "testing"

func TestReasonCodeIsDiagnosticFactNotGoError(t *testing.T) {
	var value any = ReasonParentUnresolved
	if _, ok := value.(error); ok {
		t.Fatal("organization sync reason code must not implement error")
	}
	if !ReasonParentUnresolved.Valid() {
		t.Fatal("expected stable organization reason code")
	}
}
