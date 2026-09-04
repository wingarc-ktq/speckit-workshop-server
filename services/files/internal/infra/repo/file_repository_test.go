//go:build integration

package repo_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
)

func TestFileRepository_CreateAndUpdate(t *testing.T) {
	pool, cleanup := setupFilesPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewFileRepository(pool)
	ownerID := uuid.New()
	fileID := uuid.New()
	desc := "初回説明"
	file := &domain.File{
		ID:          fileID,
		OwnerUserID: ownerID,
		Name:        "report.pdf",
		Size:        128,
		MIMEType:    "application/pdf",
		Description: &desc,
		UploadedAt:  time.Now(),
	}

	created, err := r.Create(ctx, file)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != fileID {
		t.Fatalf("created ID: got %v, want %v", created.ID, fileID)
	}

	items, err := r.List(ctx, ownerID, "report", 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items): got %d, want 1", len(items))
	}

	got, err := r.GetByID(ctx, ownerID, fileID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "report.pdf" {
		t.Fatalf("name: got %s, want %s", got.Name, "report.pdf")
	}

	newName := "report_v2.pdf"
	newDesc := "更新後の説明"
	tagID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO tags (id, name, color) VALUES ($1, $2, $3)`, tagID, "重要", "red"); err != nil {
		t.Fatal(err)
	}
	updated, err := r.UpdateMetadata(ctx, ownerID, fileID, newName, &newDesc, []uuid.UUID{tagID})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != newName {
		t.Fatalf("updated name: got %s, want %s", updated.Name, newName)
	}
	if updated.Description == nil || *updated.Description != newDesc {
		t.Fatalf("updated description: got %v, want %q", updated.Description, newDesc)
	}
	if len(updated.TagIDs) != 1 {
		t.Fatalf("tag count: got %d, want 1", len(updated.TagIDs))
	}
	got, err = r.GetByID(ctx, ownerID, fileID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.TagIDs) != 1 || got.TagIDs[0] != tagID {
		t.Fatalf("stored tag IDs: got %v, want [%v]", got.TagIDs, tagID)
	}
}

func TestFileRepository_DuplicateAndDelete(t *testing.T) {
	pool, cleanup := setupFilesPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewFileRepository(pool)
	ownerID := uuid.New()
	file := &domain.File{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
		Name:        "dup.pdf",
		Size:        128,
		MIMEType:    "application/pdf",
	}

	if _, err := r.Create(ctx, file); err != nil {
		t.Fatal(err)
	}

	second, err := r.Create(ctx, &domain.File{
		ID:          uuid.New(),
		OwnerUserID: ownerID,
		Name:        "dup.pdf",
		Size:        256,
		MIMEType:    "application/pdf",
	})
	if err != nil {
		t.Fatalf("duplicate file name should be accepted: %v", err)
	}
	if second.ID == file.ID {
		t.Fatalf("duplicate file should have a distinct ID: got %v", second.ID)
	}

	if err := r.Delete(ctx, ownerID, file.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetByID(ctx, ownerID, file.ID); !errors.Is(err, domain.ErrFileNotFound) {
		t.Fatalf("after delete err: got %v, want %v", err, domain.ErrFileNotFound)
	}
}

func setupFilesPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("files"),
		postgres.WithUsername("workshop"),
		postgres.WithPassword("workshop"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	for _, migration := range []string{
		"../../../migrations/000001_create_files_table.up.sql",
		"../../../migrations/000002_create_tags_table.up.sql",
		"../../../migrations/000003_allow_duplicate_and_empty_files.up.sql",
		"../../../migrations/000004_create_file_tags_table.up.sql",
	} {
		ddl, err := os.ReadFile(migration)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(ddl)); err != nil {
			t.Fatal(err)
		}
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}
	return pool, cleanup
}
