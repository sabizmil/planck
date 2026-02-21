# Planck Makefile
# Build and development automation

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go settings
GO ?= go
GOFLAGS ?=
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

# Directories
BUILD_DIR := build
DIST_DIR := dist

# Binary name
BINARY := planck

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/planck

# Build for all platforms
.PHONY: build-all
build-all:
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/planck
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/planck
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/planck
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/planck

# Install to GOPATH/bin
.PHONY: install
install:
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./cmd/planck

# Run tests
.PHONY: test
test:
	$(GO) test -v -race ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run short tests (no integration tests)
.PHONY: test-short
test-short:
	$(GO) test -v -short ./...

# Run integration tests only
.PHONY: test-integration
test-integration:
	$(GO) test -v -run Integration ./...

# Run linter
.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# Tidy dependencies
.PHONY: tidy
tidy:
	$(GO) mod tidy

# Verify dependencies
.PHONY: verify
verify:
	$(GO) mod verify

# Generate code (mocks, etc.)
.PHONY: generate
generate:
	$(GO) generate ./...

# Run the application
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.out coverage.html

# Development mode with hot reload (requires air)
.PHONY: dev
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to normal build..."; \
		$(MAKE) run; \
	fi

# Check for required tools
.PHONY: check-tools
check-tools:
	@echo "Checking required tools..."
	@command -v $(GO) >/dev/null 2>&1 || { echo "go is required but not installed"; exit 1; }
	@command -v tmux >/dev/null 2>&1 || { echo "tmux is required but not installed"; exit 1; }
	@command -v claude >/dev/null 2>&1 || echo "warning: claude CLI not found (optional for development)"
	@echo "All required tools present"

# Release (requires goreleaser)
.PHONY: release
release:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --clean; \
	else \
		echo "goreleaser not installed. Run: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

# Release snapshot (for testing)
.PHONY: release-snapshot
release-snapshot:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not installed. Run: go install github.com/goreleaser/goreleaser@latest"; \
		exit 1; \
	fi

# Help
.PHONY: help
help:
	@echo "Planck Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build the binary"
	@echo "  make build        Build the binary"
	@echo "  make build-all    Build for all platforms"
	@echo "  make install      Install to GOPATH/bin"
	@echo "  make test         Run all tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make test-short   Run short tests (no integration)"
	@echo "  make lint         Run linter"
	@echo "  make fmt          Format code"
	@echo "  make tidy         Tidy dependencies"
	@echo "  make run          Build and run"
	@echo "  make dev          Run with hot reload (requires air)"
	@echo "  make clean        Clean build artifacts"
	@echo "  make release      Create release (requires goreleaser)"
	@echo "  make help         Show this help"
