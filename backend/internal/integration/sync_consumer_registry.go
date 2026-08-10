package integration

import (
	myerrors "backend/internal/errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SyncConsumerMetadata 是配置中心可见的非敏感 Consumer 能力摘要。
type SyncConsumerMetadata struct {
	Code             string
	Version          int
	Name             string
	Enabled          bool
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

// SyncConsumerRegistry V1 只提供元数据和引用校验，不包含 Consume。
type SyncConsumerRegistry interface {
	ListMetadata() []SyncConsumerMetadata
	ValidateReference(SyncConsumerReference) (SyncConsumerMetadata, error)
}

type StaticSyncConsumerRegistry struct {
	mu      sync.RWMutex
	entries map[string]SyncConsumerMetadata
}

func NewSyncConsumerRegistry() *StaticSyncConsumerRegistry {
	return &StaticSyncConsumerRegistry{entries: make(map[string]SyncConsumerMetadata)}
}

func NewStaticSyncConsumerRegistry(values ...SyncConsumerMetadata) *StaticSyncConsumerRegistry {
	registry := NewSyncConsumerRegistry()
	for _, value := range values {
		registry.entries[syncConsumerKey(value.Code, value.Version)] = cloneSyncConsumerMetadata(value)
	}
	return registry
}

func (r *StaticSyncConsumerRegistry) ListMetadata() []SyncConsumerMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]SyncConsumerMetadata, 0, len(r.entries))
	for _, value := range r.entries {
		if value.Enabled {
			result = append(result, cloneSyncConsumerMetadata(value))
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

func (r *StaticSyncConsumerRegistry) ValidateReference(ref SyncConsumerReference) (SyncConsumerMetadata, error) {
	r.mu.RLock()
	value, ok := r.entries[syncConsumerKey(ref.Code, ref.Version)]
	r.mu.RUnlock()
	if !ok || !value.Enabled {
		return SyncConsumerMetadata{}, myerrors.ErrSyncConsumerNotRegistered
	}
	if ref.ResponseLimit <= 0 || value.MaxResponseBytes <= 0 || ref.ResponseLimit > value.MaxResponseBytes ||
		!containsFold(value.ContentTypes, ref.ContentType) || !containsExact(value.CheckpointModes, ref.CheckpointMode) {
		return SyncConsumerMetadata{}, myerrors.ErrSyncConsumerIncompatible
	}
	required := ref.RequestTimeout + value.MaxDuration + IntegrationCompletionMargin + IntegrationClaimSafetyMargin
	if value.MaxDuration <= 0 || ref.LeaseDuration < required || ref.LeaseDuration > IntegrationMaximumLeaseDuration {
		return SyncConsumerMetadata{}, myerrors.ErrSyncLeaseBudgetInsufficient
	}
	return cloneSyncConsumerMetadata(value), nil
}

func syncConsumerKey(code string, version int) string {
	return strings.ToLower(strings.TrimSpace(code)) + "@" + strconv.Itoa(version)
}

func cloneSyncConsumerMetadata(value SyncConsumerMetadata) SyncConsumerMetadata {
	value.ContentTypes = append([]string(nil), value.ContentTypes...)
	value.CheckpointModes = append([]string(nil), value.CheckpointModes...)
	return value
}

func containsFold(values []string, expected string) bool {
	expected = strings.TrimSpace(strings.ToLower(strings.Split(expected, ";")[0]))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == expected {
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
