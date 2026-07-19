# speckit-workshop-server — Development Guidelines

Canonical reference for all agents and developers.

## Stack

| Category | Technology                                                  |
| -------- | ----------------------------------------------------------- |
| Language | Go 1.26                                                     |
| HTTP     | echo/v4                                                     |
| API spec | OpenAPI 3.0 — oapi-codegen/v2, kin-openapi                  |
| Auth     | golang-jwt/v5 (RS256 JWT)                                   |
| DB       | pgx/v5, sqlc, golang-migrate/v4                             |
| Testing  | Standard `testing` pkg, go.uber.org/mock, testcontainers-go |

## Project Layout

```
services/<name>/
  cmd/server/        # Entry point (thin shim)
  internal/
    domain/          # Business rules, port interfaces, sentinel errors — no external deps
    infra/           # Port implementations: DB repos, JWT, bcrypt
    usecase/         # Orchestrates domain + infra
    handler/         # HTTP ↔ domain mapping (implements oapi-codegen ServerInterface)
    server/          # Wires everything (DI composition root)
  api/gen/           # Generated OpenAPI types — do not edit
  migrations/        # SQL migrations

packages/<name>/     # Cross-service shared libraries (NOT services)
                     # e.g. authjwt — JWT verification + echo auth middleware
```

## Constraints

| Rule                            | Detail                                                                      |
| ------------------------------- | --------------------------------------------------------------------------- |
| **No testify**                  | Use `if err != nil { t.Fatal(err) }` and `if got != want { t.Errorf(...) }` |
| **No cross-service imports**    | A service must not import another service's `internal/`. Shared code goes in `packages/<name>/` (a standalone module); consumers reference it via `require` + a relative `replace` (see `services/auth/go.mod`). |
| **Generated code is read-only** | Never hand-edit `api/gen/` or `infra/repo/db/`                              |

## Required Endpoints

Every service must expose `/healthz` (liveness) and `/readyz` (readiness).

## Error Handling

```go
// domain: define sentinel errors
var ErrUserNotFound = errors.New("user not found")

// wrap with context when propagating
fmt.Errorf("createUser: %w", err)

// unwrap at call sites
errors.Is(err, domain.ErrUserNotFound)
```

## Commands

Run from `services/<name>/` unless stated otherwise:

```bash
make gen                         # Regenerate OpenAPI types, SQL, mocks
make test-unit                   # Unit tests
make test-integration            # Integration tests (requires Docker)
go test ./...                    # Direct alternative to make test-unit
go test -tags integration ./...  # Direct alternative to make test-integration
```

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Comment only the _why_ — not what the code does
- No abstractions for single use; prefer explicit over concise

## Agent Behavior

- Verify changes work before reporting completion
- Create new commits; never amend unless explicitly asked
- Test locally before suggesting async code changes

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
<!-- SPECKIT END -->
