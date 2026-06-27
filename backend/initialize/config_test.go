package initialize

import (
	"backend/config"
	"reflect"
	"strings"
	"testing"
)

func TestValidateSecureConfigAllowsLocalDefaultsWhenNotRequired(t *testing.T) {
	cfg := &config.Server{}
	cfg.Session.Secret = "local-docker-session-secret-change-me"
	cfg.Conf.Salt = "local-docker-sweet-admin-salt-change-me"

	if err := validateSecureConfig("docker", cfg); err != nil {
		t.Fatalf("expected docker config to allow local defaults when secure config is not required: %v", err)
	}
}

func TestValidateSecureConfigRejectsProductionDefaults(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Session.Secret = "local-docker-session-secret-change-me"
	cfg.Conf.Salt = "local-docker-sweet-admin-salt-change-me"
	cfg.DBS.Primary.User = "root"
	cfg.DBS.Primary.Password = "sweet_admin"
	cfg.Redis.Password = "replace-with-redis-password"

	if err := validateSecureConfig("prod", cfg); err == nil {
		t.Fatal("expected prod config to reject default secrets")
	}
}

func TestValidateSecureConfigAcceptsStrongProductionValues(t *testing.T) {
	cfg := secureServerConfig()

	if err := validateSecureConfig("production", cfg); err != nil {
		t.Fatalf("expected production config to accept strong values: %v", err)
	}
}

func TestValidateSecureConfigRejectsProductionPlaceholders(t *testing.T) {
	cfg := secureServerConfig()
	cfg.DBS.Primary.Host = "replace-with-primary-postgres-host"

	if err := validateSecureConfig("pro", cfg); err == nil {
		t.Fatal("expected pro config to reject placeholder values")
	}
}

func TestValidateSecureConfigRejectsOSSPlaceholdersWhenEnabled(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Upload.Driver = "oss"
	cfg.Upload.OSS.Endpoint = "oss-cn-hangzhou.aliyuncs.com"
	cfg.Upload.OSS.BucketName = "sweet-admin-uploads"
	cfg.Upload.OSS.AccessKeyID = "replace-with-oss-access-key-id"
	cfg.Upload.OSS.AccessKeySecret = "StrongOSSSecret2026!"

	if err := validateSecureConfig("prod", cfg); err == nil {
		t.Fatal("expected prod oss config to reject placeholder credentials")
	}
}

func TestValidateSecureConfigRejectsProductionPublicPreview(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Upload.PublicPreview = true

	if err := validateSecureConfig("prod", cfg); err == nil {
		t.Fatal("expected prod config to reject public preview")
	}
}

func TestValidateSecureConfigRejectsProductionDangerousUploadExtensions(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Upload.AllowedExtensions = []string{".pdf", "html"}

	err := validateSecureConfig("prod", cfg)
	if err == nil {
		t.Fatal("expected prod config to reject dangerous upload extension")
	}
	if !strings.Contains(err.Error(), "upload.allowed_extensions") {
		t.Fatalf("expected upload extension error, got: %v", err)
	}
}

func TestValidateSecureConfigRejectsProductionDangerousUploadMimeTypes(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Upload.AllowedMimeTypes = []string{"application/pdf", "text/html; charset=utf-8"}

	err := validateSecureConfig("prod", cfg)
	if err == nil {
		t.Fatal("expected prod config to reject dangerous upload MIME type")
	}
	if !strings.Contains(err.Error(), "upload.allowed_mime_types") {
		t.Fatalf("expected upload MIME error, got: %v", err)
	}
}

func TestValidateSecureConfigAcceptsProductionDocumentUploadAllowList(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Upload.AllowedExtensions = []string{".pdf", "docx", ".xlsx"}
	cfg.Upload.AllowedMimeTypes = []string{
		"application/pdf",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/zip",
	}

	if err := validateSecureConfig("prod", cfg); err != nil {
		t.Fatalf("expected prod config to accept document upload allow-list: %v", err)
	}
}

func TestValidateSecureConfigRejectsProductionUnsafeUploadAccessURL(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Upload.BaseURL = "http://files.company.local/sweet_admin/files"

	err := validateSecureConfig("prod", cfg)
	if err == nil {
		t.Fatal("expected prod config to reject non-HTTPS upload base URL")
	}
	if !strings.Contains(err.Error(), "upload.base_url") {
		t.Fatalf("expected upload base URL error, got: %v", err)
	}
}

func TestValidateSecureConfigRejectsProductionUnsafeOSSAccessConfig(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Upload.Driver = "oss"
	cfg.Upload.OSS.Endpoint = "oss-cn-hangzhou.aliyuncs.com"
	cfg.Upload.OSS.BucketName = "sweet-admin"
	cfg.Upload.OSS.AccessKeyID = "LTAI5StrongKey"
	cfg.Upload.OSS.AccessKeySecret = "OSSStrongSecret2026!"
	cfg.Upload.OSS.BaseURL = "http://cdn.company.local/sweet-admin"
	cfg.Upload.OSS.BasePath = "../uploads"

	err := validateSecureConfig("prod", cfg)
	if err == nil {
		t.Fatal("expected prod config to reject unsafe OSS access config")
	}
	if !strings.Contains(err.Error(), "upload.oss.base_url") || !strings.Contains(err.Error(), "upload.oss.base_path") {
		t.Fatalf("expected OSS access config errors, got: %v", err)
	}
}

func TestValidateSecureConfigRejectsProductionWildcardCORS(t *testing.T) {
	cfg := secureServerConfig()
	cfg.Security.CORSAllowedOrigins = []string{"*"}

	if err := validateSecureConfig("prod", cfg); err == nil {
		t.Fatal("expected prod config to reject wildcard CORS origins")
	}
}

func TestApplySecurityEnvOverrides(t *testing.T) {
	t.Setenv("APP_SECURITY_CORS_ALLOWED_ORIGINS", "https://admin.company.local, https://ops.company.local")
	t.Setenv("APP_SECURITY_CORS_ALLOW_CREDENTIALS", "true")

	cfg := &config.Server{}
	applySecurityEnvOverrides(cfg)

	if !reflect.DeepEqual(cfg.Security.CORSAllowedOrigins, []string{"https://admin.company.local", "https://ops.company.local"}) {
		t.Fatalf("unexpected CORS origin override: %#v", cfg.Security.CORSAllowedOrigins)
	}
	if !cfg.Security.CORSAllowCredentials {
		t.Fatal("expected CORS credentials override to be true")
	}
}

func TestRequiresSecureConfigTreatsProAsProduction(t *testing.T) {
	t.Setenv("APP_REQUIRE_SECURE_CONFIG", "")

	if !requiresSecureConfig("pro") {
		t.Fatal("expected pro to require secure config")
	}
}

func TestParseBoolEnv(t *testing.T) {
	truthy := []string{"1", "true", "TRUE", "yes", "y", "on"}
	for _, value := range truthy {
		if !parseBoolEnv(value) {
			t.Fatalf("expected %q to be truthy", value)
		}
	}

	falsy := []string{"", "0", "false", "no", "off"}
	for _, value := range falsy {
		if parseBoolEnv(value) {
			t.Fatalf("expected %q to be falsy", value)
		}
	}
}

func TestApplyUploadListEnvOverrides(t *testing.T) {
	t.Setenv("APP_UPLOAD_ALLOWED_EXTENSIONS", ".svg, png, , .pdf")
	t.Setenv("APP_UPLOAD_ALLOWED_MIME_TYPES", "image/svg+xml, image/png")

	cfg := &config.Server{}
	cfg.Upload.AllowedExtensions = []string{".txt"}
	cfg.Upload.AllowedMimeTypes = []string{"text/plain"}
	applyUploadListEnvOverrides(cfg)

	if !reflect.DeepEqual(cfg.Upload.AllowedExtensions, []string{".svg", "png", ".pdf"}) {
		t.Fatalf("unexpected extension override: %#v", cfg.Upload.AllowedExtensions)
	}
	if !reflect.DeepEqual(cfg.Upload.AllowedMimeTypes, []string{"image/svg+xml", "image/png"}) {
		t.Fatalf("unexpected MIME override: %#v", cfg.Upload.AllowedMimeTypes)
	}
}

func TestRequiresCasbinPolicyCoverage(t *testing.T) {
	t.Setenv("APP_ENFORCE_CASBIN_POLICY_COVERAGE", "")
	if !requiresCasbinPolicyCoverage("prod") {
		t.Fatal("expected prod to require Casbin policy coverage")
	}
	if !requiresCasbinPolicyCoverage("pro") {
		t.Fatal("expected pro to require Casbin policy coverage")
	}
	if requiresCasbinPolicyCoverage("docker") {
		t.Fatal("expected docker to allow compatibility mode by default")
	}

	t.Setenv("APP_ENFORCE_CASBIN_POLICY_COVERAGE", "true")
	if !requiresCasbinPolicyCoverage("docker") {
		t.Fatal("expected explicit env to require Casbin policy coverage")
	}
}

func secureServerConfig() *config.Server {
	cfg := &config.Server{}
	cfg.DBS.Primary = config.DB{
		Host:     "postgres.internal",
		Port:     5432,
		Name:     "sweet_admin",
		User:     "auth_app",
		Password: "PrimaryDBStrong2026!",
	}
	cfg.Redis.Host = "redis.internal"
	cfg.Redis.Port = 6379
	cfg.Redis.Password = "RedisStrong2026!"
	cfg.Session.Secret = "9c9a5f58e1454b8ea8cd6073120f8970"
	cfg.Conf.Salt = "4f44786dfd3a4d5085bb585335230d0f"
	cfg.Security.CORSAllowedOrigins = []string{"https://admin.company.local"}
	cfg.Upload.Driver = "local"
	cfg.Upload.BaseURL = "/sweet_admin/files"
	return cfg
}
