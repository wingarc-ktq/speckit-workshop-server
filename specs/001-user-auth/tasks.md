---
description: 'Task list for ユーザー認証 (001-user-auth)'
---

# Tasks: ユーザー認証

**Input**: Design documents from `/specs/001-user-auth/`
**Prerequisites**: plan.md（必須）, spec.md（ユーザーストーリー）, research.md, data-model.md, contracts/README.md

**Tests**: Constitution IV「テスト駆動（NON-NEGOTIABLE）」に従い、テストタスクを含めます（usecase 層のテーブル駆動単体テスト、handler 層のテーブル駆動単体テスト、infra 層の統合テスト、API テスト）。

**Organization**: タスクはユーザーストーリー単位でグループ化し、各ストーリーを独立して実装・テストできるようにしています。

> **改定メモ（2026-06-28 / 実装後リファクタ）**
> 以下のタスク（[X] 完了済み）は **当初の実装時点の記録** です。完了後にクリーンアーキの抽象度向上を目的としたリファクタを行い、現在のコードは次の点が異なります（Constitution v1.1.0 に整合）:
> - **JWT は HS256 共有シークレット → RS256（非対称鍵）**。Auth が秘密鍵で署名、検証側は公開鍵。鍵は PEM パスを環境変数で注入（`AUTH_JWT_PRIVATE_KEY_PATH` / `AUTH_JWT_PUBLIC_KEY_PATH`）、開発鍵は `make keys` で生成（git 管理しない）。
> - **暗号処理をポート化**: パスワードハッシュ（`PasswordHasher`）・トークン発行/検証（`TokenIssuer` / `TokenVerifier`）を `domain` のインターフェースとして定義し、`internal/infra/{password,token}` に実装。usecase は具体技術（bcrypt/JWT）に依存しない。`issueToken` は usecase から token アダプタへ移動。
> - **リクエスト検証を OpenAPI 駆動に一本化**: ハンドラの手書きバリデーションを撤去し、`OapiRequestValidator`（kin-openapi）ミドルウェアで `schema/auth/openapi.yaml` の制約を検証。
> - `NewAuthHandler` は `jwtSecret` 引数を持たない（検証は注入された `TokenVerifier` 経由）。
> 個々のタスク記述は履歴として原文のまま残しています。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 並列実行可能（別ファイル・未完了タスクへの依存なし）
- **[Story]**: 対応するユーザーストーリー（US1, US2, US3）
- 各タスクに具体的なファイルパスを記載

## Path Conventions

- マイクロサービス構成: `services/auth/` 配下にクリーンアーキテクチャ（handler / usecase / domain / infra）を配置
- OpenAPI 仕様（Single Source of Truth）: `schema/auth/openapi.yaml`
- API テスト: リポジトリルートの `api-tests/`

---

## Phase 1: Setup（共有インフラ）

**Purpose**: プロジェクト初期化とビルド/コード生成の足回り

- [X] T001 `services/auth/go.mod` に依存（echo/v4, oapi-codegen/runtime, kin-openapi, golang-jwt/v5, pgx/v5, golang.org/x/crypto, google/uuid, uber/mock）を定義し、`go mod tidy` で `services/auth/go.sum` を生成する
- [X] T002 [P] `services/auth/Makefile` に `tools` / `gen` / `build` / `run` / `test-unit` / `test-integration` / `migrate-up` / `migrate-down` / `lint` / `fmt` ターゲットを定義する
- [X] T003 [P] `services/auth/oapi-codegen.yaml` を作成し、echo-server / models / strict-server 生成と出力先 `api/gen/server.gen.go` を設定する
- [X] T004 [P] `services/auth/sqlc.yaml` を作成し、`migrations/` を schema、`internal/infra/repo/queries/` を queries、出力先 `internal/infra/repo/db/` を pgx/v5 で設定する
- [X] T005 [P] `services/auth/.env.sample` に `AUTH_SERVICE_PORT` / `AUTH_DATABASE_URL` / `AUTH_JWT_SECRET` / `AUTH_JWT_TTL_SECONDS` を記載する
- [X] T006 [P] `services/auth/Dockerfile` にマルチステージビルド（builder → distroless/alpine ランタイム）を定義する

---

## Phase 2: Foundational（ブロッキング前提）

**Purpose**: 全ユーザーストーリーが依存する中核インフラ。完了するまでストーリー実装は開始できない

**⚠️ CRITICAL**: このフェーズが完了するまで、いかなるユーザーストーリー作業も開始できない

- [X] T007 `services/auth/internal/config/config.go` に環境変数読み込み（`Config` 構造体 + `Load()`）を実装する
- [X] T008 [P] `services/auth/migrations/000001_create_users_table.up.sql` と `services/auth/migrations/000001_create_users_table.down.sql` に users テーブル DDL（id/email/password_hash/name/created_at/updated_at、UNIQUE 制約、`idx_users_email`）を定義する
- [X] T009 [P] `services/auth/internal/infra/repo/queries/users.sql` に sqlc クエリ（`CreateUser` / `GetUserByID` / `GetUserByEmail`（`LOWER(email)=LOWER($1)`））を定義する
- [X] T010 `services/auth/internal/domain/user.go` に `User` モデル、`UserRepository` インターフェース（Create/FindByID/FindByEmail）、ドメインエラー（`ErrUserNotFound` / `ErrEmailAlreadyTaken` / `ErrInvalidCredential`）と `go:generate mockgen` ディレクティブを定義する
- [X] T011 `cd services/auth && make gen` を実行し、`api/gen/server.gen.go`（oapi-codegen）、`internal/infra/repo/db/*.go`（sqlc）、`internal/domain/mock/user_mock.go`（gomock）を生成する（T003, T004, T009, T010 に依存）
- [X] T012 `services/auth/internal/infra/repo/user_repository.go` に `UserRepository` 実装を作成する。`db.New(pool)` をラップし、`pgtype.UUID`/`pgtype.Timestamptz` ↔ `uuid.UUID`/`time.Time` を変換、`pgx.ErrNoRows` を `domain.ErrUserNotFound` にマッピング、UNIQUE 違反を `domain.ErrEmailAlreadyTaken` にマッピングする（全ストーリーが共有、T010, T011 に依存）
- [X] T013 `services/auth/internal/handler/auth_handler.go` に `AuthHandler` 構造体・コンストラクタ `NewAuthHandler(uc, jwtSecret)`・ドメインエラー → HTTP（`gen.ErrorResponse` の message+code）マッピングヘルパーを定義し、`gen.ServerInterface` の 3 メソッド（RegisterUser/LoginUser/GetCurrentUser）を未実装スタブとして用意してコンパイルを通す（T011 に依存）
- [X] T014 `services/auth/cmd/server/main.go` に DI 配線（config 読込 → `pgxpool.New` → `repo.NewUserRepository` → `usecase.NewAuthUsecase` → `handler.NewAuthHandler` → echo + `gen.RegisterHandlersWithBaseURL(e, h, "/api/v1")` → `e.Start`）を実装する（T012, T013 に依存）

**Checkpoint**: 基盤が整い、ユーザーストーリーの実装を開始できる

---

## Phase 3: User Story 1 - ユーザー登録 (Priority: P1) 🎯 MVP

**Goal**: 未登録ユーザーが email/password/name でアカウントを作成し、ユーザー情報が返る

**Independent Test**: `POST /api/v1/auth/register` に有効な情報を送り 201 とユーザー情報が返ること、重複メールで 409、8 文字未満パスワードで 400 が返ることを確認

### Tests for User Story 1 ⚠️（実装前に書き、失敗することを確認）

- [X] T015 [P] [US1] `services/auth/internal/usecase/auth_usecase_test.go` に `Register` のテーブル駆動テスト（正常登録 / メール重複（`ErrEmailAlreadyTaken`）/ email 小文字正規化 / bcrypt ハッシュ化）を gomock で実装する

### Implementation for User Story 1

- [X] T016 [US1] `services/auth/internal/usecase/auth_usecase.go` に `RegisterInput` と `Register` を実装する（`FindByEmail` 事前チェック → `strings.ToLower()` でメール正規化 → bcrypt(DefaultCost) ハッシュ化 → `uuid.New()` → `Create`）（T010 に依存）
- [X] T017 [US1] `services/auth/internal/handler/auth_handler.go` の `RegisterUser` を実装する（`gen.RegisterRequest` を bind、password 長さ 8〜128 と name 1〜100 を検証して 400 `VALIDATION_ERROR`、`ErrEmailAlreadyTaken` を 409 `EMAIL_ALREADY_TAKEN`、成功時 201 `gen.UserResponse`）（T013, T016 に依存）
- [X] T018 [US1] `api-tests/hurl/scenarios/auth/01_register_and_login.hurl` の登録パート（201 で id/email/name/createdAt を検証、重複時 409）を記述する

**Checkpoint**: ユーザー登録が単体で動作・テスト可能

---

## Phase 4: User Story 2 - ログインと JWT 発行 (Priority: P1) 🎯 MVP

**Goal**: 登録済みユーザーが email/password でログインし JWT（accessToken/tokenType/expiresIn）とユーザー情報を取得する

**Independent Test**: 登録済みユーザーで `POST /api/v1/auth/login` を実行し 200 と JWT が返ること、誤パスワード・存在しないメールで原因を区別しない 401 `AUTH_FAILED` が返ることを確認

### Tests for User Story 2 ⚠️（実装前に書き、失敗することを確認）

- [X] T019 [P] [US2] `services/auth/internal/usecase/auth_usecase_test.go` に `Login` のテーブル駆動テスト（正常ログインで JWT 発行・クレーム sub/exp/iat 検証 / 誤パスワード → `ErrInvalidCredential` / 未登録メール → `ErrInvalidCredential`）を実装する

### Implementation for User Story 2

- [X] T020 [US2] `services/auth/internal/usecase/auth_usecase.go` に `LoginOutput`・`Login`・`issueToken`（HS256、`sub`=user_id, `iat`, `exp`）を実装する。`FindByEmail` 失敗と bcrypt 不一致をともに `ErrInvalidCredential` に統一する（T010 に依存）
- [X] T021 [US2] `services/auth/internal/handler/auth_handler.go` の `LoginUser` を実装する（`gen.LoginRequest` を bind、`ErrInvalidCredential` を 401 `AUTH_FAILED`、成功時 200 `gen.LoginResponse`（tokenType="Bearer"））（T013, T020 に依存）
- [X] T022 [US2] `api-tests/hurl/scenarios/auth/02_login_failures.hurl` にログイン失敗シナリオ（誤パスワード 401 / 未登録メール 401、レスポンスが原因を区別しないこと）を記述する

**Checkpoint**: ユーザー登録 + ログインが独立して動作

---

## Phase 5: User Story 3 - JWT による認証中ユーザー取得 (Priority: P1) 🎯 MVP

**Goal**: クライアントが Bearer JWT を付けて `GET /api/v1/auth/me` で自身の情報を取得する

**Independent Test**: 有効な JWT で 200 とユーザー情報、JWT 未指定で 401、期限切れ JWT で 401 が返ることを確認

### Tests for User Story 3 ⚠️（実装前に書き、失敗することを確認）

- [X] T023 [P] [US3] `services/auth/internal/usecase/auth_usecase_test.go` に `Me` のテーブル駆動テスト（有効 user_id でユーザー取得 / 未存在 user_id で `ErrUserNotFound`）を実装する

### Implementation for User Story 3

- [X] T024 [US3] `services/auth/internal/handler/middleware.go` に JWT 検証 echo ミドルウェアを実装する（`Authorization: Bearer` から抽出、HS256 で検証、`sub` クレーム（user_id）を `echo.Context.Set()` で格納、未指定/無効/期限切れは 401 `UNAUTHORIZED`）（T011 に依存）
- [X] T025 [US3] `services/auth/internal/usecase/auth_usecase.go` に `Me`（`FindByID` 呼び出し）を実装する（T010 に依存）
- [X] T026 [US3] `services/auth/internal/handler/auth_handler.go` の `GetCurrentUser` を実装する（`echo.Context.Get()` で user_id を取得 → `uuid.Parse` → `Me` 呼び出し → 200 `gen.UserResponse`）（T013, T025 に依存）
- [X] T027 [US3] `services/auth/cmd/server/main.go` を更新し、`gen.RegisterHandlersWithOptions` の `OperationMiddlewares["getCurrentUser"]` に JWT 検証ミドルウェアを適用する（T014, T024 に依存）
- [X] T028 [US3] `api-tests/hurl/scenarios/auth/01_register_and_login.hurl` に `/auth/me` の正常系（取得した JWT で 200）と異常系（JWT 未指定で 401）を追記する

**Checkpoint**: 全 3 ユーザーストーリーが独立して動作

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 複数ストーリーに横断する品質向上・検証

- [X] T029 [P] `services/auth/internal/infra/repo/user_repository_test.go` に testcontainers-go を使った統合テスト（実 PostgreSQL に対する Create/FindByEmail/FindByID、UNIQUE 違反 → `ErrEmailAlreadyTaken`）を実装する
- [X] T030 [P] `api-tests/schemathesis/` に `schema/auth/openapi.yaml` に対する OpenAPI 準拠自動テスト設定を整備し、`make schemathesis` で実行できるようにする（SC-003）
- [X] T031 [P] `specs/001-user-auth/quickstart.md` の手順（make gen → migrate-up → build → run → curl 3 エンドポイント）を実機で検証する
- [X] T032 `cd services/auth && make lint && make fmt`（gofmt / goimports / go vet）を実行し、Constitution I 準拠を確認する
- [X] T033 [P] `services/auth/` 各パッケージの公開シンボルに GoDoc コメントを整備する（Constitution I）
- [X] T034 [P] `services/auth/internal/usecase/auth_usecase.go` の `AuthUsecase` を入力ポート interface として公開し（実装は非公開 `authUsecase`、`NewAuthUsecase` は interface を返す。`//go:generate mockgen` で `usecase/mock/auth_usecase_mock.go` を生成）、handler をこの interface に依存させたうえで、`services/auth/internal/handler/auth_handler_test.go` にモックを使ったテーブル駆動単体テスト（3 エンドポイントの成功時レスポンス整形／ドメインエラー→HTTP マッピング 400/401/409/500／不正 JSON の 400／`GetCurrentUser` の userID 未格納時に usecase 未呼び出しで 401）を実装する

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 依存なし。即時開始可能
- **Foundational (Phase 2)**: Setup 完了に依存。全ユーザーストーリーをブロック
- **User Stories (Phase 3-5)**: すべて Foundational 完了に依存
  - 基盤完了後は並列実行可能（人員が許せば）、または優先度順（すべて P1）に逐次実行
- **Polish (Phase 6)**: 対象ユーザーストーリー完了に依存

### User Story Dependencies

- **US1（登録, P1）**: Foundational 完了後に開始可能。他ストーリーに依存しない
- **US2（ログイン, P1）**: Foundational 完了後に開始可能。登録済みユーザーを前提とするが、テストは独立して実行可能（テストデータ準備で対応）
- **US3（me, P1）**: Foundational 完了後に開始可能。JWT が必要だがミドルウェア + Me ユースケースで独立してテスト可能

### Within Each User Story

- テストを先に書き、失敗を確認してから実装（TDD / Constitution IV）
- usecase → handler の順（handler は usecase に依存）
- JWT ミドルウェア（T024）→ main.go への適用（T027）

### コンパイル上の注意

`gen.ServerInterface` は 3 メソッドすべての実装を要求するため、T013 でスタブを用意してから各ストーリーで中身を埋める。これによりストーリーを段階的に実装してもビルドが通る。

### Parallel Opportunities

- Setup: T002, T003, T004, T005, T006 は並列実行可能
- Foundational: T008, T009 は並列実行可能（T010 も別ファイルだが T011 がこれらに依存）
- 各ストーリーのテストタスク（T015 / T019 / T023）は、同一ファイル `auth_usecase_test.go` を編集するためストーリー間では逐次。各ストーリー内では実装（別ファイル）と並行して着手可能
- Polish: T029, T030, T031, T033 は並列実行可能

---

## Parallel Example: Setup フェーズ

```bash
# Setup の構成ファイルをまとめて作成（別ファイル・依存なし）:
Task: "services/auth/Makefile を作成"
Task: "services/auth/oapi-codegen.yaml を作成"
Task: "services/auth/sqlc.yaml を作成"
Task: "services/auth/.env.sample を作成"
Task: "services/auth/Dockerfile を作成"
```

---

## Implementation Strategy

### MVP First（US1 のみ）

1. Phase 1: Setup を完了
2. Phase 2: Foundational を完了（CRITICAL — 全ストーリーをブロック）
3. Phase 3: US1（ユーザー登録）を完了
4. **STOP and VALIDATE**: 登録エンドポイントを独立してテスト
5. 準備できればデモ

### Incremental Delivery

1. Setup + Foundational → 基盤完成
2. US1（登録）追加 → 独立テスト → デモ（MVP!）
3. US2（ログイン）追加 → 独立テスト → デモ
4. US3（me）追加 → 独立テスト → デモ
5. 各ストーリーは前のストーリーを壊さず価値を追加

### Parallel Team Strategy

基盤（Phase 1-2）完了後:

- Developer A: US1（登録）
- Developer B: US2（ログイン）
- Developer C: US3（me）

ただし `auth_usecase.go` / `auth_handler.go` を複数ストーリーで共有するため、メソッド単位で衝突を避けて作業する。

---

## Notes

- [P] = 別ファイル・依存なしで並列実行可能
- [Story] ラベルでタスクとユーザーストーリーの対応を追跡
- 各ユーザーストーリーは独立して完成・テスト可能であること
- 実装前にテストが失敗することを確認（Constitution IV）
- 生成コード（`api/gen/`, `internal/infra/repo/db/`, `internal/domain/mock/`）は手動編集禁止（Constitution II / V）
- タスクごと、または論理的なまとまりごとにコミット
- 各チェックポイントで停止してストーリーの独立性を検証
