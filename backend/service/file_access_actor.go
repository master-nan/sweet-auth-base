package service

// FileAccessActor is the minimum caller snapshot needed for file resource and
// upload-session authorization. It contains no HTTP request state.
type FileAccessActor struct {
	UserID       int
	IsSuperAdmin bool
}
