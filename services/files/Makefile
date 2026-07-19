.PHONY: gen gen-oapi gen-sqlc gen-mocks build run test test-unit test-integration migrate-up migrate-down clean

SCHEMA_DIR := ../../schema/files
DATABASE_URL ?= postgres://workshop:workshop@localhost:5432/files?sslmode=disable

# ===== Code Generation =====
gen: gen-oapi gen-sqlc gen-mocks ## OpenAPI + SQL + mock を生成

gen-oapi: ## oapi-codegen で server interface 生成
	mkdir -p api/gen
	@if [ ! -f "$(SCHEMA_DIR)/openapi.yaml" ]; then \
		echo "❌ $(SCHEMA_DIR)/openapi.yaml がありません。Day 1 で作成してから再実行してください。"; \
		exit 1; \
	fi
	oapi-codegen -config oapi-codegen.yaml $(SCHEMA_DIR)/openapi.yaml

gen-sqlc:
	sqlc generate

gen-mocks:
	go generate ./...

# ===== Build / Run =====
build:
	go build -o bin/files ./cmd/server

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

clean:
	rm -rf bin/ api/gen/server.gen.go internal/infra/repo/db
