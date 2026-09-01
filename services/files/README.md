# Files Service

Files サービスは、文書やファイルのアップロード、一覧、検索、メタデータ更新、タグ管理、削除までを担当するマイクロサービスです。

## 機能概要

- ファイルのアップロードとダウンロード
- 一覧取得と検索（キーワード / タグ ID フィルタ）
- ファイル詳細取得
- メタデータ更新（ファイル名 / 説明 / タグ）
- 個別削除と一括削除
- タグの CRUD
- JWT 認証付きの HTTP API

## ディレクトリ構成

```text
services/files/
├── api/
│   └── gen/                 # oapi-codegen 生成コード
├── cmd/server/              # エントリポイント
├── internal/
│   ├── config/              # 環境変数
│   ├── domain/              # entity / errors / ports
│   ├── handler/             # HTTP ハンドラ
│   ├── infra/
│   │   ├── repo/            # DB repository
│   │   └── storage/         # ローカルストレージ
│   ├── server/              # DI / Echo 設定
│   └── usecase/             # アプリケーションロジック
├── migrations/              # SQL migration
├── .env.sample
├── Dockerfile
├── Makefile
├── go.mod
├── oapi-codegen.yaml
├── sqlc.yaml
└── README.md
```

## 実行手順

### 1. 依存の準備

```bash
make tools
cp .env.sample .env
make keys
```

### 2. DB を起動

```bash
cd ../..
make up-db
```

### 3. コード生成

```bash
cd services/files
make gen
```

### 4. マイグレーション

```bash
cd services/files
make migrate-up
```

### 5. ローカル起動

```bash
cd services/files
make run
```

### 6. Docker Compose でまとめて起動

```bash
cd ../..
make up
```

## インフラ構成

- PostgreSQL: `workshop/workshop` を使用
- Files DB: `files`
- Storage: `FILES_STORAGE_PATH` に保存したローカルファイルシステム
- JWT: `FILES_JWT_PUBLIC_KEY_PATH` で RS256 公開鍵を検証
- API: `http://localhost:8082/api/v1`

## テスト

```bash
cd services/files
make test-unit
make test-integration
```

## API テスト

```bash
cd api-tests/schemathesis
make install
make run-files
```

## 参考

- [schema/files/openapi.yaml](../../schema/files/openapi.yaml)
- [specs/002-document-management/quickstart.md](../../specs/002-document-management/quickstart.md)
