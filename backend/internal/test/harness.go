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

// PerformRequest 在不打开网络监听器的情况下执行 HTTP 处理器。
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

// NewHTTPServer 为外部 HTTP 测试创建生命周期受控的服务器。
func NewHTTPServer(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()
	if handler == nil {
		t.Fatal("HTTP handler is required")
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// AssertIdempotent 执行同一操作两次，并比较每次执行后的确定性快照。
// 快照必须包含稳定键和数量。
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
