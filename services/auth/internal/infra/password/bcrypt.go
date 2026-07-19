// Package password はユースケースのポート実装を提供する.
package password

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase"
)

// Bcrypt は bcrypt を用いた PasswordHasher の具象実装.
type Bcrypt struct {
	cost int
}

// NewBcrypt は bcrypt.DefaultCost を用いる Bcrypt を生成する.
func NewBcrypt() *Bcrypt {
	return &Bcrypt{cost: bcrypt.DefaultCost}
}

// インターフェース実装の静的チェック.
var _ usecase.PasswordHasher = (*Bcrypt)(nil)

// Hash は平文パスワードを bcrypt ハッシュに変換する.
func (b *Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare はハッシュと平文パスワードを照合する.
func (b *Bcrypt) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
