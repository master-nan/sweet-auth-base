package service

import (
	"backend/config"
	appErrors "backend/internal/errors"
	"backend/model"
	"crypto/md5"
	"fmt"
	"strings"
	"testing"
)

type testMultipartFile struct {
	*strings.Reader
}

func (t testMultipartFile) Close() error {
	return nil
}

func TestValidateUploadExtensionRejectsExecutableExtensions(t *testing.T) {
	cfg := config.Upload{AllowedExtensions: []string{".pdf"}}

	if err := validateUploadExtension("setup.sh", cfg); err == nil {
		t.Fatal("expected shell script extension to be rejected")
	}
	if err := validateUploadExtension("report.pdf", cfg); err != nil {
		t.Fatalf("expected pdf extension to be allowed: %v", err)
	}
}

func TestValidateUploadMimeTypeRejectsExecutableContent(t *testing.T) {
	cfg := config.Upload{AllowedMimeTypes: []string{"text/plain"}}

	if err := validateUploadMimeType("text/x-shellscript", cfg); err == nil {
		t.Fatal("expected shell script MIME to be rejected")
	}
	if err := validateUploadMimeType("text/plain; charset=utf-8", cfg); err != nil {
		t.Fatalf("expected text/plain MIME to be allowed: %v", err)
	}
}

func TestValidateUploadMimeTypeRejectsDetectedHTMLContent(t *testing.T) {
	cfg := config.Upload{AllowedMimeTypes: []string{"application/pdf", "text/plain"}}
	file := testMultipartFile{Reader: strings.NewReader("<!doctype html><script>alert(1)</script>")}

	contentType, err := detectUploadContentType(file)
	if err != nil {
		t.Fatalf("detect upload content type: %v", err)
	}
	if contentType != "text/html; charset=utf-8" {
		t.Fatalf("unexpected detected content type: %s", contentType)
	}
	if err := validateUploadMimeType(contentType, cfg); err == nil {
		t.Fatal("expected detected HTML content to be rejected")
	}
}

func TestValidateUploadSizeRejectsEmptyAndOversizedFiles(t *testing.T) {
	cfg := config.Upload{MaxSize: 1}

	if err := validateUploadSize(0, cfg); err == nil {
		t.Fatal("expected empty file to be rejected")
	}
	if err := validateUploadSize(2<<20, cfg); err == nil {
		t.Fatal("expected oversized file to be rejected")
	}
	if err := validateUploadSize(512, cfg); err != nil {
		t.Fatalf("expected small file to be allowed: %v", err)
	}
}

func TestDetectUploadContentTypeResetsReader(t *testing.T) {
	file := testMultipartFile{Reader: strings.NewReader("%PDF-1.7\ncontent")}

	contentType, err := detectUploadContentType(file)
	if err != nil {
		t.Fatalf("detect upload content type: %v", err)
	}
	if contentType != "application/pdf" {
		t.Fatalf("unexpected content type: %s", contentType)
	}

	buf := make([]byte, 5)
	if _, err := file.Read(buf); err != nil {
		t.Fatalf("read after content detection: %v", err)
	}
	if string(buf) != "%PDF-" {
		t.Fatalf("reader was not reset, got %q", string(buf))
	}
}

func TestUploadAllowListCanBeConfigured(t *testing.T) {
	cfg := config.Upload{
		AllowedExtensions: []string{"svg"},
		AllowedMimeTypes:  []string{"image/svg+xml"},
	}

	if err := validateUploadExtension("icon.svg", cfg); err != nil {
		t.Fatalf("expected configured svg extension to be allowed: %v", err)
	}
	if err := validateUploadMimeType("image/svg+xml", cfg); err != nil {
		t.Fatalf("expected configured svg MIME to be allowed: %v", err)
	}
	if err := validateUploadExtension("report.pdf", cfg); err == nil {
		t.Fatal("expected default extension to be replaced by configured allow-list")
	}
}

func TestFileAccessURLUsesConfiguredPublicBase(t *testing.T) {
	svc := &FileService{config: &config.Server{}}
	svc.config.Upload.BaseURL = "/sweet_admin/files/"

	if got := svc.fileAccessURL("file-uuid"); got != "/sweet_admin/files/file-uuid" {
		t.Fatalf("unexpected file access URL: %s", got)
	}
}

func TestEnsureFileReuseAccessRequiresOwnerOrSuperAdmin(t *testing.T) {
	ownerID := 101
	file := model.File{}
	file.CreateUser = &ownerID

	owner := FileAccessActor{UserID: ownerID}
	if err := ensureFileReuseAccess(owner, file); err != nil {
		t.Fatalf("owner should reuse file: %v", err)
	}

	admin := FileAccessActor{UserID: 202, IsSuperAdmin: true}
	if err := ensureFileReuseAccess(admin, file); err != nil {
		t.Fatalf("super admin should reuse file: %v", err)
	}

	other := FileAccessActor{UserID: 303}
	if err := ensureFileReuseAccess(other, file); err != appErrors.ErrPermissionDenied {
		t.Fatalf("expected other user to be denied file reuse, got %v", err)
	}
}

func TestValidateChunkUploadSizeRejectsInvalidChunkSizes(t *testing.T) {
	chunk := model.FileChunk{FileSize: 11, ChunkSize: 5, ChunkIndex: 2}

	if err := validateChunkUploadSize(chunk, 1); err != nil {
		t.Fatalf("expected final one-byte chunk to pass: %v", err)
	}
	if err := validateChunkUploadSize(chunk, 2); err == nil {
		t.Fatal("expected oversized final chunk to be rejected")
	}
	if err := validateChunkUploadSize(chunk, 0); err == nil {
		t.Fatal("expected empty chunk to be rejected")
	}
}

func TestValidateMergedChunkFileChecksSizeAndMD5(t *testing.T) {
	sum := md5.Sum([]byte("abc"))
	md5Text := fmt.Sprintf("%x", sum)
	chunk := model.FileChunk{FileSize: 3, FileMd5: md5Text}

	if err := validateMergedChunkFile(chunk, 3, md5Text); err != nil {
		t.Fatalf("expected matching merged file to pass: %v", err)
	}
	if err := validateMergedChunkFile(chunk, 2, md5Text); err == nil {
		t.Fatal("expected merged size mismatch to be rejected")
	}
	if err := validateMergedChunkFile(chunk, 3, "bad-md5"); err == nil {
		t.Fatal("expected merged md5 mismatch to be rejected")
	}
}
