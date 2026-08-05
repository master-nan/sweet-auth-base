package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/asynctask"
	"backend/internal/audit"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ func(*LogService, asynctask.Context, model.LoginLog) = (*LogService).CreateLoginLogAsync

var _ func(
	*SysUserService,
	asynctask.Context,
	int,
	string,
	model.CustomTime,
) = (*SysUserService).UpdateLoginStateAsync

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

func TestLogServiceRecordTransactionalAuditUsesCallerTransactionAndSafeFields(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.AccessLog{})
	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("new snowflake: %v", err)
	}
	logService := NewLogServer(nil, impl.NewAccessLogRepositoryImpl(primaryDB), sf)

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/admin/org/employee/88/bind-user",
		nil,
	)
	ctx.Set("user", model.SysUser{
		Basic:        model.Basic{Id: 42},
		UserName:     "binding_operator",
		Password:     "password-must-not-leak",
		AccessTokens: "token-must-not-leak",
	})
	ctx.Set(transactionalAuditRequestIDContextKey, "request-binding-audit")
	ctx.Set(transactionalAuditTraceIDContextKey, "trace-binding-audit")

	err = RunInTransaction(ctx, db.WithContext(ctx), func(tx *gorm.DB) error {
		return logService.RecordTransactionalAudit(ctx, tx, TransactionalAuditRecord{
			Action:       "bind_user",
			ResourceType: "org_employee",
			ResourceId:   "88",
			Changes: map[string]TransactionalAuditChange{
				"user_id": {OldValue: nil, NewValue: 501},
			},
		})
	})
	if err != nil {
		t.Fatalf("record transactional audit: %v", err)
	}

	var stored model.AccessLog
	if err = db.First(&stored).Error; err != nil {
		t.Fatalf("load transactional audit: %v", err)
	}
	if stored.UserId != 42 ||
		stored.UserName != "binding_operator" ||
		stored.Action != "bind_user" ||
		stored.ResourceType != "org_employee" ||
		stored.ResourceId != "88" ||
		!stored.Success {
		t.Fatalf("unexpected transactional audit: %+v", stored)
	}
	if stored.RequestId == "" || stored.TraceId == "" {
		t.Fatalf("transactional audit lost correlation ids: %+v", stored)
	}
	for _, forbidden := range []string{"password-must-not-leak", "token-must-not-leak", `"roles"`} {
		if strings.Contains(stored.Body, forbidden) {
			t.Fatalf("transactional audit leaked %q: %s", forbidden, stored.Body)
		}
	}
	if !strings.Contains(stored.Body, `"old_value":null`) ||
		!strings.Contains(stored.Body, `"new_value":501`) {
		t.Fatalf("transactional audit lost old/new user_id: %s", stored.Body)
	}
}

func TestLogServiceRecordTransactionalAuditContextKeepsSubjectAndCorrelation(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.AccessLog{})
	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(3)
	if err != nil {
		t.Fatalf("new snowflake: %v", err)
	}
	logService := NewLogServer(nil, impl.NewAccessLogRepositoryImpl(primaryDB), sf)
	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(73, "integration-admin"))
	ctx = audit.WithCorrelationIDs(ctx, audit.CorrelationIDs{RequestID: "request-integration", TraceID: "trace-integration"})
	err = RunInTransaction(ctx, db.WithContext(ctx), func(tx *gorm.DB) error {
		return logService.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
			Action: "integration.credential.rotate", ResourceType: "integration_credential", ResourceCode: "hr_token", ResourceId: "9",
		})
	})
	if err != nil {
		t.Fatalf("record standard-context audit: %v", err)
	}
	var stored model.AccessLog
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("load audit: %v", err)
	}
	if stored.UserId != 73 || stored.UserName != "integration-admin" || stored.RequestId != "request-integration" || stored.TraceId != "trace-integration" {
		t.Fatalf("audit context lost: %+v", stored)
	}
}

func TestLogServiceCreateLoginLogAsyncAfterRequestEnds(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.LoginLog{})
	primaryDB := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(2)
	if err != nil {
		t.Fatalf("new snowflake: %v", err)
	}
	logService := NewLogServer(impl.NewLoginLogRepositoryImpl(primaryDB), nil, sf)
	taskContext := asynctask.New(asynctask.Metadata{
		RequestID: "request-login-log",
		TraceID:   "trace-login-log",
		UserID:    42,
		UserName:  "login-user",
		ClientIP:  "127.0.0.1",
	})

	logService.CreateLoginLogAsync(taskContext, model.LoginLog{
		Ip:       "127.0.0.1",
		UserName: "login-user",
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var stored model.LoginLog
		if err = db.First(&stored).Error; err == nil {
			if stored.UserName != "login-user" || stored.Ip != "127.0.0.1" {
				t.Fatalf("unexpected login log: %+v", stored)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("异步登录日志未在请求结束后写入: %v", err)
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
