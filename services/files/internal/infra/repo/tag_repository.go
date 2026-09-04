package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo/db"
)

// TagRepository はタグの参照と関連付けを担う簡易リポジトリです。
type TagRepository struct {
	q *db.Queries
}

func NewTagRepository(pool db.DBTX) *TagRepository {
	return &TagRepository{q: db.New(pool)}
}

func (r *TagRepository) Save(ctx context.Context, tag domain.Tag) error {
	_, err := r.q.UpsertTag(ctx, db.UpsertTagParams{
		ID:    toPGUUID(tag.ID),
		Name:  tag.Name,
		Color: db.TagColor(tag.Color),
	})
	return err
}

func (r *TagRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := r.q.GetTagByID(ctx, toPGUUID(id))
	if err != nil {
		return false, err
	}
	return true, nil
}
