package asynctask

import (
	"context"
	"time"
)

const DefaultTimeout = 30 * time.Second

type metadataContextKey struct{}

// Metadata 是异步任务需要的请求元数据快照，不保留 HTTP 请求或响应对象。
type Metadata struct {
	RequestID string
	TraceID   string
	UserID    int
	UserName  string
	ClientIP  string
	UserAgent string
}

// Context 是可安全跨越请求生命周期的轻量任务上下文。
type Context struct {
	metadata Metadata
}

func New(metadata Metadata) Context {
	return Context{metadata: metadata}
}

func (c Context) WithActor(userID int, userName string) Context {
	c.metadata.UserID = userID
	c.metadata.UserName = userName
	return c
}

func (c Context) Metadata() Metadata {
	return c.metadata
}

// Run 使用独立标准 Context 执行任务，任务不会继承请求取消信号。
func Run(taskContext Context, task func(context.Context)) {
	RunWithTimeout(taskContext, DefaultTimeout, task)
}

func RunWithTimeout(taskContext Context, timeout time.Duration, task func(context.Context)) {
	if task == nil {
		return
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	go func(metadata Metadata) {
		base := context.WithValue(context.Background(), metadataContextKey{}, metadata)
		ctx, cancel := context.WithTimeout(base, timeout)
		defer cancel()
		task(ctx)
	}(taskContext.metadata)
}

func MetadataFrom(ctx context.Context) Metadata {
	if ctx == nil {
		return Metadata{}
	}
	metadata, _ := ctx.Value(metadataContextKey{}).(Metadata)
	return metadata
}
