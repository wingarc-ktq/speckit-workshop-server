// Package server は Files サービスの合成ルート（DI 組み立て）と
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
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/config"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/handler"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/repo"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/infra/storage"
	"github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
	migrationfs "github.com/wingarc-ktq/speckit-workshop-server/services/files/migrations"
)

// OpenAPI のサーバー URL（schema/files/openapi.yaml の servers.url）と一致させる API のベースパス.
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

	// Files は JWT の検証のみ行う（秘密鍵は持たない、Constitution VII）。
	// 署名できるのは秘密鍵を持つ Auth サービスだけ、という制約が
	// authjwt パッケージの型（NewVerifier は公開鍵しか受け取れない）によって保証されている.
	verifier, err := authjwt.NewVerifier(cfg.JWTPublicKey)
	if err != nil {
		return fmt.Errorf("jwt verifier: %w", err)
	}

	e, err := newEcho(pool, cfg.StorageDir, verifier)
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

	log.Printf("Files service listening on :%s", cfg.Port)
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
// DI 配線: repo + storage -> usecase -> handler
func newEcho(pool *pgxpool.Pool, storageDir string, verifier *authjwt.Verifier) (*echo.Echo, error) {
	fileRepo := repo.NewFileRepository(pool)
	fileStorage := storage.NewLocal(storageDir)
	fileUC := usecase.NewFileUsecase(fileRepo, fileStorage)
	fileHandler := handler.NewFileHandler(fileUC)

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
	// 形式・必須・文字数などの制約は schema/files/openapi.yaml を Single Source of Truth として検証する.
	e.Use(validator)

	// /healthz・/readyz には JWT ミドルウェアを掛けない。
	//
	// なぜなら:
	//   1. この2つは Docker/Kubernetes 等のインフラが定期的に叩く「死活監視」用のエンドポイントで、
	//      呼び出し元はオーケストレーター（人間のユーザーではない）であり JWT を持っていない。
	//      認証必須にすると、そもそもヘルスチェック自体が失敗して正常なコンテナが
	//      再起動され続ける、という本末転倒な事態になる。
	//   2. 仕様上も「認証なしで提供しなければならない」と明記されている
	//      （出典: spec.md FR-023「システムはヘルスチェック機能を認証なしで提供しなければならない」）。
	//   3. Constitution II の例外規定により、運用エンドポイントはそもそも OpenAPI 契約
	//      （ビジネス API）の対象外であり `/api/v1` 配下にも置かない。そのため
	//      OpenAPI 検証ミドルウェアの Skipper（下記 newOpenAPIValidator 参照）でも
	//      素通しし、ここでも個別の echo.MiddlewareFunc を一切挟まずに直接ハンドラを登録する.
	healthHandler := handler.NewHealthHandler(pool)
	e.GET("/healthz", healthHandler.Live)
	e.GET("/readyz", healthHandler.Ready)

	// files の 4 操作（一覧・アップロード・詳細・ダウンロード）はすべて認証必須
	// （出典: spec.md FR-020「すべてのエンドポイント（ヘルスチェックを除く）は
	// JWT Bearer トークンで認証しなければならない」）。
	// auth サービスでは認証必須の操作が getCurrentUser の 1 つだけだったが、
	// Files は「ヘルスチェック以外は全部」なので、4 操作すべてに同じミドルウェアを割り当てる.
	//
	// FileHandler は gen.StrictServerInterface（RequestObject/ResponseObject 形式）を
	// 実装しているが、RegisterHandlersWithOptions は echo ネイティブな gen.ServerInterface
	// しか受け付けない。NewStrictHandler がその変換アダプタになる
	// （strict 用ミドルウェアは今回使わないので第2引数は nil）.
	authMiddleware := authjwt.Middleware(verifier)
	strictHandler := gen.NewStrictHandler(fileHandler, nil)
	gen.RegisterHandlersWithOptions(e, strictHandler, gen.RegisterHandlersOptions{
		BaseURL: basePath,
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"getFiles":            {authMiddleware},
			"uploadFile":          {authMiddleware},
			"getFile":             {authMiddleware},
			"downloadFileContent": {authMiddleware},
		},
	})

	return e, nil
}

// newOpenAPIValidator は埋め込み済み OpenAPI スペックからリクエスト検証ミドルウェアを構築する.
//
//   - 認証 (bearerAuth) の実検証は authjwt.Middleware が担うため、ここでは NoopAuthenticationFunc で素通しする.
//     もしここで認証も検証しようとすると、bearerAuth の検証ロジックが二重管理になり、
//     どちらか片方を直しても片方に反映されない、といった不整合の温床になる.
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
		// /healthz・/readyz を対象外にする理由は newEcho 側のコメントを参照.
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
