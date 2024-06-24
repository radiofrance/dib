##-----------------------
## Available make targets
##-----------------------
##

default: help
help: ## Display this message
	@grep -E '(^[a-zA-Z0-9_.-]+:.*?##.*$$)|(^##)' Makefile | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | \
	sed -e 's/\[32m##/[33m/'

client/build: client.artifact

artifact: client/build ## Generate binary in dist folder
	goreleaser build --clean --snapshot --single-target

install: client/build ## Generate binary and copy it to $GOPATH/bin (equivalent to go install)
	goreleaser build --clean --snapshot --single-target -o $(GOPATH)/bin/dib

##
## ----------------------
## Q.A
## ----------------------
##

qa: lint test fmt ## Run Golang QA

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

##
## ----------------------
## Client
## ----------------------
##

client.artifact: ## Build client artifact (Svelte static site)
	rm -rf client/build
	# sed -i 's/unreleased/$(shell git describe --tags --abbrev=0)-next/' client/package.json
	NODE_ENV=CI npm -C client install
	NODE_ENV=production npm -C client run build

client.qa: client.lint ## Run client qa

client.lint: ## Run client linter (eslint & prettier)
	cd client && npm run lint

##
## ----------------------
## Doc
## ----------------------
##

.PHONY: docs
docs: ## Compile dib bin then regen CLI doc
	mkdir -p client/build && touch client/build/sample.txt
	CGO_ENABLED=0 go build -o ./dist/dib ./cmd
	./dist/dib docgen

doc.serve: docs ## Start dib static doc dev server
	( \
		. venv/bin/activate; \
		mkdocs serve; \
	)

doc.init: ## Init static doc python deps
	( \
		python3 -m virtualenv -p /usr/bin/python3 venv; \
		. venv/bin/activate; \
		pip install -r requirements.txt; \
	)
