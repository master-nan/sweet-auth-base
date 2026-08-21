package cache

import (
	"backend/config"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisOptions is the shared runtime, migration, and preflight Redis
// connection contract. Certificate verification is always enabled when TLS is
// configured.
func RedisOptions(cfg config.Redis) (*redis.Options, error) {
	options := &redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
	}
	if !cfg.TLS.Enabled {
		return options, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: strings.TrimSpace(cfg.TLS.ServerName),
	}
	if caFile := strings.TrimSpace(cfg.TLS.CAFile); caFile != "" {
		contents, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read Redis TLS CA file: %w", err)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(contents) {
			return nil, fmt.Errorf("Redis TLS CA file contains no valid certificates")
		}
		tlsConfig.RootCAs = roots
	}
	certFile := strings.TrimSpace(cfg.TLS.CertFile)
	keyFile := strings.TrimSpace(cfg.TLS.KeyFile)
	if (certFile == "") != (keyFile == "") {
		return nil, fmt.Errorf("Redis TLS client certificate and key must be configured together")
	}
	if certFile != "" {
		certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load Redis TLS client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	options.TLSConfig = tlsConfig
	return options, nil
}
