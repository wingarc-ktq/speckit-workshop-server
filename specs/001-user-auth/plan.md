# Implementation Plan: ユーザー認証

**Branch**: `001-user-auth` | **Date**: 2026-06-06 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-user-auth/spec.md`

## Summary

Auth マイクロサービスとして JWT ベースのユーザー認証機能を提供する。メールアドレス・パスワードによるユーザー登録、ログイン（JWT 発行）、認証中ユーザー情報取得の 3 エンドポイントを実装する。既存の `schema/auth/openapi.yaml` を Single Source of Truth とし、oapi-codegen による型安全なサーバーインターフェースと sqlc による型安全な DB アクセスを採用するクリーンアーキテクチャ構成。

## Technical Context

**Language/Version**: Go 1.26
**Primary Dependencies**: echo/v4（Web フレームワーク）、oapi-codegen v2（OpenAPI コード生成）、kin-openapi（OpenAPI 検証）、golang-jwt/v5（JWT 処理）、pgx/v5（PostgreSQL ドライバ）、sqlc（SQL コード生成）、golang-migrate/v4（マイグレーション）
**Storage**: PostgreSQL 17（サービス専用 DB: `auth`）
**Testing**: 標準 testing + uber/mock（単体テスト、testify は使わない）、testcontainers-go（統合テスト）、Schemathesis + Hurl（API テスト）
**Target Platform**: Linux サーバー（Docker コンテナ）
**Project Type**: マイクロサービス（Go Workspace 内の独立モジュール）
**Performance Goals**: ログインレスポンスタイム 200ms 以内（SC-001）
**Constraints**: パスワード比較は constant-time（bcrypt による SC-002 準拠）、JWT RS256（非対称鍵）署名
**Scale/Scope**: ワークショップ用教材。Auth + Files の 2 サービス構成

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| #   | 原則                                | 判定 | 根拠                                                                                                                 |
| --- | ----------------------------------- | ---- | -------------------------------------------------------------------------------------------------------------------- |
| I   | Go Idiomatic Code (NON-NEGOTIABLE)  | PASS | gofmt/goimports 適用、go vet 通過、GoDoc コメント付与、error は最後の戻り値                                          |
| II  | OpenAPI ファースト (NON-NEGOTIABLE) | PASS | `schema/auth/openapi.yaml` が設計済み。oapi-codegen v2 で `api/gen/server.gen.go` を生成。手動編集禁止               |
| III | クリーンアーキテクチャ（層分離）    | PASS | handler → usecase → domain ← infra の依存方向。domain にインターフェース定義、infra で実装                           |
| IV  | テスト駆動 (NON-NEGOTIABLE)         | PASS | usecase 層にテーブル駆動テスト（標準 testing + uber/mock）。統合テストは testcontainers-go。API テストは Schemathesis + Hurl |
| V   | 型安全な SQL（sqlc）                | PASS | `migrations/` に DDL、`queries/*.sql` にクエリ、sqlc で `db/` パッケージを生成。ORM 不使用                           |
| VI  | マイクロサービス境界                | PASS | 独立 `go.mod`、専用 DB（`auth`）、サービス間通信は JWT の引き回し                                                    |
| VII | 認証は JWT (Bearer)                 | PASS | RS256（非対称鍵）署名、秘密鍵/公開鍵を環境変数（PEM パス）で注入、echo ミドルウェアで検証                            |

**ゲート結果**: 全原則 PASS — Phase 0 に進行可能

## Project Structure

### Documentation (this feature)

```text
specs/001-user-auth/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (OpenAPI は schema/auth/ に既存)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
services/auth/
├── cmd/server/main.go                          # DI 組み立て + サーバー起動
├── api/gen/server.gen.go                       # oapi-codegen 生成（手動編集禁止）
├── internal/
│   ├── config/config.go                        # 環境変数読み込み
│   ├── domain/
│   │   ├── user.go                             # User モデル + UserRepository インターフェース + ドメインエラー
│   │   └── mock/user_mock.go                   # gomock 生成（手動編集禁止）
│   ├── usecase/
│   │   ├── auth_usecase.go                     # Register / Login / Me ユースケース
│   │   └── auth_usecase_test.go                # テーブル駆動単体テスト
│   ├── handler/                                # HTTP ハンドラ（ServerInterface 実装）— 未実装
│   └── infra/
│       └── repo/
│           ├── queries/users.sql               # sqlc クエリ定義
│           ├── db/                             # sqlc 生成（手動編集禁止）
│           │   ├── db.go
│           │   ├── models.go
│           │   └── users.sql.go
│           └── user_repository.go              # UserRepository 実装 — 未実装
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
├── oapi-codegen.yaml                           # oapi-codegen 設定
├── sqlc.yaml                                   # sqlc 設定
├── Makefile                                    # gen / build / test / migrate ターゲット
├── Dockerfile                                  # マルチステージビルド
├── go.mod
└── go.sum

schema/auth/openapi.yaml                        # API 仕様 (Single Source of Truth)
api-tests/
├── schemathesis/                               # OpenAPI 準拠自動テスト
└── hurl/scenarios/auth/                        # シナリオベース API テスト
    ├── 01_register_and_login.hurl
    └── 02_login_failures.hurl
```

**Structure Decision**: 既存のリファレンス実装のディレクトリ構成に完全準拠。`services/auth/` 配下のクリーンアーキテクチャ（handler / usecase / domain / infra）を踏襲。未実装の `handler/` パッケージと `repo/user_repository.go` が主な実装対象。

## Complexity Tracking

> 違反なし — 全原則 PASS のため、正当化が必要な複雑さの追加はない。
