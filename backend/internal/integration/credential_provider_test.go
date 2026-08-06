package integration

import (
	"backend/internal/database"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	testutil "backend/internal/test"
	"backend/model"
	"backend/repository/impl"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestCredentialProviderResolvesSupportedCredentialTypes(t *testing.T) {
	tests := []struct {
		name       string
		credential string
		secret     map[string]string
		header     string
		value      string
	}{
		{
			name:       "basic",
			credential: model.CredentialTypeBasic,
			secret:     map[string]string{"username": "runtime-user", "password": "runtime-password"},
			header:     "authorization",
			value:      "Basic " + base64.StdEncoding.EncodeToString([]byte("runtime-user:runtime-password")),
		},
		{
			name:       "api key",
			credential: model.CredentialTypeAPIKey,
			secret:     map[string]string{"api_key": "runtime-api-key"},
			header:     "x-api-key",
			value:      "runtime-api-key",
		},
		{
			name:       "bearer token",
			credential: model.CredentialTypeBearerToken,
			secret:     map[string]string{"token": "runtime-bearer-token"},
			header:     "authorization",
			value:      "Bearer runtime-bearer-token",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider, _, credential, definition, _ := newCredentialProviderFixture(t, test.credential, test.secret)
			resolution, err := provider.Resolve(context.Background(), credentialResolveRequest(t, credential, definition))
			if err != nil {
				t.Fatalf("resolve credential: %v", err)
			}
			if got := resolution.Authentication().headers[test.header]; got != test.value {
				t.Fatalf("authentication header %q = %q, want %q", test.header, got, test.value)
			}
			if resolution.CredentialCode() != credential.CredentialCode || resolution.CredentialType() != credential.CredentialType ||
				resolution.SecurityVersionSummary() != "v1" || len(resolution.FingerprintSummary()) != 12 {
				t.Fatalf("unsafe or unexpected resolution summary: %s %s %s %s", resolution.CredentialCode(), resolution.CredentialType(), resolution.SecurityVersionSummary(), resolution.FingerprintSummary())
			}
		})
	}
}

func TestCredentialProviderRejectsUnsupportedAndInvalidConfigurations(t *testing.T) {
	t.Run("request must be server complete", func(t *testing.T) {
		if _, err := NewCredentialResolveRequest(1, 2, 3, "runtime_credential", model.CredentialTypeBearerToken, ""); !errors.Is(err, myerrors.ErrIntegrationCredentialMaterialInvalid) {
			t.Fatalf("missing operation context error = %v", err)
		}
	})
	t.Run("oauth client", func(t *testing.T) {
		provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeOAuthClient, map[string]string{"client_id": "client", "client_secret": "secret"})
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialTypeUnsupported)
	})
	t.Run("credential not found", func(t *testing.T) {
		provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		request, err := NewCredentialResolveRequest(credential.ExternalSystemID, definition.Id, credential.Id+99, credential.CredentialCode, credential.CredentialType, "attempt-1")
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		assertCredentialProviderError(t, provider, request, myerrors.ErrIntegrationCredentialNotFound)
	})
	t.Run("soft deleted credential", func(t *testing.T) {
		provider, db, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		if err := db.Delete(&credential).Error; err != nil {
			t.Fatalf("soft delete credential fixture: %v", err)
		}
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialNotFound)
	})
	t.Run("external system mismatch", func(t *testing.T) {
		provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		request, err := NewCredentialResolveRequest(credential.ExternalSystemID+1, definition.Id, credential.Id, credential.CredentialCode, credential.CredentialType, "attempt-1")
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		assertCredentialProviderError(t, provider, request, myerrors.ErrIntegrationCredentialSystemMismatch)
	})
	t.Run("interface mismatch", func(t *testing.T) {
		provider, db, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		if err := db.Model(&model.InterfaceDefinition{}).Where("id = ?", definition.Id).Update("external_system_id", credential.ExternalSystemID+1).Error; err != nil {
			t.Fatalf("change interface fixture: %v", err)
		}
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialInterfaceMismatch)
	})
	for _, status := range []struct {
		name  string
		value string
		err   error
	}{
		{name: "draft", value: model.CredentialStatusDraft, err: myerrors.ErrIntegrationCredentialInactive},
		{name: "disabled", value: model.CredentialStatusDisabled, err: myerrors.ErrIntegrationCredentialInactive},
		{name: "revoked", value: model.CredentialStatusRevoked, err: myerrors.ErrIntegrationCredentialRevoked},
	} {
		t.Run(status.name, func(t *testing.T) {
			provider, db, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
			if err := db.Model(&model.Credential{}).Where("id = ?", credential.Id).Update("status", status.value).Error; err != nil {
				t.Fatalf("change credential status: %v", err)
			}
			assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), status.err)
		})
	}
	t.Run("generic state disabled", func(t *testing.T) {
		provider, db, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		if err := db.Model(&model.Credential{}).Where("id = ?", credential.Id).Update("state", false).Error; err != nil {
			t.Fatalf("disable credential state: %v", err)
		}
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialInactive)
	})
	t.Run("expired", func(t *testing.T) {
		provider, db, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		expiresAt := time.Now().Add(-time.Minute)
		if err := db.Model(&model.Credential{}).Where("id = ?", credential.Id).Update("expires_at", expiresAt).Error; err != nil {
			t.Fatalf("expire credential fixture: %v", err)
		}
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialExpired)
	})
	t.Run("secret storage reference missing", func(t *testing.T) {
		provider, db, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		if err := db.Model(&model.Credential{}).Where("id = ?", credential.Id).Update("secret_storage_ref", "").Error; err != nil {
			t.Fatalf("remove storage reference: %v", err)
		}
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialSecretMissing)
	})
}

func TestCredentialProviderRejectsInvalidSecretEnvelopeAndInjection(t *testing.T) {
	t.Run("ciphertext damaged", func(t *testing.T) {
		provider, db, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		if err := db.Model(&model.Credential{}).Where("id = ?", credential.Id).Update("secret_ciphertext", "damaged").Error; err != nil {
			t.Fatalf("damage ciphertext: %v", err)
		}
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialDecryptFailed)
	})
	t.Run("missing master key", func(t *testing.T) {
		provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		provider.protector = nil
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialDecryptFailed)
	})
	t.Run("different master key", func(t *testing.T) {
		provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "secret"})
		wrongProtector, err := security.NewCredentialSecretProtectorWithKey("different-runtime-master-key")
		if err != nil {
			t.Fatalf("create different protector: %v", err)
		}
		provider.protector = wrongProtector
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialDecryptFailed)
	})
	t.Run("api key cannot inject invalid header value", func(t *testing.T) {
		provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeAPIKey, map[string]string{"api_key": "invalid\napi-key"})
		assertCredentialProviderError(t, provider, credentialResolveRequest(t, credential, definition), myerrors.ErrIntegrationCredentialInjectionInvalid)
	})
}

func TestCredentialProviderUsesRotatedSecretWithoutFallback(t *testing.T) {
	provider, db, credential, definition, protector := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "previous-token"})
	request := credentialResolveRequest(t, credential, definition)
	first, err := provider.Resolve(context.Background(), request)
	if err != nil || first.Authentication().headers["authorization"] != "Bearer previous-token" {
		t.Fatalf("resolve previous secret: %+v %v", first, err)
	}

	payload, err := json.Marshal(map[string]string{"token": "rotated-token"})
	if err != nil {
		t.Fatalf("marshal rotated secret: %v", err)
	}
	envelope, err := protector.Seal(payload)
	if err != nil {
		t.Fatalf("seal rotated secret: %v", err)
	}
	if err := db.Model(&model.Credential{}).Where("id = ?", credential.Id).Updates(map[string]any{
		"secret_storage_ref": envelope.StorageRef,
		"secret_ciphertext":  envelope.Ciphertext,
		"secret_nonce":       envelope.Nonce,
		"secret_fingerprint": envelope.Fingerprint,
		"version":            2,
	}).Error; err != nil {
		t.Fatalf("rotate fixture: %v", err)
	}
	second, err := provider.Resolve(context.Background(), request)
	if err != nil || second.Authentication().headers["authorization"] != "Bearer rotated-token" || second.SecurityVersionSummary() != "v2" {
		t.Fatalf("resolve rotated secret: %+v %v", second, err)
	}
	if err := db.Model(&model.Credential{}).Where("id = ?", credential.Id).Update("secret_ciphertext", "damaged-after-rotation").Error; err != nil {
		t.Fatalf("damage rotated secret: %v", err)
	}
	assertCredentialProviderError(t, provider, request, myerrors.ErrIntegrationCredentialDecryptFailed)
}

func TestCredentialMaterialAndResolutionDoNotSerializeSecrets(t *testing.T) {
	plaintext := []byte(`{"username":"visible-user","password":"visible-secret"}`)
	material, err := newCredentialMaterial(model.CredentialTypeBasic, plaintext)
	if err != nil {
		t.Fatalf("new credential material: %v", err)
	}
	defer material.clear()
	if _, err := json.Marshal(material); err == nil {
		t.Fatal("credential material was serializable")
	}
	if strings.Contains(material.String(), "visible-secret") || strings.Contains(material.GoString(), "visible-user") {
		t.Fatalf("credential material string leaked: %s", material)
	}
	for index := range plaintext {
		plaintext[index] = 0
	}
	authentication, err := material.transportAuthentication()
	if err != nil || authentication.headers["authorization"] == "" {
		t.Fatalf("credential material did not isolate source bytes: %+v %v", authentication, err)
	}

	provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "resolution-secret"})
	resolution, err := provider.Resolve(context.Background(), credentialResolveRequest(t, credential, definition))
	if err != nil {
		t.Fatalf("resolve credential: %v", err)
	}
	if _, err := json.Marshal(resolution); err == nil {
		t.Fatal("credential resolution was serializable")
	}
	if strings.Contains(resolution.String(), "resolution-secret") || strings.Contains(resolution.GoString(), "resolution-secret") {
		t.Fatalf("credential resolution string leaked: %s", resolution)
	}
}

func TestCredentialRuntimeRepositoriesUseMinimalProjection(t *testing.T) {
	provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "projection-secret"})
	record, err := provider.credentials.GetRuntimeCredential(context.Background(), credential.Id)
	if err != nil || record.CredentialCode != credential.CredentialCode || record.SecretCiphertext == "" || record.SecretFingerprint == "" {
		t.Fatalf("runtime credential projection: %+v %v", record, err)
	}
	interfaceRecord, err := provider.interfaces.GetRuntimeInterfaceDefinition(context.Background(), definition.Id)
	if err != nil || interfaceRecord.ExternalSystemID != credential.ExternalSystemID || interfaceRecord.CredentialID == nil || *interfaceRecord.CredentialID != credential.Id {
		t.Fatalf("runtime interface projection: %+v %v", interfaceRecord, err)
	}
}

func TestCredentialProviderConcurrentResolve(t *testing.T) {
	provider, _, credential, definition, _ := newCredentialProviderFixture(t, model.CredentialTypeBearerToken, map[string]string{"token": "concurrent-token"})
	request := credentialResolveRequest(t, credential, definition)
	const callers = 24
	errorsChannel := make(chan error, callers)
	var group sync.WaitGroup
	for index := 0; index < callers; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			resolution, err := provider.Resolve(context.Background(), request)
			if err != nil {
				errorsChannel <- err
				return
			}
			if resolution.Authentication().headers["authorization"] != "Bearer concurrent-token" {
				errorsChannel <- errors.New("unexpected authentication")
			}
		}()
	}
	group.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Errorf("concurrent resolve: %v", err)
	}
}

func newCredentialProviderFixture(
	t *testing.T,
	credentialType string,
	secret map[string]string,
) (*CredentialProvider, *gorm.DB, model.Credential, model.InterfaceDefinition, *security.CredentialSecretProtector) {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.Credential{}, &model.InterfaceDefinition{})
	protector, err := security.NewCredentialSecretProtectorWithKey("runtime-credential-provider-master-key")
	if err != nil {
		t.Fatalf("create secret protector: %v", err)
	}
	payload, err := json.Marshal(secret)
	if err != nil {
		t.Fatalf("marshal secret fixture: %v", err)
	}
	envelope, err := protector.Seal(payload)
	if err != nil {
		t.Fatalf("seal secret fixture: %v", err)
	}
	credential := model.Credential{
		Basic:             model.Basic{Id: 101, State: true},
		ExternalSystemID:  11,
		CredentialCode:    "runtime_credential",
		Name:              "运行时凭证",
		CredentialType:    credentialType,
		Status:            model.CredentialStatusActive,
		SecretStorageRef:  envelope.StorageRef,
		SecretCiphertext:  envelope.Ciphertext,
		SecretNonce:       envelope.Nonce,
		SecretFingerprint: envelope.Fingerprint,
		Version:           1,
		Revision:          1,
	}
	testutil.MustCreate(t, db, &credential)
	credentialID := credential.Id
	definition := model.InterfaceDefinition{
		Basic:            model.Basic{Id: 201, State: true},
		ExternalSystemID: credential.ExternalSystemID,
		InterfaceCode:    "runtime_endpoint",
		Name:             "运行时接口",
		Version:          1,
		Protocol:         model.InterfaceProtocolHTTPS,
		HTTPMethod:       model.InterfaceMethodGET,
		RelativePath:     "/api/runtime",
		CredentialID:     &credentialID,
		TimeoutSeconds:   30,
		ResponseLimit:    1024,
		Status:           model.InterfaceDefinitionStatusEnabled,
		Revision:         1,
	}
	testutil.MustCreate(t, db, &definition)
	primary := &database.PrimaryDB{DB: db}
	return NewCredentialProvider(
		impl.NewCredentialRepositoryImpl(primary),
		impl.NewInterfaceDefinitionRepositoryImpl(primary),
		protector,
	), db, credential, definition, protector
}

func credentialResolveRequest(t *testing.T, credential model.Credential, definition model.InterfaceDefinition) CredentialResolveRequest {
	t.Helper()
	request, err := NewCredentialResolveRequest(
		credential.ExternalSystemID,
		definition.Id,
		credential.Id,
		credential.CredentialCode,
		credential.CredentialType,
		"attempt-1",
	)
	if err != nil {
		t.Fatalf("create credential resolve request: %v", err)
	}
	return request
}

func assertCredentialProviderError(t *testing.T, provider *CredentialProvider, request CredentialResolveRequest, expected error) {
	t.Helper()
	if _, err := provider.Resolve(context.Background(), request); !errors.Is(err, expected) {
		t.Fatalf("resolve error = %v, want %v", err, expected)
	}
}
