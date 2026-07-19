# Spec Kit Workshop (Server)

## 📖 概要

このリポジトリは、サーバーサイド開発者体験向上ワークショップ用のテンプレートです。
**Spec Kit** ワークフローを使って **OpenAPI 設計 → Go 実装 → API テスト** までを 3 日間で体験します。

姉妹リポジトリ: [speckit-workshop](https://github.com/wingarc-ktq/speckit-workshop)（フロントエンド版）

## ✨ 主な特徴

- 🧭 **仕様駆動**: spec.md から OpenAPI を設計し、コード生成で実装を加速
- 🦫 **Go マイクロサービス**: Auth サービス + Files サービスの 2 サービス構成
- 🏗 **OpenAPI ファースト**: oapi-codegen でハンドラインターフェースとリクエスト検証を自動生成
- 🗄 **PostgreSQL + sqlc**: SQL ファースト + 型安全なクエリコード生成
- 🧪 **多層テスト**: 単体テスト (標準 testing + uber/mock) / 統合テスト (testcontainers-go) / API テスト (Schemathesis + Hurl)
- 🐳 **Docker Compose**: ワンコマンドで全サービス + DB を起動

## 🛠 技術スタック

| レイヤ                 | 採用技術                                                                                                                             |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| 言語                   | Go 1.26                                                                                                                              |
| Web フレームワーク     | [echo/v4](https://echo.labstack.com/)                                                                                                |
| OpenAPI コード生成     | [oapi-codegen v2](https://github.com/oapi-codegen/oapi-codegen) + [echo-middleware](https://github.com/oapi-codegen/echo-middleware) |
| OpenAPI バリデーション | [kin-openapi](https://github.com/getkin/kin-openapi)                                                                                 |
| DB ドライバ            | [pgx/v5](https://github.com/jackc/pgx)                                                                                               |
| SQL コード生成         | [sqlc](https://github.com/sqlc-dev/sqlc)                                                                                             |
| マイグレーション       | [golang-migrate/v4](https://github.com/golang-migrate/migrate)                                                                       |
| 認証                   | [golang-jwt/v5](https://github.com/golang-jwt/jwt)                                                                                   |
| 単体テスト             | 標準 testing + [uber/mock (gomock)](https://github.com/uber-go/mock)（testify は使わない）                                           |
| 統合テスト             | [testcontainers-go](https://golang.testcontainers.org/) (PostgreSQL)                                                                 |
| API テスト             | [Schemathesis](https://schemathesis.readthedocs.io/) + [Hurl](https://hurl.dev/)                                                     |
| データストア           | PostgreSQL 16                                                                                                                        |

## 🚀 クイックスタート

### 前提

- Go 1.26+
- Docker / Docker Compose
- Python 3.11+ (Schemathesis 用、Day 3)
- [Hurl](https://hurl.dev/docs/installation.html) (Day 3)
- (任意) [air](https://github.com/cosmtrek/air) など Go のホットリロードツール

### セットアップ

```bash
# このリポジトリをクローン
git clone <this-repo-url>
cd speckit-workshop-server

# 環境変数のコピー
cp .env.sample .env

# 開発ツールのインストール (oapi-codegen, sqlc, migrate)
make tools

# OpenAPIとSQLからコード生成
make gen

# サービスを起動 (postgres + auth + files)
make up

# ログを見る
make logs
```

起動後:

- **Auth Service**: http://localhost:8081
- **Files Service**: http://localhost:8082

### よく使うコマンド

```bash
make up                 # 全サービス起動
make up-db              # PostgreSQL のみ起動 (ローカル Go 実行用)
make down               # 全停止
make gen                # OpenAPI/SQLからコード再生成
make test               # 単体 + 統合テスト
make test-unit          # 単体テストのみ
make test-integration   # testcontainers-go を使った統合テスト
make api-test           # Schemathesis + Hurl
```

## 📁 プロジェクト構成

```
speckit-workshop-server/
├── docs/                    # ワークショップ資料 (day0〜day3)
├── specs/                   # spec-kit の仕様書
│   ├── 001-user-auth/       # 認証機能 (リファレンス)
│   └── 002-document-management/  # 文書管理 (受講者の作業対象)
├── schema/                  # OpenAPI 仕様
│   ├── auth/openapi.yaml    # Auth サービス (リファレンス)
│   └── files/               # Files サービス (Day 1 で受講者が作成)
├── services/
│   ├── auth/                # Auth サービス (リファレンス実装)
│   └── files/                # Files サービス (Day 2 で実装)
├── api-tests/
│   ├── schemathesis/        # OpenAPI 駆動の自動 API テスト
│   └── hurl/                # シナリオベースの API テスト
├── migrations/init/         # docker compose 起動時の DB 初期化
├── compose.yaml
├── go.work                  # Go workspace (auth + files)
└── Makefile
```

## 📅 3日間ワークショップの流れ

| Day       | テーマ                                  | 主な成果物                                               |
| --------- | --------------------------------------- | -------------------------------------------------------- |
| **Day 1** | 仕様書のユースケースから OpenAPI を設計 | `schema/files/openapi.yaml`                              |
| **Day 2** | OpenAPI から Go 実装 + 単体テスト       | `services/files/` 内のハンドラ・ユースケース・リポジトリ |
| **Day 3** | API テスト                              | Schemathesis レポート + Hurl シナリオテスト              |

詳細は [docs/day0-overview.md](docs/day0-overview.md) を参照してください。

## 🤖 spec-kit コマンド

このリポジトリは spec-kit ワークフローに対応しています。

| コマンド             | 用途                                   |
| -------------------- | -------------------------------------- |
| `/speckit.specify`   | 自然言語から spec.md を生成            |
| `/speckit.plan`      | spec.md から実装計画 (plan.md) を生成  |
| `/speckit.tasks`     | 計画から具体的タスク (tasks.md) を生成 |
| `/speckit.implement` | タスクを実行してコードを生成           |

詳細は [docs/day0-overview.md](docs/day0-overview.md) を参照。

## 📜 ライセンス

MIT
