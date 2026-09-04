# Files MVP データモデル

## File

ファイル本体と検索可能なメタデータを表す。

| フィールド | 型 | 必須 | 制約 |
|---|---|---:|---|
| id | UUID | ○ | サーバー生成、一意 |
| name | string | ○ | multipart の元ファイル名。空文字不可 |
| size | int64 | ○ | 0 以上、10 MiB 以下 |
| mimeType | string | ○ | multipart の MIME タイプ |
| description | string | - | 最大 500 文字 |
| storageKey | string | ○ | サーバー生成値。ユーザー入力をパスに利用しない |
| uploadedAt | timestamp | ○ | DB 生成 |

`downloadUrl` は保存値ではなく、詳細・アップロードレスポンスで API のダウンロード URL として組み立てる。`tagIds` は FileTag の関連から取得する。

## Tag

P1 ではタグ CRUD を提供しない。既存タグを参照して upload の `tagIds` と list のフィルタに利用する。

| フィールド | 型 | 必須 | 制約 |
|---|---|---:|---|
| id | UUID | ○ | 一意 |
| name | string | ○ | 一意。P2 の管理 API で作成予定 |
| color | enum | ○ | `blue`, `red`, `yellow`, `green`, `purple`, `orange`, `gray` |
| createdAt | timestamp | ○ | DB 生成 |
| updatedAt | timestamp | ○ | DB 生成 |

## FileTag

File と Tag の多対多関連。`(fileId, tagId)` を主キーとし、File または Tag の削除時に cascade する。MVP ではファイル upload 時に指定された tag ID の存在を検証し、関連を同一トランザクションで登録する。

## 状態と整合性

1. 本体を storage に保存する。
2. メタデータと関連を DB トランザクションで登録する。
3. DB 登録が失敗した場合、保存済み本体を補償削除する。
4. 詳細・ダウンロードは File の存在確認後に storage key を使う。

本体とメタデータの完全な分散トランザクションは採用せず、補償削除と明示的なエラーで不整合を最小化する。
