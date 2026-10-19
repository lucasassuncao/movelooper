##@ Docs

.PHONY: docs stage-docs serve-docs demo

## Generate package docs (gomarkdoc) and config reference (generate-docs)
docs: $(GOMARKDOC)
	@$(call script,package-docs) $(GOMARKDOC) https://github.com/lucasassuncao/movelooper main ./internal/...
	@go run $(MAIN_PATH) generate-docs
	@echo ""

# The Pages job in CI runs this too, so the local preview and the published site
# are built from the same layout.
## Assemble the docs site into ./site (docs/pages, the markdown and the demo gif)
stage-docs:
	@$(call script,stage-docs) docs examples/demo/demo.gif $(SITE_DIR)

# docsify fetches the .md files at runtime, which browsers block on file://, so
# the site needs a server.
## Stage the docs site and serve it on http://localhost:8080 (Ctrl+C to stop)
serve-docs: $(CADDY) stage-docs
	@echo "Landing page: http://localhost:8080/"
	@echo "Docs:         http://localhost:8080/docs/"
	@$(CADDY) file-server --root $(SITE_DIR) --listen :8080

# The tape types `movelooper`, so the binary built from this tree goes first on
# PATH. Otherwise the recording would show whatever version is installed.
## Record examples/demo/demo.gif with VHS (needs vhs, ttyd and ffmpeg on PATH)
demo:
	@go build -o $(BUILD_DIR)/$(BINARY_NAME)$(EXE) .
	@$(call script,record-tape) $(BUILD_DIR) examples/demo/demo.tape
