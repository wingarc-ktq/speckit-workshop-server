.PHONY: install run run-auth run-files clean

AUTH_URL  ?= http://localhost:8081/api/v1
FILES_URL ?= http://localhost:8082/api/v1
REPORT_DIR := report
FIXTURE_DIR := scenarios/files/fixtures

install: ## Hurl をインストール (Mac は brew、それ以外は手順を表示)
	@if command -v brew >/dev/null 2>&1; then \
	  brew install hurl; \
	else \
	  echo "Linux/Windows: https://hurl.dev/docs/installation.html"; \
	fi

run: run-auth run-files ## 全シナリオ実行

run-auth:
	mkdir -p $(REPORT_DIR)
	hurl --test \
	  --variable auth_url=$(AUTH_URL) \
	  --variable run_id=$(shell uuidgen | tr 'A-Z' 'a-z') \
	  --report-html $(REPORT_DIR)/auth \
	  scenarios/auth/*.hurl

run-files:
	mkdir -p $(REPORT_DIR)
	mkdir -p $(FIXTURE_DIR)
	printf 'monthly report\n' > "$(FIXTURE_DIR)/月次報告書.txt"
	dd if=/dev/zero of=$(FIXTURE_DIR)/too-large.bin bs=1048576 count=11 status=none
	@run_id=$$(uuidgen | tr 'A-Z' 'a-z'); \
	printf 'project proposal\n' > "$(FIXTURE_DIR)/list-$$run_id-企画書.txt"; \
	printf 'meeting minutes\n' > "$(FIXTURE_DIR)/list-$$run_id-議事録.txt"; \
	printf 'progress report\n' > "$(FIXTURE_DIR)/list-$$run_id-報告書.txt"; \
	printf 'invoice\n' > "$(FIXTURE_DIR)/search-$$run_id-請求書.txt"; \
	printf 'meeting minutes\n' > "$(FIXTURE_DIR)/search-$$run_id-議事録.txt"; \
	hurl --test \
	  --variable auth_url=$(AUTH_URL) \
	  --variable files_url=$(FILES_URL) \
	  --variable run_id=$$run_id \
	  --report-html $(REPORT_DIR)/files \
	  scenarios/files/*.hurl; \
	status=$$?; \
	rm -f \
	  "$(FIXTURE_DIR)/list-$$run_id-企画書.txt" \
	  "$(FIXTURE_DIR)/list-$$run_id-議事録.txt" \
	  "$(FIXTURE_DIR)/list-$$run_id-報告書.txt" \
	  "$(FIXTURE_DIR)/search-$$run_id-請求書.txt" \
	  "$(FIXTURE_DIR)/search-$$run_id-議事録.txt"; \
	exit $$status

clean:
	rm -rf $(REPORT_DIR)
	rm -rf $(FIXTURE_DIR)
