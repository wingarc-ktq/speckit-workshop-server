// Package usecase は認証に関するアプリケーションロジック（登録・ログイン・自身の取得）を実装する.
package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/domain"
)

// AuthUsecase は認証に関するアプリケーションの入力ポート（ユースケース契約）.
// handler などの外側アダプタはこの interface に依存する（依存は内向き）.
//
//go:generate mockgen -source=auth_usecase.go -destination=mock/auth_usecase_mock.go -package=mock
type AuthUsecase interface {
	Register(ctx context.Context, in RegisterInput) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*LoginOutput, error)
	Me(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

// authUsecase は AuthUsecase の実装.
// パスワードハッシュ化・トークン発行はポート経由で注入され、
// 自身は bcrypt や JWT といった具体技術に依存しない.
type authUsecase struct {
	users  UserRepository
	hasher PasswordHasher
	tokens TokenIssuer
}

// NewAuthUsecase は AuthUsecase を生成する.
func NewAuthUsecase(users UserRepository, hasher PasswordHasher, tokens TokenIssuer) AuthUsecase {
	return &authUsecase{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

// RegisterInput は ユーザー登録の入力.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// Register は新しいユーザーを登録する.
func (u *authUsecase) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	// メールアドレスは小文字に正規化する（大文字小文字を区別しない）.
	email := strings.ToLower(in.Email)

	if existing, err := u.users.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, domain.ErrEmailAlreadyTaken
	} else if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	hash, err := u.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		Name:         in.Name,
	}
	if err := u.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// LoginOutput はログイン結果.
type LoginOutput struct {
	AccessToken string
	ExpiresIn   int
	User        *domain.User
}

// Login はメール+パスワードで JWT を発行する.
func (u *authUsecase) Login(ctx context.Context, email, password string) (*LoginOutput, error) {
	user, err := u.users.FindByEmail(ctx, strings.ToLower(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredential
		}
		return nil, err
	}
	if err := u.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, domain.ErrInvalidCredential
	}

	token, expiresIn, err := u.tokens.Issue(user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginOutput{
		AccessToken: token,
		ExpiresIn:   expiresIn,
		User:        user,
	}, nil
}

// Me は JWT 検証済みユーザーの情報を取得する.
func (u *authUsecase) Me(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return u.users.FindByID(ctx, userID)
}
