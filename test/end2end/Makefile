##-----------------------
## Available make targets
##-----------------------
##

ROOT_PATH=${PWD}/../..

default: help

help: ## Display this message
	@grep -E '(^[a-zA-Z0-9_.-]+:.*?##.*$$)|(^##)' Makefile | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | \
		sed -e 's/\[32m##/[33m/'

##
## ----------------------
## DIB end2end tests
## ----------------------
##

registry.start: ## Start required local Docker registry
	docker compose up --detach
	docker compose ps

dib.install: ## Generate binary and copy it to $GOPATH/bin (equivalent to go install)
	$(MAKE) --directory=${ROOT_PATH} install

dib.list: ## Update list of Docker images managed by DIB
	dib list -o go-template-file=.dib_list.tmpl > dib_images_list.md

dib.build: ## Build updated Docker images managed by DIB
	dib build
