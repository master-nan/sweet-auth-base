package errors

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

func TestErrorPackageUsesStableDomainFiles(t *testing.T) {
	allowed := map[string]struct{}{
		"auth.go": {}, "common.go": {}, "data_permission.go": {}, "errors.go": {},
		"file.go": {}, "integration.go": {}, "metadata.go": {}, "organization.go": {},
		"query_scheme.go": {},
	}
	packages, err := parser.ParseDir(token.NewFileSet(), ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	production := packages["errors"].Files
	for path, file := range production {
		name := path[strings.LastIndex(path, "/")+1:]
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			t.Fatalf("error definitions must use stable domain files, found %q", name)
		}
		for _, imported := range file.Imports {
			value, _ := strconv.Unquote(imported.Path.Value)
			if value == "net/http" || value == "backend/dto/response" {
				t.Fatalf("application errors must not depend on HTTP adapter package %q in %s", value, name)
			}
		}
	}
}

func TestErrorCodesAreUnique(t *testing.T) {
	packages, err := parser.ParseDir(token.NewFileSet(), ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	constantValues := make(map[string]int)
	seenConstants := make(map[int]string)
	for _, file := range packages["errors"].Files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, spec := range general.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				for index, name := range valueSpec.Names {
					if !strings.HasPrefix(name.Name, "ErrorCode") || index >= len(valueSpec.Values) {
						continue
					}
					literal, ok := valueSpec.Values[index].(*ast.BasicLit)
					if !ok || literal.Kind != token.INT {
						continue
					}
					code, parseErr := strconv.Atoi(literal.Value)
					if parseErr != nil {
						t.Fatalf("parse %s: %v", name.Name, parseErr)
					}
					if previous, exists := seenConstants[code]; exists {
						t.Fatalf("duplicate error code %d: %s and %s", code, previous, name.Name)
					}
					seenConstants[code] = name.Name
					constantValues[name.Name] = code
				}
			}
		}
	}

	seenErrors := make(map[int]string)
	for _, file := range packages["errors"].Files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, spec := range general.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				for index, name := range valueSpec.Names {
					if !strings.HasPrefix(name.Name, "Err") || index >= len(valueSpec.Values) {
						continue
					}
					call, ok := valueSpec.Values[index].(*ast.CallExpr)
					if !ok || len(call.Args) < 3 {
						continue
					}
					codeName, named := call.Args[2].(*ast.Ident)
					if !named || !strings.HasPrefix(codeName.Name, "ErrorCode") {
						t.Fatalf("stable error %s must use a named ErrorCode constant", name.Name)
					}
					code, ok := errorCodeValue(call.Args[2], constantValues)
					if !ok {
						continue
					}
					if previous, exists := seenErrors[code]; exists {
						t.Fatalf("duplicate stable error code %d: %s and %s", code, previous, name.Name)
					}
					seenErrors[code] = name.Name
				}
			}
		}
	}
}

func errorCodeValue(expression ast.Expr, constants map[string]int) (int, bool) {
	switch value := expression.(type) {
	case *ast.BasicLit:
		code, err := strconv.Atoi(value.Value)
		return code, err == nil
	case *ast.Ident:
		code, ok := constants[value.Name]
		return code, ok
	default:
		return 0, false
	}
}
