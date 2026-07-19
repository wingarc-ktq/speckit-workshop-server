// Package token はユースケースとハンドラのポート実装を提供する.
//
// 秘密鍵で署名 (Issue) し、公開鍵で検証 (Verify) する.
// 署名は Auth サービスのみが行い、検証は共有パッケージ authjwt.Verifier に委譲する
// (公開鍵を持つ任意のサービスが検証できる).
package token

import (
	"crypto/rsa"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase"
)

// JWT は RS256 で署名・検証するトークンサービスの具象実装.
// 検証 (Verify) は埋め込んだ authjwt.Verifier から昇格する.
type JWT struct {
	*authjwt.Verifier
	priv *rsa.PrivateKey
	ttl  time.Duration
	now  func() time.Time
}

// New は PEM 形式の秘密鍵・公開鍵と有効期限から JWT を生成する.
// privPEM が空の場合は検証専用 (Issue は使用不可) として生成する.
func New(privPEM, pubPEM []byte, ttl time.Duration) (*JWT, error) {
	verifier, err := authjwt.NewVerifier(pubPEM)
	if err != nil {
		return nil, err
	}

	j := &JWT{Verifier: verifier, ttl: ttl, now: time.Now}

	if len(privPEM) > 0 {
		priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
		if err != nil {
			return nil, err
		}
		j.priv = priv
	}
	return j, nil
}

// インターフェース実装の静的チェック.
var (
	_ usecase.TokenIssuer   = (*JWT)(nil)
	_ authjwt.TokenVerifier = (*JWT)(nil)
)

// Issue は userID を sub に持つ RS256 署名済みトークンを発行する.
func (j *JWT) Issue(userID uuid.UUID) (string, int, error) {
	if j.priv == nil {
		return "", 0, errors.New("private key is not configured; cannot issue token")
	}
	now := j.now().UTC()
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": now.Unix(),
		"exp": now.Add(j.ttl).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(j.priv)
	if err != nil {
		return "", 0, err
	}
	return signed, int(j.ttl.Seconds()), nil
}
