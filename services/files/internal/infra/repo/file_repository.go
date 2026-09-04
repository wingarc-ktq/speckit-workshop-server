// Package repo は Files サービスの永続化実装を提供する.
package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo/db"
)

const uniqueViolationCode = "23505"

// FileRepository は sqlc 生成コードをラップした domain.FileRepository の具象実装.
type FileRepository struct {
	q *db.Queries
}

// NewFileRepository は DB 接続を受け取り file repository を生成する.
func NewFileRepository(pool db.DBTX) *FileRepository {
	return &FileRepository{q: db.New(pool)}
}

// インターフェース実装の静態チェック.
var _ domain.FileRepository = (*FileRepository)(nil)

// Create は新しいファイルを永続化する.
func (r *FileRepository) Create(ctx context.Context, file *domain.File) (*domain.File, error) {
	row, err := r.q.CreateFile(ctx, db.CreateFileParams{
		ID:          toPgUUID(file.ID),
		OwnerUserID: toPgUUID(file.OwnerUserID),
		Name:        file.Name,
		Size:        file.Size,
		MimeType:    file.MIMEType,
		Description: stringPtr(file.Description),
	})
	if err != nil {
		return nil, err
	}
	return toDomainFile(row), nil
}

// List は所有者に紐づくファイル一覧を取得する.
func (r *FileRepository) List(ctx context.Context, ownerUserID uuid.UUID, keyword string, offset, limit int) ([]domain.File, error) {
	if limit <= 0 || offset < 0 {
		return nil, domain.ErrInvalidPagination
	}

	rows, err := r.q.ListFiles(ctx, db.ListFilesParams{
		Column1: toPgUUID(ownerUserID),
		Column2: keyword,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, err
	}

	files := make([]domain.File, 0, len(rows))
	for _, row := range rows {
		file := toDomainFile(row)
		file.TagIDs = r.listTagIDs(ctx, file.ID)
		files = append(files, *file)
	}
	return files, nil
}

// Count は検索条件に一致するファイル件数を返す.
func (r *FileRepository) Count(ctx context.Context, ownerUserID uuid.UUID, keyword string) (int, error) {
	count, err := r.q.CountFiles(ctx, db.CountFilesParams{
		Column1: toPgUUID(ownerUserID),
		Column2: keyword,
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetByID はファイル ID でファイルを取得する.
func (r *FileRepository) GetByID(ctx context.Context, ownerUserID, fileID uuid.UUID) (*domain.File, error) {
	row, err := r.q.GetFileByID(ctx, db.GetFileByIDParams{
		ID:      toPgUUID(fileID),
		Column2: toPgUUID(ownerUserID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}
	file := toDomainFile(row)
	file.TagIDs = r.listTagIDs(ctx, file.ID)
	return file, nil
}

// UpdateMetadata はファイルのメタデータを更新する.
func (r *FileRepository) UpdateMetadata(ctx context.Context, ownerUserID, fileID uuid.UUID, name string, description *string, tagIDs []uuid.UUID) (*domain.File, error) {
	row, err := r.q.UpdateFileMetadata(ctx, db.UpdateFileMetadataParams{
		ID:          toPgUUID(fileID),
		Column2:     toPgUUID(ownerUserID),
		Name:        name,
		Description: stringPtr(description),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}
	file := toDomainFile(row)
	if err := r.replaceTagIDs(ctx, file.ID, tagIDs); err != nil {
		return nil, err
	}
	file.TagIDs = r.listTagIDs(ctx, file.ID)
	return file, nil
}

func (r *FileRepository) listTagIDs(ctx context.Context, fileID uuid.UUID) []uuid.UUID {
	rows, err := r.q.ListFileTagIDs(ctx, toPgUUID(fileID))
	if err != nil {
		return nil
	}
	tagIDs := make([]uuid.UUID, 0, len(rows))
	for _, id := range rows {
		tagIDs = append(tagIDs, uuid.UUID(id.Bytes))
	}
	return tagIDs
}

func (r *FileRepository) replaceTagIDs(ctx context.Context, fileID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := r.q.DeleteFileTags(ctx, toPgUUID(fileID)); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := r.q.AddFileTag(ctx, db.AddFileTagParams{FileID: toPgUUID(fileID), TagID: toPgUUID(tagID)}); err != nil {
			if isForeignKeyViolation(err) {
				continue
			}
			return err
		}
	}
	return nil
}

// Delete は単一ファイルを削除する.
func (r *FileRepository) Delete(ctx context.Context, ownerUserID, fileID uuid.UUID) error {
	if err := r.q.DeleteFile(ctx, db.DeleteFileParams{
		ID:      toPgUUID(fileID),
		Column2: toPgUUID(ownerUserID),
	}); err != nil {
		return err
	}
	return nil
}

// DeleteByIDs は複数ファイルを一括削除する.
func (r *FileRepository) DeleteByIDs(ctx context.Context, ownerUserID uuid.UUID, fileIDs []uuid.UUID) error {
	ids := make([]pgtype.UUID, 0, len(fileIDs))
	for _, id := range fileIDs {
		ids = append(ids, toPgUUID(id))
	}
	if err := r.q.DeleteFilesByIDs(ctx, db.DeleteFilesByIDsParams{
		Column1: toPgUUID(ownerUserID),
		Column2: ids,
	}); err != nil {
		return err
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func stringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func toDomainFile(f db.File) *domain.File {
	file := &domain.File{
		ID:          uuid.UUID(f.ID.Bytes),
		OwnerUserID: uuid.UUID(f.OwnerUserID.Bytes),
		Name:        f.Name,
		Size:        f.Size,
		MIMEType:    f.MimeType,
		UploadedAt:  f.UploadedAt.Time,
	}
	if f.Description != nil {
		desc := *f.Description
		file.Description = &desc
	}
	return file
}
