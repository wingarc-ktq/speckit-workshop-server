.PHONY: install run run-auth register-user get-token report clean

AUTH_URL  ?= http://localhost:8081/api/v1
AUTH_SCHEMA  := ../../schema/auth/openapi.yaml
REPORT_DIR := report

# JWT 取得に使う固定テストユーザー (register-user で Auth に登録し get-token でログインする)
TEST_EMAIL    ?= taro@example.com
TEST_PASSWORD ?= P@ssw0rd!
TEST_NAME     ?= 田中太郎

# Homebrew などの Python は PEP 668 で直接 pip install できないため venv に隔離する.
VENV := .venv
ST   := $(VENV)/bin/schemathesis

install: ## Schemathesis を venv にインストール (Python 3.11+ が必要)
	python3 -m venv $(VENV)
	$(VENV)/bin/pip install --upgrade pip schemathesis

$(ST):
	@echo "❌ schemathesis 未インストール。先に 'make install' を実行してください"; exit 1

run: run-auth ## Auth サービスをテスト

run-auth: $(ST) ## Auth サービスを Schemathesis で検証 (可能ならログインし /auth/me も検証)
	mkdir -p $(REPORT_DIR)
	@# /auth/me の認証成功パスも網羅するため、ログインできればトークンを付与する.
	@# 取得できなくても未認証パスの検証は続行する (set -- でヘッダ引数の空白を保持).
	@TOKEN="$${AUTH_TOKEN:-$$($(MAKE) -s get-token)}"; \
	set --; \
	if [ -n "$$TOKEN" ] && [ "$$TOKEN" != "null" ]; then \
	  set -- --header "Authorization: Bearer $$TOKEN"; \
	else \
	  echo "⚠️  トークン未取得: 未認証パスのみ検証します"; \
	fi; \
	$(ST) run \
	  --url $(AUTH_URL) \
	  --checks all \
	  --max-examples 30 \
	  "$$@" \
	  --report junit --report-dir $(REPORT_DIR)/auth \
	  $(AUTH_SCHEMA)

register-user: ## テストユーザーを Auth に登録 (既に存在すれば 409 を無視・冪等)
	@curl -s -o /dev/null -X POST $(AUTH_URL)/auth/register \
	  -H 'Content-Type: application/json' \
	  -d '{"email":"$(TEST_EMAIL)","password":"$(TEST_PASSWORD)","name":"$(TEST_NAME)"}'

get-token: register-user ## テストユーザーでログインして accessToken を出力
	@curl -s -X POST $(AUTH_URL)/auth/login \
	  -H 'Content-Type: application/json' \
	  -d '{"email":"$(TEST_EMAIL)","password":"$(TEST_PASSWORD)"}' \
	  | jq -r '.accessToken'

clean:
	rm -rf $(REPORT_DIR)
