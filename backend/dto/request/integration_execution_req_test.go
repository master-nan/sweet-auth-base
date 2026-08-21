package request

import (
	"backend/enum"
	"testing"
	"time"
)

func TestIntegrationExecutionQueryReqBuildsControlledFiltersAndTimeRange(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	basic := (IntegrationExecutionQueryReq{
		ExternalSystemID: 10, InterfaceDefinitionID: 20,
		TriggerSource: "manual", Status: "created", CreatedFrom: &from, CreatedTo: &to,
		Expressions: []ExpressionGroup{{Logic: enum.And, Rules: []QueryRule{{
			Field: "status", ExpressionType: enum.Eq, Value: "running", Type: enum.VarcharFieldType,
		}}}},
	}).ToBasic()
	if len(basic.Filters) != 4 || len(basic.Expressions) != 2 || len(basic.Expressions[1].Rules) != 2 {
		t.Fatalf("unexpected basic query: %+v", basic)
	}
	if basic.Expressions[0].Rules[0].Field != "status" || basic.Expressions[1].Logic != enum.And ||
		basic.Expressions[1].Rules[0].ExpressionType != enum.Gte ||
		basic.Expressions[1].Rules[1].ExpressionType != enum.Lte {
		t.Fatalf("unexpected expression rules: %+v", basic.Expressions)
	}
}

func TestIntegrationLogQueryReqPreservesAdvancedExpressions(t *testing.T) {
	expressions := []ExpressionGroup{{Logic: enum.And, Rules: []QueryRule{{
		Field: "status", ExpressionType: enum.Eq, Value: "failed", Type: enum.VarcharFieldType,
	}}}}
	basic := (IntegrationLogQueryReq{Expressions: expressions}).ToBasic()

	if len(basic.Expressions) != 1 || basic.Expressions[0].Rules[0].Value != "failed" {
		t.Fatalf("advanced expressions were not preserved: %+v", basic.Expressions)
	}
}
