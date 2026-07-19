# Day 3: API テスト

## 🎯 今日のゴール

Day 2 で実装した Files サービスに対して、**Schemathesis** と **Hurl** を使った API テストを書き、CI 連携まで設計する。

**成果物**

- Schemathesis レポート（OpenAPI 準拠の自動検証結果）
- Hurl シナリオテスト（ユーザー操作の流れ）
- GitHub Actions ワークフロー（任意）

---

## ⏰ タイムテーブル（6時間想定）

| 時間        | 内容                                   |
| ----------- | -------------------------------------- |
| 10:30-11:00 | Day 2 振り返り・サービス起動確認       |
| 11:00-11:30 | 1️⃣ なぜ 2 種類のテストツールを使うのか |
| 11:30-12:00 | 2️⃣ Schemathesis セットアップ・初回実行 |
| 12:00-13:00 | 休憩                                   |
| 13:00-14:00 | 2️⃣ Schemathesis レポート読解・失敗修正 |
| 14:00-16:00 | 3️⃣ Hurl でユーザーシナリオを書く       |
| 16:00-16:30 | 4️⃣ CI 連携（GitHub Actions）           |
| 16:30-17:30 | 5️⃣ 3 日間の総括                        |

---

## 📋 事前準備チェックリスト

- [ ] Day 2 の実装が動く（`make up` でサービス起動可能）
- [ ] Python 3.11+ がインストール済み
- [ ] [Hurl](https://hurl.dev/docs/installation.html) がインストール済み (`brew install hurl`)

---

## 1️⃣ なぜ 2 種類のテストツールを使うのか（30分）

| ツール           | 役割                     | 何を保証するか                             |
| ---------------- | ------------------------ | ------------------------------------------ |
| **Schemathesis** | OpenAPI 駆動の自動テスト | サーバーが OpenAPI に **嘘をついていない** |
| **Hurl**         | シナリオベースのテスト   | **ユーザー操作の流れ** が期待通り動く      |

両方を組み合わせることで「型レベルの正しさ」と「ビジネスロジックの正しさ」を別々に担保できます。
これは単一ツール（Karate, Postman など）でやるより責務分離が明確で、フィードバックも速いです。

### Schemathesis vs Karate vs Postman

|                    | Schemathesis               | Karate         | Postman/Newman |
| ------------------ | -------------------------- | -------------- | -------------- |
| 言語               | Python                     | Java + Gherkin | JS             |
| OpenAPI ネイティブ | ◎ プロパティベース自動生成 | △ 手書き       | △ 手書き       |
| 異常系の網羅       | ◎ ファジング               | △ 書いた分のみ | △ 書いた分のみ |
| シナリオ記述       | △                          | ◎ Gherkin      | ○ コレクション |
| CI 連携            | ◎                          | ◎              | ○              |

→ 本ワークショップは **Schemathesis (網羅) + Hurl (シナリオ)** で両者の良いとこ取りをします。

---

## 2️⃣ Schemathesis でスキーマ準拠を検証（1.5時間）

### 2.1 セットアップ

```bash
cd api-tests/schemathesis
make install   # .venv を作り schemathesis を隔離インストール (PEP 668 回避)
```

### 2.2 サービスを起動

```bash
cd ../..
make up
```

### 2.3 Auth サービスをテスト

```bash
cd api-tests/schemathesis
make run-auth
```

> `make run-auth` は `get-token`（テストユーザーの登録＋ログインを兼ねる）で取得したトークンで
> `/auth/me` の認証成功パスも検証します。トークンを取得できない場合も未認証パスの検証は続行します。

Schemathesis は OpenAPI のすべての endpoint × 制約からテストケースを自動生成し、以下を検証します:

- ✅ ステータスコードが OpenAPI に書かれているもののみ
- ✅ レスポンスボディが schema に準拠
- ✅ Content-Type が一致
- ✅ 必須プロパティが揃っている
- ✅ minLength/maxLength/format/pattern などの制約に違反するリクエストへの挙動

### 2.4 Files サービスをテスト

Files 用の `run-files` ターゲットは **まだありません**。README のとおり、`run-auth` を雛形に自分で追加します
（`FILES_URL` / `FILES_SCHEMA` を定義し、URL とスキーマパスを files 向けに差し替え。Files は全エンドポイントで
JWT が必須なので、トークンは常に付与する）:

```makefile
# api-tests/schemathesis/Makefile に追記（.PHONY にも run-files を足す）
FILES_URL    ?= http://localhost:8082/api/v1
FILES_SCHEMA := ../../schema/files/openapi.yaml

run-files: $(ST) ## Files サービスを Schemathesis で検証（要 JWT）
	mkdir -p $(REPORT_DIR)
	@# AUTH_TOKEN 未設定なら get-token を実行（get-token は register-user 依存なので登録も走る）
	@TOKEN="$${AUTH_TOKEN:-$$($(MAKE) -s get-token)}"; \
	$(ST) run \
	  --url $(FILES_URL) \
	  --checks all \
	  --max-examples 30 \
	  --header "Authorization: Bearer $$TOKEN" \
	  --report junit --report-dir $(REPORT_DIR)/files \
	  $(FILES_SCHEMA)
```

追加できたら、あとは実行するだけです。`run-files` は `AUTH_TOKEN` が未設定なら内部で
`get-token`（= `register-user` によるテストユーザー登録 ＋ ログイン）を自動実行するので、
事前のトークン取得は不要です:

```bash
# これだけで OK（内部で 登録 → ログイン → トークン付与 → 検証 まで走る）
make run-files
```

> 取得済みのトークンを使い回したい場合は `export AUTH_TOKEN=<token>` で明示的に渡せます
> （設定時は `get-token` は呼ばれません）。

### 2.5 レポートを読む

一次情報は **コンソール出力** です。各失敗に「期待 vs 実際」と**再現用の `curl`** が付きます。
加えて `--report junit --report-dir report/auth`（Makefile が指定済み）で
`report/auth/` `report/files/` に JUnit XML が出力され、CI にそのまま食わせられます。

失敗があれば以下のような出力になります（Schemathesis v4 系のフォーマット）:

```
______________________________ POST /files ______________________________
1. Test Case ID: 8BN5Pa

- Undocumented HTTP status code

    Received: 500
    Documented: 201, 400, 401, 413

[500] Internal Server Error:

    `{"code":"INTERNAL","message":"unexpected error"}`

Reproduce with:

    curl -X POST http://localhost:8082/api/v1/files

    st replay 8BN5Pa
```

→ サーバーが OpenAPI に書いていない 500 を返してしまっている、というバグ検出。
失敗ごとに、見出し（チェック名）・`Received` / `Documented`（実際 vs 期待）・レスポンスボディ・
**再現用の `curl`**・`st replay <Test Case ID>`（そのケースだけ再実行）が並びます。
実行の最後には `SUMMARY` で失敗種別ごとの件数と、`Missing authentication` /
`Schema validation mismatch` などの警告（`⚠️`）がまとまって表示されます。

### 2.6 失敗を直す

失敗したら以下のいずれかが原因:

1. **OpenAPI が現実と合っていない**: `schema/files/openapi.yaml` を修正
2. **サーバーがバグっている**: `services/files/internal/...` を修正
3. **Schemathesis がノイズを出している**: `--exclude-checks` で抑制（最終手段）

---

## 3️⃣ Hurl でユーザーシナリオを書く（2時間）

### 3.1 既存シナリオの確認

`api-tests/hurl/scenarios/` に Auth のテンプレートがあります。これを雛形に files 用のシナリオを書きます:

- `auth/01_register_and_login.hurl` - 登録 → ログイン → /me
- `auth/02_login_failures.hurl` - ログイン異常系

files 用（`files/*.hurl`）はまだ無いので、これから作成します（3.3 参照）。

### 3.2 実行

```bash
cd api-tests/hurl
make run
```

HTML レポートが `report/` に生成されます。

### 3.3 シナリオを追加する

例: 検索シナリオを追加

```hurl
# scenarios/files/02_search.hurl

# ログイン
POST {{auth_url}}/auth/login
Content-Type: application/json
{ "email": "taro@example.com", "password": "P@ssw0rd!" }
HTTP 200
[Captures]
token: jsonpath "$.accessToken"

# 「請求書」というキーワードで検索
GET {{files_url}}/files?search=請求書
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.files[*].name" includes "請求書"
```

### 3.4 シナリオ設計のコツ

- **状態を残さない**: シナリオの最後で作成したリソースを削除
- **冪等にする**: 同じシナリオを 2 回流しても結果が変わらないように
- **アサートは欲張りすぎない**: 1 シナリオで検証する範囲を絞る
- **Captures を活用**: あるレスポンスから取り出した値を後続で使う

### 3.5 マスト・カバーすべきシナリオ

P1 機能ごとに 1 シナリオずつ:

- [ ] Story 1: アップロード成功 / ファイルサイズ超過 (413) / 未認証 (401)
- [ ] Story 2: 一覧取得 / ページネーション / ソート
- [ ] Story 3: 検索ヒット / 検索結果 0 件

---

## 4️⃣ CI 連携（30分）

CI はテストピラミッドに合わせて 2 つのワークフローに分けています。

| ワークフロー | 中身 | トリガー |
| ------------ | ---- | -------- |
| [`go-tests.yml`](../.github/workflows/go-tests.yml) | `lint` / `unit`（毎 PR）、`integration`（手動のみ） | `pull_request` + `workflow_dispatch` |
| [`api-tests.yml`](../.github/workflows/api-tests.yml) | Schemathesis + Hurl（compose スタック全体） | `workflow_dispatch`（手動のみ） |

速い単体テストと lint で毎 PR に即フィードバックを返し、Docker を使う重いテストは手動に寄せています。
統合テストは `go-tests.yml` の手動ジョブ、API テスト（compose 起動 + ファジング）は最重量なので
`api-tests.yml` ごと手動実行のみ、という切り分けです。以下は API テスト側の中身です。

Auth サービスを例に、Schemathesis と Hurl を実行するワークフローを
[`.github/workflows/api-tests.yml`](../.github/workflows/api-tests.yml) として **すでにコミット済み** です
（重いので手動実行のみ。起動方法は後述の「手動実行」を参照）。
Files 用のシナリオを追加したら、同じ手順で `run-files` / files シナリオを steps に足していきます。

```yaml
name: API Tests

# 重い E2E（compose 起動 + ファジング）なので手動実行のみ。Actions タブ / gh CLI から起動する。
on:
  workflow_dispatch:

# 同一ブランチで新しい push があれば、実行中のジョブをキャンセルする。
concurrency:
  group: api-tests-${{ github.ref }}
  cancel-in-progress: true

jobs:
  api-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-python@v5
        with:
          python-version: "3.11"

      - name: Install Hurl
        run: |
          # 最新リリースの .deb を取得し、依存解決のため apt 経由で入れる (jq は runner に同梱)。
          tag=$(curl -sSL https://api.github.com/repos/Orange-OpenSource/hurl/releases/latest | jq -r .tag_name)
          curl -sSLO "https://github.com/Orange-OpenSource/hurl/releases/download/${tag}/hurl_${tag}_amd64.deb"
          sudo apt-get install -y "./hurl_${tag}_amd64.deb"

      - name: Start services
        run: |
          cp .env.sample .env
          make up   # make keys で RS256 鍵を生成し、postgres + auth を起動する
          # readiness 待ち (運用エンドポイントは /api/v1 の外)。
          for i in {1..30}; do
            curl -fs http://localhost:8081/readyz && break
            sleep 2
          done
          curl -fs http://localhost:8081/readyz   # ループが空振りしたらここで step を失敗させる

      - name: Schemathesis (auth)
        run: |
          cd api-tests/schemathesis
          make install    # .venv に隔離インストール
          make run-auth    # 内部で register-user → get-token → 検証まで走る

      - name: Hurl (auth)
        run: cd api-tests/hurl && make run

      - name: Dump auth logs on failure
        if: failure()
        run: docker compose logs auth

      - name: Upload reports
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: api-test-reports
          path: |
            api-tests/schemathesis/report/
            api-tests/hurl/report/
```

ポイント:

- **Go のセットアップは不要**: サービスは `make up`（`docker compose up --build`）でビルド・起動するため、
  ホスト側に Go は要りません。
- **`make run-auth` だけでトークン取得まで完結**: 内部で `register-user`（テストユーザー登録）→
  `get-token`（ログイン）→ `/auth/me` の認証成功パス検証まで自動で走ります。
- **失敗時のログ回収**: `Dump auth logs on failure` と `Upload reports`（`if: always()`）で、
  落ちた原因を Actions のログとアーティファクトから追えます。

### 手動実行

`workflow_dispatch` を宣言してあるので、PR を作らなくても手動で実行できます。

- **GitHub UI**: リポジトリの **Actions → API Tests → Run workflow** から任意のブランチで起動。
- **gh CLI**:

  ```bash
  gh workflow run api-tests.yml --ref <branch>   # 実行を開始
  gh run watch                                   # 進行中の run をフォロー
  gh run list --workflow api-tests.yml           # 過去 run 一覧
  ```

---

## 5️⃣ 3日間の総括（1時間）

### 5.1 発表（5分/人）

**発表内容**

1. **作ったもの**: OpenAPI / Go 実装 / API テスト
2. **工夫したポイント**: spec.md → OpenAPI の翻訳、testcontainers での統合テスト戦略 など
3. **Spec Kit ワークフローの感想**: 良かった点・引っかかった点
4. **AI ツールの使い所**: どこで活躍したか / 任せきれなかったか

### 5.2 振り返り議論

- どのフェーズが一番時間かかったか？
- どのフェーズが AI に任せやすかったか？
- 自分の業務でどう適用できそうか？

---

## 📝 Day 3 振り返りチェックリスト

- [ ] Schemathesis を実行してレポートを読めた
- [ ] Schemathesis の失敗を 1 件以上潰した
- [ ] Hurl で MVP の主要シナリオを書けた
- [ ] Hurl の異常系シナリオを 1 つ以上書けた
- [ ] CI で API テストを自動化する設計が描けた

---

## 🔗 参考リンク

- [Schemathesis 公式](https://schemathesis.readthedocs.io/)
- [Hurl 公式](https://hurl.dev/)
- [Property-Based Testing 入門](https://hypothesis.works/articles/what-is-property-based-testing/)
- [テストピラミッド](https://martinfowler.com/articles/practical-test-pyramid.html)

---

## 🎓 おつかれさまでした

3日間で **OpenAPI ファースト → コード生成 → 多層テスト** のサイクルを体験しました。
ぜひ自分のプロジェクトに持ち帰って活用してください。
