.PHONY: help build build-all release install fmt lint test test-coverage deps docs run clean

# Tool versions
GORELEASER_VERSION := v2@latest
GOLANGCI_LINT_VERSION := v2.5.0
GOMARKDOC_VERSION := latest

# Project variables
BINARY_NAME := movelooper
BUILD_DIR := bin
MAIN_PATH := main.go

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $1, $2}'

build: ## Build binary with goreleaser (current platform only)
	@echo "Building..."
	@go install github.com/goreleaser/goreleaser/$(GORELEASER_VERSION)
	@goreleaser build --skip=validate --single-target --snapshot --clean

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@go install github.com/goreleaser/goreleaser/$(GORELEASER_VERSION)
	@goreleaser build --skip=validate --snapshot --clean

release: ## Create a release with goreleaser
	@echo "Creating release..."
	@go install github.com/goreleaser/goreleaser/$(GORELEASER_VERSION)
	@goreleaser release --timeout 360s

install: ## Install binary globally
	@go install

fmt: ## Format code
	@go fmt ./...

lint: ## Run linter checks
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@golangci-lint -v run ./...

test: ## Run tests
	@go test -v ./...

test-coverage: ## Run tests with coverage report
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

docs: ## Generate documentation with gomarkdoc
	@go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@$(GOMARKDOC_VERSION)
	@gomarkdoc -e -o '{{.Dir}}/README.md' ./...

run: ## Run the application
	@go run $(MAIN_PATH)

clean: ## Remove build artifacts and cache
	@rm -rf $(BUILD_DIR) dist/ coverage.out
	@go clean -cache -testcache