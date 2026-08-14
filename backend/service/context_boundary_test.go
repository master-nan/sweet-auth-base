package service

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGinContextBoundary(t *testing.T) {
	root := filepath.Clean("..")
	assertGoFilesExcludeGin(t, filepath.Join(root, "repository"))
	assertGoFilesExcludeGin(t, filepath.Join(root, "model"))
	assertGoFilesExcludeGinExcept(t, filepath.Join(root, "service"), map[string]struct{}{
		"report_service.go": {},
	})
}

func assertGoFilesExcludeGin(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		assertFileExcludesGin(t, path)
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
}

func assertGoFilesExcludeGinExcept(t *testing.T, root string, allowed map[string]struct{}) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if _, ok := allowed[entry.Name()]; ok {
			return nil
		}
		assertFileExcludesGin(t, path)
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
}

func assertFileExcludesGin(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	if strings.Contains(source, "github.com/gin-gonic/gin") || strings.Contains(source, "*gin.Context") {
		t.Errorf("Gin Context crossed into %s", path)
	}
}
