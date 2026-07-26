package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type HTTPRequest struct {
	Method string
	Target string
	Body   io.Reader
	Header http.Header
}

// PerformRequest executes an HTTP handler without opening a network listener.
func PerformRequest(t testing.TB, handler http.Handler, input HTTPRequest) *httptest.ResponseRecorder {
	t.Helper()
	if handler == nil {
		t.Fatal("HTTP handler is required")
	}
	if input.Method == "" {
		t.Fatal("HTTP method is required")
	}
	if input.Target == "" {
		t.Fatal("HTTP target is required")
	}

	request := httptest.NewRequest(input.Method, input.Target, input.Body)
	request.Header = input.Header.Clone()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// NewHTTPServer creates a lifecycle-managed server for outbound HTTP tests.
func NewHTTPServer(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()
	if handler == nil {
		t.Fatal("HTTP handler is required")
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// AssertIdempotent runs an operation twice and compares deterministic
// snapshots after each run. The snapshot must include stable keys and counts.
func AssertIdempotent[T any](
	t testing.TB,
	run func() error,
	snapshot func() (T, error),
	options ...cmp.Option,
) {
	t.Helper()
	if run == nil {
		t.Fatal("idempotent operation is required")
	}
	if snapshot == nil {
		t.Fatal("idempotency snapshot is required")
	}

	if err := run(); err != nil {
		t.Fatalf("run operation first time: %v", err)
	}
	first, err := snapshot()
	if err != nil {
		t.Fatalf("capture first idempotency snapshot: %v", err)
	}

	if err := run(); err != nil {
		t.Fatalf("run operation second time: %v", err)
	}
	second, err := snapshot()
	if err != nil {
		t.Fatalf("capture second idempotency snapshot: %v", err)
	}

	if diff := cmp.Diff(first, second, options...); diff != "" {
		t.Fatalf("operation is not idempotent (-first +second):\n%s", diff)
	}
}
