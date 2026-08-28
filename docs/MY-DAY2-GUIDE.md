# Day 2 作業ガイド（自分用）

**今日のゴール**: Day 1 で書いた `openapi.yaml` を元に、Files サービスを Go で実装する。

---

## 0. Day 1 との最大の違い

Day 1 は 1 ファイル書くだけだった。**Day 2 はサービスを丸ごと 0 から作る。**

`services/files/` には現在これしかない。

```
services/files/
├── Makefile              ✅ ある
├── oapi-codegen.yaml     ✅ ある
└── api/gen/server.gen.go ✅ Day 1 で生成済み
```

**足りないものを auth からコピーして作る必要がある。**

| ファイル | auth にある | files | 対応 |
| --- | --- | --- | --- |
| `go.mod` / `go.sum` | ✅ | ❌ | 新規作成（module 名は `.../services/files`） |
| `sqlc.yaml` | ✅ | ❌ | auth からコピーして調整 |
| `Dockerfile` | ✅ | ❌ | auth からコピーして調整 |
| `.env.sample` | ✅ | ❌ | auth からコピーして調整 |
| `cmd/server/main.go` | ✅ | ❌ | auth を参考に作成 |
| `internal/**` | ✅ | ❌ | **今日のメイン作業** |
| `migrations/` | ✅ | ❌ | 新規作成 |

### リポジトリ全体で直すもの（忘れやすい）

| ファイル | 何をする |
| --- | --- |
| `go.work` | `use` に `./services/files` を追加。**これを忘れると import が解決しない** |
| `compose.yaml` | files サービスを追加（コメントに「auth を参考に作成後、ここに追加する」と書いてある） |
| ルートの `Makefile` | `gen` / `test-unit` / `test-integration` が **auth しか呼んでいない**。files も呼ぶように追加 |

現在のルート Makefile:

```make
gen: gen-auth              # ← files が無い
test-unit:
	$(MAKE) -C services/auth test-unit    # ← files が無い
```

---

## 1. 今日の流れ

| 時間 | やること |
| --- | --- |
| 〜11:30 | 骨組みを作る（go.mod / sqlc.yaml / go.work 追加） |
| 11:30〜12:00 | `/speckit.tasks` でタスク分解 |
| 13:00〜16:00 | `/speckit.implement` で実装 |
| 16:00〜16:30 | 起動して curl で動作確認 |
| 16:30〜17:00 | テスト実行 |

---

## 2. 実装の順番（資料 3.2 の推奨順序）

**この順番は守る。** 下から積み上げないと、上の層が書けない。

```
1. migrations/*.sql            DB のテーブルを作る
2. internal/infra/repo/queries/*.sql   SQL を手書き
3. make gen-sqlc               SQL → Go コードを生成
4. internal/domain/            File の型・エラー定義（外部依存なし）
5. internal/usecase/           処理の流れ + 単体テスト
6. internal/infra/repo/        DB アクセス実装 + 統合テスト
7. internal/handler/           HTTP ハンドラ + /healthz /readyz
8. （JWT は書かない）          packages/authjwt を使うだけ
9. internal/server/            DI 配線・echo 組立
   cmd/server/main.go          薄いシム
```

### 依存の向き（憲法 III）

```
handler → usecase → domain ← infra
```

`domain` は誰にも依存しない。`infra` は `domain`（または `usecase`）が定義した interface を実装する。

---

## 3. 禁止事項・注意点

### testify は使わない（憲法 IV）

標準 `testing` + `go.uber.org/mock` のみ。

```go
if err != nil { t.Fatal(err) }
if got != want { t.Errorf("got %v, want %v", got, want) }
```

テーブル駆動（`tests := []struct{...}{...}`）が推奨。

### JWT ミドルウェアは自前で書かない（資料 3.6）

`packages/authjwt` を使う。公開鍵で検証するだけ。

```go
verifier, err := authjwt.NewVerifier(pubPEM)
gen.RegisterHandlersWithOptions(e, fileHandler, gen.RegisterHandlersOptions{
    BaseURL:     basePath,
    Middlewares: []echo.MiddlewareFunc{authjwt.Middleware(verifier)},
})
```

`/healthz` `/readyz` にはミドルウェアを掛けず、`/api/v1` の外に置く。

### 生成コードは手で直さない

- `api/gen/server.gen.go`（oapi-codegen）
- `internal/infra/repo/db/*.sql.go`（sqlc）

直したいときは元（`openapi.yaml` / `*.sql`）を直して再生成。

### 秘密鍵は持たない（憲法 VII）

Files は**公開鍵で検証するだけ**。`FILES_JWT_PUBLIC_KEY_PATH` のみ。

---

## 4. Day 1 の設計を実装に落とすとき

`openapi.yaml` と `data-model.md` で決めたことが、そのまま実装の制約になる。

| Day 1 で決めたこと | Day 2 での扱い |
| --- | --- |
| エンドポイント 4 本 | `gen.ServerInterface` の 4 メソッドを実装 |
| `ErrorResponse` = `{message, code}` | `domain` の sentinel error → `code` への変換関数を作る |
| `code`: `INVALID_PARAMETER` / `FILE_NOT_FOUND` / `FILE_TOO_LARGE` | 上の変換関数で使う |
| 401 は共有ミドルウェアが返す | 自分では書かない |
| `size` は 1〜10,485,760 | ストリーム処理 + `io.LimitReader` で検出（research.md Decision） |
| 並び順は `uploaded_at DESC` 固定 | SQL に直書き。パラメータ化しない |
| `tag_ids UUID[]`（FK なし） | `files` テーブルの配列カラム |
| `name` VARCHAR(255) / `description` VARCHAR(500) | マイグレーションの DDL |

---

## 5. 動作確認（資料 4️⃣）

```bash
make keys        # 未生成なら
make up-db       # postgres だけ起動

cd services/auth  && make migrate-up && make run &
cd services/files && make migrate-up && make run &

curl http://localhost:8082/healthz

# ユーザー登録 → ログイン → トークン取得
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"taro@example.com","password":"P@ssw0rd!","name":"田中太郎"}'

TOKEN=$(curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"taro@example.com","password":"P@ssw0rd!"}' | jq -r '.accessToken')

curl http://localhost:8082/api/v1/files -H "Authorization: Bearer $TOKEN"
```

`jq` が無ければ `sudo apt install -y jq`。

---

## 6. テスト

```bash
cd services/files
make test-unit          # 高速。Docker 不要
make test-integration   # testcontainers。Docker 必須
```

統合テストは実行のたびに PostgreSQL コンテナが立ち上がる。**初回はイメージ取得で時間がかかる。**

---

## 7. 振り返りチェックリスト（資料）

- [ ] `make gen-oapi` でハンドラインターフェースを生成できた
- [ ] `make gen-sqlc` で SQL から Go コードを生成できた
- [ ] domain / usecase / handler / infra の各層を実装できた
- [ ] gomock を使った単体テストが書けた
- [ ] testcontainers-go を使った統合テストが書けた
- [ ] サービスが起動し、curl で MVP 機能が動作した

### レビュー観点

- [ ] MVP の API が curl で叩けて期待通り返る
- [ ] 層を越えた依存がない
- [ ] usecase の単体テストカバレッジ 80% 以上
- [ ] 生成コードに手を入れていない
- [ ] エラーハンドリングが統一されている（`domain.Err*` → HTTP ステータス）

---

## 8. 時間が足りないときの切り方

優先度は User Story の順。

1. **Story 2（一覧取得）** — DB から読むだけ。ストレージが要らないので最初に動く
2. **Story 1（アップロード）** — ストレージ実装が必要
3. **Story 3（詳細・ダウンロード）** — Story 1 と 2 ができていれば早い

Story 2 だけでも「動くサービス」として成立する。全部中途半端より、1 つ完成しているほうがよい。
