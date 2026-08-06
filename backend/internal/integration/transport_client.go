package integration

import (
	"backend/internal/audit"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// TransportClient 是 Integration Runtime 调用外部 HTTP 接口的受控端口。
type TransportClient interface {
	Execute(context.Context, TransportRequest) (TransportResult, error)
}

// TransportClientOptions 仅用于服务端构造客户端；RoundTripper 用于本地测试替身。
// 生产构造不传入 RoundTripper，由客户端使用带 DNS 重校验的受控拨号器。
type TransportClientOptions struct {
	RoundTripper http.RoundTripper
	TLSConfig    *tls.Config
}

// HTTPTransportClient 不依赖 Gin、数据库或任何配置 Model。
type HTTPTransportClient struct {
	policy  EndpointPolicy
	options TransportClientOptions
}

func NewHTTPTransportClient(policy EndpointPolicy, options TransportClientOptions) (*HTTPTransportClient, error) {
	if policy.resolver == nil {
		return nil, newTransportError(TransportErrorInvalidConfig)
	}
	return &HTTPTransportClient{policy: policy, options: options}, nil
}

func (c *HTTPTransportClient) Execute(ctx context.Context, request TransportRequest) (TransportResult, error) {
	startedAt := time.Now()
	result := TransportResult{Determinacy: TransportDeterminacyConfirmed}
	if ctx == nil {
		ctx = context.Background()
	}
	target, err := request.targetURL()
	if err != nil {
		return c.finish(ctx, request, target, result, startedAt, transportErrorCategory(err), err)
	}
	if _, err = c.policy.validateTarget(ctx, target); err != nil {
		return c.finish(ctx, request, target, result, startedAt, transportErrorCategory(err), err)
	}

	requestContext, cancel := context.WithTimeout(ctx, request.timeouts.Request)
	defer cancel()
	httpRequest, err := request.newHTTPRequest(requestContext, target)
	if err != nil {
		return c.finish(ctx, request, target, result, startedAt, transportErrorCategory(err), err)
	}
	client, closeIdle := c.httpClient(request)
	defer closeIdle()
	response, err := client.Do(httpRequest)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		category := classifyTransportFailure(requestContext, err)
		result.Determinacy = TransportDeterminacyUnknown
		return c.finish(ctx, request, target, result, startedAt, category, newTransportError(category))
	}
	if response == nil || response.Body == nil {
		result.Determinacy = TransportDeterminacyUnknown
		return c.finish(ctx, request, target, result, startedAt, TransportErrorInternal, newTransportError(TransportErrorInternal))
	}

	result.StatusCode = response.StatusCode
	result.ContentType = normalizedContentType(response.Header.Get("Content-Type"))
	result.responseHeaders = safeResponseHeaders(response.Header)
	if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < http.StatusBadRequest {
		_ = response.Body.Close()
		result.CompleteResponse = false
		result.Determinacy = TransportDeterminacyUnknown
		return c.finish(ctx, request, target, result, startedAt, TransportErrorRedirectRejected, newTransportError(TransportErrorRedirectRejected))
	}
	if response.ContentLength > request.maxResponseBytes {
		result.ResponseSize = response.ContentLength
		_ = response.Body.Close()
		result.CompleteResponse = false
		result.Determinacy = TransportDeterminacyUnknown
		return c.finish(ctx, request, target, result, startedAt, TransportErrorResponseTooLarge, newTransportError(TransportErrorResponseTooLarge))
	}
	if !request.allowsContentType(result.ContentType, response.ContentLength) {
		_ = response.Body.Close()
		result.CompleteResponse = false
		return c.finish(ctx, request, target, result, startedAt, TransportErrorUnsupportedContentType, newTransportError(TransportErrorUnsupportedContentType))
	}
	body, complete, readErr := readResponseBody(response.Body, request.maxResponseBytes)
	if readErr != nil {
		category := classifyTransportFailure(requestContext, readErr)
		result.Determinacy = TransportDeterminacyUnknown
		return c.finish(ctx, request, target, result, startedAt, category, newTransportError(category))
	}
	if !complete {
		result.ResponseSize = request.maxResponseBytes + 1
		result.CompleteResponse = false
		result.Determinacy = TransportDeterminacyUnknown
		return c.finish(ctx, request, target, result, startedAt, TransportErrorResponseTooLarge, newTransportError(TransportErrorResponseTooLarge))
	}

	result.ResponseSize = int64(len(body))
	result.ResponseHash = responseHash(body)
	result.CompleteResponse = true
	result = result.withResponse(body, result.responseHeaders)
	if response.StatusCode >= http.StatusBadRequest {
		return c.finish(ctx, request, target, result, startedAt, TransportErrorRemoteHTTP, nil)
	}
	return c.finish(ctx, request, target, result, startedAt, "", nil)
}

func (c *HTTPTransportClient) httpClient(request TransportRequest) (*http.Client, func()) {
	if c.options.RoundTripper != nil {
		return &http.Client{Transport: c.options.RoundTripper, CheckRedirect: rejectRedirect}, func() {}
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           c.safeDialContext(request.timeouts.Connect),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   request.timeouts.TLSHandshake,
		ResponseHeaderTimeout: request.timeouts.ResponseHeader,
		ExpectContinueTimeout: time.Second,
		DisableCompression:    true,
		TLSClientConfig:       c.tlsConfig(),
	}
	return &http.Client{Transport: transport, CheckRedirect: rejectRedirect}, transport.CloseIdleConnections
}

func (c *HTTPTransportClient) tlsConfig() *tls.Config {
	if c.options.TLSConfig == nil {
		return &tls.Config{MinVersion: tls.VersionTLS12}
	}
	config := c.options.TLSConfig.Clone()
	if config.MinVersion < tls.VersionTLS12 {
		config.MinVersion = tls.VersionTLS12
	}
	return config
}

func (c *HTTPTransportClient) safeDialContext(connectTimeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || host == "" || port == "" {
			return nil, newTransportError(TransportErrorInvalidURL)
		}
		// 每次实际拨号重新解析并校验，且只拨向已校验 IP，避免 DNS Rebinding 改写连接目标。
		target := &url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}
		addresses, err := c.policy.validateTarget(ctx, target)
		if err != nil {
			return nil, err
		}
		dialer := &net.Dialer{Timeout: connectTimeout}
		var lastErr error
		for _, value := range addresses {
			connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(value.String(), port))
			if dialErr == nil {
				return connection, nil
			}
			lastErr = dialErr
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, newTransportError(TransportErrorNetwork)
	}
}

func rejectRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (c *HTTPTransportClient) finish(
	ctx context.Context,
	request TransportRequest,
	target *url.URL,
	result TransportResult,
	startedAt time.Time,
	category TransportErrorCategory,
	err error,
) (TransportResult, error) {
	result.Duration = time.Since(startedAt)
	result.ErrorCategory = category
	c.log(ctx, request, target, result)
	return result, err
}

func (c *HTTPTransportClient) log(ctx context.Context, request TransportRequest, target *url.URL, result TransportResult) {
	correlation := audit.GetCorrelationIDs(ctx)
	hostDigest := ""
	if target != nil {
		digest := sha256.Sum256([]byte(strings.ToLower(target.Hostname())))
		hostDigest = hex.EncodeToString(digest[:8])
	}
	zap.L().Info("integration transport completed",
		zap.String("request_id", correlation.RequestID),
		zap.String("trace_id", correlation.TraceID),
		zap.String("method", request.method),
		zap.String("target_host_hash", hostDigest),
		zap.Int("http_status", result.StatusCode),
		zap.Duration("duration", result.Duration),
		zap.String("error_category", string(result.ErrorCategory)),
	)
}

func (r TransportRequest) allowsContentType(contentType string, contentLength int64) bool {
	if contentType == "" && contentLength == 0 {
		return true
	}
	_, ok := r.allowedResponseContentTypes[contentType]
	return ok
}

func normalizedContentType(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return strings.ToLower(mediaType)
}

func classifyTransportFailure(ctx context.Context, err error) TransportErrorCategory {
	if errors.Is(ctx.Err(), context.Canceled) {
		return TransportErrorCancelled
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return TransportErrorTimeout
	}
	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		return transportErr.Category()
	}
	var tlsErr tls.RecordHeaderError
	if errors.As(err, &tlsErr) {
		return TransportErrorTLS
	}
	var certificateVerificationErr *tls.CertificateVerificationError
	if errors.As(err, &certificateVerificationErr) {
		return TransportErrorTLS
	}
	var unknownAuthorityErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthorityErr) {
		return TransportErrorTLS
	}
	var invalidCertificateErr x509.CertificateInvalidError
	if errors.As(err, &invalidCertificateErr) {
		return TransportErrorTLS
	}
	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return TransportErrorTLS
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return TransportErrorTimeout
	}
	return TransportErrorNetwork
}
