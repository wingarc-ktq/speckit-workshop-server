# JWT 署名鍵（開発用）

⚠️ **このディレクトリの鍵は開発・ワークショップ専用です。本番では絶対に使用しないでください。**

Auth サービスは JWT を **RS256（非対称鍵）** で署名します。

| ファイル | 用途 | 渡すサービス |
|---|---|---|
| `jwt_dev_private.pem` | 署名（トークン発行） | Auth のみ（`AUTH_JWT_PRIVATE_KEY_PATH`） |
| `jwt_dev_public.pem` | 検証（トークン検証） | Auth / Files（`*_JWT_PUBLIC_KEY_PATH`） |

非対称鍵にすることで、**秘密鍵を持つのは Auth サービスだけ**になり、Files は公開鍵で検証のみ行えます（秘密鍵を共有しない）。

## 鍵の生成

鍵ファイル (`*.pem`) は **git 管理しません**（`.gitignore` 済み）。クローン後にローカルで生成してください。
`make up` は内部で `make keys` を呼ぶため、未生成なら自動生成されます。

```sh
make keys   # リポジトリ直下で実行 (既存ならスキップ)
```

手動で生成する場合:

```sh
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out keys/jwt_dev_private.pem
openssl rsa -in keys/jwt_dev_private.pem -pubout -out keys/jwt_dev_public.pem
```
