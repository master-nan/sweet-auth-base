package service

import (
	"backend/config"
	myerrors "backend/internal/errors"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"backend/model"
)

func newFileAccessTestService() *FileAccessService {
	cfg := &config.Server{}
	cfg.Session.Secret = "test-file-access-secret"
	cfg.Upload.AccessTTLMinutes = 15
	cfg.Upload.MaxAccessTTLMinutes = 60
	cfg.Upload.BaseURL = "/sweet_admin/files/"
	return &FileAccessService{config: cfg, now: time.Now}
}

func TestSignedFileAccessTokenRoundTripAndPurposeIsolation(t *testing.T) {
	service := newFileAccessTestService()
	claims := signedFileAccessClaims{FileUUID: "file-uuid", ExpiresAt: time.Now().Add(time.Minute).Unix(), Purpose: FileAccessPurposePreview}
	token, err := service.sign(claims)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	got, err := service.verifyForPurpose(token, FileAccessPurposePreview)
	if err != nil || got != claims {
		t.Fatalf("verify token: got=%+v err=%v", got, err)
	}
	if _, err = service.verifyForPurpose(token, FileAccessPurposeDownload); !errors.Is(err, myerrors.ErrFileAccessPurposeMismatch) {
		t.Fatalf("preview token must not download: %v", err)
	}
}

func TestSignedFileAccessRejectsTamperExpiryAndLegacyPurpose(t *testing.T) {
	service := newFileAccessTestService()
	token, err := service.sign(signedFileAccessClaims{FileUUID: "file-uuid", ExpiresAt: time.Now().Add(time.Minute).Unix(), Purpose: FileAccessPurposeDownload})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.verify(token + "tampered"); !errors.Is(err, myerrors.ErrFileAccessSignatureInvalid) {
		t.Fatalf("expected tamper rejection, got %v", err)
	}
	expired, err := service.sign(signedFileAccessClaims{FileUUID: "file-uuid", ExpiresAt: time.Now().Add(-time.Minute).Unix(), Purpose: FileAccessPurposePreview})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.verify(expired); !errors.Is(err, myerrors.ErrFileAccessSignatureExpired) {
		t.Fatalf("expected expiry rejection, got %v", err)
	}
	legacyPayload, _ := json.Marshal(struct {
		FileUUID  string `json:"file_uuid"`
		ExpiresAt int64  `json:"expires_at"`
	}{"file-uuid", time.Now().Add(time.Minute).Unix()})
	encoded := base64.RawURLEncoding.EncodeToString(legacyPayload)
	if _, err = service.verify(encoded + "." + service.signPayload(encoded)); !errors.Is(err, myerrors.ErrFileAccessPurposeMissing) {
		t.Fatalf("expected legacy purpose rejection, got %v", err)
	}
}

func TestFileAccessPurposeValidationAndTTL(t *testing.T) {
	service := newFileAccessTestService()
	claims := signedFileAccessClaims{FileUUID: "file-uuid", ExpiresAt: time.Now().Add(time.Minute).Unix()}
	if _, err := service.sign(claims); !errors.Is(err, myerrors.ErrFileAccessPurposeMissing) {
		t.Fatalf("expected missing purpose error, got %v", err)
	}
	claims.Purpose = "share"
	if _, err := service.sign(claims); !errors.Is(err, myerrors.ErrFileAccessPurposeMismatch) {
		t.Fatalf("expected unsupported purpose error, got %v", err)
	}
	if got, err := parseFileAccessTTL("7200", service.config.Upload); err != nil || got != time.Hour {
		t.Fatalf("expected capped ttl, got %v err=%v", got, err)
	}
	if _, err := parseFileAccessTTL("", config.Upload{}); err == nil {
		t.Fatal("expected invalid ttl configuration")
	}
	accessURL := signedFileAccessURL(service.config, FileAccessPurposePreview, "file/uuid", "token.with/slash")
	parsed, err := url.Parse(accessURL)
	if err != nil || parsed.Query().Get("token") != "token.with/slash" || parsed.EscapedPath() != "/sweet_admin/files/access/preview/file%2Fuuid" {
		t.Fatalf("unexpected signed URL: %s err=%v", accessURL, err)
	}
}

func TestFileResponseContentSafety(t *testing.T) {
	if got := safeContentType("text/plain\r\nX-Evil: 1"); got != "application/octet-stream" {
		t.Fatalf("unsafe content type accepted: %s", got)
	}
	allowed := []string{"image/png", "application/pdf", "text/plain"}
	if got, ok := safeInlinePreviewContentType("application/pdf", allowed); !ok || got != "application/pdf" {
		t.Fatalf("expected pdf inline, got %s ok=%v", got, ok)
	}
	if got, ok := safeInlinePreviewContentType("text/html", allowed); ok || !strings.Contains(got, "octet-stream") {
		t.Fatalf("expected html forced download, got %s ok=%v", got, ok)
	}
}

func TestFileAccessOpenAppliesPreviewAndDownloadHeaders(t *testing.T) {
	service := newFileAccessTestService()
	service.config.Upload.InlinePreviewMimes = []string{"text/plain"}
	store := newMemoryFileStorage()
	store.files["object"] = []byte("hello")
	service.storage = store
	ownerID := 10
	file := model.File{FilePath: "object", FileName: "hello.txt", FileType: "text/plain", FileSize: 5}
	file.CreateUser = &ownerID
	resource := FileAccessResource{file: file}
	if err := service.AuthorizeActor(FileAccessActor{UserID: ownerID}, resource); err != nil {
		t.Fatalf("owner access: %v", err)
	}
	if err := service.AuthorizeActor(FileAccessActor{UserID: 20}, resource); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("other user must be denied: %v", err)
	}
	preview, err := service.Open(resource, false)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(preview.Reader)
	_ = preview.Reader.Close()
	if string(body) != "hello" || !strings.HasPrefix(preview.Headers["Content-Disposition"], "inline") || preview.Headers["Content-Security-Policy"] != "sandbox" {
		t.Fatalf("unexpected preview: body=%q headers=%v", body, preview.Headers)
	}
	download, err := service.Open(resource, true)
	if err != nil {
		t.Fatal(err)
	}
	_ = download.Reader.Close()
	if !strings.HasPrefix(download.Headers["Content-Disposition"], "attachment") || download.Headers["Cache-Control"] != "private, no-store" {
		t.Fatalf("unexpected download headers: %v", download.Headers)
	}
}
