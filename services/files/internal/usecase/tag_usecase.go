package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
)

// TagUsecase はタグ業務の入力ポート.
type TagUsecase interface {
	Create(ctx context.Context, in CreateTagInput) (*domain.Tag, error)
	List(ctx context.Context) ([]domain.Tag, error)
	Update(ctx context.Context, in UpdateTagInput) (*domain.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CreateTagInput はタグ作成の入力.
type CreateTagInput struct {
	Name  string
	Color string
}

// UpdateTagInput はタグ更新の入力.
type UpdateTagInput struct {
	TagID uuid.UUID
	Name  string
	Color string
}

type tagUsecase struct {
	repo domain.TagRepository
}

// NewTagUsecase はタグ用ユースケースを生成する.
func NewTagUsecase(repo domain.TagRepository) TagUsecase {
	return &tagUsecase{repo: repo}
}

func (u *tagUsecase) Create(ctx context.Context, in CreateTagInput) (*domain.Tag, error) {
	name := strings.TrimSpace(in.Name)
	color, err := normalizeTagColor(in.Color)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, domain.ErrInvalidTag
	}
	tag := &domain.Tag{
		ID:    uuid.New(),
		Name:  name,
		Color: color,
	}
	created, err := u.repo.Create(ctx, tag)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (u *tagUsecase) List(ctx context.Context) ([]domain.Tag, error) {
	return u.repo.List(ctx)
}

func (u *tagUsecase) Update(ctx context.Context, in UpdateTagInput) (*domain.Tag, error) {
	if in.TagID == uuid.Nil {
		return nil, domain.ErrInvalidTag
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrInvalidTag
	}
	color, err := normalizeTagColor(in.Color)
	if err != nil {
		return nil, err
	}
	return u.repo.Update(ctx, in.TagID, name, color)
}

func (u *tagUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErrInvalidTag
	}
	return u.repo.Delete(ctx, id)
}

func normalizeTagColor(value string) (domain.TagColor, error) {
	switch domain.TagColor(strings.TrimSpace(value)) {
	case domain.TagColorBlue, domain.TagColorRed, domain.TagColorYellow,
		domain.TagColorGreen, domain.TagColorPurple, domain.TagColorOrange, domain.TagColorGray:
		return domain.TagColor(strings.TrimSpace(value)), nil
	default:
		return "", domain.ErrInvalidTag
	}
}

var _ = errors.New
