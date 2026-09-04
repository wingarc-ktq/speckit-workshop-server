package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

type FileUsecase struct {
	repo     FileRepository
	storage  FileStorage
	maxBytes int64
}

func NewFileUsecase(repo FileRepository, storage FileStorage, maxBytes int64) *FileUsecase {
	if maxBytes <= 0 {
		maxBytes = domain.MaxFileSize
	}
	return &FileUsecase{repo: repo, storage: storage, maxBytes: maxBytes}
}

func (u *FileUsecase) UploadFile(ctx context.Context, input UploadInput) (domain.File, error) {
	if input.Reader == nil {
		return domain.File{}, fmt.Errorf("upload file: %w", domain.ErrInvalidFileName)
	}
	if input.Size < 0 || input.Size > u.maxBytes {
		return domain.File{}, fmt.Errorf("upload file: %w", domain.ErrFileTooLarge)
	}

	payload, err := io.ReadAll(io.LimitReader(input.Reader, u.maxBytes+1))
	if err != nil {
		return domain.File{}, fmt.Errorf("read upload: %w", err)
	}
	if int64(len(payload)) > u.maxBytes {
		return domain.File{}, fmt.Errorf("upload file: %w", domain.ErrFileTooLarge)
	}

	storageKey, err := u.storage.Save(ctx, input.FileName, bytes.NewReader(payload), u.maxBytes)
	if err != nil {
		return domain.File{}, fmt.Errorf("save storage: %w", err)
	}

	file, err := domain.NewFile(input.FileName, input.MIMEType, input.Description, storageKey, int64(len(payload)), input.TagIDs)
	if err != nil {
		if removeErr := u.storage.Delete(ctx, storageKey); removeErr != nil {
			return domain.File{}, fmt.Errorf("rollback storage: %v: %w", removeErr, err)
		}
		return domain.File{}, fmt.Errorf("create file model: %w", err)
	}

	if err := u.repo.Save(ctx, file, input.TagIDs); err != nil {
		if removeErr := u.storage.Delete(ctx, storageKey); removeErr != nil {
			return domain.File{}, fmt.Errorf("rollback storage after repo failure: %v: %w", removeErr, err)
		}
		return domain.File{}, fmt.Errorf("save file metadata: %w", err)
	}

	return file, nil
}

func (u *FileUsecase) ListFiles(ctx context.Context, input ListFilesInput) ([]domain.File, int64, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.Limit < 1 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	files, total, err := u.repo.List(ctx, input.Name, input.TagIDs, input.Page, input.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list files: %w", err)
	}
	return files, total, nil
}

func (u *FileUsecase) GetFileByID(ctx context.Context, id uuid.UUID) (domain.File, error) {
	file, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return domain.File{}, fmt.Errorf("get file: %w", err)
	}
	return file, nil
}

func (u *FileUsecase) DownloadFile(ctx context.Context, id uuid.UUID) (domain.File, io.ReadCloser, error) {
	file, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return domain.File{}, nil, fmt.Errorf("get file for download: %w", err)
	}
	reader, err := u.storage.Open(ctx, file.StorageKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.File{}, nil, fmt.Errorf("open storage file: %w", domain.ErrFileNotFound)
		}
		return domain.File{}, nil, fmt.Errorf("open storage file: %w", err)
	}
	return file, reader, nil
}
