# speckit-workshop-server Development Guidelines

Auto-generated from feature plans. Last updated: 2026-05-07

## Active Technologies

- Go 1.26 (services/auth)
- echo/v4 + oapi-codegen v2 + echo-middleware
- jackc/pgx/v5 + sqlc + golang-migrate
- golang-jwt/v5
- 標準 testing + uber/mock + testcontainers-go
- Schemathesis (Python) + Hurl

## Project Structure

```text
schema/        # OpenAPI 仕様 (SoT)
services/      # 各マイクロサービス (auth, files)
api-tests/     # Schemathesis + Hurl
specs/         # spec-kit のフィーチャー仕様
```

## Commands

`make gen && make test-unit && make test-integration`

## Code Style

Go: gofmt + go vet。Effective Go 準拠。`any` 禁止。

## Recent Changes

- 001-user-auth: JWT 認証サービス
- 002-document-management: 文書管理サービス

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
