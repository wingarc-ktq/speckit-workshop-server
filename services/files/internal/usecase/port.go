package usecase

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

type FileRepository interface {
	Save(ctx context.Context, file domain.File, tagIDs []uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.File, error)
	List(ctx context.Context, name string, tagIDs []uuid.UUID, page int, limit int) ([]domain.File, int64, error)
}

type FileStorage interface {
	Save(ctx context.Context, fileName string, reader io.Reader, maxBytes int64) (string, error)
	Open(ctx context.Context, storageKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, storageKey string) error
}

type UploadInput struct {
	FileName    string
	MIMEType    string
	Description string
	Size        int64
	TagIDs      []uuid.UUID
	Reader      io.Reader
}

type ListFilesInput struct {
	Page    int
	Limit   int
	Name    string
	TagIDs  []uuid.UUID
}
