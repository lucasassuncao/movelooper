.PHONY: run install test docs lint build release deps fmt

# Variáveis para ferramentas externas
GOMARKDOC = github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
GOLANGCI_LINT = github.com/golangci/golangci-lint/cmd/golangci-lint@latest
GORELEASER = github.com/goreleaser/goreleaser@latest

# Alvo padrão
default: run

deps:
	go install $(GOMARKDOC)
	go install $(GOLANGCI_LINT)
	go install $(GORELEASER)

run:
	go run .

install:
	go build

fmt:
	go fmt ./...

test:
	go test -v ./...

docs: deps
	gomarkdoc -e -o '{{.Dir}}/README.md' ./...

lint: deps
	golangci-lint -v run ./...

build: deps
	goreleaser build --skip=validate --snapshot --clean

release: deps
	goreleaser release --timeout 360s