# Tasks: 文書管理（Files サービス / P1 MVP）

**Input**: Design documents from `/specs/002-document-management/`
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/files-api-p1.md](./contracts/files-api-p1.md), [quickstart.md](./quickstart.md)

**Tests**: Constitution IV（テスト駆動、NON-NEGOTIABLE）によりテストタスクは必須。単体テスト（usecase/domain, `go.uber.org/mock`）・統合テスト（repo, `testcontainers-go`）・API テスト（Schemathesis + Hurl）をすべて含める。

**Organization**: タスクはユーザーストーリー（`spec.md` の P1: US1 アップロード / US2 一覧・検索 / US3 詳細・ダウンロード）ごとにグループ化。実装順序は `docs/MY-DAY2-GUIDE.md` 2章「migrations → SQL → domain → usecase(+テスト) → infra(+テスト) → handler(+テスト) → server」に従う。JWT 検証は自前実装せず `packages/authjwt` を再利用する（Constitution VII）。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 並行作業可能（別ファイル・依存タスク完了済み）
- **[Story]**: 対象ユーザーストーリー（US1/US2/US3）。Setup・Foundational・Polish フェーズには付与しない
- ファイルパスは `services/files/` を起点とした相対パスで記載

---

## Phase 1: Setup

**Purpose**: サービスの骨組み。**前回セッションで作成済み**（`go build`/`go vet` 通過確認済み）。このフェーズは状態確認のみで、変更は不要な想定。

- [ ] T001 `services/files/go.mod` が module `github.com/wingarc-ktq/speckit-workshop-server/services/files` として作成済みで `go build ./...` が通ることを確認（作成済み）
- [ ] T002 [P] `services/files/sqlc.yaml` が auth と同型で作成済みであることを確認（作成済み）
- [ ] T003 [P] `services/files/Dockerfile` が auth を雛形にポート `8082` へ調整済みであることを確認（作成済み）
- [ ] T004 [P] `services/files/.env.sample` に `FILES_SERVICE_PORT` / `FILES_DATABASE_URL` / `FILES_JWT_PUBLIC_KEY_PATH` / `FILES_STORAGE_DIR` が定義済みであることを確認（作成済み）
- [ ] T005 `services/files/internal/config/config.go` が上記環境変数を読み込む実装済みであることを確認（作成済み）
- [ ] T006 `services/files/cmd/server/main.go` が現状 `config.Load()` のみの仮実装であることを確認（Phase 6 の T042 で `internal/server.Run` 呼び出しへ差し替える）
- [ ] T007 リポジトリ直下 `go.work` の `use` に `./services/files` が追加済みであることを確認（作成済み）
- [ ] T008 リポジトリ直下 `Makefile` の `gen` / `test-unit` / `test-integration` / `lint` / `fmt` が `services/files` も呼ぶよう更新済みであることを確認（作成済み）

**Checkpoint**: 骨組みが揃っており、以降のフェーズはコード追加のみで進められる。

---

## Phase 2: Foundational（すべてのユーザーストーリーの前提）

**Purpose**: US1〜US3 すべてが依存する DB スキーマ・生成コード・ドメイン層・ポート定義・運用エンドポイントを用意する。

**⚠️ CRITICAL**: このフェーズが終わるまでユーザーストーリーの実装は開始できない。

- [X] T009 [P] `services/files/migrations/000001_create_files_table.up.sql` 作成 — `files` テーブル（id, name, size, mime_type, description, storage_key, tag_ids UUID[], uploaded_at）+ `idx_files_name` / `idx_files_uploaded_at` / `idx_files_tag_ids`(GIN) を [data-model.md](./data-model.md) の DDL どおりに作成
- [X] T010 [P] `services/files/migrations/000001_create_files_table.down.sql` 作成 — `DROP TABLE IF EXISTS files;`
- [X] T011 `services/files/migrations/embed.go` 作成 — `//go:embed *.sql` で埋め込み（`services/auth/migrations/embed.go` を参考。依存: T009, T010）
- [X] T012 `services/files/internal/infra/repo/queries/files.sql` 作成 — `CreateFile :one` / `GetFileByID :one` / `ListFiles :many`（`sqlc.narg('search')` / `sqlc.narg('tag_ids')` によるオプショナル条件 + `COUNT(*) OVER()` で総件数取得。`ORDER BY uploaded_at DESC` 固定。[data-model.md](./data-model.md) 参照。依存: T009）
- [X] T013 `make -C services/files gen-sqlc` を実行し `internal/infra/repo/db/*.sql.go` を生成（生成物は手編集禁止。依存: T012）
- [X] T014 [P] `services/files/internal/domain/file.go` 作成 — `File` 構造体（ID, Name, Size, MimeType, Description, StorageKey, TagIDs, UploadedAt）+ sentinel error（`ErrFileNotFound`, `ErrFileTooLarge`, `ErrFileEmpty`）。外部依存なし（Constitution III）
- [X] T015 `services/files/internal/usecase/port.go` 作成 — `FileRepository`（Create/FindByID/List）と `FileStorage`（Save/Open）インターフェース定義 + `//go:generate mockgen` ディレクティブ（`services/auth/internal/usecase/port.go` を参考。依存: T014）
- [X] T016 `make -C services/files gen-mocks` を実行し `internal/usecase/mock/port_mock.go` を生成（依存: T015）
- [X] T017 [P] `services/files/internal/handler/health.go` 作成 — `/healthz`（liveness）・`/readyz`（DB 接続確認込み readiness）。`services/auth/internal/handler/health.go` を参考
- [X] T018 [P] `services/files/internal/handler/health_test.go` 作成 — health ハンドラの単体テスト（依存: T017）

**Checkpoint**: DB スキーマ・生成コード・ドメイン型・ポート定義・モック・ヘルスチェックが揃い、US1〜US3 を並行して開始できる。

---

## Phase 3: User Story 1 - ファイルアップロード (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイルと説明文を送信し、ファイル情報（ID・名前・サイズ・MIME タイプ・ダウンロード URL）が返る。

**Independent Test**: `file_usecase_test.go` の `UploadFile` テーブル駆動テスト（正常系／file 未指定／10MB 超過）と `file_handler_test.go` の `UploadFile` テスト（201/400/413）で、サーバー起動なしに検証できる。

- [X] T019 [US1] `services/files/internal/usecase/file_usecase.go` に `UploadFile` 実装 — **実装時の設計変更**: `multipart.Reader` は前方読み取り専用（次の Part に進むと前の Part は破棄される）ため、実際の Part 走査は handler（T025）側に置いた。usecase は転送方式に依存しない `UploadFileInput{Content io.Reader, ...}` を受け取り、`io.LimitReader(content, MaxFileSize+1)` で 10MB 超過を検出する（[research.md](./research.md) Decision 4 の意図＝ディスクに書き切る前に超過を判定、は維持）。`FileRepository`/`FileStorage` インターフェースのみに依存（依存: T015）
- [X] T020 [US1] `services/files/internal/usecase/file_usecase_test.go` に `UploadFile` のテーブル駆動単体テスト追加 — 正常系／`file` 未指定→`ErrFileEmpty`／10MB 超過→`ErrFileTooLarge`（`go.uber.org/mock` でモック化。依存: T016, T019）
- [X] T021 [P] [US1] `services/files/internal/infra/storage/local.go` 作成 — `FileStorage` ポートのローカルファイルシステム実装（`FILES_STORAGE_DIR` 配下に `{fileID}` で保存する `Save`/`Open`。依存: T015）
- [X] T022 [P] [US1] `services/files/internal/infra/storage/local_test.go` 作成 — `t.TempDir()` を使った `Save`/`Open` ラウンドトリップの単体テスト（依存: T021）
- [X] T023 [P] [US1] `services/files/internal/infra/repo/file_repository.go` 作成 — `usecase.FileRepository` の実装。まず `Create`（sqlc 生成コードをラップし DB 生成の `uploaded_at` を書き戻す）を実装（依存: T013, T015）
- [X] T024 [US1] `services/files/internal/infra/repo/file_repository_test.go` 作成 — `Create` の `testcontainers-go` 統合テスト（依存: T023）
- [X] T025 [US1] `services/files/internal/handler/file_handler.go` 作成 — `StrictServerInterface` 実装の起点として `UploadFile` を実装。`gen.UploadFileRequestObject.Body`（`*multipart.Reader`）の Part 走査（`file`/`description`/`tagIds`）をここで行い（`file` パートは前方読み取り専用の制約上、遭遇時点で `MaxFileSize+1` まで読み切って usecase 向けの `io.Reader` を作る）、usecase を呼び出して `domain.Err*` を 400 (`INVALID_PARAMETER`) / 413 (`FILE_TOO_LARGE`) の `ErrorResponse` にマッピング（依存: T019）
- [X] T026 [US1] `services/files/internal/handler/file_handler_test.go` 作成 — `UploadFile` のテスト（201 正常系／400 `file` 未指定／413 サイズ超過。usecase はモック化。依存: T025）

**Checkpoint**: User Story 1 が usecase/handler の単体テストレベルで独立して検証可能。

---

## Phase 4: User Story 2 - ファイル一覧取得と検索 (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイル一覧を取得し、キーワード検索・タグフィルタ・ページネーションで絞り込める。

**Independent Test**: `file_usecase_test.go` の `ListFiles` テーブル駆動テスト（デフォルトページング／検索一致／タグフィルタ／0 件）と `file_handler_test.go` の `GetFiles` テスト（200 のレスポンス形状）で検証できる。ストレージ実装（US1）に依存しないため最短で動作確認できる（`docs/MY-DAY2-GUIDE.md` 8章）。

- [X] T027 [US2] `services/files/internal/usecase/file_usecase.go` に `ListFiles` 実装 — `ListFilesParams`（Page/Limit/Search/TagIDs）を受け取り `FileRepository.List` を呼び出す（同一ファイルのため依存: T019）
- [X] T028 [US2] `services/files/internal/usecase/file_usecase_test.go` に `ListFiles` のテーブル駆動単体テスト追加 — デフォルトページング／検索一致／タグフィルタ／0 件（依存: T016, T027）
- [X] T029 [US2] `services/files/internal/infra/repo/file_repository.go` に `List` 実装 — sqlc の `ListFiles`（`COUNT(*) OVER()` 込み）をラップし `total`/`page`/`limit` を返す（同一ファイルのため依存: T023）
- [X] T030 [US2] `services/files/internal/infra/repo/file_repository_test.go` に `List` のテスト追加 — 検索・タグフィルタ・ページネーション・0 件ケースを `testcontainers-go` で検証（依存: T029）
- [X] T031 [US2] `services/files/internal/handler/file_handler.go` に `GetFiles` 実装 — `gen.GetFilesParams` → `usecase.ListFilesParams` 変換、`FileListResponse` 組み立て（同一ファイルのため依存: T025）
- [X] T032 [US2] `services/files/internal/handler/file_handler_test.go` に `GetFiles` のテスト追加 — 200（一覧・総件数・ページング）／検索／タグフィルタ／0 件（依存: T031）

**Checkpoint**: User Story 1・2 がそれぞれ独立して機能する。

---

## Phase 5: User Story 3 - ファイル詳細取得とダウンロード (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイル詳細を取得し、元のファイルをダウンロードできる。存在しない ID には 404 が返る。

**Independent Test**: `file_usecase_test.go` の `GetFile`/`DownloadFile` テスト（存在する／しない ID）と `file_handler_test.go` の `GetFile`/`DownloadFileContent` テスト（200/404）で検証できる。ストレージの `Open` は US1（T021）の実装を再利用する。

- [X] T033 [US3] `services/files/internal/usecase/file_usecase.go` に `GetFile` / `DownloadFile` 実装 — `FileRepository.FindByID` と（ダウンロード時のみ）`FileStorage.Open` を呼び出す（同一ファイルのため依存: T027）
- [X] T034 [US3] `services/files/internal/usecase/file_usecase_test.go` に `GetFile`/`DownloadFile` のテーブル駆動単体テスト追加 — 正常系／存在しない ID→`ErrFileNotFound`（依存: T016, T033）
- [X] T035 [US3] `services/files/internal/infra/repo/file_repository.go` に `FindByID` 実装 — `pgx.ErrNoRows` を `domain.ErrFileNotFound` にマッピング（同一ファイルのため依存: T029）
- [X] T036 [US3] `services/files/internal/infra/repo/file_repository_test.go` に `FindByID` のテスト追加 — 存在する／しない ID（依存: T035）
- [X] T037 [US3] `services/files/internal/handler/file_handler.go` に `GetFile` 実装 — `FileResponse` 組み立て、`downloadUrl` を `/api/v1/files/{id}/content` として組み立てる（DB には保存しない。同一ファイルのため依存: T031）
- [X] T038 [US3] `services/files/internal/handler/file_handler.go` に `DownloadFileContent` 実装 — `Content-Disposition: attachment; filename="..."` ヘッダ設定、`Body` に `FileStorage.Open` が返す `io.ReadCloser` をそのまま渡す（生成コードが自動で `Close` する。同一ファイルのため依存: T037）
- [X] T039 [US3] `services/files/internal/handler/file_handler_test.go` に `GetFile`/`DownloadFileContent` のテスト追加 — 200／404 (`FILE_NOT_FOUND`)（依存: T037, T038）

**Checkpoint**: US1〜US3 すべての usecase/handler 単体テストが揃い、Constitution IV のテスト要件（usecase 80%・handler 70%）を満たす土台ができる。

---

## Phase 6: サーバー統合（ストーリー横断）

**Purpose**: `oapi-codegen` の strict-server は `file_handler` が `StrictServerInterface` を完全実装して初めてコンパイルできるため、US1〜US3 のハンドラ実装がすべて揃った後にサーバー配線を行う。

- [X] T040 `services/files/internal/server/server.go` 作成 — DI 組み立て（config → self-migration → pgxpool → storage → repo → usecase → handler）、echo セットアップ、`packages/authjwt.Middleware` を全 files 操作（`getFiles`/`uploadFile`/`getFile`/`downloadFileContent`）に適用、`/healthz`/`/readyz` は検証ミドルウェア対象外、共通エラーハンドラ（`VALIDATION_ERROR`/`METHOD_NOT_ALLOWED`/`NOT_FOUND`）を `services/auth/internal/server/server.go` の `newOpenAPIValidator` を参考に実装（依存: T017, T025, T031, T037, T038）
- [X] T041 [P] `services/files/internal/server/server_test.go` 作成 — `newEcho()` のヘルスチェック等の単体テスト（`services/auth/internal/server/server_test.go` を参考。依存: T040）
- [X] T042 `services/files/cmd/server/main.go` を `internal/server.Run(ctx)` 呼び出しへ差し替え — `signal.NotifyContext` による graceful shutdown を含め `services/auth/cmd/server/main.go` と同型にする（依存: T040）
- [X] T043 [P] リポジトリ直下 `compose.yaml` の「files サービスは未実装」コメントを実際の `files` サービス定義に置き換え — ポート `8082`、`postgres` への `depends_on`、ファイル保存用ボリューム（`FILES_STORAGE_DIR` にマウント）、`FILES_JWT_PUBLIC_KEY_PATH` 用の `keys/` マウントを追加（[plan.md](./plan.md) Project Structure 参照）
- [X] T044 `cd services/files && go mod tidy && make lint` を実行し、`pgx/v5`/`golang-migrate/v4`/`google/uuid`/`go.uber.org/mock`/`testcontainers-go` 等の依存が正しく解決され `go build ./...` / `go vet ./...` が通ることを確認（依存: T040-T043）

**実装時に見つけた問題（openapi.yaml 側、Phase 6 の実装ミスではない）**: OpenAPI 検証ミドルウェアを実際にワイヤーして `docker compose up` で E2E 確認したところ、`schema/files/openapi.yaml` の `uploadFile` にあった `encoding.file.contentType`（許可 MIME タイプのカンマ区切りリスト）が、使用している `kin-openapi v0.140.0` では単純な完全一致文字列としてしか比較されず、どんな MIME タイプを送っても常に 400 VALIDATION_ERROR になり User Story 1 が動作しない不具合が判明した。これまでバリデータ自体が未配線だったため気づかれていなかった。ユーザーに確認の上、`encoding.contentType` を削除（MIME 制限は spec.md では Edge Cases 扱いで FR による必須要件ではないため）し、`make gen-oapi` で埋め込み spec を再生成。修正後、Docker 上でアップロード→一覧→詳細→ダウンロード→404→413 まで実際に curl で確認済み。

**Checkpoint**: `make -C services/files run` でサービスが起動し、[quickstart.md](./quickstart.md) の手順で curl による動作確認ができる。**`docker compose up -d --build` での実機確認も完了**（アップロード→一覧→詳細取得→ダウンロード→存在しないID(404)→10MB超過(413)まで全シナリオ成功）。

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Constitution 準拠の最終確認と横断的な品質担保。

- [X] T045 [P] `cd services/files && go test -cover ./...` で usecase 層 80%以上・handler 層 70%以上のカバレッジを確認（Constitution IV）— usecase 87.9%、handler 80.2% でいずれも基準達成。未カバー行は `default: return nil, err`（想定外エラーをそのまま伝播する分岐）や `io.ReadAll` の異常系など、壊れた Reader を用意しないと再現できない防御的分岐のみで、追加テストは不要と判断
- [X] T046 [P] `make -C services/files test-integration` を実行し `testcontainers-go` による repo 層統合テストが通ることを確認（Docker 必須）— `-count=1` でキャッシュを無効化し実際に PostgreSQL コンテナを起動して再検証、全パッケージ PASS
- [X] T047 [quickstart.md](./quickstart.md) の手順に従い User Story 1〜3 の Acceptance Scenario をすべて curl で手動確認 — Phase 6 の `docker compose up` 実機確認で実施済み
- [ ] T048 [P] Schemathesis で `schema/files/openapi.yaml` に対する API テストを実行しエラーがないことを確認（SC-005: 全エンドポイントが API 仕様と一致）— Day 3 の範囲のため今回は対象外
- [ ] T049 [P] Hurl でアップロード→一覧→詳細→ダウンロードのシナリオテストを `api-tests/hurl` 配下に作成・実行（既存の auth 用シナリオ構成を参考）— Day 3 の範囲のため今回は対象外
- [X] T050 公開シンボル（`domain.File`, `usecase.FileRepository`/`FileStorage`, ハンドラの公開メソッド等）に GoDoc コメントが付いていることを確認（Constitution I）— 全パッケージを `go doc -all` で監査し全公開シンボルにコメント有りを確認。副次的に `internal/handler/health.go` に残っていた重複パッケージコメント（Phase 3 で `file_handler.go` にも追加されたため二重になっていた）を削除

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 依存なし。前回セッションで作成済みのため状態確認のみ
- **Foundational (Phase 2)**: Setup 完了後。**US1〜US3 すべてをブロックする**
- **User Stories (Phase 3-5)**: すべて Foundational 完了後に開始可能。US1/US2/US3 は独立した usecase・repo メソッドを実装するため理論上並行着手できるが、以下の**同一ファイル制約**に注意
- **サーバー統合 (Phase 6)**: `StrictServerInterface` の完全実装が必要なため **US1・US2・US3 すべてのハンドラ実装（T025, T031, T037, T038）完了後**
- **Polish (Phase 7)**: Phase 6 完了後

### 同一ファイル制約（[P] 判定の根拠）

`file_usecase.go` / `file_repository.go` / `file_handler.go` は US1〜US3 で共通のファイルに追記していく構成のため、以下は **同一ファイル間で並行不可**（ストーリーが異なっても直列）:

- `file_usecase.go`: T019(US1) → T027(US2) → T033(US3)
- `file_repository.go`: T023(US1) → T029(US2) → T035(US3)
- `file_handler.go`: T025(US1) → T031(US2) → T037→T038(US3)

一方、`internal/domain/file.go`・`internal/usecase/port.go`・`internal/infra/storage/local.go` は各ストーリーの usecase 実装（T019/T027/T033）と異なるファイルであり、Foundational のポート定義（T015）にのみ依存するため、T021（storage 実装）と T023（repo Create 実装）は T019（usecase 実装）と **並行して着手可能**（[P] 表記のとおり）。

### User Story Dependencies

- **User Story 1 (P1, アップロード)**: Foundational 完了後に開始可能。他ストーリーへの依存なし
- **User Story 2 (P1, 一覧・検索)**: Foundational 完了後に開始可能。ストレージ実装が不要なため US1 を待たずに独立して完了できる（ただし `file_usecase.go`/`file_repository.go`/`file_handler.go` への追記が US1 の後になる場合はその分の直列待ちが発生する）
- **User Story 3 (P1, 詳細・ダウンロード)**: `FileStorage.Open`（T021, US1 で実装）を利用するため、**ストレージ実装のみ US1 に依存**。usecase/repo/handler への追記順は同一ファイル制約に従う

### Within Each User Story

- usecase 実装 → usecase 単体テスト → infra 実装 → infra 統合テスト → handler 実装 → handler テスト、の順（`docs/MY-DAY2-GUIDE.md` 2章の推奨順序）
- テストは対応する実装タスクの直後に配置。実装前に「先に落ちるテストを書く」運用にする場合は各ストーリー内でテストタスクを実装タスクより前に並べ替えてもよい

### Parallel Opportunities

- Foundational: T009/T010（migrations の up/down）、T014（domain）、T017/T018（health）は互いに [P]
- US1: T019（usecase）, T021（storage）, T023（repo Create）は Foundational 完了後に並行着手可能。T020/T022/T024（それぞれのテスト）も対応する実装後は互いに並行可能
- Phase 6: T041（server_test.go）と T043（compose.yaml）は T040 完了後に並行可能
- Phase 7: T045/T046/T048/T049 は互いに並行可能

---

## Parallel Example: User Story 1

```bash
# Foundational 完了後、以下は並行着手可能:
Task: "internal/usecase/file_usecase.go に UploadFile 実装"
Task: "internal/infra/storage/local.go 作成（FileStorage 実装）"
Task: "internal/infra/repo/file_repository.go に Create 実装"
```

---

## Implementation Strategy

### MVP First（User Story 2 から着手する場合）

`docs/MY-DAY2-GUIDE.md` 8章の切り方に従うなら、時間が足りない場合は以下の順で「動くサービス」を最短で成立させる:

1. Phase 1（Setup, 確認のみ）+ Phase 2（Foundational）を完了
2. User Story 2（一覧取得, T027-T032）のみを実装 — ストレージ不要のため最速
3. Phase 6 の T040 は US1/US3 のハンドラが未実装だと `StrictServerInterface` を満たせないため、**この時点ではサーバーを起動できない**点に注意。動作確認は usecase/handler の単体テストレベルで行う
4. 時間があれば User Story 1 → User Story 3 の順で追加し、揃った時点で Phase 6 に進む

### Incremental Delivery（正攻法）

1. Setup + Foundational → 基盤完成
2. User Story 1（アップロード）追加 → 単体テストで検証
3. User Story 2（一覧・検索）追加 → 単体テストで検証
4. User Story 3（詳細・ダウンロード）追加 → 単体テストで検証
5. Phase 6（サーバー統合）→ `make run` して curl で E2E 確認（[quickstart.md](./quickstart.md)）
6. Phase 7（Polish）→ カバレッジ・Schemathesis・Hurl・GoDoc の最終確認

---

## Notes

- [P] タスク = 別ファイルかつ依存タスク完了済み
- [Story] ラベルはトレーサビリティのためのユーザーストーリー対応
- `internal/domain` は何にも依存しない。`internal/infra` は `internal/usecase` (`port.go`) が定義したインターフェースを実装する（Constitution III）
- JWT 検証は `packages/authjwt` を利用するのみで、自前のミドルウェア実装タスクは存在しない（Constitution VII）
- 生成コード（`api/gen/server.gen.go`, `internal/infra/repo/db/*.sql.go`, `internal/usecase/mock/*.go`）は直接編集しない。修正が必要な場合は元（`schema/files/openapi.yaml` / `*.sql` / `port.go`）を直して再生成する
- testify は使用しない。標準 `testing` + `go.uber.org/mock` + テーブル駆動テストで統一する（Constitution IV）
- コミットは各タスク、または論理的なまとまりごとに行う
