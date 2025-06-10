# SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
# SPDX-License-Identifier: Apache-2.0

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: build
build: ## Build binaries.
	go build -o bin/ ./...

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf bin/

.PHONY: tidy
tidy: ## Run `go mod tidy`.
	go mod tidy

.PHONY: get-u
get-u: ## Run `go get -u ./...`.
	go get -u ./...
	go mod tidy

.PHONY: test
test: ## Run `go test ./...`.
	rm -f cover.out cover.html
	go test ./... -coverprofile cover.out
	go tool cover -func cover.out
	go tool cover -html cover.out -o cover.html
