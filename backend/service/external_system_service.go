package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	externalSystemAuditResourceType = "integration_external_system"
	externalSystemAuditCreate       = "integration.external_system.create"
	externalSystemAuditUpdate       = "integration.external_system.update"
	externalSystemAuditEnable       = "integration.external_system.enable"
	externalSystemAuditDisable      = "integration.external_system.disable"
)

var externalSystemCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

type ExternalSystemService struct {
	repository repository.ExternalSystemRepository
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
}

func NewExternalSystemService(
	repository repository.ExternalSystemRepository,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *ExternalSystemService {
	return &ExternalSystemService{repository: repository, sf: sf, audit: audit}
}

func (s *ExternalSystemService) Create(
	ctx context.Context,
	req request.ExternalSystemCreateReq,
) (response.ExternalSystemDetailRes, error) {
	code := strings.TrimSpace(req.SystemCode)
	if !externalSystemCodePattern.MatchString(code) {
		return response.ExternalSystemDetailRes{}, myerrors.ErrExternalSystemCodeInvalid
	}
	baseURL, err := normalizeExternalSystemBaseURL(req.BaseURL)
	if err != nil {
		return response.ExternalSystemDetailRes{}, err
	}
	value := model.ExternalSystem{
		SystemCode:      code,
		Name:            strings.TrimSpace(req.Name),
		SystemType:      strings.TrimSpace(req.SystemType),
		BaseURL:         baseURL,
		OwnerIdentifier: strings.TrimSpace(req.OwnerIdentifier),
		OwnerName:       strings.TrimSpace(req.OwnerName),
		Status:          model.ExternalSystemStatusDraft,
		Description:     strings.TrimSpace(req.Description),
		Revision:        1,
	}
	if err := validateExternalSystemRequired(value); err != nil {
		return response.ExternalSystemDetailRes{}, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return response.ExternalSystemDetailRes{}, myerrors.WrapSystemError(err)
	}
	value.Basic = model.Basic{Id: int(id), State: false}
	err = RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		if err := s.repository.Create(tx, &value); err != nil {
			if isExternalSystemDuplicate(err) {
				return myerrors.ErrExternalSystemCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, externalSystemAuditCreate, value, nil)
	})
	if err != nil {
		return response.ExternalSystemDetailRes{}, err
	}
	return response.NewExternalSystemDetailRes(value), nil
}

func (s *ExternalSystemService) Get(
	ctx context.Context,
	id int,
) (response.ExternalSystemDetailRes, error) {
	value, err := s.repository.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.ExternalSystemDetailRes{}, myerrors.ErrExternalSystemNotFound
	}
	if err != nil {
		return response.ExternalSystemDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewExternalSystemDetailRes(value), nil
}

func (s *ExternalSystemService) Page(
	ctx context.Context,
	req request.ExternalSystemQueryReq,
	table model.SysTable,
) (response.ListResult[response.ExternalSystemListRes], error) {
	basic := req.ToBasic()
	result, err := s.repository.GetExternalSystemList(ctx, &basic, table)
	if err != nil {
		return response.ListResult[response.ExternalSystemListRes]{}, myerrors.WrapDatabaseError(err)
	}
	items := make([]response.ExternalSystemListRes, 0, len(result.Data))
	for _, value := range result.Data {
		items = append(items, response.NewExternalSystemListRes(value))
	}
	return response.ListResult[response.ExternalSystemListRes]{Data: items, Total: result.Total}, nil
}

func (s *ExternalSystemService) Update(
	ctx context.Context,
	id int,
	req request.ExternalSystemUpdateReq,
) (response.ExternalSystemDetailRes, error) {
	var updated model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrExternalSystemNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if req.SystemCode != nil && strings.TrimSpace(*req.SystemCode) != current.SystemCode {
			return myerrors.ErrExternalSystemFieldImmutable
		}
		updates, next, err := s.externalSystemUpdates(tx, current, req)
		if err != nil {
			return err
		}
		updates["revision"] = current.Revision + 1
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, req.Revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrExternalSystemRevisionConflict
		}
		next.Revision = current.Revision + 1
		updated = next
		return s.writeAudit(ctx, tx, externalSystemAuditUpdate, next, &current)
	})
	if err != nil {
		return response.ExternalSystemDetailRes{}, err
	}
	return response.NewExternalSystemDetailRes(updated), nil
}

func (s *ExternalSystemService) Enable(ctx context.Context, id, revision int) (response.ExternalSystemDetailRes, error) {
	return s.changeStatus(ctx, id, revision, model.ExternalSystemStatusEnabled)
}

func (s *ExternalSystemService) Disable(ctx context.Context, id, revision int) (response.ExternalSystemDetailRes, error) {
	return s.changeStatus(ctx, id, revision, model.ExternalSystemStatusDisabled)
}

func (s *ExternalSystemService) changeStatus(
	ctx context.Context,
	id int,
	revision int,
	target string,
) (response.ExternalSystemDetailRes, error) {
	var updated model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrExternalSystemNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status == target {
			updated = current
			return nil
		}
		if target == model.ExternalSystemStatusDisabled && current.Status != model.ExternalSystemStatusEnabled {
			return myerrors.ErrExternalSystemStatusInvalid
		}
		if target == model.ExternalSystemStatusEnabled {
			if current.Status != model.ExternalSystemStatusDraft && current.Status != model.ExternalSystemStatusDisabled {
				return myerrors.ErrExternalSystemStatusInvalid
			}
			if err := validateExternalSystemRequired(current); err != nil {
				return err
			}
			if _, err := normalizeExternalSystemBaseURL(current.BaseURL); err != nil {
				return err
			}
		}
		updates := map[string]any{
			"status":   target,
			"state":    target == model.ExternalSystemStatusEnabled,
			"revision": current.Revision + 1,
		}
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrExternalSystemRevisionConflict
		}
		updated = current
		updated.Status = target
		updated.State = target == model.ExternalSystemStatusEnabled
		updated.Revision++
		action := externalSystemAuditEnable
		if target == model.ExternalSystemStatusDisabled {
			action = externalSystemAuditDisable
		}
		return s.writeAudit(ctx, tx, action, updated, &current)
	})
	if err != nil {
		return response.ExternalSystemDetailRes{}, err
	}
	return response.NewExternalSystemDetailRes(updated), nil
}

func (s *ExternalSystemService) externalSystemUpdates(
	tx *gorm.DB,
	current model.ExternalSystem,
	req request.ExternalSystemUpdateReq,
) (map[string]any, model.ExternalSystem, error) {
	updates := make(map[string]any)
	next := current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
		updates["name"] = next.Name
	}
	if req.SystemType != nil {
		next.SystemType = strings.TrimSpace(*req.SystemType)
		if next.SystemType != current.SystemType {
			referenced, err := s.repository.HasConfigurationReferences(tx, current.Id)
			if err != nil {
				return nil, current, myerrors.WrapDatabaseError(err)
			}
			if referenced {
				return nil, current, myerrors.ErrExternalSystemReferenced
			}
		}
		updates["system_type"] = next.SystemType
	}
	if req.BaseURL != nil {
		normalized, err := normalizeExternalSystemBaseURL(*req.BaseURL)
		if err != nil {
			return nil, current, err
		}
		next.BaseURL = normalized
		updates["base_url"] = normalized
	}
	if req.OwnerIdentifier != nil {
		next.OwnerIdentifier = strings.TrimSpace(*req.OwnerIdentifier)
		updates["owner_identifier"] = next.OwnerIdentifier
	}
	if req.OwnerName != nil {
		next.OwnerName = strings.TrimSpace(*req.OwnerName)
		updates["owner_name"] = next.OwnerName
	}
	if req.Description != nil {
		next.Description = strings.TrimSpace(*req.Description)
		updates["description"] = next.Description
	}
	if err := validateExternalSystemRequired(next); err != nil {
		return nil, current, err
	}
	return updates, next, nil
}

func (s *ExternalSystemService) writeAudit(
	ctx context.Context,
	tx *gorm.DB,
	action string,
	value model.ExternalSystem,
	previous *model.ExternalSystem,
) error {
	changes := map[string]TransactionalAuditChange{
		"status":   {NewValue: value.Status},
		"revision": {NewValue: value.Revision},
	}
	if previous != nil {
		changes["status"] = TransactionalAuditChange{OldValue: previous.Status, NewValue: value.Status}
		changes["revision"] = TransactionalAuditChange{OldValue: previous.Revision, NewValue: value.Revision}
	}
	return s.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
		Action:       action,
		ResourceType: externalSystemAuditResourceType,
		ResourceCode: value.SystemCode,
		ResourceId:   strconv.Itoa(value.Id),
		Changes:      changes,
	})
}

func validateExternalSystemRequired(value model.ExternalSystem) error {
	if strings.TrimSpace(value.Name) == "" || strings.TrimSpace(value.SystemType) == "" ||
		strings.TrimSpace(value.BaseURL) == "" || strings.TrimSpace(value.OwnerIdentifier) == "" ||
		strings.TrimSpace(value.OwnerName) == "" {
		return myerrors.ErrExternalSystemConfiguration
	}
	if !validExternalSystemType(value.SystemType) {
		return myerrors.ErrExternalSystemConfiguration
	}
	return nil
}

func validExternalSystemType(value string) bool {
	switch value {
	case model.ExternalSystemTypeHR,
		model.ExternalSystemTypeERP,
		model.ExternalSystemTypeTMS,
		model.ExternalSystemTypeWMS,
		model.ExternalSystemTypeOther:
		return true
	default:
		return false
	}
}

func normalizeExternalSystemBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", myerrors.ErrExternalSystemBaseURLInvalid
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" {
		return "", myerrors.ErrExternalSystemBaseURLInvalid
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate()) {
		return "", myerrors.ErrExternalSystemBaseURLInvalid
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return parsed.String(), nil
}

func isExternalSystemDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return (errors.As(err, &pgErr) && pgErr.Code == "23505") ||
		strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
