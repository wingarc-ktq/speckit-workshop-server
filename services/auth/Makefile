.PHONY: tools gen gen-oapi gen-sqlc gen-mocks build run test test-unit test-integration migrate-up migrate-down lint fmt clean

SCHEMA_DIR := ../../schema/auth
DATABASE_URL ?= postgres://workshop:workshop@localhost:5432/auth?sslmode=disable

# ===== Tools =====
tools: ## コード生成・マイグレーションに必要な CLI ツールを導入
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install go.uber.org/mock/mockgen@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ===== Code Generation =====
gen: gen-oapi gen-sqlc gen-mocks ## OpenAPI + SQL + mock を生成

gen-oapi: ## oapi-codegen で server interface 生成
	mkdir -p api/gen
	oapi-codegen -config oapi-codegen.yaml $(SCHEMA_DIR)/openapi.yaml

gen-sqlc: ## sqlc で型安全な DB アクセスコード生成
	sqlc generate

gen-mocks: ## gomock でリポジトリのモック生成
	go generate ./...

# ===== Build / Run =====
build:
	go build -o bin/auth ./cmd/server

run:
	go run ./cmd/server

# ===== Tests =====
test: test-unit test-integration

test-unit:
	go test -short -race ./...

test-integration:
	go test -race -tags=integration ./...

# ===== DB Migrations =====
migrate-up:
	migrate -database "$(DATABASE_URL)" -path migrations up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path migrations down 1

# ===== Lint / Format =====
lint: ## gofmt 差分チェック + go vet（Constitution I）
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "❌ gofmt 未適用のファイルがあります" && exit 1)
	go vet ./...

fmt: ## gofmt + goimports で整形
	gofmt -w .
	goimports -w .

clean:
	rm -rf bin/ api/gen/server.gen.go internal/infra/repo/db
