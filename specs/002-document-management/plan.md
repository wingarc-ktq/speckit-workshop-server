# Implementation Plan: 文書管理（Files MVP）

**Branch**: `creeper` | **Date**: 2026-08-28 | **Spec**: [spec.md](./spec.md)

**Input**: `/specs/002-document-management/spec.md` と `schema/files/openapi.yaml`

## Summary

Files サービスの P1 MVP として、認証済みユーザーがファイルをアップロードし、メタデータを検索・一覧表示し、詳細取得とダウンロードを行える API を実装する。PostgreSQL にメタデータ、ローカルファイルストレージ実装に本体を保存し、`FileStorage` ポートで将来の差し替えに備える。タグ CRUD、ファイル編集、削除は P2 として除外するが、P1 のタグフィルタを成立させるためアップロード時の任意 `tagIds` を受け付ける。

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: Echo v4, oapi-codegen/v2, kin-openapi, pgx/v5, sqlc, golang-migrate/v4, golang-jwt/v5, `packages/authjwt`

**Storage**: Files 専用 PostgreSQL 17（メタデータ・関連）+ `internal/infra/storage` のローカルファイルシステム（本体）

**Testing**: 標準 `testing` + uber/mock、testcontainers-go PostgreSQL、Schemathesis、Hurl

**Target Platform**: Linux コンテナ上の HTTP マイクロサービス

**Project Type**: Go web service / microservice

**Performance Goals**: 5MB アップロード 5 秒以内、100 件の一覧 500ms 以内、キーワード検索 1 秒以内

**Constraints**: 1 ファイル最大 10MB、全ビジネス API は JWT RS256、共通 `code`/`message` エラー、全ユーザーが全ファイルを操作可能

**Scale/Scope**: 同時アクセス最大 50 ユーザー、P1 の upload/list/detail/download と health/ready。P2（タグ CRUD、編集、削除）は対象外

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Go idiomatic code: `gofmt`、`go vet`、公開シンボルの GoDoc
- [x] OpenAPI first: `schema/files/openapi.yaml` を先に完成させ、`make gen` で生成物を作成（現状は 0 行のため実装開始前に更新必須）
- [x] Clean Architecture: `handler → usecase → domain ← infra`、DI は `internal/server`
- [x] Test driven: usecase/domain の unit、repository の PostgreSQL integration、API の Schemathesis/Hurl
- [x] Type-safe SQL: migrations と sqlc query を SoT とし、生成コードを手編集しない
- [x] Microservice boundary: Files 専用 DB、Auth の internal package を直接 import しない
- [x] JWT RS256: `packages/authjwt` で公開鍵検証、秘密鍵は Files に持ち込まない

**Gate status**: PASS（OpenAPI の空ファイルは Phase 1 の実装前提条件として解消する）

## Project Structure

### Documentation

```text
specs/002-document-management/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/files-api.md
```

### Source Code

```text
schema/files/
└── openapi.yaml                         # P1 API の唯一の契約

services/files/
├── cmd/server/
│   └── main.go                          # signal context を作り server.Run を呼ぶ薄い shim
├── api/
│   └── gen/
│       └── server.gen.go                # oapi-codegen 生成物、手編集禁止
├── internal/
│   ├── config/
│   │   ├── config.go                    # DB、port、JWT 公開鍵、storage root、制限値
│   │   └── config_test.go
│   ├── domain/
│   │   ├── file.go                      # File エンティティとドメイン制約
│   │   ├── tag.go                       # Tag 値と色定義（P1 は参照のみ）
│   │   ├── errors.go                    # ErrFileNotFound、ErrTagNotFound 等
│   │   └── file_test.go
│   ├── usecase/
│   │   ├── port.go                      # FileRepository、TagRepository、FileStorage
│   │   ├── file_usecase.go              # upload/list/get/download の orchestration
│   │   ├── file_usecase_test.go         # mock を使う P1 主ユニットテスト
│   │   └── mock/
│   │       └── *_mock.go                # go:generate / mockgen 生成物
│   ├── handler/
│   │   ├── file_handler.go              # ServerInterface の P1 実装
│   │   ├── file_mapper.go               # gen 型と domain/usecase 型の変換
│   │   ├── error_response.go             # domain error → code/message/status
│   │   ├── health.go                    # 認証不要の healthz/readyz
│   │   └── file_handler_test.go
│   ├── infra/
│   │   ├── repo/
│   │   │   ├── file_repository.go       # metadata の永続化、一覧・検索・件数
│   │   │   ├── tag_repository.go        # tagIds の存在確認と関連登録
│   │   │   ├── repo_mapper.go           # sqlc 型と domain 型の変換
│   │   │   ├── repository_test.go       # PostgreSQL を使う integration test
│   │   │   ├── queries/
│   │   │   │   ├── files.sql            # insert/get/list/count と tag filter
│   │   │   │   └── tags.sql             # tag 存在確認、FileTag 登録・取得
│   │   │   └── db/                      # sqlc 生成物、手編集禁止
│   │   └── storage/
│   │       ├── storage.go               # UUID storage key のローカル FS adapter
│   │       ├── storage_test.go          # temp directory 上の保存・読出し
│   │       └── path.go                  # root 配下限定、パス traversal 防止
│   └── server/
│       ├── server.go                    # DI、Echo、middleware、lifecycle
│       ├── openapi.go                   # embedded spec の validator 設定
│       └── server_test.go               # route、health、auth middleware wiring
├── migrations/
│   ├── embed.go                         # migration FS の埋め込み
│   ├── 000001_create_files_table.up.sql
│   ├── 000001_create_files_table.down.sql
│   ├── 000002_create_tags_table.up.sql
│   ├── 000002_create_tags_table.down.sql
│   ├── 000003_create_file_tags_table.up.sql
│   └── 000003_create_file_tags_table.down.sql
├── api-test/
│   ├── files.hurl                      # upload/list/detail/download の固定シナリオ
│   └── fixtures/                       # API テスト用ファイル
├── oapi-codegen.yaml
├── sqlc.yaml
├── go.mod
├── go.sum
└── Makefile
```

**責務と依存方向**:

| 層 | 主な責務 | 依存してよい対象 | 依存してはいけない対象 |
|---|---|---|---|
| `domain` | File/Tag、制約、センチネルエラー、ポートの契約 | 標準ライブラリ、UUID | Echo、pgx、sqlc、filesystem |
| `usecase` | upload/list/get/download、補償削除、ページング入力の意味付け | `domain` のポート | Echo、`infra` 具象、生成 API 型 |
| `handler` | multipart/query/path の変換、レスポンス整形、HTTP エラー | `usecase`、`domain`、`api/gen` | SQL、filesystem 直接操作 |
| `infra/repo` | PostgreSQL、sqlc、transaction、検索・件数 | `domain`、pgx、sqlc 生成物 | handler、usecase の具象 |
| `infra/storage` | 本体の stream 保存・読出し・補償削除 | `usecase` の storage port、OS filesystem | HTTP、DB |
| `server` | config、migration、pool、JWT verifier、DI、route | 全 adapter と Echo | ビジネスロジックの追加 |

**リクエストの処理経路**:

1. Echo が `/api/v1` の request を OpenAPI validator で検証する。
2. `authjwt.Middleware` が Files 用 JWT 公開鍵で検証し、`userID` を context に格納する。
3. handler が multipart/query/path を usecase の入力型へ変換する。
4. upload は storage 保存 → DB transaction（File と FileTag）→ 失敗時 storage delete の順で処理する。
5. list は name の部分一致、tagIds の AND フィルタ、`COUNT(*)`、limit/offset を repository に渡す。
6. detail は metadata と tag IDs を返し、download は storage key で本体をストリーミングする。

**テスト配置**:

- `domain/*_test.go`: 10 MiB、description 500 文字、空名、tag 制約。
- `usecase/file_usecase_test.go`: 成功、未検出、サイズ超過、storage/DB 片側失敗、補償削除、list 条件。
- `handler/file_handler_test.go`: multipart/query/path の変換、共通エラー、レスポンス形状。
- `infra/repo/*_test.go`: `integration` build tag 付きで migration、insert、検索、tag filter、pagination、count。
- `infra/storage/storage_test.go`: 一時ディレクトリ、UUID key、読出し、削除、root 外アクセス拒否。
- `internal/server/server_test.go`: `/healthz`、`/readyz`、JWT 未指定/無効、route wiring。
- `api-test/files.hurl` と Schemathesis: OpenAPI 契約と upload → list → detail → download の E2E。

**Structure Decision**: 既存 Auth サービスと同じ独立 Go モジュール構成を Files に適用する。HTTP 変換は handler、ユースケースとポートは内側、PostgreSQL/sqlc とファイルシステムは infra、合成と lifecycle は server に置く。契約と生成コードはそれぞれ `schema/files/openapi.yaml` と `services/files/api/gen/` に分離する。P1 のタグフィルタ用タグは migration/fixture で投入し、タグ管理 API 自体は追加しない。

## Phase 0: Research

調査結果は [research.md](./research.md) に記録した。既存 Auth の Echo/oapi-codegen、JWT、migration、health/ready、テストパターンを再利用し、MVP の本体保存はローカルストレージ adapter とする。`schema/files/openapi.yaml` が空であるため、契約を先に作成してから生成を実行する。

## Phase 1: Design

- [data-model.md](./data-model.md): File、Tag、FileTag、保存キーと制約
- [contracts/files-api.md](./contracts/files-api.md): P1 HTTP 契約とエラー、P2 除外範囲
- [quickstart.md](./quickstart.md): DB、鍵、生成、テスト、API シナリオ

## Post-Design Constitution Check

- [x] OpenAPI 契約を `schema/files/openapi.yaml` に反映してからコード生成する
- [x] P1 の各操作に handler/usecase/infra/unit・integration・API テストの責務を割り当てる
- [x] storage は port 経由で usecase から利用し、ユーザー入力をパスにしない
- [x] upload の DB 失敗時は保存済み本体を補償削除する
- [x] P2 操作を実装範囲へ混入させない

**Post-design gate status**: PASS
