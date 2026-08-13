package audit

import (
	"context"
	"testing"
)

func TestRequestMetadataRoundTrip(t *testing.T) {
	ctx := WithRequestMetadata(context.Background(), RequestMetadata{
		Method:   " post ",
		Path:     " /admin/org/employee/1 ",
		ClientIP: " 127.0.0.1 ",
	})
	metadata := GetRequestMetadata(ctx)
	if metadata.Method != "POST" || metadata.Path != "/admin/org/employee/1" || metadata.ClientIP != "127.0.0.1" {
		t.Fatalf("unexpected request metadata: %+v", metadata)
	}
}

func TestAuditHelpersReadHTTPAdapterStringValues(t *testing.T) {
	subject := NewAuditSubject(42, "operator")
	ctx := context.WithValue(context.Background(), AuditSubjectValueKey, subject)
	ctx = context.WithValue(ctx, RequestIDValueKey, "request-42")
	ctx = context.WithValue(ctx, TraceIDValueKey, "trace-42")
	ctx = context.WithValue(ctx, RequestMetadataValueKey, RequestMetadata{Method: "POST", Path: "/admin/test"})

	gotSubject, ok := GetAuditSubject(ctx)
	if !ok || gotSubject != subject {
		t.Fatalf("audit subject = %+v, ok=%v", gotSubject, ok)
	}
	if ids := GetCorrelationIDs(ctx); ids.RequestID != "request-42" || ids.TraceID != "trace-42" {
		t.Fatalf("correlation IDs = %+v", ids)
	}
	if metadata := GetRequestMetadata(ctx); metadata.Method != "POST" || metadata.Path != "/admin/test" {
		t.Fatalf("request metadata = %+v", metadata)
	}
}
