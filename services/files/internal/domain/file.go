// Package domain は Files サービスのドメインモデルとドメインエラーを定義する.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// File はファイル管理のドメインモデル.
type File struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID
	Name        string
	Size        int64
	MIMEType    string
	Description *string
	TagIDs      []uuid.UUID
	UploadedAt  time.Time
}

// FileRepository はファイル永続化の抽象.
type FileRepository interface {
	Create(ctx context.Context, file *File) (*File, error)
	List(ctx context.Context, ownerUserID uuid.UUID, keyword string, offset, limit int) ([]File, error)
	Count(ctx context.Context, ownerUserID uuid.UUID, keyword string) (int, error)
	GetByID(ctx context.Context, ownerUserID, fileID uuid.UUID) (*File, error)
	UpdateMetadata(ctx context.Context, ownerUserID, fileID uuid.UUID, name string, description *string, tagIDs []uuid.UUID) (*File, error)
	Delete(ctx context.Context, ownerUserID, fileID uuid.UUID) error
	DeleteByIDs(ctx context.Context, ownerUserID uuid.UUID, fileIDs []uuid.UUID) error
}

// ドメインエラー.
var (
	ErrFileNotFound      = errors.New("file not found")
	ErrFileTooLarge      = errors.New("file too large")
	ErrInvalidPagination = errors.New("invalid pagination")
	ErrInvalidFile       = errors.New("invalid file")
)
