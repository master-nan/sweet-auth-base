package middleware

import (
	"backend/internal/asynctask"
	"backend/internal/audit"
	"backend/model"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestDetachedTaskContextCopiesRequestAndActorFields(t *testing.T) {
	requestContext, cancelRequest := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/async-audit", nil).WithContext(requestContext)
	request.Header.Set(RequestIDHeader, "request-copy-1")
	request.Header.Set(TraceIDHeader, "trace-copy-1")
	request.Header.Set("User-Agent", "detached-context-test")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	EnsureLogContext(ctx)
	correlation := audit.GetCorrelationIDs(ctx.Request.Context())
	if correlation.RequestID != "request-copy-1" || correlation.TraceID != "trace-copy-1" {
		t.Fatalf("标准 Context 未收到请求关联标识: %+v", correlation)
	}
	requestMetadata := audit.GetRequestMetadata(ctx.Request.Context())
	if requestMetadata.Method != http.MethodGet || requestMetadata.Path != "/async-audit" || requestMetadata.ClientIP == "" {
		t.Fatalf("标准 Context 未收到安全请求元数据: %+v", requestMetadata)
	}
	ctx.Set("user", model.SysUser{
		Basic:    model.Basic{Id: 81},
		UserName: "audit-user",
	})

	taskContext := DetachedTaskContext(ctx)
	cancelRequest()
	ctx.Set(requestIDContextKey, "mutated-request")
	ctx.Set(traceIDContextKey, "mutated-trace")
	ctx.Set("user", model.SysUser{Basic: model.Basic{Id: 99}, UserName: "mutated-user"})

	done := make(chan asynctask.Metadata, 1)
	asynctask.RunWithTimeout(taskContext, time.Second, func(taskCtx context.Context) {
		done <- asynctask.MetadataFrom(taskCtx)
	})

	select {
	case metadata := <-done:
		if metadata.RequestID != "request-copy-1" || metadata.TraceID != "trace-copy-1" {
			t.Fatalf("异步快照受请求结束后的修改影响: %+v", metadata)
		}
		if metadata.UserID != 81 || metadata.UserName != "audit-user" {
			t.Fatalf("异步快照丢失用户信息: %+v", metadata)
		}
		if metadata.UserAgent != "detached-context-test" {
			t.Fatalf("异步快照丢失 User-Agent: %+v", metadata)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("请求结束后异步任务未完成")
	}
}
