// Package server は Auth サービスの合成ルート（DI 組み立て）と
// HTTP サーバーのライフサイクル（起動・graceful shutdown）を担う.
//
// プロセス終了（os.Exit）やシグナル購読といった OS レベルの関心事は cmd/server/main.go に置き、
// ここではアプリの配線と起動・終了に専念する. Run はエラーを返すだけなので defer による
// 後始末が確実に実行され、Run / newEcho は単体テスト可能.
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
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pgx5:// スキームの DB ドライバを登録
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimiddleware "github.com/oapi-codegen/echo-middleware"

	"github.com/wingarc-ktq/speckit-workshop-server/packages/authjwt"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/config"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/password"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/repo"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/token"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase"
	migrationfs "github.com/wingarc-ktq/speckit-workshop-server/services/auth/migrations"
)

// OpenAPI のサーバー URL と一致させる API のベースパス.
const basePath = "/api/v1"

// graceful shutdown でリクエスト完了を待つ最大時間.
const shutdownTimeout = 10 * time.Second

// Run はアプリケーションの配線と起動・終了を担う.
// ctx がキャンセルされると graceful shutdown を開始する.
// エラーは log.Fatal せずに返すため、defer による後始末が確実に実行される.
func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	// サービス自身がスキーマを用意する (self-migration). これにより compose 側は
	// postgres + サービスだけで済み、サービスを増やしても migrate 定義は増えない.
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgxpool: %w", err)
	}
	defer pool.Close()

	tokenSvc, err := token.New(cfg.JWTPrivateKey, cfg.JWTPublicKey, cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("token service: %w", err)
	}

	e, err := newEcho(pool, tokenSvc)
	if err != nil {
		return err
	}

	// シグナル受信 (ctx.Done) で graceful shutdown を開始する.
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := e.Shutdown(shutCtx); err != nil {
			log.Printf("graceful shutdown: %v", err)
		}
	}()

	log.Printf("Auth service listening on :%s", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

// runMigrations は埋め込みマイグレーションを起動時に適用する (適用済みなら何もしない).
// DB ドライバはアプリ本体と同じ pgx/v5 を使う (pgx5:// スキーム).
func runMigrations(databaseURL string) error {
	src, err := iofs.New(migrationfs.FS, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}
	// database/pgx/v5 ドライバは pgx5:// で登録されるため postgres:// を読み替える.
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

// newEcho は依存を配線し、ミドルウェアとルーティングを設定した echo インスタンスを返す.
//
// DI 配線: repo + adapters -> usecase -> handler
func newEcho(pool *pgxpool.Pool, tokenSvc *token.JWT) (*echo.Echo, error) {
	hasher := password.NewBcrypt()
	userRepo := repo.NewUserRepository(pool)
	authUC := usecase.NewAuthUsecase(userRepo, hasher, tokenSvc)
	authHandler := handler.NewAuthHandler(authUC)

	validator, err := newOpenAPIValidator()
	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:  true,
		LogURI:     true,
		LogStatus:  true,
		LogLatency: true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			log.Printf("%s %s %d %s", v.Method, v.URI, v.Status, v.Latency)
			return nil
		},
	}))
	e.Use(middleware.Recover())

	// OpenAPI スキーマ駆動のリクエスト検証ミドルウェア (Constitution II).
	// 形式・必須・文字数などの制約は schema/auth/openapi.yaml を Single Source of Truth として検証する.
	e.Use(validator)

	// 運用用エンドポイント. ビジネス API（OpenAPI 契約）とは別物として /api/v1 配下に置かず、
	// 検証ミドルウェアも素通しする（validator の Skipper 参照）.
	healthHandler := handler.NewHealthHandler(pool)
	e.GET("/healthz", healthHandler.Live)
	e.GET("/readyz", healthHandler.Ready)

	// getCurrentUser (GET /auth/me) のみ JWT 検証ミドルウェアを適用する.
	gen.RegisterHandlersWithOptions(e, authHandler, gen.RegisterHandlersOptions{
		BaseURL: basePath,
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"getCurrentUser": {authjwt.Middleware(tokenSvc)},
		},
	})

	return e, nil
}

// newOpenAPIValidator は埋め込み済み OpenAPI スペックからリクエスト検証ミドルウェアを構築する.
//
//   - 認証 (bearerAuth) の実検証は authjwt.Middleware が担うため、ここでは NoopAuthenticationFunc で素通しする.
//   - Servers をホスト無しのベースパスに上書きし、Host ヘッダ検証を無効化する.
//   - 検証エラーはアプリ共通の gen.ErrorResponse 形式 (code + message) にマッピングする.
func newOpenAPIValidator() (echo.MiddlewareFunc, error) {
	spec, err := gen.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	spec.Servers = openapi3.Servers{{URL: basePath}}

	return oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		SilenceServersWarning: true,
		// /api/v1 配下のビジネス API のみ検証する（運用エンドポイント等は素通し）.
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
			// oapi-middleware はパス不一致もメソッド不許可も一律 404 に丸めるが、
			// 存在するパスへの未定義メソッドは RFC 9110 に従い 405 + Allow ヘッダを返す.
			// 許可メソッドは echo のルータがルーティング時に算出済みの値を流用する.
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
