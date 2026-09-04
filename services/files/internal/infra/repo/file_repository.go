package repo

import (
	"context"
	"errors"
	"math"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo/db"
)

type InMemoryFileRepository struct {
	mu    sync.Mutex
	files map[uuid.UUID]domain.File
}

func NewInMemoryFileRepository() *InMemoryFileRepository {
	return &InMemoryFileRepository{files: make(map[uuid.UUID]domain.File)}
}

func (r *InMemoryFileRepository) Save(ctx context.Context, file domain.File, tagIDs []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	file.TagIDs = append([]uuid.UUID(nil), tagIDs...)
	r.files[file.ID] = file
	return nil
}

func (r *InMemoryFileRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.File, error) {
	if ctx.Err() != nil {
		return domain.File{}, ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.files[id]
	if !ok {
		return domain.File{}, domain.ErrFileNotFound
	}
	return f, nil
}

func (r *InMemoryFileRepository) List(ctx context.Context, name string, tagIDs []uuid.UUID, page int, limit int) ([]domain.File, int64, error) {
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	files := make([]domain.File, 0, len(r.files))
	for _, f := range r.files {
		if name != "" && !containsFold(f.Name, name) {
			continue
		}
		if len(tagIDs) > 0 && !containsAll(f.TagIDs, tagIDs) {
			continue
		}
		files = append(files, f)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	start := (page - 1) * limit
	if start > len(files) {
		return []domain.File{}, int64(len(files)), nil
	}
	end := start + limit
	if end > len(files) {
		end = len(files)
	}
	return files[start:end], int64(len(files)), nil
}

// FileRepository は PostgreSQL の実体に対する定義を持つ sqlc wrapper である.
type FileRepository struct {
	q *db.Queries
}

func NewFileRepository(pool db.DBTX) *FileRepository {
	return &FileRepository{q: db.New(pool)}
}

func (r *FileRepository) Save(ctx context.Context, file domain.File, tagIDs []uuid.UUID) error {
	_, err := r.q.InsertFile(ctx, db.InsertFileParams{
		ID:          toPGUUID(file.ID),
		Name:        file.Name,
		MimeType:    file.MIMEType,
		Description: pgString(file.Description),
		StorageKey:  file.StorageKey,
		Size:        file.Size,
	})
	if err != nil {
		return err
	}
	for _, id := range tagIDs {
		if err := r.q.InsertFileTag(ctx, db.InsertFileTagParams{FileID: toPGUUID(file.ID), TagID: toPGUUID(id)}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return domain.ErrTagNotFound
			}
			return err
		}
	}
	return nil
}

func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.File, error) {
	row, err := r.q.GetFileByID(ctx, toPGUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.File{}, domain.ErrFileNotFound
		}
		return domain.File{}, err
	}
	tags, err := r.q.GetFileTags(ctx, toPGUUID(id))
	if err != nil {
		return domain.File{}, err
	}
	file, err := toDomainFile(row, tags)
	if err != nil {
		return domain.File{}, err
	}
	return file, nil
}

func (r *FileRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.File, error) {
	return r.GetByID(ctx, id)
}

func (r *FileRepository) List(ctx context.Context, name string, tagIDs []uuid.UUID, page int, limit int) ([]domain.File, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := int64(page-1) * int64(limit)
	if page > 1 && int64(page-1) > math.MaxInt64/int64(limit) {
		offset = math.MaxInt64
	}
	rows, err := r.q.ListFiles(ctx, db.ListFilesParams{
		Column1: name,
		Column2: int64(limit),
		Column3: offset,
	})
	if err != nil {
		return nil, 0, err
	}
	files := make([]domain.File, 0, len(rows))
	for _, row := range rows {
		tags, err := r.q.GetFileTags(ctx, row.ID)
		if err != nil {
			return nil, 0, err
		}
		fileRow := db.GetFileByIDRow{
			ID:          row.ID,
			Name:        row.Name,
			MimeType:    row.MimeType,
			Description: row.Description,
			StorageKey:  row.StorageKey,
			Size:        row.Size,
			UploadedAt:  row.UploadedAt,
		}
		file, err := toDomainFile(fileRow, tags)
		if err != nil {
			return nil, 0, err
		}
		if len(tagIDs) > 0 && !containsAll(file.TagIDs, tagIDs) {
			continue
		}
		files = append(files, file)
	}
	if len(tagIDs) > 0 {
		return files, int64(len(files)), nil
	}
	total, err := r.q.CountFiles(ctx, name)
	if err != nil {
		return nil, 0, err
	}
	return files, total, nil
}

func containsFold(s string, sub string) bool {
	return stringsContainsFold(s, sub)
}

func containsAll(haystack []uuid.UUID, needles []uuid.UUID) bool {
	for _, needle := range needles {
		matched := false
		for _, item := range haystack {
			if item == needle {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func pgString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toPGUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
