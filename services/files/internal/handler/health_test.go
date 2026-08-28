package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// fakePinger は Pinger のテスト用フェイク.
type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestHealthHandler_Live(t *testing.T) {
	t.Parallel()
	h := NewHealthHandler(fakePinger{})

	rec := invoke(t, h.Live, http.MethodGet, "/healthz")

	if rec.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Errorf("body: got %s, want %s", strings.TrimSpace(rec.Body.String()), `{"status":"ok"}`)
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pingErr  error
		wantCode int
		wantBody string
	}{
		{name: "DB 到達可能なら 200", pingErr: nil, wantCode: http.StatusOK, wantBody: `{"status":"ok"}`},
		{name: "DB 到達不可なら 503", pingErr: errors.New("connection refused"), wantCode: http.StatusServiceUnavailable, wantBody: `{"status":"unavailable"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewHealthHandler(fakePinger{err: tt.pingErr})

			rec := invoke(t, h.Ready, http.MethodGet, "/readyz")

			if rec.Code != tt.wantCode {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantCode)
			}
			if strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Errorf("body: got %s, want %s", strings.TrimSpace(rec.Body.String()), tt.wantBody)
			}
		})
	}
}

// invoke は echo ハンドラを単体で呼び出してレスポンスを記録する.
func invoke(t *testing.T, h echo.HandlerFunc, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	return rec
}
