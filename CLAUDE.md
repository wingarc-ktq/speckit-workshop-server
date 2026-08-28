@AGENTS.md

## Active Technologies
- Go 1.26 + echo/v4, oapi-codegen/v2（strict-server + echo-server, embedded-spec）, kin-openapi（リクエスト検証）, oapi-codegen/echo-middleware, golang-jwt/v5（`packages/authjwt` 経由）, pgx/v5, sqlc, golang-migrate/v4, google/uuid (002-document-management)
- PostgreSQL 17（`files` データベース、`auth` とは独立。Constitution VI）。ファイル本体はストレージポート（`FileStorage`）越しにローカルファイルシステム（`FILES_STORAGE_DIR`）へ保存。メタデータのみ PostgreSQL に永続化 (002-document-management)

## Recent Changes
- 002-document-management: Added Go 1.26 + echo/v4, oapi-codegen/v2（strict-server + echo-server, embedded-spec）, kin-openapi（リクエスト検証）, oapi-codegen/echo-middleware, golang-jwt/v5（`packages/authjwt` 経由）, pgx/v5, sqlc, golang-migrate/v4, google/uuid
