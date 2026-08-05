package middleware

import (
	"backend/internal/asynctask"
	"backend/internal/audit"
	"backend/model"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDHeader = "X-Request-ID"
	TraceIDHeader   = "X-Trace-ID"

	requestIDContextKey     = "sweet_platform_request_id"
	traceIDContextKey       = "sweet_platform_trace_id"
	accessAuditContextKey   = "sweet_platform_access_audit"
	accessAuditPersistedKey = "sweet_platform_access_audit_persisted"
	maxCorrelationIDLength  = 128
)

var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

// AuditContext 为请求级访问日志补充业务审计语义。
// 空字段不会覆盖中间件已经分类的值。
type AuditContext struct {
	MenuID       int
	Action       string
	ResourceType string
	ResourceCode string
	ResourceID   string
}

// EnsureLogContext 为每个请求初始化一次请求标识和追踪标识。
// 缺少追踪标识时继承请求标识，使独立请求在未接入分布式追踪系统时仍可关联。
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
	if ctx.Request != nil {
		requestContext := audit.WithCorrelationIDs(ctx.Request.Context(), audit.CorrelationIDs{
			RequestID: requestID,
			TraceID:   traceID,
		})
		ctx.Request = ctx.Request.WithContext(requestContext)
	}
	ctx.Header(RequestIDHeader, requestID)
	ctx.Header(TraceIDHeader, traceID)
}

func RequestID(ctx *gin.Context) string {
	return correlationIDValue(ctx, requestIDContextKey)
}

func TraceID(ctx *gin.Context) string {
	return correlationIDValue(ctx, traceIDContextKey)
}

// DetachedTaskContext 复制异步任务需要的请求字段，不保留 Gin Context 或 HTTP 请求对象。
func DetachedTaskContext(ctx *gin.Context) asynctask.Context {
	metadata := asynctask.Metadata{}
	if ctx == nil {
		return asynctask.New(metadata)
	}
	metadata.RequestID = RequestID(ctx)
	metadata.TraceID = TraceID(ctx)
	metadata.ClientIP = ctx.ClientIP()
	if ctx.Request != nil {
		metadata.UserAgent = ctx.Request.UserAgent()
	}
	if value, exists := ctx.Get("user"); exists {
		if user, ok := value.(model.SysUser); ok {
			metadata.UserID = user.Id
			metadata.UserName = user.UserName
		}
	}
	return asynctask.New(metadata)
}

// MarkAccessAuditPersisted 防止请求中间件重复记录 Service 已在写事务中提交的业务审计。
func MarkAccessAuditPersisted(ctx *gin.Context) {
	if ctx != nil {
		ctx.Set(accessAuditPersistedKey, true)
	}
}

func AccessAuditPersisted(ctx *gin.Context) bool {
	if ctx == nil {
		return false
	}
	value, exists := ctx.Get(accessAuditPersistedKey)
	persisted, ok := value.(bool)
	return exists && ok && persisted
}

// SetAuditContext 允许 Controller 或面向 Service 的 Adapter 描述操作，而不直接写入审计记录。
// 重复调用会合并非空值，便于共享中间件逐步补充上下文。
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
