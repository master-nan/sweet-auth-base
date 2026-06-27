package service

import (
	"backend/dto/request"
	"backend/enum"
	"testing"
)

func TestParseAccessLogQueryTime(t *testing.T) {
	parsed, err := parseAccessLogQueryTime("2026-06-04 14:30:00")
	if err != nil {
		t.Fatalf("expected valid datetime to parse: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed datetime")
	}

	dateOnly, err := parseAccessLogQueryTime("2026-06-04")
	if err != nil {
		t.Fatalf("expected valid date to parse: %v", err)
	}
	if dateOnly == nil {
		t.Fatal("expected parsed date")
	}

	if _, err := parseAccessLogQueryTime("not-a-date"); err == nil {
		t.Fatal("expected invalid date to be rejected")
	}
}

func TestBuildAccessLogQueryBasic(t *testing.T) {
	success := true
	basic, err := buildAccessLogQueryBasic(requestAccessLogQueryReqForTest(success))
	if err != nil {
		t.Fatalf("expected query to build: %v", err)
	}
	if basic.Page != 2 || basic.Num != 30 {
		t.Fatalf("unexpected pagination: page=%d num=%d", basic.Page, basic.Num)
	}
	if basic.Order.Field != "gmt_create" || basic.Order.IsAsc {
		t.Fatalf("unexpected default order: %+v", basic.Order)
	}
	if len(basic.Expressions) != 1 {
		t.Fatalf("expected one expression group, got %d", len(basic.Expressions))
	}
	rules := basic.Expressions[0].Rules
	expected := map[string]enum.ExpressionType{
		"user_name":     enum.Like,
		"action":        enum.Eq,
		"resource_code": enum.Like,
		"method":        enum.Eq,
		"url":           enum.Like,
		"ip":            enum.Like,
		"success":       enum.Eq,
	}
	for field, expressionType := range expected {
		rule, ok := findAccessLogRule(rules, field, expressionType)
		if !ok {
			t.Fatalf("missing rule %s/%d in %+v", field, expressionType, rules)
		}
		if field == "method" && rule.Value != "POST" {
			t.Fatalf("expected method to be normalized to POST, got %#v", rule.Value)
		}
	}
	if _, ok := findAccessLogRule(rules, "gmt_create", enum.Gte); !ok {
		t.Fatal("missing start time rule")
	}
	if _, ok := findAccessLogRule(rules, "gmt_create", enum.Lte); !ok {
		t.Fatal("missing end time rule")
	}
}

func TestBuildAccessLogQueryBasicWithoutFilters(t *testing.T) {
	basic, err := buildAccessLogQueryBasic(request.AccessLogQueryReq{})
	if err != nil {
		t.Fatalf("expected empty query to build: %v", err)
	}
	if len(basic.Expressions) != 0 {
		t.Fatalf("expected empty expressions, got %+v", basic.Expressions)
	}
}

func TestBuildAccessLogQueryBasicRejectsReversedRange(t *testing.T) {
	_, err := buildAccessLogQueryBasic(request.AccessLogQueryReq{
		StartTime: "2026-06-05 00:00:00",
		EndTime:   "2026-06-04 00:00:00",
	})
	if err == nil {
		t.Fatal("expected reversed range to be rejected")
	}
}

func requestAccessLogQueryReqForTest(success bool) request.AccessLogQueryReq {
	return request.AccessLogQueryReq{
		Basic:        request.Basic{Page: 2, Num: 30},
		UserName:     " admin ",
		Action:       "lowcode_create",
		ResourceCode: "smk_",
		Method:       " post ",
		Url:          "/admin/generalization",
		Ip:           "127.0.0.1",
		Success:      &success,
		StartTime:    "2026-06-04 00:00:00",
		EndTime:      "2026-06-05 00:00:00",
	}
}

func findAccessLogRule(rules []request.QueryRule, field string, expressionType enum.ExpressionType) (request.QueryRule, bool) {
	for _, rule := range rules {
		if rule.Field == field && rule.ExpressionType == expressionType {
			return rule, true
		}
	}
	return request.QueryRule{}, false
}
