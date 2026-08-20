package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/security"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	credentialAuditResourceType = "integration_credential"
	credentialAuditCreate       = "integration.credential.create"
	credentialAuditUpdate       = "integration.credential.update"
	credentialAuditRotate       = "integration.credential.rotate"
	credentialAuditEnable       = "integration.credential.enable"
	credentialAuditDisable      = "integration.credential.disable"
	credentialAuditRevoke       = "integration.credential.revoke"
	credentialSecretMaxBytes    = 16384
)

var credentialCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

type CredentialService struct {
	repository repository.CredentialRepository
	systems    repository.ExternalSystemRepository
	protector  *security.CredentialSecretProtector
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
	now        func() time.Time
}

func NewCredentialService(
	repository repository.CredentialRepository,
	systems repository.ExternalSystemRepository,
	protector *security.CredentialSecretProtector,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *CredentialService {
	return &CredentialService{
		repository: repository,
		systems:    systems,
		protector:  protector,
		sf:         sf,
		audit:      audit,
		now:        time.Now,
	}
}

func (s *CredentialService) Create(ctx context.Context, req request.CredentialCreateReq) (response.CredentialDetailRes, error) {
	code := strings.TrimSpace(req.CredentialCode)
	if !credentialCodePattern.MatchString(code) {
		return response.CredentialDetailRes{}, myerrors.ErrCredentialCodeInvalid
	}
	credentialType := strings.ToLower(strings.TrimSpace(req.CredentialType))
	secretPayload, err := normalizeCredentialSecret(credentialType, req.Secret)
	if err != nil {
		return response.CredentialDetailRes{}, err
	}
	envelope, err := s.protector.Seal(secretPayload)
	if err != nil {
		return response.CredentialDetailRes{}, myerrors.ErrCredentialProtectionFailed
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(s.now()) {
		return response.CredentialDetailRes{}, myerrors.ErrCredentialExpired
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return response.CredentialDetailRes{}, myerrors.WrapSystemError(err)
	}
	value := model.Credential{
		Basic:             model.Basic{Id: int(id), State: false},
		ExternalSystemID:  req.ExternalSystemID,
		CredentialCode:    code,
		Name:              strings.TrimSpace(req.Name),
		CredentialType:    credentialType,
		Status:            model.CredentialStatusDraft,
		SecretStorageRef:  envelope.StorageRef,
		SecretCiphertext:  envelope.Ciphertext,
		SecretNonce:       envelope.Nonce,
		SecretFingerprint: envelope.Fingerprint,
		ExpiresAt:         req.ExpiresAt,
		Version:           1,
		Description:       strings.TrimSpace(req.Description),
		Revision:          1,
	}
	if value.Name == "" {
		return response.CredentialDetailRes{}, myerrors.ErrCredentialSecretInvalid
	}
	var system model.ExternalSystem
	err = RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		var loadErr error
		system, loadErr = s.loadSystem(tx, value.ExternalSystemID, false)
		if loadErr != nil {
			return loadErr
		}
		if err := s.repository.Create(tx, &value); err != nil {
			if isCredentialDuplicate(err) {
				return myerrors.ErrCredentialCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, credentialAuditCreate, value, system, nil)
	})
	if err != nil {
		return response.CredentialDetailRes{}, err
	}
	return response.NewCredentialDetailRes(value, system, s.now()), nil
}

func (s *CredentialService) Get(ctx context.Context, id int) (response.CredentialDetailRes, error) {
	value, system, err := s.loadDetail(ctx, id)
	if err != nil {
		return response.CredentialDetailRes{}, err
	}
	return response.NewCredentialDetailRes(value, system, s.now()), nil
}

func (s *CredentialService) Page(ctx context.Context, req request.CredentialQueryReq) (response.ListResult[response.CredentialListRes], error) {
	basic := req.ToBasic()
	result, err := s.repository.GetCredentialList(ctx, &basic, credentialQueryTable())
	if err != nil {
		return response.ListResult[response.CredentialListRes]{}, myerrors.WrapDatabaseError(err)
	}
	systemIDs := make([]int, 0, len(result.Data))
	seen := make(map[int]struct{}, len(result.Data))
	for _, item := range result.Data {
		if _, ok := seen[item.ExternalSystemID]; !ok {
			seen[item.ExternalSystemID] = struct{}{}
			systemIDs = append(systemIDs, item.ExternalSystemID)
		}
	}
	systems, err := s.systems.WithContext(ctx).FindListByFieldIn("id", systemIDs)
	if err != nil {
		return response.ListResult[response.CredentialListRes]{}, myerrors.WrapDatabaseError(err)
	}
	byID := make(map[int]model.ExternalSystem, len(systems))
	for _, system := range systems {
		byID[system.Id] = system
	}
	now := s.now()
	items := make([]response.CredentialListRes, 0, len(result.Data))
	for _, value := range result.Data {
		items = append(items, response.NewCredentialListRes(value, byID[value.ExternalSystemID], now))
	}
	return response.ListResult[response.CredentialListRes]{Data: items, Total: result.Total}, nil
}

func (s *CredentialService) Update(ctx context.Context, id int, req request.CredentialUpdateReq) (response.CredentialDetailRes, error) {
	if req.ExternalSystemID != nil || req.CredentialCode != nil || req.CredentialType != nil {
		return response.CredentialDetailRes{}, myerrors.ErrCredentialFieldImmutable
	}
	var updated model.Credential
	var system model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrCredentialNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status == model.CredentialStatusRevoked {
			return myerrors.ErrCredentialStatusInvalid
		}
		updates := map[string]any{"revision": current.Revision + 1}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return myerrors.ErrCredentialSecretInvalid
			}
			updates["name"] = name
		}
		if req.ClearExpiresAt {
			updates["expires_at"] = nil
		} else if req.ExpiresAt != nil {
			updates["expires_at"] = req.ExpiresAt
		}
		if req.Description != nil {
			updates["description"] = strings.TrimSpace(*req.Description)
		}
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, req.Revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrCredentialRevisionConflict
		}
		updated = current
		applyCredentialUpdates(&updated, updates)
		system, err = s.loadSystem(tx, current.ExternalSystemID, false)
		if err != nil {
			return err
		}
		return s.writeAudit(ctx, tx, credentialAuditUpdate, updated, system, &current)
	})
	if err != nil {
		return response.CredentialDetailRes{}, err
	}
	return response.NewCredentialDetailRes(updated, system, s.now()), nil
}

func (s *CredentialService) Rotate(ctx context.Context, id int, req request.CredentialRotateReq) (response.CredentialDetailRes, error) {
	var updated model.Credential
	var system model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrCredentialNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status == model.CredentialStatusRevoked {
			return myerrors.ErrCredentialStatusInvalid
		}
		payload, err := normalizeCredentialSecret(current.CredentialType, req.Secret)
		if err != nil {
			return err
		}
		envelope, err := s.protector.Seal(payload)
		if err != nil {
			return myerrors.ErrCredentialProtectionFailed
		}
		now := s.now()
		updates := map[string]any{
			"secret_storage_ref": envelope.StorageRef,
			"secret_ciphertext":  envelope.Ciphertext,
			"secret_nonce":       envelope.Nonce,
			"secret_fingerprint": envelope.Fingerprint,
			"version":            current.Version + 1,
			"rotated_at":         now,
			"revision":           current.Revision + 1,
		}
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, req.Revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrCredentialRevisionConflict
		}
		updated = current
		applyCredentialUpdates(&updated, updates)
		system, err = s.loadSystem(tx, current.ExternalSystemID, false)
		if err != nil {
			return err
		}
		return s.writeAudit(ctx, tx, credentialAuditRotate, updated, system, &current)
	})
	if err != nil {
		return response.CredentialDetailRes{}, err
	}
	return response.NewCredentialDetailRes(updated, system, s.now()), nil
}

func (s *CredentialService) Enable(ctx context.Context, id, revision int) (response.CredentialDetailRes, error) {
	return s.changeStatus(ctx, id, revision, model.CredentialStatusActive, credentialAuditEnable)
}

func (s *CredentialService) Disable(ctx context.Context, id, revision int) (response.CredentialDetailRes, error) {
	return s.changeStatus(ctx, id, revision, model.CredentialStatusDisabled, credentialAuditDisable)
}

func (s *CredentialService) Revoke(ctx context.Context, id, revision int) (response.CredentialDetailRes, error) {
	return s.changeStatus(ctx, id, revision, model.CredentialStatusRevoked, credentialAuditRevoke)
}

func (s *CredentialService) changeStatus(ctx context.Context, id, revision int, target, action string) (response.CredentialDetailRes, error) {
	var updated model.Credential
	var system model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrCredentialNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status == model.CredentialStatusRevoked || !credentialTransitionAllowed(current.Status, target) {
			return myerrors.ErrCredentialStatusInvalid
		}
		if target == model.CredentialStatusActive {
			if current.ExpiresAt != nil && !current.ExpiresAt.After(s.now()) {
				return myerrors.ErrCredentialExpired
			}
			system, err = s.loadSystem(tx, current.ExternalSystemID, true)
		} else {
			system, err = s.loadSystem(tx, current.ExternalSystemID, false)
		}
		if err != nil {
			return err
		}
		updates := map[string]any{"status": target, "state": target == model.CredentialStatusActive, "revision": current.Revision + 1}
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrCredentialRevisionConflict
		}
		updated = current
		applyCredentialUpdates(&updated, updates)
		return s.writeAudit(ctx, tx, action, updated, system, &current)
	})
	if err != nil {
		return response.CredentialDetailRes{}, err
	}
	return response.NewCredentialDetailRes(updated, system, s.now()), nil
}

func (s *CredentialService) loadDetail(ctx context.Context, id int) (model.Credential, model.ExternalSystem, error) {
	value, err := s.repository.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Credential{}, model.ExternalSystem{}, myerrors.ErrCredentialNotFound
	}
	if err != nil {
		return model.Credential{}, model.ExternalSystem{}, myerrors.WrapDatabaseError(err)
	}
	system, err := s.systems.WithContext(ctx).FindById(value.ExternalSystemID)
	if err != nil {
		return model.Credential{}, model.ExternalSystem{}, myerrors.WrapDatabaseError(err)
	}
	return value, system, nil
}

func (s *CredentialService) loadSystem(tx *gorm.DB, id int, requireEnabled bool) (model.ExternalSystem, error) {
	system, err := s.systems.FindByIdForUpdate(tx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ExternalSystem{}, myerrors.ErrCredentialExternalSystemInvalid
	}
	if err != nil {
		return model.ExternalSystem{}, myerrors.WrapDatabaseError(err)
	}
	if requireEnabled && system.Status != model.ExternalSystemStatusEnabled {
		return model.ExternalSystem{}, myerrors.ErrCredentialExternalSystemInvalid
	}
	return system, nil
}

func (s *CredentialService) writeAudit(ctx context.Context, tx *gorm.DB, action string, value model.Credential, system model.ExternalSystem, old *model.Credential) error {
	changes := map[string]TransactionalAuditChange{
		"system_code":         {NewValue: system.SystemCode},
		"credential_code":     {NewValue: value.CredentialCode},
		"credential_type":     {NewValue: value.CredentialType},
		"status":              {NewValue: value.Status},
		"credential_version":  {NewValue: value.Version},
		"fingerprint_summary": {NewValue: credentialFingerprintAuditSummary(value.SecretFingerprint)},
	}
	if old != nil {
		changes["status"] = TransactionalAuditChange{OldValue: old.Status, NewValue: value.Status}
		changes["credential_version"] = TransactionalAuditChange{OldValue: old.Version, NewValue: value.Version}
		changes["fingerprint_summary"] = TransactionalAuditChange{OldValue: credentialFingerprintAuditSummary(old.SecretFingerprint), NewValue: credentialFingerprintAuditSummary(value.SecretFingerprint)}
	}
	return s.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
		Action: action, ResourceType: credentialAuditResourceType,
		ResourceCode: value.CredentialCode, ResourceId: strconv.Itoa(value.Id), Changes: changes,
	})
}

func normalizeCredentialSecret(credentialType string, secret map[string]string) ([]byte, error) {
	required := map[string][]string{
		model.CredentialTypeBasic:       {"username", "password"},
		model.CredentialTypeAPIKey:      {"api_key"},
		model.CredentialTypeBearerToken: {"token"},
		model.CredentialTypeOAuthClient: {"client_id", "client_secret"},
	}
	fields, ok := required[credentialType]
	if !ok {
		return nil, myerrors.ErrCredentialTypeInvalid
	}
	if len(secret) != len(fields) {
		return nil, myerrors.ErrCredentialSecretInvalid
	}
	normalized := make(map[string]string, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(secret[field])
		if value == "" || len(value) > 4096 {
			return nil, myerrors.ErrCredentialSecretInvalid
		}
		normalized[field] = value
	}
	payload, err := json.Marshal(normalized)
	if err != nil || len(payload) > credentialSecretMaxBytes {
		return nil, myerrors.ErrCredentialSecretInvalid
	}
	return payload, nil
}

func credentialTransitionAllowed(current, target string) bool {
	switch target {
	case model.CredentialStatusActive:
		return current == model.CredentialStatusDraft || current == model.CredentialStatusDisabled
	case model.CredentialStatusDisabled:
		return current == model.CredentialStatusActive
	case model.CredentialStatusRevoked:
		return current == model.CredentialStatusDraft || current == model.CredentialStatusActive || current == model.CredentialStatusDisabled
	default:
		return false
	}
}

func applyCredentialUpdates(value *model.Credential, updates map[string]any) {
	if item, ok := updates["name"].(string); ok {
		value.Name = item
	}
	if item, ok := updates["description"].(string); ok {
		value.Description = item
	}
	if item, ok := updates["status"].(string); ok {
		value.Status = item
	}
	if item, ok := updates["state"].(bool); ok {
		value.State = item
	}
	if item, ok := updates["revision"].(int); ok {
		value.Revision = item
	}
	if item, ok := updates["version"].(int); ok {
		value.Version = item
	}
	if item, ok := updates["secret_storage_ref"].(string); ok {
		value.SecretStorageRef = item
	}
	if item, ok := updates["secret_ciphertext"].(string); ok {
		value.SecretCiphertext = item
	}
	if item, ok := updates["secret_nonce"].(string); ok {
		value.SecretNonce = item
	}
	if item, ok := updates["secret_fingerprint"].(string); ok {
		value.SecretFingerprint = item
	}
	if item, exists := updates["expires_at"]; exists {
		if item == nil {
			value.ExpiresAt = nil
		} else if parsed, ok := item.(*time.Time); ok {
			value.ExpiresAt = parsed
		}
	}
	if item, ok := updates["rotated_at"].(time.Time); ok {
		value.RotatedAt = &item
	}
}

func credentialFingerprintAuditSummary(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func isCredentialDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" || strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
