##@ Tools

.PHONY: tools clean-tools

# Tool versions
CADDY_VERSION         := v2.11.7
GITCLIFF_VERSION      := v2.14.2
GOCOBERTURA_VERSION   := latest
GOLANGCI_LINT_VERSION := v2.14.0
GOMARKDOC_VERSION     := v1.1.0
GORELEASER_VERSION    := v2.15.2
GOSEC_VERSION         := v2.29.1-0.20261012130337-71577268350e
GOTESTSUM_VERSION     := v1.13.0
GOVULNCHECK_VERSION   := v1.9.0
GRYPE_VERSION         := v0.118.0
PSSA_VERSION          := 1.25.0
SHELLCHECK_VERSION    := v0.11.0
SYFT_VERSION          := v1.51.1

# Tools are installed into ./.gobin (gitignored) rather than taken from PATH,
# so every target runs the version pinned above and never whatever happens to
# be installed system-wide. Each tool is a file target: the first target that
# needs it installs it, later runs reuse it. Bumping a version above does not
# invalidate an already installed binary - run `make clean-tools` first.
GOBIN_DIR := $(CURDIR)/.gobin

# git-cliff is used to generate release notes and tag messages.
GITCLIFF    := $(GOBIN_DIR)/git-cliff$(EXE)

# go tools for testing, linting, and documentation generation.
GOCOBERTURA := $(GOBIN_DIR)/gocover-cobertura$(EXE)
GOLANGCI    := $(GOBIN_DIR)/golangci-lint$(EXE)
GOMARKDOC   := $(GOBIN_DIR)/gomarkdoc$(EXE)
GORELEASER  := $(GOBIN_DIR)/goreleaser$(EXE)
GOSEC       := $(GOBIN_DIR)/gosec$(EXE)
GOTESTSUM   := $(GOBIN_DIR)/gotestsum$(EXE)
GOVULNCHECK := $(GOBIN_DIR)/govulncheck$(EXE)

# tools for generating SBOMs and scanning for vulnerabilities
GRYPE       := $(GOBIN_DIR)/grype$(EXE)
SYFT        := $(GOBIN_DIR)/syft$(EXE)

# linters for scripts/bash and scripts/pwsh. PSScriptAnalyzer is a pwsh module,
# saved as <name>/<version>, and its manifest is what Import-Module takes.
SHELLCHECK  := $(GOBIN_DIR)/shellcheck$(EXE)
PSSA        := $(GOBIN_DIR)/PSScriptAnalyzer/$(PSSA_VERSION)/PSScriptAnalyzer.psd1

# caddy serves the docs site locally.
CADDY       := $(GOBIN_DIR)/caddy$(EXE)

TOOLS := $(GORELEASER) $(GOLANGCI) $(GOMARKDOC) $(GOTESTSUM) $(GOSEC) $(GOCOBERTURA) $(SYFT) $(GRYPE) $(GOVULNCHECK) $(GITCLIFF) $(SHELLCHECK) $(PSSA) $(CADDY)

# `go install` takes its destination from GOBIN in the environment and has no flag for it, so the tool rules - and only those - export it. A file-wide
# export would also redirect `install`, which is meant to put movelooper in the user's own GOBIN.
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

$(GOVULNCHECK):
	@echo "Installing govulncheck $(GOVULNCHECK_VERSION)..."
	@go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

$(GITCLIFF):
	@echo "Installing git-cliff $(GITCLIFF_VERSION)..."
	@$(call script,install-git-cliff) $(GITCLIFF_VERSION) $(GOBIN_DIR)

$(SHELLCHECK):
	@echo "Installing shellcheck $(SHELLCHECK_VERSION)..."
	@$(call script,install-shellcheck) $(SHELLCHECK_VERSION) $(GOBIN_DIR)

$(PSSA):
	@echo "Installing PSScriptAnalyzer $(PSSA_VERSION)..."
	@$(call script,install-psscriptanalyzer) $(PSSA_VERSION) $(GOBIN_DIR)

$(CADDY):
	@echo "Installing caddy $(CADDY_VERSION)..."
	@go install github.com/caddyserver/caddy/v2/cmd/caddy@$(CADDY_VERSION)

## Install every pinned tool into ./.gobin
tools: $(TOOLS)
	@echo "Tools installed in $(GOBIN_DIR)"

## Remove ./.gobin so the next target reinstalls the pinned tools
clean-tools:
	@$(call script,remove) $(GOBIN_DIR)
