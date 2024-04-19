##-----------------------
## Available make targets
##-----------------------
##

default: help
help: ## Display this message
	@grep -E '(^[a-zA-Z0-9_.-]+:.*?##.*$$)|(^##)' Makefile | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | \
	sed -e 's/\[32m##/[33m/'

artifact: ## Generate binary in dist folder
	goreleaser build --clean --snapshot --single-target

install: ## Generate binary and copy it to $GOPATH/bin (equivalent to go install)
	goreleaser build --clean --snapshot --single-target -o $(GOPATH)/bin/dib

build: ## Build the CLI binary.
	CGO_ENABLED=0 go build -o ./dist/dib ./cmd

docs: build
	./dist/dib docgen

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

PKG := "./..."
RUN := ".*"
RED := $(shell tput setaf 1)
GREEN := $(shell tput setaf 2)
BLUE := $(shell tput setaf 4)
RESET := $(shell tput sgr0)

.PHONY: test
test: ## Run tests
	@go test -v -race -failfast -coverprofile coverage.output -run $(RUN) $(PKG) | \
        sed 's/RUN/$(BLUE)RUN$(RESET)/g' | \
        sed 's/CONT/$(BLUE)CONT$(RESET)/g' | \
        sed 's/PAUSE/$(BLUE)PAUSE$(RESET)/g' | \
        sed 's/PASS/$(GREEN)PASS$(RESET)/g' | \
        sed 's/FAIL/$(RED)FAIL$(RESET)/g'

fmt: ## Run `go fmt` on all files
	find -name '*.go' -exec gofmt -w -s '{}' ';'
