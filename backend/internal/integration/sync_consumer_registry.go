package integration

import (
	"backend/internal/errors"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SyncConsumerStatusEnabled  = "enabled"
	SyncConsumerStatusDisabled = "disabled"

	SyncBusinessReasonProcessingFailed = "business_processing_failed"
)

var syncConsumerSafeCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

// SyncConsumerMetadata 是配置中心可见的非敏感 Consumer 能力摘要。
type SyncConsumerMetadata struct {
	Code             string
	Version          int
	Name             string
	Status           string
	ContentTypes     []string
	MaxResponseBytes int64
	MaxDuration      time.Duration
	CheckpointModes  []string
}

type SyncConsumerReference struct {
	Code           string
	Version        int
	ContentType    string
	ResponseLimit  int64
	CheckpointMode string
	RequestTimeout time.Duration
	LeaseDuration  time.Duration
}

// SyncConsumptionRequest 只携带 Consumer 完成业务处理所需的受控事实。
// Body 仅存在于调用栈，调用方每次读取都会获得副本。
type SyncConsumptionRequest struct {
	executionNo  string
	syncBatchNo  string
	taskCode     string
	taskVersion  int
	sliceNo      int
	windowStart  *time.Time
	windowEnd    *time.Time
	contentType  string
	responseSize int64
	responseHash string
	body         []byte
}

type SyncConsumptionRequestInput struct {
	ExecutionNo  string
	SyncBatchNo  string
	TaskCode     string
	TaskVersion  int
	SliceNo      int
	WindowStart  *time.Time
	WindowEnd    *time.Time
	ContentType  string
	ResponseSize int64
	ResponseHash string
	Body         []byte
}

func NewSyncConsumptionRequest(input SyncConsumptionRequestInput) (SyncConsumptionRequest, error) {
	executionNo := strings.TrimSpace(input.ExecutionNo)
	batchNo := strings.TrimSpace(input.SyncBatchNo)
	taskCode := strings.TrimSpace(input.TaskCode)
	contentType := normalizeConsumerContentType(input.ContentType)
	hash := strings.ToLower(strings.TrimSpace(input.ResponseHash))
	if executionNo == "" || batchNo == "" || !syncConsumerSafeCodePattern.MatchString(taskCode) ||
		input.TaskVersion <= 0 || input.SliceNo <= 0 || contentType == "" || input.ResponseSize < 0 ||
		input.ResponseSize != int64(len(input.Body)) || len(hash) != sha256.Size*2 ||
		(input.WindowStart == nil) != (input.WindowEnd == nil) ||
		(input.WindowStart != nil && !input.WindowEnd.After(*input.WindowStart)) {
		return SyncConsumptionRequest{}, errors.ErrSyncConsumptionRequestInvalid
	}
	digest := sha256.Sum256(input.Body)
	if hex.EncodeToString(digest[:]) != hash {
		return SyncConsumptionRequest{}, errors.ErrSyncConsumptionRequestInvalid
	}
	return SyncConsumptionRequest{
		executionNo: executionNo, syncBatchNo: batchNo, taskCode: taskCode, taskVersion: input.TaskVersion,
		sliceNo: input.SliceNo, windowStart: cloneConsumerTime(input.WindowStart), windowEnd: cloneConsumerTime(input.WindowEnd),
		contentType: contentType, responseSize: input.ResponseSize, responseHash: hash, body: append([]byte(nil), input.Body...),
	}, nil
}

func (r SyncConsumptionRequest) ExecutionNo() string     { return r.executionNo }
func (r SyncConsumptionRequest) SyncBatchNo() string     { return r.syncBatchNo }
func (r SyncConsumptionRequest) TaskCode() string        { return r.taskCode }
func (r SyncConsumptionRequest) TaskVersion() int        { return r.taskVersion }
func (r SyncConsumptionRequest) SliceNo() int            { return r.sliceNo }
func (r SyncConsumptionRequest) WindowStart() *time.Time { return cloneConsumerTime(r.windowStart) }
func (r SyncConsumptionRequest) WindowEnd() *time.Time   { return cloneConsumerTime(r.windowEnd) }
func (r SyncConsumptionRequest) ContentType() string     { return r.contentType }
func (r SyncConsumptionRequest) ResponseSize() int64     { return r.responseSize }
func (r SyncConsumptionRequest) ResponseHash() string    { return r.responseHash }
func (r SyncConsumptionRequest) Body() []byte            { return append([]byte(nil), r.body...) }
func (r SyncConsumptionRequest) String() string          { return "SyncConsumptionRequest{redacted}" }
func (r SyncConsumptionRequest) GoString() string        { return r.String() }

// SyncConsumptionResult 是 Consumer 返回的安全业务摘要。
type SyncConsumptionResult struct {
	success              bool
	reasonCode           string
	businessSuccessCount int
	businessFailedCount  int
	businessReference    string
}

func NewSyncConsumptionResult(success bool, reasonCode string, successCount, failedCount int, businessReference string) (SyncConsumptionResult, error) {
	reasonCode = strings.TrimSpace(reasonCode)
	businessReference = strings.TrimSpace(businessReference)
	if successCount < 0 || failedCount < 0 || len(reasonCode) > 64 || len(businessReference) > 128 ||
		(reasonCode != "" && !syncConsumerSafeCodePattern.MatchString(reasonCode)) ||
		(success && (reasonCode != "" || failedCount != 0)) || (!success && reasonCode == "") {
		return SyncConsumptionResult{}, errors.ErrSyncConsumptionResultInvalid
	}
	return SyncConsumptionResult{success: success, reasonCode: reasonCode, businessSuccessCount: successCount,
		businessFailedCount: failedCount, businessReference: businessReference}, nil
}

func (r SyncConsumptionResult) Success() bool             { return r.success }
func (r SyncConsumptionResult) ReasonCode() string        { return r.reasonCode }
func (r SyncConsumptionResult) BusinessSuccessCount() int { return r.businessSuccessCount }
func (r SyncConsumptionResult) BusinessFailedCount() int  { return r.businessFailedCount }
func (r SyncConsumptionResult) BusinessReference() string { return r.businessReference }

// SyncResultConsumer 由业务模块实现并在服务端初始化时静态注册。
type SyncResultConsumer interface {
	Consume(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error)
}

type SyncResultConsumerFunc func(context.Context, SyncConsumptionRequest) (SyncConsumptionResult, error)

func (f SyncResultConsumerFunc) Consume(ctx context.Context, request SyncConsumptionRequest) (SyncConsumptionResult, error) {
	return f(ctx, request)
}

type SyncConsumerRegistration struct {
	Metadata SyncConsumerMetadata
	Consumer SyncResultConsumer
}

type ResolvedSyncResultConsumer struct {
	metadata SyncConsumerMetadata
	consumer SyncResultConsumer
}

func (r ResolvedSyncResultConsumer) Metadata() SyncConsumerMetadata {
	return cloneSyncConsumerMetadata(r.metadata)
}

// Consume 施加 Consumer 声明的最大处理时长并恢复业务实现 panic。
// Consumer 必须遵守 Context；Integration 不创建脱离租约生命周期的后台 goroutine。
func (r ResolvedSyncResultConsumer) Consume(ctx context.Context, request SyncConsumptionRequest) (result SyncConsumptionResult, err error) {
	if ctx == nil || r.consumer == nil || r.metadata.MaxDuration <= 0 {
		return SyncConsumptionResult{}, errors.ErrSyncConsumerNotRegistered
	}
	consumerCtx, cancel := context.WithTimeout(ctx, r.metadata.MaxDuration)
	defer cancel()
	defer func() {
		if recover() != nil {
			result = SyncConsumptionResult{}
			err = errors.ErrSyncConsumerPanic
		}
	}()
	result, err = r.consumer.Consume(consumerCtx, request)
	if consumerCtx.Err() == context.DeadlineExceeded {
		return SyncConsumptionResult{}, errors.ErrSyncConsumerTimeout
	}
	if err != nil {
		return SyncConsumptionResult{}, errors.ErrSyncBusinessProcessingFailed
	}
	if err := validateSyncConsumptionResult(result); err != nil {
		return SyncConsumptionResult{}, err
	}
	return result, nil
}

// SyncResultConsumerRegistry 是 Consumer 元数据、引用校验和实现解析的唯一服务端入口。
type SyncResultConsumerRegistry interface {
	ListMetadata() []SyncConsumerMetadata
	ValidateReference(SyncConsumerReference) (SyncConsumerMetadata, error)
	Resolve(code string, version int) (ResolvedSyncResultConsumer, error)
}

// SyncConsumerRegistry 保留配置中心依赖的既有端口名称。
type SyncConsumerRegistry = SyncResultConsumerRegistry

type syncConsumerEntry struct {
	metadata SyncConsumerMetadata
	consumer SyncResultConsumer
}

type StaticSyncConsumerRegistry struct {
	mu      sync.RWMutex
	entries map[string]syncConsumerEntry
}

func NewSyncConsumerRegistry() *StaticSyncConsumerRegistry {
	return &StaticSyncConsumerRegistry{entries: make(map[string]syncConsumerEntry)}
}

func NewStaticSyncConsumerRegistry(values ...SyncConsumerRegistration) (*StaticSyncConsumerRegistry, error) {
	registry := NewSyncConsumerRegistry()
	for _, value := range values {
		metadata, err := normalizeSyncConsumerMetadata(value.Metadata)
		if err != nil || value.Consumer == nil {
			return nil, errors.ErrSyncConsumerRegistrationInvalid
		}
		key := syncConsumerKey(metadata.Code, metadata.Version)
		if _, exists := registry.entries[key]; exists {
			return nil, errors.ErrSyncConsumerDuplicate
		}
		registry.entries[key] = syncConsumerEntry{metadata: metadata, consumer: value.Consumer}
	}
	return registry, nil
}

func (r *StaticSyncConsumerRegistry) ListMetadata() []SyncConsumerMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]SyncConsumerMetadata, 0, len(r.entries))
	for _, value := range r.entries {
		if value.metadata.Status == SyncConsumerStatusEnabled {
			result = append(result, cloneSyncConsumerMetadata(value.metadata))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Code == result[j].Code {
			return result[i].Version < result[j].Version
		}
		return result[i].Code < result[j].Code
	})
	return result
}

func (r *StaticSyncConsumerRegistry) Resolve(code string, version int) (ResolvedSyncResultConsumer, error) {
	r.mu.RLock()
	value, ok := r.entries[syncConsumerKey(code, version)]
	r.mu.RUnlock()
	if !ok || value.metadata.Status != SyncConsumerStatusEnabled || value.consumer == nil {
		return ResolvedSyncResultConsumer{}, errors.ErrSyncConsumerNotRegistered
	}
	return ResolvedSyncResultConsumer{metadata: cloneSyncConsumerMetadata(value.metadata), consumer: value.consumer}, nil
}

func (r *StaticSyncConsumerRegistry) ValidateReference(ref SyncConsumerReference) (SyncConsumerMetadata, error) {
	resolved, err := r.Resolve(ref.Code, ref.Version)
	if err != nil {
		return SyncConsumerMetadata{}, err
	}
	value := resolved.Metadata()
	if ref.ResponseLimit <= 0 || value.MaxResponseBytes <= 0 || ref.ResponseLimit > value.MaxResponseBytes ||
		(ref.ContentType != "" && !containsFold(value.ContentTypes, ref.ContentType)) || !containsExact(value.CheckpointModes, ref.CheckpointMode) {
		return SyncConsumerMetadata{}, errors.ErrSyncConsumerIncompatible
	}
	required := ref.RequestTimeout + value.MaxDuration + IntegrationCompletionMargin + IntegrationClaimSafetyMargin
	if value.MaxDuration <= 0 || ref.LeaseDuration < required || ref.LeaseDuration > IntegrationMaximumLeaseDuration {
		return SyncConsumerMetadata{}, errors.ErrSyncLeaseBudgetInsufficient
	}
	return value, nil
}

func normalizeSyncConsumerMetadata(value SyncConsumerMetadata) (SyncConsumerMetadata, error) {
	value.Code = strings.ToLower(strings.TrimSpace(value.Code))
	value.Name = strings.TrimSpace(value.Name)
	value.Status = strings.ToLower(strings.TrimSpace(value.Status))
	if !syncConsumerSafeCodePattern.MatchString(value.Code) || value.Version <= 0 || value.Name == "" || len(value.Name) > 128 ||
		(value.Status != SyncConsumerStatusEnabled && value.Status != SyncConsumerStatusDisabled) ||
		value.MaxResponseBytes <= 0 || value.MaxResponseBytes > IntegrationMaxResponseBytes || value.MaxDuration <= 0 ||
		value.MaxDuration > IntegrationMaximumLeaseDuration || len(value.ContentTypes) == 0 || len(value.CheckpointModes) == 0 {
		return SyncConsumerMetadata{}, errors.ErrSyncConsumerRegistrationInvalid
	}
	contentTypes := make([]string, 0, len(value.ContentTypes))
	seenContentTypes := make(map[string]struct{}, len(value.ContentTypes))
	for _, item := range value.ContentTypes {
		item = normalizeConsumerContentType(item)
		if item == "" {
			return SyncConsumerMetadata{}, errors.ErrSyncConsumerRegistrationInvalid
		}
		if _, exists := seenContentTypes[item]; !exists {
			seenContentTypes[item] = struct{}{}
			contentTypes = append(contentTypes, item)
		}
	}
	checkpointModes := make([]string, 0, len(value.CheckpointModes))
	seenModes := make(map[string]struct{}, len(value.CheckpointModes))
	for _, item := range value.CheckpointModes {
		item = strings.TrimSpace(item)
		if item != "none" && item != "timestamp" {
			return SyncConsumerMetadata{}, errors.ErrSyncConsumerRegistrationInvalid
		}
		if _, exists := seenModes[item]; !exists {
			seenModes[item] = struct{}{}
			checkpointModes = append(checkpointModes, item)
		}
	}
	value.ContentTypes = contentTypes
	value.CheckpointModes = checkpointModes
	return value, nil
}

func validateSyncConsumptionResult(value SyncConsumptionResult) error {
	_, err := NewSyncConsumptionResult(value.success, value.reasonCode, value.businessSuccessCount, value.businessFailedCount, value.businessReference)
	return err
}

func syncConsumerKey(code string, version int) string {
	return strings.ToLower(strings.TrimSpace(code)) + "@" + strconv.Itoa(version)
}

func cloneSyncConsumerMetadata(value SyncConsumerMetadata) SyncConsumerMetadata {
	value.ContentTypes = append([]string(nil), value.ContentTypes...)
	value.CheckpointModes = append([]string(nil), value.CheckpointModes...)
	return value
}

func cloneConsumerTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func normalizeConsumerContentType(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.Split(value, ";")[0]))
}

func containsFold(values []string, expected string) bool {
	expected = normalizeConsumerContentType(expected)
	for _, value := range values {
		if normalizeConsumerContentType(value) == expected {
			return true
		}
	}
	return false
}

func containsExact(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}
