# Tasks: 文書管理（Files MVP）

**Input**: `/specs/002-document-management/` の設計ドキュメント
**Prerequisites**: plan.md（必須）、spec.md（ユーザーストーリー用）、data-model.md、contracts/files-api.md

---

## Phase 1: Setup（プロジェクト初期化）

**Purpose**: Files サービスの初期構造と OpenAPI 契約の土台を作る

- [X] T001 Create the OpenAPI 3.0 contract in schema/files/openapi.yaml
- [X] T002 [P] Define shared error schema and file response models in schema/files/openapi.yaml
- [ ] T003 [P] Create the Files service directory layout under services/files/
- [ ] T004 [P] Initialize the Go module and dependencies in services/files/go.mod
- [ ] T005 [P] Add project commands and generation targets in services/files/Makefile
- [ ] T006 [P] Add repository ignore rules to .gitignore

---

## Phase 2: Foundational（全ユーザーストーリーの前提条件）

**Purpose**: すべてのユーザーストーリー実装前に完了する共通基盤

- [ ] T007 Create configuration loading and validation in services/files/internal/config/config.go
- [ ] T008 [P] Add configuration tests in services/files/internal/config/config_test.go
- [X] T009 [P] Create the files table migration in services/files/migrations/000001_create_files_table.up.sql
- [X] T010 [P] Create the reverse migration in services/files/migrations/000001_create_files_table.down.sql
- [X] T011 [P] Create the tags table migration in services/files/migrations/000002_create_tags_table.up.sql
- [X] T012 [P] Create the reverse migration in services/files/migrations/000002_create_tags_table.down.sql
- [X] T013 [P] Create the file-tags join migration in services/files/migrations/000003_create_file_tags_table.up.sql
- [X] T014 [P] Create the reverse migration in services/files/migrations/000003_create_file_tags_table.down.sql
- [X] T015 [P] Embed migrations in services/files/migrations/embed.go
- [ ] T016 [P] Define the File and Tag domain models in services/files/internal/domain/file.go and services/files/internal/domain/tag.go
- [ ] T017 [P] Define sentinel errors in services/files/internal/domain/errors.go
- [ ] T018 Add domain validation tests in services/files/internal/domain/file_test.go
- [ ] T019 [P] Define repository and storage ports in services/files/internal/usecase/port.go
- [ ] T020 [P] Create the mock implementations in services/files/internal/usecase/mock/
- [ ] T021 [P] Implement HTTP error mapping in services/files/internal/handler/error_response.go
- [ ] T022 [P] Implement request/response mapping in services/files/internal/handler/file_mapper.go
- [ ] T023 [P] Implement health endpoints in services/files/internal/handler/health.go
- [ ] T024 [P] Implement local file storage adapter in services/files/internal/infra/storage/storage.go
- [ ] T025 [P] Add path safety checks in services/files/internal/infra/storage/path.go
- [ ] T026 Add storage unit tests in services/files/internal/infra/storage/storage_test.go
- [ ] T027 [P] Add SQL query definitions in services/files/internal/infra/repo/queries/files.sql
- [ ] T028 [P] Add SQL query definitions in services/files/internal/infra/repo/queries/tags.sql
- [ ] T029 Run sqlc generation for services/files/internal/infra/repo/db/
- [ ] T030 [P] Implement file repository logic in services/files/internal/infra/repo/file_repository.go
- [ ] T031 [P] Implement tag repository logic in services/files/internal/infra/repo/tag_repository.go
- [ ] T032 [P] Implement repository mapper logic in services/files/internal/infra/repo/repo_mapper.go
- [ ] T033 Add PostgreSQL integration tests in services/files/internal/infra/repo/repository_test.go
- [ ] T034 Build the server wiring and DI setup in services/files/internal/server/server.go
- [ ] T035 [P] Configure OpenAPI validation in services/files/internal/server/openapi.go
- [ ] T036 Add server wiring tests in services/files/internal/server/server_test.go

---

## Phase 3: User Story 1 - ファイルアップロード (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイルと説明文をアップロードし、登録されたファイル情報を返す

**Independent Test**: ファイルをアップロードし、ID・名前・サイズ・MIME・downloadUrl が返ることを確認

- [X] T037 [US1] Write upload usecase tests in services/files/internal/usecase/file_usecase_test.go
- [X] T038 [US1] Implement upload orchestration in services/files/internal/usecase/file_usecase.go
- [X] T039 [US1] Add file persistence and tag registration in services/files/internal/infra/repo/file_repository.go
- [X] T040 [US1] Add POST /files handler implementation in services/files/internal/handler/file_handler.go
- [X] T041 [US1] Validate multipart input and enforce the 10 MiB limit in services/files/internal/handler/file_handler.go
- [X] T042 [P] [US1] Add upload integration checks in services/files/internal/infra/repo/repository_test.go
- [X] T043 [P] [US1] Add handler-level validation tests in services/files/internal/handler/file_handler_test.go

**Checkpoint**: User Story 1 の upload API が独立して動作する

---

## Phase 4: User Story 2 - ファイル一覧取得と検索 (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーが一覧と検索を使ってファイルを絞り込める

**Independent Test**: page/limit/name/tagIds を指定してフィルタされた一覧が返ることを確認

- [X] T044 [US2] Write list and search usecase tests in services/files/internal/usecase/file_usecase_test.go
- [X] T045 [US2] Implement list/search orchestration in services/files/internal/usecase/file_usecase.go
- [X] T046 [US2] Add filtered list and count queries in services/files/internal/infra/repo/file_repository.go
- [X] T047 [US2] Add GET /files handler implementation in services/files/internal/handler/file_handler.go
- [X] T048 [US2] Validate query parameters and pagination in services/files/internal/handler/file_handler.go
- [ ] T049 [P] [US2] Add repository search integration tests in services/files/internal/infra/repo/repository_test.go
- [ ] T050 [P] [US2] Add handler list/query tests in services/files/internal/handler/file_handler_test.go

**Checkpoint**: User Story 2 の一覧・検索 API が独立して動作する

---

## Phase 5: User Story 3 - ファイル詳細取得とダウンロード (Priority: P1) 🎯 MVP

**Goal**: 認証済みユーザーがファイル詳細と元データのダウンロードを取得できる

**Independent Test**: 詳細の取得と download の正規化を確認し、存在しない ID は 404 を返す

- [X] T051 [US3] Write detail and download usecase tests in services/files/internal/usecase/file_usecase_test.go
- [X] T052 [US3] Implement detail and download orchestration in services/files/internal/usecase/file_usecase.go
- [X] T053 [US3] Add metadata lookup and storage stream retrieval in services/files/internal/infra/repo/file_repository.go
- [X] T054 [US3] Add GET /files/{fileId} handler implementation in services/files/internal/handler/file_handler.go
- [X] T055 [US3] Add GET /files/{fileId}/download handler implementation in services/files/internal/handler/file_handler.go
- [ ] T056 [P] [US3] Add repository detail retrieval tests in services/files/internal/infra/repo/repository_test.go
- [ ] T057 [P] [US3] Add handler detail and download tests in services/files/internal/handler/file_handler_test.go

**Checkpoint**: User Story 3 の詳細とダウンロード API が独立して動作する

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: すべての user story をまとめて確認し、実運用に近い品質を整える

- [ ] T058 [P] Create the Hurl scenario for upload → list → detail → download in services/files/api-test/files.hurl
- [ ] T059 [P] Add fixture files under services/files/api-test/fixtures/
- [ ] T060 [P] Run gofmt and go vet across services/files/ and related packages
- [ ] T061 Run unit and integration tests for services/files/ with the Go test suite
- [ ] T062 Add final README and setup notes in services/files/README.md
- [ ] T063 Review and add GoDoc comments to public symbols in services/files/
- [ ] T064 Validate the OpenAPI contract against the implementation using `make gen` and the generated code

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1**: 依存なし、すぐ着手可能
- **Phase 2**: Phase 1 完了後に開始。すべての user story をブロック
- **Phase 3-5**: Phase 2 完了後に開始。各 user story は独立して検証可能
- **Phase 6**: すべての user story 完了後に実行

### User Story Order

- **User Story 1 (P1)**: Phase 2 完了後に開始
- **User Story 2 (P1)**: Phase 2 完了後に開始（US1 と独立して開発可能）
- **User Story 3 (P1)**: Phase 2 完了後に開始（US1/US2 と独立して開発可能）

### Parallel Opportunities

- **Phase 1**: T003, T004, T005, T006 can run in parallel
- **Phase 2**: T009-T015 (migrations), T016-T018 (domain), T019-T023 (ports/handler), T024-T025 (storage), T027-T028 (queries), T030-T032 (repo), T034-T036 (server) can be parallelized by file group
- **Phase 3-5**: Each story can proceed independently once Phase 2 is complete
- **Phase 6**: T058, T059, T060 can run in parallel

---

## Parallel Execution Examples

```bash
# Phase 1: parallel setup tasks
Task: T003, T004, T005, T006

# Phase 2: migration and domain work
Task: T009, T010, T011, T012, T013, T014
Task: T016, T017, T018
Task: T024, T025, T026
Task: T030, T031, T032

# Story execution examples
Task: T037, T038, T039, T040, T041
Task: T044, T045, T046, T047, T048
Task: T051, T052, T053, T054, T055
```

---

## Implementation Strategy

### MVP First

1. Phase 1: Setup
2. Phase 2: Foundational
3. Phase 3: User Story 1
4. Validate upload independently
5. Proceed to User Story 2 and User Story 3

### Incremental Delivery

1. 完成済みの共通基盤を作る
2. US1 を先に完了し、アップロードからデモを確認
3. US2 を追加し、一覧と検索を確認
4. US3 を追加し、詳細とダウンロードを確認
5. 最後に polish と E2E 検証を行う

---

## Notes

- [P] tasks represent independent work across different files.
- Each story is independently testable and can be completed without depending on the others.
- The generated work is ordered to keep the OpenAPI contract and migration setup as the foundation for all implementation.
