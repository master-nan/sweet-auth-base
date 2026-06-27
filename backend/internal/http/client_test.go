package http

import (
	"strings"
	"testing"
)

func TestSanitizeRequestURLRedactsSensitiveQueryValues(t *testing.T) {
	rawURL := "https://example.com/api?access_token=abc123&app_secret=secret-value&password=pwd&name=demo"

	sanitized := sanitizeRequestURL(rawURL)

	for _, leaked := range []string{"abc123", "secret-value", "pwd"} {
		if strings.Contains(sanitized, leaked) {
			t.Fatalf("sanitized URL leaked %q: %s", leaked, sanitized)
		}
	}
	if !strings.Contains(sanitized, "name=demo") {
		t.Fatalf("expected non-sensitive query value to remain, got %s", sanitized)
	}
	if !strings.Contains(sanitized, "access_token=%2A%2A%2A") {
		t.Fatalf("expected access token to be redacted, got %s", sanitized)
	}
}

func TestSanitizeRequestURLHandlesInvalidURL(t *testing.T) {
	if got := sanitizeRequestURL("http://[::1"); got != "<invalid-url>" {
		t.Fatalf("expected invalid URL marker, got %s", got)
	}
}
