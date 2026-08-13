package service

import (
	"os"
	"path/filepath"
	"runtime"
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
		forbidden []string
	}{
		{
			path: filepath.Join(backendRoot, "controller", "generalization_controller.go"),
			forbidden: []string{
				"SysTableService",
				"SysTableRepository",
				"SysTableFieldRepository",
			},
		},
		{
			path: filepath.Join(backendRoot, "service", "generalization_service.go"),
			forbidden: []string{
				"SysTableService",
				"repository.SysTableRepository",
				"repository.SysTableFieldRepository",
			},
		},
		{
			path: filepath.Join(backendRoot, "service", "report_service.go"),
			forbidden: []string{
				"SysTableService",
				"repository.SysTableRepository",
				"repository.SysTableFieldRepository",
			},
		},
		{
			path: filepath.Join(backendRoot, "service", "data_ownership_config_service.go"),
			forbidden: []string{
				"repository.SysTableRepository",
				"repository.SysTableFieldRepository",
			},
		},
	}

	for _, check := range checks {
		content, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("read %s: %v", check.path, err)
		}
		for _, forbidden := range check.forbidden {
			if strings.Contains(string(content), forbidden) {
				t.Errorf("%s directly references configuration boundary %q", check.path, forbidden)
			}
		}
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
		content, err := os.ReadFile(filepath.Join(dataPermissionDir, entry.Name()))
		if err != nil {
			t.Fatalf("read Data Permission source: %v", err)
		}
		if strings.Contains(string(content), "backend/repository") || strings.Contains(string(content), "SysTableService") {
			t.Errorf("Data Permission source %s bypasses runtime metadata boundary", entry.Name())
		}
	}
}
