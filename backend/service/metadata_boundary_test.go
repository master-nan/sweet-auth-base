package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestRuntimeMetadataConsumersDoNotReachConfigurationRepositories(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve source path")
	}
	backendRoot := filepath.Dir(filepath.Dir(currentFile))
	checks := []struct {
		path      string
		forbidden map[string]map[string]struct{}
	}{
		{
			path: filepath.Join(backendRoot, "controller", "generalization_controller.go"),
			forbidden: map[string]map[string]struct{}{
				"service": {"SysTableService": {}},
			},
		},
		{
			path: filepath.Join(backendRoot, "service", "generalization_service.go"),
			forbidden: map[string]map[string]struct{}{
				"repository": {"SysTableRepository": {}, "SysTableFieldRepository": {}},
			},
		},
		{
			path: filepath.Join(backendRoot, "service", "report_service.go"),
			forbidden: map[string]map[string]struct{}{
				"repository": {"SysTableRepository": {}, "SysTableFieldRepository": {}},
			},
		},
		{
			path: filepath.Join(backendRoot, "service", "data_ownership_config_service.go"),
			forbidden: map[string]map[string]struct{}{
				"repository": {"SysTableRepository": {}, "SysTableFieldRepository": {}},
			},
		},
	}

	for _, check := range checks {
		assertFileExcludesSelectors(t, check.path, check.forbidden)
	}

	dataPermissionDir := filepath.Join(backendRoot, "internal", "datapermission")
	entries, err := os.ReadDir(dataPermissionDir)
	if err != nil {
		t.Fatalf("read Data Permission package: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dataPermissionDir, entry.Name())
		file := parseProductionFile(t, path)
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("parse import in %s: %v", path, err)
			}
			if importPath == "backend/repository" || strings.HasPrefix(importPath, "backend/repository/") {
				t.Errorf("Data Permission source %s bypasses runtime metadata boundary", entry.Name())
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok && identifier.Name == "SysTableService" {
				t.Errorf("Data Permission source %s references SysTableService", entry.Name())
			}
			return true
		})
	}
}

func assertFileExcludesSelectors(t *testing.T, path string, forbidden map[string]map[string]struct{}) {
	t.Helper()
	file := parseProductionFile(t, path)
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		qualifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if names := forbidden[qualifier.Name]; names != nil {
			if _, blocked := names[selector.Sel.Name]; blocked {
				t.Errorf("%s directly references configuration boundary %s.%s", path, qualifier.Name, selector.Sel.Name)
			}
		}
		return true
	})
}

func parseProductionFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}
