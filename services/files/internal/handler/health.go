// Package handler は HTTP ハンドラを提供する.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// readyTimeout は readiness チェック (DB ping) のタイムアウト.
const readyTimeout = 2 * time.Second

// Pinger は readiness 判定に使う依存（DB 等）への疎通確認の抽象.
// *pgxpool.Pool が満たす.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler は運用用のヘルスチェックエンドポイントを提供する.
// ビジネス API（OpenAPI 契約）とは別物として扱い、/api/v1 配下には置かない.
type HealthHandler struct {
	db Pinger
}

// NewHealthHandler は HealthHandler を生成する.
func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// healthStatus はヘルスチェックのレスポンスボディ.
type healthStatus struct {
	Status string `json:"status"`
}

// Live は liveness probe. プロセスが応答可能なら 200 を返す（依存はチェックしない）.
func (h *HealthHandler) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, healthStatus{Status: "ok"})
}

// Ready は readiness probe. DB に到達できれば 200、できなければ 503 を返す.
func (h *HealthHandler) Ready(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), readyTimeout)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		return c.JSON(http.StatusServiceUnavailable, healthStatus{Status: "unavailable"})
	}
	return c.JSON(http.StatusOK, healthStatus{Status: "ok"})
}
