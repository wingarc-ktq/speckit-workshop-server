# API Contracts: ユーザー認証

**Feature**: 001-user-auth | **Date**: 2026-06-06

## API 仕様の場所

本サービスの OpenAPI 仕様は既に設計・配置済みです:

```
schema/auth/openapi.yaml
```

Constitution II（OpenAPI ファースト）に従い、`schema/auth/openapi.yaml` が API の Single Source of Truth です。
本ディレクトリに仕様を複製しません。

## エンドポイント一覧

| Method | Path | operationId | 説明 | 認証 |
|---|---|---|---|---|
| POST | `/api/v1/auth/register` | `registerUser` | ユーザー登録 | 不要 |
| POST | `/api/v1/auth/login` | `loginUser` | ログイン（JWT 発行） | 不要 |
| GET | `/api/v1/auth/me` | `getCurrentUser` | 認証中ユーザー情報取得 | Bearer JWT |

## リクエスト / レスポンス概要

### POST /auth/register

**Request Body** (`RegisterRequest`):
```json
{
  "email": "taro@example.com",
  "password": "P@ssw0rd!",
  "name": "田中 太郎"
}
```

**201 Created** (`UserResponse`):
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "taro@example.com",
    "name": "田中 太郎",
    "createdAt": "2026-05-07T10:30:00Z",
    "updatedAt": "2026-05-07T10:30:00Z"
  }
}
```

**400 Bad Request** / **409 Conflict** (`ErrorResponse`):
```json
{
  "message": "メールアドレスは既に使用されています",
  "code": "EMAIL_ALREADY_TAKEN"
}
```

### POST /auth/login

**Request Body** (`LoginRequest`):
```json
{
  "email": "taro@example.com",
  "password": "P@ssw0rd!"
}
```

**200 OK** (`LoginResponse`):
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "tokenType": "Bearer",
  "expiresIn": 3600,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "taro@example.com",
    "name": "田中 太郎",
    "createdAt": "2026-05-07T10:30:00Z",
    "updatedAt": "2026-05-07T10:30:00Z"
  }
}
```

**401 Unauthorized** (`ErrorResponse`):
```json
{
  "message": "認証に失敗しました",
  "code": "AUTH_FAILED"
}
```

### GET /auth/me

**Headers**: `Authorization: Bearer <JWT>`

**200 OK** (`UserResponse`):
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "taro@example.com",
    "name": "田中 太郎",
    "createdAt": "2026-05-07T10:30:00Z",
    "updatedAt": "2026-05-07T10:30:00Z"
  }
}
```

**401 Unauthorized** (`ErrorResponse`):
```json
{
  "message": "認証が必要です",
  "code": "UNAUTHORIZED"
}
```

## コード生成

```bash
# oapi-codegen でサーバーインターフェース + モデルを生成
cd services/auth
oapi-codegen -config oapi-codegen.yaml ../../schema/auth/openapi.yaml
# → api/gen/server.gen.go
```

生成される主要な型:
- `gen.ServerInterface` — handler が実装するインターフェース
- `gen.StrictServerInterface` — strict モードのインターフェース
- `gen.RegisterRequest`, `gen.LoginRequest` — リクエスト型
- `gen.UserResponse`, `gen.LoginResponse`, `gen.ErrorResponse` — レスポンス型
