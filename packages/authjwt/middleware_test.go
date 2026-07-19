package authjwt_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
)

// fakeVerifier は固定の結果を返す TokenVerifier. ミドルウェア単体の挙動確認に使う.
type fakeVerifier struct {
	userID uuid.UUID
	err    error
}

func (f fakeVerifier) Verify(string) (uuid.UUID, error) { return f.userID, f.err }

func TestMiddleware_ValidToken_StoresUserID(t *testing.T) {
	t.Parallel()
	want := uuid.New()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var got uuid.UUID
	next := func(c echo.Context) error {
		id, ok := authjwt.UserIDFromContext(c)
		if !ok {
			t.Error("userID not found in context")
		}
		got = id
		return c.NoContent(http.StatusOK)
	}

	h := authjwt.Middleware(fakeVerifier{userID: want})(next)
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if got != want {
		t.Errorf("userID: got %v, want %v", got, want)
	}
}

func TestMiddleware_Unauthorized(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		authHeader string
		verifyErr  error
	}{
		{name: "Authorization ヘッダ無し", authHeader: ""},
		{name: "Bearer プレフィックス無し", authHeader: "token-without-bearer"},
		{name: "検証失敗", authHeader: "Bearer bad-token", verifyErr: authjwt.ErrInvalidToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			called := false
			next := func(c echo.Context) error {
				called = true
				return c.NoContent(http.StatusOK)
			}

			h := authjwt.Middleware(fakeVerifier{err: tt.verifyErr})(next)
			if err := h(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if called {
				t.Error("next handler should not be called on unauthorized request")
			}
		})
	}
}
