# シナリオ: ユーザー登録 → ログイン → 自身の情報取得
# メールの 01 プレフィックスはこのファイル由来であることを示す (DB に残っても追跡できる).
# run_id は Makefile が実行ごとにランダム生成するので、毎回新規ユーザーになる.

# ===== 1. 登録 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-01-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl テスト"
}
HTTP 201
[Asserts]
jsonpath "$.user.email" == "hurl-01-{{run_id}}@example.com"
jsonpath "$.user.name" == "Hurl テスト"
jsonpath "$.user.id" matches "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
jsonpath "$.user.createdAt" exists

# ===== 1-b. 重複登録は 409 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-01-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl テスト"
}
HTTP 409
[Asserts]
jsonpath "$.code" == "EMAIL_ALREADY_TAKEN"

# ===== 2. ログイン =====
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-01-{{run_id}}@example.com",
  "password": "P@ssw0rd!"
}
HTTP 200
[Captures]
access_token: jsonpath "$.accessToken"
[Asserts]
jsonpath "$.tokenType" == "Bearer"
jsonpath "$.expiresIn" >= 60
# JWT は base64url(ヘッダJSON) で始まり、ヘッダは必ず {" から始まるため先頭は必ず eyJ になる。
# 値自体（署名・iat/exp）は毎回変わり予測できないので、形式だけを検証する。
jsonpath "$.accessToken" matches "^eyJ"

# ===== 3. 自身の情報取得 (JWT 必須) =====
GET {{auth_url}}/auth/me
Authorization: Bearer {{access_token}}
HTTP 200
[Asserts]
jsonpath "$.user.email" == "hurl-01-{{run_id}}@example.com"

# ===== 3-b. JWT 未指定では 401 =====
GET {{auth_url}}/auth/me
HTTP 401
[Asserts]
jsonpath "$.code" == "UNAUTHORIZED"
