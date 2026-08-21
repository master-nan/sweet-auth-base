package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldProxyToBackend(t *testing.T) {
	proxied := []string{
		"/sweet_admin/admin/login",
		"/sweet_admin/api/app_token",
		"/sweet_admin/files/file-id",
	}
	for _, path := range proxied {
		if !shouldProxyToBackend(path) {
			t.Fatalf("expected %s to be proxied", path)
		}
	}

	static := []string{
		"/sweet_admin",
		"/sweet_admin/",
		"/sweet_admin/assets/app.js",
		"/assets/app.js",
	}
	for _, path := range static {
		if shouldProxyToBackend(path) {
			t.Fatalf("expected %s to be served by SPA", path)
		}
	}
}

func TestServeStaticStopsOnContextCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveStatic(ctx, listener, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
		}))
	}()

	response, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveStatic: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("static server did not stop after cancellation")
	}
}

func TestStaticHandlerServesAuthCenterPrefixedSPA(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("asset"), 0o600); err != nil {
		t.Fatal(err)
	}

	fs := http.Dir(root)
	handler := newStaticHandler(root, fs, http.FileServer(fs), "/sweet_admin")
	tests := []struct {
		path string
		code int
		body string
	}{
		{path: "/sweet_admin/", code: http.StatusOK, body: "index"},
		{path: "/sweet_admin/system/audit", code: http.StatusOK, body: "index"},
		{path: "/sweet_admin/assets/app.js", code: http.StatusOK, body: "asset"},
		{path: "/sweet_admin/assets/missing.js", code: http.StatusNotFound, body: "404 page not found\n"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != tt.code {
				t.Fatalf("expected %d, got %d", tt.code, recorder.Code)
			}
			if recorder.Body.String() != tt.body {
				t.Fatalf("expected %q, got %q", tt.body, recorder.Body.String())
			}
		})
	}
}
