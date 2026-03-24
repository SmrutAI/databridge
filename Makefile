# Use bash for PIPESTATUS support (required for test-run target)
SHELL := /bin/bash

.PHONY: all build build-lambda build-server build-cli build-azure build-check \
        clean test test-run test-output-dir lint format check deps vendor \
        docker-lambda docker-azure docker-server help

BINARY_LAMBDA := bootstrap
BINARY_SERVER := server
BINARY_CLI    := codewatch

DIST_DIR := dist

BUILD_FLAGS := CGO_ENABLED=1

# Test configuration
TAGS ?=
RUN ?=
PKG ?= ./...
TIMEOUT ?= 10m
TEST_FLAGS := -v

ifdef TAGS
	TEST_FLAGS += -tags=$(TAGS)
endif

ifdef RUN
	TEST_FLAGS += -run=$(RUN)
endif

# Test output log (separate unit vs integration)
TEST_OUTPUT := test-outputs/$(if $(filter integration,$(TAGS)),test_integration_output.log,test_unit_output.log)

# Lint configuration (optional: specify linters and/or files to run)
# Usage:
#   make lint                                  # Run all linters on all files
#   make lint LINTERS=contextcheck,gocritic    # Run specific linters
#   make lint FILES=file1.go,file2.go          # Run on specific files
comma := ,
LINTERS ?=
FILES ?=
LINT_FLAGS = $(if $(LINTERS),--enable-only=$(LINTERS),) $(if $(FILES),$(subst $(comma), ,$(FILES)),)

# ============================================
# HELP
# ============================================

help: ## Show this help
	@echo "databridge — code indexing pipeline"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' | sort
	@echo ""
	@echo "Test Runner Examples:"
	@echo "  make test-run                           # All unit tests"
	@echo "  make test-run TAGS=integration          # All integration tests"
	@echo "  make test-run RUN=TestMyFunction        # Single test by name"
	@echo "  make test-run PKG=./source/...          # Tests in package"
	@echo ""
	@echo "Lint Examples:"
	@echo "  make lint                               # All configured linters"
	@echo "  make lint LINTERS=contextcheck          # Only contextcheck"
	@echo "  make lint LINTERS=gocritic,gosec        # Only gocritic and gosec"

# ============================================
# BUILD
# ============================================

all: build

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

build: build-lambda build-server build-cli ## Build all binaries

build-lambda: $(DIST_DIR) ## Build AWS Lambda binary (bootstrap)
	$(BUILD_FLAGS) go build -o $(DIST_DIR)/$(BINARY_LAMBDA) ./cmd/lambda/

build-server: $(DIST_DIR) ## Build standalone/Azure Functions server binary
	$(BUILD_FLAGS) go build -o $(DIST_DIR)/$(BINARY_SERVER) ./cmd/server/

build-azure: build-server ## Alias for build-server (Azure Functions target)

build-cli: $(DIST_DIR) ## Build local CLI (codewatch)
	$(BUILD_FLAGS) go build -o $(DIST_DIR)/$(BINARY_CLI) ./cmd/codewatch/

build-check: test-output-dir ## Compile-check all packages (no output binaries, for pre-commit)
	@echo "Build check running. Read test-outputs/build-check.log for details"
	@echo "=== Production code ===" | tee test-outputs/build-check.log
	@FAILED=0; \
	echo "  databridge..." | tee -a test-outputs/build-check.log; \
	if ! bash -c 'set -o pipefail; CGO_ENABLED=1 go build -gcflags="-e" ./... 2>&1 | tee -a test-outputs/build-check.log'; then \
		echo "  ✗ FAILED: databridge" | tee -a test-outputs/build-check.log; FAILED=1; \
	fi; \
	[ $$FAILED -eq 0 ] || exit 1
	@echo "=== Unit tests ===" | tee -a test-outputs/build-check.log
	@FAILED=0; \
	echo "  databridge..." | tee -a test-outputs/build-check.log; \
	if ! bash -c 'CGO_ENABLED=1 go test -c -gcflags="-e" -o /dev/null ./... 2>&1 | (grep -v "no test files" || true) | tee -a test-outputs/build-check.log; exit $${PIPESTATUS[0]}'; then \
		echo "  ✗ FAILED: databridge" | tee -a test-outputs/build-check.log; FAILED=1; \
	fi; \
	[ $$FAILED -eq 0 ] || exit 1
	@echo "✓ Build check passed" | tee -a test-outputs/build-check.log

# ============================================
# TESTING
# ============================================

test-output-dir:
	@mkdir -p test-outputs

test: test-run ## Alias for test-run

test-run: test-output-dir ## Run Go tests (TAGS/RUN/PKG for filtering)
	@echo "═══════════════════════════════════════════════════════════════════" | tee $(TEST_OUTPUT)
	@echo "Test Run: $$(date)" | tee -a $(TEST_OUTPUT)
	@echo "═══════════════════════════════════════════════════════════════════" | tee -a $(TEST_OUTPUT)
	@echo "Config:" | tee -a $(TEST_OUTPUT)
	@echo "  TAGS:    $(if $(TAGS),$(TAGS),(none - unit tests only))" | tee -a $(TEST_OUTPUT)
	@echo "  RUN:     $(if $(RUN),$(RUN),(all tests))" | tee -a $(TEST_OUTPUT)
	@echo "  PKG:     $(PKG)" | tee -a $(TEST_OUTPUT)
	@echo "  TIMEOUT: $(TIMEOUT)" | tee -a $(TEST_OUTPUT)
	@echo "═══════════════════════════════════════════════════════════════════" | tee -a $(TEST_OUTPUT)
	@echo "" | tee -a $(TEST_OUTPUT)
	@EXIT_CODE=0; \
	CGO_ENABLED=1 go test $(TEST_FLAGS) \
		-timeout $(TIMEOUT) \
		$(PKG) 2>&1 | tee -a $(TEST_OUTPUT); \
		if [ $${PIPESTATUS[0]} -ne 0 ]; then EXIT_CODE=1; fi; \
	echo "" | tee -a $(TEST_OUTPUT); \
	echo "═══════════════════════════════════════════════════════════════════" | tee -a $(TEST_OUTPUT); \
	echo "Test completed at: $$(date)" | tee -a $(TEST_OUTPUT); \
	echo "Exit code: $$EXIT_CODE" | tee -a $(TEST_OUTPUT); \
	echo "Output saved to: $(TEST_OUTPUT)" | tee -a $(TEST_OUTPUT); \
	echo "═══════════════════════════════════════════════════════════════════" | tee -a $(TEST_OUTPUT); \
	exit $$EXIT_CODE

# ============================================
# LINT & FMT
# ============================================

format: ## Format Go code (golangci-lint --fix + goimports + gci)
	@echo "Formatting databridge..."
	@golangci-lint run --fix ./... 2>/dev/null || true
	@echo "Fixing imports with goimports..."
	@goimports -w .
	@echo "Fixing import grouping with gci..."
	@which gci > /dev/null 2>&1 || (echo "Installing gci..." && go install github.com/daixiang0/gci@latest)
	@gci write --skip-generated -s standard -s default -s "prefix(github.com/SmrutAI)" .
	@echo "✅ Format complete"

lint: format ## Run golangci-lint (check-only). Usage: make lint [LINTERS=...] [FILES=...]
	@if [ -n "$(FILES)" ]; then \
		echo "Running golangci-lint on specific files: $(FILES)$(if $(LINTERS), with linters: $(LINTERS),)"; \
		golangci-lint run $(subst $(comma), ,$(FILES)) $(LINT_FLAGS); \
	elif [ -n "$(LINTERS)" ]; then \
		echo "Running golangci-lint for specific linters: $(LINTERS)..."; \
		golangci-lint run $(LINT_FLAGS) ./... || (echo "" && echo "❌ golangci-lint found issues (see above)" && exit 1); \
	else \
		echo "Running golangci-lint..."; \
		golangci-lint run ./... || (echo "" && echo "❌ golangci-lint found issues (see above)" && exit 1); \
	fi
	@echo "✅ Lint passed"

check: format build-check lint test-run ## Run format, build-check, lint, and tests (pre-commit sequence)

# ============================================
# DOCKER
# ============================================

docker-lambda: ## Build Lambda Docker image
	docker build --target lambda-intake -t databridge-lambda .

docker-azure: ## Build Azure Functions Docker image
	docker build --target azure-functions -t databridge-azure .

docker-server: ## Build standalone server Docker image
	docker build --target standalone -t databridge-server .

# ============================================
# DEPS
# ============================================

deps: ## Download and tidy Go dependencies
	go mod download && go mod tidy

vendor: ## Tidy go module dependencies
	@echo "Tidying module..."
	go mod tidy
	@echo "✓ Module tidied"

# ============================================
# CLEAN
# ============================================

clean: ## Clean build artifacts and test outputs
	rm -rf $(DIST_DIR)
	rm -rf test-outputs
	@echo "Cleaned build artifacts"
