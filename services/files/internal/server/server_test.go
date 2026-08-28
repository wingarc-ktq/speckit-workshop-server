package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
)

func TestNewEcho_OpenAPIValidation(t *testing.T) {
	t.Parallel()

	e, err := newEcho(nil, t.TempDir(), newTestVerifier(t))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{
			// OpenAPI 検証ミドルウェアは JWT ミドルウェアより先に (echo のグローバル
			// ミドルウェアとして) 実行されるため、Authorization ヘッダが無くても
			// パラメータ形式の誤りは 401 ではなく 400 として検出できる.
			name:       "page が整数でない一覧取得は 400 VALIDATION_ERROR",
			method:     http.MethodGet,
			path:       basePath + "/files?page=abc",
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "未定義パスの api は 404 NOT_FOUND",
			method:     http.MethodGet,
			path:       basePath + "/does-not-exist",
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tt.method, tt.path, nil)
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

	e, err := newEcho(nil, t.TempDir(), newTestVerifier(t))
	if err != nil {
		t.Fatal(err)
	}

	// files の 4 操作はすべて認証必須（spec.md FR-020）。
	// ここでは代表として一覧取得のみ確認する（他の3操作も newEcho で
	// 同じ authMiddleware を割り当てているため、経路は同一）.
	t.Run("トークン無しで一覧取得すると 401 UNAUTHORIZED が返る", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, basePath+"/files", nil)
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

	e, err := newEcho(nil, t.TempDir(), newTestVerifier(t))
	if err != nil {
		t.Fatal(err)
	}

	// /healthz・/readyz は認証もOpenAPI検証も掛からない運用エンドポイント
	// （理由は newEcho 内のコメント、および spec.md FR-023 を参照）。
	// ここでは Authorization ヘッダを付けずに 200 が返ることで、それを確認する.
	t.Run("/healthz はJWT無し・OpenAPI検証MWを素通しして 200 OK が返る", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status code: got %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

// newTestVerifier はテスト用の RS256 検証器を生成する.
// Files は秘密鍵を持たないため、authjwt.NewVerifier に渡すのは公開鍵のみでよい.
func newTestVerifier(t *testing.T) *authjwt.Verifier {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	v, err := authjwt.NewVerifier(pubPEM)
	if err != nil {
		t.Fatal(err)
	}
	return v
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
