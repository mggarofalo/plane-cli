BINARY := plane
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X github.com/mggarofalo/plane-cli/cmd.version=$(VERSION) -X github.com/mggarofalo/plane-cli/cmd.commit=$(COMMIT) -X github.com/mggarofalo/plane-cli/cmd.date=$(DATE)"

.PHONY: build clean test lint install

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

install:
	go install $(LDFLAGS) .

clean:
	rm -rf bin/

test:
	go test ./... -v

lint:
	golangci-lint run ./...
