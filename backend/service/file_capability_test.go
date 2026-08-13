package service

import (
	"backend/config"
	"backend/internal/audit"
	"backend/internal/database"
	myerrors "backend/internal/errors"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"testing"
)

type memoryFileStorage struct {
	mu        sync.Mutex
	files     map[string][]byte
	saveCount int
	deleteErr error
	saveErr   error
}

func newMemoryFileStorage() *memoryFileStorage {
	return &memoryFileStorage{files: make(map[string][]byte)}
}

func (s *memoryFileStorage) Save(path string, reader io.Reader, _ string) (string, error) {
	if s.saveErr != nil {
		return "", s.saveErr
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[path] = data
	s.saveCount++
	return "/files/" + path, nil
}

func (s *memoryFileStorage) Delete(path string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, path)
	return nil
}

func (s *memoryFileStorage) Get(path string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.files[path]
	if !ok {
		return nil, errors.New("missing object")
	}
	return io.NopCloser(bytes.NewReader(append([]byte(nil), data...))), nil
}

func (s *memoryFileStorage) GetURL(path string) string { return "/files/" + path }
func (s *memoryFileStorage) Type() string              { return "memory" }

type fileCapabilityHarness struct {
	db       *database.PrimaryDB
	upload   *FileUploadService
	metadata *FileMetadataService
	store    *memoryFileStorage
}

func newFileCapabilityHarness(t *testing.T) fileCapabilityHarness {
	t.Helper()
	db := testutil.OpenSQLite(t, &model.File{}, &model.FileChunk{})
	primary := &database.PrimaryDB{DB: db}
	files := impl.NewFileRepositoryImpl(primary)
	chunks := impl.NewFileChunkRepositoryImpl(primary)
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Server{}
	cfg.Upload.Dir = t.TempDir()
	cfg.Upload.BaseURL = "/files"
	cfg.Upload.MaxSize = 10
	cfg.Upload.ChunkSize = 1
	cfg.Upload.AllowedExtensions = []string{".txt"}
	cfg.Upload.AllowedMimeTypes = []string{"text/plain"}
	store := newMemoryFileStorage()
	return fileCapabilityHarness{
		db:       primary,
		upload:   NewFileUploadService(files, chunks, sf, cfg, store),
		metadata: NewFileMetadataService(files, store),
		store:    store,
	}
}

func multipartHeader(t *testing.T, name string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, "/", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err = req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = req.MultipartForm.RemoveAll() })
	return req.MultipartForm.File["file"][0]
}

func fileTestContext(userID int) context.Context {
	return audit.WithAuditSubject(context.Background(), audit.NewAuditSubject(userID, "file-test"))
}

func TestFileUploadPersistsWhitelistDTOAndCompensatesMetadataFailure(t *testing.T) {
	h := newFileCapabilityHarness(t)
	result, err := h.upload.UploadResponse(fileTestContext(10), FileAccessActor{UserID: 10}, multipartHeader(t, "note.txt", []byte("hello")))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if result.Id == 0 || result.FileName != "note.txt" || result.FileUuid == "" {
		t.Fatalf("unexpected upload response: %+v", result)
	}
	if h.store.saveCount != 1 || len(h.store.files) != 1 {
		t.Fatalf("expected one stored object, saves=%d objects=%d", h.store.saveCount, len(h.store.files))
	}
	var persisted model.File
	if err = h.db.DB.First(&persisted, result.Id).Error; err != nil || persisted.CreateUser == nil || *persisted.CreateUser != 10 {
		t.Fatalf("AuditSubject must reach file metadata: user=%v err=%v", persisted.CreateUser, err)
	}

	if err = h.db.DB.Exec("CREATE TRIGGER reject_file_insert BEFORE INSERT ON file BEGIN SELECT RAISE(FAIL, 'reject'); END").Error; err != nil {
		t.Fatal(err)
	}
	_, err = h.upload.Upload(fileTestContext(11), FileAccessActor{UserID: 11}, multipartHeader(t, "other.txt", []byte("different")))
	if err == nil {
		t.Fatal("expected metadata failure")
	}
	if len(h.store.files) != 1 {
		t.Fatalf("metadata failure must remove newly stored object, objects=%d", len(h.store.files))
	}
}

func TestFileUploadStorageFailureDoesNotCreateMetadata(t *testing.T) {
	h := newFileCapabilityHarness(t)
	h.store.saveErr = errors.New("storage down")
	if _, err := h.upload.Upload(fileTestContext(10), FileAccessActor{UserID: 10}, multipartHeader(t, "note.txt", []byte("hello"))); err == nil {
		t.Fatal("expected storage failure")
	}
	var count int64
	if err := h.db.DB.Model(&model.File{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("metadata must not be created: count=%d err=%v", count, err)
	}
}

func TestFileFastUploadRequiresOwnerAndIntactStorage(t *testing.T) {
	h := newFileCapabilityHarness(t)
	file, err := h.upload.Upload(fileTestContext(10), FileAccessActor{UserID: 10}, multipartHeader(t, "note.txt", []byte("hello")))
	if err != nil {
		t.Fatal(err)
	}
	if reused, err := h.upload.Upload(fileTestContext(10), FileAccessActor{UserID: 10}, multipartHeader(t, "copy.txt", []byte("hello"))); err != nil || reused.Id != file.Id {
		t.Fatalf("owner fast upload failed: file=%+v err=%v", reused, err)
	}
	if _, err := h.upload.Upload(fileTestContext(20), FileAccessActor{UserID: 20}, multipartHeader(t, "other.txt", []byte("hello"))); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("cross-user fast upload must fail, got %v", err)
	}
	h.store.mu.Lock()
	delete(h.store.files, file.FilePath)
	h.store.mu.Unlock()
	if _, err := h.upload.Upload(fileTestContext(10), FileAccessActor{UserID: 10}, multipartHeader(t, "missing.txt", []byte("hello"))); !errors.Is(err, myerrors.ErrFileNotFound) {
		t.Fatalf("missing storage object must fail closed, got %v", err)
	}
}

func TestChunkSessionOwnershipAndConcurrentMerge(t *testing.T) {
	h := newFileCapabilityHarness(t)
	ownerID := 10
	ctx := fileTestContext(ownerID)
	chunks := []model.FileChunk{
		{UploadId: "upload-1", FileName: "note.txt", FileSize: 6, ChunkSize: 3, ChunkCount: 2, ChunkIndex: 0, FileType: "text/plain", FileExt: ".txt"},
		{UploadId: "upload-1", FileName: "note.txt", FileSize: 6, ChunkSize: 3, ChunkCount: 2, ChunkIndex: 1, FileType: "text/plain", FileExt: ".txt"},
	}
	for i := range chunks {
		chunks[i].Id = 100 + i
		chunks[i].CreateUser = &ownerID
	}
	if err := h.db.DB.WithContext(ctx).Create(&chunks).Error; err != nil {
		t.Fatal(err)
	}
	other := FileAccessActor{UserID: 20}
	if err := h.upload.UploadChunk(ctx, other, "upload-1", 0, multipartHeader(t, "chunk", []byte("abc"))); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("cross-user chunk upload must fail, got %v", err)
	}
	if _, err := h.upload.GetUploadProgressForUser(ctx, other, "upload-1"); !errors.Is(err, myerrors.ErrPermissionDenied) {
		t.Fatalf("cross-user progress must fail, got %v", err)
	}
	owner := FileAccessActor{UserID: ownerID}
	if err := h.upload.UploadChunk(ctx, owner, "upload-1", 0, multipartHeader(t, "chunk", []byte("abc"))); err != nil {
		t.Fatal(err)
	}
	if err := h.upload.UploadChunk(ctx, owner, "upload-1", 0, multipartHeader(t, "chunk", []byte("abc"))); err != nil {
		t.Fatalf("duplicate chunk must be idempotent: %v", err)
	}
	if _, err := h.upload.MergeChunks(ctx, owner, "upload-1"); err == nil {
		t.Fatal("merge must fail while a chunk is missing")
	}
	if err := h.upload.UploadChunk(ctx, owner, "upload-1", 1, multipartHeader(t, "chunk", []byte("def"))); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan model.File, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			file, err := h.upload.MergeChunks(ctx, owner, "upload-1")
			results <- file
			errs <- err
		}()
	}
	close(start)
	first, second := <-results, <-results
	if err1, err2 := <-errs, <-errs; err1 != nil || err2 != nil {
		t.Fatalf("concurrent merge failed: %v / %v", err1, err2)
	}
	if first.Id == 0 || first.Id != second.Id {
		t.Fatalf("merge must converge to one file: %d / %d", first.Id, second.Id)
	}
	var fileCount int64
	if err := h.db.DB.Model(&model.File{}).Count(&fileCount).Error; err != nil || fileCount != 1 || h.store.saveCount != 1 {
		t.Fatalf("expected one metadata and storage write: count=%d saves=%d err=%v", fileCount, h.store.saveCount, err)
	}
}

func TestFileDeleteIsRetryableAfterStorageCleanupFailure(t *testing.T) {
	h := newFileCapabilityHarness(t)
	ownerID := 10
	file := model.File{FileName: "note.txt", FilePath: "object.txt", FileMd5: "md5", FileUuid: "uuid", FileSize: 5}
	file.Id = 100
	file.CreateUser = &ownerID
	if err := h.db.DB.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	h.store.files[file.FilePath] = []byte("hello")
	resource, err := h.metadata.FindForDelete(fileTestContext(ownerID), file.Id)
	if err != nil {
		t.Fatal(err)
	}
	h.store.deleteErr = errors.New("cleanup failed")
	if err = h.metadata.Delete(fileTestContext(ownerID), resource); err == nil {
		t.Fatal("expected storage cleanup failure")
	}
	var deleted model.File
	if err = h.db.DB.Unscoped().First(&deleted, file.Id).Error; err != nil || !deleted.GmtDelete.Valid {
		t.Fatalf("metadata should be safely hidden before cleanup retry: %+v err=%v", deleted, err)
	}
	h.store.deleteErr = nil
	resource, err = h.metadata.FindForDelete(fileTestContext(ownerID), file.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err = h.metadata.Delete(fileTestContext(ownerID), resource); err != nil {
		t.Fatalf("repeated delete should retry cleanup: %v", err)
	}
	if len(h.store.files) != 0 {
		t.Fatal("storage object should be removed")
	}
}
