# Project variables
BINARY_NAME := mcp-for-argo-workflows
MODULE := github.com/Joibel/mcp-for-argo-workflows
DIST_DIR := dist
DOCKER_IMAGE := ghcr.io/joibel/mcp-for-argo-workflows

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildTime=$(BUILD_TIME)

# Go variables
GO := go
GOFMT := gofmt
GOIMPORTS := goimports
GOLANGCI_LINT := golangci-lint

# Source files for dependency tracking (exclude dist/ and vendor/)
GO_FILES := $(shell find . -name '*.go' -type f -not -path './dist/*' -not -path './vendor/*')
GO_MOD := go.mod go.sum

# Platform-specific binary paths
DIST_DARWIN_AMD64 := $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64
DIST_DARWIN_ARM64 := $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64
DIST_LINUX_AMD64 := $(DIST_DIR)/$(BINARY_NAME)-linux-amd64
DIST_LINUX_ARM64 := $(DIST_DIR)/$(BINARY_NAME)-linux-arm64
DIST_WINDOWS_AMD64 := $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe
DIST_CHECKSUMS := $(DIST_DIR)/checksums.txt

# All distribution binaries
DIST_BINARIES := $(DIST_DARWIN_AMD64) $(DIST_DARWIN_ARM64) $(DIST_LINUX_AMD64) $(DIST_LINUX_ARM64) $(DIST_WINDOWS_AMD64)

.PHONY: all test test-e2e test-e2e-kubernetes test-e2e-argo-server lint lint-fix fmt vet clean tools help build-all dist-clean \
	docker-build docker-build-multiarch docker-push

# Default target
all: fmt vet lint test $(DIST_LINUX_AMD64)

## test: Run tests with race detection and coverage
test:
	@echo "Running tests..."
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/...
	@echo "Coverage report: coverage.out"

## test-e2e: Run end-to-end tests (requires Docker)
## Use E2E_MODE=kubernetes (default) or E2E_MODE=argo-server to select connection mode
test-e2e:
	@echo "Running E2E tests (mode: $${E2E_MODE:-kubernetes})..."
	E2E_MODE=$${E2E_MODE:-kubernetes} $(GO) test -tags=e2e -v -timeout=20m ./test/e2e/...

## test-e2e-kubernetes: Run E2E tests using direct Kubernetes API mode
test-e2e-kubernetes:
	@echo "Running E2E tests (Kubernetes API mode)..."
	E2E_MODE=kubernetes $(GO) test -tags=e2e -v -timeout=20m ./test/e2e/...

## test-e2e-argo-server: Run E2E tests using Argo Server mode
test-e2e-argo-server:
	@echo "Running E2E tests (Argo Server mode)..."
	E2E_MODE=argo-server $(GO) test -tags=e2e -v -timeout=20m ./test/e2e/...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	$(GOLANGCI_LINT) run ./...

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	@echo "Running linter with auto-fix..."
	$(GOLANGCI_LINT) run --fix ./...

## fmt: Run gofmt and goimports
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@if command -v $(GOIMPORTS) >/dev/null 2>&1; then \
		$(GOIMPORTS) -w -local $(MODULE) .; \
	else \
		echo "goimports not installed, skipping import formatting"; \
	fi

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(DIST_DIR)/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## tools: Install development dependencies
tools:
	@echo "Installing development tools..."
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	@echo "Note: Install golangci-lint from https://golangci-lint.run/welcome/install/"

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

# =============================================================================
# Cross-compilation targets (real file targets with dependencies)
# =============================================================================

## build-all: Build binaries for all platforms
build-all: $(DIST_BINARIES) $(DIST_CHECKSUMS)
	@echo "All platform builds complete. Binaries in $(DIST_DIR)/"

# macOS Intel
$(DIST_DARWIN_AMD64): $(GO_FILES) $(GO_MOD)
	@echo "Building for darwin/amd64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $@ ./cmd/$(BINARY_NAME)

# macOS Apple Silicon
$(DIST_DARWIN_ARM64): $(GO_FILES) $(GO_MOD)
	@echo "Building for darwin/arm64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $@ ./cmd/$(BINARY_NAME)

# Linux x86_64
$(DIST_LINUX_AMD64): $(GO_FILES) $(GO_MOD)
	@echo "Building for linux/amd64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $@ ./cmd/$(BINARY_NAME)

# Linux ARM64
$(DIST_LINUX_ARM64): $(GO_FILES) $(GO_MOD)
	@echo "Building for linux/arm64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $@ ./cmd/$(BINARY_NAME)

# Windows x86_64
$(DIST_WINDOWS_AMD64): $(GO_FILES) $(GO_MOD)
	@echo "Building for windows/amd64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $@ ./cmd/$(BINARY_NAME)

## checksums: Generate SHA256 checksums for all binaries
$(DIST_CHECKSUMS): $(DIST_BINARIES)
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && shasum -a 256 $(BINARY_NAME)-* > checksums.txt
	@echo "Checksums written to $(DIST_CHECKSUMS)"

## dist-clean: Remove distribution artifacts
dist-clean:
	@echo "Cleaning dist directory..."
	@rm -rf $(DIST_DIR)/

# =============================================================================
# Docker targets
# =============================================================================

## docker-build: Build Docker image for local platform
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):$(COMMIT) \
		-t $(DOCKER_IMAGE):latest \
		.

## docker-build-multiarch: Build multi-architecture Docker image (requires docker buildx, no local load)
## Note: Multi-arch builds cannot be loaded into local docker daemon; use docker-push to push to registry
docker-build-multiarch:
	@echo "Building multi-arch Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):$(COMMIT) \
		-t $(DOCKER_IMAGE):latest \
		.

## docker-push: Build and push multi-architecture Docker image to registry
docker-push:
	@echo "Building and pushing multi-arch Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):$(COMMIT) \
		-t $(DOCKER_IMAGE):latest \
		--push \
		.
