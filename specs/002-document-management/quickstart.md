# Files MVP クイックスタート

## 前提

- Go 1.26、Docker、PostgreSQL 17
- Files サービス用の JWT 公開鍵。Auth の秘密鍵は Files に配置しない
- `schema/files/openapi.yaml` が P1 契約で埋まっていること

## 準備

```bash
make keys
make up-db
cd services/files
go mod tidy
make gen
```

生成後、`api/gen/` と `internal/infra/repo/db/` は手編集しない。環境変数で Files DB URL、JWT 公開鍵パス、storage root、port を設定する。

## 単体・静的検証

```bash
cd services/files
go fmt ./...
go vet ./...
make test-unit
```

usecase は storage/repository の mock で、10 MiB 境界、description 500 文字、未検出、補償削除、ページネーションを検証する。storage adapter は一時ディレクトリで保存・読出し・削除を検証する。

## 統合検証

```bash
cd services/files
make test-integration
```

PostgreSQL testcontainer に migration を適用し、metadata insert、名前検索、tag filter、総件数、ページングを確認する。

## API シナリオ

サービスを起動して次を確認する。

1. 有効 JWT で 5MB の multipart upload が 201 となり、id/name/size/mimeType/downloadUrl が返る。
2. 10 MiB 超の upload が 413 `FILE_TOO_LARGE` となる。
3. JWT なしの upload/list/detail/download が 401 `UNAUTHORIZED` となる。
4. list の既定値が page 1、limit 20 であり、keyword と複数 tagIds の検索結果および total が正しい。
5. detail がメタデータと downloadUrl を返し、download が元のバイナリと MIME タイプを返す。
6. 存在しない UUID の detail/download が 404 `FILE_NOT_FOUND` となる。
7. `/healthz` は DB 障害に依存せず liveness、`/readyz` は DB 接続状態を反映する。

Schemathesis は `schema/files/openapi.yaml` を対象に、Hurl は上記の固定シナリオを対象に実行する。
