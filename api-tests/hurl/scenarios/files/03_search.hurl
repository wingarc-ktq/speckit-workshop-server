# シナリオ: ファイル検索
# 検索対象となる2件のファイルを登録し、
# ファイル名の部分一致検索と検索結果0件を確認する.
# 最後に作成したファイルを削除する.

# ===== 準備1: シナリオ専用ユーザー登録 =====
POST {{auth_url}}/auth/register
Content-Type: application/json
{
  "email": "hurl-files-03-{{run_id}}@example.com",
  "password": "P@ssw0rd!",
  "name": "Hurl Files Search"
}
HTTP 201

# ===== 準備2: ログインして JWT を取得 =====
POST {{auth_url}}/auth/login
Content-Type: application/json
{
  "email": "hurl-files-03-{{run_id}}@example.com",
  "password": "P@ssw0rd!"
}
HTTP 200
[Captures]
access_token: jsonpath "$.accessToken"

# ===== 準備3: 検索対象ファイルを作成 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/search-{{run_id}}-invoice.txt; text/plain
description: search target invoice

HTTP 201
[Captures]
invoice_file_id: jsonpath "$.file.id"

# ===== 準備4: 検索対象外のファイルを作成 =====
POST {{files_url}}/files
Authorization: Bearer {{access_token}}
[Multipart]
file: file,fixtures/search-{{run_id}}-meeting.txt; text/plain
description: search target meeting

HTTP 201
[Captures]
meeting_file_id: jsonpath "$.file.id"

# ===== 1. ファイル名の部分一致検索 =====
GET {{files_url}}/files?q=invoice
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
jsonpath "$.files" count >= 1

# ===== 2. run_id を含めて今回のファイルだけ検索 =====
GET {{files_url}}/files?q={{run_id}}-invoice
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
jsonpath "$.files" count == 1
jsonpath "$.total" == 1
jsonpath "$.files[0].name" == "search-{{run_id}}-invoice.txt"

# ===== 3. 検索結果が0件 =====
GET {{files_url}}/files?q=no-match-{{run_id}}
Authorization: Bearer {{access_token}}

HTTP 200
[Asserts]
jsonpath "$.files" count == 0
jsonpath "$.total" == 0

# ===== 後処理 =====
DELETE {{files_url}}/files/{{invoice_file_id}}
Authorization: Bearer {{access_token}}

HTTP 204

DELETE {{files_url}}/files/{{meeting_file_id}}
Authorization: Bearer {{access_token}}

HTTP 204