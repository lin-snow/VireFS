.PHONY: all test test-v lint fmt vet tidy check cover clean help

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

lint: fmt vet ## Run all linters (fmt + vet)

fmt: ## Check formatting (fails if any file needs gofmt)
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Files need gofmt:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

fmt-fix: ## Auto-fix formatting
	gofmt -w .

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy and verify go.mod
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod/go.sum not tidy" && exit 1)

check: tidy fmt vet ## Full pre-commit check (tidy + fmt + vet)

## ─── Utilities ──────────────────────────────────────────────────

clean: ## Remove generated files
	rm -f coverage.out coverage.html

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
