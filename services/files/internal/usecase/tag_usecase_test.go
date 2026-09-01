package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

type fakeTagRepository struct {
	createFn func(context.Context, *domain.Tag) (*domain.Tag, error)
	listFn   func(context.Context) ([]domain.Tag, error)
	getByIDFn func(context.Context, uuid.UUID) (*domain.Tag, error)
	updateFn func(context.Context, uuid.UUID, string, domain.TagColor) (*domain.Tag, error)
	deleteFn func(context.Context, uuid.UUID) error
}

func (f *fakeTagRepository) Create(ctx context.Context, tag *domain.Tag) (*domain.Tag, error) {
	if f.createFn != nil {
		return f.createFn(ctx, tag)
	}
	return nil, nil
}

func (f *fakeTagRepository) List(ctx context.Context) ([]domain.Tag, error) {
	if f.listFn != nil {
		return f.listFn(ctx)
	}
	return nil, nil
}

func (f *fakeTagRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeTagRepository) Update(ctx context.Context, id uuid.UUID, name string, color domain.TagColor) (*domain.Tag, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, id, name, color)
	}
	return nil, nil
}

func (f *fakeTagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func TestTagUsecase_CreateAndList(t *testing.T) {
	t.Parallel()

	t.Run("create success", func(t *testing.T) {
		repo := &fakeTagRepository{
			createFn: func(_ context.Context, tag *domain.Tag) (*domain.Tag, error) {
				if tag == nil { t.Fatal("tag is nil") }
				if tag.Name != "重要" { t.Fatalf("name: got %s, want %s", tag.Name, "重要") }
				if tag.Color != domain.TagColorRed { t.Fatalf("color: got %s, want %s", tag.Color, domain.TagColorRed) }
				return &domain.Tag{ID: uuid.New(), Name: tag.Name, Color: tag.Color, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			},
		}
		uc := usecase.NewTagUsecase(repo)
		got, err := uc.Create(context.Background(), usecase.CreateTagInput{Name: "重要", Color: string(domain.TagColorRed)})
		if err != nil { t.Fatal(err) }
		if got == nil || got.Name != "重要" { t.Fatalf("tag: got %+v, want name %q", got, "重要") }
	})

	t.Run("list success", func(t *testing.T) {
		repo := &fakeTagRepository{listFn: func(context.Context) ([]domain.Tag, error) {
			return []domain.Tag{{ID: uuid.New(), Name: "重要", Color: domain.TagColorRed, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
		}}
		uc := usecase.NewTagUsecase(repo)
		got, err := uc.List(context.Background())
		if err != nil { t.Fatal(err) }
		if len(got) != 1 { t.Fatalf("len(got): got %d, want 1", len(got)) }
	})
}

func TestTagUsecase_UpdateAndDelete(t *testing.T) {
	t.Parallel()

	tagID := uuid.New()
	updated := &domain.Tag{ID: tagID, Name: "緊急", Color: domain.TagColorOrange, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	t.Run("update success", func(t *testing.T) {
		repo := &fakeTagRepository{updateFn: func(_ context.Context, gotID uuid.UUID, name string, color domain.TagColor) (*domain.Tag, error) {
			if gotID != tagID { t.Fatalf("id: got %v, want %v", gotID, tagID) }
			if name != "緊急" { t.Fatalf("name: got %s, want %s", name, "緊急") }
			if color != domain.TagColorOrange { t.Fatalf("color: got %s, want %s", color, domain.TagColorOrange) }
			return updated, nil
		}}
		uc := usecase.NewTagUsecase(repo)
		got, err := uc.Update(context.Background(), usecase.UpdateTagInput{TagID: tagID, Name: "緊急", Color: string(domain.TagColorOrange)})
		if err != nil { t.Fatal(err) }
		if got == nil || got.Name != "緊急" { t.Fatalf("tag: got %+v, want name %q", got, "緊急") }
	})

	t.Run("delete success", func(t *testing.T) {
		repo := &fakeTagRepository{deleteFn: func(_ context.Context, gotID uuid.UUID) error {
			if gotID != tagID { t.Fatalf("id: got %v, want %v", gotID, tagID) }
			return nil
		}}
		uc := usecase.NewTagUsecase(repo)
		if err := uc.Delete(context.Background(), tagID); err != nil { t.Fatal(err) }
	})

	t.Run("invalid create", func(t *testing.T) {
		uc := usecase.NewTagUsecase(&fakeTagRepository{})
		_, err := uc.Create(context.Background(), usecase.CreateTagInput{Name: "", Color: string(domain.TagColorRed)})
		if !errors.Is(err, domain.ErrInvalidTag) { t.Fatalf("err: got %v, want %v", err, domain.ErrInvalidTag) }
	})
}
