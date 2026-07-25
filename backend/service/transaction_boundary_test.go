package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestControllerLayerDoesNotOwnDatabaseTransactions(t *testing.T) {
	controllerDir := filepath.Join("..", "controller")
	entries, err := os.ReadDir(controllerDir)
	if err != nil {
		t.Fatalf("read controller directory: %v", err)
	}

	forbiddenImports := map[string]struct{}{
		"backend/internal/database": {},
		"gorm.io/gorm":              {},
	}
	transactionHelpers := map[string]struct{}{
		"DBWithContext":    {},
		"ExecuteTx":        {},
		"RunInTransaction": {},
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(controllerDir, entry.Name())
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse controller imports %s: %v", path, err)
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("read import in %s: %v", path, err)
			}
			if _, forbidden := forbiddenImports[importPath]; forbidden {
				t.Errorf("controller must not import database infrastructure %q: %s", importPath, path)
			}
		}

		file, err = parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("parse controller %s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if _, forbidden := transactionHelpers[selector.Sel.Name]; forbidden {
				position := fileSet.Position(selector.Pos())
				t.Errorf(
					"controller must not call transaction helper %s at %s",
					selector.Sel.Name,
					position,
				)
			}
			return true
		})
	}
}
