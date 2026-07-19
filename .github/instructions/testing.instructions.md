---
applyTo: '**/*_test.go'
---

# テストガイドライン

## 🧪 テスト実行コマンド

```bash
# 単体テスト
make test-unit                              # サービス全体
cd services/auth && go test -short ./...    # 個別サービス
go test -run TestAuthUsecase_Login ./...    # 特定テスト

# 統合テスト (testcontainers-go, 要 Docker)
make test-integration
go test -tags=integration ./...

# カバレッジ
go test -cover ./...
go test -coverprofile=cov.out ./... && go tool cover -html=cov.out
```

## 📝 テストの書き方

### 基本原則

- **テーブル駆動テスト** を基本とする
- **`t.Parallel()`** を使って並列実行
- テスト名は具体的に（「ログイン成功」より「正しい資格情報でログインできる」）
- 1 テストで 1 シナリオ

### 単体テスト (標準 testing + uber/mock)

testify は使わない。標準 `testing` の `t.Fatal` / `t.Errorf` と `go.uber.org/mock` で書く。
モックは `//go:generate mockgen` で生成する（`make gen-mocks`）。

```go
package usecase_test

import (
    "context"
    "errors"
    "testing"

    "go.uber.org/mock/gomock"

    "github.com/wingarc-ktq/.../internal/domain"
    "github.com/wingarc-ktq/.../internal/usecase"
    "github.com/wingarc-ktq/.../internal/usecase/mock"
)

func TestAuthUsecase_Login(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        email    string
        password string
        setup    func(*mock.MockUserRepository)
        wantErr  error
    }{
        {
            name:     "正しい資格情報でログインできる",
            email:    "taro@example.com",
            password: "P@ssw0rd!",
            setup: func(m *mock.MockUserRepository) {
                m.EXPECT().FindByEmail(gomock.Any(), "taro@example.com").Return(&domain.User{ /* ... */ }, nil)
            },
        },
        {
            name:     "存在しないユーザーは認証エラー",
            email:    "ghost@example.com",
            password: "whatever",
            setup: func(m *mock.MockUserRepository) {
                m.EXPECT().FindByEmail(gomock.Any(), "ghost@example.com").Return(nil, domain.ErrUserNotFound)
            },
            wantErr: domain.ErrInvalidCredential,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            ctrl := gomock.NewController(t)
            repo := mock.NewMockUserRepository(ctrl)
            tt.setup(repo)
            uc := usecase.NewAuthUsecase(repo, hasher, tokens)

            out, err := uc.Login(context.Background(), tt.email, tt.password)
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Fatalf("err: got %v, want %v", err, tt.wantErr)
                }
                return
            }
            if err != nil {
                t.Fatal(err)
            }
            if out.AccessToken == "" {
                t.Error("AccessToken: got empty, want non-empty")
            }
        })
    }
}
```

### 失敗の止め方（testify なし）

- `t.Fatal` / `t.Fatalf` - 続行不能なら **即停止**（testify の `require.*` 相当）
- `t.Error` / `t.Errorf` - 失敗しても続行し、複数の値をまとめてチェック（`assert.*` 相当）

```go
out, err := uc.Login(ctx, email, password)
if err != nil {
    t.Fatal(err) // err があれば停止
}
if out.AccessToken == "" {
    t.Error("AccessToken: got empty, want non-empty")
}
if out.ExpiresIn != 3600 {
    t.Errorf("ExpiresIn: got %d, want 3600", out.ExpiresIn)
}
```

### 統合テスト (testcontainers-go)

リポジトリ層は **本物の PostgreSQL** に対してテストする:

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

func setupPostgres(t *testing.T) *pgxpool.Pool {
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
    t.Cleanup(func() { _ = container.Terminate(ctx) })

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatal(err)
    }

    pool, err := pgxpool.New(ctx, connStr)
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(pool.Close)

    runMigrations(t, connStr) // migrations/ を流すヘルパー
    return pool
}
```

`-tags=integration` ビルドタグで分離する（`make test-unit` では実行されない）。

## 🚫 アンチパターン

- ❌ `Sleep` を使ってタイミング待ち（channel か明示的なポーリングでやる）
- ❌ `sql.NullString` などを direct に持ち込む（domain 層は純粋に）
- ❌ テスト同士が依存（`TestA → TestB` の順序依存）
- ❌ グローバル変数で状態を共有
- ❌ アサーションなしのテスト（実行されただけで OK にしない）

## ✅ チェックリスト

- [ ] `t.Parallel()` を呼んでいる
- [ ] テーブル駆動で書いている
- [ ] テスト名が日本語または明確な英語
- [ ] エラー判定は `errors.Is`
- [ ] 統合テストには `//go:build integration` タグがある
- [ ] gomock の `EXPECT()` で予期した呼び出しを定義している
