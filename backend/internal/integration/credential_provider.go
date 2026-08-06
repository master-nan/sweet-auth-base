package integration

import (
	"backend/internal/audit"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	"backend/model"
	"backend/repository"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const runtimeCredentialSecretMaxBytes = 16384

var runtimeCredentialCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// CredentialResolveRequest 是运行时从已确认配置和执行快照构造的凭证解析请求。
// 它没有对应的 Controller DTO，不能由客户端直接构造或覆盖。
type CredentialResolveRequest struct {
	externalSystemID      int
	interfaceDefinitionID int
	credentialID          int
	credentialCode        string
	credentialType        string
	operationContext      string
}

// NewCredentialResolveRequest 创建受控运行时凭证解析请求。
func NewCredentialResolveRequest(
	externalSystemID int,
	interfaceDefinitionID int,
	credentialID int,
	credentialCode string,
	credentialType string,
	operationContext string,
) (CredentialResolveRequest, error) {
	request := CredentialResolveRequest{
		externalSystemID:      externalSystemID,
		interfaceDefinitionID: interfaceDefinitionID,
		credentialID:          credentialID,
		credentialCode:        strings.TrimSpace(credentialCode),
		credentialType:        strings.ToLower(strings.TrimSpace(credentialType)),
		operationContext:      strings.TrimSpace(operationContext),
	}
	if request.externalSystemID <= 0 || request.interfaceDefinitionID <= 0 || request.credentialID <= 0 ||
		!runtimeCredentialCodePattern.MatchString(request.credentialCode) ||
		!knownCredentialType(request.credentialType) || request.operationContext == "" || len(request.operationContext) > 128 {
		return CredentialResolveRequest{}, myerrors.ErrIntegrationCredentialMaterialInvalid
	}
	return request, nil
}

// OperationContext 返回只用于运行时链路关联的受控操作摘要。
func (r CredentialResolveRequest) OperationContext() string {
	return r.operationContext
}

// CredentialMaterial 是单次 Attempt 内使用的短生命周期秘密材料。
// 其字段保持私有，且只在 Provider 调用栈内转化为 TransportAuthentication。
type CredentialMaterial struct {
	credentialType string
	username       []byte
	password       []byte
	apiKey         []byte
	bearerToken    []byte
}

func (m CredentialMaterial) String() string {
	return "CredentialMaterial(redacted)"
}

func (m CredentialMaterial) GoString() string {
	return "CredentialMaterial(redacted)"
}

// MarshalJSON 阻止秘密材料被意外写入 HTTP 响应、日志或持久化载荷。
func (m CredentialMaterial) MarshalJSON() ([]byte, error) {
	return nil, errors.New("credential material serialization is prohibited")
}

func (m *CredentialMaterial) clear() {
	if m == nil {
		return
	}
	for _, value := range [][]byte{m.username, m.password, m.apiKey, m.bearerToken} {
		for index := range value {
			value[index] = 0
		}
	}
	m.username = nil
	m.password = nil
	m.apiKey = nil
	m.bearerToken = nil
}

// CredentialResolution 是 Provider 的内部受控输出，不包含 CredentialMaterial、密文或存储引用。
type CredentialResolution struct {
	authentication     TransportAuthentication
	credentialCode     string
	credentialType     string
	securityVersion    string
	fingerprintSummary string
}

func (r CredentialResolution) String() string {
	return "CredentialResolution(redacted)"
}

func (r CredentialResolution) GoString() string {
	return "CredentialResolution(redacted)"
}

// MarshalJSON 阻止认证注入结果被误用为 API 响应。
func (r CredentialResolution) MarshalJSON() ([]byte, error) {
	return nil, errors.New("credential resolution serialization is prohibited")
}

func (r CredentialResolution) Authentication() TransportAuthentication {
	return r.authentication
}

func (r CredentialResolution) CredentialCode() string {
	return r.credentialCode
}

func (r CredentialResolution) CredentialType() string {
	return r.credentialType
}

func (r CredentialResolution) SecurityVersionSummary() string {
	return r.securityVersion
}

func (r CredentialResolution) FingerprintSummary() string {
	return r.fingerprintSummary
}

// CredentialProvider 负责在运行时校验并解析配置中心保存的凭证。
// 它不执行 HTTP、不更新 Execution 状态，也不缓存任何解密后的材料。
type CredentialProvider struct {
	credentials repository.CredentialRepository
	interfaces  repository.InterfaceDefinitionRepository
	protector   *security.CredentialSecretProtector
	now         func() time.Time
}

func NewCredentialProvider(
	credentials repository.CredentialRepository,
	interfaces repository.InterfaceDefinitionRepository,
	protector *security.CredentialSecretProtector,
) *CredentialProvider {
	return &CredentialProvider{
		credentials: credentials,
		interfaces:  interfaces,
		protector:   protector,
		now:         time.Now,
	}
}

// Resolve 校验归属、状态和信封后，在当前调用栈内构造受控认证注入结果。
func (p *CredentialProvider) Resolve(
	ctx context.Context,
	request CredentialResolveRequest,
) (resolution CredentialResolution, err error) {
	defer func() {
		p.logResolution(ctx, request, resolution, err)
	}()
	if p == nil || p.credentials == nil || p.interfaces == nil || p.protector == nil {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialDecryptFailed
	}
	if err := validateCredentialResolveRequest(request); err != nil {
		return CredentialResolution{}, err
	}

	credential, err := p.credentials.GetRuntimeCredential(ctx, request.credentialID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CredentialResolution{}, myerrors.ErrIntegrationCredentialNotFound
		}
		return CredentialResolution{}, myerrors.WrapDatabaseError(err)
	}
	if credential.ExternalSystemID != request.externalSystemID {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialSystemMismatch
	}
	if credential.CredentialCode != request.credentialCode || credential.CredentialType != request.credentialType {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialMaterialInvalid
	}

	definition, err := p.interfaces.GetRuntimeInterfaceDefinition(ctx, request.interfaceDefinitionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CredentialResolution{}, myerrors.ErrIntegrationCredentialInterfaceMismatch
		}
		return CredentialResolution{}, myerrors.WrapDatabaseError(err)
	}
	if definition.ExternalSystemID != request.externalSystemID || definition.ExternalSystemID != credential.ExternalSystemID ||
		definition.CredentialID == nil || *definition.CredentialID != credential.ID {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialInterfaceMismatch
	}
	if credential.Status == model.CredentialStatusRevoked {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialRevoked
	}
	if credential.Status != model.CredentialStatusActive || !credential.State {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialInactive
	}
	if credential.ExpiresAt != nil && !credential.ExpiresAt.After(p.now()) {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialExpired
	}
	if !runtimeSupportedCredentialType(credential.CredentialType) {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialTypeUnsupported
	}
	if credential.SecretStorageRef == "" || credential.SecretCiphertext == "" || credential.SecretNonce == "" {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialSecretMissing
	}
	if credential.Version <= 0 || !validCredentialStorageRef(credential.SecretStorageRef) || !validCredentialFingerprint(credential.SecretFingerprint) {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialMaterialInvalid
	}

	plaintext, err := p.protector.Open(credential.SecretCiphertext, credential.SecretNonce)
	if err != nil {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialDecryptFailed
	}
	defer clearCredentialBytes(plaintext)
	if !fingerprintMatches(plaintext, credential.SecretFingerprint) {
		return CredentialResolution{}, myerrors.ErrIntegrationCredentialDecryptFailed
	}

	material, err := newCredentialMaterial(credential.CredentialType, plaintext)
	if err != nil {
		return CredentialResolution{}, err
	}
	defer material.clear()
	authentication, err := material.transportAuthentication()
	if err != nil {
		return CredentialResolution{}, err
	}
	return CredentialResolution{
		authentication:     authentication,
		credentialCode:     credential.CredentialCode,
		credentialType:     credential.CredentialType,
		securityVersion:    "v" + strconv.Itoa(credential.Version),
		fingerprintSummary: runtimeFingerprintSummary(credential.SecretFingerprint),
	}, nil
}

func validateCredentialResolveRequest(request CredentialResolveRequest) error {
	if request.externalSystemID <= 0 || request.interfaceDefinitionID <= 0 || request.credentialID <= 0 ||
		!runtimeCredentialCodePattern.MatchString(request.credentialCode) ||
		!knownCredentialType(request.credentialType) || request.operationContext == "" || len(request.operationContext) > 128 {
		return myerrors.ErrIntegrationCredentialMaterialInvalid
	}
	return nil
}

func newCredentialMaterial(credentialType string, plaintext []byte) (CredentialMaterial, error) {
	if len(plaintext) == 0 || len(plaintext) > runtimeCredentialSecretMaxBytes {
		return CredentialMaterial{}, myerrors.ErrIntegrationCredentialMaterialInvalid
	}
	secret := make(map[string]string)
	if err := json.Unmarshal(plaintext, &secret); err != nil {
		return CredentialMaterial{}, myerrors.ErrIntegrationCredentialMaterialInvalid
	}
	var fields []string
	switch credentialType {
	case model.CredentialTypeBasic:
		fields = []string{"username", "password"}
	case model.CredentialTypeAPIKey:
		fields = []string{"api_key"}
	case model.CredentialTypeBearerToken:
		fields = []string{"token"}
	case model.CredentialTypeOAuthClient:
		return CredentialMaterial{}, myerrors.ErrIntegrationCredentialTypeUnsupported
	default:
		return CredentialMaterial{}, myerrors.ErrIntegrationCredentialTypeUnsupported
	}
	if len(secret) != len(fields) {
		return CredentialMaterial{}, myerrors.ErrIntegrationCredentialMaterialInvalid
	}
	material := CredentialMaterial{credentialType: credentialType}
	for _, field := range fields {
		value := strings.TrimSpace(secret[field])
		if value == "" || len(value) > 4096 {
			material.clear()
			return CredentialMaterial{}, myerrors.ErrIntegrationCredentialMaterialInvalid
		}
		switch field {
		case "username":
			material.username = cloneCredentialBytes([]byte(value))
		case "password":
			material.password = cloneCredentialBytes([]byte(value))
		case "api_key":
			material.apiKey = cloneCredentialBytes([]byte(value))
		case "token":
			material.bearerToken = cloneCredentialBytes([]byte(value))
		}
	}
	return material, nil
}

func (m CredentialMaterial) transportAuthentication() (TransportAuthentication, error) {
	var headers map[string]string
	switch m.credentialType {
	case model.CredentialTypeBasic:
		encoded := base64.StdEncoding.EncodeToString(append(append([]byte{}, m.username...), append([]byte{':'}, m.password...)...))
		headers = map[string]string{"Authorization": "Basic " + encoded}
	case model.CredentialTypeAPIKey:
		// 配置中心一期没有自定义 Header 配置，API Key 固定注入受控的 X-API-Key Header。
		headers = map[string]string{"X-API-Key": string(m.apiKey)}
	case model.CredentialTypeBearerToken:
		headers = map[string]string{"Authorization": "Bearer " + string(m.bearerToken)}
	default:
		return TransportAuthentication{}, myerrors.ErrIntegrationCredentialTypeUnsupported
	}
	authentication, err := NewTransportAuthentication(headers)
	if err != nil {
		return TransportAuthentication{}, myerrors.ErrIntegrationCredentialInjectionInvalid
	}
	return authentication, nil
}

func knownCredentialType(value string) bool {
	switch value {
	case model.CredentialTypeBasic, model.CredentialTypeAPIKey, model.CredentialTypeBearerToken, model.CredentialTypeOAuthClient:
		return true
	default:
		return false
	}
}

func runtimeSupportedCredentialType(value string) bool {
	return value == model.CredentialTypeBasic || value == model.CredentialTypeAPIKey || value == model.CredentialTypeBearerToken
}

func validCredentialFingerprint(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validCredentialStorageRef(value string) bool {
	if len(value) != 32 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func fingerprintMatches(plaintext []byte, expected string) bool {
	actual := sha256.Sum256(plaintext)
	encoded := hex.EncodeToString(actual[:])
	return subtle.ConstantTimeCompare([]byte(encoded), []byte(strings.ToLower(expected))) == 1
}

func runtimeFingerprintSummary(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:6])
}

func cloneCredentialBytes(value []byte) []byte {
	return append([]byte(nil), value...)
}

func clearCredentialBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

func (p *CredentialProvider) logResolution(
	ctx context.Context,
	request CredentialResolveRequest,
	resolution CredentialResolution,
	err error,
) {
	if p == nil {
		return
	}
	correlation := audit.GetCorrelationIDs(ctx)
	credentialCode := request.credentialCode
	credentialType := request.credentialType
	version := ""
	if resolution.credentialCode != "" {
		credentialCode = resolution.credentialCode
		credentialType = resolution.credentialType
		version = resolution.securityVersion
	}
	zap.L().Info("integration credential resolved",
		zap.String("request_id", correlation.RequestID),
		zap.String("trace_id", correlation.TraceID),
		zap.String("credential_code", credentialCode),
		zap.String("credential_type", credentialType),
		zap.String("security_version", version),
		zap.String("result", credentialResolutionResult(err)),
	)
}

func credentialResolutionResult(err error) string {
	if err == nil {
		return "resolved"
	}
	for candidate, category := range map[error]string{
		myerrors.ErrIntegrationCredentialNotFound:          "credential_not_found",
		myerrors.ErrIntegrationCredentialSystemMismatch:    "credential_system_mismatch",
		myerrors.ErrIntegrationCredentialInterfaceMismatch: "credential_interface_mismatch",
		myerrors.ErrIntegrationCredentialInactive:          "credential_inactive",
		myerrors.ErrIntegrationCredentialExpired:           "credential_expired",
		myerrors.ErrIntegrationCredentialRevoked:           "credential_revoked",
		myerrors.ErrIntegrationCredentialTypeUnsupported:   "credential_type_unsupported",
		myerrors.ErrIntegrationCredentialSecretMissing:     "credential_secret_missing",
		myerrors.ErrIntegrationCredentialDecryptFailed:     "credential_decrypt_failed",
		myerrors.ErrIntegrationCredentialMaterialInvalid:   "credential_material_invalid",
		myerrors.ErrIntegrationCredentialInjectionInvalid:  "credential_injection_invalid",
	} {
		if errors.Is(err, candidate) {
			return category
		}
	}
	return "internal_error"
}
