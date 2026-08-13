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
	for _, name := range []string{
		"application_service.go",
		"sys_configure_service.go",
		"sys_dict_service.go",
		"sys_menu_service.go",
		"sys_role_service.go",
		"sys_table_service.go",
		"sys_user_service.go",
		"sms_service.go",
		"org_service.go",
		"org_permission_provider.go",
		"org_permission_tree_provider.go",
		"file_upload_service.go",
		"file_access_service.go",
		"file_metadata_service.go",
		"high_risk_response.go",
		"log_service.go",
	} {
		assertFileExcludesGin(t, filepath.Join(root, "service", name))
	}
	assertFileExcludesGin(t, filepath.Join(root, "model", "basic.go"))
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
