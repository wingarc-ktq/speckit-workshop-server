---
applyTo: '**/*.go'
---

# Go コーディング規約

## 📐 基本原則

[Effective Go](https://go.dev/doc/effective_go) と [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) に準拠します。

## ✍ 命名

| 対象 | 規則 | 例 |
|---|---|---|
| パッケージ | 単数形・小文字・1 単語 | `usecase`, `domain`, `handler` |
| 公開シンボル | PascalCase | `FileRepository`, `NewAuthUsecase` |
| 非公開 | camelCase | `mapError`, `issueToken` |
| インターフェース | 「振る舞い + er」または機能名 | `UserRepository`, `Authenticator` |
| エラー変数 | `Err` プレフィックス | `ErrUserNotFound`, `ErrInvalidCredential` |
| ファイル | snake_case | `user_repository.go`, `auth_usecase_test.go` |

## 🧱 構造

### 関数の戻り値

エラーは **常に最後** の戻り値:

```go
// ✅ 良い
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

// ❌ 悪い
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (error, *domain.User)
```

### context

第一引数として `context.Context` を渡す:

```go
// ✅
func (uc *FileUsecase) List(ctx context.Context, in ListInput) (*ListOutput, error)

// ❌
func (uc *FileUsecase) List(in ListInput, ctx context.Context) (*ListOutput, error)
```

### エラーラップ

`fmt.Errorf` で原因をラップ:

```go
if err != nil {
    return nil, fmt.Errorf("find user by id: %w", err)
}
```

判定は `errors.Is` / `errors.As`:

```go
if errors.Is(err, domain.ErrUserNotFound) { ... }
```

## 📝 GoDoc

公開シンボル（大文字始まり）には必ず GoDoc コメント:

```go
// UserRepository はユーザーストアの抽象.
// 具象は internal/infra/repo で実装する.
type UserRepository interface { ... }

// Login はメール+パスワードで JWT を発行する.
func (u *AuthUsecase) Login(ctx context.Context, email, password string) (*LoginOutput, error)
```

## 🚫 禁止事項

- `any` / `interface{}` を新規コードで使う（型安全性の放棄）
- `init()` 関数（暗黙の副作用）
- グローバル可変状態
- `panic` のスロー（`main()` の起動時を除く）
- 生成コード (`*.gen.go`, `*.sql.go`) の手動編集

## 🎁 オプション値の扱い

ポインタを使う:

```go
// ✅
type UpdateInput struct {
    Name        *string
    Description *string
}

// 適用判定
if in.Name != nil {
    f.Name = *in.Name
}
```

## 🧹 整形

コミット前に必ず:

```bash
go fmt ./...
go vet ./...
```

`gofmt -d .` で差分が出たら整形漏れ。
