.PHONY: help build build-all release tag install fmt lint test test-coverage deps docs run clean

# Tool versions
GOLANGCI_LINT_VERSION := v2.5.0
GOMARKDOC_VERSION := latest

# Tools invoked via `go run` — no global install required
GORELEASER  := go run github.com/goreleaser/goreleaser/v2@latest
GOLANGCI    := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOMARKDOC   := go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@$(GOMARKDOC_VERSION)

# Project variables
BINARY_NAME := movelooper
BUILD_DIR := bin
MAIN_PATH := main.go

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $1, $2}'

build: ## Build binary with goreleaser (current platform only)
	@echo "Building..."
	@$(GORELEASER) build --skip=validate --single-target --snapshot --clean

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@$(GORELEASER) build --skip=validate --snapshot --clean

release: ## Create a release with goreleaser
	@echo "Creating release..."
	@$(GORELEASER) release --timeout 360s

tag: ## Create and push an annotated git tag (usage: make tag VERSION=v1.2.3)
ifndef VERSION
	$(error Usage: make tag VERSION=v1.2.3)
endif
	git diff --exit-code --quiet
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

install: ## Install binary globally
	@go install

fmt: ## Format code
	@go fmt ./...

lint: ## Run linter checks
	@$(GOLANGCI) -v run ./...

test: ## Run tests
	@go test -v ./...

test-coverage: ## Run tests with coverage report
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

docs: ## Generate documentation with gomarkdoc
	@$(GOMARKDOC) -e \
		--repository.url https://github.com/lucasassuncao/movelooper \
		--repository.default-branch main \
		--repository.path / \
		-o '{{.Dir}}/README.md' ./internal/...

run: ## Run the application
	@go run $(MAIN_PATH)

clean: ## Remove build artifacts and cache
	@rm -rf $(BUILD_DIR) dist/ coverage.out
	@go clean -cache -testcache