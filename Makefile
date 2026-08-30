.PHONY: help build build-all release tag install fmt lint test test-coverage test-watch security sbom vuln deps docs all run tools tools-clean clean

# Tool versions
GOLANGCI_LINT_VERSION := v2.13.2
GOMARKDOC_VERSION     := v1.1.0
GORELEASER_VERSION    := v2.15.2
GOTESTSUM_VERSION     := v1.13.0
GOSEC_VERSION         := v2.29.0
GOCOBERTURA_VERSION   := latest
SYFT_VERSION          := v1.51.1
GRYPE_VERSION         := v0.118.0

# Tools are installed into ./.gobin (gitignored) rather than taken from PATH,
# so every target runs the version pinned above and never whatever happens to
# be installed system-wide. Each tool is a file target: the first target that
# needs it installs it, later runs reuse it. Bumping a version above does not
# invalidate an already installed binary - run `make tools-clean` first.
GOBIN_DIR := $(CURDIR)/.gobin

ifeq ($(OS),Windows_NT)
EXE := .exe
else
EXE :=
endif

GORELEASER  := $(GOBIN_DIR)/goreleaser$(EXE)
GOLANGCI    := $(GOBIN_DIR)/golangci-lint$(EXE)
GOMARKDOC   := $(GOBIN_DIR)/gomarkdoc$(EXE)
GOTESTSUM   := $(GOBIN_DIR)/gotestsum$(EXE)
GOSEC       := $(GOBIN_DIR)/gosec$(EXE)
GOCOBERTURA := $(GOBIN_DIR)/gocover-cobertura$(EXE)
SYFT        := $(GOBIN_DIR)/syft$(EXE)
GRYPE       := $(GOBIN_DIR)/grype$(EXE)

TOOLS := $(GORELEASER) $(GOLANGCI) $(GOMARKDOC) $(GOTESTSUM) $(GOSEC) $(GOCOBERTURA) $(SYFT) $(GRYPE)

# `go install` takes its destination from GOBIN in the environment and has no
# flag for it, so the tool rules - and only those - export it. A file-wide
# export would also redirect `install`, which is meant to put movelooper in the
# user's own GOBIN.
$(TOOLS): export GOBIN = $(GOBIN_DIR)

$(GORELEASER):
	@echo "Installing goreleaser $(GORELEASER_VERSION)..."
	@go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

$(GOLANGCI):
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOMARKDOC):
	@echo "Installing gomarkdoc $(GOMARKDOC_VERSION)..."
	@go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@$(GOMARKDOC_VERSION)

$(GOTESTSUM):
	@echo "Installing gotestsum $(GOTESTSUM_VERSION)..."
	@go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)

$(GOSEC):
	@echo "Installing gosec $(GOSEC_VERSION)..."
	@go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

$(GOCOBERTURA):
	@echo "Installing gocover-cobertura $(GOCOBERTURA_VERSION)..."
	@go install github.com/t-yuki/gocover-cobertura@$(GOCOBERTURA_VERSION)

$(SYFT):
	@echo "Installing syft $(SYFT_VERSION)..."
	@go install github.com/anchore/syft/cmd/syft@$(SYFT_VERSION)

$(GRYPE):
	@echo "Installing grype $(GRYPE_VERSION)..."
	@go install github.com/anchore/grype/cmd/grype@$(GRYPE_VERSION)

# Coverage
COVERAGE_DIR  := coverage
COVERAGE_OUT  := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html
COVERAGE_XML  := $(COVERAGE_DIR)/coverage.xml

# Recipes run through whatever shell make finds, and on Windows that is sh.exe
# only when one is on PATH - cmd.exe otherwise. Make does not say which it
# picked: $(SHELL) reads "sh.exe" either way, even when the fallback happened.
# So a recipe line has to be valid in both, and branching on $(OS) is not enough
# to tell them apart.
#
# `mkdir -p` is not valid in both. cmd.exe has no -p flag and reads it as a
# second directory to create, so the first run leaves a stray ./-p in the repo
# and every run after that fails outright, because cmd's mkdir treats an
# existing directory as an error rather than as nothing to do.
#
# Asking make instead of the shell settles it. wildcard is expanded at parse
# time, so the "already there" case runs no command at all, and creating one
# level with a bare mkdir is the same command under every shell.
ifeq ($(wildcard $(COVERAGE_DIR)),)
MKDIR_COVERAGE = mkdir $(COVERAGE_DIR)
else
MKDIR_COVERAGE =
endif

# Project variables
BINARY_NAME := movelooper
BUILD_DIR   := bin
MAIN_PATH   := main.go
SBOM_FILE   := sbom.json

help: ## Show this help message
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "Select-String -Path $(MAKEFILE_LIST) -Pattern '^([a-zA-Z_-]+):.*## (.+)' | Sort-Object { $$_.Matches[0].Groups[1].Value } | ForEach-Object { Write-Host -NoNewline -ForegroundColor Cyan ('{0,-20}' -f $$_.Matches[0].Groups[1].Value); Write-Host (' ' + $$_.Matches[0].Groups[2].Value) }"
else
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
endif

build: $(GORELEASER) ## Build binary with goreleaser (current platform only)
	@echo "Building..."
	@$(GORELEASER) build --skip=validate --single-target --snapshot --clean

build-all: $(GORELEASER) ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@$(GORELEASER) build --skip=validate --snapshot --clean

release: $(GORELEASER) ## Create a release with goreleaser
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

lint: $(GOLANGCI) ## Run linter checks
	@$(GOLANGCI) -v run ./...

test: $(GOTESTSUM) ## Run tests with gotestsum (testdox format)
	@$(GOTESTSUM) --format testdox -- -race ./...

test-watch: $(GOTESTSUM) ## Run tests in watch mode (reruns on file changes)
	@$(GOTESTSUM) --format testdox --watch -- -race ./...

test-coverage: $(GOTESTSUM) $(GOCOBERTURA) ## Run tests with coverage (HTML + Cobertura XML)
	@$(MKDIR_COVERAGE)
	@$(GOTESTSUM) --format testdox -- -race -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE_OUT) | tail -1
	@go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@$(GOCOBERTURA) < $(COVERAGE_OUT) > $(COVERAGE_XML)
	@echo "Reports: $(COVERAGE_HTML) | $(COVERAGE_XML)"

security: $(GOSEC) ## Run security analysis with gosec
	@$(GOSEC) -stdout -severity medium ./...

sbom: $(SYFT) ## Generate a CycloneDX SBOM of the dependency tree
	@echo "Generating SBOM..."
	@$(SYFT) . --source-name $(BINARY_NAME) --exclude './.gobin/**' --exclude './$(BUILD_DIR)/**' --exclude './dist/**' -o cyclonedx-json=$(SBOM_FILE)

vuln: $(GRYPE) sbom ## Scan the SBOM for known vulnerabilities with grype
	@echo "Scanning for vulnerabilities..."
	@$(GRYPE) sbom:$(SBOM_FILE) --fail-on medium

deps: ## Download and tidy dependencies
	@go mod download
	@go mod tidy

docs: $(GOMARKDOC) ## Generate package docs (gomarkdoc) and config reference (generate-docs)
	@$(GOMARKDOC) -e \
		--repository.url https://github.com/lucasassuncao/movelooper \
		--repository.default-branch main \
		--repository.path / \
		-o '{{.Dir}}/README.md' ./internal/...
	@go run $(MAIN_PATH) generate-docs
	@echo ""

# Local, deterministic checks first, so a failure points at the code. The
# supply-chain scan goes last because it is the only step that needs the
# network: grype refreshes its vulnerability database, and a hiccup there
# should not mask a lint or test failure.
all: deps fmt docs lint security test-coverage sbom vuln

run: ## Run the application
	@go run $(MAIN_PATH)

tools: $(TOOLS) ## Install every pinned tool into ./.gobin
	@echo "Tools installed in $(GOBIN_DIR)"

tools-clean: ## Remove ./.gobin so the next target reinstalls the pinned tools
	@echo "Removing $(GOBIN_DIR)..."
	@rm -rf $(GOBIN_DIR)

clean: tools-clean ## Remove build artifacts, installed tools and cache
	@echo "Removing $(BUILD_DIR)..."
	@rm -rf $(BUILD_DIR)
	@echo ""
	@echo "Removing dist..."
	@rm -rf dist/
	@echo ""
	@echo "Removing $(COVERAGE_DIR)..."
	@rm -rf $(COVERAGE_DIR)
	@echo ""
	@echo "Removing $(SBOM_FILE)..."
	@rm -rf $(SBOM_FILE)
	@echo ""
	@echo "Cleaning Go build and test cache..."
	@go clean -cache -testcache
	@echo ""
