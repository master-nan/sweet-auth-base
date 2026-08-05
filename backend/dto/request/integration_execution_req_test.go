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
	}).ToBasic()
	if len(basic.Filters) != 4 || len(basic.Expressions) != 1 || len(basic.Expressions[0].Rules) != 2 {
		t.Fatalf("unexpected basic query: %+v", basic)
	}
	if basic.Expressions[0].Logic != enum.And ||
		basic.Expressions[0].Rules[0].ExpressionType != enum.Gte ||
		basic.Expressions[0].Rules[1].ExpressionType != enum.Lte {
		t.Fatalf("unexpected time rules: %+v", basic.Expressions[0])
	}
}
