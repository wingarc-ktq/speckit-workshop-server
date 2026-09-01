package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
)

func TestJWTAuth_HTTPFlow(t *testing.T) {
	privPEM, pubPEM := genTestPEM(t)
	verifier, err := authjwt.NewVerifier(pubPEM)
	if err != nil {
		t.Fatal(err)
	}

	e, err := newEcho(nil, verifier, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	reached := false
	e.GET("/auth-check", func(c echo.Context) error {
		reached = true
		_, ok := authjwt.UserIDFromContext(c)
		if !ok {
			return c.NoContent(http.StatusUnauthorized)
		}
		return c.NoContent(http.StatusOK)
	}, authjwt.Middleware(verifier))

	t.Run("Authorization header missing returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth-check", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		if reached {
			t.Fatal("handler should not be reached without auth header")
		}
	})

	t.Run("invalid JWT returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth-check", nil)
		req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		if reached {
			t.Fatal("handler should not be reached for invalid JWT")
		}
	})

	t.Run("valid JWT reaches handler", func(t *testing.T) {
		token := signRS256(t, privPEM, jwt.MapClaims{
			"sub": uuid.New().String(),
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		req := httptest.NewRequest(http.MethodGet, "/auth-check", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
		}
		if !reached {
			t.Fatal("handler should be reached when JWT is valid")
		}
	})
}

func genTestPEM(t *testing.T) (privPEM, pubPEM []byte) {
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
