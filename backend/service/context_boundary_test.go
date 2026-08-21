package service

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGinContextBoundary(t *testing.T) {
	root := filepath.Clean("..")
	assertGoFilesExcludeGin(t, filepath.Join(root, "repository"), nil)
	assertGoFilesExcludeGin(t, filepath.Join(root, "model"), nil)
	assertGoFilesExcludeGin(t, filepath.Join(root, "service"), map[string]struct{}{
		"report_service.go": {},
	})
}

func assertGoFilesExcludeGin(t *testing.T, root string, allowed map[string]struct{}) {
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
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if importPath == "github.com/gin-gonic/gin" {
				t.Errorf("Gin Context crossed into %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
}
