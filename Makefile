VERSION ?= v1.2.1-dev
BINARY = freelo
INSTALL_DIR = $(HOME)/bin

.PHONY: build install clean test test-integration test-live release-dry release help

## build: Build the freelo binary
build:
	go build -ldflags "-s -w -X github.com/freeloio/freelo-cli/internal/cli.Version=$(VERSION)" -o $(BINARY) ./cmd/freelo/

## install: Build and install to ~/bin
install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"
	@$(INSTALL_DIR)/$(BINARY) skill install claude 2>/dev/null && echo "Skill installed for Claude Code" || true

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## test: Run unit tests (no API required)
test:
	go test ./...

## test-integration: Run end-to-end tests against real Freelo API (needs .env.freelo-test)
test-integration:
	@if [ -f .env.freelo-test ]; then set -a && . ./.env.freelo-test && set +a; fi; \
	go test -tags=integration ./test/integration/... -v -timeout 5m

## test-live: Run all commands against the live Freelo API
test-live: build
	@echo "Running live API tests..."
	@PASS=0; FAIL=0; \
	run_test() { \
		output=$$(./$(BINARY) "$$@" --agent 2>&1); \
		if [ $$? -eq 0 ]; then \
			echo "  ✅ $$1"; \
			PASS=$$((PASS + 1)); \
		else \
			echo "  ❌ $$1"; \
			FAIL=$$((FAIL + 1)); \
		fi; \
	}; \
	echo ""; \
	run_test "version" version; \
	run_test "auth status" auth status; \
	run_test "users me" users me; \
	run_test "users list" users list; \
	run_test "projects list" projects list; \
	run_test "labels list" labels list; \
	run_test "custom-fields types" custom-fields types; \
	echo ""; \
	echo "Passed: $$PASS  Failed: $$FAIL"

## release-dry: Test GoReleaser config without publishing
release-dry:
	goreleaser release --snapshot --clean

## release: Create a real GitHub release (requires GITHUB_TOKEN)
release:
	@test -n "$(GITHUB_TOKEN)" || (echo "Error: GITHUB_TOKEN is required" && exit 1)
	goreleaser release --clean

## help: Show this help
help:
	@echo "Freelo CLI - Development Commands"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
