.PHONY: help build build-all release tag install fmt lint test test-coverage test-watch security deps docs all run clean

# Tool versions
GOLANGCI_LINT_VERSION := v2.5.0
GOMARKDOC_VERSION     := v1.1.0
GORELEASER_VERSION    := v2.15.2
GOTESTSUM_VERSION     := v1.13.0
GOSEC_VERSION         := v2.25.0
GOCOBERTURA_VERSION   := latest

# Tools invoked via go run -- no global install required
GORELEASER  := go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
GOLANGCI    := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOMARKDOC   := go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@$(GOMARKDOC_VERSION)
GOTESTSUM   := go run gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
GOSEC       := go run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
GOCOBERTURA := go run github.com/t-yuki/gocover-cobertura@$(GOCOBERTURA_VERSION)

# Coverage
COVERAGE_OUT  := coverage.out
COVERAGE_HTML := coverage.html
COVERAGE_XML  := coverage.xml

# Project variables
BINARY_NAME := movelooper
BUILD_DIR   := bin
MAIN_PATH   := main.go

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

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

test: ## Run tests with gotestsum (testdox format)
	@$(GOTESTSUM) --format testdox ./...

test-watch: ## Run tests in watch mode (reruns on file changes)
	@$(GOTESTSUM) --format testdox --watch ./...

test-coverage: ## Run tests with coverage (HTML + Cobertura XML)
	@$(GOTESTSUM) --format testdox -- -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE_OUT) | tail -1
	@go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@$(GOCOBERTURA) < $(COVERAGE_OUT) > $(COVERAGE_XML)
	@echo "Reports: $(COVERAGE_HTML) | $(COVERAGE_XML)"

security: ## Run security analysis with gosec
	@$(GOSEC) -stdout -severity medium ./...

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

docs: ## Generate documentation with gomarkdoc
	@$(GOMARKDOC) -e \
		--repository.url https://github.com/lucasassuncao/movelooper \
		--repository.default-branch main \
		--repository.path / \
		-o '{{.Dir}}/README.md' ./internal/...

all: fmt docs lint security test-coverage

run: ## Run the application
	@go run $(MAIN_PATH)

clean: ## Remove build artifacts and cache
	@rm -rf $(BUILD_DIR) dist/ $(COVERAGE_OUT) $(COVERAGE_HTML) $(COVERAGE_XML)
	@go clean -cache -testcache
