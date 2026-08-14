/**
 * @Author: Nan
 * @Date: 2023/3/18 17:04
 */

package middleware

import (
	"backend/dto/response"
	"backend/internal/utils"
	"backend/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxAuditPayloadLength = 64 * 1024
const maxAuditErrorCodeLength = 64
const maxAuditErrorMessageLength = 2048

const (
	accessAuditResultSuccess = "success"
	accessAuditResultFailed  = "failed"
)

var (
	sensitiveJSONStringPattern = regexp.MustCompile(`(?i)("([^"]*(password|token|secret|authorization|salt|captcha|otp|verify_code|verification_code|sms_code)[^"]*)"\s*:\s*)"[^"]*"`)
	sensitiveJSONArrayPattern  = regexp.MustCompile(`(?i)("([^"]*(password|token|secret|authorization|salt|captcha|otp|verify_code|verification_code|sms_code)[^"]*)"\s*:\s*)\[[^\]]*\]`)
	sensitiveURLQueryPattern   = regexp.MustCompile(`(?i)([?&](?:password|token|access_token|refresh_token|authorization|secret|salt)=)[^&#\s"']+`)
)

type accessAuditMeta struct {
	Action       string
	ResourceType string
	ResourceCode string
	ResourceId   string
	MenuId       int
}

type accessLogWriter interface {
	CreateAccessLog(ctx context.Context, log model.AccessLog) error
}

type accessAuditResult struct {
	Success      bool
	Result       string
	ErrorCode    string
	ErrorMessage string
}

func LogHandler(logService accessLogWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		EnsureLogContext(c)
		startTime := time.Now()
		var body interface{}
		var query = c.Request.URL.Query()
		if shouldCaptureRequestBody(c) {
			_ = c.ShouldBindBodyWith(&body, binding.JSON)
		}

		blw := &response.BufferedResponseWriter{
			ResponseWriter: c.Writer,
			Body:           bytes.NewBufferString(""),
			MaxBodyBytes:   maxAuditPayloadLength,
		}
		c.Writer = blw

		c.Next()
		duration := time.Since(startTime)
		responseBody := blw.Body.String()
		if blw.Truncated {
			responseBody += "...[truncated]"
		}

		sanitizedPath := sanitizeAccessLogURLPath(c.Request.URL.Path)
		queryStr, _ := json.Marshal(query)
		bodyStr, _ := json.Marshal(body)
		meta := mergeAccessAuditMeta(
			classifyAccessAudit(c.Request.Method, c.Request.URL.Path, body),
			auditContextFromGin(c),
		)
		userId, userName := currentAccessLogUser(c)
		statusCode := c.Writer.Status()
		auditResult := resolveAccessAuditResult(c, statusCode, responseBody)

		var accessLog = model.AccessLog{
			Basic:        model.Basic{},
			UserId:       userId,
			UserName:     userName,
			RequestId:    RequestID(c),
			TraceId:      TraceID(c),
			Method:       c.Request.Method,
			Ip:           c.ClientIP(),
			Locality:     "",
			Url:          sanitizedPath,
			Action:       meta.Action,
			ResourceType: meta.ResourceType,
			ResourceCode: meta.ResourceCode,
			ResourceId:   meta.ResourceId,
			MenuId:       meta.MenuId,
			StatusCode:   statusCode,
			Success:      auditResult.Success,
			Result:       auditResult.Result,
			ErrorCode:    auditResult.ErrorCode,
			ErrorMessage: auditResult.ErrorMessage,
			DurationMs:   duration.Milliseconds(),
			Body:         sanitizeAccessLogPayload(c.Request.URL.Path, string(bodyStr)),
			Query:        sanitizeAccessLogPayload(c.Request.URL.Path, string(queryStr)),
			Response:     sanitizeAccessLogPayload(c.Request.URL.Path, responseBody),
		}
		if !AccessAuditPersisted(c) {
			err := logService.CreateAccessLog(c.Request.Context(), accessLog)
			if err != nil {
				zap.L().Error(
					"access log storage failed",
					zap.Error(err),
					zap.String("request_id", accessLog.RequestId),
					zap.String("trace_id", accessLog.TraceId),
				)
			}
		}
		zap.L().Info(
			"access request completed",
			zap.String("request_id", accessLog.RequestId),
			zap.String("trace_id", accessLog.TraceId),
			zap.Int("user_id", accessLog.UserId),
			zap.Int("menu_id", accessLog.MenuId),
			zap.String("action", accessLog.Action),
			zap.String("resource_type", accessLog.ResourceType),
			zap.String("resource_code", accessLog.ResourceCode),
			zap.String("resource_id", accessLog.ResourceId),
			zap.String("path", sanitizedPath),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
			zap.Int("status_code", accessLog.StatusCode),
			zap.Bool("success", accessLog.Success),
			zap.String("result", accessLog.Result),
			zap.String("error_code", accessLog.ErrorCode),
			zap.Int64("duration_ms", accessLog.DurationMs),
		)
	}
}

func currentAccessLogUser(c *gin.Context) (int, string) {
	userVal, exists := c.Get("user")
	if !exists {
		return 0, ""
	}
	user, ok := userVal.(model.SysUser)
	if !ok {
		return 0, ""
	}
	return user.Id, user.UserName
}

func shouldCaptureRequestBody(c *gin.Context) bool {
	if c.Request == nil || c.Request.Body == nil {
		return false
	}
	if c.Request.ContentLength == 0 {
		return false
	}
	if c.Request.ContentLength > maxAuditPayloadLength {
		return false
	}
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	return strings.Contains(contentType, binding.MIMEJSON)
}

func mergeAccessAuditMeta(meta accessAuditMeta, explicit AuditContext) accessAuditMeta {
	if explicit.MenuID != 0 {
		meta.MenuId = explicit.MenuID
	}
	if explicit.Action != "" {
		meta.Action = explicit.Action
	}
	if explicit.ResourceType != "" {
		meta.ResourceType = explicit.ResourceType
	}
	if explicit.ResourceCode != "" {
		meta.ResourceCode = explicit.ResourceCode
	}
	if explicit.ResourceID != "" {
		meta.ResourceId = explicit.ResourceID
	}
	return meta
}

func resolveAccessAuditResult(c *gin.Context, statusCode int, responseBody string) accessAuditResult {
	result := accessAuditResult{
		Success: statusCode < 400 && len(c.Errors) == 0,
	}

	var payload struct {
		Success      *bool           `json:"success"`
		ErrorCode    json.RawMessage `json:"error_code"`
		ErrorMessage string          `json:"error_message"`
	}
	if err := json.Unmarshal([]byte(responseBody), &payload); err == nil {
		if payload.Success != nil {
			result.Success = result.Success && *payload.Success
		}
		result.ErrorCode = truncateAuditErrorCode(
			utils.SanitizeInput(parseAccessAuditErrorCode(payload.ErrorCode)),
		)
		result.ErrorMessage = payload.ErrorMessage
	}

	if result.Success {
		result.Result = accessAuditResultSuccess
		result.ErrorCode = ""
		result.ErrorMessage = ""
		return result
	}

	result.Result = accessAuditResultFailed
	if len(c.Errors) > 0 && (result.ErrorCode == "" || result.ErrorMessage == "") {
		clientErr, _ := toClientError(c.Errors[0].Err)
		if clientErr != nil {
			if result.ErrorCode == "" {
				result.ErrorCode = strconv.Itoa(clientErr.ErrorCode)
			}
			if result.ErrorMessage == "" {
				result.ErrorMessage = clientErr.ErrorMessage
			}
		}
	}
	result.ErrorMessage = truncateAuditErrorMessage(utils.SanitizeInput(result.ErrorMessage))
	return result
}

func parseAccessAuditErrorCode(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var stringCode string
	if err := json.Unmarshal(raw, &stringCode); err == nil {
		return strings.TrimSpace(stringCode)
	}
	return strings.TrimSpace(string(raw))
}

func truncateAuditErrorCode(code string) string {
	if len(code) <= maxAuditErrorCodeLength {
		return code
	}
	return code[:maxAuditErrorCodeLength]
}

func truncateAuditErrorMessage(message string) string {
	if len(message) <= maxAuditErrorMessageLength {
		return message
	}
	return message[:maxAuditErrorMessageLength] + "...[truncated]"
}

func classifyAccessAudit(method, path string, body interface{}) accessAuditMeta {
	data := accessBodyMap(body)
	if strings.Contains(path, "/admin/configure/test-email") && method == "POST" {
		return accessAuditMeta{
			Action:       "configure_test_email",
			ResourceType: "configure",
		}
	}
	if strings.Contains(path, "/admin/configure/") && method == "PUT" {
		return accessAuditMeta{
			Action:       "configure_update",
			ResourceType: "configure",
			ResourceId:   lastPathValue(path),
		}
	}
	if strings.Contains(path, "/admin/table/publish/") && method == "POST" {
		return accessAuditMeta{
			Action:       "table_publish",
			ResourceType: "table",
			ResourceCode: lastPathValue(path),
		}
	}
	if strings.Contains(path, "/admin/table/unpublish/") && method == "POST" {
		return accessAuditMeta{
			Action:       "table_unpublish",
			ResourceType: "table",
			ResourceCode: lastPathValue(path),
		}
	}
	if strings.Contains(path, "/admin/table/field/") {
		return accessAuditMeta{
			Action:       tableFieldAction(method),
			ResourceType: "table_field",
			ResourceId:   lastPathValue(path),
		}
	}
	if strings.HasSuffix(path, "/admin/table/field") && method == "POST" {
		return accessAuditMeta{
			Action:       "table_field_create",
			ResourceType: "table_field",
			ResourceId:   stringFromMap(data, "table_id"),
		}
	}
	if strings.Contains(path, "/admin/table/") && (method == "PUT" || method == "DELETE") {
		return accessAuditMeta{
			Action:       tableAction(method),
			ResourceType: "table",
			ResourceId:   lastPathValue(path),
		}
	}
	if strings.HasSuffix(path, "/admin/table") && method == "POST" {
		return accessAuditMeta{
			Action:       "table_create",
			ResourceType: "table",
			ResourceCode: stringFromMap(data, "table_code"),
		}
	}
	if strings.Contains(path, "/admin/generalization/query/code/") && method == "POST" {
		return accessAuditMeta{
			Action:       "lowcode_query",
			ResourceType: "lowcode_table",
			ResourceCode: firstNonEmpty(stringFromMap(data, "table_code"), lastPathValue(path)),
			MenuId:       intFromMap(data, "menu_id"),
		}
	}
	if strings.Contains(path, "/admin/generalization/create") && method == "POST" {
		return accessAuditMeta{
			Action:       "lowcode_create",
			ResourceType: "lowcode_row",
			ResourceCode: stringFromMap(data, "table_code"),
			MenuId:       intFromMap(data, "menu_id"),
		}
	}
	if strings.Contains(path, "/admin/generalization/update") && method == "PUT" {
		return accessAuditMeta{
			Action:       "lowcode_update",
			ResourceType: "lowcode_row",
			ResourceCode: stringFromMap(data, "table_code"),
			ResourceId:   stringFromMap(data, "id"),
			MenuId:       intFromMap(data, "menu_id"),
		}
	}
	if strings.Contains(path, "/admin/generalization/delete") && method == "DELETE" {
		return accessAuditMeta{
			Action:       "lowcode_delete",
			ResourceType: "lowcode_row",
			ResourceCode: stringFromMap(data, "table_code"),
			ResourceId:   stringFromMap(data, "id"),
			MenuId:       intFromMap(data, "menu_id"),
		}
	}
	return accessAuditMeta{}
}

func accessBodyMap(body interface{}) map[string]interface{} {
	data, ok := body.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return data
}

func tableAction(method string) string {
	switch method {
	case "PUT":
		return "table_update"
	case "DELETE":
		return "table_delete"
	default:
		return ""
	}
}

func tableFieldAction(method string) string {
	switch method {
	case "PUT":
		return "table_field_update"
	case "DELETE":
		return "table_field_delete"
	default:
		return ""
	}
}

func lastPathValue(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	value, err := url.PathUnescape(parts[len(parts)-1])
	if err != nil {
		return parts[len(parts)-1]
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringFromMap(data map[string]interface{}, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64:
		if math.Trunc(v) == v {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		f := float64(v)
		if math.Trunc(f) == f {
			return strconv.FormatInt(int64(f), 10)
		}
		return strconv.FormatFloat(f, 'f', -1, 32)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	default:
		return fmt.Sprint(v)
	}
}

func intFromMap(data map[string]interface{}, key string) int {
	value := stringFromMap(data, key)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func sanitizeLogPayload(input string) string {
	return sanitizeAccessLogPayload("", input)
}

func sanitizeAccessLogPayload(path, input string) string {
	if input == "" {
		return input
	}
	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err == nil {
		data = redactSensitiveData(data)
		data = redactPathSensitiveData(path, data)
		if masked, err := json.Marshal(data); err == nil {
			return truncateLogPayload(utils.SanitizeInput(string(masked)))
		}
	}
	return truncateLogPayload(utils.SanitizeInput(redactPathSensitiveText(path, redactSensitiveText(input))))
}

func truncateLogPayload(input string) string {
	if len(input) <= maxAuditPayloadLength {
		return input
	}
	return input[:maxAuditPayloadLength] + "...[truncated]"
}

func redactSensitiveData(value interface{}) interface{} {
	switch data := value.(type) {
	case map[string]interface{}:
		for key, child := range data {
			if isSensitiveLogKey(key) {
				data[key] = "***"
				continue
			}
			data[key] = redactSensitiveData(child)
		}
		return data
	case []interface{}:
		for i, child := range data {
			data[i] = redactSensitiveData(child)
		}
		return data
	case string:
		return redactSensitiveURLQuery(data)
	default:
		return value
	}
}

func isSensitiveLogKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
	normalized = strings.ReplaceAll(normalized, "-", "")
	sensitiveKeys := []string{
		"password",
		"oldpassword",
		"newpassword",
		"confirmpassword",
		"token",
		"accesstoken",
		"refreshtoken",
		"authorization",
		"secret",
		"sessionsecret",
		"salt",
		"captcha",
		"captchaid",
		"smscode",
		"verifycode",
		"verificationcode",
		"otp",
		"onetimecode",
	}
	for _, sensitiveKey := range sensitiveKeys {
		if normalized == sensitiveKey {
			return true
		}
	}
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "captcha")
}

func redactSensitiveText(input string) string {
	input = sensitiveJSONStringPattern.ReplaceAllString(input, `${1}"***"`)
	input = sensitiveJSONArrayPattern.ReplaceAllString(input, `${1}["***"]`)
	return redactSensitiveURLQuery(input)
}

func redactSensitiveURLQuery(input string) string {
	return sensitiveURLQueryPattern.ReplaceAllString(input, `${1}***`)
}

func redactPathSensitiveData(path string, value interface{}) interface{} {
	switch data := value.(type) {
	case map[string]interface{}:
		for key, child := range data {
			normalized := normalizeAuditKey(key)
			if shouldRedactOneTimeCode(path, normalized) || shouldRedactCaptchaImage(path, normalized) || shouldRedactAuthenticationPrincipal(path, normalized) {
				data[key] = "***"
				continue
			}
			data[key] = redactPathSensitiveData(path, child)
		}
		return data
	case []interface{}:
		for i, child := range data {
			data[i] = redactPathSensitiveData(path, child)
		}
		return data
	default:
		return value
	}
}

func redactPathSensitiveText(path, input string) string {
	if isOneTimeCredentialPath(path) {
		input = regexp.MustCompile(`(?i)("code"\s*:\s*)"[^"]*"`).ReplaceAllString(input, `${1}"***"`)
		input = regexp.MustCompile(`(?i)("code"\s*:\s*)\[[^\]]*\]`).ReplaceAllString(input, `${1}["***"]`)
	}
	if isCaptchaPath(path) {
		input = regexp.MustCompile(`(?i)("(?:image|captcha_id|captchaId)"\s*:\s*)"[^"]*"`).ReplaceAllString(input, `${1}"***"`)
		input = regexp.MustCompile(`(?i)("(?:image|captcha_id|captchaId)"\s*:\s*)\[[^\]]*\]`).ReplaceAllString(input, `${1}["***"]`)
	}
	return input
}

func shouldRedactOneTimeCode(path, normalizedKey string) bool {
	return isOneTimeCredentialPath(path) && normalizedKey == "code"
}

func shouldRedactCaptchaImage(path, normalizedKey string) bool {
	return isCaptchaPath(path) && (normalizedKey == "image" || normalizedKey == "captchaid")
}

func shouldRedactAuthenticationPrincipal(path, normalizedKey string) bool {
	normalizedPath := strings.ToLower(path)
	isLogin := strings.HasSuffix(normalizedPath, "/login") ||
		strings.Contains(normalizedPath, "/sms_code_login") ||
		strings.Contains(normalizedPath, "/sso_login")
	return isLogin && (normalizedKey == "username" || normalizedKey == "mobile" || normalizedKey == "email")
}

func normalizeAuditKey(key string) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	return normalized
}

func isOneTimeCredentialPath(path string) bool {
	normalized := strings.ToLower(path)
	return strings.Contains(normalized, "/sms_code_login") ||
		strings.Contains(normalized, "/send_sms/") ||
		strings.Contains(normalized, "/sso_login")
}

func isCaptchaPath(path string) bool {
	return strings.Contains(strings.ToLower(path), "/admin/captcha")
}

func sanitizeAccessLogURLPath(path string) string {
	if path == "" {
		return path
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		switch part {
		case "send_sms":
			if i+1 < len(parts) {
				parts[i+1] = "***"
			}
		case "check_sms_status":
			if i+2 < len(parts) {
				parts[i+2] = "***"
			}
		}
	}
	return strings.Join(parts, "/")
}
