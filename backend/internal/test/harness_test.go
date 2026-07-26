package testutil

import (
	"io"
	"net/http"
	"sort"
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

func TestWithRollbackRemovesFixtureChanges(t *testing.T) {
	db := OpenSQLite(t, &harnessFixture{})
	MustCreate(t, db, &harnessFixture{ID: 1, Code: "persistent"})

	WithRollback(t, db, func(tx *gorm.DB) {
		MustCreate(t, tx, &harnessFixture{ID: 2, Code: "temporary"})
		if got := fixtureCount(t, tx); got != 2 {
			t.Fatalf("transaction fixture count = %d, want 2", got)
		}
	})

	if got := fixtureCount(t, db); got != 1 {
		t.Fatalf("fixture count after rollback = %d, want 1", got)
	}
}

func TestAssertIdempotentComparesStableSnapshots(t *testing.T) {
	runCount := 0
	values := map[string]struct{}{}

	AssertIdempotent(
		t,
		func() error {
			runCount++
			values["stable-key"] = struct{}{}
			return nil
		},
		func() ([]string, error) {
			snapshot := make([]string, 0, len(values))
			for value := range values {
				snapshot = append(snapshot, value)
			}
			sort.Strings(snapshot)
			return snapshot, nil
		},
	)

	if runCount != 2 {
		t.Fatalf("operation run count = %d, want 2", runCount)
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

func TestNewHTTPServerSupportsOutboundMock(t *testing.T) {
	server := NewHTTPServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true}`))
	}))

	response, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("call outbound mock server: %v", err)
	}
	t.Cleanup(func() {
		_ = response.Body.Close()
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("mock response status = %d, want %d", response.StatusCode, http.StatusOK)
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
