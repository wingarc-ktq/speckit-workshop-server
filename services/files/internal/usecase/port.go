// Package usecase の外部依存（ポート）を定義する.
package usecase

import (
	"context"
	"io"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

// ListFilesParams は一覧取得の検索・ページネーション条件.
// 並び順は uploaded_at DESC 固定（data-model.md 参照）.
type ListFilesParams struct {
	// Page は 1 始まりのページ番号.
	Page int
	// Limit は 1 ページあたりの件数.
	Limit int
	// Search はファイル名の部分一致検索キーワード. 空文字は条件なし.
	Search string
	// TagIDs は絞り込むタグ ID の集合（配列オーバーラップ）. 空は条件なし.
	TagIDs []uuid.UUID
}

// FileRepository はファイルメタデータストアの抽象.
// 具象は internal/infra/repo で実装される.
//
//go:generate mockgen -source=port.go -destination=mock/port_mock.go -package=mock
type FileRepository interface {
	// Create は新しいファイルメタデータを永続化する.
	Create(ctx context.Context, file *domain.File) error
	// FindByID は ID でファイルメタデータを検索する.
	// 該当なしは domain.ErrFileNotFound を返す.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.File, error)
	// List は検索・ページネーション条件に一致するファイル一覧と総件数を返す.
	List(ctx context.Context, params ListFilesParams) (files []*domain.File, total int64, err error)
}

// FileStorage はファイル本体の保存先の抽象.
// 具象 (ローカルファイルシステムなど) は internal/infra/storage で実装される.
type FileStorage interface {
	// Save は key に対して r の内容を保存する.
	Save(ctx context.Context, key string, r io.Reader) error
	// Open は key で保存されたファイル本体を読み取り用に開く.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}
