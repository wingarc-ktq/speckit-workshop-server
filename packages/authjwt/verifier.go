// Package authjwt はサービス間で共有する JWT (RS256) の検証機能を提供する.
//
// 検証は公開鍵のみで行えるため、Auth サービスが署名 (Issue) したトークンを
// 公開鍵を持つ任意のサービスが検証できる. 署名 (秘密鍵) はこのパッケージに含めない
// — 広く import される共有コードに署名能力を持たせないことで、トークンを発行できるのは
// 秘密鍵を持つ Auth サービスだけ、という制約をコード構造で保証する.
package authjwt

import (
	"crypto/rsa"
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken はトークンが無効・期限切れ・改ざんされている場合のエラー.
var ErrInvalidToken = errors.New("invalid token")

// Verifier は公開鍵で RS256 トークンを検証する.
type Verifier struct {
	pub *rsa.PublicKey
}

// NewVerifier は PEM 形式の公開鍵から Verifier を生成する.
func NewVerifier(pubPEM []byte) (*Verifier, error) {
	if len(pubPEM) == 0 {
		return nil, errors.New("public key is required")
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		return nil, err
	}
	return &Verifier{pub: pub}, nil
}

// Verify は公開鍵でトークンを検証し、sub (userID) を返す.
func (v *Verifier) Verify(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidToken
		}
		return v.pub, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}
	sub, _ := claims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return userID, nil
}
