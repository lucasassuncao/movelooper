.PHONY: test test-watch test-coverage

## Run tests with gotestsum (testdox format)
test: $(GOTESTSUM)
	@$(GOTESTSUM) --format testdox -- -race ./...

## Run tests in watch mode (reruns on file changes)
test-watch: $(GOTESTSUM)
	@$(GOTESTSUM) --format testdox --watch -- -race ./...

## Run tests with coverage (HTML + Cobertura XML)
test-coverage: $(GOTESTSUM) $(GOCOBERTURA)
	@$(call script,test-coverage) $(GOTESTSUM) $(GOCOBERTURA) $(COVERAGE_DIR)
