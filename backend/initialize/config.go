/**
 * @Author: Nan
 * @Date: 2024/10/11 11:48
 */

package initialize

import (
	"backend/config"
	"backend/internal/database"
	"fmt"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"reflect"
	"strings"
)

func LoadConfig() (*config.Server, error) {
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "dev" // 默认使用本地环境的配置
	}
	filename := fmt.Sprintf("config-%s.yaml", environment)
	if _, err := os.Stat(filename); os.IsNotExist(err) && isProductionEnvironment(environment) {
		filename = "config-pro.yaml"
	}
	v := viper.New()
	v.SetConfigFile(filename)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	// 绑定环境变量
	v.AutomaticEnv()
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	bindEnvs(v, &config.Server{})

	var cfg config.Server
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	applyUploadListEnvOverrides(&cfg)
	applySecurityEnvOverrides(&cfg)
	applyIntegrationEnvOverrides(&cfg)
	if err := validateSecureConfig(environment, &cfg); err != nil {
		return nil, err
	}
	if requiresCasbinPolicyCoverage(environment) {
		cfg.Security.EnforceCasbinPolicyCoverage = true
	}
	return &cfg, nil
}

func applyIntegrationEnvOverrides(cfg *config.Server) {
	if cfg == nil {
		return
	}
	if values := parseCSVEnv(os.Getenv("APP_INTEGRATION_ENDPOINT_POLICY_APPROVED_PRIVATE_CIDRS")); len(values) > 0 {
		cfg.Integration.EndpointPolicy.ApprovedPrivateCIDRs = values
	}
}

func validateSecureConfig(environment string, cfg *config.Server) error {
	if !requiresSecureConfig(environment) {
		return nil
	}

	var problems []string
	if insecureSecureConfigValue(cfg.Session.Secret) {
		problems = append(problems, "session.secret must be set with APP_SESSION_SECRET to a non-default value with at least 32 characters")
	}
	if insecureSecureConfigValue(cfg.Conf.Salt) {
		problems = append(problems, "conf.salt must be set with APP_CONF_SALT to a non-default value with at least 32 characters")
	}

	problems = append(problems, validateSecureDB("dbs.primary", cfg.DBS.Primary)...)
	if strings.TrimSpace(cfg.DBS.Primary.Prefix) != "" {
		problems = append(problems, "dbs.primary.prefix is not supported by the production migration and preflight contract")
	}
	if cfg.DBS.Primary.TLS.Mode == database.PostgresTLSDisable {
		problems = append(problems, "dbs.primary.tls.mode must require certificate-protected transport in production")
	}
	if missingConfigValue(cfg.Redis.Host) {
		problems = append(problems, "redis.host must be set with APP_REDIS_HOST")
	}
	if cfg.Redis.Port <= 0 {
		problems = append(problems, "redis.port must be set with APP_REDIS_PORT")
	}
	if insecureCredentialValue(cfg.Redis.Password) {
		problems = append(problems, "redis.password must be set with APP_REDIS_PASSWORD to a non-default value")
	}
	if !cfg.Redis.TLS.Enabled {
		problems = append(problems, "redis.tls.enabled must be true in production")
	}
	if cfg.Redis.TLS.Enabled && missingConfigValue(cfg.Redis.TLS.ServerName) {
		problems = append(problems, "redis.tls.server_name must be set in production")
	}
	if cfg.Upload.ChunkTTLHours <= 0 {
		problems = append(problems, "upload.chunk_ttl_hours must be greater than zero in production")
	}
	if cfg.Upload.ChunkCleanupMinutes <= 0 {
		problems = append(problems, "upload.chunk_cleanup_minutes must be greater than zero in production")
	}
	if cfg.Upload.PublicPreview {
		problems = append(problems, "upload.public_preview must be false in production")
	}
	problems = append(problems, validateProductionUploadAllowList(cfg.Upload)...)
	problems = append(problems, validateProductionUploadAccess(cfg.Upload)...)
	problems = append(problems, validateProductionCORS(cfg.Security.CORSAllowedOrigins)...)

	if strings.EqualFold(strings.TrimSpace(cfg.Upload.Driver), "oss") {
		if missingConfigValue(cfg.Upload.OSS.Endpoint) {
			problems = append(problems, "upload.oss.endpoint must be set when upload.driver=oss")
		}
		if missingConfigValue(cfg.Upload.OSS.BucketName) {
			problems = append(problems, "upload.oss.bucket_name must be set when upload.driver=oss")
		}
		if insecureCredentialValue(cfg.Upload.OSS.AccessKeyID) {
			problems = append(problems, "upload.oss.access_key_id must be set to a non-default value when upload.driver=oss")
		}
		if insecureCredentialValue(cfg.Upload.OSS.AccessKeySecret) {
			problems = append(problems, "upload.oss.access_key_secret must be set to a non-default value when upload.driver=oss")
		}
	}

	if cfg.ALiYun.SMS.AccessKeyId != "" || cfg.ALiYun.SMS.AccessKeySecret != "" {
		if insecureCredentialValue(cfg.ALiYun.SMS.AccessKeyId) {
			problems = append(problems, "aliyun.sms.access_key_id must be empty or set to a non-default value")
		}
		if insecureCredentialValue(cfg.ALiYun.SMS.AccessKeySecret) {
			problems = append(problems, "aliyun.sms.access_key_secret must be empty or set to a non-default value")
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("insecure production config: %s", strings.Join(problems, "; "))
	}
	return nil
}

func requiresSecureConfig(environment string) bool {
	if isProductionEnvironment(environment) {
		return true
	}
	return parseBoolEnv(os.Getenv("APP_REQUIRE_SECURE_CONFIG"))
}

func requiresCasbinPolicyCoverage(environment string) bool {
	if isProductionEnvironment(environment) {
		return true
	}
	return parseBoolEnv(os.Getenv("APP_ENFORCE_CASBIN_POLICY_COVERAGE"))
}

func isProductionEnvironment(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "pro", "prod", "production":
		return true
	default:
		return false
	}
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func applyUploadListEnvOverrides(cfg *config.Server) {
	if cfg == nil {
		return
	}
	if values := parseCSVEnv(os.Getenv("APP_UPLOAD_ALLOWED_EXTENSIONS")); len(values) > 0 {
		cfg.Upload.AllowedExtensions = values
	}
	if values := parseCSVEnv(os.Getenv("APP_UPLOAD_ALLOWED_MIME_TYPES")); len(values) > 0 {
		cfg.Upload.AllowedMimeTypes = values
	}
}

func applySecurityEnvOverrides(cfg *config.Server) {
	if cfg == nil {
		return
	}
	if values := parseCSVEnv(firstNonEmptyEnv("APP_SECURITY_CORS_ALLOWED_ORIGINS", "APP_CORS_ALLOWED_ORIGINS")); len(values) > 0 {
		cfg.Security.CORSAllowedOrigins = values
	}
	if value := os.Getenv("APP_SECURITY_CORS_ALLOW_CREDENTIALS"); value != "" {
		cfg.Security.CORSAllowCredentials = parseBoolEnv(value)
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func parseCSVEnv(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func validateProductionCORS(origins []string) []string {
	var problems []string
	if len(origins) == 0 {
		return []string{"security.cors_allowed_origins must be set in production"}
	}
	for _, origin := range origins {
		normalized := strings.ToLower(strings.TrimSpace(origin))
		if normalized == "" || normalized == "*" || placeholderConfigValue(normalized) {
			problems = append(problems, "security.cors_allowed_origins must contain only explicit trusted origins in production")
			break
		}
	}
	return problems
}

func validateProductionUploadAllowList(upload config.Upload) []string {
	var problems []string
	for _, value := range upload.AllowedExtensions {
		ext := normalizeUploadExtension(value)
		if ext != "" && dangerousUploadExtension(ext) {
			problems = append(problems, fmt.Sprintf("upload.allowed_extensions must not include active or executable type %q in production", ext))
		}
	}
	for _, value := range upload.AllowedMimeTypes {
		mimeType := normalizeUploadMimeType(value)
		if mimeType != "" && dangerousUploadMimeType(mimeType) {
			problems = append(problems, fmt.Sprintf("upload.allowed_mime_types must not include active or executable type %q in production", mimeType))
		}
	}
	return problems
}

func validateProductionUploadAccess(upload config.Upload) []string {
	var problems []string
	if !secureUploadURLPrefix(upload.BaseURL, true) {
		problems = append(problems, "upload.base_url must be a rooted path or HTTPS URL prefix in production")
	}
	if strings.EqualFold(strings.TrimSpace(upload.Driver), "oss") {
		if value := strings.TrimSpace(upload.OSS.BaseURL); value != "" && !secureUploadURLPrefix(value, false) {
			problems = append(problems, "upload.oss.base_url must be an HTTPS URL prefix when set in production")
		}
		if invalidOSSBasePath(upload.OSS.BasePath) {
			problems = append(problems, "upload.oss.base_path must be a relative object-key prefix without path traversal")
		}
	}
	return problems
}

func secureUploadURLPrefix(value string, allowRelativePath bool) bool {
	normalized := strings.TrimSpace(value)
	if missingConfigValue(normalized) ||
		strings.ContainsAny(normalized, " \t\r\n") ||
		strings.ContainsAny(normalized, "?#") {
		return false
	}
	if strings.HasPrefix(normalized, "/") {
		return allowRelativePath &&
			!strings.HasPrefix(normalized, "//") &&
			normalized != "/" &&
			!pathHasTraversal(normalized)
	}
	parsed, err := url.Parse(normalized)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func invalidOSSBasePath(value string) bool {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return false
	}
	return strings.HasPrefix(normalized, "/") ||
		strings.HasPrefix(normalized, "\\") ||
		strings.HasPrefix(strings.ToLower(normalized), "http://") ||
		strings.HasPrefix(strings.ToLower(normalized), "https://") ||
		pathHasTraversal(normalized)
}

func pathHasTraversal(value string) bool {
	normalized := strings.ReplaceAll(value, "\\", "/")
	return normalized == ".." ||
		strings.HasPrefix(normalized, "../") ||
		strings.HasSuffix(normalized, "/..") ||
		strings.Contains(normalized, "/../")
}

func normalizeUploadExtension(value string) string {
	ext := strings.ToLower(strings.TrimSpace(value))
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

func normalizeUploadMimeType(value string) string {
	if idx := strings.Index(value, ";"); idx >= 0 {
		value = value[:idx]
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func dangerousUploadExtension(ext string) bool {
	switch ext {
	case ".html", ".htm", ".svg",
		".js", ".mjs", ".cjs", ".jsx", ".ts", ".tsx", ".vue",
		".sh", ".bash", ".zsh", ".fish", ".bat", ".cmd", ".ps1",
		".exe", ".dll", ".msi", ".app", ".dmg", ".pkg",
		".jar", ".war", ".class",
		".php", ".jsp", ".asp", ".aspx",
		".py", ".rb", ".pl", ".go", ".rs":
		return true
	default:
		return false
	}
}

func dangerousUploadMimeType(mimeType string) bool {
	switch mimeType {
	case "text/html",
		"image/svg+xml",
		"application/javascript",
		"text/javascript",
		"application/x-javascript",
		"application/ecmascript",
		"text/ecmascript",
		"text/x-shellscript",
		"application/x-sh",
		"application/x-msdownload",
		"application/x-msdos-program",
		"application/x-ms-installer",
		"application/x-dosexec",
		"application/java-archive",
		"application/x-httpd-php",
		"application/octet-stream":
		return true
	default:
		return false
	}
}

func insecureSecureConfigValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if len(normalized) < 32 {
		return true
	}
	defaultMarkers := []string{
		"replace-with",
		"change-me",
		"changeme",
		"local-docker",
		"local-external",
		"sweet-admin",
		"placeholder",
		"example",
		"secret",
	}
	for _, marker := range defaultMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func validateSecureDB(prefix string, db config.DB) []string {
	var problems []string
	if missingConfigValue(db.Host) {
		problems = append(problems, prefix+".host must be set")
	}
	if db.Port <= 0 {
		problems = append(problems, prefix+".port must be set")
	}
	if missingConfigValue(db.Name) {
		problems = append(problems, prefix+".name must be set")
	}
	if insecureDBUserValue(db.User) {
		problems = append(problems, prefix+".user must be set to a non-root, non-default value")
	}
	if insecureCredentialValue(db.Password) {
		problems = append(problems, prefix+".password must be set to a non-default value")
	}
	mode := strings.ToLower(strings.TrimSpace(db.TLS.Mode))
	if !database.SupportedPostgresTLSMode(mode) {
		problems = append(problems, prefix+".tls.mode must be one of disable, require, verify-ca, verify-full")
	}
	if (strings.TrimSpace(db.TLS.CertFile) == "") != (strings.TrimSpace(db.TLS.KeyFile) == "") {
		problems = append(problems, prefix+".tls.cert_file and tls.key_file must be configured together")
	}
	return problems
}

func missingConfigValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "" || placeholderConfigValue(normalized)
}

func insecureDBUserValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return missingConfigValue(normalized) || normalized == "root" || normalized == "admin"
}

func insecureCredentialValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || len(normalized) < 8 || placeholderConfigValue(normalized) {
		return true
	}
	defaultMarkers := []string{
		"sweet_admin",
		"admin123",
		"password",
		"123456",
		"local-docker",
		"local-external",
	}
	for _, marker := range defaultMarkers {
		if normalized == marker || strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func placeholderConfigValue(normalized string) bool {
	placeholders := []string{
		"replace-with",
		"change-me",
		"changeme",
		"placeholder",
		"example",
		"<",
		">",
	}
	for _, placeholder := range placeholders {
		if strings.Contains(normalized, placeholder) {
			return true
		}
	}
	return false
}

// 绑定结构体的所有字段到环境变量
func bindEnvs(v *viper.Viper, s interface{}, prefix ...string) {
	ps := ""
	if len(prefix) > 0 {
		ps = prefix[0] + "."
	}
	fields := reflect.TypeOf(s).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		envKey := ps + field.Tag.Get("mapstructure")
		if envKey == ps {
			envKey = ps + strings.ToLower(field.Name)
		}
		if field.Type.Kind() == reflect.Struct {
			bindEnvs(v, reflect.New(field.Type).Interface(), envKey)
		} else {
			_ = v.BindEnv(envKey)
		}
	}
}
