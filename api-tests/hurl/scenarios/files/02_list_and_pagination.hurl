# シナリオ: ファイル一覧取得とページネーション
# 3件のファイルを作成し、limit=2 で
# 1ページ目2件 / 2ページ目1件になることを確認する.
# 最後に作成したファイルをすべて削除する.

# ===== 準備1: シナリオ専用ユーザー登録 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-files-02-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl Files List"
}
HTTP 201

# ===== 準備2: ログインして JWT を取得 =====
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-files-02-{{run_id}}@example.com",
  "password": "P@ssw0rd!"
}
HTTP 200
[Captures]
access_token: jsonpath "$.accessToken"

# ===== 準備3: ファイル1を作成 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/list-{{run_id}}-1.txt; text/plain
description: pagination file 1

HTTP 201
[Captures]
file_id_1: jsonpath "$.file.id"

# ===== 準備4: ファイル2を作成 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/list-{{run_id}}-2.txt; text/plain
description: pagination file 2

HTTP 201
[Captures]
file_id_2: jsonpath "$.file.id"

# ===== 準備5: ファイル3を作成 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/list-{{run_id}}-3.txt; text/plain
description: pagination file 3

HTTP 201
[Captures]
file_id_3: jsonpath "$.file.id"

# ===== 1. 1ページ目を2件ずつ取得 =====
GET {{files_url}}/files?page=1&limit=2&q=list-{{run_id}}
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
jsonpath "$.files" count == 2
jsonpath "$.total" == 3
jsonpath "$.page" == 1
jsonpath "$.limit" == 2

# ===== 2. 2ページ目を取得 =====
GET {{files_url}}/files?page=2&limit=2&q=list-{{run_id}}
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
jsonpath "$.files" count == 1
jsonpath "$.total" == 3
jsonpath "$.page" == 2
jsonpath "$.limit" == 2

# ===== 後処理 =====
DELETE {{files_url}}/files/{{file_id_1}}
Authorization: Bearer {{access_token}}

HTTP 204

DELETE {{files_url}}/files/{{file_id_2}}
Authorization: Bearer {{access_token}}

HTTP 204

DELETE {{files_url}}/files/{{file_id_3}}
Authorization: Bearer {{access_token}}

HTTP 204