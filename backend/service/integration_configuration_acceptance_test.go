package service

import (
	"backend/dto/request"
	"backend/internal/audit"
	"backend/internal/database"
	apperrors "backend/internal/errors"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestIntegrationConfigurationAcceptanceFlow(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.ExternalSystem{}, &model.Credential{}, &model.InterfaceDefinition{})
	primary := &database.PrimaryDB{DB: db}
	sf, err := utils.NewSnowflake(4)
	if err != nil {
		t.Fatalf("create snowflake: %v", err)
	}
	protector, err := security.NewCredentialSecretProtectorWithKey("integration-acceptance-master-key")
	if err != nil {
		t.Fatalf("create secret protector: %v", err)
	}
	auditWriter := &externalSystemAuditWriter{}
	systems := impl.NewExternalSystemRepositoryImpl(primary)
	credentials := impl.NewCredentialRepositoryImpl(primary)
	interfaces := impl.NewInterfaceDefinitionRepositoryImpl(primary)
	systemService := NewExternalSystemService(systems, sf, auditWriter)
	credentialService := NewCredentialService(credentials, systems, protector, sf, auditWriter)
	interfaceService := NewInterfaceDefinitionService(interfaces, systems, credentials, sf, auditWriter)
	ctx := audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(88, "integration-acceptance-admin"))
	ctx = audit.WithCorrelationIDs(ctx, audit.CorrelationIDs{RequestID: "int-acceptance-request", TraceID: "int-acceptance-trace"})

	hrSystem, err := systemService.Create(ctx, request.ExternalSystemCreateReq{
		SystemCode: "hr_demo", Name: "HR Demo", SystemType: model.ExternalSystemTypeHR,
		BaseURL: "https://hr.example.com", OwnerIdentifier: "owner-hr", OwnerName: "HR Owner",
	})
	if err != nil || hrSystem.Status != model.ExternalSystemStatusDraft {
		t.Fatalf("create HR system: %+v err=%v", hrSystem, err)
	}
	credential, err := credentialService.Create(ctx, request.CredentialCreateReq{
		ExternalSystemID: hrSystem.Id, CredentialCode: "hr_api_token", Name: "HR API Token",
		CredentialType: model.CredentialTypeBearerToken, Secret: map[string]string{"token": "acceptance-secret-token"},
	})
	if err != nil || credential.Status != model.CredentialStatusDraft {
		t.Fatalf("create credential: %+v err=%v", credential, err)
	}
	interfaceDefinition, err := interfaceService.Create(ctx, request.InterfaceDefinitionCreateReq{
		ExternalSystemID: hrSystem.Id, InterfaceCode: "org_list", Name: "Organization List",
		Protocol: model.InterfaceProtocolHTTPS, HTTPMethod: model.InterfaceMethodGET,
		RelativePath: "/api/organizations", CredentialID: &credential.Id,
		TimeoutSeconds: 30, ResponseLimit: 10 * 1024 * 1024,
	})
	if err != nil || interfaceDefinition.Status != model.InterfaceDefinitionStatusDraft {
		t.Fatalf("create interface: %+v err=%v", interfaceDefinition, err)
	}
	hrSystem, err = systemService.Enable(ctx, hrSystem.Id, hrSystem.Revision)
	if err != nil {
		t.Fatalf("enable HR system: %v", err)
	}
	credential, err = credentialService.Enable(ctx, credential.Id, credential.Revision)
	if err != nil {
		t.Fatalf("enable credential: %v", err)
	}
	interfaceDefinition, err = interfaceService.Enable(ctx, interfaceDefinition.Id, interfaceDefinition.Revision)
	if err != nil || interfaceDefinition.EffectiveStatus != model.InterfaceDefinitionStatusEnabled {
		t.Fatalf("enable interface: %+v err=%v", interfaceDefinition, err)
	}

	otherSystem, err := systemService.Create(ctx, request.ExternalSystemCreateReq{
		SystemCode: "erp_demo", Name: "ERP Demo", SystemType: model.ExternalSystemTypeERP,
		BaseURL: "https://erp.example.com", OwnerIdentifier: "owner-erp", OwnerName: "ERP Owner",
	})
	if err != nil {
		t.Fatalf("create second system: %v", err)
	}
	otherCredential, err := credentialService.Create(ctx, request.CredentialCreateReq{
		ExternalSystemID: otherSystem.Id, CredentialCode: "erp_token", Name: "ERP Token",
		CredentialType: model.CredentialTypeBearerToken, Secret: map[string]string{"token": "other-secret"},
	})
	if err != nil {
		t.Fatalf("create second credential: %v", err)
	}
	crossSystemRequest := interfaceDefinitionCreateRequest(hrSystem.Id, "cross_system")
	crossSystemRequest.CredentialID = &otherCredential.Id
	if _, err := interfaceService.Create(ctx, crossSystemRequest); !errors.Is(err, apperrors.ErrInterfaceCredentialInvalid) {
		t.Fatalf("cross-system credential error = %v", err)
	}

	v2, err := interfaceService.CreateVersion(ctx, interfaceDefinition.Id, interfaceDefinition.Revision)
	if err != nil || v2.Version != 2 || v2.Status != model.InterfaceDefinitionStatusDraft {
		t.Fatalf("create next version: %+v err=%v", v2, err)
	}
	if _, err := interfaceService.Enable(ctx, v2.Id, v2.Revision); !errors.Is(err, apperrors.ErrInterfaceEnabledVersionConflict) {
		t.Fatalf("enabled version conflict = %v", err)
	}
	interfaceDefinition, err = interfaceService.Disable(ctx, interfaceDefinition.Id, interfaceDefinition.Revision)
	if err != nil {
		t.Fatalf("disable v1: %v", err)
	}
	v2, err = interfaceService.Enable(ctx, v2.Id, v2.Revision)
	if err != nil || v2.EffectiveStatus != model.InterfaceDefinitionStatusEnabled {
		t.Fatalf("enable v2: %+v err=%v", v2, err)
	}

	hrSystem, err = systemService.Disable(ctx, hrSystem.Id, hrSystem.Revision)
	if err != nil {
		t.Fatalf("disable HR system: %v", err)
	}
	interfaceAfterSystemDisable, err := interfaceService.Get(ctx, v2.Id)
	if err != nil || interfaceAfterSystemDisable.Status != model.InterfaceDefinitionStatusEnabled || interfaceAfterSystemDisable.EffectiveStatus != "unavailable" {
		t.Fatalf("system disable effective state: %+v err=%v", interfaceAfterSystemDisable, err)
	}
	credentialAfterSystemDisable, err := credentialService.Get(ctx, credential.Id)
	if err != nil || credentialAfterSystemDisable.Status != model.CredentialStatusActive {
		t.Fatalf("system disable changed credential state: %+v err=%v", credentialAfterSystemDisable, err)
	}

	var stored model.Credential
	if err := db.First(&stored, credential.Id).Error; err != nil {
		t.Fatalf("load stored credential: %v", err)
	}
	if stored.SecretCiphertext == "acceptance-secret-token" || strings.Contains(stored.SecretCiphertext, "acceptance-secret-token") {
		t.Fatal("credential secret was stored as plaintext")
	}
	responsePayload, _ := json.Marshal(credentialAfterSystemDisable)
	auditPayload, _ := json.Marshal(auditWriter.records)
	for _, payload := range []string{string(responsePayload), string(auditPayload)} {
		if strings.Contains(payload, "acceptance-secret-token") || strings.Contains(payload, "secret_ciphertext") || strings.Contains(payload, "secret_nonce") {
			t.Fatalf("secret boundary leak: %s", payload)
		}
	}
}
