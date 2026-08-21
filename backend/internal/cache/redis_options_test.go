package cache

import (
	"backend/config"
	"testing"
)

func TestRedisOptionsEnableVerifiedTLS(t *testing.T) {
	options, err := RedisOptions(config.Redis{
		Host: "redis.internal", Port: 6380,
		TLS: config.RedisTLS{Enabled: true, ServerName: "redis.internal"},
	})
	if err != nil {
		t.Fatalf("RedisOptions: %v", err)
	}
	if options.TLSConfig == nil || options.TLSConfig.ServerName != "redis.internal" || options.TLSConfig.InsecureSkipVerify {
		t.Fatalf("unexpected TLS config: %#v", options.TLSConfig)
	}
}

func TestRedisOptionsRejectsIncompleteClientIdentity(t *testing.T) {
	_, err := RedisOptions(config.Redis{TLS: config.RedisTLS{Enabled: true, CertFile: "client.pem"}})
	if err == nil {
		t.Fatal("expected incomplete client identity to fail")
	}
}
