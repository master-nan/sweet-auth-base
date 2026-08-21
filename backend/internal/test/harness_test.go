package testutil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type harnessFixture struct {
	ID   int    `gorm:"primaryKey"`
	Code string `gorm:"size:64;uniqueIndex"`
}

func TestOpenSQLiteMigratesAndIsolatesFixtures(t *testing.T) {
	first := OpenSQLite(t, &harnessFixture{})
	second := OpenSQLite(t, &harnessFixture{})

	MustCreate(t, first, &harnessFixture{ID: 1, Code: "same-key"})
	MustCreate(t, second, &harnessFixture{ID: 1, Code: "same-key"})

	if got := fixtureCount(t, first); got != 1 {
		t.Fatalf("first database fixture count = %d, want 1", got)
	}
	if got := fixtureCount(t, second); got != 1 {
		t.Fatalf("second database fixture count = %d, want 1", got)
	}
}

func TestPerformRequestPreservesHeadersAndBody(t *testing.T) {
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Test-Context") != "baseline" {
			t.Errorf("request header was not propagated")
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if string(body) != `{"name":"fixture"}` {
			t.Errorf("request body = %q", body)
		}
		writer.WriteHeader(http.StatusCreated)
	})

	recorder := PerformRequest(t, handler, HTTPRequest{
		Method: http.MethodPost,
		Target: "/fixtures",
		Body:   strings.NewReader(`{"name":"fixture"}`),
		Header: http.Header{"X-Test-Context": []string{"baseline"}},
	})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("response status = %d, want %d", recorder.Code, http.StatusCreated)
	}
}

func TestAssertPermissionsCoversAllowedAndDeniedCases(t *testing.T) {
	enforcer := permissionFixtureEnforcer{
		"operator|/admin/fixtures|POST": true,
	}

	AssertPermissions(
		t,
		enforcer,
		PermissionCase{
			Name:    "granted button",
			Subject: "operator",
			Path:    "/admin/fixtures",
			Method:  http.MethodPost,
			Allowed: true,
		},
		PermissionCase{
			Name:    "missing button",
			Subject: "viewer",
			Path:    "/admin/fixtures",
			Method:  http.MethodPost,
			Allowed: false,
		},
	)
}

func fixtureCount(t testing.TB, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&harnessFixture{}).Count(&count).Error; err != nil {
		t.Fatalf("count test fixtures: %v", err)
	}
	return count
}

type permissionFixtureEnforcer map[string]bool

var _ PermissionEnforcer = (*casbin.Enforcer)(nil)

func (enforcer permissionFixtureEnforcer) Enforce(values ...interface{}) (bool, error) {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, value.(string))
	}
	return enforcer[strings.Join(parts, "|")], nil
}
