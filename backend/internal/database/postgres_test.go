package database

import (
	"backend/config"
	"net/url"
	"testing"
)

func TestPostgresDSNEncodesConfigurationAndTLS(t *testing.T) {
	dsn, err := PostgresDSN(config.DB{
		Host: "db.internal", Port: 5432, Name: "sweet admin", User: "app@user", Password: "p&ss=word",
		TLS: config.PostgresTLS{Mode: PostgresTLSVerifyFull, RootCAFile: "/run/secrets/ca.pem"},
	})
	if err != nil {
		t.Fatalf("PostgresDSN: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	password, _ := parsed.User.Password()
	if parsed.User.Username() != "app@user" || password != "p&ss=word" {
		t.Fatalf("credentials were not URL encoded safely: %s", dsn)
	}
	if parsed.Query().Get("sslmode") != PostgresTLSVerifyFull || parsed.Query().Get("sslrootcert") != "/run/secrets/ca.pem" {
		t.Fatalf("unexpected TLS query: %s", parsed.RawQuery)
	}
}

func TestPostgresDSNRejectsUnsupportedTLSAndIncompleteClientIdentity(t *testing.T) {
	base := config.DB{Host: "db", Port: 5432, Name: "app", User: "app", Password: "secret"}
	base.TLS.Mode = "prefer"
	if _, err := PostgresDSN(base); err == nil {
		t.Fatal("expected unsupported TLS mode to fail")
	}
	base.TLS = config.PostgresTLS{Mode: PostgresTLSRequire, CertFile: "client.pem"}
	if _, err := PostgresDSN(base); err == nil {
		t.Fatal("expected incomplete client identity to fail")
	}
}
