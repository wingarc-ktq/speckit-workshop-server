// Package usecase の外部依存（ポート）を定義する.
package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/domain"
)

// UserRepository はユーザーストアの抽象.
// 具象は internal/infra/repo で実装される.
//
//go:generate mockgen -source=port.go -destination=mock/port_mock.go -package=mock
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

// PasswordHasher はパスワードのハッシュ化・照合の抽象.
// 具象 (bcrypt など) は internal/infra で実装される.
type PasswordHasher interface {
	// Hash は平文パスワードをハッシュ化する.
	Hash(password string) (string, error)
	// Compare はハッシュと平文パスワードを照合する. 不一致は error を返す.
	Compare(hash, password string) error
}

// TokenIssuer はアクセストークンの発行の抽象.
// 署名方式 (RS256 など) は具象に隠蔽される.
type TokenIssuer interface {
	// Issue は userID を subject に持つトークンを発行し、
	// トークン文字列と有効期限（秒）を返す.
	Issue(userID uuid.UUID) (token string, expiresIn int, err error)
}
