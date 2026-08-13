// Package audit 提供跨 HTTP、定时任务和命令行场景复用的审计主体上下文。
package audit

import "context"

type auditSubjectContextKey struct{}

// AuditSubjectValueKey is the transitional string key used by HTTP adapters
// whose context.Context implementation exposes request-scoped values by name.
const AuditSubjectValueKey = "sweet_platform_audit_subject"

// AuditSubject 表示一次操作的审计主体，不包含 HTTP 或 Gin 相关对象。
type AuditSubject struct {
	UserID   int
	UserName string
}

// NewAuditSubject 创建可写入审计上下文的主体快照。
func NewAuditSubject(userID int, userName string) AuditSubject {
	return AuditSubject{UserID: userID, UserName: userName}
}

// Valid 表示主体是否包含可用于审计的有效用户标识。
func (s AuditSubject) Valid() bool {
	return s.UserID > 0
}

// WithAuditSubject 将审计主体写入标准 Context。传入 nil 时使用后台 Context。
func WithAuditSubject(ctx context.Context, subject AuditSubject) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, auditSubjectContextKey{}, subject)
}

// GetAuditSubject 从标准 Context 读取有效审计主体。
func GetAuditSubject(ctx context.Context) (AuditSubject, bool) {
	if ctx == nil {
		return AuditSubject{}, false
	}
	subject, ok := ctx.Value(auditSubjectContextKey{}).(AuditSubject)
	if !ok {
		subject, ok = ctx.Value(AuditSubjectValueKey).(AuditSubject)
	}
	if !ok || !subject.Valid() {
		return AuditSubject{}, false
	}
	return subject, true
}
