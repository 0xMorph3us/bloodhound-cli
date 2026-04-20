SHELL := /bin/sh

BINARY_NAME ?= bloodhound-cli
GO ?= go
MODULE_PATH := $(shell sed -n 's/^module //p' go.mod | head -n 1)
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo rolling)
BUILD_DATE ?= $(shell date -u '+%d %b %Y')

LDFLAGS := -s -w \
	-X '$(MODULE_PATH)/cmd/config.Version=$(VERSION)' \
	-X '$(MODULE_PATH)/cmd/config.BuildDate=$(BUILD_DATE)'

.PHONY: all install deps build test clean print-ldflags

all: build

# Install/fetch Go module dependencies for local development.
install: deps

check-go:
	@command -v $(GO) >/dev/null 2>&1 || { \
		echo "Error: '$(GO)' not found in PATH."; \
		echo "Install Go first (for Kali/Debian: apt install golang-go) and retry."; \
		exit 127; \
	}

deps: check-go
	$(GO) mod download

build: check-go
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) main.go

test: check-go
	$(GO) test ./...

clean:
	rm -f $(BINARY_NAME)

print-ldflags:
	@echo $(LDFLAGS)
