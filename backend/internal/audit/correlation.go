package audit

import (
	"context"
	"strings"
)

type correlationContextKey struct{}

const (
	RequestIDValueKey = "sweet_platform_request_id"
	TraceIDValueKey   = "sweet_platform_trace_id"
)

// CorrelationIDs 保存跨 HTTP 与事务审计复用的请求关联标识。
type CorrelationIDs struct {
	RequestID string
	TraceID   string
}

// WithCorrelationIDs 将已校验的请求关联标识写入标准 Context。
func WithCorrelationIDs(ctx context.Context, ids CorrelationIDs) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ids.RequestID = strings.TrimSpace(ids.RequestID)
	ids.TraceID = strings.TrimSpace(ids.TraceID)
	return context.WithValue(ctx, correlationContextKey{}, ids)
}

// GetCorrelationIDs 从标准 Context 读取请求关联标识。
func GetCorrelationIDs(ctx context.Context) CorrelationIDs {
	if ctx == nil {
		return CorrelationIDs{}
	}
	ids, ok := ctx.Value(correlationContextKey{}).(CorrelationIDs)
	if !ok {
		ids.RequestID, _ = ctx.Value(RequestIDValueKey).(string)
		ids.TraceID, _ = ctx.Value(TraceIDValueKey).(string)
	}
	ids.RequestID = strings.TrimSpace(ids.RequestID)
	ids.TraceID = strings.TrimSpace(ids.TraceID)
	return ids
}
