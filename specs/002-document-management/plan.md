# Implementation Plan: 文書管理（Files サービス / P1 MVP）

**Branch**: `002-document-management` | **Date**: 2026-08-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-document-management/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Files マイクロサービスを新規作成し、`spec.md` の P1（MVP）ユーザーストーリー 3 件——(1) ファイルアップロード、(2) ファイル一覧取得と検索、(3) ファイル詳細取得とダウンロード——を実装する。対応する OpenAPI 契約は `schema/files/openapi.yaml` に既に P1 スコープで確定しており（`GET/POST /files`, `GET /files/{fileId}`, `GET /files/{fileId}/content`）、`services/files/api/gen/server.gen.go` も生成済みである。技術的アプローチは `services/auth` と同一のクリーンアーキテクチャ層構成（domain/usecase/handler/infra/server）を踏襲し、JWT 検証は既存の共有パッケージ `packages/authjwt`（RS256 公開鍵検証）を再利用する。ファイル本体はストレージポートで抽象化し、P1 ではローカルファイルシステム実装を用いる。タグ管理・ファイル編集・削除（User Story 4〜6, P2）は本フェーズの実装対象外。

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: echo/v4, oapi-codegen/v2（strict-server + echo-server, embedded-spec）, kin-openapi（リクエスト検証）, oapi-codegen/echo-middleware, golang-jwt/v5（`packages/authjwt` 経由）, pgx/v5, sqlc, golang-migrate/v4, google/uuid

**Storage**: PostgreSQL 17（`files` データベース、`auth` とは独立。Constitution VI）。ファイル本体はストレージポート（`FileStorage`）越しにローカルファイルシステム（`FILES_STORAGE_DIR`）へ保存。メタデータのみ PostgreSQL に永続化

**Testing**: 標準 `testing` + `go.uber.org/mock`（usecase/handler 単体テスト）、`testcontainers-go`（PostgreSQL を用いた repo 層統合テスト）、Schemathesis + Hurl（OpenAPI 駆動 API テスト）

**Target Platform**: Linux サーバー（Docker コンテナ、`docker compose` で auth・postgres と併走）

**Project Type**: Web service（マイクロサービス。既存の `services/<name>/` レイアウトに従う単一サービスプロジェクト）

**Performance Goals**: SC-001 アップロード（5MB）5 秒以内 / SC-002 一覧取得 500ms 以内（100 件時）/ SC-003 キーワード検索 1 秒以内（100 件時）

**Constraints**: 1 ファイル最大 10MB（超過時 413 `FILE_TOO_LARGE`）、ページネーションはデフォルト page=1/limit=20（最大100）、全エンドポイント（`/healthz`・`/readyz` 除く）JWT Bearer 必須、エラーレスポンスは `{message, code}` に統一

**Scale/Scope**: 同時アクセスユーザー最大 50 人を想定。P1 スコープはエンドポイント 4 本（`getFiles`, `uploadFile`, `getFile`, `downloadFileContent`）+ 運用エンドポイント 2 本（`/healthz`, `/readyz`）

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 原則 | 判定 | 根拠 |
|---|---|---|
| I. Go Idiomatic Code | PASS | `gofmt`/`go vet`/Effective Go 準拠で実装。auth と同じ Lint 設定を流用 |
| II. OpenAPI ファースト | PASS | `schema/files/openapi.yaml` が先に確定済みで P1 の 4 エンドポイントを定義。`api/gen/server.gen.go` は oapi-codegen 生成済みで手編集しない。`/healthz`/`/readyz` は契約対象外として `/api/v1` 配下に置かない |
| III. クリーンアーキテクチャ | PASS | `domain`（File エンティティ・sentinel error）→ `usecase`（`FileRepository`/`FileStorage` ポート定義・オーケストレーション）→ `handler`（`StrictServerInterface` 実装）、`infra`（PostgreSQL repo・ローカル FS storage）、`server`（DI 組み立て・ライフサイクル）の層分離を auth と同一パターンで踏襲 |
| IV. テスト駆動 | PASS | usecase 層 80%・handler 層 70% を目標に単体テスト（gomock）、repo 層は testcontainers-go、API 層は Schemathesis + Hurl を quickstart.md に手順化 |
| V. 型安全な SQL（sqlc） | PASS | `internal/infra/repo/queries/files.sql` に SQL を直書き、`internal/infra/repo/db/*.sql.go` を sqlc 生成（手編集禁止）。ORM 不使用 |
| VI. マイクロサービス境界 | PASS | 独立 Go module（`services/files/go.mod`）、独立 PostgreSQL DB、auth の `internal/` を import しない。共有コードは既存の `packages/authjwt` のみ再利用（新規共有コードは作らない） |
| VII. 認証 JWT (RS256) | PASS | Files は検証専用（`FILES_JWT_PUBLIC_KEY_PATH` で公開鍵読み込み）。秘密鍵は保持しない。`packages/authjwt.Middleware` を全ビジネスエンドポイントに適用 |

**違反なし。Complexity Tracking は不要。**

## Project Structure

### Documentation (this feature)

```text
specs/002-document-management/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── files-api-p1.md  # P1 スコープの索引（正本は schema/files/openapi.yaml）
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

**Structure Decision**: 既存の `services/<name>/` マイクロサービスレイアウト（`AGENTS.md` Project Layout）をそのまま採用する。`services/auth` と対称的な構成にし、共有コードは `packages/authjwt` のみを再利用する。

```text
services/files/
├── cmd/server/
│   └── main.go                       # 薄いエントリポイント（signal → server.Run）
├── internal/
│   ├── config/
│   │   └── config.go                 # FILES_SERVICE_PORT / FILES_DATABASE_URL /
│   │                                  # FILES_JWT_PUBLIC_KEY_PATH / FILES_STORAGE_DIR
│   ├── domain/
│   │   └── file.go                   # File エンティティ + sentinel error
│   │                                  # (ErrFileNotFound, ErrFileTooLarge, ErrFileEmpty)
│   ├── usecase/
│   │   ├── port.go                   # FileRepository / FileStorage インターフェース定義
│   │   ├── file_usecase.go           # UploadFile / ListFiles / GetFile / DownloadFile
│   │   ├── file_usecase_test.go
│   │   └── mock/                     # go:generate mockgen 出力
│   ├── handler/
│   │   ├── file_handler.go           # StrictServerInterface 実装（gen.*RequestObject ⇄ domain）
│   │   ├── file_handler_test.go
│   │   ├── health.go                 # /healthz, /readyz（DB 接続確認込み）
│   │   └── health_test.go
│   ├── infra/
│   │   ├── repo/
│   │   │   ├── queries/
│   │   │   │   └── files.sql         # CreateFile / GetFileByID / ListFiles（sqlc アノテーション）
│   │   │   ├── db/                   # sqlc 生成（手編集禁止）
│   │   │   ├── file_repository.go    # usecase.FileRepository の実装
│   │   │   └── file_repository_test.go  # testcontainers-go 統合テスト
│   │   └── storage/
│   │       ├── local.go              # usecase.FileStorage のローカル FS 実装
│   │       └── local_test.go
│   └── server/
│       ├── server.go                 # DI 組み立て・self-migration・echo 起動/終了
│       └── server_test.go
├── api/gen/
│   └── server.gen.go                 # 生成済み（本フェーズでの変更対象外）
├── migrations/
│   ├── 000001_create_files_table.up.sql
│   ├── 000001_create_files_table.down.sql
│   └── embed.go                      # go:embed によるバイナリ埋め込み（self-migration用）
├── oapi-codegen.yaml                 # 既存
├── sqlc.yaml                         # 新規（auth と同型）
├── Makefile                          # 既存（gen/build/run/test/migrate ターゲット定義済み）
├── Dockerfile                        # 新規（auth と同型、GOWORK=off + replace で packages/authjwt を解決）
├── .env.sample                       # 新規
├── go.mod / go.sum                   # 新規（独立 module）
└── (repository root)
    ├── go.work                       # ./services/files を use に追加
    └── compose.yaml                  # files サービス定義を追加（ポート 8082、ストレージ用ボリューム）
```

## Complexity Tracking

*本フェーズでは Constitution 違反なし。記載事項なし。*
