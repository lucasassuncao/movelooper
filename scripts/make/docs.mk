##@ Docs

.PHONY: docs serve-docs demo

## Generate package docs (gomarkdoc) and config reference (generate-docs)
docs: $(GOMARKDOC)
	@$(call script,package-docs) $(GOMARKDOC) https://github.com/lucasassuncao/movelooper main ./internal/...
	@go run $(MAIN_PATH) generate-docs
	@echo ""

# docsify fetches the .md files at runtime, which browsers block on file://, so
# the site needs a server. docs/ is the root because docsify's basePath is ../../.
## Serve the docs site on http://localhost:8080 (Ctrl+C to stop)
serve-docs: $(CADDY)
	@echo "Landing page: http://localhost:8080/pages/"
	@echo "Docs:         http://localhost:8080/pages/docs/"
	@$(CADDY) file-server --root docs --listen :8080

# The tape types `movelooper`, so the binary built from this tree goes first on
# PATH. Otherwise the recording would show whatever version is installed.
## Record examples/demo/demo.gif with VHS (needs vhs, ttyd and ffmpeg on PATH)
demo:
	@go build -o $(BUILD_DIR)/$(BINARY_NAME)$(EXE) .
	@$(call script,record-tape) $(BUILD_DIR) examples/demo/demo.tape
