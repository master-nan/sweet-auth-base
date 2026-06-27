package controller

import (
	"backend/config"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestContentDispositionEscapesFileName(t *testing.T) {
	header := contentDisposition("attachment", "测试 file \"name\".txt")

	if !strings.HasPrefix(header, "attachment; filename*=UTF-8''") {
		t.Fatalf("unexpected content disposition: %s", header)
	}
	encodedName := strings.TrimPrefix(header, "attachment; filename*=UTF-8''")
	if strings.Contains(encodedName, "\"") || strings.Contains(encodedName, " ") {
		t.Fatalf("content disposition should escape filename quotes and spaces: %s", header)
	}
}

func TestSafeContentTypeDefaultsToOctetStream(t *testing.T) {
	if got := safeContentType(" "); got != "application/octet-stream" {
		t.Fatalf("unexpected default content type: %s", got)
	}
	if got := safeContentType("text/plain; charset=utf-8"); got != "text/plain" {
		t.Fatalf("unexpected content type: %s", got)
	}
	if got := safeContentType("text/plain\r\nX-Evil: 1"); got != "application/octet-stream" {
		t.Fatalf("unexpected unsafe content type: %s", got)
	}
}

func TestSafeInlinePreviewContentTypeAllowList(t *testing.T) {
	allowList := []string{"image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf", "text/plain", "text/csv"}
	for _, contentType := range allowList {
		if got, ok := safeInlinePreviewContentType(contentType, allowList); !ok || got != contentType {
			t.Fatalf("expected %s to be safe inline, got %s ok=%v", contentType, got, ok)
		}
	}

	for _, contentType := range []string{"image/svg+xml", "text/html", "application/javascript", "application/vnd.ms-excel"} {
		if got, ok := safeInlinePreviewContentType(contentType, allowList); ok || got != "application/octet-stream" {
			t.Fatalf("expected %s to be forced download, got %s ok=%v", contentType, got, ok)
		}
	}
}

func TestSignedFileAccessTokenRoundTrip(t *testing.T) {
	controller := &FileController{config: &config.Server{}}
	controller.config.Session.Secret = "test-file-access-secret"
	expiresAt := time.Now().Add(time.Minute).Unix()

	token, err := controller.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  "file-uuid",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	claims, err := controller.verifyFileAccessToken(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.FileUuid != "file-uuid" || claims.ExpiresAt != expiresAt {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestSignedFileAccessTokenRejectsTamperingAndExpiredTokens(t *testing.T) {
	controller := &FileController{config: &config.Server{}}
	controller.config.Session.Secret = "test-file-access-secret"

	token, err := controller.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  "file-uuid",
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if _, err := controller.verifyFileAccessToken(token + "tampered"); err == nil {
		t.Fatal("expected tampered token to be rejected")
	}

	expiredToken, err := controller.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  "file-uuid",
		ExpiresAt: time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	if _, err := controller.verifyFileAccessToken(expiredToken); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestFileAccessTTLAndBaseURL(t *testing.T) {
	upload := config.Upload{AccessTTLMinutes: 15, MaxAccessTTLMinutes: 60}
	if got, err := parseFileAccessTTL("7200", upload); err != nil || got != time.Hour {
		t.Fatalf("expected ttl to be capped at %s, got %s", time.Hour, got)
	}
	if got, err := parseFileAccessTTL("bad", upload); err != nil || got != 15*time.Minute {
		t.Fatalf("expected invalid ttl to use default %s, got %s", 15*time.Minute, got)
	}
	if _, err := parseFileAccessTTL("", config.Upload{}); err == nil {
		t.Fatal("expected invalid ttl config to be rejected")
	}

	cfg := &config.Server{}
	cfg.Upload.BaseURL = "/sweet_admin/files/"
	if got := fileAccessBaseURL(cfg); got != "/sweet_admin/files" {
		t.Fatalf("unexpected access base url: %s", got)
	}

	accessURL := signedFileAccessURL(cfg, "preview", "file/uuid", "token.with/slash")
	parsed, err := url.Parse(accessURL)
	if err != nil {
		t.Fatalf("parse signed access url: %v", err)
	}
	if parsed.EscapedPath() != "/sweet_admin/files/access/preview/file%2Fuuid" {
		t.Fatalf("expected escaped file uuid in path, got %s", parsed.EscapedPath())
	}
	if got := parsed.Query().Get("token"); got != "token.with/slash" {
		t.Fatalf("expected token in query, got %s", got)
	}
}
