//go:build integration

package repo_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

func TestFileRepository_Create(t *testing.T) {
	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewFileRepository(pool)

	file := &domain.File{
		ID:          uuid.New(),
		Name:        "invoice.pdf",
		Size:        1024,
		MimeType:    "application/pdf",
		Description: "テスト用ファイル",
		StorageKey:  uuid.New().String(),
		TagIDs:      []uuid.UUID{uuid.New()},
	}

	if err := r.Create(ctx, file); err != nil {
		t.Fatal(err)
	}
	if file.UploadedAt.IsZero() {
		t.Error("UploadedAt: got zero, want non-zero after Create")
	}
}

func TestFileRepository_Create_EmptyTagIDs(t *testing.T) {
	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewFileRepository(pool)

	file := &domain.File{
		ID:         uuid.New(),
		Name:       "no-tags.pdf",
		Size:       10,
		MimeType:   "application/pdf",
		StorageKey: uuid.New().String(),
		TagIDs:     []uuid.UUID{},
	}

	if err := r.Create(ctx, file); err != nil {
		t.Fatal(err)
	}
}

func TestFileRepository_List(t *testing.T) {
	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewFileRepository(pool)

	tag := uuid.New()
	other := uuid.New()

	invoice := &domain.File{ID: uuid.New(), Name: "田中商事_請求書.pdf", Size: 100, MimeType: "application/pdf", StorageKey: uuid.New().String(), TagIDs: []uuid.UUID{tag}}
	report := &domain.File{ID: uuid.New(), Name: "月次報告書.docx", Size: 200, MimeType: "application/msword", StorageKey: uuid.New().String(), TagIDs: []uuid.UUID{other}}
	photo := &domain.File{ID: uuid.New(), Name: "写真.jpg", Size: 300, MimeType: "image/jpeg", StorageKey: uuid.New().String(), TagIDs: []uuid.UUID{}}

	// uploaded_at DESC でソートされるため、Create した順の逆順が期待値になる.
	for _, f := range []*domain.File{invoice, report, photo} {
		if err := r.Create(ctx, f); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name      string
		params    usecase.ListFilesParams
		wantIDs   []uuid.UUID
		wantTotal int64
	}{
		{
			name:      "デフォルトページングで全件取得",
			params:    usecase.ListFilesParams{Page: 1, Limit: 20},
			wantIDs:   []uuid.UUID{photo.ID, report.ID, invoice.ID},
			wantTotal: 3,
		},
		{
			name:      "キーワード検索の部分一致",
			params:    usecase.ListFilesParams{Page: 1, Limit: 20, Search: "請求書"},
			wantIDs:   []uuid.UUID{invoice.ID},
			wantTotal: 1,
		},
		{
			name:      "タグIDフィルタ",
			params:    usecase.ListFilesParams{Page: 1, Limit: 20, TagIDs: []uuid.UUID{tag}},
			wantIDs:   []uuid.UUID{invoice.ID},
			wantTotal: 1,
		},
		{
			name:      "ページネーション（1件ずつ2ページ目）",
			params:    usecase.ListFilesParams{Page: 2, Limit: 1},
			wantIDs:   []uuid.UUID{report.ID},
			wantTotal: 3,
		},
		{
			name:      "一致しない検索は0件",
			params:    usecase.ListFilesParams{Page: 1, Limit: 20, Search: "存在しないキーワード"},
			wantIDs:   nil,
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, total, err := r.List(ctx, tt.params)
			if err != nil {
				t.Fatal(err)
			}
			if total != tt.wantTotal {
				t.Errorf("total: got %d, want %d", total, tt.wantTotal)
			}
			if len(files) != len(tt.wantIDs) {
				t.Fatalf("files length: got %d, want %d", len(files), len(tt.wantIDs))
			}
			for i, f := range files {
				if f.ID != tt.wantIDs[i] {
					t.Errorf("files[%d].ID: got %v, want %v", i, f.ID, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestFileRepository_FindByID(t *testing.T) {
	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewFileRepository(pool)

	created := &domain.File{
		ID:          uuid.New(),
		Name:        "invoice.pdf",
		Size:        1024,
		MimeType:    "application/pdf",
		Description: "テスト用ファイル",
		StorageKey:  uuid.New().String(),
		TagIDs:      []uuid.UUID{uuid.New()},
	}
	if err := r.Create(ctx, created); err != nil {
		t.Fatal(err)
	}

	t.Run("存在するIDで検索できる", func(t *testing.T) {
		got, err := r.FindByID(ctx, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != created.ID {
			t.Errorf("ID: got %v, want %v", got.ID, created.ID)
		}
		if got.Name != created.Name {
			t.Errorf("name: got %s, want %s", got.Name, created.Name)
		}
		if len(got.TagIDs) != len(created.TagIDs) {
			t.Errorf("tagIDs length: got %d, want %d", len(got.TagIDs), len(created.TagIDs))
		}
	})

	t.Run("存在しないIDはErrFileNotFound", func(t *testing.T) {
		_, err := r.FindByID(ctx, uuid.New())
		if !errors.Is(err, domain.ErrFileNotFound) {
			t.Errorf("err: got %v, want %v", err, domain.ErrFileNotFound)
		}
	})
}

// setupPostgres は testcontainers-go で実 PostgreSQL を起動し、
// files テーブルのマイグレーションを適用したプールとクリーンアップ関数を返す.
func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
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

	ddl, err := os.ReadFile("../../../migrations/000001_create_files_table.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(ddl)); err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}
	return pool, cleanup
}
