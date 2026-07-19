// Package handler は OpenAPI で生成された gen.ServerInterface を実装する
// HTTP ハンドラ層を提供する.
//
// リクエストの形式・制約の検証は cmd/server で組み込む OpenAPI 検証ミドルウェア
// (OapiRequestValidator) が担うため、ハンドラはビジネス的な変換・整形に専念する.
package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase"
)

// AuthHandler は認証 API の HTTP ハンドラ.
// gen.ServerInterface を実装する.
type AuthHandler struct {
	uc usecase.AuthUsecase
}

// NewAuthHandler は AuthHandler を生成する.
func NewAuthHandler(uc usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// gen.ServerInterface 実装の静的チェック.
var _ gen.ServerInterface = (*AuthHandler)(nil)

// RegisterUser はユーザー登録 (POST /auth/register).
func (h *AuthHandler) RegisterUser(ctx echo.Context) error {
	var req gen.RegisterRequest
	if err := ctx.Bind(&req); err != nil {
		return validationError(ctx, "リクエストボディが不正です")
	}

	user, err := h.uc.Register(ctx.Request().Context(), usecase.RegisterInput{
		Email:    string(req.Email),
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		return h.errorResponse(ctx, err)
	}

	return ctx.JSON(http.StatusCreated, gen.UserResponse{User: toGenUser(user)})
}

// LoginUser はログイン (POST /auth/login).
func (h *AuthHandler) LoginUser(ctx echo.Context) error {
	var req gen.LoginRequest
	if err := ctx.Bind(&req); err != nil {
		return validationError(ctx, "リクエストボディが不正です")
	}

	out, err := h.uc.Login(ctx.Request().Context(), string(req.Email), req.Password)
	if err != nil {
		return h.errorResponse(ctx, err)
	}

	return ctx.JSON(http.StatusOK, gen.LoginResponse{
		AccessToken: out.AccessToken,
		TokenType:   gen.Bearer,
		ExpiresIn:   out.ExpiresIn,
		User:        toGenUser(out.User),
	})
}

// GetCurrentUser は認証中ユーザー取得 (GET /auth/me).
// authjwt.Middleware が格納した userID を echo.Context から取得して使う.
func (h *AuthHandler) GetCurrentUser(ctx echo.Context) error {
	userID, ok := authjwt.UserIDFromContext(ctx)
	if !ok {
		// authjwt.Middleware が userID を格納するため通常は到達しない防御的分岐.
		// 認証情報が無い状態として ErrUserNotFound の 401 マッピングに集約する.
		return h.errorResponse(ctx, domain.ErrUserNotFound)
	}

	user, err := h.uc.Me(ctx.Request().Context(), userID)
	if err != nil {
		return h.errorResponse(ctx, err)
	}

	return ctx.JSON(http.StatusOK, gen.UserResponse{User: toGenUser(user)})
}

// validationError は 400 バリデーションエラーレスポンスを返す.
func validationError(ctx echo.Context, msg string) error {
	return ctx.JSON(http.StatusBadRequest, gen.ErrorResponse{
		Code:    "VALIDATION_ERROR",
		Message: msg,
	})
}

// toGenUser は domain.User を API レスポンス用の gen.User に変換する.
func toGenUser(u *domain.User) gen.User {
	return gen.User{
		Id:        openapi_types.UUID(u.ID),
		Email:     openapi_types.Email(u.Email),
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// errorResponse はドメインエラーを HTTP ステータス + gen.ErrorResponse にマッピングする.
func (h *AuthHandler) errorResponse(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrEmailAlreadyTaken):
		return ctx.JSON(http.StatusConflict, gen.ErrorResponse{
			Code:    "EMAIL_ALREADY_TAKEN",
			Message: "このメールアドレスは既に登録されています",
		})
	case errors.Is(err, domain.ErrInvalidCredential):
		return ctx.JSON(http.StatusUnauthorized, gen.ErrorResponse{
			Code:    "AUTH_FAILED",
			Message: "メールアドレスまたはパスワードが正しくありません",
		})
	case errors.Is(err, domain.ErrUserNotFound):
		return ctx.JSON(http.StatusUnauthorized, gen.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "認証が必要です",
		})
	default:
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "内部エラーが発生しました",
		})
	}
}
