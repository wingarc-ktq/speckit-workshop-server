// Package domain は Files サービスの外部ストレージ抽象化の契約を定義する.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FileContent は保存対象のファイル内容を表す.
type FileContent struct {
	OriginalName string
	MIMEType     string
	Size         int64
	Data         []byte
}

// StoredFile はストレージ上に保存されたファイルの参照情報を表す.
type StoredFile struct {
	ID        uuid.UUID
	Name      string
	Path      string
	Size      int64
	MIMEType  string
	CreatedAt time.Time
}

// FileStorage はファイルの保存・読込・削除を抽象化するポート.
type FileStorage interface {
	Save(ctx context.Context, content FileContent) (*StoredFile, error)
	Open(ctx context.Context, storedFile *StoredFile) ([]byte, error)
	OpenByFile(ctx context.Context, file *File) ([]byte, error)
	Delete(ctx context.Context, storedFile *StoredFile) error
	DeleteByFile(ctx context.Context, file *File) error
}
