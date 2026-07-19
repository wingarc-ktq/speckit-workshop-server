# Hurl シナリオテスト

[Hurl](https://hurl.dev/) はプレーンテキストファイルで HTTP リクエストの並びを記述できる API テストツールです。

## 何を検証するか

Schemathesis が「OpenAPI 準拠」を網羅的に検証するのに対し、Hurl は **ユーザーシナリオ** を検証します:

- 「ユーザー登録 → ログイン → 自身の情報取得」のような状態遷移
- 取得したレスポンス値（JWT など）を後続リクエストで使う
- ビジネスロジックの正しさ（例: 誤ったパスワードで 401 が返るか）

## セットアップ

```bash
# Mac
brew install hurl

# Linux/Windows: https://hurl.dev/docs/installation.html
```

## 実行

```bash
# サービスを起動
cd ../.. && make up

# シナリオ実行 (auth)
make run
```

> Files サービスは未実装のため対象外。auth を参考に作成後、`scenarios/files/` と `run-files` を追加する。

## ディレクトリ構成

```
api-tests/hurl/
├── scenarios/
│   └── auth/
│       ├── 01_register_and_login.hurl
│       └── 02_login_failures.hurl
└── report/           # HTML レポート出力先 (gitignore)
```

## レポートの確認

```bash
make run
open report/auth/index.html
```

## .hurl の書き方 (要点)

```hurl
# リクエスト行
POST {{base_url}}/auth/login
Content-Type: application/json
{ "email": "...", "password": "..." }

# 期待するステータス
HTTP 200

# レスポンスから値を取り出して変数に保存
[Captures]
token: jsonpath "$.accessToken"

# レスポンスのアサーション
[Asserts]
jsonpath "$.tokenType" == "Bearer"
jsonpath "$.expiresIn" >= 60
```

詳しくは [Hurl 公式ドキュメント](https://hurl.dev/docs/manual.html) を参照。
