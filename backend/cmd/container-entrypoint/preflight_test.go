package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestEnvironmentWithOverridesStrictPreflight(t *testing.T) {
	t.Setenv("APP_DB_PREFLIGHT_REQUIRE_MIGRATED", "false")
	environment := environmentWith("APP_DB_PREFLIGHT_REQUIRE_MIGRATED", "true")
	count := 0
	for _, item := range environment {
		if strings.HasPrefix(item, "APP_DB_PREFLIGHT_REQUIRE_MIGRATED=") {
			count++
			if item != "APP_DB_PREFLIGHT_REQUIRE_MIGRATED=true" {
				t.Fatalf("preflight environment = %q", item)
			}
		}
	}
	if count != 1 {
		t.Fatalf("preflight environment has %d overrides, want 1", count)
	}
}

func TestExecApplicationReplacesEntrypointProcess(t *testing.T) {
	original := syscallExec
	t.Cleanup(func() { syscallExec = original })

	var gotPath string
	var gotArgs, gotEnvironment []string
	syscallExec = func(path string, args []string, environment []string) error {
		gotPath = path
		gotArgs = append([]string(nil), args...)
		gotEnvironment = append([]string(nil), environment...)
		return nil
	}

	environment := []string{"APP_ENV=production"}
	if err := execApplication("/app/sweet_admin", environment); err != nil {
		t.Fatalf("execApplication: %v", err)
	}
	if gotPath != "/app/sweet_admin" || !reflect.DeepEqual(gotArgs, []string{"/app/sweet_admin"}) || !reflect.DeepEqual(gotEnvironment, environment) {
		t.Fatalf("unexpected exec call: path=%q args=%v env=%v", gotPath, gotArgs, gotEnvironment)
	}
}
