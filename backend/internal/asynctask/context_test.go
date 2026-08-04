package asynctask

import (
	"context"
	"testing"
	"time"
)

func TestContextKeepsDetachedMetadata(t *testing.T) {
	taskContext := New(Metadata{
		RequestID: "request-async-1",
		TraceID:   "trace-async-1",
		ClientIP:  "127.0.0.1",
		UserAgent: "async-test-agent",
	}).WithActor(42, "operator")

	metadata := taskContext.Metadata()
	if metadata.RequestID != "request-async-1" || metadata.TraceID != "trace-async-1" {
		t.Fatalf("异步上下文丢失请求标识: %+v", metadata)
	}
	if metadata.UserID != 42 || metadata.UserName != "operator" {
		t.Fatalf("异步上下文丢失用户信息: %+v", metadata)
	}
	if metadata.ClientIP != "127.0.0.1" || metadata.UserAgent != "async-test-agent" {
		t.Fatalf("异步上下文丢失客户端信息: %+v", metadata)
	}
}

func TestRunDoesNotInheritCanceledRequestLifecycle(t *testing.T) {
	requestContext, cancelRequest := context.WithCancel(context.Background())
	cancelRequest()
	if requestContext.Err() == nil {
		t.Fatal("测试请求 Context 应已取消")
	}

	taskContext := New(Metadata{
		RequestID: "request-detached",
		TraceID:   "trace-detached",
		UserID:    7,
		UserName:  "async-user",
	})
	type result struct {
		metadata Metadata
		err      error
		deadline bool
	}
	done := make(chan result, 1)
	RunWithTimeout(taskContext, time.Second, func(ctx context.Context) {
		_, deadline := ctx.Deadline()
		done <- result{
			metadata: MetadataFrom(ctx),
			err:      ctx.Err(),
			deadline: deadline,
		}
	})

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("异步任务错误继承了请求取消状态: %v", got.err)
		}
		if !got.deadline {
			t.Fatal("异步任务缺少独立超时")
		}
		if got.metadata.RequestID != "request-detached" ||
			got.metadata.TraceID != "trace-detached" ||
			got.metadata.UserID != 7 ||
			got.metadata.UserName != "async-user" {
			t.Fatalf("异步任务元数据不完整: %+v", got.metadata)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("请求结束后异步任务未执行")
	}
}
