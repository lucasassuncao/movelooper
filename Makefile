# Targets live in scripts/make/*.mk, one file per area. A recipe that needs more
# than portable commands calls a script instead, and every script exists in both
# scripts/bash and scripts/pwsh with the same name and arguments.

# The includes are read before any target of this file, so without this the
# first target of tools.mk would become the default goal.
.DEFAULT_GOAL := help

# On Windows make may run recipes under cmd.exe or sh, and $(SHELL) does not
# say which, so recipes stay shell-neutral and shell logic goes in a script.
ifeq ($(OS),Windows_NT)
EXE        := .exe
SCRIPT_DIR := scripts/pwsh
SCRIPT_EXT := .ps1
SCRIPT_RUN := pwsh -NoProfile -ExecutionPolicy Bypass -File
else
EXE        :=
SCRIPT_DIR := scripts/bash
SCRIPT_EXT := .sh
SCRIPT_RUN := bash
endif

# $(call script,name) is the command that runs this OS's version of a script.
script = $(SCRIPT_RUN) $(SCRIPT_DIR)/$(1)$(SCRIPT_EXT)

# Project variables
BINARY_NAME     := movelooper
BUILD_DIR       := bin
MAIN_PATH       := main.go
SBOM_FILE       := sbom.json
COVERAGE_DIR    := coverage
COMPLETIONS_DIR := completions

MAKE_DIR := scripts/make

# tools.mk first: the others name $(GOLANGCI), $(GOTESTSUM) and friends as
# prerequisites, and a prerequisite is expanded when its rule is read.
include $(MAKE_DIR)/tools.mk
include $(MAKE_DIR)/go.mk
include $(MAKE_DIR)/build.mk
include $(MAKE_DIR)/test.mk
include $(MAKE_DIR)/security.mk

.PHONY: help check-scripts all clean

# A `## text` line right above a target is its help entry. make appends every
# included file to MAKEFILE_LIST, so help sees the targets of all of them.
## Show this help message
help:
	@$(call script,help) $(MAKEFILE_LIST)

## Fail if a script in scripts/bash or scripts/pwsh has no twin in the other
check-scripts:
	@$(call script,check-parity) scripts/bash scripts/pwsh

# Local, deterministic checks first, so a failure points at the code. The
# supply-chain scans go last because they are the only steps that need the
# network: govulncheck and grype both fetch a vulnerability database, and a
# hiccup there should not mask a lint or test failure. govulncheck runs before
# grype so the reachability answer is printed even when grype fails the build
# on an advisory this binary never executes.
all: deps fmt docs lint security test-coverage govulncheck sbom vuln

## Remove build artifacts, installed tools and cache
clean: clean-tools
	@$(call script,remove) $(BUILD_DIR) dist $(COVERAGE_DIR) $(COMPLETIONS_DIR) $(SBOM_FILE)
	@echo "Cleaning Go build and test cache..."
	@go clean -cache -testcache
