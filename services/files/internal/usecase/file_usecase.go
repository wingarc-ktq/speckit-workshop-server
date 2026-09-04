// Package usecase は Files サービスのアプリケーションロジックを提供する.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

// FileUsecase はファイルに関するユースケース契約.
type FileUsecase interface {
	Upload(ctx context.Context, in UploadInput) (*UploadOutput, error)
	List(ctx context.Context, in ListInput) (*ListOutput, error)
	Get(ctx context.Context, in GetInput) (*domain.File, error)
	Download(ctx context.Context, in DownloadInput) (*DownloadOutput, error)
	UpdateMetadata(ctx context.Context, in UpdateMetadataInput) (*domain.File, error)
	Delete(ctx context.Context, in DeleteInput) error
	DeleteFiles(ctx context.Context, in DeleteFilesInput) error
}

// UploadInput はファイルアップロードの入力.
type UploadInput struct {
	OwnerUserID uuid.UUID
	FileName    string
	MIMEType    string
	Description *string
	Data        []byte
}

// UploadOutput はファイルアップロードの結果.
type UploadOutput struct {
	File        *domain.File
	DownloadURL string
}

// ListInput はファイル一覧取得の入力.
type ListInput struct {
	OwnerUserID uuid.UUID
	Keyword     string
	Page        int
	Limit       int
}

// ListOutput はファイル一覧取得の結果.
type ListOutput struct {
	Files []domain.File
	Total int
	Page  int
	Limit int
}

// GetInput はファイル詳細取得の入力.
type GetInput struct {
	OwnerUserID uuid.UUID
	FileID      uuid.UUID
}

// DownloadInput はファイルダウンロードの入力.
type DownloadInput struct {
	OwnerUserID uuid.UUID
	FileID      uuid.UUID
}

// DownloadOutput はファイルダウンロードの結果.
type DownloadOutput struct {
	File *domain.File
	Data []byte
}

// UpdateMetadataInput はファイルメタデータ更新の入力.
type UpdateMetadataInput struct {
	OwnerUserID uuid.UUID
	FileID      uuid.UUID
	Name        *string
	Description *string
	TagIDs      []uuid.UUID
}

// DeleteInput はファイル個別削除の入力.
type DeleteInput struct {
	OwnerUserID uuid.UUID
	FileID      uuid.UUID
}

// DeleteFilesInput はファイル一括削除の入力.
type DeleteFilesInput struct {
	OwnerUserID uuid.UUID
	FileIDs     []uuid.UUID
}

type fileUsecase struct {
	repo    domain.FileRepository
	storage domain.FileStorage
	maxSize int64
}

// NewFileUsecase はファイルアップロードのユースケースを生成する.
func NewFileUsecase(repo domain.FileRepository, storage domain.FileStorage, maxSize int64) FileUsecase {
	return &fileUsecase{repo: repo, storage: storage, maxSize: maxSize}
}

// Upload はファイルを保存し、メタデータを DB に登録する.
func (u *fileUsecase) Upload(ctx context.Context, in UploadInput) (*UploadOutput, error) {
	if in.OwnerUserID == uuid.Nil {
		return nil, domain.ErrInvalidFile
	}
	if strings.TrimSpace(in.FileName) == "" {
		return nil, domain.ErrInvalidFile
	}
	if strings.IndexByte(in.FileName, 0) >= 0 || (in.Description != nil && strings.IndexByte(*in.Description, 0) >= 0) {
		return nil, domain.ErrInvalidFile
	}
	if int64(len(in.Data)) > u.maxSize {
		return nil, domain.ErrFileTooLarge
	}
	stored, err := u.storage.Save(ctx, domain.FileContent{
		OriginalName: in.FileName,
		MIMEType:     in.MIMEType,
		Size:         int64(len(in.Data)),
		Data:         in.Data,
	})
	if err != nil {
		return nil, err
	}

	file := &domain.File{
		ID:          stored.ID,
		OwnerUserID: in.OwnerUserID,
		Name:        in.FileName,
		Size:        stored.Size,
		MIMEType:    in.MIMEType,
		Description: in.Description,
		UploadedAt:  time.Now(),
	}

	created, err := u.repo.Create(ctx, file)
	if err != nil {
		if cleanupErr := u.storage.Delete(ctx, stored); cleanupErr != nil {
			return nil, fmt.Errorf("upload: create file DB failed and cleanup storage: %w", errors.Join(err, cleanupErr))
		}
		return nil, err
	}

	return &UploadOutput{
		File:        created,
		DownloadURL: fmt.Sprintf("/api/v1/files/%s/download", created.ID.String()),
	}, nil
}

// List はファイル一覧を取得し、ページネーションと検索条件を適用する.
func (u *fileUsecase) List(ctx context.Context, in ListInput) (*ListOutput, error) {
	if in.OwnerUserID == uuid.Nil {
		return nil, domain.ErrInvalidFile
	}
	if in.Page < 1 || in.Limit < 1 || in.Limit > 100 {
		return nil, domain.ErrInvalidPagination
	}

	if strings.IndexByte(in.Keyword, 0) >= 0 {
		return nil, domain.ErrInvalidFile
	}
	if in.Page-1 > math.MaxInt/in.Limit {
		count, err := u.repo.Count(ctx, in.OwnerUserID, in.Keyword)
		if err != nil {
			return nil, err
		}
		return &ListOutput{Files: []domain.File{}, Total: count, Page: in.Page, Limit: in.Limit}, nil
	}
	offset := (in.Page - 1) * in.Limit
	if offset > math.MaxInt32 {
		count, err := u.repo.Count(ctx, in.OwnerUserID, in.Keyword)
		if err != nil {
			return nil, err
		}
		return &ListOutput{Files: []domain.File{}, Total: count, Page: in.Page, Limit: in.Limit}, nil
	}
	files, err := u.repo.List(ctx, in.OwnerUserID, in.Keyword, offset, in.Limit)
	if err != nil {
		return nil, err
	}
	count, err := u.repo.Count(ctx, in.OwnerUserID, in.Keyword)
	if err != nil {
		return nil, err
	}

	return &ListOutput{
		Files: files,
		Total: count,
		Page:  in.Page,
		Limit: in.Limit,
	}, nil
}

// Get はファイル詳細を取得する.
func (u *fileUsecase) Get(ctx context.Context, in GetInput) (*domain.File, error) {
	if in.OwnerUserID == uuid.Nil || in.FileID == uuid.Nil {
		return nil, domain.ErrInvalidFile
	}
	file, err := u.repo.GetByID(ctx, in.OwnerUserID, in.FileID)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Download はファイルの本体を取得する.
func (u *fileUsecase) Download(ctx context.Context, in DownloadInput) (*DownloadOutput, error) {
	if in.OwnerUserID == uuid.Nil || in.FileID == uuid.Nil {
		return nil, domain.ErrInvalidFile
	}
	file, err := u.repo.GetByID(ctx, in.OwnerUserID, in.FileID)
	if err != nil {
		return nil, err
	}
	data, err := u.storage.OpenByFile(ctx, file)
	if err != nil {
		return nil, err
	}
	return &DownloadOutput{File: file, Data: data}, nil
}

// UpdateMetadata はファイル名・説明・タグ一覧をまとめて更新する.
func (u *fileUsecase) UpdateMetadata(ctx context.Context, in UpdateMetadataInput) (*domain.File, error) {
	if in.OwnerUserID == uuid.Nil || in.FileID == uuid.Nil {
		return nil, domain.ErrInvalidFile
	}
	current, err := u.repo.GetByID(ctx, in.OwnerUserID, in.FileID)
	if err != nil {
		return nil, err
	}

	name := current.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if name == "" || strings.IndexByte(name, 0) >= 0 {
			return nil, domain.ErrInvalidFile
		}
	}

	description := current.Description
	if in.Description != nil {
		if strings.IndexByte(*in.Description, 0) >= 0 {
			return nil, domain.ErrInvalidFile
		}
		trimmed := strings.TrimSpace(*in.Description)
		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	}

	tagIDs := current.TagIDs
	if in.TagIDs != nil {
		tagIDs = append([]uuid.UUID(nil), in.TagIDs...)
	}

	updated, err := u.repo.UpdateMetadata(ctx, in.OwnerUserID, in.FileID, name, description, tagIDs)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		updated.TagIDs = tagIDs
	}
	return updated, nil
}

// Delete は単一ファイルを削除する.
func (u *fileUsecase) Delete(ctx context.Context, in DeleteInput) error {
	if in.OwnerUserID == uuid.Nil || in.FileID == uuid.Nil {
		return domain.ErrInvalidFile
	}
	file, err := u.repo.GetByID(ctx, in.OwnerUserID, in.FileID)
	if err != nil {
		return err
	}
	if u.storage != nil {
		if err := u.storage.DeleteByFile(ctx, file); err != nil {
			return err
		}
	}
	return u.repo.Delete(ctx, in.OwnerUserID, in.FileID)
}

// DeleteFiles は複数ファイルを一括削除する.
func (u *fileUsecase) DeleteFiles(ctx context.Context, in DeleteFilesInput) error {
	if in.OwnerUserID == uuid.Nil {
		return domain.ErrInvalidFile
	}
	if len(in.FileIDs) == 0 {
		return domain.ErrInvalidFile
	}
	for _, fileID := range in.FileIDs {
		if fileID == uuid.Nil {
			return domain.ErrInvalidFile
		}
		file, err := u.repo.GetByID(ctx, in.OwnerUserID, fileID)
		if err != nil {
			if errors.Is(err, domain.ErrFileNotFound) {
				continue
			}
			return err
		}
		if u.storage != nil {
			if err := u.storage.DeleteByFile(ctx, file); err != nil {
				return err
			}
		}
	}
	return u.repo.DeleteByIDs(ctx, in.OwnerUserID, in.FileIDs)
}
