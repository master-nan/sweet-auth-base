package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/integration"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	interfaceDefinitionAuditResourceType = "integration_interface_definition"
	interfaceDefinitionAuditCreate       = "integration.interface_definition.create"
	interfaceDefinitionAuditUpdate       = "integration.interface_definition.update"
	interfaceDefinitionAuditVersion      = "integration.interface_definition.create_version"
	interfaceDefinitionAuditEnable       = "integration.interface_definition.enable"
	interfaceDefinitionAuditDisable      = "integration.interface_definition.disable"
)

var interfaceCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

type InterfaceDefinitionService struct {
	repository  repository.InterfaceDefinitionRepository
	systems     repository.ExternalSystemRepository
	credentials repository.CredentialRepository
	policies    repository.RetryPolicyRepository
	sf          *utils.Snowflake
	audit       StandardContextAuditWriter
	now         func() time.Time
}

func NewInterfaceDefinitionService(
	repository repository.InterfaceDefinitionRepository,
	systems repository.ExternalSystemRepository,
	credentials repository.CredentialRepository,
	policies repository.RetryPolicyRepository,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *InterfaceDefinitionService {
	return &InterfaceDefinitionService{
		repository: repository, systems: systems, credentials: credentials, policies: policies, sf: sf, audit: audit, now: time.Now,
	}
}

func (s *InterfaceDefinitionService) Create(ctx context.Context, req request.InterfaceDefinitionCreateReq) (response.InterfaceDefinitionDetailRes, error) {
	code := strings.TrimSpace(req.InterfaceCode)
	if !interfaceCodePattern.MatchString(code) {
		return response.InterfaceDefinitionDetailRes{}, myerrors.ErrInterfaceCodeInvalid
	}
	path, err := normalizeInterfaceRelativePath(req.RelativePath)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	contract, err := integration.NormalizeInputContract(req.InputContract, req.HTTPMethod, path)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	value := model.InterfaceDefinition{
		ExternalSystemID: req.ExternalSystemID, InterfaceCode: code, Name: strings.TrimSpace(req.Name), Version: 1,
		Protocol: strings.ToLower(strings.TrimSpace(req.Protocol)), HTTPMethod: strings.ToUpper(strings.TrimSpace(req.HTTPMethod)),
		RelativePath: path, InputContract: datatypes.JSON(contract), CredentialID: req.CredentialID, TimeoutSeconds: req.TimeoutSeconds,
		ResponseLimit: req.ResponseLimit, RetryPolicyID: req.RetryPolicyID,
		Status: model.InterfaceDefinitionStatusDraft, Description: strings.TrimSpace(req.Description), Revision: 1,
	}
	if err := validateInterfaceConfiguration(value); err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, myerrors.WrapSystemError(err)
	}
	value.Basic = model.Basic{Id: int(id), State: false}
	var system model.ExternalSystem
	err = RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		var err error
		system, err = s.loadSystemForConfiguration(tx, value.ExternalSystemID, false)
		if err != nil {
			return err
		}
		if _, err := s.validateReferences(tx, value, false); err != nil {
			return err
		}
		if err := s.repository.Create(tx, &value); err != nil {
			if isInterfaceDefinitionDuplicate(err) {
				return myerrors.ErrInterfaceCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, interfaceDefinitionAuditCreate, value, system, nil)
	})
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	credential, err := s.loadCredentialSummary(ctx, value.CredentialID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	policy, err := s.loadRetryPolicySummary(ctx, value.RetryPolicyID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	return response.NewInterfaceDefinitionDetailRes(value, system, credential, s.now(), policy), nil
}

func (s *InterfaceDefinitionService) Get(ctx context.Context, id int) (response.InterfaceDefinitionDetailRes, error) {
	value, err := s.repository.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.InterfaceDefinitionDetailRes{}, myerrors.ErrInterfaceDefinitionNotFound
	}
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	system, err := s.systems.WithContext(ctx).FindById(value.ExternalSystemID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	credential, err := s.loadCredentialSummary(ctx, value.CredentialID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	policy, err := s.loadRetryPolicySummary(ctx, value.RetryPolicyID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	return response.NewInterfaceDefinitionDetailRes(value, system, credential, s.now(), policy), nil
}

func (s *InterfaceDefinitionService) Page(ctx context.Context, req request.InterfaceDefinitionQueryReq, table model.SysTable) (response.ListResult[response.InterfaceDefinitionListRes], error) {
	basic := req.ToBasic()
	result, err := s.repository.GetInterfaceDefinitionList(ctx, &basic, table)
	if err != nil {
		return response.ListResult[response.InterfaceDefinitionListRes]{}, myerrors.WrapDatabaseError(err)
	}
	ids := make([]int, 0, len(result.Data))
	credentialIDs := make([]int, 0, len(result.Data))
	policyIDs := make([]int, 0, len(result.Data))
	seen := make(map[int]struct{}, len(result.Data))
	for _, item := range result.Data {
		if _, exists := seen[item.ExternalSystemID]; !exists {
			seen[item.ExternalSystemID] = struct{}{}
			ids = append(ids, item.ExternalSystemID)
		}
		if item.CredentialID != nil {
			credentialIDs = append(credentialIDs, *item.CredentialID)
		}
		if item.RetryPolicyID != nil {
			policyIDs = append(policyIDs, *item.RetryPolicyID)
		}
	}
	systems, err := s.systems.WithContext(ctx).FindListByFieldIn("id", ids)
	if err != nil {
		return response.ListResult[response.InterfaceDefinitionListRes]{}, myerrors.WrapDatabaseError(err)
	}
	byID := make(map[int]model.ExternalSystem, len(systems))
	for _, system := range systems {
		byID[system.Id] = system
	}
	credentials, err := s.credentials.WithContext(ctx).FindListByFieldIn("id", credentialIDs)
	if err != nil {
		return response.ListResult[response.InterfaceDefinitionListRes]{}, myerrors.WrapDatabaseError(err)
	}
	credentialsByID := make(map[int]model.Credential, len(credentials))
	for _, credential := range credentials {
		credentialsByID[credential.Id] = credential
	}
	policies, err := s.policies.WithContext(ctx).FindListByFieldIn("id", policyIDs)
	if err != nil {
		return response.ListResult[response.InterfaceDefinitionListRes]{}, myerrors.WrapDatabaseError(err)
	}
	policiesByID := make(map[int]model.RetryPolicy, len(policies))
	for _, policy := range policies {
		policiesByID[policy.Id] = policy
	}
	now := s.now()
	items := make([]response.InterfaceDefinitionListRes, 0, len(result.Data))
	for _, value := range result.Data {
		var credential *model.Credential
		if value.CredentialID != nil {
			if item, ok := credentialsByID[*value.CredentialID]; ok {
				credential = &item
			}
		}
		var policy *model.RetryPolicy
		if value.RetryPolicyID != nil {
			if item, ok := policiesByID[*value.RetryPolicyID]; ok {
				policy = &item
			}
		}
		items = append(items, response.NewInterfaceDefinitionListRes(value, byID[value.ExternalSystemID], credential, now, policy))
	}
	return response.ListResult[response.InterfaceDefinitionListRes]{Data: items, Total: result.Total}, nil
}

func (s *InterfaceDefinitionService) Update(ctx context.Context, id int, req request.InterfaceDefinitionUpdateReq) (response.InterfaceDefinitionDetailRes, error) {
	var updated model.InterfaceDefinition
	var system model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrInterfaceDefinitionNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status != model.InterfaceDefinitionStatusDraft {
			return myerrors.ErrInterfaceStatusInvalid
		}
		if interfaceIdentityChanged(current, req) {
			return myerrors.ErrInterfaceFieldImmutable
		}
		updates, next, err := interfaceDefinitionUpdates(current, req)
		if err != nil {
			return err
		}
		system, err = s.loadSystemForConfiguration(tx, current.ExternalSystemID, false)
		if err != nil {
			return err
		}
		if _, err := s.validateReferences(tx, next, false); err != nil {
			return err
		}
		updates["revision"] = current.Revision + 1
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, req.Revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrInterfaceRevisionConflict
		}
		next.Revision = current.Revision + 1
		updated = next
		return s.writeAudit(ctx, tx, interfaceDefinitionAuditUpdate, updated, system, &current)
	})
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	credential, err := s.loadCredentialSummary(ctx, updated.CredentialID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	policy, err := s.loadRetryPolicySummary(ctx, updated.RetryPolicyID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	return response.NewInterfaceDefinitionDetailRes(updated, system, credential, s.now(), policy), nil
}

func (s *InterfaceDefinitionService) CreateVersion(ctx context.Context, id, revision int) (response.InterfaceDefinitionDetailRes, error) {
	var created model.InterfaceDefinition
	var system model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrInterfaceDefinitionNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Revision != revision {
			return myerrors.ErrInterfaceRevisionConflict
		}
		if current.Status == model.InterfaceDefinitionStatusDraft {
			return myerrors.ErrInterfaceStatusInvalid
		}
		nextVersion, err := s.repository.NextVersion(tx, current.ExternalSystemID, current.InterfaceCode)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		newID, err := s.sf.GenerateUniqueID()
		if err != nil {
			return myerrors.WrapSystemError(err)
		}
		created = current
		created.Basic = model.Basic{Id: int(newID), State: false}
		created.Version = nextVersion
		created.Status = model.InterfaceDefinitionStatusDraft
		created.Revision = 1
		system, err = s.loadSystemForConfiguration(tx, created.ExternalSystemID, false)
		if err != nil {
			return err
		}
		if err := s.repository.Create(tx, &created); err != nil {
			if isInterfaceDefinitionDuplicate(err) {
				return myerrors.ErrInterfaceCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, interfaceDefinitionAuditVersion, created, system, nil)
	})
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	credential, err := s.loadCredentialSummary(ctx, created.CredentialID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	policy, err := s.loadRetryPolicySummary(ctx, created.RetryPolicyID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	return response.NewInterfaceDefinitionDetailRes(created, system, credential, s.now(), policy), nil
}

func (s *InterfaceDefinitionService) Enable(ctx context.Context, id, revision int) (response.InterfaceDefinitionDetailRes, error) {
	return s.changeStatus(ctx, id, revision, model.InterfaceDefinitionStatusEnabled)
}

func (s *InterfaceDefinitionService) Disable(ctx context.Context, id, revision int) (response.InterfaceDefinitionDetailRes, error) {
	return s.changeStatus(ctx, id, revision, model.InterfaceDefinitionStatusDisabled)
}

func (s *InterfaceDefinitionService) changeStatus(ctx context.Context, id, revision int, target string) (response.InterfaceDefinitionDetailRes, error) {
	var updated model.InterfaceDefinition
	var system model.ExternalSystem
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrInterfaceDefinitionNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status == target {
			updated = current
			system, err = s.loadSystemForConfiguration(tx, current.ExternalSystemID, target == model.InterfaceDefinitionStatusEnabled)
			return err
		}
		if target == model.InterfaceDefinitionStatusDisabled && current.Status != model.InterfaceDefinitionStatusEnabled {
			return myerrors.ErrInterfaceStatusInvalid
		}
		if target == model.InterfaceDefinitionStatusEnabled && current.Status != model.InterfaceDefinitionStatusDraft && current.Status != model.InterfaceDefinitionStatusDisabled {
			return myerrors.ErrInterfaceStatusInvalid
		}
		system, err = s.loadSystemForConfiguration(tx, current.ExternalSystemID, target == model.InterfaceDefinitionStatusEnabled)
		if err != nil {
			return err
		}
		if target == model.InterfaceDefinitionStatusEnabled {
			if err := validateInterfaceConfiguration(current); err != nil {
				if errors.Is(err, myerrors.ErrIntegrationTimeoutOutOfRange) || errors.Is(err, myerrors.ErrIntegrationResponseLimitOutOfRange) {
					return myerrors.ErrIntegrationInterfaceRuntimeIncompatible
				}
				return err
			}
			if _, err := s.validateReferences(tx, current, true); err != nil {
				return err
			}
			conflict, err := s.repository.HasEnabledVersion(tx, current.ExternalSystemID, current.InterfaceCode, current.Id)
			if err != nil {
				return myerrors.WrapDatabaseError(err)
			}
			if conflict {
				return myerrors.ErrInterfaceEnabledVersionConflict
			}
		}
		updates := map[string]any{"status": target, "state": target == model.InterfaceDefinitionStatusEnabled, "revision": current.Revision + 1}
		ok, err := s.repository.UpdateFieldsByRevision(tx, current.Id, revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrInterfaceRevisionConflict
		}
		updated = current
		updated.Status = target
		updated.State = target == model.InterfaceDefinitionStatusEnabled
		updated.Revision++
		action := interfaceDefinitionAuditEnable
		if target == model.InterfaceDefinitionStatusDisabled {
			action = interfaceDefinitionAuditDisable
		}
		return s.writeAudit(ctx, tx, action, updated, system, &current)
	})
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	credential, err := s.loadCredentialSummary(ctx, updated.CredentialID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	policy, err := s.loadRetryPolicySummary(ctx, updated.RetryPolicyID)
	if err != nil {
		return response.InterfaceDefinitionDetailRes{}, err
	}
	return response.NewInterfaceDefinitionDetailRes(updated, system, credential, s.now(), policy), nil
}

func (s *InterfaceDefinitionService) loadSystemForConfiguration(tx *gorm.DB, id int, requireEnabled bool) (model.ExternalSystem, error) {
	system, err := s.systems.FindByIdForUpdate(tx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ExternalSystem{}, myerrors.ErrInterfaceExternalSystemInvalid
	}
	if err != nil {
		return model.ExternalSystem{}, myerrors.WrapDatabaseError(err)
	}
	if system.Status == model.ExternalSystemStatusDisabled || requireEnabled && system.Status != model.ExternalSystemStatusEnabled {
		return model.ExternalSystem{}, myerrors.ErrInterfaceExternalSystemInvalid
	}
	return system, nil
}

func (s *InterfaceDefinitionService) validateReferences(tx *gorm.DB, value model.InterfaceDefinition, requireUsable bool) (*model.Credential, error) {
	var credential *model.Credential
	if value.CredentialID != nil {
		item, err := s.credentials.FindByIdForUpdate(tx, *value.CredentialID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, myerrors.ErrInterfaceCredentialInvalid
		}
		if err != nil {
			return nil, myerrors.WrapDatabaseError(err)
		}
		if item.ExternalSystemID != value.ExternalSystemID || item.Status == model.CredentialStatusRevoked {
			return nil, myerrors.ErrInterfaceCredentialInvalid
		}
		if requireUsable && (item.Status != model.CredentialStatusActive || item.ExpiresAt != nil && !item.ExpiresAt.After(s.now())) {
			return nil, myerrors.ErrInterfaceCredentialInvalid
		}
		credential = &item
	}
	if value.RetryPolicyID != nil {
		policy, err := s.policies.FindByIdForUpdate(tx, *value.RetryPolicyID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, myerrors.ErrInterfaceRetryPolicyInvalid
		}
		if err != nil {
			return nil, myerrors.WrapDatabaseError(err)
		}
		if policy.Status != model.RetryPolicyStatusEnabled || !policy.State {
			return nil, myerrors.ErrInterfaceRetryPolicyInvalid
		}
	}
	return credential, nil
}

func (s *InterfaceDefinitionService) loadRetryPolicySummary(ctx context.Context, id *int) (*model.RetryPolicy, error) {
	if id == nil {
		return nil, nil
	}
	policy, err := s.policies.WithContext(ctx).FindById(*id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	return &policy, nil
}

func (s *InterfaceDefinitionService) loadCredentialSummary(ctx context.Context, id *int) (*model.Credential, error) {
	if id == nil {
		return nil, nil
	}
	credential, err := s.credentials.WithContext(ctx).FindById(*id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, myerrors.WrapDatabaseError(err)
	}
	return &credential, nil
}

func (s *InterfaceDefinitionService) writeAudit(ctx context.Context, tx *gorm.DB, action string, value model.InterfaceDefinition, system model.ExternalSystem, previous *model.InterfaceDefinition) error {
	changes := map[string]TransactionalAuditChange{
		"system": {NewValue: system.SystemCode}, "interface_code": {NewValue: value.InterfaceCode},
		"version": {NewValue: value.Version}, "status": {NewValue: value.Status}, "revision": {NewValue: value.Revision},
	}
	if previous != nil {
		changes["status"] = TransactionalAuditChange{OldValue: previous.Status, NewValue: value.Status}
		changes["revision"] = TransactionalAuditChange{OldValue: previous.Revision, NewValue: value.Revision}
	}
	return s.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
		Action: action, ResourceType: interfaceDefinitionAuditResourceType,
		ResourceCode: system.SystemCode + "/" + value.InterfaceCode + "@" + strconv.Itoa(value.Version),
		ResourceId:   strconv.Itoa(value.Id), Changes: changes,
	})
}

func interfaceIdentityChanged(current model.InterfaceDefinition, req request.InterfaceDefinitionUpdateReq) bool {
	return req.ExternalSystemID != nil && *req.ExternalSystemID != current.ExternalSystemID ||
		req.InterfaceCode != nil && strings.TrimSpace(*req.InterfaceCode) != current.InterfaceCode ||
		req.Version != nil && *req.Version != current.Version
}

func interfaceDefinitionUpdates(current model.InterfaceDefinition, req request.InterfaceDefinitionUpdateReq) (map[string]any, model.InterfaceDefinition, error) {
	next := current
	updates := make(map[string]any)
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
		updates["name"] = next.Name
	}
	if req.Protocol != nil {
		next.Protocol = strings.ToLower(strings.TrimSpace(*req.Protocol))
		updates["protocol"] = next.Protocol
	}
	if req.HTTPMethod != nil {
		next.HTTPMethod = strings.ToUpper(strings.TrimSpace(*req.HTTPMethod))
		updates["http_method"] = next.HTTPMethod
	}
	if req.RelativePath != nil {
		path, err := normalizeInterfaceRelativePath(*req.RelativePath)
		if err != nil {
			return nil, current, err
		}
		next.RelativePath = path
		updates["relative_path"] = path
	}
	contractChanged := len(req.InputContract) > 0
	if contractChanged {
		next.InputContract = append(datatypes.JSON(nil), req.InputContract...)
	}
	if req.ClearCredential {
		next.CredentialID = nil
		updates["credential_id"] = nil
	} else if req.CredentialID != nil {
		next.CredentialID = req.CredentialID
		updates["credential_id"] = *req.CredentialID
	}
	if req.TimeoutSeconds != nil {
		next.TimeoutSeconds = *req.TimeoutSeconds
		updates["timeout_seconds"] = next.TimeoutSeconds
	}
	if req.ResponseLimit != nil {
		next.ResponseLimit = *req.ResponseLimit
		updates["response_limit"] = next.ResponseLimit
	}
	if req.ClearRetryPolicy {
		next.RetryPolicyID = nil
		updates["retry_policy_id"] = nil
	} else if req.RetryPolicyID != nil {
		next.RetryPolicyID = req.RetryPolicyID
		updates["retry_policy_id"] = *req.RetryPolicyID
	}
	if req.Description != nil {
		next.Description = strings.TrimSpace(*req.Description)
		updates["description"] = next.Description
	}
	if err := validateInterfaceConfiguration(next); err != nil {
		return nil, current, err
	}
	if contractChanged {
		contract, err := integration.NormalizeInputContract(next.InputContract, next.HTTPMethod, next.RelativePath)
		if err != nil {
			return nil, current, err
		}
		next.InputContract = datatypes.JSON(contract)
		updates["input_contract"] = next.InputContract
	}
	return updates, next, nil
}

func validateInterfaceConfiguration(value model.InterfaceDefinition) error {
	if strings.TrimSpace(value.Name) == "" || !interfaceCodePattern.MatchString(value.InterfaceCode) ||
		!validInterfaceProtocol(value.Protocol) || !validInterfaceMethod(value.HTTPMethod) || value.Version <= 0 ||
		value.TimeoutSeconds <= 0 || value.ResponseLimit <= 0 {
		return myerrors.ErrInterfaceConfigurationInvalid
	}
	if err := integration.ValidateInterfaceRuntimeContract(value.TimeoutSeconds, value.ResponseLimit); err != nil {
		return err
	}
	path, err := normalizeInterfaceRelativePath(value.RelativePath)
	if err != nil {
		return err
	}
	_, err = integration.NormalizeInputContract(value.InputContract, value.HTTPMethod, path)
	return err
}

func validInterfaceProtocol(value string) bool {
	return value == model.InterfaceProtocolHTTP || value == model.InterfaceProtocolHTTPS
}

func validInterfaceMethod(value string) bool {
	switch value {
	case model.InterfaceMethodGET, model.InterfaceMethodPOST, model.InterfaceMethodPUT, model.InterfaceMethodPATCH, model.InterfaceMethodDELETE:
		return true
	default:
		return false
	}
}

func normalizeInterfaceRelativePath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > 512 || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") {
		return "", myerrors.ErrInterfacePathInvalid
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", myerrors.ErrInterfacePathInvalid
		}
	}
	lower := strings.ToLower(value)
	for _, forbidden := range []string{"%2e", "%2f", "%5c"} {
		if strings.Contains(lower, forbidden) {
			return "", myerrors.ErrInterfacePathInvalid
		}
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", myerrors.ErrInterfacePathInvalid
	}
	decoded, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return "", myerrors.ErrInterfacePathInvalid
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return "", myerrors.ErrInterfacePathInvalid
		}
	}
	return parsed.Path, nil
}

func isInterfaceDefinitionDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint") || strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}
