// Package repo はユースケースのポート実装を提供する.
package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/repo/db"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase"
)

// uniqueViolationCode は PostgreSQL の UNIQUE 制約違反を示す SQLSTATE.
const uniqueViolationCode = "23505"

// UserRepository は sqlc 生成コード (db.Queries) をラップした
// usecase.UserRepository の具象実装.
type UserRepository struct {
	q *db.Queries
}

// NewUserRepository は *pgxpool.Pool（DBTX を満たす）を受け取り UserRepository を生成する.
func NewUserRepository(pool db.DBTX) *UserRepository {
	return &UserRepository{q: db.New(pool)}
}

// インターフェース実装の静的チェック.
var _ usecase.UserRepository = (*UserRepository)(nil)

// Create は新しいユーザーを永続化する.
// UNIQUE 制約違反は domain.ErrEmailAlreadyTaken にマッピングする.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		ID:           toPgUUID(user.ID),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return domain.ErrEmailAlreadyTaken
		}
		return err
	}
	// DB 側で生成された created_at / updated_at を書き戻す.
	user.CreatedAt = row.CreatedAt.Time
	user.UpdatedAt = row.UpdatedAt.Time
	return nil
}

// FindByID は ID でユーザーを検索する.
// 該当なしは domain.ErrUserNotFound にマッピングする.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := r.q.GetUserByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

// FindByEmail はメールアドレス (case-insensitive) でユーザーを検索する.
// 該当なしは domain.ErrUserNotFound にマッピングする.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

// toPgUUID は uuid.UUID を pgtype.UUID に変換する.
func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// toDomainUser は sqlc の db.User を domain.User に変換する.
func toDomainUser(u db.User) *domain.User {
	return &domain.User{
		ID:           uuid.UUID(u.ID.Bytes),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}
