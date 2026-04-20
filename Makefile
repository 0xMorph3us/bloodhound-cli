SHELL := /bin/sh

BINARY_NAME ?= bloodhound-cli
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo rolling)
BUILD_DATE ?= $(shell date -u '+%d %b %Y')

LDFLAGS := -s -w \
	-X 'github.com/SpecterOps/BloodHound_CLI/cmd/config.Version=$(VERSION)' \
	-X 'github.com/SpecterOps/BloodHound_CLI/cmd/config.BuildDate=$(BUILD_DATE)'

.PHONY: all install deps build test clean

all: build

# Install/fetch Go module dependencies for local development.
install: deps

deps:
	go mod download

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) main.go

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME)
