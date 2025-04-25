##-----------------------
## Available make targets
##-----------------------
##

##########################
# Configuration
##########################

GO ?= go
GOOS ?= $(shell $(GO) env GOOS)

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
	CGO_ENABLED=0 GOOS=$(GOOS) $(GO) build -o ./dist/dib ./cmd

docs: build
	./dist/dib docgen

##
## ----------------------
## Q.A
## ----------------------
##

qa: lint test

# renovate: datasource=github-releases depName=radiofrance/lint-config
LINT_CONFIG_VERSION = v1.1.1

lint: ## Lint source code
	curl -o .golangci.yml -sS \
		"https://raw.githubusercontent.com/radiofrance/lint-config/$(LINT_CONFIG_VERSION)/.golangci.yml"
	golangci-lint run --verbose

PKG = ./...
RUN = ".*"
RED = $(shell tput setaf 1)
GREEN = $(shell tput setaf 2)
BLUE = $(shell tput setaf 4)
RESET = $(shell tput sgr0)

.PHONY: test
test: ## Run tests
	@go test -v -race -failfast -coverprofile coverage.out -covermode atomic -run $(RUN) $(PKG) | \
        sed 's/RUN/$(BLUE)RUN$(RESET)/g' | \
        sed 's/CONT/$(BLUE)CONT$(RESET)/g' | \
        sed 's/PAUSE/$(BLUE)PAUSE$(RESET)/g' | \
        sed 's/PASS/$(GREEN)PASS$(RESET)/g' | \
        sed 's/FAIL/$(RED)FAIL$(RESET)/g'

coverage: test ## Run test, then generate coverage html report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "To open the html coverage file, use one of the following commands:"
	@echo "open coverage.html on mac"
	@echo "xdg-open coverage.html on linux"
