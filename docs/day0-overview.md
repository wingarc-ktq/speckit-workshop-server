# Day 0: Spec Kit Workshop (Server) 全体像

## 🎯 この資料について

3日間でサーバーサイドの開発体験を **Spec Kit ワークフロー** で体験するためのガイドです。
Day 1〜3の実習に入る前に、まず本資料で全体の流れと各ツールの役割を把握してください。

**読了目安**: 10分

**この資料でわかること**

- なぜ「OpenAPI ファースト + コード生成」なのか
- 3日間で扱う技術スタックの全体像
- Spec Kit のスキルコマンドの役割と使い方
- 自然言語の要求から動く API までの流れ

---

## 📖 Spec Kit とは

[Spec Kit](https://github.com/github/spec-kit) は、AI 支援による **仕様駆動開発** ワークフローシステムです。

「コードを書く」ことに特化した AI 支援ではなく、**仕様の作成 → 設計 → タスク分解 → 実装 → テスト** という開発ライフサイクル全体をオーケストレーションします。

### なぜ Spec Kit が向いているか（サーバー編）

サーバーの API とデータは「他人と長く共有する」ため、壊れたときの影響が大きいです:

- **APIの安定性**: 使う相手（他サービス・フロント・外部）を把握しきれない。壊すと本番で一斉に動かなくなる
- **ロジックの変更容易性**: 業務要件は頻繁に変わる。テストが無いと変更のたびにデータ破壊や課金ミスのリスクを負う
- **サービス間の一貫性**: 別々に作ると仕様がズレ、結合時に初めて壊れる。原因もサービス跨ぎで追いにくい

Spec Kit は spec.md → OpenAPI → コード → テスト の流れを構造化し、OpenAPI ただ一つを「正」と決めて実装・クライアント・テストをそこに合わせることで、これらを担保します。

---

## 🏗 プロジェクト憲法（Constitution）

このリポジトリには [.specify/memory/constitution.md](../.specify/memory/constitution.md) として **守るべき原則** があります。

**主要原則**

| #   | 原則                   | 概要                                                                              |
| --- | ---------------------- | --------------------------------------------------------------------------------- |
| I   | Go Idiomatic Code      | gofmt / go vet パス、Effective Go 準拠                                            |
| II  | OpenAPI ファースト     | 実装前に必ず `schema/<service>/openapi.yaml`                                      |
| III | クリーンアーキテクチャ | handler / usecase / domain / infrastructure の層分離                              |
| IV  | テスト駆動             | 単体 (標準 testing + uber/mock) / 統合 (testcontainers) / API (Schemathesis+Hurl) |
| V   | 型安全な SQL (sqlc)    | SQL は手書き、Go バインドは生成                                                   |
| VI  | マイクロサービス境界   | 各サービス独立 module、共有コード禁止                                             |
| VII | 認証は JWT (Bearer)    | RS256 (非対称鍵)、Auth が秘密鍵で署名・発行、各サービスは公開鍵で検証             |

---

## 🛠 主要な Spec Kit コマンド

ワークショップで使用する **4つのコマンド** を中心に説明します。

### 1️⃣ `/speckit.specify` - 仕様書作成

自然言語の機能説明から構造化された `specs/[番号-機能名]/spec.md` を生成します。

このワークショップでは `specs/002-document-management/spec.md` が用意済みなので、このコマンドは使いません。

### 2️⃣ `/speckit.plan` - 実装計画作成

spec.md と OpenAPI から、技術的な実装計画 (`plan.md`, `research.md`, `data-model.md`) を生成します。

**Day 1 の終盤で使用します。**

```
/speckit.plan

以下を元に実装プランを作成してください。

仕様書: specs/002-document-management/spec.md
OpenAPI: schema/files/openapi.yaml
技術スタック: Go 1.26 / echo / oapi-codegen / pgx / sqlc / testcontainers-go
```

### 3️⃣ `/speckit.tasks` - タスク分解

実装計画を、実行可能な具体的タスク (`tasks.md`) に分解します。

**Day 2 の冒頭で使用します。**

### 4️⃣ `/speckit.implement` - コード生成と実装

タスクリストを実行して、実際の Go コードを生成します。

**Day 2 のメインで使用します。**

---

## 🔄 ワークショップ全体の流れ

```
spec.md (用意済み)
    ↓
[Day 1] OpenAPI 設計
    ↓
schema/files/openapi.yaml ─┐
                           │
                           ├─→ make gen ─→ ハンドラ I/F・型・SQL アクセスコード
                           │
[Day 2] /speckit.tasks → /speckit.implement
                           ↓
                services/files/ 内のコード + 単体テスト + 統合テスト
                           ↓
[Day 3] make api-test
                           ↓
                Schemathesis レポート + Hurl シナリオ結果
```

---

## 🛠 技術スタックの全体像

| レイヤ           | 採用技術                                  | 役割                                                      |
| ---------------- | ----------------------------------------- | --------------------------------------------------------- |
| 言語             | **Go 1.26**                               | サーバーサイド実装                                        |
| Web FW           | **echo/v4**                               | HTTP サーバー、ミドルウェア                               |
| OpenAPI 生成     | **oapi-codegen v2** + **echo-middleware** | ServerInterface・型・リクエスト検証ミドルウェアの自動生成 |
| OpenAPI 検証     | **kin-openapi**                           | ランタイムでスキーマ検証                                  |
| DB               | **PostgreSQL 17** + **pgx/v5**            | データストア                                              |
| SQL コード生成   | **sqlc**                                  | SQL → 型安全な Go コード                                  |
| マイグレーション | **golang-migrate**                        | スキーマバージョン管理                                    |
| 認証             | **golang-jwt/v5**                         | JWT (RS256 / 非対称鍵) 発行・検証                         |
| 単体テスト       | **標準 testing** + **uber/mock**          | usecase 層のテスト（testify は使わない）                  |
| 統合テスト       | **testcontainers-go**                     | リポジトリ層を本物の Postgres でテスト                    |
| API テスト       | **Schemathesis** + **Hurl**               | OpenAPI 準拠 + シナリオ                                   |

---

## 📂 リポジトリ構成

```
speckit-workshop-server/
├── docs/                          # ワークショップ資料 (本資料含む)
├── specs/
│   ├── 001-user-auth/             # リファレンス例 (実装済み)
│   └── 002-document-management/   # 受講者の作業対象
├── schema/
│   ├── auth/openapi.yaml          # Auth サービス (リファレンス)
│   └── files/openapi.yaml         # Files サービス (Day 1 で作成 / 現状は未作成)
├── services/
│   ├── auth/                      # リファレンス実装
│   └── files/                     # Day 2 の実装対象
├── packages/
│   └── authjwt/                   # JWT 検証 + echo 認証ミドルウェア (サービス間共有)
├── api-tests/
│   ├── schemathesis/              # OpenAPI 駆動の自動テスト
│   └── hurl/                      # シナリオベースのテスト
├── migrations/init/               # DB 初期化 (auth / files データベースを作成)
├── keys/                          # 開発用 RS256 鍵 (make keys で生成、*.pem は git 管理外)
├── compose.yaml                   # postgres + auth + files
├── go.work                        # Go workspace (services/* + packages/*)
└── Makefile                       # 上位の make ターゲット
```

---

## 📅 3日間のスケジュール

### Day 1: OpenAPI 設計フェーズ

**ゴール**: spec.md を読み解き、`schema/files/openapi.yaml` を設計する。

1. spec.md からエンドポイントと型を洗い出す
2. OpenAPI 3.x で書く（kin-openapi で検証）
3. `/speckit.plan` で実装計画を生成

**成果物**: `schema/files/openapi.yaml` + `specs/002-document-management/plan.md`

詳細: [day1-design.md](day1-design.md)

### Day 2: 実装フェーズ

**ゴール**: OpenAPI から Go 実装と単体テストを書き、サービスを起動する。

1. `make gen` で oapi-codegen + sqlc を実行
2. `/speckit.tasks` でタスク分解
3. `/speckit.implement` で domain / usecase / handler / infrastructure を実装
4. 標準 testing + uber/mock (gomock) で単体テスト
5. testcontainers-go で統合テスト

**成果物**: 動作する Files サービス + テストスイート

詳細: [day2-implementation.md](day2-implementation.md)

### Day 3: API テストフェーズ

**ゴール**: 起動したサービスに対して API テストで品質を担保する。

1. Schemathesis で OpenAPI 準拠を網羅的に検証
2. Hurl でユーザーシナリオを記述
3. CI で自動化する設計

**成果物**: API テストスイート + テストレポート

詳細: [day3-testing.md](day3-testing.md)

---

## 🚀 環境準備チェックリスト

> **⚠️ Windows の方へ**: 本ワークショップの手順は `make` と Unix シェル (`command -v` / `uuidgen` / `mkdir -p` など) を前提としています。ネイティブの cmd / PowerShell では動かないため、**WSL2** または **Git Bash** など Unix 互換シェル上で実行してください。以降のコマンドはすべてそのシェルで動かす想定です。

開始前に以下が揃っているか確認してください。

- [ ] Go 1.26 以上 (`go version`)
- [ ] Docker / Docker Compose
- [ ] Python 3.11 以上 (Day 3 用)
- [ ] [Hurl](https://hurl.dev/docs/installation.html) (Day 3 用)
- [ ] 下記のいずれかのAIツールを利用可能
  - GitHub Copilot
  - Claude Code
  - Gemini Cli (Antigravity Cli)
- [ ] このリポジトリをクローン済み
- [ ] `cp .env.sample .env` 済み
- [ ] `make tools` で oapi-codegen, sqlc, migrate をインストール済み

### 動作確認

```bash
# 開発用 RS256 鍵を生成 (未生成なら / make up も内部で呼ぶ)
make keys

# 全サービスを起動 (postgres + auth)
# Auth の生成コード (api/gen, infra/repo/db, usecase/mock) はコミット済みなので make gen は不要
make up
make logs

# Auth のヘルスチェック (liveness / readiness。運用エンドポイントで /api/v1 の外)
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

---

## 💡 学習のための原則

### 1. OpenAPI を最初に書く

実装に飛びつかない。型・エンドポイント・エラーレスポンスを `openapi.yaml` で先に定義する。

### 2. 生成コードは触らない

`api/gen/server.gen.go` や `internal/infra/repo/db/*.sql.go` は手動編集禁止。
変更は SoT (`openapi.yaml` または `*.sql`) を編集 → `make gen`。

### 3. 層を越えて依存しない

`handler` は `usecase` を呼ぶ。`domain` は誰にも依存しない。`infrastructure` は `domain` の interface を実装する。

### 4. テストはピラミッド

単体テスト > 統合テスト > API テスト

### 5. AI 支援、人間主導

`/speckit.implement` は強力ですが、生成されたコードは必ずレビュー。Constitution 違反は修正。

---

## ➡️ 次のステップ

[Day 1: OpenAPI 設計](day1-design.md) に進んでください。

**Day 1 の予習**

- `specs/002-document-management/spec.md` に目を通す
- `schema/auth/openapi.yaml` を眺めて OpenAPI 3.x の書き方を思い出す
