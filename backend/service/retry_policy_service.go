package service

import (
	"backend/dto/request"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/utils"
	"backend/model"
	"backend/repository"
	"context"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	retryPolicyAuditResourceType = "integration_retry_policy"
	retryPolicyAuditCreate       = "integration.retry_policy.create"
	retryPolicyAuditUpdate       = "integration.retry_policy.update"
	retryPolicyAuditVersion      = "integration.retry_policy.create_version"
	retryPolicyAuditEnable       = "integration.retry_policy.enable"
	retryPolicyAuditDisable      = "integration.retry_policy.disable"

	retryPolicyDefaultMaxAttempts       = 3
	retryPolicyDefaultInitialDelayMs    = int64(5_000)
	retryPolicyDefaultMaxDelayMs        = int64(300_000)
	retryPolicyDefaultBackoffMultiplier = 2.0
	retryPolicyDefaultJitterRatio       = 1.0
	retryPolicyDefaultWindowMs          = int64(86_400_000)
	retryPolicyMinInitialDelayMs        = int64(1_000)
	retryPolicyMaxInitialDelayMs        = int64(3_600_000)
	retryPolicyMaxDelayMs               = int64(86_400_000)
	retryPolicyMinWindowMs              = int64(60_000)
	retryPolicyMaxWindowMs              = int64(604_800_000)
)

var retryPolicyCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

var retryPolicyAllowedErrorCategories = map[string]struct{}{
	"network": {},
	"timeout": {},
	"remote":  {},
}

var retryPolicyAllowedHTTPStatuses = map[int]struct{}{
	429: {},
	502: {},
	503: {},
	504: {},
}

type RetryPolicyService struct {
	repository repository.RetryPolicyRepository
	sf         *utils.Snowflake
	audit      StandardContextAuditWriter
}

func NewRetryPolicyService(
	repository repository.RetryPolicyRepository,
	sf *utils.Snowflake,
	audit StandardContextAuditWriter,
) *RetryPolicyService {
	return &RetryPolicyService{repository: repository, sf: sf, audit: audit}
}

func (s *RetryPolicyService) CreateRetryPolicy(ctx context.Context, req request.RetryPolicyCreateReq) (response.RetryPolicyDetailRes, error) {
	value, err := newRetryPolicy(req)
	if err != nil {
		return response.RetryPolicyDetailRes{}, err
	}
	id, err := s.sf.GenerateUniqueID()
	if err != nil {
		return response.RetryPolicyDetailRes{}, myerrors.WrapSystemError(err)
	}
	value.Basic = model.Basic{Id: int(id), State: false}
	err = RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		nextVersion, err := s.repository.NextVersion(tx, value.PolicyCode)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if nextVersion != 1 {
			return myerrors.ErrRetryPolicyCodeDuplicate
		}
		if err := s.repository.Create(tx, &value); err != nil {
			if isRetryPolicyDuplicate(err) {
				return myerrors.ErrRetryPolicyCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, retryPolicyAuditCreate, value, nil)
	})
	if err != nil {
		return response.RetryPolicyDetailRes{}, err
	}
	return response.NewRetryPolicyDetailRes(value), nil
}

func (s *RetryPolicyService) PageRetryPolicy(ctx context.Context, req request.RetryPolicyQueryReq) (response.ListResult[response.RetryPolicyListRes], error) {
	basic := req.ToBasic()
	var values []model.RetryPolicy
	total, err := s.repository.WithContext(ctx).PaginateAndCountAsync(&basic, &values, retryPolicyQueryTable())
	if err != nil {
		return response.ListResult[response.RetryPolicyListRes]{}, myerrors.WrapDatabaseError(err)
	}
	items := make([]response.RetryPolicyListRes, 0, len(values))
	for _, value := range values {
		items = append(items, response.NewRetryPolicyListRes(value))
	}
	return response.ListResult[response.RetryPolicyListRes]{Data: items, Total: int(total)}, nil
}

func (s *RetryPolicyService) GetRetryPolicy(ctx context.Context, id int) (response.RetryPolicyDetailRes, error) {
	value, err := s.repository.WithContext(ctx).FindById(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.RetryPolicyDetailRes{}, myerrors.ErrRetryPolicyNotFound
	}
	if err != nil {
		return response.RetryPolicyDetailRes{}, myerrors.WrapDatabaseError(err)
	}
	return response.NewRetryPolicyDetailRes(value), nil
}

func (s *RetryPolicyService) UpdateDraftRetryPolicy(ctx context.Context, id int, req request.RetryPolicyUpdateReq) (response.RetryPolicyDetailRes, error) {
	var updated model.RetryPolicy
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrRetryPolicyNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Status != model.RetryPolicyStatusDraft {
			return myerrors.ErrRetryPolicyFieldImmutable
		}
		updates, next, err := retryPolicyUpdates(current, req)
		if err != nil {
			return err
		}
		updates["revision"] = current.Revision + 1
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, req.Revision, updates)
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrRetryPolicyRevisionConflict
		}
		next.Revision = current.Revision + 1
		updated = next
		return s.writeAudit(ctx, tx, retryPolicyAuditUpdate, updated, &current)
	})
	if err != nil {
		return response.RetryPolicyDetailRes{}, err
	}
	return response.NewRetryPolicyDetailRes(updated), nil
}

func (s *RetryPolicyService) CreateRetryPolicyVersion(ctx context.Context, id, revision int) (response.RetryPolicyDetailRes, error) {
	var created model.RetryPolicy
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrRetryPolicyNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Revision != revision {
			return myerrors.ErrRetryPolicyRevisionConflict
		}
		if current.Status == model.RetryPolicyStatusDraft {
			return myerrors.ErrRetryPolicyStatusInvalid
		}
		nextVersion, err := s.repository.NextVersion(tx, current.PolicyCode)
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
		created.Status = model.RetryPolicyStatusDraft
		created.Revision = 1
		if err := s.repository.Create(tx, &created); err != nil {
			if isRetryPolicyDuplicate(err) {
				return myerrors.ErrRetryPolicyCodeDuplicate
			}
			return myerrors.WrapDatabaseError(err)
		}
		return s.writeAudit(ctx, tx, retryPolicyAuditVersion, created, nil)
	})
	if err != nil {
		return response.RetryPolicyDetailRes{}, err
	}
	return response.NewRetryPolicyDetailRes(created), nil
}

func (s *RetryPolicyService) EnableRetryPolicy(ctx context.Context, id, revision int) (response.RetryPolicyDetailRes, error) {
	return s.changeRetryPolicyStatus(ctx, id, revision, model.RetryPolicyStatusEnabled)
}

func (s *RetryPolicyService) DisableRetryPolicy(ctx context.Context, id, revision int) (response.RetryPolicyDetailRes, error) {
	return s.changeRetryPolicyStatus(ctx, id, revision, model.RetryPolicyStatusDisabled)
}

func (s *RetryPolicyService) changeRetryPolicyStatus(ctx context.Context, id, revision int, target string) (response.RetryPolicyDetailRes, error) {
	var updated model.RetryPolicy
	err := RunInTransaction(ctx, s.repository.DBWithContext(ctx), func(tx *gorm.DB) error {
		current, err := s.repository.FindByIdForUpdate(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return myerrors.ErrRetryPolicyNotFound
		}
		if err != nil {
			return myerrors.WrapDatabaseError(err)
		}
		if current.Revision != revision {
			return myerrors.ErrRetryPolicyRevisionConflict
		}
		if current.Status == target {
			return responseRetryPolicyUnchanged(&updated, current)
		}
		if target == model.RetryPolicyStatusEnabled {
			if current.Status != model.RetryPolicyStatusDraft && current.Status != model.RetryPolicyStatusDisabled {
				return myerrors.ErrRetryPolicyStatusInvalid
			}
			if err := validateRetryPolicyConfiguration(current); err != nil {
				return err
			}
			conflict, err := s.repository.HasEnabledVersion(tx, current.PolicyCode, current.Id)
			if err != nil {
				return myerrors.WrapDatabaseError(err)
			}
			if conflict {
				return myerrors.ErrRetryPolicyEnabledConflict
			}
		} else {
			if current.Status != model.RetryPolicyStatusEnabled {
				return myerrors.ErrRetryPolicyStatusInvalid
			}
			references, err := s.repository.CountEnabledInterfaceReferences(tx, current.Id)
			if err != nil {
				return myerrors.WrapDatabaseError(err)
			}
			if references > 0 {
				return myerrors.ErrRetryPolicyReferenced
			}
		}
		updates := map[string]any{"status": target, "state": target == model.RetryPolicyStatusEnabled, "revision": current.Revision + 1}
		ok, err := s.repository.UpdateFieldsByRevision(tx, id, revision, updates)
		if err != nil {
			if isRetryPolicyDuplicate(err) {
				return myerrors.ErrRetryPolicyEnabledConflict
			}
			return myerrors.WrapDatabaseError(err)
		}
		if !ok {
			return myerrors.ErrRetryPolicyRevisionConflict
		}
		updated = current
		updated.Status = target
		updated.State = target == model.RetryPolicyStatusEnabled
		updated.Revision++
		action := retryPolicyAuditEnable
		if target == model.RetryPolicyStatusDisabled {
			action = retryPolicyAuditDisable
		}
		return s.writeAudit(ctx, tx, action, updated, &current)
	})
	if err != nil {
		return response.RetryPolicyDetailRes{}, err
	}
	return response.NewRetryPolicyDetailRes(updated), nil
}

func responseRetryPolicyUnchanged(target *model.RetryPolicy, current model.RetryPolicy) error {
	*target = current
	return nil
}

func newRetryPolicy(req request.RetryPolicyCreateReq) (model.RetryPolicy, error) {
	code := strings.TrimSpace(req.PolicyCode)
	if !retryPolicyCodePattern.MatchString(code) {
		return model.RetryPolicy{}, myerrors.ErrRetryPolicyCodeInvalid
	}
	respectRetryAfter := true
	if req.RespectRetryAfter != nil {
		respectRetryAfter = *req.RespectRetryAfter
	}
	value := model.RetryPolicy{
		PolicyCode: code, PolicyName: strings.TrimSpace(req.PolicyName), Version: 1,
		Description: strings.TrimSpace(req.Description), Status: model.RetryPolicyStatusDraft,
		MaxAttempts:       valueOrDefault(req.MaxAttempts, retryPolicyDefaultMaxAttempts),
		InitialDelayMs:    valueOrDefault(req.InitialDelayMs, retryPolicyDefaultInitialDelayMs),
		MaxDelayMs:        valueOrDefault(req.MaxDelayMs, retryPolicyDefaultMaxDelayMs),
		BackoffType:       strings.ToLower(strings.TrimSpace(req.BackoffType)),
		BackoffMultiplier: valueOrDefault(req.BackoffMultiplier, retryPolicyDefaultBackoffMultiplier),
		JitterType:        strings.ToLower(strings.TrimSpace(req.JitterType)),
		JitterRatio:       valueOrDefault(req.JitterRatio, retryPolicyDefaultJitterRatio),
		RetryWindowMs:     valueOrDefault(req.RetryWindowMs, retryPolicyDefaultWindowMs),
		RespectRetryAfter: respectRetryAfter, Revision: 1,
	}
	if value.BackoffType == "" {
		value.BackoffType = model.RetryBackoffTypeExponential
	}
	if value.JitterType == "" {
		value.JitterType = model.RetryJitterTypeFull
	}
	normalizeInactiveRetryParameters(&value, map[string]any{})
	categories := req.RetryableErrorCategories
	if categories == nil {
		categories = []string{"network", "timeout", "remote"}
	}
	statuses := req.RetryableHTTPStatuses
	if statuses == nil {
		statuses = []int{429, 502, 503, 504}
	}
	if err := setRetryPolicyLists(&value, categories, statuses); err != nil {
		return model.RetryPolicy{}, err
	}
	if err := validateRetryPolicyConfiguration(value); err != nil {
		return model.RetryPolicy{}, err
	}
	return value, nil
}

func valueOrDefault[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}

func retryPolicyUpdates(current model.RetryPolicy, req request.RetryPolicyUpdateReq) (map[string]any, model.RetryPolicy, error) {
	next := current
	updates := make(map[string]any)
	if req.PolicyName != nil {
		next.PolicyName = strings.TrimSpace(*req.PolicyName)
		updates["policy_name"] = next.PolicyName
	}
	if req.Description != nil {
		next.Description = strings.TrimSpace(*req.Description)
		updates["description"] = next.Description
	}
	copyRetryPolicyNumberUpdates(&next, updates, req)
	if req.BackoffType != nil {
		next.BackoffType = strings.ToLower(strings.TrimSpace(*req.BackoffType))
		updates["backoff_type"] = next.BackoffType
	}
	if req.JitterType != nil {
		next.JitterType = strings.ToLower(strings.TrimSpace(*req.JitterType))
		updates["jitter_type"] = next.JitterType
	}
	if req.RetryableErrorCategories != nil || req.RetryableHTTPStatuses != nil {
		categories, statuses, err := retryPolicyLists(next)
		if err != nil {
			return nil, current, myerrors.ErrRetryPolicyConfigurationInvalid
		}
		if req.RetryableErrorCategories != nil {
			categories = *req.RetryableErrorCategories
		}
		if req.RetryableHTTPStatuses != nil {
			statuses = *req.RetryableHTTPStatuses
		}
		if err := setRetryPolicyLists(&next, categories, statuses); err != nil {
			return nil, current, err
		}
		updates["retryable_error_categories"] = next.RetryableErrorCategories
		updates["retryable_http_statuses"] = next.RetryableHTTPStatuses
	}
	if req.RespectRetryAfter != nil {
		next.RespectRetryAfter = *req.RespectRetryAfter
		updates["respect_retry_after"] = next.RespectRetryAfter
	}
	normalizeInactiveRetryParameters(&next, updates)
	if err := validateRetryPolicyConfiguration(next); err != nil {
		return nil, current, err
	}
	return updates, next, nil
}

func copyRetryPolicyNumberUpdates(next *model.RetryPolicy, updates map[string]any, req request.RetryPolicyUpdateReq) {
	if req.MaxAttempts != nil {
		next.MaxAttempts = *req.MaxAttempts
		updates["max_attempts"] = next.MaxAttempts
	}
	if req.InitialDelayMs != nil {
		next.InitialDelayMs = *req.InitialDelayMs
		updates["initial_delay_ms"] = next.InitialDelayMs
	}
	if req.MaxDelayMs != nil {
		next.MaxDelayMs = *req.MaxDelayMs
		updates["max_delay_ms"] = next.MaxDelayMs
	}
	if req.BackoffMultiplier != nil {
		next.BackoffMultiplier = *req.BackoffMultiplier
		updates["backoff_multiplier"] = next.BackoffMultiplier
	}
	if req.JitterRatio != nil {
		next.JitterRatio = *req.JitterRatio
		updates["jitter_ratio"] = next.JitterRatio
	}
	if req.RetryWindowMs != nil {
		next.RetryWindowMs = *req.RetryWindowMs
		updates["retry_window_ms"] = next.RetryWindowMs
	}
}

func normalizeInactiveRetryParameters(value *model.RetryPolicy, updates map[string]any) {
	if value.BackoffType == model.RetryBackoffTypeFixed {
		value.BackoffMultiplier = 1
		updates["backoff_multiplier"] = float64(1)
	}
	if value.JitterType == model.RetryJitterTypeNone {
		value.JitterRatio = 0
		updates["jitter_ratio"] = float64(0)
	}
}

func validateRetryPolicyConfiguration(value model.RetryPolicy) error {
	if !retryPolicyCodePattern.MatchString(value.PolicyCode) || strings.TrimSpace(value.PolicyName) == "" || value.Version < 1 ||
		value.MaxAttempts < 1 || value.MaxAttempts > 10 ||
		value.InitialDelayMs < retryPolicyMinInitialDelayMs || value.InitialDelayMs > retryPolicyMaxInitialDelayMs ||
		value.MaxDelayMs < value.InitialDelayMs || value.MaxDelayMs > retryPolicyMaxDelayMs ||
		value.RetryWindowMs < retryPolicyMinWindowMs || value.RetryWindowMs > retryPolicyMaxWindowMs ||
		value.RetryWindowMs < value.InitialDelayMs {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	if value.BackoffType == model.RetryBackoffTypeFixed {
		if value.BackoffMultiplier != 1 {
			return myerrors.ErrRetryPolicyConfigurationInvalid
		}
	} else if value.BackoffType == model.RetryBackoffTypeExponential {
		if value.BackoffMultiplier < 1.1 || value.BackoffMultiplier > 4 {
			return myerrors.ErrRetryPolicyConfigurationInvalid
		}
	} else {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	if value.JitterType == model.RetryJitterTypeNone {
		if value.JitterRatio != 0 {
			return myerrors.ErrRetryPolicyConfigurationInvalid
		}
	} else if value.JitterType == model.RetryJitterTypeFull {
		if value.JitterRatio != 1 {
			return myerrors.ErrRetryPolicyConfigurationInvalid
		}
	} else {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	if value.RetryWindowMs < retryPolicyRequiredWindowMs(value) {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	categories, statuses, err := retryPolicyLists(value)
	if err != nil || !validRetryPolicyCategories(categories) || !validRetryPolicyStatuses(statuses) {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	return nil
}

func retryPolicyRequiredWindowMs(value model.RetryPolicy) int64 {
	if value.MaxAttempts <= 1 {
		return 0
	}
	delay := value.InitialDelayMs
	var total int64
	for attempt := 1; attempt < value.MaxAttempts; attempt++ {
		total += delay
		if value.BackoffType == model.RetryBackoffTypeExponential {
			next := int64(math.Ceil(float64(delay) * value.BackoffMultiplier))
			if next > value.MaxDelayMs {
				next = value.MaxDelayMs
			}
			delay = next
		}
	}
	return total
}

func setRetryPolicyLists(value *model.RetryPolicy, categories []string, statuses []int) error {
	categories = sortedUniqueStrings(categories)
	statuses = sortedUniqueInts(statuses)
	if !validRetryPolicyCategories(categories) || !validRetryPolicyStatuses(statuses) {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	categoryJSON, err := json.Marshal(categories)
	if err != nil {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	statusJSON, err := json.Marshal(statuses)
	if err != nil {
		return myerrors.ErrRetryPolicyConfigurationInvalid
	}
	value.RetryableErrorCategories = datatypes.JSON(categoryJSON)
	value.RetryableHTTPStatuses = datatypes.JSON(statusJSON)
	return nil
}

func retryPolicyLists(value model.RetryPolicy) ([]string, []int, error) {
	var categories []string
	if err := json.Unmarshal(value.RetryableErrorCategories, &categories); err != nil {
		return nil, nil, err
	}
	var statuses []int
	if err := json.Unmarshal(value.RetryableHTTPStatuses, &statuses); err != nil {
		return nil, nil, err
	}
	return categories, statuses, nil
}

func validRetryPolicyCategories(values []string) bool {
	for _, value := range values {
		if _, ok := retryPolicyAllowedErrorCategories[value]; !ok {
			return false
		}
	}
	return true
}

func validRetryPolicyStatuses(values []int) bool {
	for _, value := range values {
		if _, ok := retryPolicyAllowedHTTPStatuses[value]; !ok {
			return false
		}
	}
	return true
}

func sortedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func sortedUniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Ints(result)
	return result
}

func (s *RetryPolicyService) writeAudit(ctx context.Context, tx *gorm.DB, action string, value model.RetryPolicy, previous *model.RetryPolicy) error {
	changes := map[string]TransactionalAuditChange{
		"policy_code": {NewValue: value.PolicyCode}, "version": {NewValue: value.Version},
		"status": {NewValue: value.Status}, "max_attempts": {NewValue: value.MaxAttempts},
		"backoff_type": {NewValue: value.BackoffType}, "revision": {NewValue: value.Revision},
	}
	if previous != nil {
		changes["status"] = TransactionalAuditChange{OldValue: previous.Status, NewValue: value.Status}
		changes["revision"] = TransactionalAuditChange{OldValue: previous.Revision, NewValue: value.Revision}
	}
	return s.audit.RecordTransactionalAuditContext(ctx, tx, TransactionalAuditRecord{
		Action: action, ResourceType: retryPolicyAuditResourceType,
		ResourceCode: value.PolicyCode + "@" + strconv.Itoa(value.Version),
		ResourceId:   strconv.Itoa(value.Id), Changes: changes,
	})
}

func isRetryPolicyDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "unique constraint") || strings.Contains(lower, "duplicate key")
}
