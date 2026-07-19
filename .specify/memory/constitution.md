<!--
Sync Impact Report (現行 v1.1.3 / 直近の変更):
- 1.1.3 (PATCH) 原則 IV / 技術スタック: 単体テストを testify 併用 → 標準 testing + uber/mock に統一 (AGENTS.md・実装と整合)
- 1.1.2 (PATCH) 原則 II: 運用エンドポイント (/healthz・/readyz・/metrics 等) を OpenAPI 契約の対象外と明記
- 1.1.1 (PATCH) 原則 III: 合成ルート (DI 組立・サーバ起動) を internal/server に集約し cmd を薄いシムに
- 1.1.0 (MINOR) 原則 VII: JWT を HS256 共有シークレット → RS256 非対称鍵へ (TokenIssuer/TokenVerifier ポートで infra に隠蔽)
-->

# Spec Kit Workshop (Server) Constitution

## Core Principles

### I. Go Idiomatic Code (NON-NEGOTIABLE)

すべての Go コードは標準的な慣習に従わなければならない。

**Rules**:

- `gofmt` / `goimports` でフォーマット済みであること（CI で検証）
- `go vet ./...` がエラー無しで通ること
- 変数・関数の命名は Go 公式の [Effective Go](https://go.dev/doc/effective_go) に準拠
- パッケージ名は単数形・小文字・1単語
- 戻り値の `error` は最後の戻り値として配置
- パブリック API（公開シンボル）には GoDoc コメント必須

**Rationale**: Go は規約が厳格であり、慣習に従うことで読みやすさとレビューコストを下げる。

### II. OpenAPI ファースト (NON-NEGOTIABLE)

すべての HTTP API は実装前に OpenAPI 3.x 仕様書として定義しなければならない。

**Rules**:

- API 仕様は `schema/<service>/openapi.yaml` に定義
- ハンドラインターフェース・型・リクエスト検証ミドルウェアは [oapi-codegen v2](https://github.com/oapi-codegen/oapi-codegen) で生成
- 生成されたファイルは手動編集禁止
- スキーマ変更は必ず `openapi.yaml` を先に更新し、`make gen` で再生成
- 各エンドポイントには `summary`, `operationId`, レスポンス例を必ず記述
- 例外: 運用エンドポイント（`/healthz`・`/readyz`・`/metrics` 等）はビジネス API ではないため OpenAPI 契約の対象外とする。`/api/v1` 配下に置かず、検証ミドルウェアからも除外する

**Rationale**: OpenAPI を Single Source of Truth とすることで、サーバー実装・クライアント・テストの整合性が保たれる。

### III. クリーンアーキテクチャ（層分離）

各サービスはクリーンアーキテクチャの原則で層を分離する。

**Layers**（依存方向は内側へ）:

```
handler  →  usecase  →  domain  ←  infrastructure
(HTTP)      (アプリケーションロジック)   (DB, 外部API)
```

**Rules**:

- `internal/domain/`: ドメインモデル・ビジネスルール・エラー型・**インターフェース定義**。外部依存禁止
- `internal/usecase/`: ユースケース。`domain` のインターフェースに依存し、infrastructure は注入される
- `internal/handler/`: HTTP ハンドラ。oapi-codegen の `ServerInterface` を実装。リクエストの検証・変換・レスポンス整形のみ
- `internal/infra/`: PostgreSQL リポジトリ実装、外部 API クライアントなど。`domain` のインターフェースを実装
- `internal/server/`: 依存性注入（DI）の組み立て・echo セットアップ・サーバーのライフサイクル（起動・graceful shutdown）。`Run(ctx)` を公開
- `cmd/server/main.go`: 薄いエントリポイント。シグナル context を生成して `internal/server.Run` を呼ぶだけ

**Rationale**: 層を明確にすることで、テスタビリティ（特に usecase の単体テスト）と技術スタックの差し替え可能性が高まる。

### IV. テスト駆動 (NON-NEGOTIABLE)

テストの無いコードはマージしない。

**Rules**:

- **単体テスト**: usecase / domain ロジックは `*_test.go` で標準 `testing` + uber/mock (gomock) を使ったテストを必ず書く（testify は使わない。`if err != nil { t.Fatal(err) }` / `if got != want { t.Errorf(...) }`）
- **統合テスト**: リポジトリ層は [testcontainers-go](https://golang.testcontainers.org/) で本物の PostgreSQL を立ててテスト
- **API テスト**: Schemathesis (OpenAPI 駆動) + Hurl (シナリオ) で起動済みサービスを検証
- カバレッジ目標: usecase 層 80%以上、handler 層 70%以上
- テストファイル名は対象ファイル + `_test.go`（例: `file_usecase.go` → `file_usecase_test.go`）
- テーブル駆動テスト（`tests := []struct{...}{...}`）を推奨

**Rationale**: 多層テストにより、ユニット粒度の正しさとシステム全体の振る舞いの両方を保証する。

### V. 型安全な SQL（sqlc）

SQL は手書きしつつ、Go コードへのバインドは [sqlc](https://sqlc.dev/) で生成する。

**Rules**:

- クエリは `services/<service>/internal/infra/repo/queries/*.sql` に記述
- スキーマ DDL は `services/<service>/migrations/` 配下に `golang-migrate` 形式で配置
- クエリには `-- name: QueryName :one|:many|:exec` 注釈をつける
- 生成された Go ファイル（`*.sql.go`）は手動編集禁止
- ORM（GORM 等）は使用しない

**Rationale**: SQL を直接書くことで PostgreSQL の機能を活用しつつ、型安全性を `sqlc` で確保する。OpenAPI → Go と同じ「スキーマからコード生成」の哲学が一貫する。

### VI. マイクロサービス境界

サービス間の関心事を明確に分離する。

**Rules**:

- 各サービスは独立した Go モジュール（`go.mod`）として配置
- 各サービスは独自の PostgreSQL データベースを持つ（共有 DB 禁止）
- サービス間通信は HTTP（OpenAPI 定義済み）または検証された JWT トークンの引き回しに限定
- 共有コードは原則作らない（重複を許容してでも独立性を優先）
- どうしても必要な場合のみ、`/pkg/shared/` に配置を検討

**Rationale**: サービス独立性を担保することで、デプロイ・スケーリング・障害分離が可能になる。共有コードはマイクロサービスの最大の落とし穴。

### VII. 認証は JWT (Bearer / RS256)

認証は JWT (RS256 / 非対称鍵) で統一する。

**Rules**:

- Auth サービスがログイン時に**秘密鍵**で JWT を署名・発行する
- Files など他サービスは**公開鍵**で JWT を検証する。秘密鍵は共有しない（秘密鍵を持つのは Auth のみ）
- JWT の発行は usecase が依存する `TokenIssuer`、検証は `TokenVerifier` ポート経由とし、署名方式 (RS256) の具象は infra に隠蔽する（第III原則と整合）
- JWT 検証は echo ミドルウェアとして実装
- 鍵は PEM ファイルのパスを環境変数で渡す（`AUTH_JWT_PRIVATE_KEY_PATH` / `AUTH_JWT_PUBLIC_KEY_PATH` / `FILES_JWT_PUBLIC_KEY_PATH`）
- 開発用の鍵ペアは `make keys` で生成し、**git 管理しない**（秘密鍵をコミットしない）
- トークンの有効期限・クレーム構造・`bearerAuth` セキュリティスキームは `schema/auth/openapi.yaml` に定義

**Rationale**: 非対称鍵 (RS256) にすることで、署名できるのは秘密鍵を持つ Auth サービスのみとなり、検証側サービスに秘密鍵を配布する必要がなくなる。これは共有シークレット (HS256) よりマイクロサービスの信頼境界として安全で、サービス独立性（第VI原則）とも整合する。

## 技術スタック

| レイヤ | 採用技術 |
|---|---|
| 言語 | Go 1.26 |
| Web フレームワーク | echo/v4 |
| OpenAPI 生成 | oapi-codegen v2 + echo-middleware |
| OpenAPI 検証 | kin-openapi |
| DB ドライバ | jackc/pgx/v5 |
| SQL コード生成 | sqlc |
| マイグレーション | golang-migrate/v4 |
| 認証 | golang-jwt/v5（JWT RS256 / 非対称鍵） |
| 単体テスト | 標準 testing + uber/mock (gomock) |
| 統合テスト | testcontainers-go (PostgreSQL) |
| API テスト | Schemathesis + Hurl |
| データストア | PostgreSQL 17 |

## 開発ワークフロー

### 機能ブランチ

`feature/[番号-機能名]` 形式で spec-kit ワークフローに従う。

### マージ前チェック

- [ ] `make lint` 通過
- [ ] `make fmt` 適用済み
- [ ] `make test-unit` 通過
- [ ] `make test-integration` 通過
- [ ] OpenAPI 変更時は `make gen` 再実行 + コミット

## ガバナンス

このConstitutionは spec-kit ワークフロー上のすべての判断に優先する。
変更は PR + チームレビューで承認すること。

**Version**: 1.1.3 | **Ratified**: 2026-05-07 | **Last Amended**: 2026-07-19
