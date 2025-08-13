# Common makefile commands & variables between projects
include .make/common.mk

# Common Golang makefile commands & variables between projects
include .make/go.mk

## Set default repository details if not provided
REPO_NAME  ?= go-pre-commit
REPO_OWNER ?= mrz1836

# Variables
BINARY_NAME := go-pre-commit
BINARY_PATH := ./cmd/$(BINARY_NAME)/$(BINARY_NAME)
GO := go
MODULE := github.com/mrz1836/go-pre-commit

# Build flags with version injection
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC' 2>/dev/null || echo "unknown")

LDFLAGS := -ldflags="-s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildDate=$(BUILD_DATE)"
BUILD_FLAGS := -trimpath

## build: Build the pre-commit binary
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@$(GO) build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/$(BINARY_NAME)
	@echo "Binary built: $(BINARY_PATH) (v$(VERSION))"

## clean: Clean build artifacts
clean:
	@echo "Cleaning $(BINARY_NAME) artifacts..."
	@rm -f $(BINARY_PATH)
	@$(GO) clean ./...
	@echo "Clean complete"

## install: Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@$(GO) install ./cmd/$(BINARY_NAME)
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

### uninstall: Remove the binary from GOPATH/bin
#uninstall:
#	@echo "Uninstalling $(BINARY_NAME)..."
#	@rm -f $$(go env GOPATH)/bin/$(BINARY_NAME)
#	@echo "Uninstalled $(BINARY_NAME)"

# Development helpers
## dev-install: Quick build and install for development
dev-install:
	@$(GO) build -o $(BINARY_PATH) ./cmd/$(BINARY_NAME) && \
		cp $(BINARY_PATH) $$(go env GOPATH)/bin/ && \
		echo "Development build installed"

## dev-test: Run tests for a specific package
dev-test:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make dev-test PKG=./internal/config [TEST=TestFunctionName]"; \
	else \
		echo "Running tests for $(PKG)..."; \
		$(GO) test -v $(if $(TEST),-run $(TEST)) $(PKG); \
	fi

## version: Display version information
version:
	@echo "Module: $(MODULE)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Go version: $$(go version)"
	@echo "Build time: $$(date)"
