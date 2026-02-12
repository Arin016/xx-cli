APP_NAME := xx
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
