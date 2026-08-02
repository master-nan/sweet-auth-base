package datapermission

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolverSummaryUsesWhitelistSerializationAndRequestContext(t *testing.T) {
	summary, err := NewResolverSummary(ResolverSummaryInput{
		ResourceCode:       "transport_order",
		Operation:          "query",
		Decision:           DataScopeDecisionFiltered,
		CheckedGrantCount:  3,
		CheckedPolicyCount: 2,
	})
	if err != nil {
		t.Fatalf("create Resolver summary: %v", err)
	}

	payload, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal Resolver summary: %v", err)
	}
	encoded := string(payload)
	for _, forbidden := range []string{"policy_id", "grant_id", "role_ids", "sql", "table", "field"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("summary leaks %q: %s", forbidden, encoded)
		}
	}

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	if err = StoreResolverSummary(ctx, summary); err != nil {
		t.Fatalf("store Resolver summary: %v", err)
	}
	stored, ok := ResolverSummaryFromContext(ctx)
	if !ok {
		t.Fatal("Resolver summary was not stored in request context")
	}
	if stored.ResourceCode() != "transport_order" || stored.Operation() != "query" ||
		stored.Decision() != DataScopeDecisionFiltered || stored.CheckedGrantCount() != 3 ||
		stored.CheckedPolicyCount() != 2 {
		t.Fatalf("unexpected stored summary: %s", encoded)
	}
}

func TestResolverSummaryRejectsInvalidCounts(t *testing.T) {
	_, err := NewResolverSummary(ResolverSummaryInput{
		ResourceCode:       "transport_order",
		Operation:          "query",
		Decision:           DataScopeDecisionNone,
		CheckedGrantCount:  1,
		CheckedPolicyCount: 2,
	})
	if err == nil {
		t.Fatal("expected invalid Resolver summary counts to fail")
	}
}
