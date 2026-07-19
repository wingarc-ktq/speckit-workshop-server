package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/token"
)

func TestNewEcho_OpenAPIValidation(t *testing.T) {
	t.Parallel()

	e, err := newEcho(nil, newTestToken(t))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "空ボディの login は 400 VALIDATION_ERROR",
			method:     http.MethodPost,
			path:       basePath + "/auth/login",
			body:       "{}",
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "未定義パスの api は 404 NOT_FOUND",
			method:     http.MethodGet,
			path:       basePath + "/does-not-exist",
			body:       "",
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if code := decodeCode(t, rec); code != tt.wantCode {
				t.Errorf("code: got %s, want %s", code, tt.wantCode)
			}
		})
	}
}

func TestNewEcho_JWTAuth(t *testing.T) {
	t.Parallel()

	e, err := newEcho(nil, newTestToken(t))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("トークン無しで /auth/me にアクセスすると 401 UNAUTHORIZED が返る", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, basePath+"/auth/me", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status code: got %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		if code := decodeCode(t, rec); code != "UNAUTHORIZED" {
			t.Errorf("code: got %s, want UNAUTHORIZED", code)
		}
	})
}

func TestNewEcho_OperationalEndpoints(t *testing.T) {
	t.Parallel()

	e, err := newEcho(nil, newTestToken(t))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("/healthz は OpenAPI検証 MW を素通しして 200 OK が返る", func(t *testing.T) {
		// 運用エンドポイントは OpenAPI 契約外。Skipper が効かないと 404 NOT_FOUND になる.
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status code: got %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

// newTestToken はテスト用の RS256 トークンサービスを生成する.
func newTestToken(t *testing.T) *token.JWT {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	tk, err := token.New(privPEM, pubPEM, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

// decodeCode はレスポンスボディ (gen.ErrorResponse) の code フィールドを取り出す.
func decodeCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Code
}
