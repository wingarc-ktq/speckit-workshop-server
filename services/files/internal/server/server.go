package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/config"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/handler"
	repo "github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/storage"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
	migrationfs "github.com/wingarc-ktq/speckit-workshop-server/services/files/migrations"
)

const shutdownTimeout = 10 * time.Second
const basePath = "/api/v1"

// Run creates the app and starts the Echo server. The context is used for graceful shutdown.
func Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	if cfg.DatabaseURL != "" {
		if err := runMigrations(cfg.DatabaseURL); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
	}

	st, err := storage.NewLocalStorage(cfg.StorageRoot)
	if err != nil {
		return fmt.Errorf("new local storage: %w", err)
	}

	var fileRepo usecase.FileRepository = repo.NewInMemoryFileRepository()
	var dbPinger handler.Pinger
	if cfg.DatabaseURL != "" {
		pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("pgxpool: %w", err)
		}
		defer pool.Close()
		fileRepo = repo.NewFileRepository(pool)
		dbPinger = pool
	}

	uc := usecase.NewFileUsecase(fileRepo, st, cfg.MaxUploadBytes)
	h := handler.NewFilesHandler(uc)

	var verifier authjwt.TokenVerifier
	if len(cfg.JWTPublicKey) > 0 {
		v, err := authjwt.NewVerifier(cfg.JWTPublicKey)
		if err != nil {
			return fmt.Errorf("jwt verifier: %w", err)
		}
		verifier = v
	}

	e, err := newEcho(h, dbPinger, verifier)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := e.Shutdown(shutCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	log.Printf("Files service listening on :%s", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

func runMigrations(databaseURL string) error {
	src, err := iofs.New(migrationfs.FS, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}
	dbURL := "pgx5://" + strings.TrimPrefix(databaseURL, "postgres://")
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func newEcho(h *handler.FilesHandler, db handler.Pinger, verifier authjwt.TokenVerifier) (*echo.Echo, error) {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	validator, err := newOpenAPIValidator()
	if err != nil {
		return nil, err
	}
	e.Use(validator)

	health := handler.NewHealthHandler(db)
	e.GET("/healthz", health.Live)
	e.GET("/readyz", health.Ready)

	middlewares := map[string][]echo.MiddlewareFunc{}
	if verifier != nil {
		mw := authjwt.Middleware(verifier)
		middlewares = map[string][]echo.MiddlewareFunc{
			"getFiles":         {mw},
			"uploadFile":       {mw},
			"getFileById":      {mw},
			"downloadFileById": {mw},
		}
	}
	gen.RegisterHandlersWithOptions(e, h, gen.RegisterHandlersOptions{BaseURL: basePath, OperationMiddlewares: middlewares})
	return e, nil
}

func httpErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := http.StatusText(status)
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		status = httpErr.Code
		message = fmt.Sprint(httpErr.Message)
		switch status {
		case http.StatusBadRequest:
			code = "VALIDATION_ERROR"
		case http.StatusUnauthorized:
			code = "UNAUTHORIZED"
		case http.StatusNotFound:
			code = "NOT_FOUND"
		}
	}
	_ = c.JSON(status, gen.ErrorResponse{Code: code, Message: message})
}
