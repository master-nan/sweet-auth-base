package integration

import (
	myerrors "backend/internal/errors"
	"time"
)

const (
	IntegrationMinRequestTimeout        = time.Second
	IntegrationMaxRequestTimeout        = 120 * time.Second
	IntegrationDefaultRequestTimeout    = 30 * time.Second
	IntegrationMaxConnectTimeout        = 30 * time.Second
	IntegrationDefaultConnectTimeout    = 5 * time.Second
	IntegrationMaxTLSHandshakeTimeout   = 30 * time.Second
	IntegrationDefaultTLSHandshake      = 10 * time.Second
	IntegrationMaxResponseHeaderTimeout = 120 * time.Second
	IntegrationDefaultResponseHeader    = 15 * time.Second
	IntegrationMinResponseBytes         = int64(1024)
	IntegrationMaxResponseBytes         = int64(64 * 1024 * 1024)
	IntegrationDefaultResponseBytes     = int64(10 * 1024 * 1024)
	IntegrationCompletionMargin         = 30 * time.Second
	IntegrationClaimSafetyMargin        = 15 * time.Second
	IntegrationMinimumLeaseDuration     = IntegrationMaxRequestTimeout + IntegrationCompletionMargin + IntegrationClaimSafetyMargin
	IntegrationDefaultLeaseDuration     = 3 * time.Minute
	IntegrationMaximumLeaseDuration     = 10 * time.Minute
)

// IntegrationRuntimeLimits 是配置中心、Transport、Engine 与 Runner 共用的唯一运行参数契约。
type IntegrationRuntimeLimits struct {
	MinRequestTimeout        time.Duration
	MaxRequestTimeout        time.Duration
	DefaultRequestTimeout    time.Duration
	MaxConnectTimeout        time.Duration
	DefaultConnectTimeout    time.Duration
	MaxTLSHandshakeTimeout   time.Duration
	DefaultTLSHandshake      time.Duration
	MaxResponseHeaderTimeout time.Duration
	DefaultResponseHeader    time.Duration
	MinResponseBytes         int64
	MaxResponseBytes         int64
	DefaultResponseBytes     int64
	CompletionMargin         time.Duration
	ClaimSafetyMargin        time.Duration
	MinimumLeaseDuration     time.Duration
	DefaultLeaseDuration     time.Duration
	MaximumLeaseDuration     time.Duration
}

// RuntimeLimits 返回只读值对象，调用方不能修改平台全局边界。
func RuntimeLimits() IntegrationRuntimeLimits {
	return IntegrationRuntimeLimits{
		MinRequestTimeout: IntegrationMinRequestTimeout, MaxRequestTimeout: IntegrationMaxRequestTimeout,
		DefaultRequestTimeout: IntegrationDefaultRequestTimeout, MaxConnectTimeout: IntegrationMaxConnectTimeout,
		DefaultConnectTimeout: IntegrationDefaultConnectTimeout, MaxTLSHandshakeTimeout: IntegrationMaxTLSHandshakeTimeout,
		DefaultTLSHandshake: IntegrationDefaultTLSHandshake, MaxResponseHeaderTimeout: IntegrationMaxResponseHeaderTimeout,
		DefaultResponseHeader: IntegrationDefaultResponseHeader, MinResponseBytes: IntegrationMinResponseBytes,
		MaxResponseBytes: IntegrationMaxResponseBytes, DefaultResponseBytes: IntegrationDefaultResponseBytes,
		CompletionMargin: IntegrationCompletionMargin, ClaimSafetyMargin: IntegrationClaimSafetyMargin,
		MinimumLeaseDuration: IntegrationMinimumLeaseDuration, DefaultLeaseDuration: IntegrationDefaultLeaseDuration,
		MaximumLeaseDuration: IntegrationMaximumLeaseDuration,
	}
}

// ValidateInterfaceRuntimeContract 校验接口配置是否能被当前 Transport 完整执行。
func ValidateInterfaceRuntimeContract(timeoutSeconds int, responseLimit int64) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout < IntegrationMinRequestTimeout || timeout > IntegrationMaxRequestTimeout {
		return myerrors.ErrIntegrationTimeoutOutOfRange
	}
	if responseLimit < IntegrationMinResponseBytes || responseLimit > IntegrationMaxResponseBytes {
		return myerrors.ErrIntegrationResponseLimitOutOfRange
	}
	return nil
}

// ValidateLeaseDuration 保证固定租约覆盖最大合法请求及两段完成安全余量。
func ValidateLeaseDuration(value time.Duration) error {
	if value < IntegrationMinimumLeaseDuration || value > IntegrationMaximumLeaseDuration {
		if value > 0 && value < IntegrationMinimumLeaseDuration {
			return myerrors.ErrIntegrationLeaseMarginInsufficient
		}
		return myerrors.ErrIntegrationLeaseDurationInvalid
	}
	return nil
}
