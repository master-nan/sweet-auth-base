// Package integration 提供集成运行时可复用的内部技术能力。
package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxRequestBodyBytes = int64(1024 * 1024)
	maxRequestBodyBytes        = int64(4 * 1024 * 1024)
	maxHeaderCount             = 32
	maxHeaderNameLength        = 64
	maxHeaderValueLength       = 4096
	maxHeaderTotalLength       = 16 * 1024
	maxQueryParameterCount     = 64
	maxQueryNameLength         = 128
	maxQueryValueLength        = 2048
	maxPathParameterLength     = 256
)

var pathParameterPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

// TransportErrorCategory 是 Transport 的稳定技术错误分类，不向 Controller 传递底层网络文本。
type TransportErrorCategory string

const (
	TransportErrorInvalidConfig          TransportErrorCategory = "invalid_config"
	TransportErrorInvalidURL             TransportErrorCategory = "invalid_url"
	TransportErrorSSRFRejected           TransportErrorCategory = "ssrf_rejected"
	TransportErrorTimeout                TransportErrorCategory = "timeout"
	TransportErrorNetwork                TransportErrorCategory = "network_error"
	TransportErrorTLS                    TransportErrorCategory = "tls_error"
	TransportErrorResponseTooLarge       TransportErrorCategory = "response_too_large"
	TransportErrorUnsupportedContentType TransportErrorCategory = "unsupported_content_type"
	TransportErrorRemoteHTTP             TransportErrorCategory = "remote_http_error"
	TransportErrorRedirectRejected       TransportErrorCategory = "redirect_rejected"
	TransportErrorCancelled              TransportErrorCategory = "cancelled"
	TransportErrorInternal               TransportErrorCategory = "internal_error"
)

// TransportDeterminacy 表示本地无法确认远端业务是否已经产生效果时的保守结论。
type TransportDeterminacy string

const (
	TransportDeterminacyConfirmed TransportDeterminacy = "confirmed"
	TransportDeterminacyUnknown   TransportDeterminacy = "unknown"
)

// TransportError 仅暴露稳定分类；底层错误只保留在内部日志中。
type TransportError struct {
	category TransportErrorCategory
}

func (e *TransportError) Error() string {
	return "integration transport: " + string(e.category)
}

func (e *TransportError) Category() TransportErrorCategory {
	if e == nil {
		return ""
	}
	return e.category
}

func newTransportError(category TransportErrorCategory) error {
	return &TransportError{category: category}
}

func transportErrorCategory(err error) TransportErrorCategory {
	var transportErr *TransportError
	if err != nil && errors.As(err, &transportErr) {
		return transportErr.Category()
	}
	return TransportErrorInternal
}

// TransportTimeouts 为一次调用定义受控超时；零值由构造器替换为平台默认值。
type TransportTimeouts struct {
	Connect        time.Duration
	TLSHandshake   time.Duration
	Request        time.Duration
	ResponseHeader time.Duration
}

// TransportRequestInput 只能由上层使用已确认的系统和接口配置构造。
// BaseURL 与 RelativePath 分别来自服务端配置；该对象不接收客户端完整 URL。
type TransportRequestInput struct {
	Method                      string
	BaseURL                     string
	RelativePath                string
	PathParameters              map[string]string
	QueryParameters             map[string][]string
	Headers                     map[string]string
	JSONBody                    []byte
	Timeouts                    TransportTimeouts
	MaxResponseBytes            int64
	AllowedResponseContentTypes []string
}

// TransportRequest 是不可变请求值对象。其映射和 Body 均在构造时复制。
type TransportRequest struct {
	method                      string
	baseURL                     string
	relativePath                string
	pathParameters              map[string]string
	queryParameters             map[string][]string
	headers                     map[string]string
	authenticationHeaders       map[string]string
	jsonBody                    []byte
	timeouts                    TransportTimeouts
	maxResponseBytes            int64
	allowedResponseContentTypes map[string]struct{}
}

// TransportAuthentication 是 Credential Provider 已准备好的认证注入结果。
// 普通 Header 不能携带 Authorization、Cookie 或代理认证信息。
type TransportAuthentication struct {
	headers map[string]string
}

// String 防止认证 Header 被格式化日志或错误消息意外展开。
func (TransportAuthentication) String() string {
	return "TransportAuthentication{redacted}"
}

// GoString 防止 %#v 等调试格式泄露认证材料。
func (TransportAuthentication) GoString() string {
	return "TransportAuthentication{redacted}"
}

// MarshalJSON 阻止认证注入结果被误用为 API 响应或持久化载荷。
func (TransportAuthentication) MarshalJSON() ([]byte, error) {
	return nil, newTransportError(TransportErrorInvalidConfig)
}

// NewTransportAuthentication 创建受控认证注入结果。当前只接受平台一期所需的认证 Header。
func NewTransportAuthentication(headers map[string]string) (TransportAuthentication, error) {
	cloned := make(map[string]string, len(headers))
	for name, value := range headers {
		normalizedName := strings.ToLower(strings.TrimSpace(name))
		if normalizedName != "authorization" && normalizedName != "x-api-key" {
			return TransportAuthentication{}, newTransportError(TransportErrorInvalidConfig)
		}
		if err := validateHeader(normalizedName, value); err != nil {
			return TransportAuthentication{}, err
		}
		cloned[normalizedName] = strings.TrimSpace(value)
	}
	return TransportAuthentication{headers: cloned}, nil
}

// NewTransportRequest 创建受控请求，不允许调用方直接给出最终 URL。
func NewTransportRequest(input TransportRequestInput) (TransportRequest, error) {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if !isAllowedMethod(method) {
		return TransportRequest{}, newTransportError(TransportErrorInvalidConfig)
	}
	if err := validateBaseURL(input.BaseURL); err != nil {
		return TransportRequest{}, err
	}
	if err := validateRelativePath(input.RelativePath); err != nil {
		return TransportRequest{}, err
	}
	pathParameters, err := cloneAndValidatePathParameters(input.PathParameters)
	if err != nil {
		return TransportRequest{}, err
	}
	if _, _, err = renderRelativePath(input.RelativePath, pathParameters); err != nil {
		return TransportRequest{}, err
	}
	queryParameters, err := cloneAndValidateQueryParameters(input.QueryParameters)
	if err != nil {
		return TransportRequest{}, err
	}
	headers, err := cloneAndValidateHeaders(input.Headers, false)
	if err != nil {
		return TransportRequest{}, err
	}
	body := append([]byte(nil), input.JSONBody...)
	if int64(len(body)) > defaultMaxRequestBodyBytes || int64(len(body)) > maxRequestBodyBytes {
		return TransportRequest{}, newTransportError(TransportErrorInvalidConfig)
	}
	if len(body) > 0 {
		if method == http.MethodGet || !json.Valid(body) {
			return TransportRequest{}, newTransportError(TransportErrorInvalidConfig)
		}
		if _, exists := headers["content-type"]; !exists {
			headers["content-type"] = "application/json"
		}
		if !isJSONContentType(headers["content-type"]) {
			return TransportRequest{}, newTransportError(TransportErrorInvalidConfig)
		}
	}
	timeouts, err := normalizeTimeouts(input.Timeouts)
	if err != nil {
		return TransportRequest{}, err
	}
	maxResponse, err := normalizeMaxResponseBytes(input.MaxResponseBytes)
	if err != nil {
		return TransportRequest{}, err
	}
	allowedTypes, err := normalizeContentTypes(input.AllowedResponseContentTypes)
	if err != nil {
		return TransportRequest{}, err
	}
	return TransportRequest{
		method: method, baseURL: strings.TrimSpace(input.BaseURL), relativePath: input.RelativePath,
		pathParameters: pathParameters, queryParameters: queryParameters, headers: headers,
		jsonBody: body, timeouts: timeouts, maxResponseBytes: maxResponse,
		allowedResponseContentTypes: allowedTypes,
	}, nil
}

// WithAuthentication 返回带有认证注入副本的新请求，原请求不会改变。
func (r TransportRequest) WithAuthentication(authentication TransportAuthentication) (TransportRequest, error) {
	clone := r.clone()
	for name, value := range authentication.headers {
		if _, exists := clone.headers[name]; exists {
			return TransportRequest{}, newTransportError(TransportErrorInvalidConfig)
		}
		clone.authenticationHeaders[name] = value
	}
	mergedHeaders := cloneStringMap(clone.headers)
	for name, value := range clone.authenticationHeaders {
		mergedHeaders[name] = value
	}
	if err := validateHeaderBudget(mergedHeaders); err != nil {
		return TransportRequest{}, err
	}
	if _, err := clone.targetURL(); err != nil {
		return TransportRequest{}, err
	}
	return clone, nil
}

func (r TransportRequest) clone() TransportRequest {
	return TransportRequest{
		method: r.method, baseURL: r.baseURL, relativePath: r.relativePath,
		pathParameters: cloneStringMap(r.pathParameters), queryParameters: cloneStringSliceMap(r.queryParameters),
		headers: cloneStringMap(r.headers), authenticationHeaders: cloneStringMap(r.authenticationHeaders),
		jsonBody: append([]byte(nil), r.jsonBody...), timeouts: r.timeouts, maxResponseBytes: r.maxResponseBytes,
		allowedResponseContentTypes: cloneContentTypes(r.allowedResponseContentTypes),
	}
}

func (r TransportRequest) targetURL() (*url.URL, error) {
	base, err := url.Parse(r.baseURL)
	if err != nil || base == nil {
		return nil, newTransportError(TransportErrorInvalidURL)
	}
	rawRelativePath, relativePath, err := renderRelativePath(r.relativePath, r.pathParameters)
	if err != nil {
		return nil, err
	}
	target := *base
	target.Path = strings.TrimRight(base.Path, "/") + relativePath
	target.RawPath = strings.TrimRight(base.EscapedPath(), "/") + rawRelativePath
	if target.Path == "" {
		target.Path = "/"
	}
	values := make(url.Values, len(r.queryParameters))
	for name, rawValues := range r.queryParameters {
		values[name] = append([]string(nil), rawValues...)
	}
	target.RawQuery = values.Encode()
	if err := validateTargetURL(&target); err != nil {
		return nil, err
	}
	return &target, nil
}

func (r TransportRequest) newHTTPRequest(ctx context.Context, target *url.URL) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, r.method, target.String(), bytes.NewReader(r.jsonBody))
	if err != nil {
		return nil, newTransportError(TransportErrorInvalidURL)
	}
	for name, value := range r.headers {
		req.Header.Set(name, value)
	}
	for name, value := range r.authenticationHeaders {
		req.Header.Set(name, value)
	}
	return req, nil
}

// TransportResult 是结构化调用结果。响应正文仅可通过 Body 获取副本，避免调用方修改内部缓冲区。
type TransportResult struct {
	StatusCode       int
	ContentType      string
	ResponseSize     int64
	ResponseHash     string
	Duration         time.Duration
	CompleteResponse bool
	Determinacy      TransportDeterminacy
	ErrorCategory    TransportErrorCategory

	responseHeaders map[string]string
	body            []byte
}

// ResponseHeaders 返回安全 Header 摘要副本，不包含认证、Cookie 等敏感 Header。
func (r TransportResult) ResponseHeaders() map[string]string {
	return cloneStringMap(r.responseHeaders)
}

// Body 返回受控响应正文副本。日志和审计不得直接记录该内容。
func (r TransportResult) Body() []byte {
	return append([]byte(nil), r.body...)
}

func (r TransportResult) withResponse(body []byte, headers map[string]string) TransportResult {
	r.body = append([]byte(nil), body...)
	r.responseHeaders = cloneStringMap(headers)
	return r
}

func responseHash(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func normalizeTimeouts(input TransportTimeouts) (TransportTimeouts, error) {
	value := input
	if value.Connect == 0 {
		value.Connect = IntegrationDefaultConnectTimeout
	}
	if value.TLSHandshake == 0 {
		value.TLSHandshake = IntegrationDefaultTLSHandshake
	}
	if value.Request == 0 {
		value.Request = IntegrationDefaultRequestTimeout
	}
	if value.ResponseHeader == 0 {
		value.ResponseHeader = IntegrationDefaultResponseHeader
		if value.ResponseHeader > value.Request {
			value.ResponseHeader = value.Request
		}
	}
	if value.Connect <= 0 || value.TLSHandshake <= 0 || value.Request <= 0 || value.ResponseHeader <= 0 ||
		value.Connect > IntegrationMaxConnectTimeout || value.TLSHandshake > IntegrationMaxTLSHandshakeTimeout ||
		value.ResponseHeader > value.Request || value.ResponseHeader > IntegrationMaxResponseHeaderTimeout ||
		value.Request > IntegrationMaxRequestTimeout {
		return TransportTimeouts{}, newTransportError(TransportErrorInvalidConfig)
	}
	return value, nil
}

func normalizeMaxResponseBytes(value int64) (int64, error) {
	if value == 0 {
		return IntegrationDefaultResponseBytes, nil
	}
	if value < IntegrationMinResponseBytes || value > IntegrationMaxResponseBytes {
		return 0, newTransportError(TransportErrorInvalidConfig)
	}
	return value, nil
}

func normalizeContentTypes(values []string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return map[string]struct{}{"application/json": {}}, nil
	}
	if len(values) > 16 {
		return nil, newTransportError(TransportErrorInvalidConfig)
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil || mediaType == "" {
			return nil, newTransportError(TransportErrorInvalidConfig)
		}
		result[strings.ToLower(mediaType)] = struct{}{}
	}
	return result, nil
}

func isAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func cloneAndValidatePathParameters(source map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(source))
	for name, value := range source {
		if !pathParameterPattern.MatchString(name) || len(value) == 0 || len(value) > maxPathParameterLength ||
			strings.ContainsAny(value, "/\\?&#%") || strings.Contains(value, "..") || containsControlCharacter(value) {
			return nil, newTransportError(TransportErrorInvalidURL)
		}
		result[name] = value
	}
	return result, nil
}

func cloneAndValidateQueryParameters(source map[string][]string) (map[string][]string, error) {
	if len(source) > maxQueryParameterCount {
		return nil, newTransportError(TransportErrorInvalidConfig)
	}
	result := make(map[string][]string, len(source))
	for name, values := range source {
		name = strings.TrimSpace(name)
		if name == "" || len(name) > maxQueryNameLength || containsControlCharacter(name) || len(values) == 0 {
			return nil, newTransportError(TransportErrorInvalidConfig)
		}
		cloned := make([]string, len(values))
		for index, value := range values {
			if len(value) > maxQueryValueLength || containsControlCharacter(value) {
				return nil, newTransportError(TransportErrorInvalidConfig)
			}
			cloned[index] = value
		}
		result[name] = cloned
	}
	return result, nil
}

func cloneAndValidateHeaders(source map[string]string, authentication bool) (map[string]string, error) {
	result := make(map[string]string, len(source))
	for name, value := range source {
		normalizedName := strings.ToLower(strings.TrimSpace(name))
		if !isAllowedHeader(normalizedName, authentication) {
			return nil, newTransportError(TransportErrorInvalidConfig)
		}
		if err := validateHeader(normalizedName, value); err != nil {
			return nil, err
		}
		result[normalizedName] = strings.TrimSpace(value)
	}
	if err := validateHeaderBudget(result); err != nil {
		return nil, err
	}
	return result, nil
}

func validateHeader(name string, value string) error {
	if name == "" || len(name) > maxHeaderNameLength || len(value) == 0 || len(value) > maxHeaderValueLength ||
		containsControlCharacter(name) || containsControlCharacter(value) {
		return newTransportError(TransportErrorInvalidConfig)
	}
	return nil
}

func validateHeaderBudget(headers map[string]string) error {
	if len(headers) > maxHeaderCount {
		return newTransportError(TransportErrorInvalidConfig)
	}
	total := 0
	for name, value := range headers {
		total += len(name) + len(value)
	}
	if total > maxHeaderTotalLength {
		return newTransportError(TransportErrorInvalidConfig)
	}
	return nil
}

func isAllowedHeader(name string, authentication bool) bool {
	if authentication {
		return name == "authorization" || name == "x-api-key"
	}
	switch name {
	case "accept", "accept-language", "content-type", "user-agent", "x-request-id", "x-trace-id", "x-correlation-id", "idempotency-key", "x-idempotency-key":
		return true
	default:
		return false
	}
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(mediaType, "application/json")
}

func validateBaseURL(raw string) error {
	value, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || value == nil || value.Scheme == "" || value.Host == "" || value.User != nil ||
		value.RawQuery != "" || value.Fragment != "" || value.RawPath != "" || strings.Contains(value.Path, "//") ||
		strings.Contains(value.Path, "\\") || strings.Contains(value.Path, "%") {
		return newTransportError(TransportErrorInvalidURL)
	}
	return validateTargetURL(value)
}

func validateRelativePath(raw string) error {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "?#\\%") ||
		strings.Contains(raw, "//") || strings.Contains(raw, "://") || containsControlCharacter(raw) {
		return newTransportError(TransportErrorInvalidURL)
	}
	for _, segment := range strings.Split(raw, "/") {
		if segment == "." || segment == ".." {
			return newTransportError(TransportErrorInvalidURL)
		}
	}
	return nil
}

func renderRelativePath(template string, parameters map[string]string) (string, string, error) {
	value := template
	for name, parameter := range parameters {
		placeholder := "{" + name + "}"
		if !strings.Contains(value, placeholder) {
			return "", "", newTransportError(TransportErrorInvalidURL)
		}
		value = strings.ReplaceAll(value, placeholder, url.PathEscape(parameter))
	}
	if strings.ContainsAny(value, "{}") {
		return "", "", newTransportError(TransportErrorInvalidURL)
	}
	decoded, err := url.PathUnescape(value)
	if err != nil || strings.ContainsAny(decoded, "\\?#") || strings.Contains(decoded, "//") {
		return "", "", newTransportError(TransportErrorInvalidURL)
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return "", "", newTransportError(TransportErrorInvalidURL)
		}
	}
	return value, decoded, nil
}

func validateTargetURL(value *url.URL) error {
	if value == nil || (value.Scheme != "https" && value.Scheme != "http") || value.Host == "" || value.User != nil ||
		value.Fragment != "" || strings.Contains(value.Path, "//") || strings.Contains(value.Path, "\\") || strings.Contains(value.Path, "%") {
		return newTransportError(TransportErrorInvalidURL)
	}
	if port := value.Port(); port != "" {
		parsed, err := strconv.Atoi(port)
		if err != nil || parsed <= 0 || parsed > 65535 {
			return newTransportError(TransportErrorInvalidURL)
		}
	}
	return nil
}

func containsControlCharacter(value string) bool {
	return strings.ContainsAny(value, "\r\n\x00")
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneStringSliceMap(source map[string][]string) map[string][]string {
	if len(source) == 0 {
		return map[string][]string{}
	}
	result := make(map[string][]string, len(source))
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}

func cloneContentTypes(source map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(source))
	for value := range source {
		result[value] = struct{}{}
	}
	return result
}

func safeResponseHeaders(headers http.Header) map[string]string {
	allowed := []string{"Content-Type", "Content-Length", "ETag", "Last-Modified", "Retry-After", "X-Request-ID"}
	result := make(map[string]string, len(allowed))
	for _, name := range allowed {
		value := strings.TrimSpace(headers.Get(name))
		if value != "" && len(value) <= maxHeaderValueLength && !containsControlCharacter(value) {
			result[strings.ToLower(name)] = value
		}
	}
	return result
}

func readResponseBody(body io.ReadCloser, maxBytes int64) ([]byte, bool, error) {
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > maxBytes {
		return nil, false, nil
	}
	return data, true, nil
}
