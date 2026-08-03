package service

import (
	"testing"

	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
)

// The demo policy deliberately keeps the reviewed self-and-descendants
// relation. Until Resolver expansion is implemented, it must fail closed
// instead of silently degrading to exact or all access.
func TestDataPermissionDemoAcceptanceDescendantPolicyFailsClosed(t *testing.T) {
	resolver, db, provider, fixtures := newPolicyResolverTestSubject(t)
	structureCode := "DP-ACCEPTANCE-MGMT"
	if err := db.Model(&model.DataPolicyRule{}).
		Where("id = ?", fixtures.rule.Id).
		Updates(map[string]any{
			"relation":       model.DataPolicyRelationSelfAndDescendants,
			"structure_code": structureCode,
		}).Error; err != nil {
		t.Fatalf("configure acceptance descendant Rule: %v", err)
	}

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	assertPolicyResolverError(
		t,
		err,
		myerrors.ErrorCodeDataPermissionResolverConfigConflict,
	)
	assertPolicyResolverNoAccess(t, result)
	if result.Decision() == datapermission.DataScopeDecisionAll {
		t.Fatal("unsupported descendant relation expanded access to all data")
	}
	if provider.calls != 0 {
		t.Fatalf("Provider calls = %d, want 0 after relation validation failure", provider.calls)
	}
}
