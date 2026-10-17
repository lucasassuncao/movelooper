.PHONY: deps fmt lint docs install run

## Download and tidy dependencies
deps:
	@go mod download
	@go mod tidy

## Format code
fmt:
	@go fmt ./...

## Run linter checks
lint: $(GOLANGCI)
	@$(GOLANGCI) -v run ./...

## Generate package docs (gomarkdoc) and config reference (generate-docs)
docs: $(GOMARKDOC)
	@$(call script,package-docs) $(GOMARKDOC) https://github.com/lucasassuncao/movelooper main ./internal/...
	@go run $(MAIN_PATH) generate-docs
	@echo ""

## Install binary globally
install:
	@go install

## Run the application
run:
	@go run $(MAIN_PATH)
