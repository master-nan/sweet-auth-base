package testutil

import "testing"

type PermissionEnforcer interface {
	Enforce(rvals ...interface{}) (bool, error)
}

type PermissionCase struct {
	Name    string
	Subject string
	Path    string
	Method  string
	Allowed bool
}

// AssertPermissions 使用调用方权限测试所用的同一执行器，校验 Casbin 允许和拒绝路径。
func AssertPermissions(t testing.TB, enforcer PermissionEnforcer, cases ...PermissionCase) {
	t.Helper()
	if enforcer == nil {
		t.Fatal("permission enforcer is required")
	}
	for _, item := range cases {
		allowed, err := enforcer.Enforce(item.Subject, item.Path, item.Method)
		if err != nil {
			t.Fatalf("%s: enforce permission: %v", permissionCaseName(item), err)
		}
		if allowed != item.Allowed {
			t.Errorf(
				"%s: allowed = %t, want %t",
				permissionCaseName(item),
				allowed,
				item.Allowed,
			)
		}
	}
}

func permissionCaseName(item PermissionCase) string {
	if item.Name != "" {
		return item.Name
	}
	return item.Subject + " " + item.Method + " " + item.Path
}
