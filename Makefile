GOARCH ?= amd64
GOPATH ?= $(shell go env GOPATH)

EXECUTABLE=kibini
VERSION ?= $(shell git describe --tags --always --long --dirty)
LINUX_BIN_NAME=$(EXECUTABLE)-linux-$(GOARCH)
DARWIN_BIN_NAME=$(EXECUTABLE)-darwin-$(GOARCH)
KIBINI_MAIN=./cmd/kibini/main.go
GO_BUILD_COMMAND=go build -i -installsuffix cgo -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all lint bin clean

all: lint bin  ## Lint and build binaries
	@echo Done.

linux: $(LINUX_BIN_NAME)  ## Build binary for linux

darwin: $(DARWIN_BIN_NAME)  ## Build binary for macOS

$(LINUX_BIN_NAME):
	env CGO_ENABLED=0 GOOS=linux  $(GO_BUILD_COMMAND) -o $(LINUX_BIN_NAME) $(KIBINI_MAIN)

$(DARWIN_BIN_NAME):
	env CGO_ENABLED=0 GOOS=darwin $(GO_BUILD_COMMAND) -o $(DARWIN_BIN_NAME) $(KIBINI_MAIN)

bin: linux darwin  ## Build binaries
	@echo Successfully Built binaries with version: $(VERSION)

fmt:  ## Format code
	@go fmt $(shell go list ./... | grep -v /vendor/)

lint:  ## Lint
	./hack/lint/install.sh
	./hack/lint/run.sh

clean:  ## Clean previously built binaries
	rm -rf $(GOPATH)/bin/kibini-*

help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
