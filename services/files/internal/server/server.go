package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimiddleware "github.com/oapi-codegen/echo-middleware"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/config"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/storage"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
	migrationfs "github.com/wingarc-ktq/speckit-workshop-server/services/files/migrations"
)

const basePath = "/api/v1"
const shutdownTimeout = 10 * time.Second

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgxpool: %w", err)
	}
	defer pool.Close()

	verifier, err := authjwt.NewVerifier(cfg.JWTPublicKey)
	if err != nil {
		return fmt.Errorf("jwt verifier: %w", err)
	}

	e, err := newEcho(pool, verifier, cfg.StoragePath)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := e.Shutdown(shutCtx); err != nil {
			log.Printf("graceful shutdown: %v", err)
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

func newEcho(pool *pgxpool.Pool, verifier authjwt.TokenVerifier, storagePath string) (*echo.Echo, error) {
	fileStorage := storage.NewLocalStorage(storagePath)
	fileRepository := repo.NewFileRepository(pool)
	tagRepository := repo.NewTagRepository(pool)
	fileUC := usecase.NewFileUsecase(fileRepository, fileStorage, 5*1024*1024)
	tagUC := usecase.NewTagUsecase(tagRepository)
	fileHandler := handler.NewFileHandler(fileUC, tagUC)

	validator, err := newOpenAPIValidator()
	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogLatency: true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			log.Printf("%s %s %d %s", v.Method, v.URI, v.Status, v.Latency)
			return nil
		},
	}))
	e.Use(middleware.Recover())
	e.Use(validator)

	healthHandler := handler.NewHealthHandler(pool)
	e.GET("/healthz", healthHandler.Live)
	e.GET("/readyz", healthHandler.Ready)

	middlewareMap := map[string][]echo.MiddlewareFunc{
		"uploadFile":      {authjwt.Middleware(verifier)},
		"listFiles":       {authjwt.Middleware(verifier)},
		"batchDeleteFiles": {authjwt.Middleware(verifier)},
		"deleteFile":      {authjwt.Middleware(verifier)},
		"getFile":         {authjwt.Middleware(verifier)},
		"updateFile":      {authjwt.Middleware(verifier)},
		"downloadFile":    {authjwt.Middleware(verifier)},
		"createTag":       {authjwt.Middleware(verifier)},
		"listTags":        {authjwt.Middleware(verifier)},
		"deleteTag":       {authjwt.Middleware(verifier)},
		"updateTag":       {authjwt.Middleware(verifier)},
	}

	gen.RegisterHandlersWithOptions(e, fileHandler, gen.RegisterHandlersOptions{
		BaseURL: basePath,
		OperationMiddlewares: middlewareMap,
	})

	return e, nil
}

func newOpenAPIValidator() (echo.MiddlewareFunc, error) {
	spec, err := gen.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	spec.Servers = openapi3.Servers{{URL: basePath}}

	return oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		SilenceServersWarning: true,
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Request().URL.Path, basePath)
		},
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		ErrorHandler: func(c echo.Context, err *echo.HTTPError) error {
			code := "VALIDATION_ERROR"
			status := err.Code
			msg := fmt.Sprintf("%v", err.Message)
			switch {
			case err.Code == http.StatusNotFound && msg == routers.ErrMethodNotAllowed.Error():
				status = http.StatusMethodNotAllowed
				code = "METHOD_NOT_ALLOWED"
				if allow, ok := c.Get(echo.ContextKeyHeaderAllow).(string); ok && allow != "" {
					c.Response().Header().Set(echo.HeaderAllow, allow)
				}
			case err.Code == http.StatusNotFound:
				code = "NOT_FOUND"
			}
			return c.JSON(status, gen.ErrorResponse{
				Code:    code,
				Message: msg,
			})
		},
	}), nil
}
