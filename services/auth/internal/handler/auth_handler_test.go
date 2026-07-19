package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/mock/gomock"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase/mock"
)

// sampleUser はレスポンス整形の検証に使う固定ユーザー.
var sampleUser = &domain.User{
	ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	Email:     "taro@example.com",
	Name:      "田中太郎",
	CreatedAt: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
	UpdatedAt: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
}

func TestAuthHandler_RegisterUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		setup      func(*mock.MockAuthUsecase)
		wantStatus int
		wantCode   string // エラー時のみ検証
	}{
		{
			name: "成功なら 201 とユーザー情報",
			body: `{"email":"taro@example.com","password":"P@ssw0rd!","name":"田中太郎"}`,
			setup: func(m *mock.MockAuthUsecase) {
				// bind したリクエストが正しく RegisterInput に写像されることも検証する.
				m.EXPECT().Register(gomock.Any(), usecase.RegisterInput{
					Email:    "taro@example.com",
					Password: "P@ssw0rd!",
					Name:     "田中太郎",
				}).Return(sampleUser, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "メール重複なら 409 EMAIL_ALREADY_TAKEN",
			body: `{"email":"taro@example.com","password":"P@ssw0rd!","name":"田中太郎"}`,
			setup: func(m *mock.MockAuthUsecase) {
				m.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil, domain.ErrEmailAlreadyTaken)
			},
			wantStatus: http.StatusConflict,
			wantCode:   "EMAIL_ALREADY_TAKEN",
		},
		{
			name: "想定外エラーなら 500 INTERNAL_ERROR",
			body: `{"email":"taro@example.com","password":"P@ssw0rd!","name":"田中太郎"}`,
			setup: func(m *mock.MockAuthUsecase) {
				m.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil, errUnexpected)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
		{
			name:       "不正な JSON なら 400 VALIDATION_ERROR",
			body:       `{"email":`,
			setup:      func(*mock.MockAuthUsecase) {}, // usecase まで到達しない
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock.NewMockAuthUsecase(ctrl)
			tt.setup(uc)
			h := handler.NewAuthHandler(uc)
			c, rec := newContext(http.MethodPost, "/auth/register", tt.body)

			if err := h.RegisterUser(c); err != nil {
				t.Fatal(err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode != "" {
				if code := decodeError(t, rec).Code; code != tt.wantCode {
					t.Errorf("code: got %s, want %s", code, tt.wantCode)
				}
				return
			}
			user := decodeUser(t, rec)
			if user.Id != sampleUser.ID {
				t.Errorf("id: got %v, want %v", user.Id, sampleUser.ID)
			}
			if string(user.Email) != sampleUser.Email {
				t.Errorf("email: got %s, want %s", user.Email, sampleUser.Email)
			}
		})
	}
}

func TestAuthHandler_LoginUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(*mock.MockAuthUsecase)
		wantStatus int
		wantCode   string
	}{
		{
			name: "成功なら 200 と Bearer トークン",
			setup: func(m *mock.MockAuthUsecase) {
				m.EXPECT().Login(gomock.Any(), "taro@example.com", "P@ssw0rd!").
					Return(&usecase.LoginOutput{AccessToken: "token-abc", ExpiresIn: 3600, User: sampleUser}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "認証失敗なら 401 AUTH_FAILED",
			setup: func(m *mock.MockAuthUsecase) {
				m.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, domain.ErrInvalidCredential)
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "AUTH_FAILED",
		},
		{
			name: "想定外エラーなら 500 INTERNAL_ERROR",
			setup: func(m *mock.MockAuthUsecase) {
				m.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errUnexpected)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock.NewMockAuthUsecase(ctrl)
			tt.setup(uc)
			h := handler.NewAuthHandler(uc)
			c, rec := newContext(http.MethodPost, "/auth/login",
				`{"email":"taro@example.com","password":"P@ssw0rd!"}`)

			if err := h.LoginUser(c); err != nil {
				t.Fatal(err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode != "" {
				if code := decodeError(t, rec).Code; code != tt.wantCode {
					t.Errorf("code: got %s, want %s", code, tt.wantCode)
				}
				return
			}
			var resp gen.LoginResponse
			decodeJSON(t, rec, &resp)
			if resp.AccessToken != "token-abc" {
				t.Errorf("accessToken: got %s, want token-abc", resp.AccessToken)
			}
			if resp.TokenType != gen.Bearer {
				t.Errorf("tokenType: got %s, want %s", resp.TokenType, gen.Bearer)
			}
			if resp.ExpiresIn != 3600 {
				t.Errorf("expiresIn: got %d, want 3600", resp.ExpiresIn)
			}
		})
	}
}

func TestAuthHandler_GetCurrentUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setUserID  bool // authjwt.Middleware が userID を格納した状態を模す
		setup      func(*mock.MockAuthUsecase)
		wantStatus int
		wantCode   string
	}{
		{
			name:      "有効な userID なら 200 とユーザー情報",
			setUserID: true,
			setup: func(m *mock.MockAuthUsecase) {
				m.EXPECT().Me(gomock.Any(), sampleUser.ID).Return(sampleUser, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "userID 未格納なら Me を呼ばず 401 UNAUTHORIZED",
			setUserID:  false,
			setup:      func(*mock.MockAuthUsecase) {}, // usecase まで到達しない
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:      "ユーザー未存在なら 401 UNAUTHORIZED",
			setUserID: true,
			setup: func(m *mock.MockAuthUsecase) {
				m.EXPECT().Me(gomock.Any(), sampleUser.ID).Return(nil, domain.ErrUserNotFound)
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc := mock.NewMockAuthUsecase(ctrl)
			tt.setup(uc)
			h := handler.NewAuthHandler(uc)
			c, rec := newContext(http.MethodGet, "/auth/me", "")
			if tt.setUserID {
				// authjwt.Middleware が使う context キー ("userID") に合わせる.
				c.Set("userID", sampleUser.ID)
			}

			if err := h.GetCurrentUser(c); err != nil {
				t.Fatal(err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode != "" {
				if code := decodeError(t, rec).Code; code != tt.wantCode {
					t.Errorf("code: got %s, want %s", code, tt.wantCode)
				}
				return
			}
			if user := decodeUser(t, rec); string(user.Email) != sampleUser.Email {
				t.Errorf("email: got %s, want %s", user.Email, sampleUser.Email)
			}
		})
	}
}

// errUnexpected はドメインエラー以外（500 に落ちる系）を表すセンチネル.
var errUnexpected = errors.New("unexpected")

// newContext は JSON リクエストの echo.Context とレスポンスレコーダを組み立てる.
func newContext(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return echo.New().NewContext(req, rec), rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, rec.Body.String())
	}
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) gen.ErrorResponse {
	t.Helper()
	var e gen.ErrorResponse
	decodeJSON(t, rec, &e)
	return e
}

func decodeUser(t *testing.T, rec *httptest.ResponseRecorder) gen.User {
	t.Helper()
	var r gen.UserResponse
	decodeJSON(t, rec, &r)
	return r.User
}
