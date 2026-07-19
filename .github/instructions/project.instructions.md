---
applyTo: '**/*.go'
---

# プロジェクト概要

## 📋 技術スタック

- Go 1.26 + echo/v4
- アーキテクチャ: Clean Architecture (handler / usecase / domain / infrastructure)
- API: OpenAPI 3.0 + oapi-codegen v2 + echo-middleware
- DB: PostgreSQL 17 + pgx/v5 + sqlc
- マイグレーション: golang-migrate/v4
- 認証: JWT (RS256 / 非対称鍵, golang-jwt/v5)
- 単体テスト: 標準 testing + uber/mock (gomock)（testify は使わない）
- 統合テスト: testcontainers-go (PostgreSQL モジュール)
- API テスト: Schemathesis (OpenAPI 駆動) + Hurl (シナリオ)

## 📁 アーキテクチャ

### サービスのディレクトリ構造

```
services/<service>/
├── cmd/server/main.go                   # 薄いエントリポイント (signal + server.Run)
├── internal/
│   ├── config/                          # 環境変数読み込み
│   ├── domain/                          # エンティティ・interface・ドメインエラー
│   ├── usecase/                         # アプリケーションロジック
│   ├── handler/                         # HTTP ハンドラ + JWT ミドルウェア
│   ├── infra/repo/                      # pgx + sqlc を使った DB アクセス
│   └── server/                          # DI 組立・echo セットアップ・起動 (Run)
├── api/gen/                             # oapi-codegen 出力 (手動編集禁止)
├── migrations/                          # golang-migrate の SQL
├── oapi-codegen.yaml
├── sqlc.yaml
└── go.mod
```

### 分離指針

- **handler**: HTTP の入出力変換のみ。ビジネスロジック禁止。oapi-codegen の `ServerInterface` を実装
- **usecase**: domain の interface に依存。infrastructure を直接 import しない
- **domain**: 外部依存ゼロ。`*_test.go` 以外で標準ライブラリ + uuid だけ
- **infra**: domain の interface を実装。pgx, sqlc 生成コードはここでだけ使う

依存方向: `handler → usecase → domain ← infra`

## 🔍 開発時の確認コマンド

```bash
# コード生成
make gen                  # OpenAPI + SQL から Go コード生成
make gen-oapi             # oapi-codegen のみ
make gen-sqlc             # sqlc のみ
make gen-mocks            # gomock のみ

# サーバー起動
make up                   # Docker Compose で全サービス起動
make up-db                # PostgreSQL のみ
cd services/<svc> && make run  # 個別サービスをローカル実行

# コード品質
go fmt ./...
go vet ./...

# テスト
make test                 # 単体 + 統合
make test-unit            # 単体のみ (高速)
make test-integration     # testcontainers (要 Docker)
make api-test             # Schemathesis + Hurl
```

## ✅ コーディングチェックリスト

- [ ] `gofmt` / `goimports` 適用済み
- [ ] `go vet ./...` がエラーなし
- [ ] public シンボルに GoDoc コメントがある
- [ ] エラーは最後の戻り値、`error` インターフェースを返す
- [ ] ドメインエラーは `domain.Err*` で定義し `errors.Is` で判定
- [ ] 生成コード (`api/gen/`, `internal/infra/repo/db/`) を手動編集していない
- [ ] テストはテーブル駆動で `t.Parallel()` を使う
- [ ] DB に依存するテストは `//go:build integration` タグで分離
