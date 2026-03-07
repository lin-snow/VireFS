.PHONY: all test test-v lint lint-fix fmt fmt-fix vet tidy check cover clean help

all: check test ## Run all checks and tests

## ─── Testing ────────────────────────────────────────────────────

test: ## Run tests with race detector
	go test -race ./...

test-v: ## Run tests with verbose output
	go test -race -v ./...

cover: ## Run tests and open coverage report in browser
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## ─── Code Quality ───────────────────────────────────────────────

lint: ## Run golangci-lint (install if missing)
	@which golangci-lint > /dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	}
	golangci-lint run ./...

lint-fix: ## Run golangci-lint with auto-fix
	@which golangci-lint > /dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	}
	golangci-lint run --fix ./...

fmt: ## Check formatting via golangci-lint formatters
	@which golangci-lint > /dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	}
	golangci-lint fmt ./...

fmt-fix: ## Auto-fix formatting via golangci-lint
	@which golangci-lint > /dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	}
	golangci-lint fmt --fix ./...

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy and verify go.mod
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod/go.sum not tidy" && exit 1)

check: tidy lint ## Full pre-commit check (tidy + lint)

## ─── Utilities ──────────────────────────────────────────────────

clean: ## Remove generated files
	rm -f coverage.out coverage.html

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
