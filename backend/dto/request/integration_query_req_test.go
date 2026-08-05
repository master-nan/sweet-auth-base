package request

import (
	"backend/enum"
	"testing"
)

func TestExternalSystemQueryReqToBasicBuildsControlledFilters(t *testing.T) {
	basic := (ExternalSystemQueryReq{
		SystemType: "erp",
		Status:     "enabled",
		Owner:      " owner-1 ",
	}).ToBasic()

	if basic.Filters["system_type"] != "erp" || basic.Filters["status"] != "enabled" {
		t.Fatalf("unexpected filters: %+v", basic.Filters)
	}
	if len(basic.Expressions) != 1 || basic.Expressions[0].Logic != enum.Or || len(basic.Expressions[0].Rules) != 2 {
		t.Fatalf("unexpected owner expression: %+v", basic.Expressions)
	}
	for _, rule := range basic.Expressions[0].Rules {
		if rule.ExpressionType != enum.Like || rule.Value != "owner-1" {
			t.Fatalf("unexpected owner rule: %+v", rule)
		}
	}
}

func TestInterfaceDefinitionQueryReqToBasicBuildsControlledFilters(t *testing.T) {
	basic := (InterfaceDefinitionQueryReq{
		ExternalSystemID: 10,
		HTTPMethod:       "POST",
		Status:           "draft",
	}).ToBasic()

	if basic.Filters["external_system_id"] != 10 ||
		basic.Filters["http_method"] != "POST" ||
		basic.Filters["status"] != "draft" {
		t.Fatalf("unexpected filters: %+v", basic.Filters)
	}
}

func TestCredentialQueryReqToBasicBuildsExpiredExpression(t *testing.T) {
	basic := (CredentialQueryReq{
		ExternalSystemID: 10,
		CredentialType:   "api_key",
		Status:           "expired",
	}).ToBasic()

	if basic.Filters["external_system_id"] != 10 || basic.Filters["credential_type"] != "api_key" {
		t.Fatalf("unexpected filters: %+v", basic.Filters)
	}
	if _, exists := basic.Filters["status"]; exists {
		t.Fatalf("computed expired status must not be persisted filter: %+v", basic.Filters)
	}
	if len(basic.Expressions) != 1 || basic.Expressions[0].Logic != enum.And || len(basic.Expressions[0].Rules) != 2 {
		t.Fatalf("unexpected expired expression: %+v", basic.Expressions)
	}
	if basic.Expressions[0].Rules[0].Field != "status" || basic.Expressions[0].Rules[1].Field != "expires_at" {
		t.Fatalf("unexpected expired rules: %+v", basic.Expressions[0].Rules)
	}
}
