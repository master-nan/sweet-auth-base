package service

import (
	"backend/config"
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/storage"
	"backend/model"
	"backend/repository"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type FileAccessPurpose string

const (
	FileAccessPurposePreview  FileAccessPurpose = "preview"
	FileAccessPurposeDownload FileAccessPurpose = "download"
)

type FileAccessResource struct {
	file model.File
}

type FileAccessReference struct {
	ID           int
	UUID         string
	CreateUserID int
}

func (r FileAccessResource) Reference() FileAccessReference {
	ref := FileAccessReference{ID: r.file.Id, UUID: r.file.FileUuid}
	if r.file.CreateUser != nil {
		ref.CreateUserID = *r.file.CreateUser
	}
	return ref
}

type FileStream struct {
	Reader  io.ReadCloser
	Headers map[string]string
}

type signedFileAccessClaims struct {
	FileUUID  string            `json:"file_uuid"`
	ExpiresAt int64             `json:"expires_at"`
	Purpose   FileAccessPurpose `json:"purpose"`
}

type FileAccessService struct {
	files   repository.FileRepository
	storage storage.Storage
	config  *config.Server
	now     func() time.Time
}

func NewFileAccessService(files repository.FileRepository, store storage.Storage, cfg *config.Server) *FileAccessService {
	return &FileAccessService{files: files, storage: store, config: cfg, now: time.Now}
}

func (s *FileAccessService) FindByUUID(ctx context.Context, uuid string) (FileAccessResource, error) {
	file, err := s.files.FindByFileUuid(ctx, uuid)
	return s.resource(file, err)
}

func (s *FileAccessService) resource(file model.File, err error) (FileAccessResource, error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return FileAccessResource{}, myerrors.ErrFileNotFound
		}
		return FileAccessResource{}, myerrors.WrapSystemError(err)
	}
	return FileAccessResource{file: file}, nil
}

func (s *FileAccessService) AuthorizeActor(actor FileAccessActor, resource FileAccessResource) error {
	if actor.IsSuperAdmin || (actor.UserID > 0 && resource.Reference().CreateUserID == actor.UserID) {
		return nil
	}
	return myerrors.ErrPermissionDenied
}

func (s *FileAccessService) PublicPreviewEnabled() bool {
	return s.config != nil && s.config.Upload.PublicPreview
}

func (s *FileAccessService) IssueSignedURL(resource FileAccessResource, purpose FileAccessPurpose, rawTTL string) (response.FileAccessURLRes, error) {
	ttl, err := parseFileAccessTTL(rawTTL, s.config.Upload)
	if err != nil {
		return response.FileAccessURLRes{}, err
	}
	expiresAt := s.currentTime().Add(ttl).Unix()
	token, err := s.sign(signedFileAccessClaims{FileUUID: resource.file.FileUuid, ExpiresAt: expiresAt, Purpose: purpose})
	if err != nil {
		return response.FileAccessURLRes{}, err
	}
	return response.FileAccessURLRes{URL: signedFileAccessURL(s.config, purpose, resource.file.FileUuid, token), ExpiresAt: expiresAt}, nil
}

func (s *FileAccessService) ResolveSigned(ctx context.Context, uuid, token string, purpose FileAccessPurpose) (FileAccessResource, error) {
	claims, err := s.verifyForPurpose(token, purpose)
	if err != nil {
		return FileAccessResource{}, err
	}
	if uuid != claims.FileUUID {
		return FileAccessResource{}, myerrors.ErrFileAccessSignatureInvalid
	}
	return s.FindByUUID(ctx, claims.FileUUID)
}

func (s *FileAccessService) Open(resource FileAccessResource, attachment bool) (FileStream, error) {
	reader, err := s.storage.Get(resource.file.FilePath)
	if err != nil {
		return FileStream{}, myerrors.ErrFileNotFound
	}
	contentType := safeContentType(resource.file.FileType)
	forceAttachment := attachment
	if !attachment {
		var safeInline bool
		contentType, safeInline = safeInlinePreviewContentType(contentType, s.config.Upload.InlinePreviewMimes)
		forceAttachment = !safeInline
	}
	disposition := "inline"
	cacheControl := "private, max-age=300"
	if forceAttachment {
		disposition = "attachment"
		cacheControl = "private, no-store"
	}
	headers := map[string]string{
		"Content-Type":           contentType,
		"Content-Length":         strconv.FormatInt(resource.file.FileSize, 10),
		"Content-Disposition":    contentDisposition(disposition, resource.file.FileName),
		"Cache-Control":          cacheControl,
		"X-Content-Type-Options": "nosniff",
	}
	if !forceAttachment {
		headers["Content-Security-Policy"] = "sandbox"
	}
	return FileStream{Reader: reader, Headers: headers}, nil
}

func (s *FileAccessService) sign(claims signedFileAccessClaims) (string, error) {
	if strings.TrimSpace(claims.FileUUID) == "" || claims.ExpiresAt <= 0 {
		return "", myerrors.ErrParamInvalid
	}
	if err := validateFileAccessPurpose(claims.Purpose); err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", myerrors.WrapSystemError(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.signPayload(encoded), nil
}

func (s *FileAccessService) verify(token string) (signedFileAccessClaims, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || !hmac.Equal([]byte(parts[1]), []byte(s.signPayload(parts[0]))) {
		return signedFileAccessClaims{}, myerrors.ErrFileAccessSignatureInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return signedFileAccessClaims{}, myerrors.ErrFileAccessSignatureInvalid
	}
	var claims signedFileAccessClaims
	if json.Unmarshal(payload, &claims) != nil || strings.TrimSpace(claims.FileUUID) == "" {
		return signedFileAccessClaims{}, myerrors.ErrFileAccessSignatureInvalid
	}
	if claims.ExpiresAt <= s.currentTime().Unix() {
		return signedFileAccessClaims{}, myerrors.ErrFileAccessSignatureExpired
	}
	if err := validateFileAccessPurpose(claims.Purpose); err != nil {
		return signedFileAccessClaims{}, err
	}
	return claims, nil
}

func (s *FileAccessService) verifyForPurpose(token string, purpose FileAccessPurpose) (signedFileAccessClaims, error) {
	claims, err := s.verify(token)
	if err != nil {
		return signedFileAccessClaims{}, err
	}
	if err := validateFileAccessPurpose(purpose); err != nil {
		return signedFileAccessClaims{}, err
	}
	if claims.Purpose != purpose {
		return signedFileAccessClaims{}, myerrors.ErrFileAccessPurposeMismatch
	}
	return claims, nil
}

func validateFileAccessPurpose(purpose FileAccessPurpose) error {
	if strings.TrimSpace(string(purpose)) == "" {
		return myerrors.ErrFileAccessPurposeMissing
	}
	if purpose != FileAccessPurposePreview && purpose != FileAccessPurposeDownload {
		return myerrors.ErrFileAccessPurposeMismatch
	}
	return nil
}

func (s *FileAccessService) signPayload(payload string) string {
	h := hmac.New(sha256.New, []byte(s.config.Session.Secret))
	_, _ = h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (s *FileAccessService) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func safeContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil || strings.TrimSpace(mediaType) == "" || strings.ContainsAny(mediaType, "\r\n") {
		return "application/octet-stream"
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}

func safeInlinePreviewContentType(contentType string, allowed []string) (string, bool) {
	contentType = safeContentType(contentType)
	for _, item := range allowed {
		if contentType == safeContentType(item) {
			return contentType, true
		}
	}
	return "application/octet-stream", false
}

func contentDisposition(disposition, fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = "download"
	}
	return fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, url.PathEscape(fileName))
}

func parseFileAccessTTL(raw string, upload config.Upload) (time.Duration, error) {
	if upload.AccessTTLMinutes <= 0 {
		return 0, myerrors.NewValidationError("文件访问有效期配置错误")
	}
	maxMinutes := upload.MaxAccessTTLMinutes
	if maxMinutes <= 0 {
		maxMinutes = upload.AccessTTLMinutes
	}
	defaultTTL := time.Duration(upload.AccessTTLMinutes) * time.Minute
	seconds, err := strconv.Atoi(raw)
	if raw == "" || err != nil || seconds <= 0 {
		return defaultTTL, nil
	}
	ttl := time.Duration(seconds) * time.Second
	maxTTL := time.Duration(maxMinutes) * time.Minute
	if ttl > maxTTL {
		return maxTTL, nil
	}
	return ttl, nil
}

func signedFileAccessURL(cfg *config.Server, purpose FileAccessPurpose, uuid, token string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.Upload.BaseURL), "/")
	return fmt.Sprintf("%s/access/%s/%s?token=%s", baseURL, purpose, url.PathEscape(uuid), url.QueryEscape(token))
}
