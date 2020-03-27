GOARCH ?= amd64
GOPATH ?= $(shell go env GOPATH)

EXECUTABLE=kibini
VERSION ?= $(shell git describe --tags --always --long --dirty)
LINUX_BIN_NAME=$(EXECUTABLE)-linux-$(GOARCH)
DARWIN_BIN_NAME=$(EXECUTABLE)-darwin-$(GOARCH)
KIBINI_MAIN=./cmd/kibini/main.go
GO_BUILD_COMMAND=go build -i -v -installsuffix cgo -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all lint bin clean


linux: $(LINUX_BIN_NAME)  ## Build binary for linux

darwin: $(DARWIN_BIN_NAME)  ## Build binary for macOS

$(LINUX_BIN_NAME):
	env CGO_ENABLED=0 GOOS=linux  $(GO_BUILD_COMMAND) -o $(GOPATH)/bin/$(LINUX_BIN_NAME) $(KIBINI_MAIN)

$(DARWIN_BIN_NAME):
	env CGO_ENABLED=0 GOOS=darwin $(GO_BUILD_COMMAND) -o $(GOPATH)/bin/$(DARWIN_BIN_NAME) $(KIBINI_MAIN)

bin: linux darwin  ## Build binaries
	@echo Successfully Built binaries with version: $(VERSION)

lint:  ## Lint
	@echo Installing linters...
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.24.0

	@echo Linting...
	@$(GOPATH)/bin/golangci-lint run \
		--deadline=300s \
		--disable-all \
		--enable=deadcode \
		--enable=goconst \
		--enable=gofmt \
		--enable=golint \
		--enable=gosimple \
		--enable=ineffassign \
		--enable=interfacer \
		--enable=misspell \
		--enable=staticcheck \
		--enable=unconvert \
		--enable=varcheck \
		--enable=vet \
		--enable=vetshadow \
		--enable=errcheck \
		--exclude="_test.go" \
		--exclude="comment on" \
		--exclude="error should be the last" \
		--exclude="should have comment" \
		./pkg/...
	@echo Done.

all: lint bin  ## Lint and build binaries
	@echo Done.

clean:  ## Clean previously built binaries
	rm -rf $(GOPATH)/bin/kibini-*

help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
