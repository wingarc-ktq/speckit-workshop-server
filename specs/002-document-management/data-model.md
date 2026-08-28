# データモデル: Filesサービス

`spec.md` と `openapi.yaml` から抽出したエンティティとデータ構造を定義します。

## 1. エンティティ

### File (ファイル)

システムで管理されるファイルを表す中心的なエンティティ。

| フィールド名 | 型 | 説明 | 制約 |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | 一意な識別子 | Primary Key |
| `name` | `string` | ファイル名 | `maxLength: 255`, Not Null |
| `size` | `int64` | ファイルサイズ (バイト) | Not Null, `> 0` |
| `mime_type` | `string` | MIMEタイプ | Not Null |
| `description` | `string` | ファイルの説明文 | `maxLength: 500`, Nullable |
| `uploaded_at` | `timestamp` | アップロード日時 | Not Null |

### Tag (タグ)

ファイルを分類するためのタグ。**MVPスコープ外**ですが、将来の拡張のために定義します。

| フィールド名 | 型 | 説明 | 制約 |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | 一意な識別子 | Primary Key |
| `name` | `string` | タグ名 | `maxLength: 50`, Not Null, Unique |
| `color` | `string` | タグの色 | Not Null, Enum制約あり |
| `created_at` | `timestamp` | 作成日時 | Not Null |
| `updated_at` | `timestamp` | 更新日時 | Not Null |

## 2. リレーションシップ

- **File と Tag**: 多対多 (Many-to-Many)
  - 1つのファイルは複数のタグを持つことができる。
  - 1つのタグは複数のファイルに付けられる。
  - これを実現するために、中間テーブル `file_tags` (`file_id`, `tag_id`) が必要になる（P2機能で実装）。

## 3. 状態遷移

- **File**:
  - `(存在しない)` -> `作成` -> `存在する`
  - `存在する` -> `更新` -> `存在する` (メタデータのみ)
  - `存在する` -> `削除` -> `(存在しない)`

- **Tag**:
  - `(存在しない)` -> `作成` -> `存在する`
  - `存在する` -> `更新` -> `存在する`
  - `存在する` -> `削除` -> `(存在しない)`
