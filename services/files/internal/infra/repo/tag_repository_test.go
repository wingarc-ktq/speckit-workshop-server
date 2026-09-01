//go:build integration

package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
)

func TestTagRepository_CRUD(t *testing.T) {
	pool, cleanup := setupFilesPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewTagRepository(pool)
	tag := &domain.Tag{
		ID:        uuid.New(),
		Name:      "重要",
		Color:     domain.TagColorRed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	created, err := r.Create(ctx, tag)
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "重要" {
		t.Fatalf("name: got %s, want %s", created.Name, "重要")
	}

	items, err := r.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(tags): got %d, want 1", len(items))
	}

	found, err := r.GetByID(ctx, tag.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Color != domain.TagColorRed {
		t.Fatalf("color: got %s, want %s", found.Color, domain.TagColorRed)
	}

	updated, err := r.Update(ctx, tag.ID, "至急", domain.TagColorBlue)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "至急" {
		t.Fatalf("updated name: got %s, want %s", updated.Name, "至急")
	}
	if updated.Color != domain.TagColorBlue {
		t.Fatalf("updated color: got %s, want %s", updated.Color, domain.TagColorBlue)
	}

	if err := r.Delete(ctx, tag.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetByID(ctx, tag.ID); !errors.Is(err, domain.ErrTagNotFound) {
		t.Fatalf("after delete err: got %v, want %v", err, domain.ErrTagNotFound)
	}
}

func TestTagRepository_DuplicateName(t *testing.T) {
	pool, cleanup := setupFilesPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewTagRepository(pool)

	first := &domain.Tag{ID: uuid.New(), Name: "重複防止", Color: domain.TagColorGreen}
	if _, err := r.Create(ctx, first); err != nil {
		t.Fatal(err)
	}

	_, err := r.Create(ctx, &domain.Tag{ID: uuid.New(), Name: "重複防止", Color: domain.TagColorYellow})
	if !errors.Is(err, domain.ErrDuplicateTagName) {
		t.Fatalf("duplicate err: got %v, want %v", err, domain.ErrDuplicateTagName)
	}
}
