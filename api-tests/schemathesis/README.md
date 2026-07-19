# Schemathesis API テスト

OpenAPI スキーマからプロパティベースのテストを自動生成し、サーバーがスキーマに準拠しているかを検証します。

## 何を検証するか

Schemathesis は OpenAPI に書かれた制約から無数のテストケースを自動生成します:

- **Status code conformance**: スキーマに記載のないステータスコードを返さないか
- **Response schema conformance**: レスポンスボディがスキーマに準拠しているか
- **Content type conformance**: `Content-Type` ヘッダがスキーマと一致するか
- **Header conformance**: 必須ヘッダが揃っているか
- **境界値・異常系テスト**: minLength / maxLength / format などをファジングで検証

つまり「OpenAPI が嘘をついていない」ことを自動検証してくれます。

## セットアップ

```bash
# Python 3.11+ が必要
make install
```

`make install` はローカルに Python 仮想環境 `.venv/` を作り、そこへ Schemathesis を入れます
(Homebrew などの外部管理 Python は PEP 668 で直接 `pip install` できないため)。
`.venv/` はコミット対象外です。以降の `make run-*` はこの `.venv` の Schemathesis を使います。

## 実行

```bash
# サービスを起動
cd ../.. && make up

# Auth サービスのテスト
make run-auth
```

> Files サービスは未実装のため対象外。auth を参考に作成後、`FILES_URL` / `run-files` を追加する。

## レポートの確認

Schemathesis のレポートは 3 段階で確認できます。

### 1. コンソール出力（一次情報）

`make run-auth` の実行ログそのものが最も詳しいレポートです。各失敗には
「何が期待され、何が返ったか」と **再現用の `curl` コマンド**が付きます。

### 2. JUnit XML（CI / エディタ向け）

```bash
ls report/auth/
# junit-<timestamp>.xml が生成される
```

`--report junit --report-dir report/auth` で出力されます。構造化されており、
CI にそのまま食わせられるほか、エディタで開けば各 `<failure>` に再現 `curl` が入っています。

### 3. HTML ダッシュボード（任意）

見やすい HTML で見たい場合は Allure 形式で出力します（`allure` CLI が別途必要: `brew install allure`）。

```bash
.venv/bin/schemathesis run --url http://localhost:8081/api/v1 \
  --checks all --report allure --report-dir report/auth \
  ../../schema/auth/openapi.yaml
allure serve report/auth
```

> 失敗の再現は、出力に含まれる `curl` をそのまま実行するのが確実です
> (`st replay` は想定外例外のクラッシュ時のみ使えます)。

## CI 連携

現状 CI 用の workflow は未整備です。`make up`（サービス起動）→ `make run-auth`
の順に実行すれば、そのまま CI ジョブ化できます
(`report/**/junit-*.xml` をアーティファクト/テストレポートとして収集すると見やすい)。

## 参考リンク

- [Schemathesis 公式ドキュメント](https://schemathesis.readthedocs.io/)
- [Property-Based Testing の概要](https://schemathesis.readthedocs.io/en/stable/explanation.html)
