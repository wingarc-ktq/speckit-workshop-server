.PHONY: install run run-auth clean

AUTH_URL  ?= http://localhost:8081/api/v1
REPORT_DIR := report

install: ## Hurl をインストール (Mac は brew、それ以外は手順を表示)
	@if command -v brew >/dev/null 2>&1; then \
	  brew install hurl; \
	else \
	  echo "Linux/Windows: https://hurl.dev/docs/installation.html"; \
	fi

run: run-auth ## 全シナリオ実行

run-auth:
	mkdir -p $(REPORT_DIR)
	hurl --test \
	  --variable auth_url=$(AUTH_URL) \
	  --variable run_id=$(shell uuidgen | tr 'A-Z' 'a-z') \
	  --report-html $(REPORT_DIR)/auth \
	  scenarios/auth/*.hurl

clean:
	rm -rf $(REPORT_DIR)
