# Data Model: ユーザー認証

**Feature**: 001-user-auth | **Date**: 2026-06-06

## エンティティ

### User

認証サービスの中核エンティティ。文書管理システムにアクセスするユーザーを表す。

| フィールド      | 型           | 制約                    | 説明                                     |
| --------------- | ------------ | ----------------------- | ---------------------------------------- |
| `id`            | UUID         | PK                      | ユーザー一意識別子（アプリ側で生成）     |
| `email`         | VARCHAR(255) | NOT NULL, UNIQUE, INDEX | メールアドレス（小文字に正規化して保存） |
| `password_hash` | VARCHAR(255) | NOT NULL                | bcrypt ハッシュ化済みパスワード          |
| `name`          | VARCHAR(100) | NOT NULL                | ユーザー氏名                             |
| `created_at`    | TIMESTAMPTZ  | NOT NULL, DEFAULT NOW() | 作成日時                                 |
| `updated_at`    | TIMESTAMPTZ  | NOT NULL, DEFAULT NOW() | 更新日時                                 |

### リレーションシップ

```text
User (1) ──── (N) [Files サービスの Document]
              ↑ JWT の sub クレームで紐付け（DB 外参照）
```

- Auth サービスの `users` テーブルと Files サービスは DB を共有しない（Constitution VI）
- Files サービスは JWT の `sub`（user_id）でユーザーを識別する

## バリデーションルール

### ユーザー登録時の入力バリデーション

| フィールド   | ルール                                       | エラー時                  |
| ------------ | -------------------------------------------- | ------------------------- |
| `email`      | 必須、email 形式、最大 255 文字              | 400 `VALIDATION_ERROR`    |
| `password`   | 必須、8〜128 文字                            | 400 `VALIDATION_ERROR`    |
| `name`       | 必須、1〜100 文字                            | 400 `VALIDATION_ERROR`    |
| `email` 重複 | DB の UNIQUE 制約 + usecase 層の事前チェック | 409 `EMAIL_ALREADY_TAKEN` |

### ビジネスルール

| ルール                                         | 実装箇所                                                |
| ---------------------------------------------- | ------------------------------------------------------- |
| パスワードは bcrypt (DefaultCost) でハッシュ化 | usecase 層 `Register`                                   |
| メールアドレスは小文字に正規化して保存         | usecase 層 `Register`（`strings.ToLower()`）            |
| メール検索は case-insensitive                  | SQL クエリ `LOWER(email) = LOWER($1)`                   |
| ログイン失敗時に原因を区別しない               | usecase 層 `Login`（統一エラー `ErrInvalidCredential`） |
| パスワード比較は constant-time                 | `bcrypt.CompareHashAndPassword()`（bcrypt 内部で保証）  |

## 状態遷移

User エンティティは明示的な状態フィールドを持たない。ライフサイクルは以下の通り:

```text
[未登録] ──Register──→ [登録済み] ──Login──→ [JWT 発行済み]
                                              │
                                     JWT 期限切れ → 再ログイン必要
```

## DDL（マイグレーション）

### UP (`000001_create_users_table.up.sql`)

```sql
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(100) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
```

### DOWN (`000001_create_users_table.down.sql`)

```sql
DROP TABLE IF EXISTS users;
```

## 層別のデータ表現

### domain 層 (`domain.User`)

```go
type User struct {
    ID           uuid.UUID   // github.com/google/uuid
    Email        string
    PasswordHash string
    Name         string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### infrastructure 層 (`db.User` — sqlc 生成)

```go
type User struct {
    ID           pgtype.UUID        // jackc/pgx/v5/pgtype
    Email        string
    PasswordHash string
    Name         string
    CreatedAt    pgtype.Timestamptz
    UpdatedAt    pgtype.Timestamptz
}
```

### API 層 (`gen.User` — oapi-codegen 生成)

```go
type User struct {
    CreatedAt time.Time           `json:"createdAt"`
    Email     openapi_types.Email `json:"email"`
    Id        openapi_types.UUID  `json:"id"`
    Name      string              `json:"name"`
    UpdatedAt time.Time           `json:"updatedAt"`
}
```

### 型変換の流れ

```text
HTTP Request (JSON)
    ↓ echo.Bind
gen.RegisterRequest / gen.LoginRequest
    ↓ handler で変換
usecase.RegisterInput / (email, password)
    ↓ usecase で処理
domain.User
    ↓ repo アダプターで変換
db.CreateUserParams / db.User (pgtype)
    ↓ sqlc → PostgreSQL
```

## sqlc クエリ定義

```sql
-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, name)
VALUES ($1, $2, $3, $4)
RETURNING id, email, password_hash, name, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, password_hash, name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, name, created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER($1);
```
