.PHONY: all build test lint fmt clean

BINARY := pub
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

all: fmt lint test build

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run

fmt:
	goimports -w .
	gofmt -w .

clean:
	rm -f $(BINARY)
