package audit

import (
	"context"
	"strings"
)

type requestMetadataContextKey struct{}

// RequestMetadataValueKey supports HTTP adapter contexts during the migration
// to request.Context. The value contains no HTTP request or framework object.
const RequestMetadataValueKey = "sweet_platform_request_metadata"

// RequestMetadata contains the safe request facts needed by transactional audit.
type RequestMetadata struct {
	Method   string
	Path     string
	ClientIP string
}

func WithRequestMetadata(ctx context.Context, metadata RequestMetadata) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestMetadataContextKey{}, normalizeRequestMetadata(metadata))
}

func GetRequestMetadata(ctx context.Context) RequestMetadata {
	if ctx == nil {
		return RequestMetadata{}
	}
	metadata, ok := ctx.Value(requestMetadataContextKey{}).(RequestMetadata)
	if !ok {
		metadata, _ = ctx.Value(RequestMetadataValueKey).(RequestMetadata)
	}
	return normalizeRequestMetadata(metadata)
}

func normalizeRequestMetadata(metadata RequestMetadata) RequestMetadata {
	metadata.Method = strings.ToUpper(strings.TrimSpace(metadata.Method))
	metadata.Path = strings.TrimSpace(metadata.Path)
	metadata.ClientIP = strings.TrimSpace(metadata.ClientIP)
	return metadata
}
