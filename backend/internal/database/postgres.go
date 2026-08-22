package database

import (
	"backend/config"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	PostgresTLSDisable    = "disable"
	PostgresTLSRequire    = "require"
	PostgresTLSVerifyCA   = "verify-ca"
	PostgresTLSVerifyFull = "verify-full"
)

// PostgresDSN 是构造PostgreSQL连接参数的唯一边界；URL编码可防止配置值注入额外驱动参数。
func PostgresDSN(cfg config.DB) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.TLS.Mode))
	if !SupportedPostgresTLSMode(mode) {
		return "", fmt.Errorf("unsupported PostgreSQL TLS mode %q", cfg.TLS.Mode)
	}
	if (strings.TrimSpace(cfg.TLS.CertFile) == "") != (strings.TrimSpace(cfg.TLS.KeyFile) == "") {
		return "", fmt.Errorf("PostgreSQL TLS client certificate and key must be configured together")
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Path:   "/" + cfg.Name,
	}
	query := dsn.Query()
	query.Set("sslmode", mode)
	query.Set("TimeZone", "Asia/Shanghai")
	query.Set("connect_timeout", "10")
	if value := strings.TrimSpace(cfg.TLS.RootCAFile); value != "" {
		query.Set("sslrootcert", value)
	}
	if value := strings.TrimSpace(cfg.TLS.CertFile); value != "" {
		query.Set("sslcert", value)
	}
	if value := strings.TrimSpace(cfg.TLS.KeyFile); value != "" {
		query.Set("sslkey", value)
	}
	dsn.RawQuery = query.Encode()
	return dsn.String(), nil
}

func SupportedPostgresTLSMode(mode string) bool {
	switch mode {
	case PostgresTLSDisable, PostgresTLSRequire, PostgresTLSVerifyCA, PostgresTLSVerifyFull:
		return true
	default:
		return false
	}
}
