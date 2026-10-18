##@ Go

.PHONY: deps fmt lint install run

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

## Install binary globally
install:
	@go install

## Run the application
run:
	@go run $(MAIN_PATH)
