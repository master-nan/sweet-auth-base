package service

import (
	"backend/internal/audit"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"strings"
	"testing"
)

func TestAuthAuditPersistsSafeStructuredContext(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.AccessLog{}, &model.LoginLog{})
	primary := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(4)
	if err != nil {
		t.Fatal(err)
	}
	recorder := NewAuthAuditService(NewLogServer(
		impl.NewLoginLogRepositoryImpl(primary),
		impl.NewAccessLogRepositoryImpl(primary),
		sf,
	))
	ctx := audit.WithCorrelationIDs(context.Background(), audit.CorrelationIDs{RequestID: "request-auth", TraceID: "trace-auth"})
	ctx = audit.WithRequestMetadata(ctx, audit.RequestMetadata{Method: "POST", Path: "/admin/login", ClientIP: "127.0.0.1", UserAgent: "test-agent"})
	ctx = audit.WithAccessAuditState(ctx)
	if err := recorder.RecordAuthEvent(ctx, AuthAuditEvent{
		Channel: AuthChannelAdminPassword, CredentialType: AuthCredentialPassword,
		Success: false, ReasonCode: "credential_invalid", Principal: "unknown@example.test",
	}); err != nil {
		t.Fatal(err)
	}
	var access model.AccessLog
	if err := db.First(&access).Error; err != nil {
		t.Fatal(err)
	}
	var login model.LoginLog
	if err := db.First(&login).Error; err != nil {
		t.Fatal(err)
	}
	if access.RequestId != "request-auth" || access.TraceId != "trace-auth" || access.ErrorCode != "credential_invalid" || access.Success {
		t.Fatalf("unexpected authentication audit: %+v", access)
	}
	for _, stored := range []string{access.UserName, access.ResourceId, login.UserName, access.Body} {
		if strings.Contains(stored, "unknown@example.test") {
			t.Fatalf("authentication audit leaked unknown principal: %s", stored)
		}
	}
	if !strings.Contains(access.Body, "test-agent") || strings.Contains(access.Body, "very-secret-password") {
		t.Fatalf("unexpected safe audit body: %s", access.Body)
	}
	if !audit.AccessAuditPersisted(ctx) {
		t.Fatal("authentication audit did not suppress duplicate request-level persistence")
	}
	if err := recorder.RecordAuthEvent(ctx, AuthAuditEvent{
		Channel: AuthChannelRefresh, CredentialType: AuthCredentialRefresh,
		Success: true, ReasonCode: "refresh_succeeded", UserID: 1,
	}); err != nil {
		t.Fatal(err)
	}
	var loginCount int64
	if err := db.Model(&model.LoginLog{}).Count(&loginCount).Error; err != nil || loginCount != 1 {
		t.Fatalf("refresh must not be recorded as a credential login: count=%d err=%v", loginCount, err)
	}
}
