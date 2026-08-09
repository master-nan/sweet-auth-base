package service

import (
	"backend/dto/request"
	"backend/enum"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestRetryPolicyServiceLifecycleVersioningAndAudit(t *testing.T) {
	writer := &externalSystemAuditWriter{}
	svc, db := newRetryPolicyTestSubject(t, writer)
	ctx := context.Background()

	created, err := svc.CreateRetryPolicy(ctx, request.RetryPolicyCreateReq{PolicyCode: "hr_retry", PolicyName: "HR 重试"})
	if err != nil {
		t.Fatalf("create retry policy: %v", err)
	}
	if created.Status != model.RetryPolicyStatusDraft || created.Version != 1 || created.Revision != 1 || created.MaxAttempts != 3 {
		t.Fatalf("unexpected draft: %+v", created)
	}
	name := "HR 稳定重试"
	updated, err := svc.UpdateDraftRetryPolicy(ctx, created.Id, request.RetryPolicyUpdateReq{PolicyName: &name, Revision: created.Revision})
	if err != nil || updated.PolicyName != name || updated.Revision != 2 {
		t.Fatalf("update draft=%+v err=%v", updated, err)
	}
	enabled, err := svc.EnableRetryPolicy(ctx, created.Id, updated.Revision)
	if err != nil || enabled.Status != model.RetryPolicyStatusEnabled || enabled.Revision != 3 {
		t.Fatalf("enable policy=%+v err=%v", enabled, err)
	}
	if _, err := svc.UpdateDraftRetryPolicy(ctx, enabled.Id, request.RetryPolicyUpdateReq{PolicyName: &name, Revision: enabled.Revision}); !errors.Is(err, apperrors.ErrRetryPolicyFieldImmutable) {
		t.Fatalf("enabled update error=%v", err)
	}
	v2, err := svc.CreateRetryPolicyVersion(ctx, enabled.Id, enabled.Revision)
	if err != nil || v2.Version != 2 || v2.Status != model.RetryPolicyStatusDraft || v2.Revision != 1 {
		t.Fatalf("create version=%+v err=%v", v2, err)
	}
	if _, err := svc.EnableRetryPolicy(ctx, v2.Id, v2.Revision); !errors.Is(err, apperrors.ErrRetryPolicyEnabledConflict) {
		t.Fatalf("single-enabled conflict=%v", err)
	}
	disabled, err := svc.DisableRetryPolicy(ctx, enabled.Id, enabled.Revision)
	if err != nil || disabled.Status != model.RetryPolicyStatusDisabled {
		t.Fatalf("disable v1=%+v err=%v", disabled, err)
	}
	enabledV2, err := svc.EnableRetryPolicy(ctx, v2.Id, v2.Revision)
	if err != nil || enabledV2.Status != model.RetryPolicyStatusEnabled {
		t.Fatalf("enable v2=%+v err=%v", enabledV2, err)
	}

	var count int64
	if err := db.Model(&model.RetryPolicy{}).Where("policy_code = ?", "hr_retry").Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("version count=%d err=%v", count, err)
	}
	if len(writer.records) != 6 {
		t.Fatalf("audit count=%d want 6", len(writer.records))
	}
}

func TestRetryPolicyServiceValidationWhitelistRevisionAndDTO(t *testing.T) {
	svc, _ := newRetryPolicyTestSubject(t, &externalSystemAuditWriter{})
	ctx := context.Background()
	zero := 0
	badDelay := int64(999)
	badMultiplier := 1.0
	badRatio := 0.5
	tenAttempts := 10
	oneHour := int64(3_600_000)
	twoHours := int64(7_200_000)
	tests := []struct {
		name string
		req  request.RetryPolicyCreateReq
	}{
		{name: "attempts", req: request.RetryPolicyCreateReq{PolicyCode: "bad_attempt", PolicyName: "bad", MaxAttempts: &zero}},
		{name: "delay", req: request.RetryPolicyCreateReq{PolicyCode: "bad_delay", PolicyName: "bad", InitialDelayMs: &badDelay}},
		{name: "multiplier", req: request.RetryPolicyCreateReq{PolicyCode: "bad_multiplier", PolicyName: "bad", BackoffType: model.RetryBackoffTypeExponential, BackoffMultiplier: &badMultiplier}},
		{name: "jitter", req: request.RetryPolicyCreateReq{PolicyCode: "bad_jitter", PolicyName: "bad", JitterType: model.RetryJitterTypeFull, JitterRatio: &badRatio}},
		{name: "category", req: request.RetryPolicyCreateReq{PolicyCode: "bad_category", PolicyName: "bad", RetryableErrorCategories: []string{"credential"}}},
		{name: "http", req: request.RetryPolicyCreateReq{PolicyCode: "bad_http", PolicyName: "bad", RetryableHTTPStatuses: []int{500}}},
		{name: "window cannot cover attempts", req: request.RetryPolicyCreateReq{PolicyCode: "bad_window", PolicyName: "bad", MaxAttempts: &tenAttempts, InitialDelayMs: &oneHour, MaxDelayMs: &oneHour, BackoffType: model.RetryBackoffTypeFixed, RetryWindowMs: &twoHours}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.CreateRetryPolicy(ctx, tt.req); !errors.Is(err, apperrors.ErrRetryPolicyConfigurationInvalid) {
				t.Fatalf("validation error=%v", err)
			}
		})
	}
	created, err := svc.CreateRetryPolicy(ctx, request.RetryPolicyCreateReq{PolicyCode: "a", PolicyName: "单字符编码"})
	if err != nil {
		t.Fatalf("one-character stable code: %v", err)
	}
	if _, err := svc.UpdateDraftRetryPolicy(ctx, created.Id, request.RetryPolicyUpdateReq{Revision: created.Revision + 1}); !errors.Is(err, apperrors.ErrRetryPolicyRevisionConflict) {
		t.Fatalf("revision error=%v", err)
	}
	payload, err := json.Marshal(created)
	if err != nil {
		t.Fatalf("marshal DTO: %v", err)
	}
	for _, forbidden := range []string{"gmt_delete", "delete_user", "create_user", "state\"", "retry_policy_snapshot"} {
		if strings.Contains(strings.ToLower(string(payload)), forbidden) {
			t.Fatalf("detail DTO leaked %q: %s", forbidden, payload)
		}
	}
}

func TestRetryPolicyServiceReferenceProtectionAndPage(t *testing.T) {
	svc, db := newRetryPolicyTestSubject(t, &externalSystemAuditWriter{})
	created, err := svc.CreateRetryPolicy(context.Background(), request.RetryPolicyCreateReq{PolicyCode: "order_retry", PolicyName: "订单重试"})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	enabled, err := svc.EnableRetryPolicy(context.Background(), created.Id, created.Revision)
	if err != nil {
		t.Fatalf("enable policy: %v", err)
	}
	system := model.ExternalSystem{Basic: model.Basic{Id: 801, State: true}, SystemCode: "retry_erp", Name: "Retry ERP", SystemType: model.ExternalSystemTypeERP, BaseURL: "https://example.com", OwnerIdentifier: "owner", OwnerName: "owner", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	definition := model.InterfaceDefinition{Basic: model.Basic{Id: 802, State: true}, ExternalSystemID: system.Id, InterfaceCode: "orders", Name: "Orders", Version: 1, Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET, RelativePath: "/orders", TimeoutSeconds: 30, ResponseLimit: 1024, RetryPolicyID: &enabled.Id, Status: model.InterfaceDefinitionStatusEnabled, Revision: 1}
	testutil.MustCreate(t, db, &system)
	testutil.MustCreate(t, db, &definition)
	if _, err := svc.DisableRetryPolicy(context.Background(), enabled.Id, enabled.Revision); !errors.Is(err, apperrors.ErrRetryPolicyReferenced) {
		t.Fatalf("referenced disable error=%v", err)
	}
	page, err := svc.PageRetryPolicy(context.Background(), request.RetryPolicyQueryReq{Page: 1, Num: 10, QuickQuery: &request.QuickQuery{Keyword: "order"}, Status: model.RetryPolicyStatusEnabled}, retryPolicyQueryTableForTest())
	if err != nil || page.Total != 1 || len(page.Data) != 1 || page.Data[0].PolicyCode != "order_retry" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}

func newRetryPolicyTestSubject(t *testing.T, writer StandardContextAuditWriter) (*RetryPolicyService, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.RetryPolicy{}, &model.ExternalSystem{}, &model.InterfaceDefinition{})
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	return NewRetryPolicyService(impl.NewRetryPolicyRepositoryImpl(&database.PrimaryDB{DB: db}), sf, writer), db
}

func retryPolicyQueryTableForTest() model.SysTable {
	return model.SysTable{TableCode: "integration_retry_policy", TableFields: []model.SysTableField{
		{Basic: model.Basic{State: true}, FieldCode: "policy_code", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "policy_name", FieldType: enum.VarcharFieldType, IsQuickSearch: true},
		{Basic: model.Basic{State: true}, FieldCode: "status", FieldType: enum.VarcharFieldType},
		{Basic: model.Basic{State: true}, FieldCode: "backoff_type", FieldType: enum.VarcharFieldType},
	}}
}
