package service

import (
	"backend/dto/request"
	"backend/internal/audit"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"backend/internal/database"
	"gorm.io/gorm"
)

type credentialAuditWriter struct {
	mu      sync.Mutex
	records []TransactionalAuditRecord
}

func (w *credentialAuditWriter) RecordTransactionalAuditContext(_ context.Context, _ *gorm.DB, record TransactionalAuditRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.records = append(w.records, record)
	return nil
}

func TestCredentialServiceLifecycleSecretBoundaryAndAudit(t *testing.T) {
	svc, db, writer, system := newCredentialTestSubject(t)
	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(21, "integration-admin"))
	created, err := svc.Create(ctx, credentialCreateRequest(system.Id, "erp_api"))
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if created.Status != model.CredentialStatusDraft || created.Version != 1 || created.FingerprintSummary == "" {
		t.Fatalf("unexpected created credential: %+v", created)
	}
	var stored model.Credential
	if err := db.First(&stored, created.Id).Error; err != nil {
		t.Fatalf("load stored credential: %v", err)
	}
	if stored.SecretCiphertext == "secret-api-key" || strings.Contains(stored.SecretCiphertext, "secret-api-key") || stored.SecretStorageRef == "" {
		t.Fatalf("secret was not protected: %+v", stored)
	}
	payload, _ := json.Marshal(created)
	for _, forbidden := range []string{"secret_ciphertext", "secret_nonce", "secret_storage_ref", "secret_fingerprint", "secret-api-key"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("response leaked %q: %s", forbidden, payload)
		}
	}

	enabled, err := svc.Enable(ctx, created.Id, created.Revision)
	if err != nil {
		t.Fatalf("enable credential: %v", err)
	}
	rotated, err := svc.Rotate(ctx, created.Id, request.CredentialRotateReq{Secret: map[string]string{"api_key": "rotated-api-key"}, Revision: enabled.Revision})
	if err != nil {
		t.Fatalf("rotate credential: %v", err)
	}
	if rotated.Version != 2 || rotated.FingerprintSummary == created.FingerprintSummary || rotated.RotatedAt == nil {
		t.Fatalf("unexpected rotation: %+v", rotated)
	}
	disabled, err := svc.Disable(ctx, created.Id, rotated.Revision)
	if err != nil {
		t.Fatalf("disable credential: %v", err)
	}
	revoked, err := svc.Revoke(ctx, created.Id, disabled.Revision)
	if err != nil {
		t.Fatalf("revoke credential: %v", err)
	}
	if _, err := svc.Enable(ctx, created.Id, revoked.Revision); !errors.Is(err, myerrors.ErrCredentialStatusInvalid) {
		t.Fatalf("revoked enable error = %v", err)
	}
	if _, err := svc.Rotate(ctx, created.Id, request.CredentialRotateReq{Secret: map[string]string{"api_key": "again"}, Revision: revoked.Revision}); !errors.Is(err, myerrors.ErrCredentialStatusInvalid) {
		t.Fatalf("revoked rotate error = %v", err)
	}
	if len(writer.records) != 5 {
		t.Fatalf("audit count = %d", len(writer.records))
	}
	auditPayload, _ := json.Marshal(writer.records)
	if strings.Contains(string(auditPayload), "secret-api-key") || strings.Contains(string(auditPayload), "rotated-api-key") {
		t.Fatalf("audit leaked secret: %s", auditPayload)
	}
}

func TestCredentialServiceValidationUniquenessAndExpiry(t *testing.T) {
	svc, _, _, system := newCredentialTestSubject(t)
	ctx := context.Background()
	created, err := svc.Create(ctx, credentialCreateRequest(system.Id, "shared_key"))
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	if _, err := svc.Create(ctx, credentialCreateRequest(system.Id, "shared_key")); !errors.Is(err, myerrors.ErrCredentialCodeDuplicate) {
		t.Fatalf("duplicate error = %v", err)
	}
	invalid := credentialCreateRequest(system.Id, "Bad Code")
	if _, err := svc.Create(ctx, invalid); !errors.Is(err, myerrors.ErrCredentialCodeInvalid) {
		t.Fatalf("invalid code error = %v", err)
	}
	invalidSecret := credentialCreateRequest(system.Id, "invalid_secret")
	invalidSecret.Secret = map[string]string{"api_key": "", "extra": "x"}
	if _, err := svc.Create(ctx, invalidSecret); !errors.Is(err, myerrors.ErrCredentialSecretInvalid) {
		t.Fatalf("invalid secret error = %v", err)
	}
	changedCode := "other"
	if _, err := svc.Update(ctx, created.Id, request.CredentialUpdateReq{CredentialCode: &changedCode, Revision: created.Revision}); !errors.Is(err, myerrors.ErrCredentialFieldImmutable) {
		t.Fatalf("immutable error = %v", err)
	}
	expiredAt := time.Now().Add(-time.Hour)
	name := "expired later"
	updated, err := svc.Update(ctx, created.Id, request.CredentialUpdateReq{Name: &name, ExpiresAt: &expiredAt, Revision: created.Revision})
	if err != nil {
		t.Fatalf("update expiry: %v", err)
	}
	if _, err := svc.Enable(ctx, created.Id, updated.Revision); !errors.Is(err, myerrors.ErrCredentialExpired) {
		t.Fatalf("expired enable error = %v", err)
	}
	page, err := svc.Page(ctx, request.CredentialQueryReq{Page: 1, Num: 10, Status: "expired"})
	if err != nil || page.Total != 1 || page.Data[0].EffectiveStatus != "expired" {
		t.Fatalf("expired page = %+v err=%v", page, err)
	}
}

func newCredentialTestSubject(t *testing.T) (*CredentialService, *gorm.DB, *credentialAuditWriter, model.ExternalSystem) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.Credential{})
	system := model.ExternalSystem{Basic: model.Basic{Id: 100, State: true}, SystemCode: "demo_erp", Name: "Demo ERP", SystemType: model.ExternalSystemTypeERP, BaseURL: "https://erp.example.com", OwnerIdentifier: "owner", OwnerName: "负责人", Status: model.ExternalSystemStatusEnabled, Revision: 1}
	if err := db.Create(&system).Error; err != nil {
		t.Fatalf("create system: %v", err)
	}
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("snowflake: %v", err)
	}
	protector, err := security.NewCredentialSecretProtectorWithKey("credential-service-test-master-key")
	if err != nil {
		t.Fatalf("protector: %v", err)
	}
	writer := &credentialAuditWriter{}
	svc := NewCredentialService(impl.NewCredentialRepositoryImpl(&database.PrimaryDB{DB: db}), impl.NewExternalSystemRepositoryImpl(&database.PrimaryDB{DB: db}), protector, sf, writer)
	return svc, db, writer, system
}

func credentialCreateRequest(systemID int, code string) request.CredentialCreateReq {
	return request.CredentialCreateReq{ExternalSystemID: systemID, CredentialCode: code, Name: "ERP API Key", CredentialType: model.CredentialTypeAPIKey, Secret: map[string]string{"api_key": "secret-api-key"}, Description: "测试凭证"}
}
