package service

import (
	"backend/dto/response"
	myerrors "backend/internal/errors"
	"backend/internal/storage"
	"backend/model"
	"backend/repository"
	"context"

	"gorm.io/gorm"
)

type FileMetadataService struct {
	files   repository.FileRepository
	storage storage.Storage
}

func NewFileMetadataService(files repository.FileRepository, store storage.Storage) *FileMetadataService {
	return &FileMetadataService{files: files, storage: store}
}

func (s *FileMetadataService) FindByID(ctx context.Context, id int) (FileAccessResource, error) {
	file, err := s.files.FindByIdWithDB(s.files.DBWithContext(ctx), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FileAccessResource{}, myerrors.ErrFileNotFound
		}
		return FileAccessResource{}, myerrors.WrapSystemError(err)
	}
	return FileAccessResource{file: file}, nil
}

func (s *FileMetadataService) FindForDelete(ctx context.Context, id int) (FileAccessResource, error) {
	file, err := s.files.WithUnscoped().FindByIdWithDB(s.files.DBWithContext(ctx).Unscoped(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FileAccessResource{}, myerrors.ErrFileNotFound
		}
		return FileAccessResource{}, myerrors.WrapSystemError(err)
	}
	return FileAccessResource{file: file}, nil
}

func (s *FileMetadataService) Detail(resource FileAccessResource) response.FileDetailRes {
	return fileDetailResponse(resource.file)
}

func (s *FileMetadataService) Delete(ctx context.Context, resource FileAccessResource) error {
	file := resource.file
	if !file.GmtDelete.Valid {
		if err := RunInTransaction(ctx, s.files.DBWithContext(ctx), func(tx *gorm.DB) error {
			return s.files.DeleteFile(tx, file)
		}); err != nil {
			return myerrors.WrapSystemError(err)
		}
	}
	if file.FilePath != "" {
		if err := s.storage.Delete(file.FilePath); err != nil {
			return myerrors.WrapSystemError(err)
		}
	}
	return nil
}

func fileDetailResponse(data model.File) response.FileDetailRes {
	return response.FileDetailRes{
		BasicRes: response.NewBasicRes(data.Basic),
		FileName: data.FileName, FileType: data.FileType, FileUrl: data.FileUrl,
		FileSize: data.FileSize, FileExt: data.FileExt, FileUuid: data.FileUuid,
	}
}
