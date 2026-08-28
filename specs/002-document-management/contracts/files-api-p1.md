# Interface Contract: Files API（P1 スコープ）

正本は `schema/files/openapi.yaml`（Constitution II: OpenAPI ファースト）。本ファイルはその中から **本実装フェーズ（P1）の対象** を一覧化した索引であり、内容を重複定義しない。

## 実装対象エンドポイント（P1）

| Method | Path | operationId | 認証 | 対応するユーザーストーリー |
|---|---|---|---|---|
| GET | `/files` | `getFiles` | 必須 | User Story 2（一覧・検索） |
| POST | `/files` | `uploadFile` | 必須 | User Story 1（アップロード） |
| GET | `/files/{fileId}` | `getFile` | 必須 | User Story 3（詳細） |
| GET | `/files/{fileId}/content` | `downloadFileContent` | 必須 | User Story 3（ダウンロード） |

運用エンドポイント `GET /healthz` / `GET /readyz` は Constitution II の例外規定によりこの OpenAPI 契約の対象外・認証不要（`/api/v1` 配下に置かない）。

## 実装対象外（P2, 本フェーズではスキーマにも未定義）

以下は `spec.md` User Story 4〜6 に対応するが、`schema/files/openapi.yaml` 自体に未定義のため、本実装フェーズのタスクには含まれない。P2 着手時に別途 OpenAPI を先に更新すること（Constitution II）。

- タグ CRUD（`POST/GET/PATCH/DELETE /tags` 相当, FR-011〜016）
- ファイルメタデータ編集（`PATCH /files/{fileId}` 相当, FR-017）
- ファイル削除・一括削除（`DELETE /files/{fileId}`, 一括削除エンドポイント相当, FR-018〜019）

## エラーコード一覧（P1 で使用するもの）

`schema/files/openapi.yaml` の `ErrorResponse`（`message` + `code`）に準拠。

| HTTP Status | code | 発生条件 |
|---|---|---|
| 400 | `INVALID_PARAMETER` | 必須パラメータ欠如、ページネーション不正など |
| 401 | `UNAUTHORIZED` | JWT 未指定・無効 |
| 404 | `FILE_NOT_FOUND` | 存在しない `fileId` |
| 413 | `FILE_TOO_LARGE` | ファイルサイズが 10MB 超過 |

内部的な検証エラー（OpenAPI バリデーションミドルウェア由来）は auth と同様に `VALIDATION_ERROR` / `NOT_FOUND` / `METHOD_NOT_ALLOWED` にもマッピングされ得る（`services/auth/internal/server/server.go` の `newOpenAPIValidator` と同じエラーハンドラを Files でも踏襲する）。

## 生成済みコードとの対応

`services/files/api/gen/server.gen.go`（`make gen-oapi` 済み）の `StrictServerInterface` を `internal/handler` で実装する:

```go
type StrictServerInterface interface {
    GetFiles(ctx context.Context, request GetFilesRequestObject) (GetFilesResponseObject, error)
    UploadFile(ctx context.Context, request UploadFileRequestObject) (UploadFileResponseObject, error)
    GetFile(ctx context.Context, request GetFileRequestObject) (GetFileResponseObject, error)
    DownloadFileContent(ctx context.Context, request DownloadFileContentRequestObject) (DownloadFileContentResponseObject, error)
}
```

注意: `UploadFileRequestObject.Body` は `*multipart.Reader`（型付き struct ではない）。詳細は [research.md](../research.md) Decision 4、および [data-model.md](../data-model.md) を参照。
