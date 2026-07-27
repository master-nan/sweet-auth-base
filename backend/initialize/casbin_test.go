package initialize

import (
	"errors"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

type failingCasbinAdapter struct {
	err error
}

func (adapter failingCasbinAdapter) LoadPolicy(model.Model) error {
	return adapter.err
}

func (failingCasbinAdapter) SavePolicy(model.Model) error {
	return nil
}

func (failingCasbinAdapter) AddPolicy(string, string, []string) error {
	return nil
}

func (failingCasbinAdapter) RemovePolicy(string, string, []string) error {
	return nil
}

func (failingCasbinAdapter) RemoveFilteredPolicy(string, string, int, ...string) error {
	return nil
}

func TestSyncedEnforcerLoadFailureKeepsLastValidPolicy(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer("../casbin_model.conf")
	if err != nil {
		t.Fatalf("new synced enforcer: %v", err)
	}
	if _, err = enforcer.AddPolicy("stable_role", "/admin/stable", "GET"); err != nil {
		t.Fatalf("add stable policy: %v", err)
	}

	loadErr := errors.New("adapter unavailable")
	enforcer.SetAdapter(failingCasbinAdapter{err: loadErr})
	if err = enforcer.LoadPolicy(); !errors.Is(err, loadErr) {
		t.Fatalf("load policy error=%v, want %v", err, loadErr)
	}

	allowed, err := enforcer.Enforce("stable_role", "/admin/stable", "GET")
	if err != nil {
		t.Fatalf("enforce retained policy: %v", err)
	}
	if !allowed {
		t.Fatal("failed policy reload replaced the last valid policy")
	}
}
