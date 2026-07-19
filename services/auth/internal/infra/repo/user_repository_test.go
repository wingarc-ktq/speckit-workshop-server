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

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/repo"
)

func TestUserRepository_CreateAndFind(t *testing.T) {
	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewUserRepository(pool)

	user := newUser("taro@example.com")
	if err := r.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt: got zero, want non-zero after Create")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("UpdatedAt: got zero, want non-zero after Create")
	}

	tests := []struct {
		name string
		find func() (*domain.User, error)
	}{
		{
			name: "IDで検索",
			find: func() (*domain.User, error) { return r.FindByID(ctx, user.ID) },
		},
		{
			name: "メールで検索は大文字小文字を区別しない",
			find: func() (*domain.User, error) { return r.FindByEmail(ctx, "Taro@Example.COM") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.find()
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != user.ID {
				t.Errorf("ID: got %v, want %v", got.ID, user.ID)
			}
			if got.Email != user.Email {
				t.Errorf("email: got %s, want %s", got.Email, user.Email)
			}
		})
	}
}

func TestUserRepository_DuplicateEmail(t *testing.T) {
	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewUserRepository(pool)

	if err := r.Create(ctx, newUser("dup@example.com")); err != nil {
		t.Fatal(err)
	}

	err := r.Create(ctx, newUser("dup@example.com"))
	if !errors.Is(err, domain.ErrEmailAlreadyTaken) {
		t.Errorf("err: got %v, want %v", err, domain.ErrEmailAlreadyTaken)
	}
}

func TestUserRepository_NotFound(t *testing.T) {
	pool, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	r := repo.NewUserRepository(pool)

	tests := []struct {
		name string
		find func() error
	}{
		{
			name: "IDで検索",
			find: func() error { _, err := r.FindByID(ctx, uuid.New()); return err },
		},
		{
			name: "メールで検索",
			find: func() error { _, err := r.FindByEmail(ctx, "ghost@example.com"); return err },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.find(); !errors.Is(err, domain.ErrUserNotFound) {
				t.Errorf("err: got %v, want %v", err, domain.ErrUserNotFound)
			}
		})
	}
}

// setupPostgres は testcontainers-go で実 PostgreSQL を起動し、
// users テーブルのマイグレーションを適用したプールとクリーンアップ関数を返す.
func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("auth"),
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

	ddl, err := os.ReadFile("../../../migrations/000001_create_users_table.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, string(ddl))
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}
	return pool, cleanup
}

func newUser(email string) *domain.User {
	return &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "$2a$10$dummyhashdummyhashdummyhashdummyhashdummyha",
		Name:         "テスト ユーザー",
	}
}
