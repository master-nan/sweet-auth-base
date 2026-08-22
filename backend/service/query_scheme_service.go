package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/metadata"
	"backend/internal/queryscheme"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// QuerySchemeService 负责查询方案管理写入与运行时读取编排，
// 在持久化前统一执行Scope授权、Metadata校验、可见性和revision检查。
type QuerySchemeService struct {
	repository repository.QuerySchemeRepository
	scopes     queryscheme.ScopeReader
	validator  *queryscheme.Validator
	bindings   *queryscheme.BindingResolver
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
}

func NewQuerySchemeService(
	repository repository.QuerySchemeRepository,
	scopes *queryscheme.Registry,
	metadataReader metadata.RuntimeReader,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *QuerySchemeService {
	return &QuerySchemeService{
		repository: repository,
		scopes:     scopes,
		validator:  queryscheme.NewValidator(metadataReader),
		bindings:   queryscheme.NewBindingResolver(queryscheme.SystemClock(), model.AppLocation()),
		sf:         sf,
		audit:      audit,
	}
}

func (service *QuerySchemeService) CreatePersonal(
	ctx context.Context,
	req request.QuerySchemePersonalCreateReq,
) (response.QuerySchemeDetailRes, error) {
	actor, config, _, err := service.authorizeScope(ctx, req.ScopeCode)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	requestPayload, err := decodeQuerySchemeRequestPayload(req.Payload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	payload, raw, _, err := service.preparePayload(ctx, config, requestPayload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	name, err := normalizeSchemeName(req.Name)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	id, err := service.nextID()
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	value := model.QueryScheme{
		Basic:              model.Basic{Id: id, State: true},
		Name:               name,
		ScopeCode:          strings.TrimSpace(req.ScopeCode),
		SchemeType:         model.QuerySchemeTypePersonal,
		OwnerUserID:        &actor.UserID,
		QuerySchemaVersion: queryscheme.SchemaVersion,
		QueryPayload:       datatypes.JSON(raw),
		IsDefault:          req.IsDefault,
		Enabled:            true,
		Revision:           1,
	}
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		if value.IsDefault {
			if err := service.repository.ClearDefault(tx, value.SchemeType, actor.UserID, value.ScopeCode, 0); err != nil {
				return err
			}
		}
		if err := service.repository.Create(tx, &value); err != nil {
			return err
		}
		return service.writeAudit(ctx, tx, value, "create")
	})
	if err != nil {
		return response.QuerySchemeDetailRes{}, querySchemePersistenceError(err)
	}
	return service.detailResponse(ctx, value, payload, queryscheme.ValidationResult{Status: queryscheme.ValidationValid}, nil)
}

func (service *QuerySchemeService) UpdatePersonal(
	ctx context.Context,
	id int,
	req request.QuerySchemePersonalUpdateReq,
) (response.QuerySchemeDetailRes, error) {
	actor, err := service.actor(ctx)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	name, err := normalizeSchemeName(req.Name)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	current, config, err := service.preflightMutation(ctx, actor, id, req.Revision, false)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	requestPayload, err := decodeQuerySchemeRequestPayload(req.Payload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	_, raw, _, err := service.preparePayload(ctx, config, requestPayload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	var updated model.QueryScheme
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		locked, findErr := service.repository.FindByIDWithDB(tx.WithContext(ctx), id, true)
		if findErr != nil {
			return querySchemeLookupError(findErr)
		}
		if mutationErr := validateLockedMutation(locked, current, actor.UserID, req.Revision, false); mutationErr != nil {
			return mutationErr
		}
		if req.IsDefault {
			if err := service.repository.ClearDefault(tx, locked.SchemeType, actor.UserID, locked.ScopeCode, locked.Id); err != nil {
				return err
			}
		}
		matched, updateErr := service.repository.UpdateFieldsByRevision(tx.WithContext(ctx), id, req.Revision, map[string]any{
			"name":                 name,
			"query_payload":        datatypes.JSON(raw),
			"query_schema_version": queryscheme.SchemaVersion,
			"is_default":           req.IsDefault,
			"revision":             gorm.Expr("revision + 1"),
		})
		if updateErr != nil {
			return updateErr
		}
		if !matched {
			return myerrors.ErrQuerySchemeRevisionConflict
		}
		updated, updateErr = service.repository.FindByIDWithDB(tx.WithContext(ctx), id, false)
		if updateErr != nil {
			return updateErr
		}
		return service.writeAudit(ctx, tx, updated, "update")
	})
	if err != nil {
		return response.QuerySchemeDetailRes{}, querySchemePersistenceError(err)
	}
	return service.buildDetail(ctx, updated)
}

func (service *QuerySchemeService) DeletePersonal(ctx context.Context, id, revision int) error {
	return service.deleteScheme(ctx, id, revision, false)
}

func (service *QuerySchemeService) SetPersonalDefault(
	ctx context.Context,
	id int,
	req request.QuerySchemeDefaultReq,
) (response.QuerySchemeDetailRes, error) {
	actor, err := service.actor(ctx)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	current, _, err := service.preflightMutation(ctx, actor, id, req.Revision, false)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	var updated model.QueryScheme
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		locked, findErr := service.repository.FindByIDWithDB(tx.WithContext(ctx), id, true)
		if findErr != nil {
			return querySchemeLookupError(findErr)
		}
		if mutationErr := validateLockedMutation(locked, current, actor.UserID, req.Revision, false); mutationErr != nil {
			return mutationErr
		}
		if req.IsDefault {
			if err := service.repository.ClearDefault(tx, locked.SchemeType, actor.UserID, locked.ScopeCode, locked.Id); err != nil {
				return err
			}
		}
		matched, updateErr := service.repository.UpdateFieldsByRevision(tx.WithContext(ctx), id, req.Revision, map[string]any{
			"is_default": req.IsDefault,
			"revision":   gorm.Expr("revision + 1"),
		})
		if updateErr != nil {
			return updateErr
		}
		if !matched {
			return myerrors.ErrQuerySchemeRevisionConflict
		}
		updated, updateErr = service.repository.FindByIDWithDB(tx.WithContext(ctx), id, false)
		if updateErr != nil {
			return updateErr
		}
		action := "clear_default"
		if req.IsDefault {
			action = "set_default"
		}
		return service.writeAudit(ctx, tx, updated, action)
	})
	if err != nil {
		return response.QuerySchemeDetailRes{}, querySchemePersistenceError(err)
	}
	return service.buildDetail(ctx, updated)
}

func (service *QuerySchemeService) CreateShared(
	ctx context.Context,
	req request.QuerySchemeSharedCreateReq,
) (response.QuerySchemeDetailRes, error) {
	actor, config, _, err := service.authorizeScope(ctx, req.ScopeCode)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	if err := service.requireSharedManager(ctx, actor.UserID); err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	if !sharedSchemeType(req.SchemeType) || (req.SchemeType != model.QuerySchemeTypePageDefault && req.IsDefault) {
		return response.QuerySchemeDetailRes{}, myerrors.ErrQuerySchemeTypeInvalid
	}
	if err := service.validateRoles(ctx, req.SchemeType, req.RoleIDs); err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	requestPayload, err := decodeQuerySchemeRequestPayload(req.Payload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	payload, raw, _, err := service.preparePayload(ctx, config, requestPayload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	name, err := normalizeSchemeName(req.Name)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	id, err := service.nextID()
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	value := model.QueryScheme{
		Basic:              model.Basic{Id: id, State: true},
		Name:               name,
		ScopeCode:          strings.TrimSpace(req.ScopeCode),
		SchemeType:         req.SchemeType,
		QuerySchemaVersion: queryscheme.SchemaVersion,
		QueryPayload:       datatypes.JSON(raw),
		IsDefault:          req.IsDefault,
		Enabled:            req.Enabled,
		Revision:           1,
	}
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		if value.SchemeType == model.QuerySchemeTypePageDefault && value.IsDefault && value.Enabled {
			if err := service.repository.ClearDefault(tx, value.SchemeType, 0, value.ScopeCode, 0); err != nil {
				return err
			}
		}
		if err := service.repository.Create(tx, &value); err != nil {
			return err
		}
		if err := service.repository.ReplaceRoles(tx.WithContext(ctx), value.Id, req.RoleIDs); err != nil {
			return err
		}
		return service.writeAudit(ctx, tx, value, "create_shared")
	})
	if err != nil {
		return response.QuerySchemeDetailRes{}, querySchemePersistenceError(err)
	}
	return service.detailResponse(ctx, value, payload, queryscheme.ValidationResult{Status: queryscheme.ValidationValid}, req.RoleIDs)
}

func (service *QuerySchemeService) UpdateShared(
	ctx context.Context,
	id int,
	req request.QuerySchemeSharedUpdateReq,
) (response.QuerySchemeDetailRes, error) {
	actor, err := service.actor(ctx)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	if err := service.requireSharedManager(ctx, actor.UserID); err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	name, err := normalizeSchemeName(req.Name)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	current, config, err := service.preflightMutation(ctx, actor, id, req.Revision, true)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	if current.SchemeType != model.QuerySchemeTypePageDefault && req.IsDefault {
		return response.QuerySchemeDetailRes{}, myerrors.ErrQuerySchemeTypeInvalid
	}
	if err := service.validateRoles(ctx, current.SchemeType, req.RoleIDs); err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	requestPayload, err := decodeQuerySchemeRequestPayload(req.Payload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	_, raw, _, err := service.preparePayload(ctx, config, requestPayload)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	var updated model.QueryScheme
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		locked, findErr := service.repository.FindByIDWithDB(tx.WithContext(ctx), id, true)
		if findErr != nil {
			return querySchemeLookupError(findErr)
		}
		if mutationErr := validateLockedMutation(locked, current, actor.UserID, req.Revision, true); mutationErr != nil {
			return mutationErr
		}
		if locked.SchemeType == model.QuerySchemeTypePageDefault && req.IsDefault && locked.Enabled {
			if err := service.repository.ClearDefault(tx, locked.SchemeType, 0, locked.ScopeCode, locked.Id); err != nil {
				return err
			}
		}
		matched, updateErr := service.repository.UpdateFieldsByRevision(tx.WithContext(ctx), id, req.Revision, map[string]any{
			"name":                 name,
			"query_payload":        datatypes.JSON(raw),
			"query_schema_version": queryscheme.SchemaVersion,
			"is_default":           req.IsDefault,
			"revision":             gorm.Expr("revision + 1"),
		})
		if updateErr != nil {
			return updateErr
		}
		if !matched {
			return myerrors.ErrQuerySchemeRevisionConflict
		}
		if err := service.repository.ReplaceRoles(tx.WithContext(ctx), id, req.RoleIDs); err != nil {
			return err
		}
		updated, updateErr = service.repository.FindByIDWithDB(tx.WithContext(ctx), id, false)
		if updateErr != nil {
			return updateErr
		}
		return service.writeAudit(ctx, tx, updated, "update_shared")
	})
	if err != nil {
		return response.QuerySchemeDetailRes{}, querySchemePersistenceError(err)
	}
	return service.buildDetail(ctx, updated)
}

func (service *QuerySchemeService) DeleteShared(ctx context.Context, id, revision int) error {
	return service.deleteScheme(ctx, id, revision, true)
}

func (service *QuerySchemeService) SetSharedEnabled(
	ctx context.Context,
	id int,
	req request.QuerySchemeEnabledReq,
) (response.QuerySchemeDetailRes, error) {
	actor, err := service.actor(ctx)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	if err := service.requireSharedManager(ctx, actor.UserID); err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	current, _, err := service.preflightMutation(ctx, actor, id, req.Revision, true)
	if err != nil {
		return response.QuerySchemeDetailRes{}, err
	}
	var updated model.QueryScheme
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		locked, findErr := service.repository.FindByIDWithDB(tx.WithContext(ctx), id, true)
		if findErr != nil {
			return querySchemeLookupError(findErr)
		}
		if mutationErr := validateLockedMutation(locked, current, actor.UserID, req.Revision, true); mutationErr != nil {
			return mutationErr
		}
		if req.Enabled && locked.SchemeType == model.QuerySchemeTypePageDefault && locked.IsDefault {
			if err := service.repository.ClearDefault(tx, locked.SchemeType, 0, locked.ScopeCode, locked.Id); err != nil {
				return err
			}
		}
		matched, updateErr := service.repository.UpdateFieldsByRevision(tx.WithContext(ctx), id, req.Revision, map[string]any{
			"enabled":  req.Enabled,
			"revision": gorm.Expr("revision + 1"),
		})
		if updateErr != nil {
			return updateErr
		}
		if !matched {
			return myerrors.ErrQuerySchemeRevisionConflict
		}
		updated, updateErr = service.repository.FindByIDWithDB(tx.WithContext(ctx), id, false)
		if updateErr != nil {
			return updateErr
		}
		return service.writeAudit(ctx, tx, updated, "set_enabled")
	})
	if err != nil {
		return response.QuerySchemeDetailRes{}, querySchemePersistenceError(err)
	}
	return service.buildDetail(ctx, updated)
}

func (service *QuerySchemeService) deleteScheme(ctx context.Context, id, revision int, shared bool) error {
	actor, err := service.actor(ctx)
	if err != nil {
		return err
	}
	if shared {
		if err := service.requireSharedManager(ctx, actor.UserID); err != nil {
			return err
		}
	}
	current, _, err := service.preflightMutation(ctx, actor, id, revision, shared)
	if err != nil {
		return err
	}
	err = RunInTransaction(ctx, service.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		locked, findErr := service.repository.FindByIDWithDB(tx.WithContext(ctx), id, true)
		if findErr != nil {
			return querySchemeLookupError(findErr)
		}
		if mutationErr := validateLockedMutation(locked, current, actor.UserID, revision, shared); mutationErr != nil {
			return mutationErr
		}
		deleted, deleteErr := service.repository.DeleteByRevision(tx.WithContext(ctx), id, revision)
		if deleteErr != nil {
			return deleteErr
		}
		if !deleted {
			return myerrors.ErrQuerySchemeRevisionConflict
		}
		return service.writeAudit(ctx, tx, locked, map[bool]string{true: "delete_shared", false: "delete"}[shared])
	})
	if err != nil {
		return querySchemePersistenceError(err)
	}
	return nil
}

func (service *QuerySchemeService) preflightMutation(
	ctx context.Context,
	actor queryscheme.Subject,
	id, revision int,
	shared bool,
) (model.QueryScheme, queryscheme.ScopeConfig, error) {
	current, err := service.repository.FindByIDWithDB(service.repository.DBWithContext(ctx), id, false)
	if err != nil {
		return model.QueryScheme{}, queryscheme.ScopeConfig{}, querySchemeLookupError(err)
	}
	if err := validateSchemeMutation(current, actor.UserID, revision, shared); err != nil {
		return model.QueryScheme{}, queryscheme.ScopeConfig{}, err
	}
	config, _, err := service.authorizeScopeForActor(ctx, actor, current.ScopeCode)
	return current, config, err
}

func validateLockedMutation(
	locked model.QueryScheme,
	preflight model.QueryScheme,
	userID, revision int,
	shared bool,
) error {
	if locked.ScopeCode != preflight.ScopeCode || locked.SchemeType != preflight.SchemeType {
		return myerrors.ErrQuerySchemeRevisionConflict
	}
	return validateSchemeMutation(locked, userID, revision, shared)
}

func validateSchemeMutation(value model.QueryScheme, userID, revision int, shared bool) error {
	if shared != sharedSchemeType(value.SchemeType) {
		return myerrors.ErrQuerySchemeTypeInvalid
	}
	if !shared && (value.OwnerUserID == nil || *value.OwnerUserID != userID) {
		return myerrors.ErrQuerySchemeOwnerForbidden
	}
	if value.Revision != revision {
		return myerrors.ErrQuerySchemeRevisionConflict
	}
	return nil
}

func (service *QuerySchemeService) preparePayload(
	ctx context.Context,
	config queryscheme.ScopeConfig,
	payload queryscheme.QuerySchemePayloadV1,
) (queryscheme.QuerySchemePayloadV1, []byte, queryscheme.ValidationResult, error) {
	normalized := queryscheme.Normalize(payload)
	schemaResult := queryscheme.ValidateSchema(normalized)
	if schemaResult.Status != queryscheme.ValidationValid {
		for _, issue := range schemaResult.Issues {
			if issue.Code == queryscheme.IssueBindingUnavailable {
				return normalized, nil, schemaResult, myerrors.ErrQuerySchemeBindingInvalid
			}
		}
		return normalized, nil, schemaResult, myerrors.ErrQuerySchemePayloadInvalid
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return normalized, nil, schemaResult, myerrors.WrapSystemError(err)
	}
	if len(raw) > queryscheme.MaxPayloadBytes {
		return normalized, nil, schemaResult, myerrors.ErrQuerySchemePayloadTooLarge
	}
	metadataResult, err := service.validator.ValidateMetadata(ctx, config, normalized)
	if err != nil {
		return normalized, nil, metadataResult, myerrors.WrapSystemError(err)
	}
	if metadataResult.Status == queryscheme.ValidationDegraded {
		return normalized, nil, metadataResult, myerrors.ErrQuerySchemeMetadataDegraded
	}
	if metadataResult.Status != queryscheme.ValidationValid {
		return normalized, nil, metadataResult, myerrors.ErrQuerySchemeInvalid
	}
	return normalized, raw, metadataResult, nil
}

func decodeQuerySchemeRequestPayload(raw json.RawMessage) (queryscheme.QuerySchemePayloadV1, error) {
	if len(raw) > queryscheme.MaxPayloadBytes {
		return queryscheme.QuerySchemePayloadV1{}, myerrors.ErrQuerySchemePayloadTooLarge
	}
	payload, err := queryscheme.DecodePayload(raw)
	if err != nil {
		return queryscheme.QuerySchemePayloadV1{}, myerrors.ErrQuerySchemePayloadInvalid
	}
	return payload, nil
}

func normalizeSchemeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > queryscheme.MaxNameLength {
		return "", myerrors.ErrQuerySchemePayloadInvalid
	}
	return value, nil
}

func (service *QuerySchemeService) nextID() (int, error) {
	if service == nil || service.sf == nil {
		return 0, myerrors.WrapSystemError(errors.New("query scheme id generator is unavailable"))
	}
	id, err := service.sf.GenerateUniqueID()
	if err != nil {
		return 0, myerrors.WrapSystemError(err)
	}
	return int(id), nil
}

func sharedSchemeType(value model.QuerySchemeType) bool {
	return value == model.QuerySchemeTypePublic || value == model.QuerySchemeTypeRole || value == model.QuerySchemeTypePageDefault
}

func querySchemeLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return myerrors.ErrQuerySchemeNotFound
	}
	return myerrors.WrapDatabaseError(err)
}

func querySchemePersistenceError(err error) error {
	if _, ok := myerrors.AsApplicationError(err); ok {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "uni_query_scheme_personal_default", "uni_query_scheme_page_default":
			return myerrors.ErrQuerySchemeDefaultConflict
		default:
			return myerrors.ErrQuerySchemeNameConflict
		}
	}
	return myerrors.WrapDatabaseError(err)
}

func (service *QuerySchemeService) writeAudit(ctx context.Context, tx *gorm.DB, scheme model.QueryScheme, action string) error {
	if service.audit == nil {
		return nil
	}
	return service.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: "query_scheme",
		ResourceCode: scheme.ScopeCode,
		ResourceId:   strconv.Itoa(scheme.Id),
		Changes: map[string]TransactionalAuditChange{
			"scheme": {NewValue: map[string]any{"type": scheme.SchemeType, "scope": scheme.ScopeCode, "action": action}},
		},
	})
}
