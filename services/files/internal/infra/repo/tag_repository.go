package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo/db"
)

// TagRepository は SQLC ベースのタグ永続化実装.
type TagRepository struct {
	q *db.Queries
}

// NewTagRepository は DB 接続を受け取り tag repository を生成する.
func NewTagRepository(pool db.DBTX) *TagRepository {
	return &TagRepository{q: db.New(pool)}
}

var _ domain.TagRepository = (*TagRepository)(nil)

func (r *TagRepository) Create(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	row, err := r.q.CreateTag(ctx, db.CreateTagParams{
		ID:    toPgUUID(tag.ID),
		Name:  tag.Name,
		Color: string(tag.Color),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrDuplicateTagName
		}
		return nil, err
	}
	return toDomainTag(row), nil
}

func (r *TagRepository) List(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.q.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Tag, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toDomainTag(row))
	}
	return out, nil
}

func (r *TagRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	row, err := r.q.GetTagByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTagNotFound
		}
		return nil, err
	}
	return toDomainTag(row), nil
}

func (r *TagRepository) Update(ctx context.Context, id uuid.UUID, name string, color domain.TagColor) (*domain.Tag, error) {
	row, err := r.q.UpdateTag(ctx, db.UpdateTagParams{
		ID:    toPgUUID(id),
		Name:  name,
		Color: string(color),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTagNotFound
		}
		if isUniqueViolation(err) {
			return nil, domain.ErrDuplicateTagName
		}
		return nil, err
	}
	return toDomainTag(row), nil
}

func (r *TagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.q.DeleteTag(ctx, toPgUUID(id)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTagNotFound
		}
		return err
	}
	return nil
}

func toDomainTag(row db.Tag) *domain.Tag {
	return &domain.Tag{
		ID:        uuid.UUID(row.ID.Bytes),
		Name:      row.Name,
		Color:     domain.TagColor(row.Color),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

