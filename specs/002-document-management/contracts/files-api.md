# Files MVP HTTP 契約

正規の OpenAPI ソースは `schema/files/openapi.yaml` とする。本書は P1 の契約レビュー用の要約であり、実装前に OpenAPI へ反映し、`services/files/make gen` で型・ServerInterface を生成する。

## 共通

- ベースパス: `/api/v1`
- 認証: `Authorization: Bearer <RS256 JWT>`。health/ready を除く全エンドポイントで必須
- エラー JSON: `{ "code": "ERROR_CODE", "message": "..." }`
- 最大アップロードサイズ: 10 MiB
- すべての ID は UUID

## P1 エンドポイント

| Method | Path | 用途 | 成功 |
|---|---|---|---|
| POST | `/files` | multipart で本体、任意 description、任意 tagIds を登録 | 201 File |
| GET | `/files` | page/limit、name keyword、tagIds で一覧・検索 | 200 FileList |
| GET | `/files/{fileId}` | メタデータと downloadUrl を取得 | 200 File |
| GET | `/files/{fileId}/download` | 本体をストリーム返却 | 200 binary |
| GET | `/healthz` | liveness | 200 |
| GET | `/readyz` | DB readiness | 200/503 |

一覧の既定値は `page=1`、`limit=20`。`page >= 1`、`1 <= limit <= 100` とし、不正値は 400 `VALIDATION_ERROR`。keyword はファイル名の部分一致、複数 tagIds は指定したタグをすべて含むファイルを返す。

## エラー

| HTTP | code | 条件 |
|---:|---|---|
| 400 | `VALIDATION_ERROR` | multipart、description、ページネーション、UUID が不正 |
| 401 | `UNAUTHORIZED` | JWT 未指定・無効 |
| 404 | `FILE_NOT_FOUND` | fileId が存在しない |
| 409 | `TAG_NOT_FOUND` | upload 指定 tag ID が存在しない |
| 413 | `FILE_TOO_LARGE` | 10 MiB 超過 |
| 500 | `INTERNAL_ERROR` | 予期しない永続化・ストレージ障害 |

## P2 の明示的除外

タグ作成・一覧・更新・削除、ファイルメタデータ編集、個別/一括削除はこの MVP では実装しない。
