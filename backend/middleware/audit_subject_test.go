package middleware

import (
	"backend/internal/audit"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInjectAuditSubjectWritesStandardRequestContext(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/audit-subject", nil)

	InjectAuditSubject(ctx, audit.NewAuditSubject(27, "middleware-user"))

	subject, ok := audit.GetAuditSubject(ctx.Request.Context())
	if !ok {
		t.Fatal("middleware did not inject audit subject into request context")
	}
	if subject.UserID != 27 || subject.UserName != "middleware-user" {
		t.Fatalf("unexpected audit subject: %+v", subject)
	}
}

func TestInjectAuditSubjectHandlesMissingRequest(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	InjectAuditSubject(ctx, audit.NewAuditSubject(27, "middleware-user"))
}
