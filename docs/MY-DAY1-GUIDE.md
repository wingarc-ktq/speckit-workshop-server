# Day 1 作業ガイド（自分用）

**今日のゴール**: `schema/files/openapi.yaml` を書く。それだけ。Go のコードは 1 行も書かない。

- [1. フォルダ構成](#1-フォルダ構成--どこに何があるか)
- [2. 今日の流れ](#2-今日の流れ)
- [3. すでに決まっていること](#3-すでに決まっていること自分で決めない部分)
- [4. MVP の範囲（資料の誤りに注意）](#4-mvp-の範囲資料の誤りに注意)
- [5. 【作業】エンドポイント表](#5-作業エンドポイント表)
- [6. 【作業】FR カバレッジ表](#6-作業fr-カバレッジ表)
- [7. 【作業】スキーマ設計](#7-作業スキーマ設計)
- [8. 【作業】自分で決めること](#8-作業自分で決めること)
- [9. 書く順番と検証](#9-書く順番と検証)
- [10. 夕方の /speckit.plan](#10-夕方の-speckitplan)

---

## 1. フォルダ構成 — どこに何があるか

### 今日さわるもの

| パス | 何か | 状態 |
| --- | --- | --- |
| `schema/files/openapi.yaml` | **今日の成果物** | ⬜ これから作る |
| `docs/MY-DAY1-GUIDE.md` | このファイル（作業メモ兼用） | ✅ |
| `specs/002-document-management/plan.md` | `/speckit.plan` の出力 | ⬜ 夕方に生成 |

### 今日読むもの（お手本・材料）

| パス | 何か |
| --- | --- |
| `specs/002-document-management/spec.md` | **今日の課題**。何を作るかが書いてある |
| `schema/auth/openapi.yaml` | **OpenAPI の書き方のお手本**。`ErrorResponse` の形もここ |
| `.specify/memory/constitution.md` | プロジェクト憲法。守るべきルール |
| `docs/day1-design.md` | 今日の進行 |

### 今日はさわらないもの（Day 2 以降）

| パス | 何か |
| --- | --- |
| `services/files/` | Files サービスの実装置き場。**今は `Makefile` と `oapi-codegen.yaml` だけ**。Go コードは Day 2 |
| `services/auth/` | 参照実装。Day 2 で構造を真似る |
| `migrations/init/` | DB 初期化。`auth` と `files` の DB は**作成済み** |
| `packages/authjwt/` | JWT 検証ミドルウェア（**実装済み・共有**） |
| `api-tests/` | Day 3 の API テスト |
| `compose.yaml` | Files サービスは**まだ未登録**（コメントに「auth を参考に作成後、ここに追加する」） |
| `keys/` | 開発用 RS256 鍵。`make keys` で生成済み。git 管理外 |

### Day 2 で作ることになる構造（参照実装 `services/auth/` の実際の中身）

憲法 III のクリーンアーキテクチャが、こう具現化されています。

```
services/auth/
├── cmd/server/main.go          # 薄いシム。server.Run を呼ぶだけ
├── api/gen/server.gen.go       # oapi-codegen の生成物（手動編集禁止）
├── migrations/                 # golang-migrate 形式の DDL
└── internal/
    ├── server/server.go        # DI 組み立て・echo セットアップ・graceful shutdown
    ├── handler/                # HTTP ハンドラ。ServerInterface を実装
    │   ├── auth_handler.go
    │   └── health.go           # /healthz /readyz（OpenAPI の対象外）
    ├── usecase/                # アプリケーションロジック
    │   ├── auth_usecase.go
    │   ├── port.go             # 依存する外部の interface
    │   └── mock/               # gomock の生成物
    ├── domain/user.go          # ドメインモデル。外部依存なし
    ├── config/config.go        # 環境変数
    └── infra/                  # domain の interface を実装する側
        ├── repo/               # PostgreSQL
        │   ├── user_repository.go
        │   ├── queries/        # 手書き SQL
        │   └── db/             # sqlc の生成物（手動編集禁止）
        ├── token/jwt.go        # JWT 発行
        └── password/bcrypt.go
```

**依存の向き**: `handler → usecase → domain ← infra`。`domain` は誰にも依存しない。

---

## 2. 今日の流れ

| 時間 | やること | 成果 |
| --- | --- | --- |
| 〜12:00 | **設計を紙の上で固める** | 5〜8 章の表が埋まる |
| 13:00〜14:30 | **openapi.yaml を書く** | 検証が通る yaml |
| 14:30〜15:00 | 設計レビュー | 講師に説明 |
| 15:00〜16:00 | `/speckit.plan` | plan.md 生成 |
| 16:00〜 | 振り返り | |

**午前はコードを書かない。** 表を埋めて設計を固めることに使う。午後の記述が速くなる。

---

## 3. すでに決まっていること（自分で決めない部分）

repo を読んで判明した制約。ここを外すと Day 3 の API テストで落ちる。

### エラーレスポンスの形

`schema/auth/openapi.yaml` で確定済み。Files も同じ形にする。

```yaml
ErrorResponse:
  type: object
  required: [message, code]
  properties:
    message: { type: string, example: "認証に失敗しました" }
    code:    { type: string, example: "AUTH_FAILED" }
```

`code` は `AUTH_FAILED` のような大文字スネークケース。

### 401 のレスポンス内容

`packages/authjwt/middleware.go` にハードコードされており、**共有ミドルウェアが返す**。

```go
c.JSON(http.StatusUnauthorized, errorBody{
    Code:    "UNAUTHORIZED",
    Message: "認証が必要です",
})
```

OpenAPI に書く 401 は、この実装と食い違わないようにするだけ。

### `/healthz` `/readyz` は OpenAPI に書かない

憲法 II の例外規定。

> 運用エンドポイント（`/healthz`・`/readyz`・`/metrics` 等）はビジネス API ではないため OpenAPI 契約の対象外とする。`/api/v1` 配下に置かず、検証ミドルウェアからも除外する

FR-023（ヘルスチェックは認証なし）は、**OpenAPI ではなく実装側で**満たす。

### 各エンドポイントに必須の記述

憲法 II のルール。`day1-design.md` のチェックリストには載っていないので注意。

> 各エンドポイントには `summary`, `operationId`, **レスポンス例**を必ず記述

`operationId` は生成される Go の関数名になるので、`getFiles` `uploadFile` のようなキャメルケースの動詞句にする。

### サーバー URL

- Auth: `http://localhost:8081/api/v1`
- **Files: `http://localhost:8082/api/v1`**

### DB

`migrations/init/00_create_databases.sql` で `files` データベースは作成済み。テーブルは Day 2 で自分で作る。

---

## 4. MVP の範囲（資料の誤りに注意）

`day1-design.md` の 1.1 の表はフロントエンド版からのコピーで、**このリポジトリの `spec.md` とストーリー番号が食い違っている**。

| | day1-design.md の表 | 実際の spec.md |
| --- | --- | --- |
| Story 2 | 文書一覧の表示 | 一覧取得**と検索** |
| Story 3 | キーワード検索（P1） | **ファイル詳細取得とダウンロード**（P1 🎯） |
| 文書詳細 | P2 と記載 | **P1 MVP** |
| 削除 | P3 と記載 | **P2**（Story 6） |

**`spec.md` が正。** 今日設計するのは以下の 3 つ。

- Story 1: アップロード
- Story 2: 一覧取得 + ページネーション + キーワード検索 + タグフィルタ
- Story 3: 詳細取得 + **ダウンロード**

---

## 5. 【作業】エンドポイント表

`spec.md` の User Story 1〜3 を HTTP に置き換える。

| # | メソッド | パス | 認証 | リクエスト | 成功 | 失敗 | ストーリー |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | POST | `/files` | 要 | multipart/form-data (file, description) | 201 | 400 / 401 / 413 | Story 1 |
| 2 | GET | `/files` | 要 | query: page, limit, search, tagIds | 200 | 400 / 401 | Story 2 |
| 3 | GET | `/files/{fileId}` | 要 | — | 200 | 401 / 404 | Story 3 |
| 4 | GET | `/files/{fileId}/content` | 要 | — | 200 | 401 / 404 | Story 3（ダウンロード） |

`operationId` はこうする（生成される Go の関数名になる）。

| # | operationId |
| --- | --- |
| 1 | `uploadFile` |
| 2 | `getFiles` |
| 3 | `getFile` |
| 4 | `downloadFileContent` |

---

## 6. 【作業】FR カバレッジ表

各 FR を、どのエンドポイントが満たすか埋める。空欄が残る＝エンドポイントか項目が足りない。

### ファイルアップロード（Story 1）

| FR | 内容 | 担当 |
| --- | --- | --- |
| FR-001 | ファイルと説明文でアップロード | #1 `POST /files`（`multipart/form-data`） |
| FR-002 | file 必須 / description 任意・最大 500 字 | #1 の requestBody: `required: [file]`、description に `maxLength: 500` |
| FR-003 | 最大 10MB、超過時エラー | #1 の `413` レスポンス。サイズ自体は OpenAPI で表現できないので実装側で検査（description に明記） |

### 一覧・検索（Story 2）

| FR | 内容 | 担当 |
| --- | --- | --- |
| FR-004 | ページネーション（既定 1 ページ目・20 件） | #2 の query `page`（`minimum: 1, default: 1`）`limit`（`minimum: 1, maximum: 100, default: 20`） |
| FR-005 | ファイル名の部分一致検索 | #2 の query `search`（`type: string`） |
| FR-006 | タグ ID（複数可）でフィルタ | #2 の query `tagIds`（`type: array`, `explode: true`） |
| FR-007 | 一覧・総件数・ページ番号・件数制限を返す | #2 の `FileListResponse`（`files` / `total` / `page` / `limit`） |

### 詳細・ダウンロード（Story 3）

| FR | 内容 | 担当 |
| --- | --- | --- |
| FR-008 | ファイル ID で詳細を返す | #3 `GET /files/{fileId}` → `FileResponse` |
| FR-009 | ファイル ID で本体を返す | #4 `GET /files/{fileId}/content` → `application/octet-stream`（`format: binary`） |
| FR-010 | 存在しない ID は 404 | #3 と #4 の `404` レスポンス（`ErrorResponse`、code は `FILE_NOT_FOUND`） |

### 横断（全エンドポイント）

| FR | 内容 | 対応方法 |
| --- | --- | --- |
| FR-020 | ヘルスチェック以外は JWT Bearer 認証 | ルート直下に `security: [{ bearerAuth: [] }]` を書き、全エンドポイントに一括適用 |
| FR-021 | JWT 未指定・無効は 401 | 全エンドポイントに `401` レスポンスを記述。中身は共有ミドルウェアが返す（3 章） |
| FR-022 | エラーは統一形式 | `ErrorResponse` を全エンドポイントで使う（3 章） |
| FR-023 | ヘルスチェックは認証なし | **OpenAPI には書かない**（3 章） |

---

## 7. 【作業】スキーマ設計

`paths` より先に `components/schemas` を決める。`$ref` 先が無いと検証が通らないため。

| スキーマ | 用途 | プロパティ |
| --- | --- | --- |
| `ErrorResponse` | 全エラー共通 | `message`, `code`（3 章で確定・両方 required） |
| `FileInfo` | ファイル 1 件の情報 | `id` `name` `size` `mimeType` `description` `uploadedAt` `downloadUrl` `tagIds` |
| `FileResponse` | 詳細・アップロードの返り値 | `file: FileInfo`（required） |
| `FileListResponse` | 一覧の返り値 | `files: [FileInfo]` `total` `page` `limit`（すべて required） |
| `TagInfo` | タグ（P2 だが型は先に） | `id` `name` `color` `createdAt` `updatedAt` |

### `FileInfo` の詳細

| プロパティ | 型 | required | 備考 |
| --- | --- | --- | --- |
| `id` | `string` / `format: uuid` | ✅ | auth の `User.id` に倣う |
| `name` | `string` | ✅ | |
| `size` | `integer` / `format: int64` | ✅ | バイト数 |
| `mimeType` | `string` | ✅ | 例 `application/pdf` |
| `description` | `string` / `maxLength: 500` | — | 任意項目（FR-002） |
| `uploadedAt` | `string` / `format: date-time` | ✅ | |
| `downloadUrl` | `string` | ✅ | 相対パス。判断 5 |
| `tagIds` | `array[string(uuid)]` | ✅ | タグ無しは空配列 |

### `TagInfo.color` は列挙型にする

`spec.md` の Edge Cases に「定義済みの列挙値から選択」とあるため。

```yaml
color:
  type: string
  enum: [blue, red, yellow, green, purple, orange, gray]
```

### なぜ `FileResponse` を `{ file: FileInfo }` にするのか

`schema/auth/openapi.yaml` の `UserResponse` が `{ user: User }` という「1 段包む」形になっているため、それに合わせる。将来レスポンスに項目を足すときに壊れにくい。

```yaml
UserResponse:
  type: object
  required: [user]
  properties:
    user:
      $ref: "#/components/schemas/User"
```

---

## 8. 【作業】自分で決めること

repo に前例がなく、レビューで理由を聞かれる部分。

レビューで理由を聞かれる部分。**納得できないものは変えてよい。**

| # | 論点 | 判断 | 理由 |
| --- | --- | --- | --- |
| 1 | ダウンロードのパス | **`/files/{fileId}/content`** | REST では URL は名詞、動作は HTTP メソッドが担当する。`download` は動詞なので `GET` と役割が重なる。「ファイルの中身」を表す名詞として `content` を選んだ |
| 2 | 検索クエリ名 | **`search`** | `day1-design.md` の例が `GET /files?search=` になっており、資料の想定に合わせた |
| 3 | `tagIds` の渡し方 | **繰り返し**（`?tagIds=a&tagIds=b`） | OpenAPI 標準の `explode: true`。oapi-codegen がそのまま `[]string` を生成するので、実装側でカンマ分割の処理が要らない |
| 4 | サイズ超過 | **413** | `day1-design.md` の例が 413。400 でも間違いではないが、HTTP には「ペイロードが大きすぎる」専用のコードがあるのでそちらが正確 |
| 5 | `downloadUrl` | **相対パス**（`/api/v1/files/{id}/content`） | 絶対 URL だとホスト名が埋め込まれ、開発・ステージング・本番で値が変わる。相対ならどの環境でも同じレスポンスになる |
| 6 | `FileInfo` の共通化 | **共通の 1 つにする** | `spec.md` の Key Entities が `File` 1 つなのでそれに従う。型を分けると oapi-codegen の生成型が増え、フロント側も 2 つの型を扱うことになる。一覧で数項目多く返す実害は小さい |

### 判断 6 の別案

`spec.md` を厳密に読むと、一覧は 5 項目（ID・名前・サイズ・タグ ID・アップロード日時）、詳細は 8 項目（＋ MIME タイプ・説明・ダウンロード URL）と書き分けられている。

そこを重視するなら `FileSummary`（一覧用）と `FileInfo`（詳細用）に分ける設計もある。**転送量を抑えたい・仕様に厳密に従いたい**ならこちら。レビューで「なぜ分けなかったのか」と聞かれたら、上の理由を答えられればよい。

### 判断 3 の別案

カンマ区切り（`?tagIds=a,b`）は URL が短くなる利点がある。ただし OpenAPI では `explode: false` の指定が必要で、実装側でも分割処理が要る。**URL 長が問題になる規模でなければ繰り返しのほうが素直。**

---

## 9. 書く順番と検証

`$ref` の参照先が無いと検証が通らないため、下から積み上げる。

1. `openapi` / `info` / `servers` / `security` / `tags` の骨組み
2. `components/securitySchemes`（`bearerAuth`）
3. `components/schemas` — `ErrorResponse` → `FileInfo` → 各レスポンス型
4. `paths` を 1 本ずつ追加し、**そのつど検証**

```bash
cd services/files
make gen-oapi
```

`api/gen/server.gen.go` が生成されれば構文 OK。エラーが出たら直前に足した 1 本が原因なので、特定しやすい。

> `make gen` は使わない。sqlc と mockgen も動いて、まだ存在しない Go コードでエラーになる。**Day 1 は `make gen-oapi` だけ。**

### 骨組みのひな形

```yaml
openapi: 3.0.3
info:
  title: File Management Service API
  version: 1.0.0
  description: 文書管理サービスのファイル管理 API
servers:
  - url: http://localhost:8082/api/v1
    description: Development server
security:
  - bearerAuth: []
tags:
  - name: files
  - name: tags
paths:
  # /healthz /readyz は書かない（憲法 II の例外規定）
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  schemas:
    ErrorResponse:
      type: object
      required: [message, code]
      properties:
        message: { type: string, example: "ファイルが見つかりません" }
        code:    { type: string, example: "FILE_NOT_FOUND" }
```

---

## 10. 夕方の `/speckit.plan`

`openapi.yaml` が固まってから実行する。

```
/speckit.plan

以下を元に Files サービスの実装プランを作成してください。

仕様書: specs/002-document-management/spec.md
OpenAPI: schema/files/openapi.yaml

技術スタック・アーキテクチャ・テスト方針は憲法（.specify/memory/constitution.md）に従ってください。
MVP 機能（P1）に絞ってください。
```

生成物は `plan.md` / `research.md` / `data-model.md`。確認ポイントは 4 つ。

- [ ] P1 の機能がすべて含まれているか
- [ ] DB スキーマが `spec.md` の Key Entities と整合しているか
- [ ] 層構造が憲法 III（handler / usecase / domain / infra）に沿っているか
- [ ] テスト方針が憲法 IV（標準 testing + uber/mock、testcontainers、Schemathesis + Hurl）に沿っているか

---

## 今日やらないこと

- Go のコードを書く（Day 2）
- `services/files/` に何かを作る（Day 2）
- `compose.yaml` に files を追加する（Day 2）
- DB のテーブルを作る（Day 2）
- Python / Hurl のインストール（Day 3）
