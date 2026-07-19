.PHONY: help keys up down logs ps tools gen test test-unit test-integration api-test schemathesis hurl lint fmt

SHELL := /bin/bash

KEYS_DIR := keys

help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ====== JWT Keys (RS256, 開発用) ======
keys: $(KEYS_DIR)/jwt_dev_private.pem ## 開発用 RS256 鍵ペアを生成 (既存ならスキップ / DEV ONLY)

$(KEYS_DIR)/jwt_dev_private.pem:
	@mkdir -p $(KEYS_DIR)
	openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out $(KEYS_DIR)/jwt_dev_private.pem
	openssl rsa -in $(KEYS_DIR)/jwt_dev_private.pem -pubout -out $(KEYS_DIR)/jwt_dev_public.pem
	@echo "✅ 開発用 JWT 鍵を $(KEYS_DIR)/ に生成しました (DEV ONLY / git 管理しない)"

# ====== Docker Compose ======
up: keys ## サービス全体を起動 (postgres + auth)
	docker compose up -d --build
	@echo "Auth:  http://localhost:8081"

up-db: ## PostgreSQLのみ起動 (ローカル開発用)
	docker compose up -d postgres

down: ## サービスを停止
	docker compose down

logs: ## サービスのログを表示
	docker compose logs -f

ps: ## サービスの状態を表示
	docker compose ps

# ====== Code Generation ======
tools: ## 開発ツールのインストール (oapi-codegen, sqlc, migrate)
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

gen: gen-auth ## OpenAPIとSQLからGoコードを生成

gen-auth: ## authサービスのコード生成
	$(MAKE) -C services/auth gen

# ====== Tests ======
test: test-unit test-integration ## 全テスト実行

test-unit: ## 単体テスト実行
	$(MAKE) -C services/auth test-unit

test-integration: ## 統合テスト実行 (testcontainers-go)
	$(MAKE) -C services/auth test-integration

# ====== API Tests ======
api-test: schemathesis hurl ## APIテストをすべて実行

schemathesis: ## Schemathesis でOpenAPI準拠の自動テスト
	$(MAKE) -C api-tests/schemathesis run

hurl: ## Hurl でシナリオベースのAPIテスト
	$(MAKE) -C api-tests/hurl run

# ====== Lint / Format ======
lint:
	cd services/auth && go vet ./...

fmt:
	cd services/auth && go fmt ./...
