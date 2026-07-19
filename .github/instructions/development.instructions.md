---
applyTo: '**'
---

# 開発ガイドライン（統合版）

詳細なガイドラインは以下のファイルに分割されています:

- `project.instructions.md` - プロジェクト概要とアーキテクチャ
- `go.instructions.md` - Go コーディング規約
- `testing.instructions.md` - テストガイドライン

## 🛠 主要コマンド

```bash
# コード生成 (OpenAPI / SQL → Go)
make gen
make gen-oapi
make gen-sqlc
make gen-mocks

# サーバー起動
make up                           # 全サービス (postgres + auth + files)
make up-db                        # PostgreSQL のみ
cd services/auth && make run      # サービスをローカル実行

# テスト
make test                         # 単体 + 統合
make test-unit                    # 単体のみ
make test-integration             # testcontainers (要 Docker)
make api-test                     # Schemathesis + Hurl

# コード品質
go fmt ./...
go vet ./...
```

## 🔌 MCP (Model Context Protocol) ガイドライン

このプロジェクトの MCP 設定は `.vscode/mcp.json` (VSCode/Copilot) と `.mcp.json` (Claude Code) にあります。

ワークショップ初期状態では MCP サーバーは登録されていません。
必要に応じて以下のような MCP サーバーを追加できます:

### 想定されうる MCP サーバー

| サーバー | 用途 |
|---|---|
| GitHub MCP | issue/PR の参照・操作 |
| Postgres MCP | DB スキーマや実データの確認 |
| Filesystem MCP | プロジェクト全体の俯瞰 |

設定方法は [Model Context Protocol 公式](https://modelcontextprotocol.io/) を参照。

## 🤖 spec-kit コマンドの使い分け

| コマンド | 使うフェーズ | 主な目的 |
|---|---|---|
| `/speckit.specify` | 仕様起票時 | 自然言語 → spec.md |
| `/speckit.plan` | Day 1 終盤 | spec.md + OpenAPI → plan.md |
| `/speckit.tasks` | Day 2 冒頭 | plan.md → tasks.md |
| `/speckit.implement` | Day 2 メイン | tasks.md → Go コード |
| `/speckit.analyze` | 整合性チェック | 3 アーティファクトの不整合検出 |

## ⚠ 生成コードの取り扱い

以下のファイルは **手動編集禁止** です。SoT (`openapi.yaml` または `*.sql`) を編集して `make gen` で再生成してください。

- `services/*/api/gen/server.gen.go` - oapi-codegen 出力
- `services/*/internal/infra/repo/db/*.sql.go` - sqlc 出力
- `services/*/internal/domain/mock/*_mock.go` - gomock 出力
