// Package domain は Auth サービスのドメインモデルとドメインエラーを定義する.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User はドメイン層のユーザーモデル.
// インフラ層 (DB) の表現とは独立して定義する.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ドメインエラー
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyTaken = errors.New("email already taken")
	ErrInvalidCredential = errors.New("invalid email or password")
)
