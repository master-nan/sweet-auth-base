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

// AssertPermissions verifies both allowed and denied Casbin paths against the
// same enforcer used by the caller's permission test.
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
