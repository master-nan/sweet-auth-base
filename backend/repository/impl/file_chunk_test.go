package impl

import (
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/model"
	"context"
	"testing"
)

func TestFileChunkUploadIndexIsUnique(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.FileChunk{})
	repository := NewFileChunkRepositoryImpl(&database.PrimaryDB{DB: db})
	first := model.FileChunk{UploadId: "upload-1", ChunkIndex: 0}
	first.Id = 1
	if err := repository.Create(repository.DBWithContext(context.Background()), &first); err != nil {
		t.Fatalf("create first chunk: %v", err)
	}
	duplicate := model.FileChunk{UploadId: "upload-1", ChunkIndex: 0}
	duplicate.Id = 2
	if err := repository.Create(repository.DBWithContext(context.Background()), &duplicate); err == nil {
		t.Fatal("expected duplicate upload/chunk index to be rejected")
	}
}

func TestFindUnfinishedUploadIsScopedToOwner(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.FileChunk{})
	repository := NewFileChunkRepositoryImpl(&database.PrimaryDB{DB: db})
	ownerOne, ownerTwo := 10, 20
	chunks := []model.FileChunk{
		{UploadId: "owner-one", FileMd5: "same", FileSize: 5, FileName: "same.txt"},
		{UploadId: "owner-two", FileMd5: "same", FileSize: 5, FileName: "same.txt"},
	}
	chunks[0].CreateUser = &ownerOne
	chunks[1].CreateUser = &ownerTwo
	for index := range chunks {
		chunks[index].Id = index + 1
	}
	if err := db.Create(&chunks).Error; err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindUnfinishedUpload(context.Background(), "same", 5, "same.txt", ownerOne, false)
	if err != nil || found.UploadId != "owner-one" {
		t.Fatalf("expected owner-scoped upload, got %+v err=%v", found, err)
	}
}
