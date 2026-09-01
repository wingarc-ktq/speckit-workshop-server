/speckit.implement/speckit.implement---
description: 'Task list for 文書管理 (002-document-management)'
---

# Tasks: 文書管理

**Input**: Design documents from `/specs/002-document-management/`
**Prerequisites**: plan.md（必須）, spec.md（ユーザーストーリー）, research.md, data-model.md

**Tests**: 実装前に各ストーリーの独立テストを作成し、失敗を確認してから実装する。MVP では US1〜US3 を最優先とし、P2 のタグ管理/編集/削除は後続の実装対象とする。

**Organization**: タスクはユーザーストーリー単位でグループ化し、各ストーリーを独立して実装・テストできるようにする。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 並列実行可能（別ファイル・未完了タスクへの依存なし）
- **[Story]**: 対応するユーザーストーリー（US1, US2, US3, US4, US5, US6）
- 各タスクに具体的なファイルパスを記載

---

## Phase 1: Setup（共有インフラ）

**Purpose**: Files サービスの初期化とコード生成基盤のセットアップ

- [x] T001 `services/files/go.mod` に echo/v4, oapi-codegen/runtime, kin-openapi, pgx/v5, sqlc, google/uuid, golang-jwt/v5, go.uber.org/mock, testcontainers-go を定義し、go mod tidy を実行する
- [x] T002 [P] `services/files/Makefile` に gen / gen-oapi / gen-sqlc / test-unit / test-integration / build / run / fmt / lint / migrate-up / migrate-down のターゲットを定義する
- [x] T003 [P] `services/files/oapi-codegen.yaml` を作成し、openapi.yaml から echo server と型定義を生成する設定を定義する
- [x] T004 [P] `services/files/sqlc.yaml` を作成し、migrations と queries を繋いで `internal/infra/repo/db/` に SQL コードを出力する設定を定義する
- [x] T005 [P] `services/files/Dockerfile` にマルチステージビルドを定義し、コンテナ実行環境を用意する
- [x] T006 [P] `services/files/.env.sample` に `FILES_SERVICE_PORT`, `FILES_DATABASE_URL`, `FILES_STORAGE_PATH`, `FILES_JWT_PUBLIC_KEY_PATH` を定義し、ローカル起動用の設定例を作成する

---

## Phase 2: Foundational（ブロッキング前提）

**Purpose**: 全ユーザーストーリーが依存する中核インフラ。完了するまでストーリー実装は開始できない

**⚠️ CRITICAL**: このフェーズが完了するまで、いかなるユーザーストーリー作業も開始できない

- [x] T007 `services/files/internal/config/config.go` に環境変数読み込みと `Config` 構造体を実装する
- [x] T008 [P] `services/files/migrations/000001_create_files_table.up.sql` と `services/files/migrations/000001_create_files_table.down.sql` に files テーブル DDL を定義する
- [x] T009 [P] `services/files/internal/infra/repo/queries/files.sql` に `CreateFile`, `ListFiles`, `GetFileByID`, `UpdateFileMetadata`, `DeleteFile`, `DeleteFilesByIDs`, `CountFiles` を定義する
- [x] T010 `services/files/internal/domain/file.go` に `File` エンティティ、`FileRepository` インターフェース、ドメインエラー（`ErrFileNotFound`, `ErrFileTooLarge`, `ErrInvalidPagination`, `ErrDuplicateFileName` など）を定義する
- [x] T011 `services/files/internal/domain/storage.go` に `FileStorage` インターフェースと `FileContent`/`StoredFile` の交差型を定義し、ローカルストレージ抽象化の契約を確立する
- [x] T012 `cd services/files && make gen-oapi && make gen-sqlc` を実行し、`api/gen/server.gen.go` と `internal/infra/repo/db/*.sql.go` を生成する
- [x] T013 `services/files/internal/infra/repo/file_repository.go` に `FileRepository` 実装を作成し、`pgx.ErrNoRows` と `UNIQUE` 違反をドメインエラーへマッピングする
- [x] T014 `services/files/internal/infra/storage/local_storage.go` にローカルファイルシステム実装を作成し、アップロード・取得・削除の処理を抽象化された `FileStorage` へ実装する
- [x] T015 `services/files/internal/server/server.go` と `services/files/cmd/server/main.go` に DI 配線を実装し、JWT 検証ミドルウェア、ハンドラ、ルーティング、ヘルスチェックを組み立てる
- [x] T016 `services/files/internal/handler/file_handler.go` に `ServerInterface` のスタブを実装し、各メソッドがコンパイル可能になるように準備する

**Checkpoint**: 基盤が整い、ユーザーストーリーの実装を開始できる

---

## Phase 3: User Story 1 - ファイルアップロード (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイルと説明文を送信し、ファイル情報を受け取れるようにする

**Independent Test**: `POST /files` に有効な JWT と 5MB 未満のファイルを送信し、201 とファイル情報が返ることを確認。サイズ超過と未認証をそれぞれ 413/401 で確認。

### Tests for User Story 1 ⚠️

- [x] T017 [P] [US1] `services/files/internal/usecase/file_usecase_test.go` にアップロード正常系テスト（ファイル保存 + DB 登録 + downloadUrl）を追加する
- [x] T018 [P] [US1] `services/files/internal/usecase/file_usecase_test.go` にサイズ超過・未認証・ストレージ失敗の異常系テストを追加する
- [x] T019 [P] [US1] `services/files/internal/handler/file_handler_test.go` に `CreateFile` の HTTP レスポンス整形テストを追加する

### Implementation for User Story 1

- [x] T020 [US1] `services/files/internal/usecase/file_usecase.go` に `UploadFile` 入出力とビジネスロジックを実装する（JWT ユーザー ID を保持し、保存内容とメタデータを DB に登録する）
- [x] T021 [US1] `services/files/internal/handler/file_handler.go` の `CreateFile` を実装し、multipart/form-data から `file` と `description` を受け取り、正規化とエラーマッピングを行う
- [x] T022 [US1] `services/files/internal/infra/repo/file_repository.go` の `CreateFile` 実装と `internal/infra/repo/db` の SQL 連携を整備する
- [x] T023 [US1] `services/files/internal/infra/storage/local_storage.go` のアップロード処理を `FileStorage` に合わせて実装し、保存先ディレクトリとファイル名の一意性を担保する
- [x] T024 [US1] `schema/files/openapi.yaml` との整合性を確認し、`POST /files` の request/response とエラーコードを実装に合わせて調整する

**Checkpoint**: ユーザー登録が単体で動作・テスト可能

---

## Phase 4: User Story 2 - ファイル一覧取得と検索 (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイル一覧をページネーション付きで取得し、キーワード検索・タグフィルタを利用できるようにする

**Independent Test**: `GET /files` で一覧・総件数・page・limit を確認し、キーワード検索とタグフィルタの両方が機能することを確認。

### Tests for User Story 2 ⚠️

- [x] T025 [P] [US2] `services/files/internal/usecase/file_usecase_test.go` に `ListFiles` の正常系テストを追加する
- [x] T026 [P] [US2] `services/files/internal/usecase/file_usecase_test.go` に検索・タグ絞り込み・ページネーション異常系テストを追加する
- [x] T027 [P] [US2] `services/files/internal/handler/file_handler_test.go` に一覧 API の HTTP 応答整形テストを追加する

### Implementation for User Story 2

- [x] T028 [US2] `services/files/internal/usecase/file_usecase.go` に `ListFiles` を実装し、検索条件・ページネーション・総件数を返せるようにする
- [x] T029 [US2] `services/files/internal/handler/file_handler.go` の `ListFiles` を実装し、クエリパラメータに応じたバリデーションとレスポンス整形を行う
- [x] T030 [US2] `services/files/internal/infra/repo/file_repository.go` と `services/files/internal/infra/repo/queries/files.sql` に一覧取得・件数取得・検索条件クエリを追加する
- [x] T031 [US2] `api-tests/hurl/scenarios/files/01_list_and_search.hurl` に一覧・検索・ページネーションの実行ケースを記述する

**Checkpoint**: ファイル一覧と検索が単体で動作

---

## Phase 5: User Story 3 - ファイル詳細取得とダウンロード (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイルの詳細情報を取得し、元のファイルをダウンロードできるようにする

**Independent Test**: `GET /files/{fileId}` と `GET /files/{fileId}/download` を実行し、200 と元データが返ること、存在しない ID で 404 を確認。

### Tests for User Story 3 ⚠️

- [x] T032 [P] [US3] `services/files/internal/usecase/file_usecase_test.go` に詳細取得とダウンロード正常系テストを追加する
- [x] T033 [P] [US3] `services/files/internal/usecase/file_usecase_test.go` に未存在 ID での `ErrFileNotFound` テストを追加する
- [x] T034 [P] [US3] `services/files/internal/handler/file_handler_test.go` に詳細・ダウンロード API の HTTP 整形テストを追加する

### Implementation for User Story 3

- [x] T035 [US3] `services/files/internal/usecase/file_usecase.go` に `GetFile` と `DownloadFile` を実装し、ストレージから実データとメタデータを取得する
- [x] T036 [US3] `services/files/internal/handler/file_handler.go` の `GetFile` と `DownloadFile` を実装し、ID 検証・レスポンス変換・404 マッピングを行う
- [x] T037 [US3] `services/files/internal/infra/repo/file_repository.go` に詳細取得用クエリとバイナリ読み出し対応を追加する
- [x] T038 [US3] `api-tests/hurl/scenarios/files/02_detail_and_download.hurl` に詳細取得とダウンロードの成功/失敗シナリオを記述する

**Checkpoint**: MVP の 3 つのファイル API が独立して動作

---

## Phase 6: User Story 4 - タグ管理 (Priority: P2)

**Goal**: 認証済みユーザーがタグを作成・一覧取得・更新・削除できるようにする

**Independent Test**: タグの作成・一覧取得・更新・削除と重複エラーが各々独立して動作することを確認。

### Tests for User Story 4 ⚠️

- [x] T039 [P] [US4] `services/files/internal/usecase/tag_usecase_test.go` にタグ管理の正常系・異常系テストを追加する
- [x] T040 [P] [US4] `services/files/internal/handler/tag_handler_test.go` にタグ API の HTTP テストを追加する

### Implementation for User Story 4

- [x] T041 [US4] `services/files/internal/domain/tag.go` に `Tag` エンティティと `TagRepository` インターフェースを定義する
- [x] T042 [US4] `services/files/internal/infra/repo/tag_repository.go` と `services/files/internal/infra/repo/queries/tags.sql` に CRUD 処理を追加する
- [x] T043 [US4] `services/files/internal/usecase/tag_usecase.go` にタグ作成/取得/更新/削除のユースケースを実装する
- [x] T044 [US4] `services/files/internal/handler/tag_handler.go` にタグ API のハンドラ実装を追加し、重複エラー・存在しない ID を HTTP に変換する

**Checkpoint**: タグ管理が独立して動作

---

## Phase 7: User Story 5 - ファイルメタデータ編集 (Priority: P2)

**Goal**: ファイル名、説明、タグ ID 一覧を更新できるようにする

**Independent Test**: 既存ファイルのメタデータ変更が取得時に反映され、存在しない ID で 404 を返すことを確認。

### Tests for User Story 5 ⚠️

- [x] T045 [P] [US5] `services/files/internal/usecase/file_usecase_test.go` にメタデータ更新の成功/失敗テストを追加する
- [x] T046 [P] [US5] `services/files/internal/handler/file_handler_test.go` に更新 API の HTTP テストを追加する

### Implementation for User Story 5

- [x] T047 [US5] `services/files/internal/usecase/file_usecase.go` に `UpdateFileMetadata` を実装し、ファイル名・説明・タグ ID 一覧の変更をまとめて扱う
- [x] T048 [US5] `services/files/internal/handler/file_handler.go` の `UpdateFile` を実装し、PATCH/PUT 形式のリクエストを受けて変換する
- [x] T049 [US5] `services/files/internal/infra/repo/file_repository.go` と `services/files/internal/infra/repo/queries/files.sql` にメタデータ更新とタグ関連更新クエリを追加する
- [x] T050 [US5] `api-tests/hurl/scenarios/files/03_update_metadata.hurl` にファイルメタデータ更新のシナリオを記述する

**Checkpoint**: メタデータ修正が独立して動作

---

## Phase 8: User Story 6 - ファイル削除 (Priority: P2)

**Goal**: 個別削除と一括削除をサポートし、関連するタグ関連データとストレージ本体も削除できるようにする

**Independent Test**: 指定したファイルが削除されること、存在しない ID で 404 を返すこと、一括削除が正しく動くことを確認。

### Tests for User Story 6 ⚠️

- [x] T051 [P] [US6] `services/files/internal/usecase/file_usecase_test.go` に個別削除と一括削除の正常系/異常系テストを追加する
- [x] T052 [P] [US6] `services/files/internal/handler/file_handler_test.go` に削除 API の HTTP テストを追加する

### Implementation for User Story 6

- [x] T053 [US6] `services/files/internal/usecase/file_usecase.go` に `DeleteFile` と `DeleteFiles` を実装する
- [x] T054 [US6] `services/files/internal/handler/file_handler.go` の `DeleteFile` と `DeleteFiles` を実装し、ID 一覧の検証とレスポンス整形を行う
- [x] T055 [US6] `services/files/internal/infra/repo/file_repository.go` と `services/files/internal/infra/repo/queries/files.sql` に削除クエリと関連レコード削除を追加する
- [x] T056 [US6] `services/files/internal/infra/storage/local_storage.go` にファイル削除処理を追加し、削除時に同名ファイルの衝突を避ける
- [x] T057 [US6] `api-tests/hurl/scenarios/files/04_delete_files.hurl` に個別削除と一括削除のシナリオを記述する

**Checkpoint**: ファイルライフサイクルの削除処理が独立して動作

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: 複数ストーリーにまたがる品質改善と最終検証

- [x] T058 [P] `services/files/internal/handler` / `services/files/internal/usecase` / `services/files/internal/infra` のエラーコードと HTTP マッピングを統一し、`schema/files/openapi.yaml` と照合する
- [x] T059 [P] `services/files/internal/infra/repo/file_repository_test.go` に testcontainers-go を使った統合テストを追加する
- [x] T060 [P] `services/files/internal/infra/repo/tag_repository_test.go` にタグ関連の統合テストを追加する
- [x] T061 [P] `api-tests/schemathesis/` に OpenAPI 準拠の自動テスト設定を追加する
- [x] T062 `services/files/` の全 Go ファイルに `gofmt` / `go vet` を適用し、Constitution I を確認する
- [x] T063 [P] `specs/002-document-management/quickstart.md` にローカル起動手順と API 検証コマンドを整備する
- [x] T064 [P] `services/files/` に `README.md` の実行手順とインフラ構成を追記する
- [x] T065 `cd services/files && make test-unit && make test-integration` を実行し、MVP ストーリーと P2 ストーリーが独立して成功することを確認する

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 依存なし。即時開始可能
- **Foundational (Phase 2)**: Setup 完了に依存。全ユーザーストーリーをブロック
- **User Stories (Phase 3-8)**: Foundational 完了後に開始可能
  - US1〜US3 は MVP として優先実装
  - US4〜US6 は P2 として後続実装
- **Polish (Phase 9)**: 全ストーリー完了後に実行

### User Story Dependencies

- **US1（ファイルアップロード, P1）**: Foundational 完了後に開始可能。他ストーリーに依存しない
- **US2（一覧取得と検索, P1）**: Foundational 完了後に開始可能。US1 を前提にしないが、共通の file repository を利用する
- **US3（ファイル詳細取得とダウンロード, P1）**: Foundational 完了後に開始可能。US1 と US2 のデータ形式を再利用する
- **US4（タグ管理, P2）**: US1〜US3 完了後に着手可能。タグを一覧・検索に紐付けるための基礎機能
- **US5（ファイルメタデータ編集, P2）**: US4 のタグ情報が必要なため、US4 完了後に着手
- **US6（ファイル削除, P2）**: US4 と US5 の後に着手し、関連データの削除整合性を担保する

### Within Each User Story

- テストを先に書き、失敗を確認してから実装する
- usecase → handler → repository の順で実装する
- ストレージ実装や DB 変更は、ユースケースのテストを先に通しながら段階的に追加する
- 1 ストーリーごとに独立して動作確認を行う

### Parallel Opportunities

- Setup: T002〜T006 は並列実行可能
- Foundational: T008, T009, T011, T014 は別ファイルで並列に着手可能
- US1〜US3 のテストタスクは各ストーリー内で並列に動かせる
- US4〜US6 の実装は別ファイルで並列に進められるが、タスク単位で依存順序を守る
- Polish: T058〜T064 は複数の観点を並列に検証可能

---

## Parallel Example: User Story 1

```bash
# User Story 1 のテストをまとめて実行
Task: "services/files/internal/usecase/file_usecase_test.go にアップロード正常系テストを追加"
Task: "services/files/internal/usecase/file_usecase_test.go に異常系テストを追加"
Task: "services/files/internal/handler/file_handler_test.go に HTTP テストを追加"

# User Story 1 に必要な実装を複数並列で進める
Task: "services/files/internal/usecase/file_usecase.go に UploadFile を実装"
Task: "services/files/internal/handler/file_handler.go に CreateFile を実装"
Task: "services/files/internal/infra/storage/local_storage.go に保存処理を実装"
```

---

## Implementation Strategy

### MVP First (US1〜US3 のみ)

1. Phase 1: Setup を完了する
2. Phase 2: Foundational を完了する
3. Phase 3: US1（ファイルアップロード）を完了する
4. Phase 4: US2（一覧取得と検索）を完了する
5. Phase 5: US3（詳細取得とダウンロード）を完了する
6. **STOP and VALIDATE**: MVP の3機能を独立テストし、デモ可能な状態を確認する

### Incremental Delivery

1. Phase 1 + Phase 2 → 基盤完成
2. US1 → 独立テスト → デモ
3. US2 → 独立テスト → デモ
4. US3 → 独立テスト → デモ
5. US4〜US6 → 追加の管理機能を順に公開
6. 各ストーリーは前のストーリーを壊さず価値を追加する

### Parallel Team Strategy

基盤完了後:

- Developer A: US1（ファイルアップロード）
- Developer B: US2（一覧取得と検索）
- Developer C: US3（詳細取得とダウンロード）
- 後続で Developer D: US4〜US6（タグ管理・編集・削除）

ただし `file_usecase.go` と `file_handler.go` は共通ファイルを複数ストーリーで編集するため、メソッド単位で分担して衝突を避ける。

---

## Notes

- [P] = 別ファイル・依存なしで並列実行可能
- [Story] ラベルでタスクとユーザーストーリーの対応を追跡
- 各ユーザーストーリーは独立して完成・テスト可能であること
- 実装前にテストを作成し、失敗を確認してから実装する
- 生成コード（`api/gen/`, `internal/infra/repo/db/`）は手動編集禁止とする
- タスクごと、または論理的なまとまりごとにコミットする
- 各チェックポイントで停止してストーリーの独立性を検証する
