# シナリオ: ファイル情報の修正
# 登録したファイルの名前と説明文を変更できることを確認する.
# 更新後に詳細取得を行い、変更内容が保存されていることも確認する.
# 最後に作成したファイルを削除する.

# ===== 準備1: シナリオ専用ユーザー登録 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-files-05-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl Files Update"
}
HTTP 201

# ===== 準備2: ログインして JWT を取得 =====
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-files-05-{{run_id}}@example.com",
  "password": "P@ssw0rd!"
}
HTTP 200
[Captures]
access_token: jsonpath "$.accessToken"

# ===== 準備3: 修正対象の会議メモを登録 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/update-{{run_id}}-会議メモ.txt; text/plain
description: 開発定例のメモ

HTTP 201
[Captures]
file_id: jsonpath "$.file.id"

# ===== 1. ファイル名と説明文を修正 =====
PATCH {{files_url}}/files/{{file_id}}
Authorization: Bearer {{access_token}}
Content-Type: application/json
{
  "name": "開発定例-議事録-{{run_id}}.txt",
  "description": "2026年9月 開発定例の正式な議事録"
}

HTTP 200
[Asserts]
jsonpath "$.file.id" == "{{file_id}}"
jsonpath "$.file.name" == "開発定例-議事録-{{run_id}}.txt"
jsonpath "$.file.description" == "2026年9月 開発定例の正式な議事録"

# ===== 2. 詳細取得でも修正内容が反映されている =====
GET {{files_url}}/files/{{file_id}}
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
jsonpath "$.file.id" == "{{file_id}}"
jsonpath "$.file.name" == "開発定例-議事録-{{run_id}}.txt"
jsonpath "$.file.description" == "2026年9月 開発定例の正式な議事録"

# ===== 後処理 =====
DELETE {{files_url}}/files/{{file_id}}
Authorization: Bearer {{access_token}}

HTTP 204