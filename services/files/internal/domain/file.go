// Package domain は Files サービスのドメインモデルとドメインエラーを定義する.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// File はドメイン層のファイルメタデータモデル.
// インフラ層 (DB) の表現とは独立して定義する.
type File struct {
	ID          uuid.UUID
	Name        string
	Size        int64
	MimeType    string
	Description string
	// StorageKey は FileStorage ポートに渡す一意キー. API レスポンスには含めない.
	StorageKey string
	TagIDs     []uuid.UUID
	UploadedAt time.Time
}

// ドメインエラー
var (
	ErrFileNotFound = errors.New("file not found")
	ErrFileTooLarge = errors.New("file too large")
	ErrFileEmpty    = errors.New("file is required")
)
