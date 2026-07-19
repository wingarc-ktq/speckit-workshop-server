# Quickstart: ユーザー認証サービス

**Feature**: 001-user-auth | **Date**: 2026-06-06

## 前提条件

- Go 1.26+
- Docker / Docker Compose
- 開発ツール（`make tools` でインストール可能）:
  - `oapi-codegen` — OpenAPI からの Go コード生成
  - `sqlc` — SQL からの Go コード生成
  - `migrate` — DB マイグレーション

## セットアップ手順

### 1. 開発ツールのインストール

```bash
make tools
```

### 2. 環境変数の設定

```bash
cp .env.sample .env
# 必要に応じて値を編集
```

主要な環境変数:

| 変数名                 | デフォルト値                                                       | 説明                  |
| ---------------------- | ------------------------------------------------------------------ | --------------------- |
| `AUTH_SERVICE_PORT`    | `8081`                                                             | Auth サービスのポート |
| `AUTH_DATABASE_URL`    | `postgres://workshop:workshop@localhost:5432/auth?sslmode=disable` | PostgreSQL 接続文字列 |
| `AUTH_JWT_PRIVATE_KEY_PATH` | `../../keys/jwt_dev_private.pem`                              | JWT 署名用 秘密鍵（RS256）の PEM パス |
| `AUTH_JWT_PUBLIC_KEY_PATH`  | `../../keys/jwt_dev_public.pem`                              | JWT 検証用 公開鍵（RS256）の PEM パス |
| `AUTH_JWT_TTL_SECONDS` | `3600`                                                             | JWT 有効期限（秒）    |

### 3. PostgreSQL の起動

```bash
# PostgreSQL のみ起動（ローカル開発用）
make up-db

# auth / files の DB が自動作成される（migrations/init/00_create_databases.sql）
```

### 4. コード生成

```bash
# OpenAPI + sqlc + gomock のコード生成
cd services/auth
make gen
```

生成されるファイル:

- `api/gen/server.gen.go` — oapi-codegen 生成
- `internal/infra/repo/db/*.go` — sqlc 生成
- `internal/domain/mock/user_mock.go`、`internal/domain/mock/security_mock.go` — gomock 生成

### 5. DB マイグレーション

```bash
cd services/auth
make migrate-up
```

### 6. ビルドと起動

```bash
# JWT 署名鍵（RS256）の生成（リポジトリルートで1回だけ。既存ならスキップ）
# 鍵は git 管理しないため、ローカルで生成する。
make keys

# ビルド
cd services/auth
make build

# 起動
make run
# → Auth service listening on :8081
```

### 7. Docker Compose での全体起動

```bash
# リポジトリルートから
make up
# Auth:  http://localhost:8081
```

## 動作確認

### ユーザー登録

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "taro@example.com",
    "password": "P@ssw0rd!",
    "name": "田中 太郎"
  }' | jq .
```

期待されるレスポンス（201 Created）:

```json
{
  "user": {
    "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "email": "taro@example.com",
    "name": "田中 太郎",
    "createdAt": "2026-...",
    "updatedAt": "2026-..."
  }
}
```

### ログイン

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "taro@example.com",
    "password": "P@ssw0rd!"
  }' | jq .
```

期待されるレスポンス（200 OK）:

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "tokenType": "Bearer",
  "expiresIn": 3600,
  "user": { ... }
}
```

### 認証中ユーザー情報取得

```bash
TOKEN="<上記で取得した accessToken>"
curl -s http://localhost:8081/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN" | jq .
```

期待されるレスポンス（200 OK）:

```json
{
  "user": {
    "id": "...",
    "email": "taro@example.com",
    "name": "田中 太郎",
    "createdAt": "...",
    "updatedAt": "..."
  }
}
```

## テスト実行

### 単体テスト

```bash
cd services/auth
make test-unit
```

### 統合テスト（testcontainers-go / Docker 必須）

```bash
cd services/auth
make test-integration
```

### API テスト（サービス起動後）

```bash
# リポジトリルートから
make api-test       # Schemathesis + Hurl を実行
make schemathesis   # OpenAPI 準拠テストのみ
make hurl           # シナリオテストのみ
```

## 開発ワークフロー

```text
1. schema/auth/openapi.yaml を編集
2. make gen-auth（コード再生成）
3. handler / usecase / infrastructure を実装
4. make test-unit（単体テスト）
5. make test-integration（統合テスト）
6. make up → make api-test（API テスト）
7. make lint && make fmt（コードチェック）
```
