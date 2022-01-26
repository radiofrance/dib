##-----------------------
## Available make targets
##-----------------------
##

default: help
help: ## Display this message
	@grep -E '(^[a-zA-Z0-9_.-]+:.*?##.*$$)|(^##)' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

clean: ## Clean tmp files
	rm -rf dist

BINARY_NAME=dib
ARCH = $(shell arch | sed 's|x86_64|amd64|g')
CURRENT_OS = $(shell uname | tr '[:upper:]' '[:lower:]')

artifact: clean ## Build & copy sk binary
	goreleaser build --snapshot --rm-dist --single-target

##
## ----------------------
## Q.A
## ----------------------
##

qa: lint test ## Run all QA process

lint: ## Lint source code
	golangci-lint run -v

lint.fix: ## Lint and fix source code
	golangci-lint run --fix -v

.PHONY: test
test: ## Run tests
	go test -race -v ./... -coverprofile coverage.output

fmt: ## Run `go fmt` on all files
	find -name '*.go' -exec gofmt -w -s '{}' ';'
