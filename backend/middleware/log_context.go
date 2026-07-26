package middleware

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDHeader = "X-Request-ID"
	TraceIDHeader   = "X-Trace-ID"

	requestIDContextKey    = "sweet_platform_request_id"
	traceIDContextKey      = "sweet_platform_trace_id"
	accessAuditContextKey  = "sweet_platform_access_audit"
	maxCorrelationIDLength = 128
)

var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

// AuditContext adds business audit semantics to the request-wide access log.
// Empty fields leave any middleware-classified value unchanged.
type AuditContext struct {
	MenuID       int
	Action       string
	ResourceType string
	ResourceCode string
	ResourceID   string
}

// EnsureLogContext initializes request and trace identifiers once per request.
// A missing trace ID inherits the request ID so standalone requests remain
// correlatable without requiring a distributed tracing system.
func EnsureLogContext(ctx *gin.Context) {
	if ctx == nil {
		return
	}

	requestID := correlationIDValue(ctx, requestIDContextKey)
	if requestID == "" {
		requestID = normalizeCorrelationID(ctx.GetHeader(RequestIDHeader))
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}

	traceID := correlationIDValue(ctx, traceIDContextKey)
	if traceID == "" {
		traceID = normalizeCorrelationID(ctx.GetHeader(TraceIDHeader))
	}
	if traceID == "" {
		traceID = requestID
	}

	ctx.Set(requestIDContextKey, requestID)
	ctx.Set(traceIDContextKey, traceID)
	ctx.Header(RequestIDHeader, requestID)
	ctx.Header(TraceIDHeader, traceID)
}

func RequestID(ctx *gin.Context) string {
	return correlationIDValue(ctx, requestIDContextKey)
}

func TraceID(ctx *gin.Context) string {
	return correlationIDValue(ctx, traceIDContextKey)
}

// SetAuditContext lets a controller or service-facing adapter describe the
// operation without writing an audit record itself. Repeated calls merge
// non-empty values so shared middleware can add context incrementally.
func SetAuditContext(ctx *gin.Context, next AuditContext) {
	if ctx == nil {
		return
	}
	current := auditContextFromGin(ctx)
	if next.MenuID != 0 {
		current.MenuID = next.MenuID
	}
	if next.Action != "" {
		current.Action = next.Action
	}
	if next.ResourceType != "" {
		current.ResourceType = next.ResourceType
	}
	if next.ResourceCode != "" {
		current.ResourceCode = next.ResourceCode
	}
	if next.ResourceID != "" {
		current.ResourceID = next.ResourceID
	}
	ctx.Set(accessAuditContextKey, current)
}

func auditContextFromGin(ctx *gin.Context) AuditContext {
	if ctx == nil {
		return AuditContext{}
	}
	value, exists := ctx.Get(accessAuditContextKey)
	if !exists {
		return AuditContext{}
	}
	audit, ok := value.(AuditContext)
	if !ok {
		return AuditContext{}
	}
	return audit
}

func correlationIDValue(ctx *gin.Context, key string) string {
	if ctx == nil {
		return ""
	}
	value, exists := ctx.Get(key)
	if !exists {
		return ""
	}
	id, ok := value.(string)
	if !ok {
		return ""
	}
	return normalizeCorrelationID(id)
}

func normalizeCorrelationID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxCorrelationIDLength || !correlationIDPattern.MatchString(value) {
		return ""
	}
	return value
}
