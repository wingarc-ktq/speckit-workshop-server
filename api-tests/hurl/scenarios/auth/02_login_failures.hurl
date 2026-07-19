# シナリオ: ログインの異常系 (自己完結)
# hurl の --test は既定でファイルを並列実行するため、01 の登録には依存できない.
# このファイル自身が専用ユーザー hurl-02-{{run_id}}@example.com を登録してから検証する.
# メールの 02 プレフィックスはこのファイル由来であることを示す (DB に残っても追跡できる).
# run_id は Makefile が実行ごとにランダム生成するので、毎回新規ユーザーになる.

# ===== 準備: このテスト専用ユーザーを登録 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-02-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl ログイン異常系"
}
HTTP 201

# ===== 正常ログイン（前提の確認: このユーザー・正パスワードなら通る）=====
# 先に成功を確認しておくことで、後続の 401 が「ユーザー不在」ではなく
# 「パスワード誤り」に起因することを切り分けられる.
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-02-{{run_id}}@example.com",
  "password": "P@ssw0rd!"
}
HTTP 200
[Asserts]
jsonpath "$.accessToken" matches "^eyJ"

# ===== 誤パスワード（正しいユーザー・誤ったパスワード → 401）=====
# 未登録メールと同じ AUTH_FAILED を返し、原因（不在か誤パスか）を区別しない.
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-02-{{run_id}}@example.com",
  "password": "WrongPassword!"
}
HTTP 401
[Asserts]
jsonpath "$.message" exists
jsonpath "$.code" == "AUTH_FAILED"

# ===== 存在しないユーザー（401 / 誤パスワードと同じ AUTH_FAILED）=====
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "ghost@example.com",
  "password": "whatever"
}
HTTP 401
[Asserts]
jsonpath "$.message" exists
jsonpath "$.code" == "AUTH_FAILED"

# ===== 空ボディ =====
POST {{auth_url}}/auth/login
Content-Type: application/json
{}
HTTP 400

# ===== JWT なしで /auth/me =====
GET {{auth_url}}/auth/me
HTTP 401
