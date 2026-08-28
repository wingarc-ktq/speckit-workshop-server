# Phase 0 Research: Files サービス（P1 MVP）

**対象**: `specs/002-document-management/spec.md` の User Story 1〜3（P1）
**対象外**: User Story 4〜6（P2: タグ管理・ファイルメタデータ編集・ファイル削除）は本フェーズの実装スコープに含めない。

## 前提となる既存資産

- `schema/files/openapi.yaml`: P1 の 4 エンドポイントのみ定義済み（`GET/POST /files`, `GET /files/{fileId}`, `GET /files/{fileId}/content`）。タグ管理・編集・削除のエンドポイントは存在しない → OpenAPI が Single Source of Truth（Constitution II）である以上、これが P1 スコープの正となる。
- `services/files/api/gen/server.gen.go`: 上記 4 エンドポイント分の `StrictServerInterface` が生成済み（`oapi-codegen.yaml`: `strict-server: true`, `echo-server: true`, `embedded-spec: true`）。
- `services/auth`: 認証サービスのリファレンス実装。層構成・命名・エラーハンドリング・DI 配線パターンを Files でも踏襲する（`cmd/server/main.go` のコメントに「受講者が files サービスを実装する際の参考にしてください」と明記）。
- `packages/authjwt`: JWT 検証ミドルウェア（RS256 公開鍵検証）。Files サービスはこれをそのまま再利用する（Constitution VI: 共有コードは原則作らない、の例外として既に許容されている数少ない共有パッケージ）。

## Decision 1: サービス全体のアーキテクチャ

- **Decision**: `services/auth` と同一の層構成（domain / usecase / handler / infra / server）を採用し、独立した Go module（`services/files/go.mod`）として作成する。
- **Rationale**: Constitution III（クリーンアーキテクチャ）・VI（マイクロサービス境界）の要求、および auth との一貫性（レビューコスト低減、ワークショップの教材としての対称性）。
- **Alternatives considered**: auth と files で共通の internal ライブラリを作る案 → Constitution VI「共有コードは原則作らない」に反するため却下。

## Decision 2: 認証方式

- **Decision**: `packages/authjwt.Middleware(verifier)` を全 files エンドポイント（`/healthz`, `/readyz` を除く）に適用する。公開鍵は `FILES_JWT_PUBLIC_KEY_PATH` 環境変数（Constitution VII に明記済みの変数名）で読み込む。
- **Rationale**: Files は JWT の検証のみ行い、署名は Auth サービスのみが行う（秘密鍵は共有しない）という Constitution VII の原則をそのまま適用する。
- **Alternatives considered**: 独自の JWT 検証実装 → 車輪の再発明であり `packages/authjwt` が既に存在するため却下。

## Decision 3: ファイル本体の保存方式

- **Decision**: `usecase` 層に `FileStorage` ポート（`Save(ctx, key, io.Reader) error` / `Open(ctx, key) (io.ReadCloser, error)`）を定義し、P1 では `internal/infra/storage` にローカルファイルシステム実装（`FILES_STORAGE_DIR` 配下に `{fileID}` で保存）を用意する。
- **Rationale**: 仕様の Assumptions に「ファイルストレージはストレージインターフェースで抽象化する」と明記されている。ローカル FS 実装はワークショップの Docker Compose 構成で完結し（ボリュームマウントのみで永続化）、将来 S3 等に差し替える場合も `usecase` 以上の層に影響を与えない。
- **Alternatives considered**:
  - PostgreSQL の `bytea` にバイナリを直接格納 → 10MB 制限・同時アクセス 50 人程度の規模では致命的ではないが、「ストレージインターフェースで抽象化」という要求に対しては DB 実装がストレージ実装を兼ねる形になり、抽象境界が曖昧になるため却下。
  - S3 互換オブジェクトストレージ（MinIO 等） → P1 の要求(10MB, 50同時接続) に対して過剰、かつ Compose 構成が複雑化する。ポートで抽象化してあるため P2 以降で必要になれば無停止で差し替え可能。

## Decision 4: ファイルサイズ超過（10MB）の検出

- **Decision**: `UploadFile` ハンドラで `*multipart.Reader` を Part 単位でストリーム処理し、`file` パートを `io.LimitReader(part, maxFileSize+1)` でラップして読み込む。読み込みバイト数が `maxFileSize`（10MB）を超えた場合は `domain.ErrFileTooLarge` を返し 413 + `FILE_TOO_LARGE` を返す。加えて echo 側に `middleware.BodyLimit("12MB")`（multipart のオーバーヘッド分の余裕を見た粗いガード）を設定し、悪意のある巨大リクエストによるメモリ枯渇を防ぐ。
- **Rationale**: oapi-codegen の strict-server はバイナリを含む `multipart/form-data` に対して型付き構造体を生成せず `*multipart.Reader` を渡す（生成コードで確認済み）。そのため OpenAPI の `maxLength`/`format: binary` 制約だけでは実サイズを検証できず、ハンドラ側で明示的にチェックする必要がある。ストリーム処理により 10MB 超のファイルでも全量をメモリ/ディスクに書き切る前に打ち切れる。
- **Alternatives considered**: リクエストボディ全体をメモリに読み切ってから判定 → 大きいファイルで無駄なメモリ確保が発生するため却下。

## Decision 5: 一覧・検索のクエリ設計

- **Decision**: `internal/infra/repo/queries/files.sql` に 1 本の `ListFiles :many` クエリを定義し、`sqlc.narg` によるオプショナル引数（`search`, `tag_ids`）と `COUNT(*) OVER()` ウィンドウ関数で `total` を同一クエリから取得する。`ORDER BY` は `sort` 列挙値（`name` / `-name` / `uploadedAt` / `-uploadedAt` / `size` / `-size`）を Go 側でホワイトリスト検証した上で SQL 文字列の ORDER BY 句にマッピングする（値そのものをバインドせず、6 パターンを Go の switch で安全に組み立てる = SQL インジェクション回避）。
- **Rationale**: 往復回数を減らし SC-002/SC-003（500ms/1s 以内）を満たしやすくする。sqlc は動的 ORDER BY を生成できないため、許可された列挙値のみを Go 側で分岐させる方式が安全かつシンプル。
- **Alternatives considered**: `COUNT(*)` を別クエリで取得 → クエリ 2 回になり往復コストが増えるため却下。ORM の動的クエリビルダ → Constitution V で ORM 禁止。

## Decision 6: タグ（tagIds）の扱い（P1 スコープでの簡略化）

- **Decision**: `files` テーブルに `tag_ids UUID[] NOT NULL DEFAULT '{}'` 列を持たせ、アップロード時に受け取った `tagIds` をそのまま配列として保存する。P1 ではタグ自体（名前・色）の CRUD は実装しないため、`tag_ids` に対する外部キー制約は設けない。一覧の `tagIds` フィルタは `tag_ids && $1::uuid[]`（配列オーバーラップ演算子 + GIN インデックス）で実現する。
- **Rationale**: `schema/files/openapi.yaml`（P1 として既にマージ済みの契約）は `FileInfo.tagIds` を必須フィールドとして持つため、P1 の File エンティティとして扱う必要がある。一方で Tag エンティティの CRUD（FR-011〜016, User Story 4）は明確に P2 と定義されているため、本フェーズでは実装しない。配列列は「タグ ID の集合を保持する」という P1 の要求を満たしつつ、P2 で `tags` テーブル・中間テーブルへ移行する際も `file_tags(file_id, tag_id)` への移行パスを塞がない最小実装。
- **Alternatives considered**:
  - P1 で `tags` テーブルと中間テーブルまで作る → Tag CRUD が無い状態で FK 整合性を保証する経路が無く、YAGNI（Constitution 全体の思想「単一用途の抽象化を避ける」）に反するため却下。
  - `tagIds` を無視する（保存しない） → OpenAPI 契約（P1 契約）に反するため却下。

## Decision 7: テスト戦略

- **Decision**: Constitution IV に従い 3 層で実施する。
  - 単体テスト: `internal/usecase/*_test.go`（標準 `testing` + `go.uber.org/mock`、`FileRepository`/`FileStorage` をモック化）。カバレッジ目標 80%。
  - 統合テスト: `internal/infra/repo/*_test.go`（`testcontainers-go` で実 PostgreSQL 17 を起動し `ListFiles` の検索・フィルタ・ページネーション・ソートを検証）。
  - API テスト: Schemathesis（`schema/files/openapi.yaml` 駆動のファジング）+ Hurl（アップロード→一覧→詳細→ダウンロードのシナリオ）。起動済みサービスに対して CI もしくは手動で実行する（`quickstart.md` に手順を記載）。
- **Rationale**: Constitution 必須要件。auth サービスの既存テスト構成（`*_test.go` の配置、mock の `go:generate` パターン）をそのまま踏襲する。
- **Alternatives considered**: testify の利用 → Constitution I/AGENTS.md で明示的に禁止。

## Decision 8: Docker / Compose 統合

- **Decision**: `services/auth/Dockerfile` と同型の Dockerfile を `services/files/Dockerfile` に作成し、`compose.yaml` の `# files サービスは未実装...` コメント部分を `files` サービス定義に置き換える。ポートは OpenAPI の `servers` で宣言済みの `8082` を使用し、ファイル本体の永続化用ボリューム（例: `./files-data:/data`）をマウントして `FILES_STORAGE_DIR=/data` を設定する。
- **Rationale**: auth との対称性、および OpenAPI に既に `http://localhost:8082/api/v1` と明記されているため。
- **Alternatives considered**: なし（契約上ポートが固定されている）。

## 未解決事項

なし（Technical Context の NEEDS CLARIFICATION はすべて解消済み）。
