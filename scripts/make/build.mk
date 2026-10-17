.PHONY: build build-all release release-notes tag completions

## Build binary with goreleaser (current platform only)
build: $(GORELEASER)
	@echo "Building..."
	@$(GORELEASER) build --skip=validate --single-target --snapshot --clean

## Build binaries for all platforms
build-all: $(GORELEASER)
	@echo "Building for all platforms..."
	@$(GORELEASER) build --skip=validate --snapshot --clean

## Create a release with goreleaser
release: $(GORELEASER) completions release-notes
	@echo "Creating release..."
	@$(GORELEASER) release --timeout 360s --release-notes=$(RELEASE_NOTES)

# The tag at HEAD, so it matches what `make tag` wrote into the tag message.
## Write the git-cliff notes of the tag at HEAD into release-notes.md
release-notes: $(GITCLIFF)
	@$(GITCLIFF) --current --strip all -o $(RELEASE_NOTES)

## Create and push an annotated git tag with git-cliff notes (usage: make tag VERSION=v1.2.3)
tag: $(GITCLIFF)
ifndef VERSION
	$(error Usage: make tag VERSION=v1.2.3)
endif
	git diff --exit-code --quiet
	@$(call script,tag) $(GITCLIFF) $(VERSION)
	git push origin $(VERSION)

# Cobra generates these from the command tree, so they are always in sync with
# the real flags. They are release inputs, not tracked files: `release` depends
# on this target because .goreleaser.yaml attaches $(COMPLETIONS_DIR)/* to the
# release, and globbing a missing directory is a hard error there.
## Generate shell completion scripts into ./completions
completions:
	@$(call script,completions) $(MAIN_PATH) $(BINARY_NAME) $(COMPLETIONS_DIR)
