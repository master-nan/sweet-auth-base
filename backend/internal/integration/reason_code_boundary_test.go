package integration

import "testing"

func TestRuntimeReasonCodesAreDiagnosticFactsNotGoErrors(t *testing.T) {
	for _, value := range []any{RetryReasonAttemptsExhausted, SyncBusinessReasonProcessingFailed} {
		if _, ok := value.(error); ok {
			t.Fatalf("integration reason code %v must not implement error", value)
		}
	}
}
