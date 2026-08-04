package middleware

import (
	"backend/internal/audit"

	"github.com/gin-gonic/gin"
)

// InjectAuditSubject 将认证后的审计主体写入请求标准 Context，不保存 Gin 对象。
func InjectAuditSubject(ctx *gin.Context, subject audit.AuditSubject) {
	if ctx == nil || ctx.Request == nil {
		return
	}
	ctx.Request = ctx.Request.WithContext(audit.WithAuditSubject(ctx.Request.Context(), subject))
}
