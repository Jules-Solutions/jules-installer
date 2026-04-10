.PHONY: build run test lint clean

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  = -s -w \
           -X github.com/Jules-Solutions/jules-installer/pkg/version.Version=$(VERSION) \
           -X github.com/Jules-Solutions/jules-installer/pkg/version.Commit=$(COMMIT) \
           -X github.com/Jules-Solutions/jules-installer/pkg/version.BuildDate=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/jules-setup ./cmd/jules-setup

run:
	go run ./cmd/jules-setup

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
