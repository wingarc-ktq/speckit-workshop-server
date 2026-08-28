# Phase 1 Data Model: Files サービス（P1 MVP）

対象は `spec.md` の Key Entities のうち、P1（User Story 1〜3）で実際に永続化・operates が必要なものに限定する。
Tag エンティティ自体の CRUD（P2, User Story 4）は対象外。ただし `File.tagIds` は P1 契約（`schema/files/openapi.yaml`）に存在するため保持する（[research.md](./research.md) Decision 6）。

## File（ファイル）

### ドメインモデル（`internal/domain/file.go`）

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| ID | `uuid.UUID` | ✓ | 主キー |
| Name | `string` | ✓ | アップロード時の元ファイル名。最大 255 文字。一意制約なし（FR に基づき同名ファイルは別ファイルとして許容） |
| Size | `int64` | ✓ | バイト数。1〜10,485,760（10MB）の範囲 |
| MimeType | `string` | ✓ | アップロード時に multipart パートの `Content-Type` から取得 |
| Description | `string` | - | 最大 500 文字。空文字許容 |
| StorageKey | `string` | ✓ | `FileStorage` ポートに渡す一意キー（P1 実装では `ID.String()`）。API レスポンスには非公開（ダウンロード URL 経由でのみアクセス） |
| TagIDs | `[]uuid.UUID` | ✓（空配列可） | アップロード時に指定されたタグ ID の集合。P1 では存在検証を行わない（Tag CRUD 未実装のため） |
| UploadedAt | `time.Time` | ✓ | DB 側 `DEFAULT NOW()` |

### ドメインエラー（`internal/domain/file.go`）

```go
var (
    ErrFileNotFound = errors.New("file not found")
    ErrFileTooLarge = errors.New("file too large")
    ErrFileEmpty    = errors.New("file is required")
)
```

### バリデーションルール（FR-001〜003, FR-010, Edge Cases）

- `file` パートは必須。欠如時は 400 + `INVALID_PARAMETER`。
- ファイルサイズは 10MB（10,485,760 バイト）以下。超過時は 413 + `FILE_TOO_LARGE`（[research.md](./research.md) Decision 4）。
- `description` は最大 500 文字。OpenAPI 検証ミドルウェア（kin-openapi）でリクエストレベル検証済み。
- 存在しない `fileId` を指定した場合は 404 + `FILE_NOT_FOUND`（`ErrFileNotFound` を handler で HTTP にマッピング）。

### 状態遷移

Files エンティティは P1 の範囲では単純な作成・参照のみで、状態遷移は持たない（更新・削除は P2 の User Story 5/6 で追加される）。

## 永続化スキーマ（PostgreSQL, `files` データベース）

`services/files/migrations/000001_create_files_table.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS files (
    id           UUID PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    size         BIGINT NOT NULL,
    mime_type    VARCHAR(255) NOT NULL,
    description  VARCHAR(500) NOT NULL DEFAULT '',
    storage_key  VARCHAR(255) NOT NULL,
    tag_ids      UUID[] NOT NULL DEFAULT '{}',
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_files_name ON files (name);
CREATE INDEX IF NOT EXISTS idx_files_uploaded_at ON files (uploaded_at);
CREATE INDEX IF NOT EXISTS idx_files_tag_ids ON files USING GIN (tag_ids);
```

- `idx_files_name`: `ILIKE '%keyword%'` 検索（FR-005）を高速化する目的だが、部分一致検索の性質上フルスキャンに近くなる可能性がある。SC-003（100 件規模で 1 秒以内）は満たすが、件数が大きく増える場合は `pg_trgm` 拡張の検討が P2 以降の課題となる（本 MVP では不要）。
- `idx_files_uploaded_at`: デフォルトソート（`-uploadedAt`）および日時ソートの高速化。
- `idx_files_tag_ids`（GIN）: タグ ID 配列に対する `&&`（オーバーラップ）検索（FR-006）の高速化。

## sqlc クエリ設計（`internal/infra/repo/queries/files.sql`）

| クエリ名 | 種別 | 用途 |
|---|---|---|
| `CreateFile` | `:one` | アップロード時の INSERT。DB 生成の `uploaded_at` を返す |
| `GetFileByID` | `:one` | 詳細取得・ダウンロード時のメタデータ取得。`pgx.ErrNoRows` → `domain.ErrFileNotFound` |
| `ListFiles` | `:many` | 一覧取得。`sqlc.narg('search')`, `sqlc.narg('tag_ids')` によるオプショナル条件 + `COUNT(*) OVER()` で総件数を同時取得。`ORDER BY` は Go 側で許可された列に限定して組み立てる（[research.md](./research.md) Decision 5） |

## API DTO とドメインの対応（`internal/handler`）

| OpenAPI スキーマ | ドメイン | 変換方向 |
|---|---|---|
| `FileInfo` | `domain.File` | handler が `domain.File` → `gen.FileInfo` へマッピング。`downloadUrl` はハンドラで `/api/v1/files/{id}/content` を組み立てる（DB には保存しない） |
| `FileResponse` | `domain.File` 1件 | `UploadFile`（201）, `GetFile`（200）で使用 |
| `FileListResponse` | `[]domain.File` + `total`/`page`/`limit` | `GetFiles`（200）で使用 |
| `ErrorResponse` | `domain` の sentinel error | handler の共通エラーマッピング関数でコード（`INVALID_PARAMETER` / `UNAUTHORIZED` / `FILE_NOT_FOUND` / `FILE_TOO_LARGE`）に変換 |

## ページネーション・検索パラメータ（usecase 層, `internal/usecase/port.go`）

```go
type ListFilesParams struct {
    Page    int           // 1 始まり。デフォルト 1
    Limit   int           // デフォルト 20、最大 100
    Search  string        // ファイル名部分一致。空文字は条件なし
    TagIDs  []uuid.UUID   // 空は条件なし（OR ではなく "指定したタグのいずれかを含む" = 配列オーバーラップ）
}
```

- `page < 1` または `limit` が範囲外（1〜100 逸脱）の場合、OpenAPI 検証ミドルウェアが 400 を返す（FR: 不正なページネーションパラメータにはバリデーションエラー）。
- `Offset = (Page - 1) * Limit` として `ListFiles` クエリに渡す。
- **並び順は `uploaded_at DESC` 固定**（内部仕様）。`spec.md` にソートの機能要件が存在しないため、API パラメータとしては公開しない。一覧に安定した順序は必要なので、実装側で新しい順に固定する。ソート機能が要求された時点で `sort` クエリパラメータと `SortField` 型を追加する。
