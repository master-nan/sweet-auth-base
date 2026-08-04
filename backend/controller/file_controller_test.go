package controller

import (
	"backend/config"
	myerrors "backend/internal/errors"
	"encoding/base64"
	"encoding/json"
	"errors"
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
		Purpose:   fileAccessPurposePreview,
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	claims, err := controller.verifyFileAccessToken(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.FileUuid != "file-uuid" || claims.ExpiresAt != expiresAt || claims.Purpose != fileAccessPurposePreview {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestSignedFileAccessTokenIsBoundToPurpose(t *testing.T) {
	controller := newFileAccessTestController()
	expiresAt := time.Now().Add(time.Minute).Unix()

	previewToken, err := controller.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  "file-uuid",
		ExpiresAt: expiresAt,
		Purpose:   fileAccessPurposePreview,
	})
	if err != nil {
		t.Fatalf("sign preview token: %v", err)
	}
	if _, err := controller.verifyFileAccessTokenForPurpose(previewToken, fileAccessPurposePreview); err != nil {
		t.Fatalf("preview token should access preview: %v", err)
	}
	if _, err := controller.verifyFileAccessTokenForPurpose(previewToken, fileAccessPurposeDownload); !errors.Is(err, myerrors.ErrFileAccessPurposeMismatch) {
		t.Fatalf("preview token should not access download, got %v", err)
	}

	downloadToken, err := controller.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  "file-uuid",
		ExpiresAt: expiresAt,
		Purpose:   fileAccessPurposeDownload,
	})
	if err != nil {
		t.Fatalf("sign download token: %v", err)
	}
	if _, err := controller.verifyFileAccessTokenForPurpose(downloadToken, fileAccessPurposeDownload); err != nil {
		t.Fatalf("download token should access download: %v", err)
	}
	if _, err := controller.verifyFileAccessTokenForPurpose(downloadToken, fileAccessPurposePreview); !errors.Is(err, myerrors.ErrFileAccessPurposeMismatch) {
		t.Fatalf("download token should not access preview, got %v", err)
	}
}

func TestSignedFileAccessTokenRejectsTamperingAndExpiredTokens(t *testing.T) {
	controller := newFileAccessTestController()

	token, err := controller.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  "file-uuid",
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		Purpose:   fileAccessPurposePreview,
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
		Purpose:   fileAccessPurposePreview,
	})
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	if _, err := controller.verifyFileAccessToken(expiredToken); !errors.Is(err, myerrors.ErrFileAccessSignatureExpired) {
		t.Fatalf("expected expired token error, got %v", err)
	}
}

func TestSignedFileAccessTokenRejectsPurposeTamperingAndMissingPurpose(t *testing.T) {
	controller := newFileAccessTestController()
	expiresAt := time.Now().Add(time.Minute).Unix()
	token, err := controller.signFileAccessToken(signedFileAccessClaims{
		FileUuid:  "file-uuid",
		ExpiresAt: expiresAt,
		Purpose:   fileAccessPurposePreview,
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	parts := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	var claims signedFileAccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal token payload: %v", err)
	}
	claims.Purpose = fileAccessPurposeDownload
	tamperedPayload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal tampered claims: %v", err)
	}
	tamperedToken := base64.RawURLEncoding.EncodeToString(tamperedPayload) + "." + parts[1]
	if _, err := controller.verifyFileAccessToken(tamperedToken); !errors.Is(err, myerrors.ErrFileAccessSignatureInvalid) {
		t.Fatalf("expected purpose tampering to invalidate signature, got %v", err)
	}

	legacyPayload, err := json.Marshal(struct {
		FileUuid  string `json:"file_uuid"`
		ExpiresAt int64  `json:"expires_at"`
	}{
		FileUuid:  "file-uuid",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("marshal legacy claims: %v", err)
	}
	encodedLegacyPayload := base64.RawURLEncoding.EncodeToString(legacyPayload)
	legacyToken := encodedLegacyPayload + "." + controller.signFileAccessPayload(encodedLegacyPayload)
	if _, err := controller.verifyFileAccessToken(legacyToken); !errors.Is(err, myerrors.ErrFileAccessPurposeMissing) {
		t.Fatalf("expected token without purpose to be rejected, got %v", err)
	}
}

func TestSignedFileAccessTokenRejectsMissingOrUnsupportedPurposeWhenSigning(t *testing.T) {
	controller := newFileAccessTestController()
	claims := signedFileAccessClaims{FileUuid: "file-uuid", ExpiresAt: time.Now().Add(time.Minute).Unix()}
	if _, err := controller.signFileAccessToken(claims); !errors.Is(err, myerrors.ErrFileAccessPurposeMissing) {
		t.Fatalf("expected missing purpose error, got %v", err)
	}
	claims.Purpose = "share"
	if _, err := controller.signFileAccessToken(claims); !errors.Is(err, myerrors.ErrFileAccessPurposeMismatch) {
		t.Fatalf("expected unsupported purpose error, got %v", err)
	}
}

func newFileAccessTestController() *FileController {
	controller := &FileController{config: &config.Server{}}
	controller.config.Session.Secret = "test-file-access-secret"
	return controller
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
