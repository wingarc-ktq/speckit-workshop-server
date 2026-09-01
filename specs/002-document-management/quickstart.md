# Quickstart: Files Service

**Feature**: 002-document-management | **Date**: 2026-09-01

## 前提条件

- Go 1.26+
- Docker / Docker Compose
- Python 3.11+（Schemathesis を使う場合）
- `make tools` で入る開発ツール:
  - `oapi-codegen`
  - `sqlc`
  - `migrate`

## セットアップ手順

### 1. 開発ツールのインストール

```bash
make tools
```

### 2. 環境変数を設定する

```bash
cp .env.sample .env
```

`Files` サービスは以下の環境変数を使用します。

| 変数名 | デフォルト | 説明 |
| --- | --- | --- |
| `FILES_SERVICE_PORT` | `8082` | Files サービスのポート |
| `FILES_DATABASE_URL` | `postgres://workshop:workshop@localhost:5432/files?sslmode=disable` | PostgreSQL 接続先 |
| `FILES_STORAGE_PATH` | `./storage` | ローカル保存先 |
| `FILES_JWT_PUBLIC_KEY_PATH` | `./keys/jwt_dev_public.pem` | JWT 検証用公開鍵 |
| `FILES_JWT_TTL_SECONDS` | `3600` | JWT 有効期限 |

### 3. PostgreSQL と鍵を準備する

```bash
make keys
make up-db
```

### 4. コード生成

```bash
cd services/files
make gen
```

### 5. マイグレーションを適用する

```bash
cd services/files
make migrate-up
```

### 6. サービス起動

```bash
cd services/files
make run
```

または、リポジトリ直下から Docker Compose でまとめて起動します。

```bash
make up
```

Files API は `http://localhost:8082/api/v1` で起動します。

## 動作確認

### ログインしてトークンを取得

Auth サービスが起動している前提です。

```bash
curl -s -X POST http://localhost:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"taro@example.com","password":"P@ssw0rd!","name":"田中 太郎"}'

curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"taro@example.com","password":"P@ssw0rd!"}' | jq .
```

### ファイルアップロード

```bash
TOKEN="<accessToken>"
cat <<'EOF' >/tmp/report.txt
hello files
EOF

curl -s -X POST http://localhost:8082/api/v1/files \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/tmp/report.txt" \
  -F "description=業務報告" | jq .
```

### 一覧取得

```bash
curl -s http://localhost:8082/api/v1/files?page=1\&limit=20 \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### ファイル削除

```bash
curl -s -o /dev/null -w '%{http_code}\n' -X DELETE http://localhost:8082/api/v1/files/<fileId> \
  -H "Authorization: Bearer $TOKEN"
```

## テスト実行

### 単体テスト

```bash
cd services/files
make test-unit
```

### 統合テスト（Docker 必須）

```bash
cd services/files
make test-integration
```

### Schemathesis

```bash
cd api-tests/schemathesis
make install
make run-files
```

## 主要な機能

- ファイルのアップロード・一覧取得・詳細取得・ダウンロード
- ファイル名・説明・タグ一覧の更新
- タグの CRUD
- 個別削除および一括削除
- JWT 認証付きの API 公開
