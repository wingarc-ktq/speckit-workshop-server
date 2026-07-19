package authjwt_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
)

func TestVerifier_Verify_Success(t *testing.T) {
	t.Parallel()
	privPEM, pubPEM := genPEM(t)
	userID := uuid.New()
	tok := signRS256(t, privPEM, jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	v, err := authjwt.NewVerifier(pubPEM)
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != userID {
		t.Errorf("userID: got %v, want %v", got, userID)
	}
}

func TestVerifier_Verify_Failures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T) (verifier *authjwt.Verifier, tok string)
	}{
		{
			name: "期限切れトークンは ErrInvalidToken",
			setup: func(t *testing.T) (*authjwt.Verifier, string) {
				privPEM, pubPEM := genPEM(t)
				tok := signRS256(t, privPEM, jwt.MapClaims{
					"sub": uuid.New().String(),
					"exp": time.Now().Add(-time.Hour).Unix(),
				})
				return newVerifier(t, pubPEM), tok
			},
		},
		{
			name: "別の鍵で署名されたトークンは ErrInvalidToken",
			setup: func(t *testing.T) (*authjwt.Verifier, string) {
				priv1, _ := genPEM(t)
				_, pub2 := genPEM(t)
				tok := signRS256(t, priv1, jwt.MapClaims{
					"sub": uuid.New().String(),
					"exp": time.Now().Add(time.Hour).Unix(),
				})
				return newVerifier(t, pub2), tok
			},
		},
		{
			name: "壊れた文字列は ErrInvalidToken",
			setup: func(t *testing.T) (*authjwt.Verifier, string) {
				_, pubPEM := genPEM(t)
				return newVerifier(t, pubPEM), "not-a-jwt-token"
			},
		},
		{
			name: "sub が UUID でないトークンは ErrInvalidToken",
			setup: func(t *testing.T) (*authjwt.Verifier, string) {
				privPEM, pubPEM := genPEM(t)
				tok := signRS256(t, privPEM, jwt.MapClaims{
					"sub": "not-a-uuid",
					"exp": time.Now().Add(time.Hour).Unix(),
				})
				return newVerifier(t, pubPEM), tok
			},
		},
		{
			name: "RS256 以外で署名されたトークンは ErrInvalidToken",
			setup: func(t *testing.T) (*authjwt.Verifier, string) {
				_, pubPEM := genPEM(t)
				claims := jwt.MapClaims{
					"sub": uuid.New().String(),
					"exp": time.Now().Add(time.Hour).Unix(),
				}
				tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
				if err != nil {
					t.Fatal(err)
				}
				return newVerifier(t, pubPEM), tok
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			verifier, tok := tt.setup(t)
			if _, err := verifier.Verify(tok); err != authjwt.ErrInvalidToken {
				t.Errorf("err: got %v, want %v", err, authjwt.ErrInvalidToken)
			}
		})
	}
}

// newVerifier は公開鍵 PEM から Verifier を生成する (失敗時は t.Fatal).
func newVerifier(t *testing.T, pubPEM []byte) *authjwt.Verifier {
	t.Helper()
	v, err := authjwt.NewVerifier(pubPEM)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// signRS256 は秘密鍵 PEM で claims を RS256 署名したトークンを返す.
func signRS256(t *testing.T, privPEM []byte, claims jwt.MapClaims) string {
	t.Helper()
	priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// genPEM はテスト用の RSA 鍵ペアを PEM (PKCS#8 / PKIX) で生成する.
func genPEM(t *testing.T) (privPEM, pubPEM []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	privPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privPEM, pubPEM
}
