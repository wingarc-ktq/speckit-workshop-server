# Research: ユーザー認証

**Feature**: 001-user-auth | **Date**: 2026-06-06

## 概要

Technical Context に NEEDS CLARIFICATION はなかった。本 research.md では、実装にあたって事前に確認すべき技術的判断とベストプラクティスを整理する。

---

## 1. handler 層の実装パターン（oapi-codegen strict server）

### 調査対象

oapi-codegen v2 の `strict-server: true` 設定により `StrictServerInterface` が生成されている。handler 層でこれを実装するか、通常の `ServerInterface`（echo.Context ベース）を実装するかの判断。

### Decision: `ServerInterface`（echo.Context ベース）を実装する

### Rationale

- `main.go` のコメントアウト済みコードが `gen.RegisterHandlersWithBaseURL(e, authHandler, "/api/v1")` を使用しており、`ServerInterface` の直接実装を想定している
- `ServerInterface` は echo.Context を直接受け取るため、JWT ミドルウェアからの Context 値取得（ユーザー ID 抽出）が自然にできる
- リファレンス実装として handler 層の責務（リクエストバインド → usecase 呼び出し → レスポンス整形）を明示的に見せられる

### Alternatives considered

- `StrictServerInterface` 実装: リクエストの自動バインドが得られるが、echo.Context にアクセスしづらく JWT ミドルウェアとの連携が不自然になる

---

## 2. JWT 検証ミドルウェアの設計

### 調査対象

`GET /auth/me` は `security: [bearerAuth: []]` が設定されている。JWT 検証をどの層でどう行うか。

### Decision: echo ミドルウェアとして JWT 検証を実装し、handler 層で Context からユーザー ID を取得する

### Rationale

- Constitution VII「JWT 検証は echo ミドルウェアとして実装」に準拠
- oapi-codegen の `RegisterHandlersWithOptions` で per-operation middleware（`OperationMiddlewares`）を使い、`getCurrentUser` にのみ JWT 検証ミドルウェアを適用できる
- ミドルウェアで検証した `sub` クレーム（user_id）を `echo.Context.Set()` で格納し、handler で `echo.Context.Get()` により取得する
- ログイン・登録エンドポイントは認証不要なので、ルート全体にミドルウェアを適用するのではなく、per-operation で適用する

### Alternatives considered

- handler 内で直接 JWT を検証: ミドルウェアの再利用性が低下し、Constitution VII に反する
- echo の JWT ミドルウェアパッケージ（`echo-jwt`）使用: 依存追加となるが、自前実装の方がワークショップの学習効果が高く、Claims 構造のカスタマイズも容易

---

## 3. UserRepository の infrastructure 実装

### 調査対象

sqlc 生成コード（`db.Queries`）をラップして `domain.UserRepository` インターフェースを実装する方法。

### Decision: `persistence.UserRepository` 構造体で `db.Queries` をラップし、`pgtype` ↔ `domain` 型の変換を行う

### Rationale

- sqlc が生成する `db.User` は `pgtype.UUID` / `pgtype.Timestamptz` を使うが、domain 層は `uuid.UUID` / `time.Time` を使う
- infrastructure 層でこの変換を吸収することで、domain / usecase 層は PostgreSQL 固有の型に依存しない
- `main.go` のコメントが `persistence.NewUserRepository(pool)` を呼び出しており、`*pgxpool.Pool` を受け取るコンストラクタを想定
- `db.New(pool)` で `*db.Queries` を生成し、それを内部に保持する構成

### Alternatives considered

- domain 層で `pgtype` を直接使う: 層分離（Constitution III）に違反
- repository パターンを使わず usecase から直接 sqlc を呼ぶ: Constitution III「domain のインターフェースに依存」に違反

---

## 4. メールアドレスの正規化

### 調査対象

spec の Edge Cases に「メールアドレスは大文字小文字を区別しない（小文字に正規化して保存）」とある。正規化をどの層で行うか。

### Decision: usecase 層の `Register` メソッドで `strings.ToLower()` を適用してから repository に渡す

### Rationale

- メールアドレスの正規化はビジネスルールであり、usecase 層の責務
- SQL クエリ側（`GetUserByEmail`）は `LOWER(email) = LOWER($1)` で検索するため、DB 側でも case-insensitive な検索が保証される
- 登録時に小文字で保存しておくことで、DB インデックスの効率も維持できる

### Alternatives considered

- handler 層で正規化: handler はリクエスト変換のみの責務（Constitution III）に限定すべき
- domain 層の Value Object で正規化: 妥当だが、ワークショップの複雑さを考慮し usecase で直接行う

---

## 5. エラーレスポンスの統一形式

### 調査対象

FR-011「エラーレスポンスは統一された形式（メッセージとエラーコードを含む）で返す」の実現方法。

### Decision: OpenAPI の `ErrorResponse` スキーマ（`message` + `code`）を使い、handler 層でドメインエラーを HTTP レスポンスにマッピングする

### Rationale

- `ErrorResponse` は oapi-codegen で `gen.ErrorResponse` として生成済み
- 各操作のエラーレスポンス型（`RegisterUser400JSONResponse`, `LoginUser401JSONResponse` 等）は `ErrorResponse` のエイリアスとして生成されている
- handler 層でドメインエラー（`domain.ErrEmailAlreadyTaken` 等）を判定し、適切な HTTP ステータスコードとエラーコードにマッピングする

### エラーコード一覧

| ドメインエラー | HTTP Status | エラーコード |
|---|---|---|
| バリデーションエラー（パスワード短すぎ等） | 400 | `VALIDATION_ERROR` |
| `domain.ErrEmailAlreadyTaken` | 409 | `EMAIL_ALREADY_TAKEN` |
| `domain.ErrInvalidCredential` | 401 | `AUTH_FAILED` |
| JWT 未指定 / 無効 / 期限切れ | 401 | `UNAUTHORIZED` |

### Alternatives considered

- echo のエラーハンドラーでグローバルにマッピング: OpenAPI 生成型との整合性が取りにくい
- ドメイン層にエラーコードを持たせる: HTTP 固有の概念が domain に入り、層分離に反する

---

## 6. パスワードバリデーションの実装箇所

### 調査対象

FR-004「パスワードが 8 文字未満または 128 文字超の場合、バリデーションエラーを返す」の実装箇所。

### Decision: OpenAPI スキーマの `minLength: 8` / `maxLength: 128` による自動検証 + handler 層での補完

### Rationale

- `RegisterRequest.password` に `minLength: 8` / `maxLength: 128` が定義済み
- oapi-codegen + kin-openapi のリクエスト検証ミドルウェアを使えば、OpenAPI スキーマに基づく自動バリデーションが可能
- ただしリファレンス実装では検証ミドルウェアの導入は任意であり、handler 層で明示的にバリデーションを行うことも可能
- ワークショップの教材としては handler 層で明示的に検証する方が学習効果が高い

### Alternatives considered

- usecase 層でバリデーション: 入力値検証は handler の責務（リクエスト検証）であり usecase に置くべきでない
- kin-openapi ミドルウェアのみに委ねる: ミドルウェアの設定が追加で必要になり、エラーメッセージのカスタマイズが難しい
