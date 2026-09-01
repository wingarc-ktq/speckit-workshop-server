package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const readyTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db Pinger
}

func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

type healthStatus struct {
	Status string `json:"status"`
}

func (h *HealthHandler) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, healthStatus{Status: "ok"})
}

func (h *HealthHandler) Ready(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), readyTimeout)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		return c.JSON(http.StatusServiceUnavailable, healthStatus{Status: "unavailable"})
	}
	return c.JSON(http.StatusOK, healthStatus{Status: "ok"})
}
