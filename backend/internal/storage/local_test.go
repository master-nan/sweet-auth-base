package storage

import (
	"io"
	"strings"
	"testing"
)

func TestLocalStorageAllowsSafeRelativePaths(t *testing.T) {
	store := NewLocalStorage(t.TempDir(), "/files")

	url, err := store.Save("2026/06/demo.txt", strings.NewReader("hello"), "text/plain")
	if err != nil {
		t.Fatalf("save safe path: %v", err)
	}
	if url != "/files/2026/06/demo.txt" {
		t.Fatalf("unexpected url: %s", url)
	}

	reader, err := store.Get("2026/06/demo.txt")
	if err != nil {
		t.Fatalf("get safe path: %v", err)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected file content: %q", string(content))
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	store := NewLocalStorage(t.TempDir(), "/files")

	for _, path := range []string{"../secret.txt", "2026/../../secret.txt", "/tmp/secret.txt"} {
		if _, err := store.Save(path, strings.NewReader("x"), "text/plain"); err == nil {
			t.Fatalf("expected Save to reject path %q", path)
		}
		if _, err := store.Get(path); err == nil {
			t.Fatalf("expected Get to reject path %q", path)
		}
		if err := store.Delete(path); err == nil {
			t.Fatalf("expected Delete to reject path %q", path)
		}
	}
}
