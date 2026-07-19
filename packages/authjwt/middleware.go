package authjwt

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// TokenVerifier はアクセストークン検証の抽象.
// 具象 (Verifier など) を差し替え可能にし、Middleware をトークン技術から独立させる.
type TokenVerifier interface {
	// Verify はトークンを検証し、subject (userID) を返す.
	// 無効・期限切れの場合は error を返す.
	Verify(token string) (userID uuid.UUID, err error)
}

// userIDContextKey は echo.Context に検証済み user_id を格納する際のキー.
const userIDContextKey = "userID"

// Middleware は Authorization: Bearer ヘッダのトークンを TokenVerifier で検証する echo ミドルウェアを返す.
// 検証に成功すると userID (uuid.UUID) を echo.Context に格納し、後続ハンドラへ渡す.
// 未指定・無効・期限切れの場合は 401 UNAUTHORIZED を返す.
//
// 署名方式 (RS256 など) の詳細は verifier の具象実装に隠蔽されており、
// このミドルウェアはトークン技術に依存しない.
func Middleware(verifier TokenVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			const prefix = "Bearer "
			auth := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(auth, prefix) {
				return unauthorized(c)
			}

			userID, err := verifier.Verify(strings.TrimPrefix(auth, prefix))
			if err != nil {
				return unauthorized(c)
			}

			c.Set(userIDContextKey, userID)
			return next(c)
		}
	}
}

// UserIDFromContext は Middleware が格納した userID を取り出す.
func UserIDFromContext(c echo.Context) (uuid.UUID, bool) {
	id, ok := c.Get(userIDContextKey).(uuid.UUID)
	return id, ok
}

// errorBody は全サービス共通のエラーレスポンス形状 (code + message).
// 各サービスの生成型 (gen.ErrorResponse 等) に依存せず同一 JSON を出力する.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// unauthorized は 401 UNAUTHORIZED レスポンスを返す.
func unauthorized(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, errorBody{
		Code:    "UNAUTHORIZED",
		Message: "認証が必要です",
	})
}
