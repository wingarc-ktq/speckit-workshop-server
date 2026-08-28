# Quickstart: Files サービス（P1）動作確認

本フェーズ（P1: アップロード / 一覧・検索 / 詳細・ダウンロード）が実装された後、以下の手順でエンドツーエンドに動作確認する。
API 詳細は [contracts/files-api-p1.md](./contracts/files-api-p1.md)、データ設計は [data-model.md](./data-model.md) を参照。

## 前提条件

- Go 1.26 / Docker / `migrate` CLI / `sqlc` / `oapi-codegen` がインストール済み（`services/auth` と同じ開発環境）
- リポジトリ直下で `make keys`（未実行の場合）により RS256 開発鍵ペアを生成済み
- `services/auth` が起動しログインできる状態（JWT 取得のため）

## 1. 依存関係とコード生成

```bash
cd services/files
go mod tidy            # go.mod 作成後、初回のみ
make gen               # oapi-codegen + sqlc + go generate(mock)
```

期待結果: `api/gen/server.gen.go` が最新化され、`internal/infra/repo/db/*.sql.go` と `internal/usecase/mock/*.go` が生成される。

## 2. DB 起動 と マイグレーション

```bash
# リポジトリ直下
docker compose up -d postgres
createdb  # 不要: サービス自身が起動時に self-migration するため事前準備不要（services/auth と同様）
```

`internal/server.Run` が起動時に `migrations/*.sql`（埋め込み）を自動適用するため、`make migrate-up` の手動実行は必須ではない（確認したい場合のみ個別実行可）。

## 3. サービス起動

```bash
cp .env.sample .env   # 未作成の場合
export $(cat .env | xargs)
make run              # go run ./cmd/server, デフォルト :8082
```

期待結果: ログに `Files service listening on :8082` が出力される。

```bash
curl -s http://localhost:8082/healthz   # => 200
curl -s http://localhost:8082/readyz    # => 200 (DB接続確認込み)
```

## 4. JWT 取得（auth サービス経由）

```bash
TOKEN=$(curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"password123"}' | jq -r .accessToken)
```

Files サービスは Auth と同じ鍵ペア（公開鍵）を `FILES_JWT_PUBLIC_KEY_PATH` に設定していること。

## 5. User Story 1: アップロード（Acceptance Scenario 1〜3 の確認）

```bash
# 正常系: ファイル + 説明文
curl -s -X POST http://localhost:8082/api/v1/files \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@./sample.pdf" \
  -F "description=2026年8月分の請求書" | jq

# 異常系: 10MB超過 → 413 FILE_TOO_LARGE
dd if=/dev/zero of=/tmp/big.bin bs=1M count=11
curl -s -o /dev/null -w '%{http_code}\n' -X POST http://localhost:8082/api/v1/files \
  -H "Authorization: Bearer $TOKEN" -F "file=@/tmp/big.bin"   # => 413

# 異常系: JWT無し → 401 UNAUTHORIZED
curl -s -o /dev/null -w '%{http_code}\n' -X POST http://localhost:8082/api/v1/files \
  -F "file=@./sample.pdf"   # => 401
```

期待結果: 正常系はファイル情報（id, name, size, mimeType, downloadUrl 等）を含む 201 レスポンス。SC-001 準拠（5MB ファイルで 5 秒以内）を体感確認する。

## 6. User Story 2: 一覧・検索（Acceptance Scenario 1〜5 の確認）

```bash
curl -s "http://localhost:8082/api/v1/files?page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" | jq '.total, (.files | length)'

curl -s "http://localhost:8082/api/v1/files?search=請求書" \
  -H "Authorization: Bearer $TOKEN" | jq '.files[].name'

curl -s "http://localhost:8082/api/v1/files?tagIds=<TAG_UUID>" \
  -H "Authorization: Bearer $TOKEN" | jq '.total'

curl -s "http://localhost:8082/api/v1/files?search=存在しないキーワード" \
  -H "Authorization: Bearer $TOKEN" | jq '.total'   # => 0, files: []
```

期待結果: SC-002/SC-003（100 件登録時 500ms/1s 以内）を負荷ツール（例: `hurl --test` や `hyperfine`）で確認することを推奨。

## 7. User Story 3: 詳細・ダウンロード（Acceptance Scenario 1〜3 の確認）

```bash
FILE_ID=$(curl -s "http://localhost:8082/api/v1/files?limit=1" -H "Authorization: Bearer $TOKEN" | jq -r '.files[0].id')

curl -s "http://localhost:8082/api/v1/files/$FILE_ID" -H "Authorization: Bearer $TOKEN" | jq

curl -s -o /tmp/downloaded.pdf -D - \
  "http://localhost:8082/api/v1/files/$FILE_ID/content" \
  -H "Authorization: Bearer $TOKEN"
diff ./sample.pdf /tmp/downloaded.pdf   # バイナリ一致を確認

curl -s -o /dev/null -w '%{http_code}\n' \
  "http://localhost:8082/api/v1/files/00000000-0000-0000-0000-000000000000" \
  -H "Authorization: Bearer $TOKEN"   # => 404 FILE_NOT_FOUND
```

## 8. 自動テストの実行（Constitution IV 準拠の確認）

```bash
cd services/files
make test-unit          # usecase / handler の単体テスト（go.uber.org/mock）
make test-integration    # testcontainers-go による repo 層の統合テスト（Docker必須）
```

## 9. API テスト（Schemathesis + Hurl）

サービスを起動した状態で:

```bash
schemathesis run ../../schema/files/openapi.yaml \
  --base-url http://localhost:8082/api/v1 \
  --header "Authorization: Bearer $TOKEN"

hurl --test files.hurl   # アップロード→一覧→詳細→ダウンロードの一連シナリオ（P2以降で拡充）
```

期待結果: すべて成功。失敗した場合は OpenAPI 契約と実装の乖離（SC-005）を疑う。

## 完了の判定基準

- [ ] User Story 1〜3 の Acceptance Scenario がすべて手動確認で成功
- [ ] `make test-unit` / `make test-integration` が通過
- [ ] Schemathesis がエラー無く完走
- [ ] `docker compose up` で auth + files + postgres が揃って起動する
