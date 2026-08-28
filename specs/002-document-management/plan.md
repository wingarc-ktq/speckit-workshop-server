# 実装計画: Files マイクロサービス

**Feature**: 文書管理 (Document Management)
**Spec**: `specs/002-document-management/spec.md`
**OpenAPI**: `schema/files/openapi.yaml`

## 1. 概要

本計画は、文書管理マイクロサービス（`files`サービス）のバックエンドAPI実装に関するものです。
仕様書とOpenAPI定義に基づき、Go言語とクリーンアーキテクチャを用いて、MVP（Minimum Viable Product）機能であるファイルアップロード、一覧取得・検索、詳細取得・ダウンロードを実装します。

## 2. 技術コンテキスト

| カテゴリ | 技術 | 備考 |
| :--- | :--- | :--- |
| 言語 | Go 1.26 | プロジェクト標準 |
| フレームワーク | echo/v4 | プロジェクト標準 |
| API仕様 | OpenAPI 3.0 | `oapi-codegen` でサーバーインターフェースと型を生成 |
| データベース | PostgreSQL 16 | プロジェクト標準 |
| DBアクセス | pgx/v5, sqlc | SQLファーストで型安全なコードを生成 |
| 認証 | JWT (RS256) | `packages/authjwt` の共有ミドルウェアで検証 |
| テスト | 標準 `testing`, `go.uber.org/mock`, `testcontainers-go` | 単体・統合テスト |
| ファイルストレージ | ローカルファイルシステム | **NEEDS CLARIFICATION**: 本番環境ではオブジェクトストレージ（S3等）を想定。今回はインターフェースで抽象化し、開発用にローカルストレージ実装を提供。 |

## 3. 憲法（Constitution）チェック

| # | 原則 | 遵守状況 | 備考 |
| :- | :--- | :--- | :--- |
| I | Go Idiomatic Code | ✅ 遵守 | `gofmt`, `go vet` を適用 |
| II | OpenAPI ファースト | ✅ 遵守 | `schema/files/openapi.yaml` を正とする |
| III | クリーンアーキテクチャ | ✅ 遵守 | `handler`, `usecase`, `domain`, `infra` の層分離 |
| IV | テスト駆動 | ✅ 遵守 | 単体・統合・APIテストを実装 |
| V | 型安全な SQL (sqlc) | ✅ 遵守 | `sqlc` を用いてDBアクセスコードを生成 |
| VI | マイクロサービス境界 | ✅ 遵守 | `auth` サービスとは独立したモジュールとして実装 |
| VII | 認証は JWT (Bearer) | ✅ 遵守 | `packages/authjwt` を利用して公開鍵で検証 |

## 4. MVPスコープ (P1機能)

本計画では、以下のMVP機能に焦点を当てます。

- **User Story 1: ファイルアップロード**
  - `POST /files`
- **User Story 2: ファイル一覧取得と検索**
  - `GET /files` (ページネーション、キーワード検索)
- **User Story 3: ファイル詳細取得とダウンロード**
  - `GET /files/{fileId}`
  - `GET /files/{fileId}/download`

P2機能（タグ管理、メタデータ編集、ファイル削除）は本計画のスコープ外とします。

## 5. 実装フェーズ

### Phase 0: 準備

1.  **プロジェクトセットアップ**:
    -   `services/auth` を参考に `services/files` ディレクトリを作成。
    -   `go.mod`, `Makefile`, `oapi-codegen.yaml`, `sqlc.yaml` 等の設定ファイルをコピー・修正。
    -   ルートの `go.work` に `services/files` を追加。
2.  **コード生成 (oapi-codegen)**:
    -   `make gen-oapi` を実行し、`api/gen/server.gen.go` を生成。

### Phase 1: データベース設計とマイグレーション

1.  **DBスキーマ定義**:
    -   `files` テーブルを定義するマイグレーションファイルを作成 (`migrations/000001_create_files.up.sql`)。
    -   `files` テーブル: `id`, `name`, `size`, `mime_type`, `description`, `uploaded_at`
    -   `tags` との関連は多対多になるが、MVPではタグ機能を実装しないため、中間テーブルは後続フェーズで作成。
2.  **SQLクエリ作成**:
    -   `internal/infra/repo/queries/files.sql` に、CRUD操作に対応するクエリを記述。
3.  **コード生成 (sqlc)**:
    -   `make gen-sqlc` を実行し、`internal/infra/repo/db/*.sql.go` を生成。

### Phase 2: ドメインとユースケースの実装

1.  **ドメイン層 (`internal/domain`)**:
    -   `file.go`: `File` エンティティ、`FileRepository` インターフェース、ドメインエラー (`ErrFileNotFound` 等) を定義。
    -   `storage.go`: `FileStorage` インターフェースを定義し、ファイル保存ロジックを抽象化。
2.  **ユースケース層 (`internal/usecase`)**:
    -   `file_usecase.go`: `FileUsecase` を実装。`FileRepository` と `FileStorage` に依存。
        -   `UploadFile`: ファイル保存とDB登録のロジック。
        -   `ListFiles`: ファイル一覧取得と検索のロジック。
        -   `GetFile`: ファイル詳細取得のロジック。
        -   `DownloadFile`: ファイル取得のロジック。
    -   **単体テスト**: `go.uber.org/mock` を用いてリポジトリとストレージをモック化し、テーブル駆動テストを作成。

### Phase 3: インフラとハンドラの実装

1.  **インフラ層 (`internal/infra`)**:
    -   `file_repository.go`: `FileRepository` インターフェースを実装。`sqlc` が生成したコードを利用。
    -   `local_storage.go`: `FileStorage` インターフェースを実装。ローカルファイルシステムにファイルを保存。
    -   **統合テスト**: `testcontainers-go` を用いて、実際のPostgreSQLコンテナに対してリポジトリ層のテストを実施。
2.  **ハンドラ層 (`internal/handler`)**:
    -   `file_handler.go`: `oapi-codegen` が生成した `ServerInterface` を実装。
    -   各ハンドラメソッドは、リクエストをユースケースの入力に変換し、ユースケースを呼び出す。
    -   ユースケースからの結果をHTTPレスポンスに変換する。
3.  **サーバーセットアップ (`internal/server`, `cmd/server`)**:
    -   `internal/server/server.go`: 依存性注入（DI）を行い、echoインスタンス、ミドルウェア（JWT検証等）、ルーティングをセットアップ。
    -   `cmd/server/main.go`: `server.Run()` を呼び出す薄いエントリポイント。

## 6. テスト戦略

- **単体テスト**: ユースケース層のビジネスロジックを `go.uber.org/mock` を使ってテストする。
- **統合テスト**: インフラ層（特にリポジトリ）を `testcontainers-go` を使って実際のDBに対してテストする。
- **APIテスト (Day 3)**: `Schemathesis` でOpenAPIスキーマ準拠を、`Hurl` でユーザーシナリオをテストする。

## 7. 成果物

- `services/files/` 以下のGoソースコード
- DBマイグレーションファイル
- SQLクエリファイル
- 単体テスト・統合テストコード
- `data-model.md`
- `research.md`

