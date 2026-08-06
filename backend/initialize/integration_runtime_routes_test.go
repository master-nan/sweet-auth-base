package initialize

import (
	"backend/config"
	"testing"
)

func TestIntegrationRuntimeRoutesDoNotExposeEngineStateCommands(t *testing.T) {
	router := InitRouter(&App{Config: &config.Server{}})
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, forbidden := range []string{
		"PUT /admin/integration/execution/:id/start",
		"PUT /admin/integration/execution/:id/complete",
		"PUT /admin/integration/execution/:id/fail",
	} {
		if _, exists := routes[forbidden]; exists {
			t.Fatalf("forbidden management route is registered: %s", forbidden)
		}
	}
	for _, required := range []string{
		"POST /admin/integration/execution/query",
		"GET /admin/integration/execution/:id",
		"POST /admin/integration/execution",
		"PUT /admin/integration/execution/:id/cancel",
		"POST /admin/integration/log/query",
		"GET /admin/integration/log/:id",
	} {
		if _, exists := routes[required]; !exists {
			t.Fatalf("required management route is missing: %s", required)
		}
	}
}
