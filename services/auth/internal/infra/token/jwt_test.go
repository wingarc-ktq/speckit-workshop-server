package token_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/token"
)

func TestJWT_IssueAndVerify(t *testing.T) {
	t.Parallel()
	privPEM, pubPEM := genPEM(t)

	j, err := token.New(privPEM, pubPEM, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	userID := uuid.New()
	tok, expiresIn, err := j.Issue(userID)
	if err != nil {
		t.Fatal(err)
	}
	if expiresIn != 3600 {
		t.Errorf("expiresIn: got %d, want 3600", expiresIn)
	}
	if tok == "" {
		t.Error("tok should not be empty")
	}

	got, err := j.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if got != userID {
		t.Errorf("userID: got %v, want %v", got, userID)
	}
}

func TestJWT_VerifyOnly_CannotIssue(t *testing.T) {
	t.Parallel()
	_, pubPEM := genPEM(t)

	verifier, err := token.New(nil, pubPEM, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = verifier.Issue(uuid.New())
	if err == nil {
		t.Error("Issue: got nil, want error")
	}
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
