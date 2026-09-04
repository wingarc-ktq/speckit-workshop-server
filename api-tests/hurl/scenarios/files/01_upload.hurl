# シナリオ: ファイルアップロード
# このファイル単体で認証ユーザーを準備し、
# アップロード成功 / サイズ超過 / 未認証を検証する.
# 作成したファイルは最後に削除し、Files サービスに状態を残さない.

# ===== 準備1: このシナリオ専用ユーザーを登録 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-files-01-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl Files Upload"
}
HTTP 201

# ===== 準備2: ログインして JWT を取得 =====
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-files-01-{{run_id}}@example.com",
  "password": "P@ssw0rd!"
}
HTTP 200
[Captures]
access_token: jsonpath "$.accessToken"
[Asserts]
jsonpath "$.accessToken" matches "^eyJ"

# ===== 1. 正常なファイルアップロード =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/upload.txt; text/plain
description: Hurl upload scenario

HTTP 201
[Captures]
file_id: jsonpath "$.file.id"
[Asserts]
jsonpath "$.file.id" exists
jsonpath "$.file.name" == "upload.txt"
jsonpath "$.file.size" > 0
jsonpath "$.file.mimeType" exists
jsonpath "$.file.downloadUrl" exists
jsonpath "$.file.uploadedAt" exists

# ===== 2. 10MB を超えるファイルは 413 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/too-large.bin; application/octet-stream
description: too large file

HTTP 413
[Asserts]
jsonpath "$.code" == "FILE_TOO_LARGE"

# ===== 3. JWT 未指定では 401 =====
POST {{files_url}}/files
[Multipart]
file: file,fixtures/upload.txt; text/plain
description: unauthorized upload

HTTP 401

# ===== 後処理: 正常系で作成したファイルを削除 =====
DELETE {{files_url}}/files/{{file_id}}
Authorization: Bearer {{access_token}}

HTTP 204