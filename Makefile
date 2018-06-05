NAME := kubernetes-ipfs
REGISTRY := docker.io/controlplane
GIT_TAG ?= $(shell bash -c 'TAG=$$(git tag | tail -n1); echo "$${TAG:-none}"')

CONTAINER_TAG ?= $(GIT_TAG)
CONTAINER_NAME := $(REGISTRY)/$(NAME):$(CONTAINER_TAG)
CONTAINER_NAME_LATEST := $(REGISTRY)/$(NAME):latest

all: help

docker-build: ## builds the test docker image
	@echo "+ $@"
	docker build -f Dockerfile --tag "$(CONTAINER_NAME)" .
	docker tag "$(CONTAINER_NAME)" "$(CONTAINER_NAME_LATEST)"
	@echo "Successfully tagged $(CONTAINER_NAME) as $(CONTAINER_NAME_LATEST)"

docker-push: ## pushes the docker image
	@echo "+ $@"
	docker push "$(CONTAINER_NAME)"
	docker push "$(CONTAINER_NAME_LATEST)"

help: ## parse jobs and descriptions from this Makefile
	@grep -E '^[ a-zA-Z0-9_-]+:([^=]|$$)' $(MAKEFILE_LIST) \
		| grep -Ev '^help\b[[:space:]]*:' \
		| sort \
		| awk 'BEGIN {FS = ":.*?##"}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: all docker-build docker-push help
