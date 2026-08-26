package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type HTTPRequest struct {
	Method string
	Target string
	Body   io.Reader
	Header http.Header
}

// Eventually 在超时前轮询异步条件，避免测试散落固定时长的 Sleep。
func Eventually(timeout, interval time.Duration, condition func() bool) bool {
	if condition == nil || timeout <= 0 {
		return false
	}
	if condition() {
		return true
	}
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if condition() {
				return true
			}
		case <-timer.C:
			return false
		}
	}
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
