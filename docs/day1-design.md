# Day 1: OpenAPI 設計

## 🎯 今日のゴール

`specs/002-document-management/spec.md` を読み解き、Files サービスの OpenAPI を設計する。

**成果物**

- `schema/files/openapi.yaml`（自分で設計したもの）
- `specs/002-document-management/plan.md`（`/speckit.plan` で生成）

---

## ⏰ タイムテーブル（6時間想定）

| 時間        | 内容                                          |
| ----------- | --------------------------------------------- |
| 10:30-11:00 | オリエンテーション・環境確認                  |
| 11:00-12:00 | 1️⃣ spec.md を読み解く                          |
| 12:00-13:00 | 休憩                                          |
| 13:00-14:30 | 2️⃣ OpenAPI を書く（検証まで含む）              |
| 14:30-15:00 | 3️⃣ 設計レビュー                                |
| 15:00-16:00 | 4️⃣ `/speckit.plan` で plan.md を作成           |
| 16:00-16:30 | 5️⃣ 振り返り                                    |

---

## 📋 事前準備チェックリスト

- [ ] [day0-overview.md](day0-overview.md) を読了
- [ ] リポジトリをクローン済み
- [ ] `make tools` で開発ツールを入れた
- [ ] `cp .env.sample .env` 済み
- [ ] `schema/auth/openapi.yaml` をざっと眺めた

---

## 1️⃣ spec.md を読み解く（1時間）

`specs/002-document-management/spec.md` を開いてユースケースを把握します。

🎯 マークがついているのが MVP（Day 1 の対象）です。

| Priority | User Story | 概要                             |
| -------- | ---------- | -------------------------------- |
| P1 🎯    | Story 1    | 文書のアップロードと基本情報登録 |
| P1 🎯    | Story 2    | 文書一覧の表示と閲覧             |
| P1 🎯    | Story 3    | キーワード検索で文書を探す       |

### 1.1 ユースケース → エンドポイントの抽出

各ストーリーを HTTP エンドポイントに置き換えてみてください（紙でもエディタでも OK）:

| ストーリー             | 候補エンドポイント                  |
| ---------------------- | ----------------------------------- |
| Story 1 (アップロード) | `POST /files` (multipart/form-data) |
| Story 2 (一覧)         | `GET /files?page=&limit=&sort=`     |
| Story 3 (検索)         | `GET /files?search=`                |
| 文書詳細 (P2)          | `GET /files/{fileId}`               |
| 削除 (P3)              | `DELETE /files/{fileId}`            |
| タグ管理 (P2)          | `GET/POST/PUT/DELETE /tags`         |

### 1.2 リソース設計のチェックリスト

設計時に以下を意識してください:

- [ ] **REST 原則**: リソース指向の URL（動詞ではなく名詞）
- [ ] **ステータスコードは正確に**: 201 vs 200, 204 vs 200, 401 vs 403
- [ ] **エラーは統一形式**: `ErrorResponse` を一貫して使う
- [ ] **ページネーション**: `?page=` `?limit=` または cursor-based
- [ ] **認可**: 全ビジネスエンドポイントに `bearerAuth` をかける（`/healthz`・`/readyz` など運用エンドポイントは OpenAPI 仕様の対象外）
- [ ] **冪等性**: PUT, DELETE は冪等
- [ ] **バリデーション**: minLength / maxLength / format / pattern を埋める

---

## 2️⃣ OpenAPI を書く（1.5時間）

`schema/files/openapi.yaml` を **手で書きます**。AI 補完を使うのは構いませんが、内容は自分で理解してください。

### 2.1 ひな形

`schema/auth/openapi.yaml` を参考に、以下のような骨組みから始めます:

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
  # 運用エンドポイント (/healthz・/readyz) は OpenAPI 仕様の対象外なのでここには書かない
  # サーバー側で /api/v1 の外に直接ルーティングする (Auth サービスの実装を参照)
  /files:
    get:
      summary: ファイル一覧取得
      operationId: getFiles
      tags: [files]
      responses:
        "200":
          description: 成功
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/FileListResponse"

  # TODO: /files の post、/files/{fileId}、/tags, ... を埋めていく

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  schemas:
    # TODO: FileInfo, FileListResponse, TagInfo, ErrorResponse, ...
```

### 2.2 設計のヒント

**ファイルアップロードの multipart/form-data**

```yaml
/files:
  post:
    summary: ファイルアップロード
    operationId: uploadFile
    tags: [files]
    requestBody:
      required: true
      content:
        multipart/form-data:
          schema:
            type: object
            required: [file]
            properties:
              file:
                type: string
                format: binary
              description:
                type: string
                maxLength: 500
              tagIds:
                type: array
                items:
                  type: string
    responses:
      "201":
        description: 作成成功
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/FileResponse"
      "413":
        description: ファイルサイズ超過
```

**ページネーションパラメータ**

```yaml
parameters:
  - name: page
    in: query
    schema:
      type: integer
      minimum: 1
      default: 1
  - name: limit
    in: query
    schema:
      type: integer
      minimum: 1
      maximum: 100
      default: 20
```

### 2.3 検証

書き終わったら `oapi-codegen` を走らせてエラーが出ないか確認します。
内部で kin-openapi がパースするため、構文・構造エラーはここで検出できます:

```bash
cd services/files
make gen-oapi
# エラーなく api/gen/server.gen.go が生成されれば OK
```

---

## 3️⃣ 設計レビュー（30分）

書き終わったら、講師に設計を確認してもらいます。以下の観点で自己チェックしたうえで、迷った点や判断の根拠を説明できるようにしておいてください。

**レビューの観点**

- [ ] エンドポイントは網羅できたか
- [ ] ステータスコード設計は適切か（201/204/400/401/404/409/413）
- [ ] スキーマの required / minLength / maxLength は十分か
- [ ] エラーレスポンスは統一されているか
- [ ] `securitySchemes: bearerAuth` を適切に運用できているか

完璧でなくて OK。**自分の設計の意図** が説明できるかが重要です。

---

## 4️⃣ `/speckit.plan` で実装計画作成（1時間）

設計した OpenAPI を元に、`/speckit.plan` で実装計画を生成します。

### 4.1 実行

Claude Code または GitHub Copilot Chat で:

```
/speckit.plan

以下を元に Files サービスの実装プランを作成してください。

仕様書: specs/002-document-management/spec.md
OpenAPI: schema/files/openapi.yaml

技術スタック・アーキテクチャ・テスト方針は憲法（.specify/memory/constitution.md）に従ってください。
MVP 機能（P1）に絞ってください。
```

### 4.2 生成される成果物

- `specs/002-document-management/plan.md` - 実装計画
- `specs/002-document-management/research.md` - 技術選定の根拠
- `specs/002-document-management/data-model.md` - エンティティ定義

### 4.3 plan.md 確認ポイント

- [ ] MVP 機能（P1）がすべて含まれている
- [ ] DB スキーマは spec.md の Key Entities と整合
- [ ] echo + oapi-codegen の wiring 手順が書かれている
- [ ] テスト戦略（単体・統合・API）が明記されている
- [ ] 憲法（Constitution）に違反していない

違和感があれば直接 plan.md を編集して整えます。

---

## 5️⃣ 振り返り（30分）

- 一人 3〜4 分で OpenAPI とプランを画面共有
- 工夫した点・迷った点を共有
- フィードバックは「提案」の形で

---

## 📝 Day 1 振り返りチェックリスト

- [ ] spec.md の MVP 機能を理解できた
- [ ] `schema/files/openapi.yaml` を自分で書けた
- [ ] `make gen-oapi` がエラーなく通った
- [ ] `/speckit.plan` で plan.md を生成できた
- [ ] plan.md が憲法に準拠している

---

## 🔗 参考リンク

- [OpenAPI 3.0 公式仕様](https://swagger.io/specification/v3/)
- [kin-openapi](https://github.com/getkin/kin-openapi)
- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
- [REST API 設計のベストプラクティス](https://learn.microsoft.com/en-us/azure/architecture/best-practices/api-design)

---

## ➡️ 次回予告: Day 2

Day 2 では `/speckit.tasks` でタスク分解 → `/speckit.implement` で Go 実装と単体テストを書いていきます。

**事前準備**

- [ ] `schema/files/openapi.yaml` をコミット
- [ ] `plan.md` を確定
