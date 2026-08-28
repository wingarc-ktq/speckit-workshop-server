// Package repo はユースケースのポート実装を提供する.
package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo/db"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

// FileRepository は sqlc 生成コード (db.Queries) をラップした
// usecase.FileRepository の具象実装.
type FileRepository struct {
	q *db.Queries
}

// NewFileRepository は *pgxpool.Pool（DBTX を満たす）を受け取り FileRepository を生成する.
func NewFileRepository(pool db.DBTX) *FileRepository {
	return &FileRepository{q: db.New(pool)}
}

// インターフェース実装の静的チェック.
// Create/List/FindByID がすべて揃った（Phase 3〜5）ことで usecase.FileRepository を
// 満たすようになったため、ここで宣言できるようになった.
var _ usecase.FileRepository = (*FileRepository)(nil)

// Create は新しいファイルメタデータを永続化する.
func (r *FileRepository) Create(ctx context.Context, file *domain.File) error {
	row, err := r.q.CreateFile(ctx, db.CreateFileParams{
		ID:          toPgUUID(file.ID),
		Name:        file.Name,
		Size:        file.Size,
		MimeType:    file.MimeType,
		Description: file.Description,
		StorageKey:  file.StorageKey,
		TagIds:      toPgUUIDs(file.TagIDs),
	})
	if err != nil {
		return err
	}
	// DB 側で生成された uploaded_at を書き戻す.
	file.UploadedAt = row.UploadedAt.Time
	return nil
}

// FindByID は ID でファイルメタデータを検索する.
// SQL がヒットしない場合、sqlc 生成コードは pgx.ErrNoRows を返す。
// これはあくまで pgx（DB ドライバ）が使う内部的なエラー型であり、
// usecase/handler 層に漏れ出すと DB の詳細に依存してしまう。
// そのため、ここで domain.ErrFileNotFound（このサービスのドメイン語彙）に変換してから返す
// （出典: spec.md FR-010「存在しないファイル ID に対して存在しないエラーを返さなければならない」）。
func (r *FileRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	row, err := r.q.GetFileByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}
	return &domain.File{
		ID:          uuid.UUID(row.ID.Bytes),
		Name:        row.Name,
		Size:        row.Size,
		MimeType:    row.MimeType,
		Description: row.Description,
		StorageKey:  row.StorageKey,
		TagIDs:      toDomainUUIDs(row.TagIds),
		UploadedAt:  row.UploadedAt.Time,
	}, nil
}

// List は検索・ページネーション条件に一致するファイル一覧と総件数を返す.
// Offset は (Page-1)*Limit として算出する（data-model.md 参照）.
func (r *FileRepository) List(ctx context.Context, params usecase.ListFilesParams) ([]*domain.File, int64, error) {
	var search *string
	if params.Search != "" {
		search = &params.Search
	}
	var tagIDs []pgtype.UUID
	if len(params.TagIDs) > 0 {
		tagIDs = toPgUUIDs(params.TagIDs)
	}

	rows, err := r.q.ListFiles(ctx, db.ListFilesParams{
		Search: search,
		TagIds: tagIDs,
		Off:    int32((params.Page - 1) * params.Limit),
		Lim:    int32(params.Limit),
	})
	if err != nil {
		return nil, 0, err
	}

	files := make([]*domain.File, len(rows))
	var total int64
	for i, row := range rows {
		files[i] = &domain.File{
			ID:          uuid.UUID(row.ID.Bytes),
			Name:        row.Name,
			Size:        row.Size,
			MimeType:    row.MimeType,
			Description: row.Description,
			StorageKey:  row.StorageKey,
			TagIDs:      toDomainUUIDs(row.TagIds),
			UploadedAt:  row.UploadedAt.Time,
		}
		total = row.TotalCount
	}
	return files, total, nil
}

func toDomainUUIDs(ids []pgtype.UUID) []uuid.UUID {
	out := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		out[i] = uuid.UUID(id.Bytes)
	}
	return out
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgUUIDs(ids []uuid.UUID) []pgtype.UUID {
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		out[i] = toPgUUID(id)
	}
	return out
}
