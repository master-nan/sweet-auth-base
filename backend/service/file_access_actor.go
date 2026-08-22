package service

// FileAccessActor 是文件资源和Upload Session授权所需的最小调用者快照，
// 不携带HTTP请求状态。
type FileAccessActor struct {
	UserID       int
	IsSuperAdmin bool
}
