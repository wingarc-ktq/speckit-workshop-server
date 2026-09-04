# Research: Files サービス MVP

**Feature**: 002-document-management  
**調査日**: 2026-08-28  
**対象**: P1（アップロード、一覧/検索/タグフィルタ、詳細、ダウンロード、JWT、health/ready）

## 調査サマリー

- Auth は `handler → usecase → domain ← infra` のクリーンアーキテクチャで実装されている。Files も同じ層構成を採用する。
- HTTP 契約は OpenAPI を Single Source of Truth とし、oapi-codegen の Echo `ServerInterface`、埋め込み spec、kin-openapi ベースの検証を使う。
- JWT は Auth が秘密鍵で発行し、Files は共有パッケージ `packages/authjwt` と公開鍵だけで検証する。秘密鍵を Files に配布しない。
- DB は Files 専用 PostgreSQL とし、SQL は sqlc で生成する。Auth の `internal/` は import しない。
- 既存コードにファイル本体ストレージの抽象化や実装は存在しない。MVP では `FileStorage` ポートとローカルファイルシステム実装を分離し、メタデータは PostgreSQL に保存する構成を推奨する。
- `schema/files/openapi.yaml` は存在するが、2026-08-28 の調査時点で **0 行（空ファイル）**。実装・コード生成の前に P1 契約を定義する必要がある。

## 1. Files のサービス構成

### Decision

`services/files/` に Auth と同じサービス構成を作る。

```text
services/files/
├── cmd/server/main.go
├── internal/
│   ├── config/
│   ├── domain/
│   ├── usecase/
│   ├── handler/
│   ├── infra/
│   │   ├── repo/
│   │   └── storage/
│   └── server/
├── api/gen/
├── migrations/
├── oapi-codegen.yaml
├── sqlc.yaml
└── go.mod
```

### Rationale

- Auth の `internal/server` が DI、Echo、ミドルウェア、ルーティング、graceful shutdown を一箇所に集約しているため、Files でも同じ配線パターンを再利用できる [`services/auth/internal/server/server.go:1-6`, `services/auth/internal/server/server.go:88-151`]。
- `cmd/server/main.go` はシグナルから context を作り `server.Run` を呼ぶだけの薄いエントリポイントである [`services/auth/cmd/server/main.go:1-29`]。
- Constitution は `handler → usecase → domain ← infrastructure` の依存方向と、`internal/domain` の外部依存排除を要求している [` .specify/memory/constitution.md:56-83`]。

### Alternatives

- **Auth のコードを Files から直接参照する**: `internal/` は Go の import 制約とマイクロサービス境界に反し、Auth の DB モデルやユースケースへ結合するため不採用。
- **単一パッケージに全実装を置く**: 小規模には見えるが、ストレージ差し替え、ユースケース単体テスト、API 層の責務分離が困難になるため不採用。

## 2. OpenAPI と HTTP ハンドラ

### Decision

先に `schema/files/openapi.yaml` を P1 の契約として定義し、`services/files/oapi-codegen.yaml` で `ServerInterface`、モデル、埋め込み spec を生成する。ハンドラは生成された通常の Echo `ServerInterface` を実装し、リクエストの bind、usecase 入力への変換、レスポンス整形、ドメインエラーの HTTP マッピングだけを担当する。

P1 の契約には少なくとも次を定義する。

- `POST /files`: multipart upload（必須 file、任意 description、最大 10MB）
- `GET /files`: page/pageSize、keyword、複数 tag ID の一覧・検索・フィルタ
- `GET /files/{id}`: メタデータ詳細
- `GET /files/{id}/download`: ファイル本体のストリーミング
- 全ビジネスエンドポイントの `bearerAuth`
- 共通 `ErrorResponse`（`code` + `message`）

運用エンドポイントは Auth と同じく `/healthz`、`/readyz` を `/api/v1` の外側に置き、OpenAPI 検証 middleware の対象外にする。

### Rationale

- Auth の生成設定は `models: true`、`echo-server: true`、`embedded-spec: true`、`strict-server: true` である [`services/auth/oapi-codegen.yaml:1-10`]。Files の Makefile も同じ生成方針を前提にしている [`services/files/Makefile:1-22`]。
- Auth は kin-openapi の validator を全体 middleware として登録し、`/api/v1` 外を skipper で除外している [`services/auth/internal/server/server.go:113-151`]。
- Auth handler は `ctx.Bind` → usecase 呼び出し → `gen.*` レスポンス変換を行い、ドメインエラーを HTTP status と共通エラーコードへ写像している [`services/auth/internal/handler/auth_handler.go:25-123`]。
- OpenAPI ファーストは Constitution の必須原則であり、生成コードを手編集せず schema 更新後に `make gen` を実行する [` .specify/memory/constitution.md:29-53`]。
- **重要な現状**: `schema/files/openapi.yaml` はファイル自体は存在するものの、PowerShell で確認した行数は `LINES=0`。Files の `make gen-oapi` も空 spec では API 型を生成できないため、実装開始前に契約を作成する必要がある [`services/files/Makefile:10-18`]。

### Alternatives

- **`StrictServerInterface` を採用する**: 自動 bind の利点はあるが、既存 Auth の学習・実装パターンと異なり、Echo context に格納した JWT user ID や multipart/streaming の扱いが複雑になるため、まずは通常の `ServerInterface` を推奨。
- **OpenAPI validator を導入しない**: handler 内の個別検証は可能だが、仕様と実装の二重管理になり、Files MVP のページネーション・10MB・multipart 制約を一貫して検証しにくいため不採用。

## 3. JWT 認証

### Decision

Files は `packages/authjwt` の `Verifier` を使い、`FILES_JWT_PUBLIC_KEY_PATH` から公開鍵を読み込んで Echo middleware を構成する。JWT middleware は全 P1 ビジネスルートに適用し、検証済み `sub` の UUID を context から handler/usecase に渡す。Auth の秘密鍵は Files に渡さない。

### Rationale

- 共有パッケージは公開鍵で RS256 を検証し、署名能力を持たない設計である [`packages/authjwt/verifier.go:1-30`]。
- middleware は `Authorization: Bearer <token>` を検証し、成功時に user ID を Echo context に保存し、未指定・不正・期限切れは 401 を返す [`packages/authjwt/middleware.go:9-58`]。
- Auth の JWT 実装は秘密鍵で Issue し、共有 verifier を埋め込んで検証する。署名方式の具体実装は usecase の `TokenIssuer` から隠蔽されている [`services/auth/internal/infra/token/jwt.go:1-73`, `services/auth/internal/usecase/port.go:20-39`]。
- Auth の server 配線は `getCurrentUser` に operation middleware を適用している [`services/auth/internal/server/server.go:126-145`]。Files では P1 の upload/list/detail/download の各 operation に同じ考え方で適用する。
- Auth の JWT verifier テストは成功、期限切れ、別鍵、壊れた token、不正な `sub`、RS256 以外を検証している [`packages/authjwt/verifier_test.go:17-107`]。middleware テストは header 不在、Bearer 形式不正、検証失敗、next 未呼び出しを検証している [`packages/authjwt/middleware_test.go:24-96`]。

### Alternatives

- **HS256 の共有 secret**: Files に署名可能な secret を配布することになり、Auth だけが発行者であるという信頼境界を壊すため不採用。
- **Files が Auth の `/auth/me` を毎回呼ぶ**: ネットワーク依存・レイテンシ・障害連鎖が増える。Files が必要とする user ID は JWT の検証で得られるため不採用。
- **handler 内で JWT を直接検証する**: ルートごとの重複と認証漏れを招き、Constitution の middleware 方針に反するため不採用。

## 4. ドメイン、usecase、repository

### Decision

Files の usecase 層に、少なくとも次のポートを定義する。

- `FileRepository`: メタデータの create/list/find と必要な削除・関連検索
- `FileStorage`: ファイル本体の保存、取得、削除
- 必要に応じて `TagRepository` またはタグ ID 検証用の repository

usecase は `userID` を必ず入力に含め、MVP の仕様どおり全ユーザー共有であっても、将来の所有者/権限条件を SQL と API 境界に追加できる形にする。DB 実装は `internal/infra/repo` で sqlc 生成コードをラップし、domain 型との変換をそこで吸収する。

### Rationale

- Auth の usecase port は `UserRepository`、`PasswordHasher`、`TokenIssuer` を interface として定義し、具体実装を注入している [`services/auth/internal/usecase/port.go:1-39`]。
- Auth usecase は repository・hasher・token にだけ依存し、bcrypt/JWT の具体技術を直接参照しない [`services/auth/internal/usecase/auth_usecase.go:18-50`]。
- Auth repository は sqlc の `db.Queries` をラップし、`pgtype.UUID` と domain の `uuid.UUID` を infra 内で変換する [`services/auth/internal/infra/repo/user_repository.go:17-31`, `services/auth/internal/infra/repo/user_repository.go:39-84`]。
- SQL は `queries/*.sql` に置き、`-- name: ...` 注釈で sqlc 生成する方針である [`services/auth/internal/infra/repo/queries/users.sql:1-18`, `.specify/memory/constitution.md:101-117`]。
- Files の Makefile は sqlc と mock 生成を `make gen` に含めている [`services/files/Makefile:1-22`]。

### Alternatives

- **usecase から sqlc を直接呼ぶ**: DB 型が内側の層へ漏れ、モック差し替えと単体テストが難しくなるため不採用。
- **ファイル本体を PostgreSQL bytea に保存する**: 10MB MVP では動作可能だが、DB バックアップ・接続プール・レスポンスストリーミングへの負荷が増え、ストレージ交換も難しくなるため第一候補にはしない。
- **メタデータと本体を別サービスに分離する**: 将来的には有効だが、P1 のサービス境界と運用コストを増やすため MVP では採用しない。

## 5. ファイルストレージ抽象化

### Decision

`internal/usecase`（または domain の port 集約方針に合わせた内側の package）に、ストリームを扱える `FileStorage` ポートを定義する。MVP の具象は `internal/infra/storage` のローカルファイルシステム実装とし、DB には論理メタデータと opaque な storage key のみを保存する。

推奨する責務は次のとおり。

- `Put(ctx, key, reader, size)`: multipart の入力をストリームで保存し、途中失敗時に不完全なオブジェクトを残さない
- `Open(ctx, key)`: download handler が `io.ReadSeeker` または `io.ReadCloser` とサイズ/Content-Type を取得できる
- `Delete(ctx, key)`: メタデータ削除と組み合わせて孤児本体を残さない

storage key は元ファイル名や user input をパスとして使わず、UUID 等のサーバー生成値にする。元のファイル名、MIME、サイズ、description、tag IDs、uploadedAt、storage key は PostgreSQL の Files 専用 DB に保存する。upload のユースケースは、ストレージ保存とメタデータ insert の順序、insert 失敗時の補償 delete を明示する。

### Rationale

- 既存リポジトリには `internal/infra/storage`、blob/S3 adapter、ファイル保存 port は存在しない。したがってこれは既存パターンを Files 向けに拡張する新規設計となる。
- usecase port にしておけば、handler は multipart の HTTP 表現だけを扱い、ローカルディスクから S3/MinIO などへの変更で usecase 契約を変えずに済む。
- `io.Reader` / `io.ReadCloser` ベースなら、10MB 制限を超える入力をメモリへ全読み込みせず、HTTP ストリーミングとダウンロードの Content-Length を保てる。
- ローカル実装はワークショップと単体テストの導入が容易で、将来のオブジェクトストレージ実装を interface の契約テストで置き換えられる。
- DB と本体を同一サービス内で管理するため、P1 では外部ストレージの認証・署名 URL・追加サービス運用を避けられる。

### Alternatives

- **`os.WriteFile` / `os.ReadFile` で全量バッファリング**: 実装は簡単だが、入力サイズに比例したメモリ使用と大きなファイルの同時処理リスクがあるため不採用。
- **S3/MinIO を MVP の初期実装にする**: 本番スケールには適するが、追加の資格情報、Compose 構成、統合テスト依存が増える。まず local adapter、必要になった時点で S3-compatible adapter を追加する。
- **ファイル本体を DB に保存**: トランザクション一貫性は得やすいが、DB 容量・バックアップ・I/O の責務が膨らむため不採用。
- **storage port を handler に置く**: HTTP 層に永続化の知識が入り、download/upload の別 transport 実装や usecase 単体テストの再利用性が下がるため不採用。

## 6. health / ready とサーバーライフサイクル

### Decision

Auth と同じく `/healthz` は依存を見ない liveness、`/readyz` は Files DB の `Ping` を 2 秒 timeout 付きで確認する readiness とする。`internal/server.Run(ctx)` で config 読み込み、migration、DB pool、JWT verifier、repositories、storage、Echo を配線し、context cancellation で graceful shutdown する。

### Rationale

- Auth の `HealthHandler` は liveness で常に 200、readiness で DB 到達時 200 / 失敗時 503 を返し、DB 依存を `Pinger` interface に抽象化している [`services/auth/internal/handler/health.go:1-60`]。
- health/ready の handler テストは fake pinger で成功・失敗を分離している [`services/auth/internal/handler/health_test.go:1-72`]。
- Auth server は起動時 self-migration、`pgxpool`、JWT、Echo の DI と、10 秒の graceful shutdown を実装している [`services/auth/internal/server/server.go:36-86`]。
- Constitution は運用 endpoint を OpenAPI 契約外かつ `/api/v1` 外に置くことを明記している [` .specify/memory/constitution.md:43-53`]。

### Alternatives

- **`/readyz` を常に 200 にする**: DB 障害時にロードバランサーが不健全なインスタンスへ送るため不採用。
- **liveness でも DB を確認する**: DB 障害でプロセス再起動を誘発し、復旧可能な依存障害を悪化させるため不採用。
- **Compose の migration job にのみ依存する**: Auth は self-migration を採用しており、サービス単体起動時の再現性が高いため同じ方式を推奨。

## 7. 依存関係とモジュール境界

### Decision

Files は独立した `go.mod` を持ち、Auth と同系統の依存を最小限追加する。少なくとも Echo、oapi-codegen runtime/echo-middleware、kin-openapi、pgx/v5、sqlc 生成コード、golang-migrate/v4、golang-jwt/v5、google/uuid、uber/mock、testcontainers-go を検討する。JWT 検証のみ `packages/authjwt` を相対 `replace` で参照する。

### Rationale

- Auth の直接依存は Echo、kin-openapi、oapi-codegen、pgx、migrate、JWT、uuid、testcontainers、uber/mock、x/crypto と、共有 `authjwt` で構成される [`services/auth/go.mod:1-28`]。
- `packages/authjwt` 自体の実行時依存は JWT、uuid、Echo に限定される [`packages/authjwt/go.mod:1-12`]。
- Auth の Dockerfile はリポジトリ直下を build context とし、共有モジュールをコピーしたうえで `GOWORK=off` と相対 replace で単体ビルドしている [`services/auth/Dockerfile:1-23`]。Files も同じ制約を前提に Dockerfile/Compose を設計する。
- 現在の `go.work` は `packages/authjwt` と `services/auth` のみを use している [`go.work:1-6`]。Files を実装対象にする場合は workspace への追加も必要になる。
- Compose では Auth は `8081`、Files は未実装のため未登録であり、Files サービス追加時に `8082`、専用 DB、公開鍵 read-only mount、healthcheck を追加する必要がある [`compose.yaml:1-38`]。

### Alternatives

- **Auth の go.mod を Files から共有する**: Go module とサービスの独立性を壊すため不採用。
- **JWT verifier を Files にコピーする**: 共有契約の重複と実装差分を生むため、既存 `packages/authjwt` を利用する。
- **Files から Auth の `internal` を import する**: 明示的に禁止されている cross-service import であり不採用。

## 8. テスト構成

### Decision

Auth と同じ三層テストを Files MVP に適用する。

1. **Unit**: usecase、storage adapter、handler、JWT middleware の標準 `testing` + `go.uber.org/mock`。テーブル駆動、`t.Parallel()`、testify 不使用。
2. **Integration**: `//go:build integration` で PostgreSQL testcontainer を起動し、migration、metadata CRUD、一覧検索、tag filter、ページネーションを検証。local storage は一時ディレクトリで実体を検証する。
3. **API**: OpenAPI 駆動 Schemathesis と Hurl で upload → list/search/filter → detail → download、未認証 401、10MB 超過、404、health/ready を検証。

### Rationale

- Auth usecase は gomock の repository/hasher/token を注入するテーブル駆動テストで成功・重複・認証失敗を検証している [`services/auth/internal/usecase/auth_usecase_test.go:14-263`]。
- Auth handler は Echo の httptest context と gomock usecase で bind、レスポンス、エラーコードを検証している [`services/auth/internal/handler/auth_handler_test.go:1-220`]。
- repository 統合テストは `//go:build integration`、PostgreSQL 17 testcontainer、実 SQL/migration、NotFound/unique violation を使う [`services/auth/internal/infra/repo/user_repository_test.go:1-146`]。
- Makefile は unit を `go test -short -race ./...`、integration を `go test -race -tags=integration ./...` と分離している [`services/auth/Makefile:39-47`, `services/files/Makefile:34-42`]。
- API テストには Auth の Hurl シナリオがあり、登録・ログイン・JWT 必須の `/auth/me` と異常系を検証する [`api-tests/hurl/scenarios/auth/01_register_and_login.hurl:1-49`, `api-tests/hurl/scenarios/auth/02_login_failures.hurl:1-61`]。
- Constitution のテスト原則は usecase/handler のカバレッジ目標、実 PostgreSQL、Schemathesis + Hurl を要求する [` .specify/memory/constitution.md:85-99`]。

### Alternatives

- **unit test のみ**: SQL の pagination/filter、migration、ファイル実体と metadata の不整合を検出できないため不十分。
- **testcontainers を通常の unit test で常時実行する**: Docker 依存と実行時間が増えるため integration build tag で分離する。
- **Hurl シナリオ間で状態を共有する**: 実行順・並列性に依存するため、Auth の異常系と同じく各シナリオを自己完結させる。

## 9. MVP 実装時の注意点

### Decision

P1 では仕様にある次の制約を OpenAPI、handler/usecase、DB query、テストの全層で同じ意味にする。

- upload は multipart、file 必須、description 任意（最大 500 文字）
- 最大サイズ 10MB。HTTP body の上限と保存前の実サイズ確認を併用する
- 一覧は page 1 / pageSize 20 を既定値とし、総件数を返す
- keyword はファイル名部分一致、tag ID は複数指定可能
- download は保存済み MIME と元ファイル名を response header に反映する
- metadata がない場合は 404、JWT なし/不正は 401、予期しないエラーは共通 500
- upload の片側失敗（storage 成功後 DB 失敗、DB 成功後 storage 失敗）を補償または明示的な cleanup で処理する

### Rationale

- Files の spec は P1 の独立テストとして upload、list/search/filter、detail/download を定義し、10MB、20 件既定 page、共通 JWT、health check を要求している [`specs/002-document-management/spec.md:1-85`, `specs/002-document-management/spec.md:89-155`]。
- Auth は OpenAPI 検証を先に適用し、handler ではドメインエラーと共通 `code`/`message` を返すため、Files でも制約の責務を明確に分けられる [`services/auth/internal/server/server.go:113-151`, `services/auth/internal/handler/auth_handler.go:91-123`]。

### Alternatives

- **10MB 制限を handler の `Content-Length` だけで判定する**: chunked transfer などで不十分なため、保存ストリーム側でも上限を確認する。
- **tag filter を Go で全件取得して絞り込む**: 総件数・ページネーション・500ms 目標に不利なため、SQL で filter/count/page を行う。
- **download URL を永続保存する**: host/path 変更や認証制御に弱いため、MVP では API route から決定的に生成するかレスポンス時に組み立てる。

## 未解決・plan で確定すべき事項

- `schema/files/openapi.yaml` に multipart の schema 表現、tag filter の query 形式、download response header、具体的な error code を確定する。
- Files の DB スキーマ（file metadata、tags、file_tags）が未作成のため、`data-model.md` と migration/query を設計する。
- ローカルストレージの root directory、容量上限、cleanup 方針、Compose volume を config として確定する。
- `compose.yaml` の Files service、Files DB、`go.work` の Files module 追加を plan/tasks に含める。
- ファイル削除・更新は spec 上 P2 だが、upload 失敗時 cleanup に必要な storage delete を内部補償操作としてどう扱うかを決める。
