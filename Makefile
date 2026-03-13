BINARY := plane
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell git log -1 --format=%cd --date=format:%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-s -w -X github.com/mggarofalo/plane-cli/cmd.version=$(VERSION) -X github.com/mggarofalo/plane-cli/cmd.commit=$(COMMIT) -X github.com/mggarofalo/plane-cli/cmd.date=$(DATE)"
EXT := $(shell go env GOEXE)

.PHONY: build clean test lint install hooks

build:
	go build $(LDFLAGS) -o bin/$(BINARY)$(EXT) .

GOBIN := $(subst \,/,$(shell go env GOPATH))/bin

install:
	go build $(LDFLAGS) -o $(GOBIN)/$(BINARY)$(EXT) .

clean:
ifeq ($(OS),Windows_NT)
	if exist bin rmdir /s /q bin
else
	rm -rf bin/
endif

test:
	go test ./... -v

lint:
	golangci-lint run ./...

hooks:
	git config core.hooksPath .githooks
