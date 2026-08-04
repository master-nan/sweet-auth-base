package service

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/datapermission"
	"backend/model"

	"github.com/gin-gonic/gin"
)

func TestDataPermissionResolverEngineCombinesPolicyRulesWithAnd(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	state.addLegalEntityRule(t, 301, 303)

	result, err := resolver.Resolve(nil, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve multi-Rule Policy: %v", err)
	}
	if result.Decision() != datapermission.DataScopeDecisionFiltered {
		t.Fatalf("decision = %s, want filtered", result.Decision())
	}
	groups := result.ConditionGroups()
	if len(groups) != 1 {
		t.Fatalf("condition group count = %d, want one AND group", len(groups))
	}
	conditions := groups[0].Conditions()
	if len(conditions) != 2 {
		t.Fatalf("condition count = %d, want two AND conditions", len(conditions))
	}
	seen := make(map[string]int)
	for _, condition := range conditions {
		seen[condition.OwnershipCode()] = condition.DimensionId()
	}
	if seen["owner_org"] != 201 || seen["legal_entity"] != 211 {
		t.Fatalf("unexpected AND conditions: %+v", seen)
	}
}

func TestDataPermissionResolverEnginePublishesSafeSummaryAndUsesRequestCache(t *testing.T) {
	resolver, state := newGrantMergeResolver(t)
	state.grants = append(state.grants, grantMergeGrant(411, 7, state.grants[0].PolicyId))
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	result, err := resolver.Resolve(ctx, policyResolverInput(t))
	if err != nil {
		t.Fatalf("resolve cached Grants: %v", err)
	}
	summary, ok := datapermission.ResolverSummaryFromContext(ctx)
	if !ok {
		t.Fatal("Resolver summary missing from request context")
	}
	if summary.ResourceCode() != policyResolverResourceCode ||
		summary.Operation() != model.DataPermissionOperationQuery ||
		summary.Decision() != result.Decision() || summary.CheckedGrantCount() != 2 ||
		summary.CheckedPolicyCount() != 1 {
		t.Fatalf("unexpected Resolver summary: grants=%d policies=%d decision=%s",
			summary.CheckedGrantCount(), summary.CheckedPolicyCount(), summary.Decision())
	}
	if state.resourceCalls != 1 || state.policyCalls[301] != 1 || state.ruleCalls[301] != 1 ||
		len(state.providerCalls) != 1 {
		t.Fatalf(
			"request cache misses: resource=%d policy=%d rules=%d provider=%d",
			state.resourceCalls,
			state.policyCalls[301],
			state.ruleCalls[301],
			len(state.providerCalls),
		)
	}

	payload, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal Resolver summary: %v", err)
	}
	encoded := string(payload)
	for _, forbidden := range []string{"policy_id", "grant_id", "role_ids", "subject_id", "sql", "table"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("Resolver summary leaks %q: %s", forbidden, encoded)
		}
	}
}

func (state *grantMergeResolverState) addLegalEntityRule(
	t *testing.T,
	policyId int,
	ruleId int,
) {
	t.Helper()
	state.dimensions[211] = policyResolverDimension(211, datapermission.DimensionCodeLegalEntity)
	state.ownerships["legal_entity"] = grantMergeOwnership(212, 211, "legal_entity")
	rule := grantMergeRule(
		ruleId,
		policyId,
		211,
		"legal_entity",
		model.DataPolicyScopeSourceEffectiveLegalEntities,
	)
	rule.Sequence = len(state.rules[policyId]) + 1
	state.rules[policyId] = append(state.rules[policyId], rule)
	values, err := datapermission.NewDimensionValues(
		datapermission.DimensionCodeLegalEntity,
		datapermission.DataScopeValueTypeBigint,
		[]any{int64(21)},
	)
	if err != nil {
		t.Fatalf("create legal entity Provider values: %v", err)
	}
	state.providerValues[datapermission.DimensionCodeLegalEntity] = values
}
