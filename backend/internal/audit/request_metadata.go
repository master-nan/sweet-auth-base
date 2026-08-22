package audit

import (
	"context"
	"strings"
	"sync/atomic"
)

type requestMetadataContextKey struct{}

// RequestMetadataValueKey 供HTTP Adapter向request.Context传递安全请求元数据；
// 其中不包含HTTP Request或框架对象。
const RequestMetadataValueKey = "sweet_platform_request_metadata"

type accessAuditState struct {
	persisted atomic.Bool
}

type accessAuditStateContextKey struct{}

// RequestMetadata 只包含事务审计需要的安全请求事实。
type RequestMetadata struct {
	Method    string
	Path      string
	ClientIP  string
	UserAgent string
}

func WithRequestMetadata(ctx context.Context, metadata RequestMetadata) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestMetadataContextKey{}, normalizeRequestMetadata(metadata))
}

func WithAccessAuditState(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(accessAuditStateContextKey{}).(*accessAuditState); ok {
		return ctx
	}
	return context.WithValue(ctx, accessAuditStateContextKey{}, &accessAuditState{})
}

func MarkAccessAuditPersisted(ctx context.Context) {
	if state, ok := accessAuditStateFromContext(ctx); ok {
		state.persisted.Store(true)
	}
}

func AccessAuditPersisted(ctx context.Context) bool {
	state, ok := accessAuditStateFromContext(ctx)
	return ok && state.persisted.Load()
}

func accessAuditStateFromContext(ctx context.Context) (*accessAuditState, bool) {
	if ctx == nil {
		return nil, false
	}
	state, ok := ctx.Value(accessAuditStateContextKey{}).(*accessAuditState)
	return state, ok && state != nil
}

func GetRequestMetadata(ctx context.Context) RequestMetadata {
	if ctx == nil {
		return RequestMetadata{}
	}
	metadata, ok := ctx.Value(requestMetadataContextKey{}).(RequestMetadata)
	if !ok {
		metadata, _ = ctx.Value(RequestMetadataValueKey).(RequestMetadata)
	}
	return normalizeRequestMetadata(metadata)
}

func normalizeRequestMetadata(metadata RequestMetadata) RequestMetadata {
	metadata.Method = strings.ToUpper(strings.TrimSpace(metadata.Method))
	metadata.Path = strings.TrimSpace(metadata.Path)
	metadata.ClientIP = strings.TrimSpace(metadata.ClientIP)
	metadata.UserAgent = strings.TrimSpace(metadata.UserAgent)
	if len(metadata.UserAgent) > 256 {
		metadata.UserAgent = metadata.UserAgent[:256]
	}
	return metadata
}
