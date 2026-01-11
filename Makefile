.PHONY: all build test lint fmt clean

BINARY := pub
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/jonandersen/pub/cmd.Version=$(VERSION)"

all: fmt lint test build

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -v -race -cover ./...

GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)

lint:
	@test -x $(GOLANGCI_LINT) || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" && exit 1)
	$(GOLANGCI_LINT) run

fmt:
	go tool goimports -w .
	gofmt -w .

clean:
	rm -f $(BINARY)
