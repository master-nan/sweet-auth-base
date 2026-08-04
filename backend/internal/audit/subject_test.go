package audit

import (
	"context"
	"sync"
	"testing"
)

func TestWithAuditSubjectRoundTrip(t *testing.T) {
	subject := NewAuditSubject(18, "audit-user")
	ctx := WithAuditSubject(context.Background(), subject)

	got, ok := GetAuditSubject(ctx)
	if !ok {
		t.Fatal("expected audit subject")
	}
	if got != subject {
		t.Fatalf("unexpected subject: %+v", got)
	}
}

func TestGetAuditSubjectHandlesMissingOrInvalidSubject(t *testing.T) {
	if _, ok := GetAuditSubject(nil); ok {
		t.Fatal("nil context must not contain an audit subject")
	}
	if _, ok := GetAuditSubject(context.Background()); ok {
		t.Fatal("empty context must not contain an audit subject")
	}

	invalid := WithAuditSubject(context.Background(), NewAuditSubject(0, "anonymous"))
	if _, ok := GetAuditSubject(invalid); ok {
		t.Fatal("subject without user ID must not be accepted")
	}
}

func TestAuditSubjectContextIsSafeForConcurrentReads(t *testing.T) {
	ctx := WithAuditSubject(context.Background(), NewAuditSubject(36, "concurrent-user"))
	const readers = 64

	var group sync.WaitGroup
	group.Add(readers)
	for range readers {
		go func() {
			defer group.Done()
			subject, ok := GetAuditSubject(ctx)
			if !ok || subject.UserID != 36 || subject.UserName != "concurrent-user" {
				t.Errorf("unexpected concurrent subject: %+v, ok=%v", subject, ok)
			}
		}()
	}
	group.Wait()
}
