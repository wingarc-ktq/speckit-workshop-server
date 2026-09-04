# シナリオ: ファイル詳細取得とダウンロード
# 登録したファイルの詳細情報を取得し、
# 元のファイルをダウンロードできることを確認する.
# 存在しないファイルIDを指定した場合の404も確認する.
# 最後に作成したファイルを削除する.

# ===== 準備1: シナリオ専用ユーザー登録 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-files-04-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl Files Download"
}
HTTP 201

# ===== 準備2: ログインして JWT を取得 =====
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-files-04-{{run_id}}@example.com",
  "password": "P@ssw0rd!"
}
HTTP 200
[Captures]
access_token: jsonpath "$.accessToken"

# ===== 準備3: ダウンロード確認用の議事録を登録 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/download-{{run_id}}-議事録.txt; text/plain
description: 開発定例 議事録

HTTP 201
[Captures]
file_id: jsonpath "$.file.id"

# ===== 1. 登録したファイルの詳細を取得 =====
GET {{files_url}}/files/{{file_id}}
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
jsonpath "$.file.id" == "{{file_id}}"
jsonpath "$.file.name" == "download-{{run_id}}-議事録.txt"
jsonpath "$.file.description" == "開発定例 議事録"
jsonpath "$.file.downloadUrl" exists

# ===== 2. 登録したファイルをダウンロード =====
GET {{files_url}}/files/{{file_id}}/download
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/octet-stream"
header "Content-Disposition" contains "download-{{run_id}}-議事録.txt"
body contains "開発定例の議事録"

# ===== 3. 存在しないファイルの詳細取得は404 =====
GET {{files_url}}/files/99999999-9999-4999-8999-999999999999
Authorization: Bearer {{access_token}}

HTTP 404
[Asserts]
jsonpath "$.code" == "FILE_NOT_FOUND"

# ===== 後処理 =====
DELETE {{files_url}}/files/{{file_id}}
Authorization: Bearer {{access_token}}

HTTP 204