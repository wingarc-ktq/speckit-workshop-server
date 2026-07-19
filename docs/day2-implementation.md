# Day 2: Go 実装と単体テスト

## 🎯 今日のゴール

Day 1 で設計した OpenAPI を元に、Files サービスを Go で実装し、単体テスト・統合テストを書く。

**成果物**

- `services/files/` 内の動作するハンドラ・ユースケース・リポジトリ
- 単体テスト（標準 testing + uber/mock）
- 統合テスト（testcontainers-go）
- `tasks.md`（`/speckit.tasks` の出力）

---

## ⏰ タイムテーブル（6時間想定）

| 時間        | 内容                                  |
| ----------- | ------------------------------------- |
| 10:30-11:00 | Day 1 振り返り・環境確認              |
| 11:00-11:30 | 1️⃣ プロジェクト構成の確認・`make gen` |
| 11:30-12:00 | 2️⃣ `/speckit.tasks` でタスク分解      |
| 12:00-13:00 | 休憩                                  |
| 13:00-16:00 | 3️⃣ `/speckit.implement` で実装        |
| 16:00-16:30 | 4️⃣ サービス起動・動作確認             |
| 16:30-17:00 | 5️⃣ テスト実行                         |
| 17:00-17:30 | 6️⃣ レビュー                           |

---

## 📋 事前準備チェックリスト

- [ ] Day 1 で `schema/files/openapi.yaml` を作成済み
- [ ] `plan.md` を作成済み
- [ ] `make tools` で開発ツールを入れた
- [ ] Docker が起動している

---

## 1️⃣ プロジェクト構成の確認（30分）

### 1.1 ディレクトリ構造（参考例）

下記は **一例** です。層のディレクトリ構成（`cmd/` と `internal/{domain,usecase,handler,infra,server}`）と生成物の置き場所（`api/gen/`・`internal/infra/repo/db/`）は Constitution（クリーンアーキテクチャ）準拠で固定ですが、**各層内のファイル分割・命名は設計次第で変わって構いません**（例: `domain` を `file.go`/`tag.go` に分けても 1 ファイルにまとめてもよい）。

```
services/files/
├── cmd/server/main.go              # 薄いエントリポイント (signal + server.Run)
├── internal/
│   ├── config/                     # 環境変数読み込み
│   ├── domain/                     # ビジネスモデル + interface
│   │   ├── file.go                 # File entity, FileRepository interface
│   │   ├── tag.go                  # Tag entity, TagRepository interface
│   │   └── errors.go               # ドメインエラー
│   ├── usecase/                    # アプリケーションロジック
│   │   ├── file_usecase.go
│   │   ├── file_usecase_test.go    # gomock を使った単体テスト
│   │   ├── tag_usecase.go
│   │   └── tag_usecase_test.go
│   ├── handler/                    # HTTP ハンドラ (oapi-codegen のインターフェースを実装)
│   │   ├── file_handler.go
│   │   ├── tag_handler.go
│   │   └── health.go               # /healthz (liveness) + /readyz (readiness)
│   ├── infra/repo/
│   │   ├── queries/
│   │   │   ├── files.sql           # sqlc 用クエリ
│   │   │   └── tags.sql
│   │   ├── db/                     # sqlc 生成コード (触らない)
│   │   │   ├── querier.go
│   │   │   ├── files.sql.go
│   │   │   └── tags.sql.go
│   │   ├── file_repository.go      # FileRepository 実装
│   │   └── file_repository_test.go # testcontainers-go の統合テスト
│   └── server/                     # DI 組立・echo セットアップ・起動 (Run)
├── api/gen/server.gen.go           # oapi-codegen 生成 (触らない)
├── migrations/
│   ├── 000001_create_files.up.sql
│   ├── 000001_create_files.down.sql
│   ├── 000002_create_tags.up.sql
│   └── 000002_create_tags.down.sql
├── go.mod
├── oapi-codegen.yaml
├── sqlc.yaml
└── Makefile
```

Auth サービスがリファレンスです。困ったら [services/auth/](../services/auth/) を覗いてください。

> `services/files/` はまだ存在しません。**Auth を雛形に 0 から作成** します（`go.mod`・`Makefile`・`oapi-codegen.yaml`・`sqlc.yaml`・`cmd/server/main.go`・`internal/config/` などを `services/auth/` からコピーして `files` 用に調整）。作成後、`go.work` の `use` に `./services/files` を追加してください。

### 1.2 `make gen` を実行する

OpenAPI からハンドラインターフェースを生成します:

```bash
cd services/files
make gen-oapi
```

✅ `api/gen/server.gen.go` ができていれば成功。

ただし sqlc 用のマイグレーション（`migrations/`）と SQL（`internal/infra/repo/queries/`）は **これから書きます**。

---

## 2️⃣ `/speckit.tasks` でタスク分解（30分）

```
/speckit.tasks
```

`tasks.md` が生成されます。だいたい 30〜60 個のタスクに分解されます。

**タスクの構造例**

```
- [ ] T001 [P] Setup migrations: 000001_create_files.up.sql / .down.sql
- [ ] T002 [P] Setup migrations: 000002_create_tags.up.sql / .down.sql
- [ ] T003 [US1] internal/domain/file.go - File struct + FileRepository interface
- [ ] T004 [US1] internal/domain/errors.go - ErrFileNotFound, ErrFileTooLarge
- [ ] T005 [US1] internal/usecase/file_usecase.go - UploadFile()
- [ ] T006 [US1] internal/usecase/file_usecase_test.go - Upload の単体テスト
- ...
```

`[P]` がついたタスクは並列実行可能。`[US1]`〜`[US3]` は対応する User Story。

---

## 3️⃣ `/speckit.implement` で実装（3時間）

### 3.1 タスク単位の実行

```
/speckit.implement

T001 と T002 を並列で実装してください。マイグレーションファイルを作成します。
```

```
/speckit.implement

User Story 1 のすべてのタスクを実装してください。
```

### 3.2 実装の進め方（推奨順序）

1. **マイグレーション** (`migrations/*.sql`)
2. **SQL クエリ** (`internal/infra/repo/queries/*.sql`)
3. **`make gen-sqlc`** で `db/` を生成
4. **ドメイン層** (`internal/domain/`)
5. **ユースケース層** + **単体テスト** (`internal/usecase/`)
6. **インフラ層** + **統合テスト** (`internal/infra/repo/`)
7. **ハンドラ層** (`internal/handler/`) + **ヘルスチェック** (`/healthz`・`/readyz`)
8. **JWT 検証** — 共有パッケージ `packages/authjwt` の `Middleware` を利用（自前実装は不要）
9. **`internal/server/`** で DI 配線・echo 組立・起動 (`Run`)、**`cmd/server/main.go`** は薄いシム (signal + `server.Run`)

### 3.3 単体テストのパターン (uber/mock)

testify は使いません（Constitution）。標準 `testing` の `t.Fatal` / `t.Errorf` と
`go.uber.org/mock`（`//go:generate mockgen` で生成）を使い、テーブル駆動で書きます。
`services/auth/internal/usecase/auth_usecase_test.go` が実例です:

```go
import (
    "context"
    "errors"
    "testing"

    "go.uber.org/mock/gomock"
)

func TestFileUsecase_Upload(t *testing.T) {
    t.Parallel()
    tests := []struct {
        name    string
        input   usecase.UploadInput
        setup   func(*mock.MockFileRepository)
        wantErr error
    }{
        {
            name:  "success",
            input: usecase.UploadInput{ /* ... */ },
            setup: func(m *mock.MockFileRepository) {
                m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
            },
        },
        // ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            ctrl := gomock.NewController(t)
            repo := mock.NewMockFileRepository(ctrl)
            tt.setup(repo)
            uc := usecase.NewFileUsecase(repo /* , ... */)

            got, err := uc.Upload(context.Background(), tt.input)
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Fatalf("err: got %v, want %v", err, tt.wantErr)
                }
                return
            }
            if err != nil {
                t.Fatal(err)
            }
            if got.ID == "" {
                t.Error("ID: got empty, want non-empty")
            }
        })
    }
}
```

> モックは `//go:generate mockgen -source=port.go -destination=mock/port_mock.go -package=mock`
> のようにインターフェース定義の直前で宣言し、`make gen-mocks`（= `go generate ./...`）で生成します。

### 3.4 統合テストのパターン (testcontainers-go)

リポジトリ層は **本物の PostgreSQL** に対してテストします:

```go
//go:build integration

package repo_test

import (
    "context"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestFileRepository_Create(t *testing.T) {
    pool, cleanup := setupPostgres(t)
    defer cleanup()

    r := repo.NewFileRepository(pool)
    if err := r.Create(context.Background(), &domain.File{ /* ... */ }); err != nil {
        t.Fatal(err)
    }
}

func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
    t.Helper()
    ctx := context.Background()
    container, err := postgres.Run(ctx, "postgres:17-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
        ),
    )
    if err != nil {
        t.Fatal(err)
    }

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatal(err)
    }

    pool, err := pgxpool.New(ctx, connStr)
    if err != nil {
        t.Fatal(err)
    }

    // migrations を流す
    runMigrations(t, connStr)

    return pool, func() {
        pool.Close()
        _ = container.Terminate(ctx)
    }
}
```

`-tags=integration` を付けて実行:

```bash
go test -race -tags=integration ./internal/infra/...
```

または `make test-integration`

### 3.5 ハンドラ層の書き方

`make gen-oapi` で `api/gen/server.gen.go` の中に **ServerInterface** が生成されます。
これを実装します:

```go
package handler

import (
    "net/http"

    "github.com/labstack/echo/v4"

    "github.com/wingarc-ktq/speckit-workshop-server/services/files/api/gen"
    "github.com/wingarc-ktq/speckit-workshop-server/services/files/internal/usecase"
)

type FileHandler struct {
    uc *usecase.FileUsecase
}

var _ gen.ServerInterface = (*FileHandler)(nil)

func (h *FileHandler) GetFiles(ctx echo.Context, params gen.GetFilesParams) error {
    files, err := h.uc.List(ctx.Request().Context(), usecase.ListInput{
        Search: deref(params.Search),
        Page:   deref(params.Page),
        Limit:  deref(params.Limit),
    })
    if err != nil {
        return mapError(ctx, err)
    }
    return ctx.JSON(http.StatusOK, toFileListResponse(files))
}
// ...
```

### 3.6 JWT 検証（共有パッケージ authjwt）

JWT 検証ミドルウェアは **自前で書きません**。Files サービスは Auth が RS256 の秘密鍵で発行したトークンを
**公開鍵で検証するだけ** なので、共有パッケージ [packages/authjwt](../packages/authjwt/) を使います
（`AUTH`/`FILES` 双方が同じ検証ロジックを共有し、実装の重複と食い違いを防ぐ）。

`server/` の配線で、公開鍵から検証器を作り、oapi-codegen が生成した各オペレーションに
`authjwt.Middleware` を適用します:

```go
// internal/server/server.go（抜粋）
pubPEM, err := os.ReadFile(cfg.JWTPublicKeyPath)
if err != nil {
    return err
}
verifier, err := authjwt.NewVerifier(pubPEM) // RS256 公開鍵で検証
if err != nil {
    return err
}

// /api/v1 配下のビジネス API 全体に JWT 検証を適用する。
gen.RegisterHandlersWithOptions(e, fileHandler, gen.RegisterHandlersOptions{
    BaseURL:     basePath,
    Middlewares: []echo.MiddlewareFunc{authjwt.Middleware(verifier)},
})
```

ハンドラ側では、検証済みの userID を context から取り出します:

```go
userID, ok := authjwt.UserIDFromContext(c) // uuid.UUID
if !ok {
    return echo.NewHTTPError(http.StatusUnauthorized)
}
```

> `/healthz`・`/readyz` は `authjwt.Middleware` を掛けず、`/api/v1` の外に直接ルーティングします。
> 実例は [services/auth/internal/server/server.go](../services/auth/internal/server/server.go) を参照。

---

## 4️⃣ サービスを起動して動作確認（30分）

```bash
# 開発用 RS256 鍵を生成 (未生成なら) + DB のみ起動
make keys
make up-db

# 別ターミナルで auth と files を起動
cd services/auth && make migrate-up && make run &
cd services/files && make migrate-up && make run &

# ヘルスチェック (運用エンドポイント。/api/v1 の外)
curl http://localhost:8081/healthz   # auth liveness
curl http://localhost:8081/readyz    # auth readiness (DB 疎通)
curl http://localhost:8082/healthz   # files liveness

# ユーザー登録 → ログイン
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"taro@example.com","password":"P@ssw0rd!","name":"田中太郎"}'

TOKEN=$(curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"taro@example.com","password":"P@ssw0rd!"}' \
  | jq -r '.accessToken')

# ファイル一覧
curl http://localhost:8082/api/v1/files \
  -H "Authorization: Bearer $TOKEN"
```

または Docker Compose でまとめて:

```bash
make up
make logs
```

---

## 5️⃣ 単体テスト・統合テストの実行（30分）

```bash
# 全テスト
cd services/files && make test

# 単体のみ (高速)
make test-unit

# 統合のみ (Docker が必要)
make test-integration

# カバレッジ
go test -cover ./...
```

---

## 6️⃣ レビュー（30分）

### 観点

- [ ] MVP の API が curl で叩けて期待通り返ってくる
- [ ] handler / usecase / domain / infrastructure が層を越えて依存していない
- [ ] usecase の単体テストカバレッジが 80% 以上
- [ ] 生成コード（`api/gen/`, `db/`）に手を入れていない
- [ ] エラーハンドリングが統一されている（domain.Err\* → HTTP ステータス）

---

## 📝 Day 2 振り返りチェックリスト

- [ ] OpenAPI からハンドラインターフェースを生成できた (`make gen-oapi`)
- [ ] sqlc で SQL から Go コードを生成できた (`make gen-sqlc`)
- [ ] domain / usecase / handler / infrastructure の各層を実装できた
- [ ] gomock を使った単体テストが書けた
- [ ] testcontainers-go を使った統合テストが書けた
- [ ] サービスが起動し、curl で MVP 機能が動作した

---

## ➡️ 次回予告: Day 3

Day 3 では起動した Files サービスに対して、Schemathesis と Hurl を使った API テストを書きます。

**事前準備**

- [ ] サービスが安定起動する状態にする (`make up` で OK)
