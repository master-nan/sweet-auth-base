package audit

import (
	"context"
	"testing"
)

func TestCorrelationIDsRoundTrip(t *testing.T) {
	ctx := WithCorrelationIDs(context.Background(), CorrelationIDs{RequestID: " request-1 ", TraceID: "trace-1"})
	ids := GetCorrelationIDs(ctx)
	if ids.RequestID != "request-1" || ids.TraceID != "trace-1" {
		t.Fatalf("unexpected correlation IDs: %+v", ids)
	}
	if empty := GetCorrelationIDs(nil); empty != (CorrelationIDs{}) {
		t.Fatalf("nil context should return empty IDs: %+v", empty)
	}
}
