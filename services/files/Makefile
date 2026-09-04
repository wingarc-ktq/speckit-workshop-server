.PHONY: gen gen-oapi gen-sqlc gen-mocks build run test test-unit test-integration migrate-up migrate-down clean

SCHEMA_DIR := ../../schema/files
DATABASE_URL ?= postgres://workshop:workshop@localhost:5432/files?sslmode=disable
FILES_SERVICE_PORT ?= 8082
FILES_STORAGE_ROOT ?= /tmp/files-storage
FILES_JWT_PUBLIC_KEY_PATH ?= ../../keys/jwt_dev_public.pem
FILES_MAX_UPLOAD_BYTES ?= 10485760

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
	FILES_SERVICE_PORT="$(FILES_SERVICE_PORT)" \
	FILES_DATABASE_URL="$(DATABASE_URL)" \
	FILES_STORAGE_ROOT="$(FILES_STORAGE_ROOT)" \
	FILES_JWT_PUBLIC_KEY_PATH="$(FILES_JWT_PUBLIC_KEY_PATH)" \
	FILES_MAX_UPLOAD_BYTES="$(FILES_MAX_UPLOAD_BYTES)" \
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
